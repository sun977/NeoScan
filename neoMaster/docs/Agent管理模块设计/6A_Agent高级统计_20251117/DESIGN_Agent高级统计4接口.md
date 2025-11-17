# 设计文档 - Agent高级统计4接口

## 架构概览
- 数据来源：Master库 `agent_metrics`（单快照/每agent一条）
- 分层：Handler(analysis.go) → Service(monitor.go) → Repository(metrics.go)
- 路由接线：`/agent/statistics|load-balance|performance|capacity` → `AgentHandler` 对应方法

## 接口定义
- GET /api/v1/agent/statistics
  - 入参：`window_seconds`(int, 可选, 默认180)
  - 出参：`AgentStatisticsResponse`
- GET /api/v1/agent/load-balance
  - 入参：`window_seconds`、`top_n`(默认5)
  - 出参：`AgentLoadBalanceResponse`
- GET /api/v1/agent/performance
  - 入参：`window_seconds`、`top_n`(默认5)
  - 出参：`AgentPerformanceAnalysisResponse`
- GET /api/v1/agent/capacity
  - 入参：`window_seconds`、`cpu_threshold`、`memory_threshold`、`disk_threshold`(默认80)
  - 出参：`AgentCapacityAnalysisResponse`

## 数据口径
- 在线：`timestamp >= now - window_seconds`
- 负载评分：`0.5*CPU + 0.5*Memory + 5*RunningTasks`
- 容量余量：`100 - max(cpu,mem,disk)` 的均值作为 `capacity_score`

## 异常与日志
- Handler/Service 使用 `LogBusinessOperation` / `LogBusinessError`
- Repository 使用 `LogInfo` / `LogError`