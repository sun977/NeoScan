/**
 * @title: AgentMetricsRepository
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent 性能指标管理实现
 * @func: 提供Agent性能指标的CRUD操作，不包含业务逻辑
 * - CreateMetrics 创建Agent性能指标记录
 * - GetLatestMetrics 获取Agent最新的性能指标（每个Agent唯一快照）
 * - GetMetricsList 获取Agent性能指标列表（分页查询）
 * - UpdateAgentMetrics 更新Agent性能指标记录
 * 重构说明：agent.go 中分离出
 * - 统一使用 logger.LogInfo 和 logger.LogError 进行结构化日志记录
 * - 对齐模型字段：使用 BaseModel 的 CreatedAt/UpdatedAt 以及 AgentMetrics.Timestamp
 */
package agent

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
)

// ============== Metrics 状态监测功能实现 ==============
// CreateMetrics 创建Agent性能指标记录
// 说明：
// - 校验 metrics 与 AgentID
// - 统一设置 Timestamp 为当前时间；CreatedAt/UpdatedAt 交由 GORM 自动维护
func (r *agentRepository) CreateMetrics(metrics *agentModel.AgentMetrics) error {
	// 校验 metrics 与 AgentID
	if metrics == nil || metrics.AgentID == "" {
		logger.LogError(
			fmt.Errorf("metrics is invalid"),
			"", 0, "",
			"repo.mysql.agent.CreateMetrics", "gorm",
			map[string]interface{}{
				"operation": "create_metrics",
				"option":    "validate.metrics",
				"func_name": "repo.mysql.agent.CreateMetrics",
			},
		)
		return gorm.ErrInvalidData
	}

	// 设置指标时间戳（使用模型定义的 Timestamp 字段）
	metrics.Timestamp = time.Now()

	if err := r.db.Create(metrics).Error; err != nil {
		logger.LogError(
			err,
			"", 0, "",
			"repo.mysql.agent.CreateMetrics", "gorm",
			map[string]interface{}{
				"operation": "create_metrics",
				"option":    "db.Create(agent_metrics)",
				"func_name": "repo.mysql.agent.CreateMetrics",
				"agent_id":  metrics.AgentID,
			},
		)
		return err
	}

	logger.LogInfo(
		"Agent metrics created successfully",
		"", 0, "",
		"repo.mysql.agent.CreateMetrics", "gorm",
		map[string]interface{}{
			"operation": "create_metrics",
			"option":    "result.success",
			"func_name": "repo.mysql.agent.CreateMetrics",
			"agent_id":  metrics.AgentID,
		},
	)
	return nil
}

// GetLatestMetrics 获取Agent最新的性能指标（每个Agent唯一快照）
// 说明：
// - 模型中 AgentID 为 uniqueIndex，按设计每个Agent仅保留一条最新快照
// - 因此直接按 agent_id 查询第一条即可，无需 create_time 排序
// - 若未来切换到“历史时序模式”，推荐改为按 Timestamp DESC 排序
func (r *agentRepository) GetLatestMetrics(agentID string) (*agentModel.AgentMetrics, error) {
	// 校验 agentID
	if agentID == "" {
		logger.LogError(
			fmt.Errorf("agentID is empty"),
			"", 0, "",
			"repo.mysql.agent.GetLatestMetrics", "gorm",
			map[string]interface{}{
				"operation": "get_latest_metrics",
				"option":    "validate.agentID",
				"func_name": "repo.mysql.agent.GetLatestMetrics",
			},
		)
		return nil, gorm.ErrInvalidData
	}

	var metrics agentModel.AgentMetrics
	err := r.db.Where("agent_id = ?", agentID).
		Order("timestamp DESC").
		First(&metrics).Error
	if err != nil {
		logger.LogError(
			err,
			"", 0, "",
			"repo.mysql.agent.GetLatestMetrics", "gorm",
			map[string]interface{}{
				"operation": "get_latest_metrics",
				"option":    "db.First(agent_metrics)",
				"func_name": "repo.mysql.agent.GetLatestMetrics",
				"agent_id":  agentID,
			},
		)
		return nil, err
	}
	return &metrics, nil
}

// GetMetricsList 性能指标批量查询（分页 + 过滤）
// 说明：
// - 支持按 work_status、scan_type 和 agent_id 关键词过滤
// - 统一按 UpdatedAt DESC（最新更新时间）排序
func (r *agentRepository) GetMetricsList(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetrics, int64, error) {
	var metricsList []*agentModel.AgentMetrics
	var total int64

	// 防御性处理：页码与页大小需为正数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	query := r.db.Model(&agentModel.AgentMetrics{})

	if workStatus != nil {
		query = query.Where("work_status = ?", *workStatus)
	}
	if scanType != nil {
		query = query.Where("scan_type = ?", *scanType)
	}
	if keyword != nil && *keyword != "" {
		like := fmt.Sprintf("%%%s%%", *keyword)
		// AgentMetrics 没有 remark 字段，这里仅使用 agent_id 关键词匹配
		query = query.Where("agent_id LIKE ?", like)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetMetricsList", "gorm", map[string]interface{}{
			"operation": "get_metrics_list",
			"option":    "agentRepository.GetMetricsList.count",
			"func_name": "repo.agent.GetMetricsList",
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
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据，按更新时间戳倒序，必要时可增加次级排序保证稳定性
	if err := query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&metricsList).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetMetricsList", "gorm", map[string]interface{}{
			"operation": "get_metrics_list",
			"option":    "agentRepository.GetMetricsList.query",
			"func_name": "repo.agent.GetMetricsList",
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
		return nil, 0, err
	}

	logger.LogInfo("Agent metrics list retrieved successfully", "", 0, "", "repo.agent.GetMetricsList", "", map[string]interface{}{
		"operation": "get_metrics_list",
		"option":    "agentRepository.GetMetricsList",
		"func_name": "repo.agent.GetMetricsList",
		"count":     len(metricsList),
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

	return metricsList, total, nil
}

// UpdateAgentMetrics 更新指定Agent的最新性能指标（通常用于心跳或状态更新）
// 说明：
// - 若该Agent不存在指标记录，则创建；否则更新唯一快照
// - 统一更新 Timestamp 为当前时间
func (r *agentRepository) UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
	// 校验 agentID
	if agentID == "" || metrics == nil {
		logger.LogError(
			fmt.Errorf("agentID or metrics invalid"),
			"", 0, "",
			"repo.mysql.agent.UpdateAgentMetrics", "gorm",
			map[string]interface{}{
				"operation": "update_metrics",
				"option":    "validate.input",
				"func_name": "repo.mysql.agent.UpdateAgentMetrics",
			},
		)
		return gorm.ErrInvalidData
	}

	// 设置 AgentID 和 Timestamp
	metrics.AgentID = agentID
	metrics.Timestamp = time.Now()

	// 查找是否已存在该Agent的性能指标记录
	var existingMetrics agentModel.AgentMetrics
	result := r.db.Where("agent_id = ?", agentID).First(&existingMetrics)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// 不存在记录，创建新记录
			createResult := r.db.Create(metrics)
			if createResult.Error != nil {
				logger.LogError(createResult.Error, "", 0, "", "repo.agent.UpdateAgentMetrics", "gorm", map[string]interface{}{
					"operation": "update_metrics",
					"option":    "create_metrics",
					"func_name": "repo.mysql.agent.UpdateAgentMetrics",
					"agent_id":  agentID,
				})
				return createResult.Error
			}

			logger.LogInfo("Agent metrics created successfully", "", 0, "", "repo.agent.UpdateAgentMetrics", "gorm", map[string]interface{}{
				"operation": "create_agent_metrics",
				"option":    "create_metrics",
				"func_name": "repo.mysql.agent.UpdateAgentMetrics",
				"agent_id":  agentID,
			})
		} else {
			// 查询出错
			logger.LogError(result.Error, "", 0, "", "repo.agent.UpdateAgentMetrics", "gorm", map[string]interface{}{
				"operation": "query_agent_metrics",
				"option":    "query_metrics",
				"func_name": "repo.mysql.agent.UpdateAgentMetrics",
				"agent_id":  agentID,
			})
			return result.Error
		}
	} else {
		// 存在记录，更新现有记录
		// 构建更新字段映射，避免空指针访问
		updateFields := map[string]interface{}{
			"cpu_usage":          metrics.CPUUsage,
			"memory_usage":       metrics.MemoryUsage,
			"disk_usage":         metrics.DiskUsage,
			"network_bytes_sent": metrics.NetworkBytesSent,
			"network_bytes_recv": metrics.NetworkBytesRecv,
			"active_connections": metrics.ActiveConnections,
			"running_tasks":      metrics.RunningTasks,
			"completed_tasks":    metrics.CompletedTasks,
			"failed_tasks":       metrics.FailedTasks,
			"work_status":        metrics.WorkStatus,
			"scan_type":          metrics.ScanType,
			"timestamp":          metrics.Timestamp,
		}

		// 安全处理PluginStatus字段 这是一个 JSON 字段 {"nmap": {"pid": 12345, "status": "running"}, "nuclei": {"status": "idle"}}
		if metrics.PluginStatus != nil {
			updateFields["plugin_status"] = metrics.PluginStatus
		}

		updateResult := r.db.Model(&existingMetrics).Updates(updateFields)

		if updateResult.Error != nil {
			logger.LogError(updateResult.Error, "", 0, "", "repo.agent.UpdateAgentMetrics", "gorm", map[string]interface{}{
				"operation": "update_agent_metrics",
				"option":    "update_metrics",
				"func_name": "repo.mysql.agent.UpdateAgentMetrics",
				"agent_id":  agentID,
			})
			return updateResult.Error
		}

		logger.LogInfo("Agent metrics updated successfully", "", 0, "", "repo.agent.UpdateAgentMetrics", "gorm", map[string]interface{}{
			"operation": "update_agent_metrics",
			"option":    "update_metrics",
			"func_name": "repo.mysql.agent.UpdateAgentMetrics",
			"agent_id":  agentID,
		})
	}
	return nil
}

// ============== Metrics 数据分析功能实现 ==============
// GetAllMetrics 获取所有Agent的最新性能快照（单表全量）
// 说明：当前为单快照模型（每个agent_id一条记录），全量读取用于Master端聚合分析
func (r *agentRepository) GetAllMetrics() ([]*agentModel.AgentMetrics, error) {
	var list []*agentModel.AgentMetrics
	if err := r.db.Model(&agentModel.AgentMetrics{}).Order("timestamp DESC").Find(&list).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetAllMetrics", "gorm", map[string]interface{}{
			"operation": "get_all_metrics",
			"option":    "db.Find(agent_metrics)",
			"func_name": "repo.agent.GetAllMetrics",
		})
		return nil, err
	}
	logger.LogInfo("All agent metrics retrieved", "", 0, "", "repo.agent.GetAllMetrics", "gorm", map[string]interface{}{
		"operation": "get_all_metrics",
		"option":    "result.success",
		"func_name": "repo.agent.GetAllMetrics",
		"count":     len(list),
	})
	return list, nil
}

// GetMetricsSince 获取指定时间窗口内的快照（timestamp >= since）
// 用于“在线”判定与窗口内分析
func (r *agentRepository) GetMetricsSince(since time.Time) ([]*agentModel.AgentMetrics, error) {
	var list []*agentModel.AgentMetrics
	if err := r.db.Model(&agentModel.AgentMetrics{}).Where("timestamp >= ?", since).Order("timestamp DESC").Find(&list).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetMetricsSince", "gorm", map[string]interface{}{
			"operation": "get_metrics_since",
			"option":    "db.Find(agent_metrics)",
			"func_name": "repo.agent.GetMetricsSince",
			"since":     since,
		})
		return nil, err
	}
	logger.LogInfo("Window agent metrics retrieved", "", 0, "", "repo.agent.GetMetricsSince", "gorm", map[string]interface{}{
		"operation": "get_metrics_since",
		"option":    "result.success",
		"func_name": "repo.agent.GetMetricsSince",
		"count":     len(list),
		"since":     since,
	})
	return list, nil
}

// GetMetricsByAgentIDs 按代理ID集合过滤获取快照
func (r *agentRepository) GetMetricsByAgentIDs(agentIDs []string) ([]*agentModel.AgentMetrics, error) {
    var list []*agentModel.AgentMetrics
    if len(agentIDs) == 0 {
        return list, nil
    }
    if err := r.db.Model(&agentModel.AgentMetrics{}).Where("agent_id IN ?", agentIDs).Order("timestamp DESC").Find(&list).Error; err != nil {
        logger.LogError(err, "", 0, "", "repo.agent.GetMetricsByAgentIDs", "gorm", map[string]interface{}{
            "operation": "get_metrics_by_agent_ids",
            "option":    "db.Find(agent_metrics)",
            "func_name": "repo.agent.GetMetricsByAgentIDs",
            "count_ids": len(agentIDs),
        })
        return nil, err
    }
    logger.LogInfo("Metrics by agent IDs retrieved", "", 0, "", "repo.agent.GetMetricsByAgentIDs", "gorm", map[string]interface{}{
        "operation": "get_metrics_by_agent_ids",
        "option":    "result.success",
        "func_name": "repo.agent.GetMetricsByAgentIDs",
        "count_ids": len(agentIDs),
        "count":     len(list),
    })
    return list, nil
}

// GetMetricsByAgentIDsSince 按代理ID集合+时间窗口过滤获取快照
func (r *agentRepository) GetMetricsByAgentIDsSince(agentIDs []string, since time.Time) ([]*agentModel.AgentMetrics, error) {
    var list []*agentModel.AgentMetrics
    if len(agentIDs) == 0 {
        return list, nil
    }
    if err := r.db.Model(&agentModel.AgentMetrics{}).
        Where("agent_id IN ? AND timestamp >= ?", agentIDs, since).
        Order("timestamp DESC").Find(&list).Error; err != nil {
        logger.LogError(err, "", 0, "", "repo.agent.GetMetricsByAgentIDsSince", "gorm", map[string]interface{}{
            "operation": "get_metrics_by_agent_ids_since",
            "option":    "db.Find(agent_metrics)",
            "func_name": "repo.agent.GetMetricsByAgentIDsSince",
            "count_ids": len(agentIDs),
            "since":     since,
        })
        return nil, err
    }
    logger.LogInfo("Metrics by agent IDs since retrieved", "", 0, "", "repo.agent.GetMetricsByAgentIDsSince", "gorm", map[string]interface{}{
        "operation": "get_metrics_by_agent_ids_since",
        "option":    "result.success",
        "func_name": "repo.agent.GetMetricsByAgentIDsSince",
        "count_ids": len(agentIDs),
        "since":     since,
        "count":     len(list),
    })
    return list, nil
}
