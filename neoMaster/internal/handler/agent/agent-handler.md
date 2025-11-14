# Agent Handler 与路由映射说明

本文档用于对齐 neoMaster 中 Agent 相关 Handler 文件与路由 `internal/app/master/router/agent_routers.go` 的对应关系，明确每个路由的用途、当前接线状态（是否直接调用 Handler 方法还是仍为 Router 层占位符），以及后续迁移建议。

注意事项：
- 路由前缀统一为 `/api/v1`（在 Router 里通过 `v1 *gin.RouterGroup` 传入），下文均以 `/agent` 为根路径说明。
- 日志规范统一使用 `internal/pkg/logger` 包的 `LogBusinessOperation` 和 `LogBusinessError`。
- 层级调用严格遵循：Controller/Handler → Service → Repository → Database。

## 目录与文件角色

- agent_handler.go
  - 角色：AgentHandler 基类，包括构造函数和通用校验、错误码映射方法（`validateRegisterRequest`、`validateHeartbeatRequest`、`getErrorStatusCode`）。
  - 用途：为各具体 Handler 文件提供通用能力，避免重复实现。

- base.go（基础管理）
  - 方法：RegisterAgent、GetAgentInfo、GetAgentList、UpdateAgentStatus、DeleteAgent
  - 用途：Agent 基础信息的注册、查询、列表分页与过滤、状态更新、删除等，Master 端完全自给自足（数据库驱动）。

- base_metadata.go（分组与标签管理）
  - 方法：GetAgentTags、AddAgentTag、RemoveAgentTag、UpdateAgentTags
  - 用途：Agent 元数据管理（标签管理）。分组相关路由当前仍为 Router 层占位符，后续拟添加到本文件或独立文件。

- health.go（健康与心跳）
  - 方法：ProcessHeartbeat、HealthCheckAgent、PingAgent
  - 用途：公开心跳处理；健康检查、Ping 为占位实现，用于连通性与状态探测。

- metrics.go（性能指标管理）
  - 方法：GetAgentMetrics、GetAgentListAllMetrics、CreateAgentMetrics、UpdateAgentMetrics
  - 用途：Master 端读取/写入 Agent 性能快照（agent_metrics 表）。拉取动作（pull）路由目前为 Router 层占位符，后续对接 Service 并实现与 Agent 的交互。

- communication.go（通信与控制）
  - 方法：SendCommand、SyncConfig、GetCommandStatus、UpgradeVersion、ResetAgent
  - 用途：Master → Agent 通信控制的占位 Handler；当前路由仍指向 Router 层占位符，后续将替换为本文件中的 Handler 方法。

- agent_control.go（进程控制）
  - 方法：StartAgentProcess、StopAgentProcess、RestartAgentProcess、GetAgentRuntimeStatus
  - 用途：Agent 进程生命周期控制的占位 Handler；当前路由仍指向 Router 层占位符，后续将替换为本文件中的 Handler 方法。

- config_push.go（配置管理）
  - 方法：GetAgentConfig、UpdateAgentConfig
  - 用途：Agent 配置查询与更新的占位 Handler；当前路由仍指向 Router 层占位符，后续将替换为本文件中的 Handler 方法。

- task.go（任务管理）
  - 方法：GetAgentTasks、CreateAgentTask、GetAgentTaskByID、DeleteAgentTask
  - 用途：Agent 任务查询、创建、删除的占位 Handler；当前路由仍指向 Router 层占位符，后续将替换为本文件中的 Handler 方法。

- analysis.go（高级查询与统计）
  - 方法：GetAgentStatistics、GetAgentLoadBalance、GetAgentPerformanceAnalysis、GetAgentCapacityAnalysis
  - 用途：Master 端数据分析与统计的占位 Handler；当前路由仍指向 Router 层占位符，后续将替换为本文件中的 Handler 方法。

- instant_alerts.go（监控与告警）
  - 方法：GetAgentAlerts、CreateAgentAlert、UpdateAgentAlert、DeleteAgentAlert、GetAgentMonitorStatus、StartAgentMonitor、StopAgentMonitor
  - 用途：监控与告警的占位 Handler；当前路由仍指向 Router 层占位符，后续将替换为本文件中的 Handler 方法。

- log_collection.go（日志管理）
  - 方法：GetAgentLogs
  - 用途：Agent 日志查询的占位 Handler；当前路由仍指向 Router 层占位符，后续将替换为本文件中的 Handler 方法。

## 路由与 Handler 映射明细

以下按照 `internal/app/master/router/agent_routers.go` 的结构进行说明。

### 一、公开路由（无需认证）

- POST /agent/register
  - 当前映射：`r.agentHandler.RegisterAgent`（base.go）
  - 作用：Agent 注册（解析请求、校验、调用 Service 持久化）。
  - 状态：已接线（直接调用 Handler）。

- POST /agent/heartbeat
  - 当前映射：`r.agentHandler.ProcessHeartbeat`（health.go）
  - 作用：处理 Agent 心跳（解析、校验、调用监控 Service 入库/维护状态）。
  - 状态：已接线（直接调用 Handler）。

### 二、管理路由（需要认证与用户激活中间件）

1) 基础管理（Master 端完全独立实现）
- GET /agent
  - 当前映射：`r.agentHandler.GetAgentList`（base.go）
  - 作用：分页、状态过滤、关键字、标签、能力过滤。
  - 状态：已接线。
- GET /agent/:id
  - 当前映射：`r.agentHandler.GetAgentInfo`（base.go）
  - 作用：根据 ID 查询 Agent 详情。
  - 状态：已接线。
- PATCH /agent/:id/status
  - 当前映射：`r.agentHandler.UpdateAgentStatus`（base.go）
  - 作用：部分更新 Agent 状态字段。
  - 状态：已接线。
- DELETE /agent/:id
  - 当前映射：`r.agentHandler.DeleteAgent`（base.go）
  - 作用：删除 Agent。
  - 状态：已接线。

2) 进程控制（需要 Agent 端配合）
- POST /agent/:id/start
  - 当前映射：`r.agentStartPlaceholder`（Router 占位）
  - 目标映射：`h.StartAgentProcess`（agent_control.go）
  - 作用：启动 Agent 进程。
  - 状态：未接线（占位）。
- POST /agent/:id/stop
  - 当前映射：`r.agentStopPlaceholder`（Router 占位）
  - 目标映射：`h.StopAgentProcess`（agent_control.go）
  - 作用：停止 Agent 进程。
  - 状态：未接线（占位）。
- POST /agent/:id/restart
  - 当前映射：`r.agentRestartPlaceholder`（Router 占位）
  - 目标映射：`h.RestartAgentProcess`（agent_control.go）
  - 作用：重启 Agent 进程。
  - 状态：未接线（占位）。
- GET /agent/:id/status
  - 当前映射：`r.agentStatusPlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentRuntimeStatus`（agent_control.go）
  - 作用：查询 Agent 运行时状态。
  - 状态：未接线（占位）。

3) 配置管理（Master 存储 + Agent 应用）
- GET /agent/:id/config
  - 当前映射：`r.agentGetConfigPlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentConfig`（config_push.go）
  - 作用：查询 Agent 配置。
  - 状态：未接线（占位）。
- PUT /agent/:id/config
  - 当前映射：`r.agentUpdateConfigPlaceholder`（Router 占位）
  - 目标映射：`h.UpdateAgentConfig`（config_push.go）
  - 作用：更新 Agent 配置（Master 端存储，必要时推送到 Agent）。
  - 状态：未接线（占位）。

4) 任务管理（需要 Agent 端执行）
- GET /agent/:id/tasks
  - 当前映射：`r.agentGetTasksPlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentTasks`（task.go）
  - 作用：查询 Agent 正在执行的任务列表。
  - 状态：未接线（占位）。
- POST /agent/:id/tasks
  - 当前映射：`r.agentCreateTaskPlaceholder`（Router 占位）
  - 目标映射：`h.CreateAgentTask`（task.go）
  - 作用：为 Agent 创建/下发任务。
  - 状态：未接线（占位）。
- GET /agent/:id/tasks/:task_id
  - 当前映射：`r.agentGetTaskPlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentTaskByID`（task.go）
  - 作用：查询任务执行状态与进度。
  - 状态：未接线（占位）。
- DELETE /agent/:id/tasks/:task_id
  - 当前映射：`r.agentDeleteTaskPlaceholder`（Router 占位）
  - 目标映射：`h.DeleteAgentTask`（task.go）
  - 作用：取消/删除任务。
  - 状态：未接线（占位）。

5) 性能指标管理（Master 读库 + Agent 读接口）
- GET /agent/:id/metrics
  - 当前映射：`r.agentHandler.GetAgentMetrics`（metrics.go）
  - 作用：从 Master 库读取指定 Agent 的最新性能快照。
  - 状态：已接线。
- GET /agent/metrics
  - 当前映射：`r.agentHandler.GetAgentListAllMetrics`（metrics.go）
  - 作用：分页读取所有 Agent 的最新性能快照。
  - 状态：已接线。
- POST /agent/:id/metrics/pull
  - 当前映射：`r.agentPullMetricsPlaceholder`（Router 占位）
  - 目标映射：Service 层实现 `PullAndUpdateOne`，后续对接 Handler。
  - 作用：主动从 Agent 拉取该 Agent 的实时性能并写入 Master。
  - 状态：未接线（占位）。
- POST /agent/metrics/pull
  - 当前映射：`r.agentBatchPullMetricsPlaceholder`（Router 占位）
  - 目标映射：Service 层实现 `BatchPullAndUpdate`，后续对接 Handler。
  - 作用：批量从所有 Agent 拉取性能并写入 Master。
  - 状态：未接线（占位）。
- POST /agent/:id/metrics
  - 当前映射：`r.agentHandler.CreateAgentMetrics`（metrics.go）
  - 作用：创建/上报性能快照（保留，受限权限）。
  - 状态：已接线。
- PUT /agent/:id/metrics
  - 当前映射：`r.agentHandler.UpdateAgentMetrics`（metrics.go）
  - 作用：更新性能快照（手动修复/回填最新快照）。
  - 状态：已接线。

6) 高级查询与统计（Master 端独立实现）
- GET /agent/statistics
  - 当前映射：`r.agentGetStatisticsPlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentStatistics`（analysis.go）
  - 作用：统计在线数量、状态分布、性能统计聚合等。
  - 状态：未接线（占位）。
- GET /agent/load-balance
  - 当前映射：`r.agentGetLoadBalancePlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentLoadBalance`（analysis.go）
  - 作用：任务分配与资源使用率的负载均衡分析。
  - 状态：未接线（占位）。
- GET /agent/performance
  - 当前映射：`r.agentGetPerformancePlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentPerformanceAnalysis`（analysis.go）
  - 作用：性能分析与趋势。
  - 状态：未接线（占位）。
- GET /agent/capacity
  - 当前映射：`r.agentGetCapacityPlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentCapacityAnalysis`（analysis.go）
  - 作用：容量分析与扩容建议。
  - 状态：未接线（占位）。

7) 分组和标签管理（Master 端独立实现）
- 说明：当前 Router 提供分组相关路由（GET/POST/PUT/DELETE 以及 Agent 与分组的关联/移除）均为 Router 层占位符；标签相关路由已接线到 `base_metadata.go`。
- GET /agent/:id/tags → `r.agentHandler.GetAgentTags`（base_metadata.go）
- POST /agent/:id/tags → `r.agentHandler.AddAgentTag`（base_metadata.go）
- PUT /agent/:id/tags → `r.agentHandler.UpdateAgentTags`（base_metadata.go）
- DELETE /agent/:id/tags → `r.agentHandler.RemoveAgentTag`（base_metadata.go）

8) 通信与控制（需要 Agent 端配合）
- POST /agent/:id/command
  - 当前映射：`r.agentSendCommandPlaceholder`（Router 占位）
  - 目标映射：`h.SendCommand`（communication.go）
  - 作用：向 Agent 发送控制命令。
  - 状态：未接线（占位）。
- GET /agent/:id/command/:cmd_id
  - 当前映射：`r.agentGetCommandStatusPlaceholder`（Router 占位）
  - 目标映射：`h.GetCommandStatus`（communication.go）
  - 作用：获取命令执行状态。
  - 状态：未接线（占位）。
- POST /agent/:id/sync
  - 当前映射：`r.agentSyncConfigPlaceholder`（Router 占位）
  - 目标映射：`h.SyncConfig`（communication.go）
  - 作用：同步配置到 Agent。
  - 状态：未接线（占位）。
- POST /agent/:id/upgrade
  - 当前映射：`r.agentUpgradePlaceholder`（Router 占位）
  - 目标映射：`h.UpgradeVersion`（communication.go）
  - 作用：升级 Agent 版本。
  - 状态：未接线（占位）。
- POST /agent/:id/reset
  - 当前映射：`r.agentResetPlaceholder`（Router 占位）
  - 目标映射：`h.ResetAgent`（communication.go）
  - 作用：重置 Agent 配置。
  - 状态：未接线（占位）。

9) 监控与告警（需要 Agent 端配合）
- GET /agent/:id/alerts
  - 当前映射：`r.agentGetAlertsPlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentAlerts`（instant_alerts.go）
  - 作用：查询告警列表（Master 存储 + Agent 实时）。
  - 状态：未接线（占位）。
- POST /agent/:id/alerts
  - 当前映射：`r.agentCreateAlertPlaceholder`（Router 占位）
  - 目标映射：`h.CreateAgentAlert`（instant_alerts.go）
  - 作用：创建告警规则（Master 存储）。
  - 状态：未接线（占位）。
- PUT /agent/:id/alerts/:alert_id
  - 当前映射：`r.agentUpdateAlertPlaceholder`（Router 占位）
  - 目标映射：`h.UpdateAgentAlert`（instant_alerts.go）
  - 作用：更新告警规则（Master 存储）。
  - 状态：未接线（占位）。
- DELETE /agent/:id/alerts/:alert_id
  - 当前映射：`r.agentDeleteAlertPlaceholder`（Router 占位）
  - 目标映射：`h.DeleteAgentAlert`（instant_alerts.go）
  - 作用：删除告警规则（Master 存储）。
  - 状态：未接线（占位）。
- GET /agent/:id/monitor
  - 当前映射：`r.agentGetMonitorPlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentMonitorStatus`（instant_alerts.go）
  - 作用：获取 Agent 监控状态。
  - 状态：未接线（占位）。
- POST /agent/:id/monitor/start
  - 当前映射：`r.agentStartMonitorPlaceholder`（Router 占位）
  - 目标映射：`h.StartAgentMonitor`（instant_alerts.go）
  - 作用：启动监控。
  - 状态：未接线（占位）。
- POST /agent/:id/monitor/stop
  - 当前映射：`r.agentStopMonitorPlaceholder`（Router 占位）
  - 目标映射：`h.StopAgentMonitor`（instant_alerts.go）
  - 作用：停止监控。
  - 状态：未接线（占位）。

10) 日志管理（混合实现：Master 收集 + Agent 实时）
- GET /agent/:id/logs
  - 当前映射：`r.agentGetLogsPlaceholder`（Router 占位）
  - 目标映射：`h.GetAgentLogs`（log_collection.go）
  - 作用：查询 Agent 日志（Master 存储或实时拉取）。
  - 状态：未接线（占位）。

11) 健康检查（混合实现）
- GET /agent/:id/health
  - 当前映射：`r.agentHealthCheckPlaceholder`（Router 占位）
  - 目标映射：`h.HealthCheckAgent`（health.go）
  - 作用：Agent 健康检查。
  - 状态：未接线（占位）。
- GET /agent/:id/ping
  - 当前映射：`r.agentPingPlaceholder`（Router 占位）
  - 目标映射：`h.PingAgent`（health.go）
  - 作用：Agent 连通性检查。
  - 状态：未接线（占位）。

## 迁移与接线建议

- 优先级建议：
  1) 进程控制（agent_control.go）与通信控制（communication.go）是与 Agent 交互的基础能力，建议先将 Router 占位符替换为对应 Handler 方法，并在 Service 层引入 Agent HTTP/消息通道客户端。
  2) 配置管理（config_push.go）与任务管理（task.go）可在完成基础通信后接线，统一走 Service。
  3) 监控与告警（instant_alerts.go）与性能拉取（metrics pull）作为增强能力，后续接线。

- 实施要点：
  - 保持日志字段规范：path（c.Request.URL.String）、operation、option、func_name、method、user_agent、agent_id 等。
  - 错误码映射统一复用 `getErrorStatusCode`，避免各文件重复逻辑。
  - 严格遵守分层：Handler 仅负责入参解析、调用 Service、出参封装，不直接操作 DB。

- 测试与配置：
  - 测试环境需提供 `.env` 的 JWT Secret 与数据库连接，确保 `go test ./...` 可执行基础集成测试。
  - 对新增路由接线，补充对应的测试用例，存放在 `test/日期目录`（例如 `test/20251107`）。

## 总结

当前基础管理、心跳与性能快照读写已完成接线，其余路由仍以 Router 层占位符存在，对应的 Handler 已在 `internal/handler/agent` 中提供占位实现。本文档明确了每条路由的用途与目标映射，便于后续逐步替换 Router 占位符为具体 Handler 方法，提升代码聚合度与可维护性。