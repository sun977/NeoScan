# Agent 高级统计与分析 API 说明

本说明基于现有路由与处理函数实现，覆盖 4 个只读分析接口：统计、负载均衡、性能分析、容量分析。接口均要求用户已登录并处于激活状态，鉴权通过 JWT Bearer 令牌。

- 路由注册位置：`internal/app/master/router/agent_routers.go:67-70`
- Handler 实现：
  - 统计：`internal/handler/agent/analysis.go:24-66`
  - 负载均衡：`internal/handler/agent/analysis.go:68-117`
  - 性能分析：`internal/handler/agent/analysis.go:119-169`
  - 容量分析：`internal/handler/agent/analysis.go:171-236`
- Service 计算逻辑：`internal/service/agent/monitor.go:363-861`

## 基本信息
- 基础路径：`/api/v1`
- 模块前缀：`/agent`
- 鉴权方式：`Authorization: Bearer <access_token>`
- 统一响应包装：`system.APIResponse`（`internal/model/system/response.go:55-63`）
  - `code` int
  - `status` string，通常为 `success` 或 `error/failed`
  - `message` string
  - `data` any（各接口自定义结构）
  - `error` string（错误信息，可选）

---

## 1) 获取Agent统计信息
- 方法与路径：`GET /api/v1/agent/statistics`
- 代码位置：
  - 路由：`internal/app/master/router/agent_routers.go:67`
  - Handler：`internal/handler/agent/analysis.go:24-66`
  - Service：`internal/service/agent/monitor.go:363-506`

### 查询参数
- `window_seconds` int，默认 180，统计窗口（秒）
- `group_id` string，可选，限定统计到指定分组

### 成功响应 `data` 结构（节选）
- `total_agents` int64：快照中的Agent数量
- `online_agents` int64：窗口内在线数量
- `offline_agents` int64：离线数量
- `work_status_distribution` map：工作状态分布
- `scan_type_distribution` map：扫描类型分布
- `performance`：聚合性能（CPU/内存/磁盘均值/极值、任务总数）

### 示例
请求：
```
GET /api/v1/agent/statistics?window_seconds=300&group_id=core
Authorization: Bearer <access_token>
```

响应：
```json
{
  "code": 200,
  "status": "success",
  "message": "OK",
  "data": {
    "total_agents": 12,
    "online_agents": 10,
    "offline_agents": 2,
    "work_status_distribution": {"idle": 6, "working": 4, "exception": 0},
    "scan_type_distribution": {"fullPortScan": 3, "webScan": 2, "idle": 7},
    "performance": {
      "cpu_avg": 23.5, "cpu_max": 88.1, "cpu_min": 1.2,
      "memory_avg": 41.3, "memory_max": 92.0, "memory_min": 8.3,
      "disk_avg": 58.1, "disk_max": 79.5, "disk_min": 15.2,
      "running_tasks_total": 5, "completed_tasks_total": 132, "failed_tasks_total": 3
    }
  }
}
```

---

## 2) 获取Agent负载均衡信息
- 方法与路径：`GET /api/v1/agent/load-balance`
- 代码位置：
  - 路由：`internal/app/master/router/agent_routers.go:68`
  - Handler：`internal/handler/agent/analysis.go:68-117`
  - Service：`internal/service/agent/monitor.go:508-590`

### 查询参数
- `window_seconds` int，默认 180，统计窗口（秒）
- `top_n` int，默认 5，返回TopN数量
- `group_id` string，可选，限定到分组

### 成功响应 `data` 结构（节选）
- `top_busy_agents`[ ]：高负载TopN列表（含 `load_score`、`running_tasks`、`work_status` 等）
- `top_idle_agents`[ ]：低负载TopN列表
- `advice` string：调度建议

### 示例
请求：
```
GET /api/v1/agent/load-balance?window_seconds=300&top_n=3
Authorization: Bearer <access_token>
```

响应：
```json
{
  "code": 200,
  "status": "success",
  "message": "OK",
  "data": {
    "top_busy_agents": [
      {"agent_id": "a-01", "cpu_usage": 82.3, "memory_usage": 77.2, "running_tasks": 3, "load_score": 171.0, "work_status": "working", "scan_type": "fullPortScan", "timestamp": "2025-11-17T08:00:00Z"}
    ],
    "top_idle_agents": [
      {"agent_id": "a-07", "cpu_usage": 3.2, "memory_usage": 8.1, "running_tasks": 0, "load_score": 5.7, "work_status": "idle", "scan_type": "idle", "timestamp": "2025-11-17T08:00:00Z"}
    ],
    "advice": "优先将新任务调度到负载较低的Agent；对高负载Agent限流或延迟分配。"
  }
}
```

---

## 3) 获取Agent性能分析
- 方法与路径：`GET /api/v1/agent/performance`
- 代码位置：
  - 路由：`internal/app/master/router/agent_routers.go:69`
  - Handler：`internal/handler/agent/analysis.go:119-169`
  - Service：`internal/service/agent/monitor.go:592-739`

### 查询参数
- `window_seconds` int，默认 180
- `top_n` int，默认 5
- `group_id` string，可选

### 成功响应 `data` 结构（节选）
- `aggregated`：聚合性能（均值/极值、任务总数）
- `top_cpu`[ ]：CPU使用率TopN
- `top_memory`[ ]：内存使用率TopN
- `top_network`[ ]：网络字节（发送+接收）TopN
- `top_failed`[ ]：失败任务数TopN

### 示例
```json
{
  "code": 200,
  "status": "success",
  "message": "OK",
  "data": {
    "aggregated": {"cpu_avg": 31.2, "cpu_max": 90.4, "cpu_min": 2.1, "memory_avg": 49.5, "memory_max": 93.1, "memory_min": 7.4, "disk_avg": 55.0, "disk_max": 82.9, "disk_min": 20.3, "running_tasks_total": 7, "completed_tasks_total": 280, "failed_tasks_total": 5},
    "top_cpu": [{"agent_id": "a-03", "cpu_usage": 90.4, "memory_usage": 71.3, "disk_usage": 62.1, "network_bytes_sent": 12400, "network_bytes_recv": 5600, "failed_tasks": 0, "timestamp": "2025-11-17T08:00:00Z"}]
  }
}
```

---

## 4) 获取Agent容量分析
- 方法与路径：`GET /api/v1/agent/capacity`
- 代码位置：
  - 路由：`internal/app/master/router/agent_routers.go:70`
  - Handler：`internal/handler/agent/analysis.go:171-236`
  - Service：`internal/service/agent/monitor.go:741-861`

### 查询参数
- `window_seconds` int，默认 180
- `group_id` string，可选
- `cpu_threshold` number，默认 80
- `memory_threshold` number，默认 80
- `disk_threshold` number，默认 80

### 成功响应 `data` 结构（节选）
- `online_agents` int64：在线数量
- `overloaded_agents` int64：过载数量
- `average_headroom` float64：平均余量（100 - max(cpu,mem,disk)）
- `capacity_score` float64：容量评分（简化为平均余量）
- `bottlenecks` map：瓶颈计数（cpu/memory/disk）
- `recommendations` string：扩容建议
- `overloaded_list`[ ]：过载明细

### 示例
```json
{
  "code": 200,
  "status": "success",
  "message": "OK",
  "data": {
    "online_agents": 10,
    "overloaded_agents": 2,
    "cpu_threshold": 80,
    "memory_threshold": 80,
    "disk_threshold": 80,
    "average_headroom": 42.6,
    "capacity_score": 42.6,
    "bottlenecks": {"cpu": 1, "memory": 1, "disk": 0},
    "recommendations": "容量总体健康",
    "overloaded_list": [
      {"agent_id": "a-02", "cpu_usage": 88.1, "memory_usage": 79.2, "disk_usage": 70.3, "reason": "cpu", "timestamp": "2025-11-17T08:00:00Z"}
    ]
  }
}
```

---

## 错误与状态码
- `400` 参数错误；`401/403` 鉴权失败或用户未激活；`500` 服务内部错误
- 统一错误体：`system.APIResponse`，`status` 为 `error/failed`，`message/error` 包含说明

## 测试与导入
- 已生成 OpenAPI 文档：`docs/Agent管理模块设计/agent_analysis_openapi.yaml`
- 可直接导入 Apifox，服务地址为 `http://<host>:<port>/api/v1`