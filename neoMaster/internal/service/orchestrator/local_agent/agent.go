// LocalAgent 本地Agent (原 SystemWorker)
// 负责执行 Master 内部产生的系统任务 (如: 标签传播、资产清洗等)
package local_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	assetModel "neomaster/internal/model/asset"
	"neomaster/internal/model/orchestrator"
	"neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/matcher"
	orcrepo "neomaster/internal/repo/mysql/orchestrator"
)

// LocalAgent 本地Agent
// 运行在 Master 进程内的特殊 Worker，负责执行数据密集型或高权限的系统任务
// 它直接连接数据库，不通过 HTTP 协议
// 对应文档: 1.1 Orchestrator - LocalAgent
type LocalAgent struct {
	db        *gorm.DB
	taskRepo  orcrepo.TaskRepository
	isRunning bool
	stopChan  chan struct{}
	wg        sync.WaitGroup
	interval  time.Duration
}

// NewLocalAgent 创建本地Agent实例
func NewLocalAgent(db *gorm.DB, taskRepo orcrepo.TaskRepository) *LocalAgent {
	return &LocalAgent{
		db:       db,
		taskRepo: taskRepo,
		stopChan: make(chan struct{}),
		interval: 5 * time.Second, // 默认每5秒轮询一次
	}
}

// SetInterval 设置轮询间隔 (用于测试或配置调整)
func (w *LocalAgent) SetInterval(interval time.Duration) {
	w.interval = interval
}

// Start 启动执行器
func (w *LocalAgent) Start() {
	if w.isRunning {
		return
	}
	w.isRunning = true
	w.wg.Add(1)
	go w.run()

	logger.WithFields(map[string]interface{}{
		"path":      "service.orchestrator.local_agent",
		"operation": "start",
	}).Info("LocalAgent started")
}

// Stop 停止执行器
func (w *LocalAgent) Stop() {
	if !w.isRunning {
		return
	}
	close(w.stopChan)
	w.wg.Wait()
	w.isRunning = false

	logger.WithFields(map[string]interface{}{
		"path":      "service.orchestrator.local_agent",
		"operation": "stop",
	}).Info("LocalAgent stopped")
}

// run 主循环
func (w *LocalAgent) run() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.processTasks()
		}
	}
}

// processTasks 处理待执行的系统任务
func (w *LocalAgent) processTasks() {
	ctx := context.Background()

	// 1. 获取待执行的系统任务
	tasks, err := w.taskRepo.GetPendingTasks(ctx, "system", 10) // 每次处理10个
	if err != nil {
		logger.LogError(err, "failed to get pending system tasks", 0, "", "service.orchestrator.local_agent.processTasks", "REPO", nil)
		return
	}

	if len(tasks) == 0 {
		return
	}

	// 2. 遍历执行
	for _, task := range tasks {
		// 2. 更新状态为 running
		if err := w.taskRepo.UpdateTaskStatus(ctx, task.TaskID, "running"); err != nil {
			logger.LogWarn("Failed to update task status to running", "", 0, "", "local_agent.processTasks", "", map[string]interface{}{"error": err, "task_id": task.TaskID})
			continue
		}

		// 3. 执行任务
		var result interface{}
		var execErr error

		// 根据 ToolName 分发任务
		// 约定: 系统任务的 ToolName 以 "sys_" 开头
		switch task.ToolName {
		case "sys_tag_propagation":
			result, execErr = w.handleTagPropagation(ctx, task)
		case "sys_asset_cleanup":
			result, execErr = w.handleAssetCleanup(ctx, task)
		default:
			execErr = fmt.Errorf("unknown system tool: %s", task.ToolName)
		}

		// 4. 更新任务结果
		status := "completed"
		errorMsg := ""
		resultJSON := "{}"

		if execErr != nil {
			// 检查重试逻辑
			if task.RetryCount < task.MaxRetries {
				retryCount := task.RetryCount + 1
				retryMsg := fmt.Sprintf("System Task Retry %d/%d: %v", retryCount, task.MaxRetries, execErr)
				logger.LogWarn("System task failed, retrying...", "", 0, "", "local_agent.processTasks", "", map[string]interface{}{
					"task_id":     task.TaskID,
					"retry_count": retryCount,
					"reason":      execErr.Error(),
				})
				if err := w.taskRepo.RetryTask(ctx, task.TaskID, retryCount, retryMsg); err != nil {
					logger.LogWarn("Failed to retry task", "", 0, "", "local_agent.processTasks", "", map[string]interface{}{"error": err, "task_id": task.TaskID})
				}
				continue // Skip update result to avoid marking as failed
			}

			status = "failed"
			errorMsg = execErr.Error()
		}

		if result != nil {
			if b, err := json.Marshal(result); err == nil {
				resultJSON = string(b)
			}
		}

		if err := w.taskRepo.UpdateTaskResult(ctx, task.TaskID, resultJSON, errorMsg, status); err != nil {
			logger.LogWarn("Failed to update task result", "", 0, "", "local_agent.processTasks", "", map[string]interface{}{"error": err, "task_id": task.TaskID})
		}
	}
}

// TagPropagationPayload 标签传播任务载荷
type TagPropagationPayload struct {
	TargetType string            `json:"target_type"` // host, web, network
	Action     string            `json:"action"`      // add, remove
	Tags       []string          `json:"tags"`
	TagIDs     []uint64          `json:"tag_ids"`
	Rule       matcher.MatchRule `json:"rule"`
	RuleID     uint64            `json:"rule_id"`
}

// handleTagPropagation 处理标签传播任务
func (w *LocalAgent) handleTagPropagation(ctx context.Context, task *orchestrator.AgentTask) (map[string]interface{}, error) {
	var payload TagPropagationPayload
	if err := json.Unmarshal([]byte(task.ToolParams), &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}

	// 默认 action 为 add
	if payload.Action == "" {
		payload.Action = "add"
	}

	var count int64
	var err error

	switch payload.TargetType {
	case "host":
		count, err = w.processAssetHost(ctx, payload)
	case "web":
		count, err = w.processAssetWeb(ctx, payload)
	case "network":
		count, err = w.processAssetNetwork(ctx, payload)
	default:
		return nil, fmt.Errorf("unsupported target_type: %s", payload.TargetType)
	}

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"processed_count": count,
		"target_type":     payload.TargetType,
		"action":          payload.Action,
	}, nil
}

// processAssetHost 处理主机资产标签
func (w *LocalAgent) processAssetHost(ctx context.Context, payload TagPropagationPayload) (int64, error) {
	var count int64
	var batchSize = 100
	var assets []assetModel.AssetHost

	err := w.db.WithContext(ctx).Model(&assetModel.AssetHost{}).FindInBatches(&assets, batchSize, func(tx *gorm.DB, batch int) error {
		for _, asset := range assets {
			// 1. 转换为 Map 用于匹配
			assetData, err := structToMap(asset)
			if err != nil {
				continue
			}

			// 2. 执行匹配
			matched, err := matcher.Match(assetData, payload.Rule)
			if err != nil {
				logger.LogWarn("Matcher error", "", 0, "", "local_agent.processAssetHost", "", map[string]interface{}{"error": err, "asset_id": asset.ID})
				continue
			}

			if matched {
				count++
				w.syncEntityTags(ctx, "host", strconv.FormatUint(asset.ID, 10), payload)
			}
		}
		return nil
	}).Error

	return count, err
}

// processAssetWeb 处理Web资产标签
func (w *LocalAgent) processAssetWeb(ctx context.Context, payload TagPropagationPayload) (int64, error) {
	var count int64
	var batchSize = 100
	var assets []assetModel.AssetWeb

	err := w.db.WithContext(ctx).Model(&assetModel.AssetWeb{}).FindInBatches(&assets, batchSize, func(tx *gorm.DB, batch int) error {
		for _, asset := range assets {
			assetData, err := structToMap(asset)
			if err != nil {
				continue
			}

			matched, err := matcher.Match(assetData, payload.Rule)
			if err != nil {
				logger.LogWarn("Matcher error", "", 0, "", "local_agent.processAssetWeb", "", map[string]interface{}{"error": err, "asset_id": asset.ID})
				continue
			}

			if matched {
				count++
				w.syncEntityTags(ctx, "web", strconv.FormatUint(asset.ID, 10), payload)
			}
		}
		return nil
	}).Error

	return count, err
}

// processAssetNetwork 处理网段资产标签
func (w *LocalAgent) processAssetNetwork(ctx context.Context, payload TagPropagationPayload) (int64, error) {
	var count int64
	var batchSize = 100
	var assets []assetModel.AssetNetwork

	err := w.db.WithContext(ctx).Model(&assetModel.AssetNetwork{}).FindInBatches(&assets, batchSize, func(tx *gorm.DB, batch int) error {
		// 0. 批量获取标签
		var assetIDs []string
		for _, asset := range assets {
			assetIDs = append(assetIDs, strconv.FormatUint(asset.ID, 10))
		}
		tagsMap, _ := w.fetchTagsForAssets(ctx, "network", assetIDs)

		for _, asset := range assets {
			assetData, err := structToMap(asset)
			if err != nil {
				continue
			}

			// 注入标签数据
			if tags, ok := tagsMap[strconv.FormatUint(asset.ID, 10)]; ok {
				assetData["tags"] = tags
			}

			matched, err := matcher.Match(assetData, payload.Rule)
			if err != nil {
				logger.LogWarn("Matcher error", "", 0, "", "local_agent.processAssetNetwork", "", map[string]interface{}{"error": err, "asset_id": asset.ID})
				continue
			}

			if matched {
				count++
				w.syncEntityTags(ctx, "network", strconv.FormatUint(asset.ID, 10), payload)
			}
		}
		return nil
	}).Error

	return count, err
}

// AssetCleanupPayload 资产清洗任务载荷
type AssetCleanupPayload struct {
	TargetType string            `json:"target_type"` // host, web, network
	Rule       matcher.MatchRule `json:"rule"`
}

// handleAssetCleanup 处理资产清洗任务
func (w *LocalAgent) handleAssetCleanup(ctx context.Context, task *orchestrator.AgentTask) (map[string]interface{}, error) {
	var payload AssetCleanupPayload
	if err := json.Unmarshal([]byte(task.ToolParams), &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}

	var count int64
	var err error

	switch payload.TargetType {
	case "host":
		count, err = w.processCleanupHost(ctx, payload)
	case "web":
		count, err = w.processCleanupWeb(ctx, payload)
	case "network":
		count, err = w.processCleanupNetwork(ctx, payload)
	default:
		return nil, fmt.Errorf("unsupported target_type: %s", payload.TargetType)
	}

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"deleted_count": count,
		"target_type":   payload.TargetType,
	}, nil
}

// processCleanupHost 清洗主机资产
func (w *LocalAgent) processCleanupHost(ctx context.Context, payload AssetCleanupPayload) (int64, error) {
	var count int64
	var batchSize = 100
	var assets []assetModel.AssetHost

	err := w.db.WithContext(ctx).Model(&assetModel.AssetHost{}).FindInBatches(&assets, batchSize, func(tx *gorm.DB, batch int) error {
		for _, asset := range assets {
			assetData, err := structToMap(asset)
			if err != nil {
				continue
			}

			matched, err := matcher.Match(assetData, payload.Rule)
			if err != nil {
				logger.LogWarn("Matcher error", "", 0, "", "local_agent.processCleanupHost", "", map[string]interface{}{"error": err, "asset_id": asset.ID})
				continue
			}

			if matched {
				if err := w.db.Delete(&asset).Error; err == nil {
					count++
				}
			}
		}
		return nil
	}).Error

	return count, err
}

// processCleanupWeb 清洗Web资产
func (w *LocalAgent) processCleanupWeb(ctx context.Context, payload AssetCleanupPayload) (int64, error) {
	var count int64
	var batchSize = 100
	var assets []assetModel.AssetWeb

	err := w.db.WithContext(ctx).Model(&assetModel.AssetWeb{}).FindInBatches(&assets, batchSize, func(tx *gorm.DB, batch int) error {
		// 0. 批量获取标签
		var assetIDs []string
		for _, asset := range assets {
			assetIDs = append(assetIDs, strconv.FormatUint(asset.ID, 10))
		}
		tagsMap, _ := w.fetchTagsForAssets(ctx, "web", assetIDs)

		for _, asset := range assets {
			assetData, err := structToMap(asset)
			if err != nil {
				continue
			}

			// 注入标签数据
			if tags, ok := tagsMap[strconv.FormatUint(asset.ID, 10)]; ok {
				assetData["tags"] = tags
			}

			matched, err := matcher.Match(assetData, payload.Rule)
			if err != nil {
				logger.LogWarn("Matcher error", "", 0, "", "local_agent.processCleanupWeb", "", map[string]interface{}{"error": err, "asset_id": asset.ID})
				continue
			}

			if matched {
				if err := w.db.Delete(&asset).Error; err == nil {
					count++
				}
			}
		}
		return nil
	}).Error

	return count, err
}

// processCleanupNetwork 清洗网段资产
func (w *LocalAgent) processCleanupNetwork(ctx context.Context, payload AssetCleanupPayload) (int64, error) {
	var count int64
	var batchSize = 100
	var assets []assetModel.AssetNetwork

	err := w.db.WithContext(ctx).Model(&assetModel.AssetNetwork{}).FindInBatches(&assets, batchSize, func(tx *gorm.DB, batch int) error {
		// 0. 批量获取标签
		var assetIDs []string
		for _, asset := range assets {
			assetIDs = append(assetIDs, strconv.FormatUint(asset.ID, 10))
		}
		tagsMap, _ := w.fetchTagsForAssets(ctx, "network", assetIDs)

		for _, asset := range assets {
			assetData, err := structToMap(asset)
			if err != nil {
				continue
			}

			// 注入标签数据
			if tags, ok := tagsMap[strconv.FormatUint(asset.ID, 10)]; ok {
				assetData["tags"] = tags
			}

			matched, err := matcher.Match(assetData, payload.Rule)
			if err != nil {
				logger.LogWarn("Matcher error", "", 0, "", "local_agent.processCleanupNetwork", "", map[string]interface{}{"error": err, "asset_id": asset.ID})
				continue
			}

			if matched {
				if err := w.db.Delete(&asset).Error; err == nil {
					count++
				}
			}
		}
		return nil
	}).Error

	return count, err
}

// syncEntityTags 同步实体关联标签表
func (w *LocalAgent) syncEntityTags(ctx context.Context, entityType string, entityID string, payload TagPropagationPayload) {
	if len(payload.TagIDs) == 0 {
		return
	}

	switch payload.Action {
	case "add":
		for _, tagID := range payload.TagIDs {
			entityTag := tag_system.SysEntityTag{
				EntityType: entityType,
				EntityID:   entityID,
				TagID:      tagID,
				Source:     "auto",
				RuleID:     payload.RuleID,
			}
			// 使用 Upsert 避免重复添加，并更新 Source 和 RuleID
			w.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "entity_type"}, {Name: "entity_id"}, {Name: "tag_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"source", "rule_id"}),
			}).Create(&entityTag)
		}
	case "remove":
		w.db.WithContext(ctx).Where("entity_type = ? AND entity_id = ? AND tag_id IN ?",
			entityType, entityID, payload.TagIDs).
			Delete(&tag_system.SysEntityTag{})
	}
}

// fetchTagsForAssets 批量获取资产标签
func (w *LocalAgent) fetchTagsForAssets(ctx context.Context, entityType string, entityIDs []string) (map[string][]string, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}

	type AssetTag struct {
		EntityID string
		Name     string
	}
	var results []AssetTag

	err := w.db.WithContext(ctx).Table("sys_entity_tags").
		Select("sys_entity_tags.entity_id, sys_tags.name").
		Joins("JOIN sys_tags ON sys_entity_tags.tag_id = sys_tags.id").
		Where("sys_entity_tags.entity_type = ? AND sys_entity_tags.entity_id IN ?", entityType, entityIDs).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	tagMap := make(map[string][]string)
	for _, r := range results {
		tagMap[r.EntityID] = append(tagMap[r.EntityID], r.Name)
	}
	return tagMap, nil
}

// structToMap 辅助函数：结构体转Map
func structToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	err = json.Unmarshal(data, &res)
	return res, err
}
