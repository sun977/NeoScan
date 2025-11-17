# 任务拆分文档 - Agent高级统计4接口

## 任务列表
1. 扩展模型：新增统计/分析响应结构（已完成）
2. 扩展仓储：GetAllMetrics / GetMetricsSince（已完成）
3. 扩展服务：四个分析方法实现（已完成）
4. 实现Handler：解析查询参数 + 返回（已完成）
5. 路由接线：四条路径指向Handler（已完成）
6. 构建验证：`go build -o neoMaster.exe ./cmd/master`（已完成）

## 验收标准
- 路由返回结构化 JSON 且无panic
- 参数可按需求影响计算窗口与阈值
- 构建通过，无未使用依赖