package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/matcher" // 引入 matcher 匹配器引擎，用于复杂过滤规则

	"github.com/tidwall/gjson" // JSON 解析库
	"gorm.io/gorm"
)

// PreviousStageConfig 扩展配置结构
type PreviousStageConfig struct {
	ResultType  []string `json:"result_type"`  // 过滤 ResultType
	StageName   string   `json:"stage_name"`   // 指定 StageName，为空则默认 "prev"
	StageStatus []string `json:"stage_status"` // 指定 StageStatus (依赖 AgentTask 状态)  用于识别 AgentTask 状态,避免读取到正在运行任务产生的不完整数据
}

// UnwindConfig 展开配置结构
type UnwindConfig struct {
	Path   string            `json:"path"`   // e.g., "attributes.ports"
	Filter matcher.MatchRule `json:"filter"` // 复杂过滤规则
}

// GenerateConfig 生成配置结构
type GenerateConfig struct {
	Type          string            `json:"type"`           // e.g., "ip_port"
	ValueTemplate string            `json:"value_template"` // e.g., "{{target_value}}:{{item.port}}"
	MetaMap       map[string]string `json:"meta_map"`       // 元数据映射
}

// PreviousStageProvider 上一阶段结果提供者
// 用于将上一个扫描阶段的输出作为当前阶段的输入
// StageResult 是上一个扫描阶段的输出结果
// 它包含了上一个阶段发现的资产、漏洞、配置等信息
// 这些信息可以作为当前阶段的输入，用于进一步的扫描或分析
type PreviousStageProvider struct {
	db *gorm.DB
}

// NewPreviousStageProvider 创建新的 PreviousStageProvider
func NewPreviousStageProvider(db *gorm.DB) *PreviousStageProvider {
	return &PreviousStageProvider{
		db: db,
	}
}

func (p *PreviousStageProvider) Name() string { return "previous_stage" }

func (p *PreviousStageProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	logger.LogInfo("[PreviousStageProvider] Provide called", "", 0, "", "Provide", "", nil)

	// 事项：
	// 1. 获取当前 Workflow 上下文
	// 2. 查找上一阶段的执行结果
	// 3. 提取目标数据

	// 1. 获取上下文信息
	projectID, ok1 := ctx.Value(CtxKeyProjectID).(uint64)
	workflowID, ok2 := ctx.Value(CtxKeyWorkflowID).(uint64)
	currentStageID, ok3 := ctx.Value(CtxKeyStageID).(uint64)

	if !ok1 || !ok2 || !ok3 {
		return nil, fmt.Errorf("missing context: project_id, workflow_id or current_stage_id")
	}

	// 2. 解析配置
	var filterConfig PreviousStageConfig
	if len(config.FilterRules) > 0 {
		if err := json.Unmarshal(config.FilterRules, &filterConfig); err != nil {
			return nil, fmt.Errorf("invalid filter_rules: %w", err)
		}
	}

	var unwindConfig UnwindConfig
	var generateConfig GenerateConfig
	// 假设这些配置存储在 config.ParserConfig 中 (因为 TargetSourceConfig 没有专门的 Unwind/Generate 字段)
	// 这里我们需要一种约定，或者复用 ParserConfig
	if len(config.ParserConfig) > 0 {
		// 临时定义一个聚合结构用于解析
		type ParserWrapper struct {
			Unwind   UnwindConfig   `json:"unwind"`
			Generate GenerateConfig `json:"generate"`
		}
		var wrapper ParserWrapper
		if err := json.Unmarshal(config.ParserConfig, &wrapper); err != nil {
			return nil, fmt.Errorf("invalid parser_config: %w", err)
		}
		unwindConfig = wrapper.Unwind
		generateConfig = wrapper.Generate
	}

	// 3. 确定目标 Stage IDs
	sourceStageIDs, err := p.resolveSourceStageIDs(ctx, workflowID, currentStageID, filterConfig.StageName)
	if err != nil {
		return nil, err
	}

	// 3.5 如果指定了 StageStatus，需要先查询 AgentTask 获取合法的 AgentID
	var validAgentIDs []string
	if len(filterConfig.StageStatus) > 0 {
		var agentIDs []string
		err := p.db.WithContext(ctx).Model(&orcModel.AgentTask{}).
			Where("workflow_id = ? AND stage_id IN ? AND status IN ?", workflowID, sourceStageIDs, filterConfig.StageStatus).
			Pluck("agent_id", &agentIDs).Error
		if err != nil {
			return nil, fmt.Errorf("failed to query agent tasks status: %w", err)
		}
		if len(agentIDs) == 0 {
			// 没有符合状态的任务，直接返回空
			return []Target{}, nil
		}
		validAgentIDs = agentIDs
	}

	// 4. 查询结果
	var results []orcModel.StageResult
	query := p.db.WithContext(ctx).
		Where("project_id = ? AND workflow_id = ? AND stage_id IN ?", projectID, workflowID, sourceStageIDs)

	// DEBUG LOG
	logger.LogInfo("[PreviousStageProvider] Querying results", "", 0, "", "Provide", "", map[string]interface{}{
		"project_id":       projectID,
		"workflow_id":      workflowID,
		"source_stage_ids": sourceStageIDs,
	})

	if len(validAgentIDs) > 0 {
		query = query.Where("agent_id IN ?", validAgentIDs)
	}

	if len(filterConfig.ResultType) > 0 {
		query = query.Where("result_type IN ?", filterConfig.ResultType)
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to query stage results: %w", err)
	}
	logger.LogInfo(fmt.Sprintf("[PreviousStageProvider] Found %d results", len(results)), "", 0, "", "Provide", "", nil)

	// 5. 处理结果并生成 Target
	var targets []Target
	for _, result := range results {
		logger.LogInfo(fmt.Sprintf("[PreviousStageProvider] Processing result: ID=%d", result.ID), "", 0, "", "Provide", "", map[string]interface{}{
			"attributes": result.Attributes,
		})
		newTargets := p.processResult(result, unwindConfig, generateConfig)
		logger.LogInfo(fmt.Sprintf("[PreviousStageProvider] Generated %d targets from result %d", len(newTargets), result.ID), "", 0, "", "Provide", "", nil)
		targets = append(targets, newTargets...)
	}

	return targets, nil
}

// resolveSourceStageIDs 解析目标 StageIDs (DAG 支持)
func (p *PreviousStageProvider) resolveSourceStageIDs(ctx context.Context, workflowID uint64, currentStageID uint64, stageName string) ([]uint64, error) {
	// 如果指定了 stage_name，直接按名称查找
	if stageName != "" && stageName != "prev" {
		var stage orcModel.ScanStage
		err := p.db.WithContext(ctx).
			Where("workflow_id = ? AND stage_name = ?", workflowID, stageName).
			First(&stage).Error
		if err != nil {
			return nil, fmt.Errorf("stage not found: %s", stageName)
		}
		return []uint64{uint64(stage.ID)}, nil
	}

	// 否则查找前置 Stage (Predecessors)
	var currentStage orcModel.ScanStage
	if err := p.db.WithContext(ctx).First(&currentStage, currentStageID).Error; err != nil {
		return nil, fmt.Errorf("failed to load current stage: %w", err)
	}

	predecessors := currentStage.Predecessors

	if len(predecessors) == 0 {
		return nil, fmt.Errorf("no previous stage found (no predecessors)")
	}

	return predecessors, nil
}

// processResult 处理单条结果 --- 核心实现逻辑
// 1. 如果没有 Unwind 配置，直接使用 Result 本身生成 Target --- Unwind Json 展开数据
// 2. 如果有 Unwind 配置，根据路径展开 JSON 数组
// 3. 应用过滤条件
// 4. 渲染生成 Target
func (p *PreviousStageProvider) processResult(result orcModel.StageResult, unwind UnwindConfig, generate GenerateConfig) []Target {
	// 如果没有 Unwind 配置，直接使用 Result 本身生成 Target
	if unwind.Path == "" {
		return []Target{{
			Type:   result.TargetType,
			Value:  result.TargetValue,
			Source: fmt.Sprintf("stage:%d", result.StageID),
			Meta:   map[string]string{"result_type": result.ResultType},
		}}
	}

	// 有 Unwind 配置，进行展开
	// 1. 获取 JSON 路径下的内容
	// StageResult 的 Attributes 是 JSON 字符串
	jsonStr := result.Attributes
	if jsonStr == "" {
		return nil
	}

	// 使用 gjson 高效展开 JSON 数组
	gjsonResult := gjson.Get(jsonStr, unwind.Path)
	if !gjsonResult.IsArray() {
		return nil
	}

	var targets []Target
	// 2. 遍历数组
	gjsonResult.ForEach(func(key, value gjson.Result) bool {
		// 3. 过滤逻辑 (复杂实现：使用 Matcher)
		val := value.Value()
		matched, err := matcher.Match(val, unwind.Filter)
		if err != nil {
			// 如果规则执行出错，视为不匹配
			return true // continue
		}
		if !matched {
			return true // continue
		}

		// 【这里需要结合stageResult的返回数据结构进行解析】
		// 4. 渲染模板 支持 {{target_value}} 和 {{item.field}}
		targetValue := generate.ValueTemplate
		// 替换 {{target_value}}
		targetValue = strings.ReplaceAll(targetValue, "{{target_value}}", result.TargetValue)
		// 替换 {{item.xxx}}
		// 简单的正则替换可能不够健壮，这里使用简单的遍历替换
		// 假设模板中只有简单的 {{item.field}}
		value.ForEach(func(k, v gjson.Result) bool {
			placeholder := fmt.Sprintf("{{item.%s}}", k.String())
			targetValue = strings.ReplaceAll(targetValue, placeholder, v.String())
			return true
		})

		// 替换 {{item}} 本身 (如果是基本类型)
		if value.Type != gjson.JSON {
			targetValue = strings.ReplaceAll(targetValue, "{{item}}", value.String())
		}

		// 5. 生成 Target
		t := Target{
			Type:   generate.Type,
			Value:  targetValue,
			Source: fmt.Sprintf("stage:%d", result.StageID),
			Meta:   make(map[string]string),
		}
		// 复制 Meta
		for k, v := range generate.MetaMap {
			t.Meta[k] = v
		}

		// 6. 加入结果列表
		targets = append(targets, t)
		return true
	})

	return targets
}

func (p *PreviousStageProvider) HealthCheck(ctx context.Context) error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
