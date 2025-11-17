# 需求对齐文档 - Agent高级统计4接口

## 原始需求
- 实现并接线以下4个路由，数据全部来自 `agent_metrics` 表：
  - GET /api/v1/agent/statistics
  - GET /api/v1/agent/load-balance
  - GET /api/v1/agent/performance
  - GET /api/v1/agent/capacity
- 控制器逻辑位于 `internal/handler/agent/analysis.go`，服务逻辑扩展于 `internal/service/agent/monitor.go`，仓储扩展于 `internal/repo/mysql/agent/metrics.go`。

## 项目上下文
- 技术栈：Go + Gin + GORM + MySQL
- 分层：Handler → Service → Repository → Database（严格遵守）
- 日志：Handler/Service 使用 `LogBusinessOperation`/`LogBusinessError`；Repository 使用 `LogInfo`/`LogError`
- agent_metrics 为“单快照模型”：每个 agent_id 维护一条最新记录

## 统计口径与约束
- 在线判定：`metrics.timestamp >= now - window_seconds` 即视为在线；默认 `window_seconds=180`
- 状态分布：按 `work_status` 聚合（idle/working/exception），与“在线/离线”无关
- 扫描类型分布：按 `scan_type` 聚合（空字符串计入 unknown）
- 性能统计：对当前快照做 `avg/min/max`，任务计数做 `sum`
- 负载评分（load_score）：`0.5*CPU + 0.5*Memory + 5*RunningTasks`（可微调）
- 容量阈值：CPU/Mem/Disk 超过阈值（默认 80%）即判定为过载；容量得分根据平均余量估算

## 边界确认（Out of Scope）
- 不引入历史时序分析（当前表仅单快照）；“趋势”用当前分布与Top列表替代
- 不跨表读取 `agents`（全部基于 `agent_metrics`）

## 疑问与决策
- 响应时间/吞吐量：`agent_metrics` 无直观字段，暂以 CPU/Memory/NetworkBytes 的分布与Top替代，并在返回与文档中明确口径

## 验收标准
- 四个路由返回结构化 JSON，遵循统一响应 `system.APIResponse`
- 支持基础查询参数：`window_seconds`、`top_n`、`cpu_threshold`、`memory_threshold`、`disk_threshold`
- 构建通过：`cd neoMaster && go build cmd/master -o neoMaster.exe`