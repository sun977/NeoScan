/**
 * 服务层:Agent监控服务
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent监控核心业务逻辑，遵循"好品味"原则 - 专注监控职责
 * @func: Agent心跳处理、性能指标监控、健康状态检查、负载监控
 */
package agent

import (
	"context"
	"fmt"
	agentRepository "neomaster/internal/repo/mysql/agent"
	"sort"
	"time"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/tag_system"
)

// AgentMonitorService Agent监控服务接口
// 专门负责Agent的监控相关功能，遵循单一职责原则
type AgentMonitorService interface {
	// Agent 心跳和状态监控
	ProcessHeartbeat(req *agentModel.HeartbeatRequest) (*agentModel.HeartbeatResponse, error)                                                                                                        // 处理Agent发送过来的心跳，更新状态和指标
	GetAgentMetricsFromDB(agentID string) (*agentModel.AgentMetricsResponse, error)                                                                                                                  // 从数据库获取Agent最新的性能指标
	GetAgentListAllMetricsFromDB(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetricsResponse, int64, error) // 从数据库分页获取Agent的最新性能指标（支持状态与关键词过滤）
	PullAgentMetrics(agentID string) (*agentModel.AgentMetricsResponse, error)                                                                                                                       // 从Agent端拉取最新的性能指标
	PullAgentListAllMetrics() ([]*agentModel.AgentMetricsResponse, error)                                                                                                                            // 从Agent端拉取所有Agent的最新性能指标
	CreateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error                                                                                                                       // 创建Agent性能指标
	UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error                                                                                                                       // 更新Agent性能指标

	// Agent 数据分析 (可按标签聚合)
	GetAgentStatistics(windowSeconds int, tagIDs []uint64) (*agentModel.AgentStatisticsResponse, error)                                              // 获取Agent统计信息
	GetAgentLoadBalance(windowSeconds int, topN int, tagIDs []uint64) (*agentModel.AgentLoadBalanceResponse, error)                                  // 获取负载均衡分析
	GetAgentPerformanceAnalysis(windowSeconds int, topN int, tagIDs []uint64) (*agentModel.AgentPerformanceAnalysisResponse, error)                  // 获取性能分析
	GetAgentCapacityAnalysis(windowSeconds int, cpuThr, memThr, diskThr float64, tagIDs []uint64) (*agentModel.AgentCapacityAnalysisResponse, error) // 获取容量分析
}

// agentMonitorService Agent监控服务实现
type agentMonitorService struct {
	agentRepo     agentRepository.AgentRepository // Agent数据访问层
	tagService    tag_system.TagService           // Tag服务
	updateService AgentUpdateService              // 规则更新服务,用于获取规则版本信息返回给Agent
}

// NewAgentMonitorService 创建Agent监控服务实例
// 遵循依赖注入原则，保持代码的可测试性
func NewAgentMonitorService(agentRepo agentRepository.AgentRepository, tagService tag_system.TagService, updateService AgentUpdateService) AgentMonitorService {
	return &agentMonitorService{
		agentRepo:     agentRepo,
		tagService:    tagService,
		updateService: updateService,
	}
}

// ProcessHeartbeat 处理Agent心跳请求 - 优化后的版本
// 将心跳状态更新和性能指标存储分离，体现"好品味"的数据处理逻辑
func (s *agentMonitorService) ProcessHeartbeat(req *agentModel.HeartbeatRequest) (*agentModel.HeartbeatResponse, error) {
	// 1. 更新Agent心跳状态信息到agents表
	// 只更新last_heartbeat、updated_at、status字段，其他字段在注册时已确定
	// 更新 status 字段 - agents 表
	err := s.agentRepo.UpdateStatus(req.AgentID, req.Status)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.UpdateStatus",
			"func_name": "service.agent.monitor.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
		return nil, err
	}

	// 更新最后心跳时间 - agents 表 (同时更新 updated_at 和 last_heartbeat 字段)
	err = s.agentRepo.UpdateLastHeartbeat(req.AgentID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.UpdateLastHeartbeat",
			"func_name": "service.agent.monitor.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
		return nil, err
	}

	// 2. 处理性能指标数据到agent_metrics表
	// Agent已经提供了完整的性能指标数据，直接使用即可
	if req.Metrics != nil {
		// 确保AgentID正确设置
		req.Metrics.AgentID = req.AgentID

		// 更新性能指标到agent_metrics表（使用upsert逻辑：存在则更新，不存在则创建）
		err = s.agentRepo.UpdateAgentMetrics(req.AgentID, req.Metrics)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
				"operation": "process_heartbeat",
				"option":    "agentRepo.UpdateAgentMetrics",
				"func_name": "service.agent.monitor.ProcessHeartbeat",
				"agent_id":  req.AgentID,
			})
			return nil, err
		}

		logger.LogInfo("Agent性能指标更新成功", "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.UpdateAgentMetrics",
			"func_name": "service.agent.monitor.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
	}

	logger.LogInfo("Agent心跳处理成功", "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
		"operation": "process_heartbeat",
		"option":    "ProcessHeartbeat",
		"func_name": "service.agent.monitor.ProcessHeartbeat",
		"agent_id":  req.AgentID,
		"status":    string(req.Status),
	})

	// 获取规则版本信息
	// 这里的 GetSnapshotInfo 已经实现了内存缓存 (Lazy Cache)，所以调用开销非常小
	// 我们目前只关注 Fingerprint，后续可扩展
	ruleVersions := make(map[string]string)

	// 指纹规则
	if info, err := s.updateService.GetSnapshotInfo(context.Background(), RuleTypeFingerprint); err == nil && info != nil {
		ruleVersions[string(RuleTypeFingerprint)] = info.VersionHash
	}

	// POC规则 (如果有)
	// if info, err := s.updateService.GetSnapshotInfo(context.Background(), RuleTypePOC); err == nil && info != nil {
	// 	ruleVersions[string(RuleTypePOC)] = info.VersionHash
	// }

	// 构造响应
	response := &agentModel.HeartbeatResponse{
		AgentID:      req.AgentID,
		Status:       "success",
		Message:      "Heartbeat processed successfully",
		Timestamp:    time.Now(),
		RuleVersions: ruleVersions, // 规则版本信息
	}

	return response, nil
}

// GetAgentMetricsFromDB 获取指定Agent性能指标服务 - 从数据库表 agent_metrics 查询
func (s *agentMonitorService) GetAgentMetricsFromDB(agentID string) (*agentModel.AgentMetricsResponse, error) {
	// 输入校验：agentID不能为空
	if agentID == "" {
		err := fmt.Errorf("agentID 不能为空")
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentMetricsFromDB", "", map[string]interface{}{
			"operation": "get_agent_metrics_from_db",
			"option":    "validate.agentID",
			"func_name": "service.agent.monitor.GetAgentMetricsFromDB",
			"agent_id":  agentID,
		})
		return nil, err
	}

	// 通过仓储层获取该Agent的最新性能指标（遵循分层：Service -> Repository）
	metrics, err := s.agentRepo.GetLatestMetrics(agentID)
	if err != nil {
		// 数据查询错误
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentMetricsFromDB", "", map[string]interface{}{
			"operation": "get_agent_metrics_from_db",
			"option":    "agentRepo.GetLatestMetrics",
			"func_name": "service.agent.monitor.GetAgentMetricsFromDB",
			"agent_id":  agentID,
		})
		return nil, fmt.Errorf("查询Agent性能指标失败: %v", err)
	}

	// 未找到记录（根据当前单快照模型，表示尚未有该Agent的指标被写入）
	if metrics == nil {
		err := fmt.Errorf("未找到Agent的性能指标记录")
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentMetricsFromDB", "", map[string]interface{}{
			"operation": "get_agent_metrics_from_db",
			"option":    "agentRepo.GetLatestMetrics.not_found",
			"func_name": "service.agent.monitor.GetAgentMetricsFromDB",
			"agent_id":  agentID,
		})
		return nil, err
	}

	// 组装响应结构（与AgentMetricsResponse字段一一对应）
	resp := &agentModel.AgentMetricsResponse{
		AgentID:           metrics.AgentID,
		CPUUsage:          metrics.CPUUsage,
		MemoryUsage:       metrics.MemoryUsage,
		DiskUsage:         metrics.DiskUsage,
		NetworkBytesSent:  metrics.NetworkBytesSent,
		NetworkBytesRecv:  metrics.NetworkBytesRecv,
		ActiveConnections: metrics.ActiveConnections,
		RunningTasks:      metrics.RunningTasks,
		CompletedTasks:    metrics.CompletedTasks,
		FailedTasks:       metrics.FailedTasks,
		WorkStatus:        metrics.WorkStatus,
		ScanType:          metrics.ScanType,
		// PluginStatusJSON 是 map[string]interface{} 的自定义类型，直接类型转换即可
		PluginStatus: map[string]interface{}(metrics.PluginStatus),
		Timestamp:    metrics.Timestamp,
	}

	// 成功日志记录
	logger.LogInfo("获取Agent性能指标成功", "", 0, "", "service.agent.monitor.GetAgentMetricsFromDB", "", map[string]interface{}{
		"operation": "get_agent_metrics_from_db",
		"option":    "compose.AgentMetricsResponse",
		"func_name": "service.agent.monitor.GetAgentMetricsFromDB",
		"agent_id":  agentID,
	})

	return resp, nil
}

// GetAgentListAllMetricsFromDB 获取所有Agent性能指标服务 - 从数据表 agent_metrics 批量查询
func (s *agentMonitorService) GetAgentListAllMetricsFromDB(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetricsResponse, int64, error) {
	// 输入校验与默认值处理（防御性编程）
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 直接通过仓储层对 agent_metrics 表进行分页查询（单快照模型，避免N+1与全量加载）
	metricsList, total, err := s.agentRepo.GetMetricsList(page, pageSize, workStatus, scanType, keyword)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentListAllMetricsFromDB", "", map[string]interface{}{
			"operation": "get_agent_list_all_metrics_from_db",
			"option":    "agentRepo.GetMetricsList",
			"func_name": "service.agent.monitor.GetAgentListAllMetricsFromDB",
			"page":      page,
			"page_size": pageSize,
			"work_status": func() interface{} {
				if workStatus != nil {
					return *workStatus
				}
				return nil
			}(),
			"scan_type": func() interface{} {
				if scanType != nil {
					return *scanType
				}
				return nil
			}(),
			"keyword": func() interface{} {
				if keyword != nil {
					return *keyword
				}
				return nil
			}(),
		})
		return nil, 0, fmt.Errorf("查询Agent性能指标分页失败: %v", err)
	}

	// 组装响应结构
	results := make([]*agentModel.AgentMetricsResponse, 0, len(metricsList))
	for _, metrics := range metricsList {
		if metrics == nil {
			continue
		}
		resp := &agentModel.AgentMetricsResponse{
			AgentID:           metrics.AgentID,
			CPUUsage:          metrics.CPUUsage,
			MemoryUsage:       metrics.MemoryUsage,
			DiskUsage:         metrics.DiskUsage,
			NetworkBytesSent:  metrics.NetworkBytesSent,
			NetworkBytesRecv:  metrics.NetworkBytesRecv,
			ActiveConnections: metrics.ActiveConnections,
			RunningTasks:      metrics.RunningTasks,
			CompletedTasks:    metrics.CompletedTasks,
			FailedTasks:       metrics.FailedTasks,
			WorkStatus:        metrics.WorkStatus,
			ScanType:          metrics.ScanType,
			PluginStatus:      map[string]interface{}(metrics.PluginStatus),
			Timestamp:         metrics.Timestamp,
		}
		results = append(results, resp)
	}

	// 成功日志记录
	logger.LogInfo("获取所有Agent性能指标成功（分页）", "", 0, "", "service.agent.monitor.GetAgentListAllMetricsFromDB", "", map[string]interface{}{
		"operation": "get_agent_list_all_metrics_from_db",
		"option":    "agentRepo.GetMetricsList",
		"func_name": "service.agent.monitor.GetAgentListAllMetricsFromDB",
		"count":     len(results),
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"work_status": func() interface{} {
			if workStatus != nil {
				return *workStatus
			}
			return nil
		}(),
		"scan_type": func() interface{} {
			if scanType != nil {
				return *scanType
			}
			return nil
		}(),
		"keyword": func() interface{} {
			if keyword != nil {
				return *keyword
			}
			return nil
		}(),
	})

	return results, total, nil
}

// PullAgentMetrics 从 agent 端拉取性能指标数据服务 - 从 agent 端调用 /metrics 接口获取数据 (拉取后同步更新数据表 agent_metrics 中数据)
func (s *agentMonitorService) PullAgentMetrics(agentID string) (*agentModel.AgentMetricsResponse, error) {
	// TODO: 实现从 agent 端调用 /metrics 接口获取数据
	logger.LogInfo("从 agent 端拉取性能指标数据", "", 0, "", "service.agent.monitor.pullAgentMetrics", "", map[string]interface{}{
		"operation": "pull_agent_metrics",
		"option":    "agentMonitorService.PullAgentMetrics",
		"func_name": "service.agent.monitor.PullAgentMetrics",
		"agent_id":  agentID,
	})
	return nil, fmt.Errorf("功能暂未实现")
}

// PullAgentListAllMetrics 从所有 agent 端拉取性能指标数据服务 - 从所有 agent 端调用 /metrics 接口获取数据 (拉取后同步更新数据表 agent_metrics 中数据)
func (s *agentMonitorService) PullAgentListAllMetrics() ([]*agentModel.AgentMetricsResponse, error) {
	// TODO: 实现从所有 agent 端调用 /metrics 接口获取数据
	logger.LogInfo("从所有 agent 端拉取性能指标数据", "", 0, "", "service.agent.monitor.pullAgentListAllMetrics", "", map[string]interface{}{
		"operation": "pull_agent_list_all_metrics",
		"option":    "agentMonitorService.PullAgentListAllMetrics",
		"func_name": "service.agent.monitor.PullAgentListAllMetrics",
	})
	return nil, fmt.Errorf("功能暂未实现")
}

// CreateAgentMetrics 创建Agent性能指标服务 (保留服务,用于agent上报性能指标数据或master创建性能指标数据)
func (s *agentMonitorService) CreateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
	// TODO: 实现创建Agent性能指标到数据表 agent_metrics 中
	if err := s.agentRepo.CreateMetrics(metrics); err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.CreateAgentMetrics", "", map[string]interface{}{
			"operation": "create_agent_metrics",
			"option":    "agentMonitorService.CreateAgentMetrics",
			"func_name": "service.agent.monitor.CreateAgentMetrics",
			"agent_id":  agentID,
		})
		return fmt.Errorf("创建Agent性能指标失败: %v", err)
	}

	logger.LogInfo("Agent性能指标创建成功", "", 0, "", "service.agent.monitor.CreateAgentMetrics", "", map[string]interface{}{
		"operation": "create_agent_metrics",
		"option":    "agentMonitorService.CreateAgentMetrics",
		"func_name": "service.agent.monitor.CreateAgentMetrics",
		"agent_id":  agentID,
	})

	return nil
}

// UpdateAgentMetrics 更新Agent性能指标服务 (保留服务,用于master恢复性能指标数据)
func (s *agentMonitorService) UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
	if err := s.agentRepo.UpdateAgentMetrics(agentID, metrics); err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.UpdateAgentMetrics", "", map[string]interface{}{
			"operation": "update_agent_metrics",
			"option":    "agentMonitorService.UpdateAgentMetrics",
			"func_name": "service.agent.monitor.UpdateAgentMetrics",
			"agent_id":  agentID,
		})
		return fmt.Errorf("更新Agent性能指标失败: %v", err)
	}

	logger.LogInfo("Agent性能指标更新成功", "", 0, "", "service.agent.monitor.UpdateAgentMetrics", "", map[string]interface{}{
		"operation": "update_agent_metrics",
		"option":    "agentMonitorService.UpdateAgentMetrics",
		"func_name": "service.agent.monitor.UpdateAgentMetrics",
		"agent_id":  agentID,
	})

	return nil
}

// ==================== 数据分析实现 ====================

// GetAgentStatistics 获取Agent统计信息（仅基于agent_metrics快照）
func (s *agentMonitorService) GetAgentStatistics(windowSeconds int, tagIDs []uint64) (*agentModel.AgentStatisticsResponse, error) {
	if windowSeconds <= 0 {
		windowSeconds = 180
	}
	since := time.Now().Add(-time.Duration(windowSeconds) * time.Second)

	// 全量与窗口内快照
	var all []*agentModel.AgentMetrics
	var window []*agentModel.AgentMetrics
	var err error

	if len(tagIDs) > 0 {
		// 按标签过滤
		var agentIDs []string
		agentIDs, err = s.tagService.GetEntityIDsByTagIDs(context.Background(), "agent", tagIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentStatistics", "", map[string]interface{}{
				"operation": "get_agent_statistics",
				"option":    "tagService.GetEntityIDsByTagIDs",
				"func_name": "service.agent.monitor.GetAgentStatistics",
				"tag_ids":   tagIDs,
			})
			return nil, err
		}

		// 如果指定了标签但没有找到对应的Agent，直接返回空结果
		if len(agentIDs) == 0 {
			return &agentModel.AgentStatisticsResponse{
				Performance:            agentModel.AggregatedPerformance{},
				WorkStatusDistribution: map[string]int64{},
				ScanTypeDistribution:   map[string]int64{},
			}, nil
		}

		all, err = s.agentRepo.GetMetricsByAgentIDs(agentIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentStatistics", "", map[string]interface{}{
				"operation": "get_agent_statistics",
				"option":    "repo.GetMetricsByAgentIDs",
				"func_name": "service.agent.monitor.GetAgentStatistics",
				"count_ids": len(agentIDs),
			})
			return nil, err
		}

		window, err = s.agentRepo.GetMetricsByAgentIDsSince(agentIDs, since)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentStatistics", "", map[string]interface{}{
				"operation": "get_agent_statistics",
				"option":    "repo.GetMetricsByAgentIDsSince",
				"func_name": "service.agent.monitor.GetAgentStatistics",
				"count_ids": len(agentIDs),
				"since":     since,
			})
			return nil, err
		}
	} else {
		// 无标签过滤，查询所有
		all, err = s.agentRepo.GetAllMetrics()
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentStatistics", "", map[string]interface{}{
				"operation": "get_agent_statistics",
				"option":    "repo.GetAllMetrics",
				"func_name": "service.agent.monitor.GetAgentStatistics",
			})
			return nil, err
		}
		window, err = s.agentRepo.GetMetricsSince(since)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentStatistics", "", map[string]interface{}{
				"operation": "get_agent_statistics",
				"option":    "repo.GetMetricsSince",
				"func_name": "service.agent.monitor.GetAgentStatistics",
				"since":     since,
			})
			return nil, err
		}
	}

	total := int64(len(all))
	online := int64(0)
	onlineSet := make(map[string]struct{}, len(window))
	for _, m := range window {
		if m == nil {
			continue
		}
		onlineSet[m.AgentID] = struct{}{}
	}
	online = int64(len(onlineSet))
	offline := total - online

	// 分布统计与聚合
	workDist := map[string]int64{}
	scanDist := map[string]int64{}
	var cpuSum, memSum, diskSum float64
	var cpuMax, memMax, diskMax float64
	var cpuMin, memMin, diskMin float64
	var runningSum, completedSum, failedSum int64

	initialized := false
	for _, m := range all {
		if m == nil {
			continue
		}
		workDist[string(m.WorkStatus)] += 1
		st := m.ScanType
		if st == "" {
			st = "unknown"
		}
		scanDist[st] += 1

		cpuSum += m.CPUUsage
		memSum += m.MemoryUsage
		diskSum += m.DiskUsage
		runningSum += int64(m.RunningTasks)
		completedSum += int64(m.CompletedTasks)
		failedSum += int64(m.FailedTasks)

		if !initialized {
			cpuMax, memMax, diskMax = m.CPUUsage, m.MemoryUsage, m.DiskUsage
			cpuMin, memMin, diskMin = m.CPUUsage, m.MemoryUsage, m.DiskUsage
			initialized = true
		} else {
			if m.CPUUsage > cpuMax {
				cpuMax = m.CPUUsage
			}
			if m.MemoryUsage > memMax {
				memMax = m.MemoryUsage
			}
			if m.DiskUsage > diskMax {
				diskMax = m.DiskUsage
			}
			if m.CPUUsage < cpuMin {
				cpuMin = m.CPUUsage
			}
			if m.MemoryUsage < memMin {
				memMin = m.MemoryUsage
			}
			if m.DiskUsage < diskMin {
				diskMin = m.DiskUsage
			}
		}
	}

	avgDiv := func(sum float64, n int64) float64 {
		if n == 0 {
			return 0
		}
		return sum / float64(n)
	}
	aggregated := agentModel.AggregatedPerformance{
		CPUAvg: avgDiv(cpuSum, total), CPUMax: cpuMax, CPUMin: cpuMin,
		MemAvg: avgDiv(memSum, total), MemMax: memMax, MemMin: memMin,
		DiskAvg: avgDiv(diskSum, total), DiskMax: diskMax, DiskMin: diskMin,
		RunningTasksTotal: runningSum, CompletedTasksTotal: completedSum, FailedTasksTotal: failedSum,
	}

	resp := &agentModel.AgentStatisticsResponse{
		TotalAgents:            total,
		OnlineAgents:           online,
		OfflineAgents:          offline,
		WorkStatusDistribution: workDist,
		ScanTypeDistribution:   scanDist,
		Performance:            aggregated,
	}

	logger.LogInfo("Agent统计计算成功", "", 0, "", "service.agent.monitor.GetAgentStatistics", "", map[string]interface{}{
		"operation": "get_agent_statistics",
		"option":    "compose.response",
		"func_name": "service.agent.monitor.GetAgentStatistics",
		"total":     total,
		"online":    online,
	})

	return resp, nil
}

// GetAgentLoadBalance 负载均衡分析
func (s *agentMonitorService) GetAgentLoadBalance(windowSeconds int, topN int, tagIDs []uint64) (*agentModel.AgentLoadBalanceResponse, error) {
	if windowSeconds <= 0 {
		windowSeconds = 180
	}
	if topN <= 0 {
		topN = 5
	}
	since := time.Now().Add(-time.Duration(windowSeconds) * time.Second)

	var metrics []*agentModel.AgentMetrics
	var err error

	if len(tagIDs) > 0 {
		var agentIDs []string
		agentIDs, err = s.tagService.GetEntityIDsByTagIDs(context.Background(), "agent", tagIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentLoadBalance", "", map[string]interface{}{
				"operation": "get_agent_load_balance",
				"option":    "tagService.GetEntityIDsByTagIDs",
				"func_name": "service.agent.monitor.GetAgentLoadBalance",
				"tag_ids":   tagIDs,
			})
			return nil, err
		}

		if len(agentIDs) == 0 {
			return &agentModel.AgentLoadBalanceResponse{
				TopBusyAgents: []agentModel.AgentLoadItem{},
				TopIdleAgents: []agentModel.AgentLoadItem{},
				Advice:        "无Agent数据",
			}, nil
		}

		metrics, err = s.agentRepo.GetMetricsByAgentIDsSince(agentIDs, since)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentLoadBalance", "", map[string]interface{}{
				"operation": "get_agent_load_balance",
				"option":    "repo.GetMetricsByAgentIDsSince",
				"func_name": "service.agent.monitor.GetAgentLoadBalance",
				"count_ids": len(agentIDs),
				"since":     since,
			})
			return nil, err
		}
	} else {
		metrics, err = s.agentRepo.GetMetricsSince(since)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentLoadBalance", "", map[string]interface{}{
				"operation": "get_agent_load_balance",
				"option":    "repo.GetMetricsSince",
				"func_name": "service.agent.monitor.GetAgentLoadBalance",
				"since":     since,
			})
			return nil, err
		}
	}

	// 计算负载分数
	items := make([]agentModel.AgentLoadItem, 0, len(metrics))
	for _, m := range metrics {
		if m == nil {
			continue
		}
		score := 0.5*m.CPUUsage + 0.5*m.MemoryUsage + 5.0*float64(m.RunningTasks)
		items = append(items, agentModel.AgentLoadItem{
			AgentID: m.AgentID, CPUUsage: m.CPUUsage, MemoryUsage: m.MemoryUsage,
			RunningTasks: m.RunningTasks, LoadScore: score, WorkStatus: m.WorkStatus,
			ScanType: m.ScanType, Timestamp: m.Timestamp,
		})
	}

	// 排序
	sort.Slice(items, func(i, j int) bool { return items[i].LoadScore > items[j].LoadScore })
	limit := func(n int) int {
		if n < 0 {
			return 0
		}
		if n > len(items) {
			return len(items)
		}
		return n
	}
	busy := make([]agentModel.AgentLoadItem, 0)
	idle := make([]agentModel.AgentLoadItem, 0)
	if len(items) > 0 {
		n := limit(topN)
		busy = append(busy, items[:n]...)
		// 反向取最空闲
		sortedAsc := make([]agentModel.AgentLoadItem, len(items))
		copy(sortedAsc, items)
		sort.Slice(sortedAsc, func(i, j int) bool { return sortedAsc[i].LoadScore < sortedAsc[j].LoadScore })
		m := limit(topN)
		idle = append(idle, sortedAsc[:m]...)
	}

	advice := ""
	if len(busy) > 0 && len(idle) > 0 {
		advice = "优先将新任务调度到负载较低的Agent；对高负载Agent限流或延迟分配。"
	}

	resp := &agentModel.AgentLoadBalanceResponse{TopBusyAgents: busy, TopIdleAgents: idle, Advice: advice}
	logger.LogInfo("Agent负载均衡分析完成", "", 0, "", "service.agent.monitor.GetAgentLoadBalance", "", map[string]interface{}{
		"operation": "get_agent_load_balance",
		"option":    "compose.response",
		"func_name": "service.agent.monitor.GetAgentLoadBalance",
		"busy":      len(busy),
		"idle":      len(idle),
	})
	return resp, nil
}

// GetAgentPerformanceAnalysis 性能分析
func (s *agentMonitorService) GetAgentPerformanceAnalysis(windowSeconds int, topN int, tagIDs []uint64) (*agentModel.AgentPerformanceAnalysisResponse, error) {
	if windowSeconds <= 0 {
		windowSeconds = 180
	}
	if topN <= 0 {
		topN = 5
	}
	since := time.Now().Add(-time.Duration(windowSeconds) * time.Second)

	var metrics []*agentModel.AgentMetrics
	var err error

	if len(tagIDs) > 0 {
		var agentIDs []string
		agentIDs, err = s.tagService.GetEntityIDsByTagIDs(context.Background(), "agent", tagIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentPerformanceAnalysis", "", map[string]interface{}{
				"operation": "get_agent_performance",
				"option":    "tagService.GetEntityIDsByTagIDs",
				"func_name": "service.agent.monitor.GetAgentPerformanceAnalysis",
				"tag_ids":   tagIDs,
			})
			return nil, err
		}

		if len(agentIDs) == 0 {
			return &agentModel.AgentPerformanceAnalysisResponse{
				Aggregated:     agentModel.AggregatedPerformance{},
				TopCPU:         []agentModel.AgentPerformanceItem{},
				TopMemory:      []agentModel.AgentPerformanceItem{},
				TopNetwork:     []agentModel.AgentPerformanceItem{},
				TopFailedTasks: []agentModel.AgentPerformanceItem{},
			}, nil
		}

		metrics, err = s.agentRepo.GetMetricsByAgentIDsSince(agentIDs, since)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentPerformanceAnalysis", "", map[string]interface{}{
				"operation": "get_agent_performance",
				"option":    "repo.GetMetricsByAgentIDsSince",
				"func_name": "service.agent.monitor.GetAgentPerformanceAnalysis",
				"count_ids": len(agentIDs),
				"since":     since,
			})
			return nil, err
		}
	} else {
		metrics, err = s.agentRepo.GetMetricsSince(since)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentPerformanceAnalysis", "", map[string]interface{}{
				"operation": "get_agent_performance",
				"option":    "repo.GetMetricsSince",
				"func_name": "service.agent.monitor.GetAgentPerformanceAnalysis",
				"since":     since,
			})
			return nil, err
		}
	}

	// 聚合统计
	var cpuSum, memSum, diskSum float64
	var cpuMax, memMax, diskMax float64
	var cpuMin, memMin, diskMin float64
	var runningSum, completedSum, failedSum int64
	initialized := false

	// 排名列表容器
	type pair struct {
		item agentModel.AgentPerformanceItem
		key  float64
		key2 int64
	}
	cpuList := []pair{}
	memList := []pair{}
	netList := []pair{}
	failList := []pair{}

	for _, m := range metrics {
		if m == nil {
			continue
		}
		cpuSum += m.CPUUsage
		memSum += m.MemoryUsage
		diskSum += m.DiskUsage
		runningSum += int64(m.RunningTasks)
		completedSum += int64(m.CompletedTasks)
		failedSum += int64(m.FailedTasks)
		if !initialized {
			cpuMax, memMax, diskMax = m.CPUUsage, m.MemoryUsage, m.DiskUsage
			cpuMin, memMin, diskMin = m.CPUUsage, m.MemoryUsage, m.DiskUsage
			initialized = true
		} else {
			if m.CPUUsage > cpuMax {
				cpuMax = m.CPUUsage
			}
			if m.MemoryUsage > memMax {
				memMax = m.MemoryUsage
			}
			if m.DiskUsage > diskMax {
				diskMax = m.DiskUsage
			}
			if m.CPUUsage < cpuMin {
				cpuMin = m.CPUUsage
			}
			if m.MemoryUsage < memMin {
				memMin = m.MemoryUsage
			}
			if m.DiskUsage < diskMin {
				diskMin = m.DiskUsage
			}
		}

		perf := agentModel.AgentPerformanceItem{
			AgentID: m.AgentID, CPUUsage: m.CPUUsage, MemoryUsage: m.MemoryUsage, DiskUsage: m.DiskUsage,
			NetworkBytesSent: m.NetworkBytesSent, NetworkBytesRecv: m.NetworkBytesRecv, FailedTasks: m.FailedTasks,
			Timestamp: m.Timestamp,
		}
		cpuList = append(cpuList, pair{item: perf, key: m.CPUUsage})
		memList = append(memList, pair{item: perf, key: m.MemoryUsage})
		netList = append(netList, pair{item: perf, key: float64(m.NetworkBytesSent + m.NetworkBytesRecv)})
		failList = append(failList, pair{item: perf, key2: int64(m.FailedTasks)})
	}

	avgDiv := func(sum float64, n int64) float64 {
		if n == 0 {
			return 0
		}
		return sum / float64(n)
	}
	aggregated := agentModel.AggregatedPerformance{
		CPUAvg: avgDiv(cpuSum, int64(len(metrics))), CPUMax: cpuMax, CPUMin: cpuMin,
		MemAvg: avgDiv(memSum, int64(len(metrics))), MemMax: memMax, MemMin: memMin,
		DiskAvg: avgDiv(diskSum, int64(len(metrics))), DiskMax: diskMax, DiskMin: diskMin,
		RunningTasksTotal: runningSum, CompletedTasksTotal: completedSum, FailedTasksTotal: failedSum,
	}

	// 排序取TopN
	sortPairs := func(list []pair, less func(i, j int) bool) []agentModel.AgentPerformanceItem {
		sort.Slice(list, less)
		n := topN
		if n > len(list) {
			n = len(list)
		}
		out := make([]agentModel.AgentPerformanceItem, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, list[i].item)
		}
		return out
	}
	topCPU := sortPairs(cpuList, func(i, j int) bool { return cpuList[i].key > cpuList[j].key })
	topMem := sortPairs(memList, func(i, j int) bool { return memList[i].key > memList[j].key })
	topNet := sortPairs(netList, func(i, j int) bool { return netList[i].key > netList[j].key })
	sort.Slice(failList, func(i, j int) bool { return failList[i].key2 > failList[j].key2 })
	n := topN
	if n > len(failList) {
		n = len(failList)
	}
	topFail := make([]agentModel.AgentPerformanceItem, 0, n)
	for i := 0; i < n; i++ {
		topFail = append(topFail, failList[i].item)
	}

	resp := &agentModel.AgentPerformanceAnalysisResponse{
		Aggregated:     aggregated,
		TopCPU:         topCPU,
		TopMemory:      topMem,
		TopNetwork:     topNet,
		TopFailedTasks: topFail,
	}
	logger.LogInfo("Agent性能分析完成", "", 0, "", "service.agent.monitor.GetAgentPerformanceAnalysis", "", map[string]interface{}{
		"operation": "get_agent_performance",
		"option":    "compose.response",
		"func_name": "service.agent.monitor.GetAgentPerformanceAnalysis",
		"count":     len(metrics),
	})
	return resp, nil
}

// GetAgentCapacityAnalysis 容量分析
func (s *agentMonitorService) GetAgentCapacityAnalysis(windowSeconds int, cpuThr, memThr, diskThr float64, tagIDs []uint64) (*agentModel.AgentCapacityAnalysisResponse, error) {
	if windowSeconds <= 0 {
		windowSeconds = 180
	}
	if cpuThr <= 0 {
		cpuThr = 80
	}
	if memThr <= 0 {
		memThr = 80
	}
	if diskThr <= 0 {
		diskThr = 80
	}
	since := time.Now().Add(-time.Duration(windowSeconds) * time.Second)

	var metrics []*agentModel.AgentMetrics
	var err error

	if len(tagIDs) > 0 {
		var agentIDs []string
		agentIDs, err = s.tagService.GetEntityIDsByTagIDs(context.Background(), "agent", tagIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentCapacityAnalysis", "", map[string]interface{}{
				"operation": "get_agent_capacity",
				"option":    "tagService.GetEntityIDsByTagIDs",
				"func_name": "service.agent.monitor.GetAgentCapacityAnalysis",
				"tag_ids":   tagIDs,
			})
			return nil, err
		}

		if len(agentIDs) == 0 {
			return &agentModel.AgentCapacityAnalysisResponse{
				OnlineAgents:    0,
				Overloaded:      0,
				CPUThreshold:    cpuThr,
				MemThreshold:    memThr,
				DiskThreshold:   diskThr,
				AverageHeadroom: 0,
				CapacityScore:   0,
				Bottlenecks:     map[string]int64{"cpu": 0, "memory": 0, "disk": 0},
				Recommendations: "无Agent数据",
				OverloadedList:  []agentModel.AgentCapacityItem{},
			}, nil
		}

		metrics, err = s.agentRepo.GetMetricsByAgentIDsSince(agentIDs, since)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentCapacityAnalysis", "", map[string]interface{}{
				"operation": "get_agent_capacity",
				"option":    "repo.GetMetricsByAgentIDsSince",
				"func_name": "service.agent.monitor.GetAgentCapacityAnalysis",
				"count_ids": len(agentIDs),
				"since":     since,
			})
			return nil, err
		}
	} else {
		metrics, err = s.agentRepo.GetMetricsSince(since)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentCapacityAnalysis", "", map[string]interface{}{
				"operation": "get_agent_capacity",
				"option":    "repo.GetMetricsSince",
				"func_name": "service.agent.monitor.GetAgentCapacityAnalysis",
				"since":     since,
			})
			return nil, err
		}
	}

	var online int64
	var overloaded int64
	bottlenecks := map[string]int64{"cpu": 0, "memory": 0, "disk": 0}
	var headroomSum float64
	overloadedList := make([]agentModel.AgentCapacityItem, 0)

	for _, m := range metrics {
		if m == nil {
			continue
		}
		online++
		// 余量按三者最大值计算
		maxUsage := m.CPUUsage
		if m.MemoryUsage > maxUsage {
			maxUsage = m.MemoryUsage
		}
		if m.DiskUsage > maxUsage {
			maxUsage = m.DiskUsage
		}
		headroom := 100.0 - maxUsage
		if headroom < 0 {
			headroom = 0
		}
		headroomSum += headroom

		// 过载判定与原因
		reason := ""
		if m.CPUUsage >= cpuThr {
			bottlenecks["cpu"]++
			reason = "cpu"
		}
		if m.MemoryUsage >= memThr {
			bottlenecks["memory"]++
			if reason == "" {
				reason = "memory"
			}
		}
		if m.DiskUsage >= diskThr {
			bottlenecks["disk"]++
			if reason == "" {
				reason = "disk"
			}
		}
		if reason != "" {
			overloaded++
			overloadedList = append(overloadedList, agentModel.AgentCapacityItem{
				AgentID: m.AgentID, CPUUsage: m.CPUUsage, MemoryUsage: m.MemoryUsage, DiskUsage: m.DiskUsage, Reason: reason, Timestamp: m.Timestamp,
			})
		}
	}

	avgHeadroom := 0.0
	if online > 0 {
		avgHeadroom = headroomSum / float64(online)
	}
	capacityScore := avgHeadroom // 简化：余量均值即得分（0-100）
	advice := "容量总体健康"
	if capacityScore < 30 {
		advice = "容量紧张：建议立即扩容或限流"
	} else if capacityScore < 60 {
		advice = "容量一般：建议按需扩容并优化任务分配"
	}

	resp := &agentModel.AgentCapacityAnalysisResponse{
		OnlineAgents:    online,
		Overloaded:      overloaded,
		CPUThreshold:    cpuThr,
		MemThreshold:    memThr,
		DiskThreshold:   diskThr,
		AverageHeadroom: avgHeadroom,
		CapacityScore:   capacityScore,
		Bottlenecks:     bottlenecks,
		Recommendations: advice,
		OverloadedList:  overloadedList,
	}

	logger.LogInfo("Agent容量分析完成", "", 0, "", "service.agent.monitor.GetAgentCapacityAnalysis", "", map[string]interface{}{
		"operation":  "get_agent_capacity",
		"option":     "compose.response",
		"func_name":  "service.agent.monitor.GetAgentCapacityAnalysis",
		"online":     online,
		"overloaded": overloaded,
	})
	return resp, nil
}
