
# NeoScan Master-Agent 交互数据协议文档 (v1.0)

Agent Pull 自己状态给 Master

Master 获取 Agent 状态的方式非常简单粗暴： 靠 Agent 主动上报心跳 (Heartbeat) 。

Master 收到心跳后，会立即更新数据库中的 Agent 快照 。

Agent --- master(db) --- master 

当 Master 的调度器 (Dispatcher) 决定是否给这个 Agent 分配新任务时，它会查这张表：

1. 查状态 ：如果 LastHeartbeat 超过 1 分钟没更新 -> 判定为 离线 (Offline) -> 不发任务 。
2. 查负载 ：如果 CPUUsage > 90% 或 RunningTasks >= MaxConcurrent -> 判定为 过载 (Overload) -> 不发任务 。

Master 不需要"连接" Agent，它只需要 读数据库里的快照 。

- Agent ：负责定期刷新快照。
- Master ：负责根据快照做决策。
这就是为什么你不需要在 Agent 端开任何端口，Master 依然对 Agent 的一举一动了如指掌。


## 1. 协议概览

Master 与 Agent 之间通过 HTTP/RESTful API 进行通信。所有请求和响应均使用 JSON 格式。

**通信鉴权**：
- Agent 在注册后会获得 Token。
- 后续所有请求（心跳、拉取任务、上报结果）均需在 Header 中携带：
  ```http
  Authorization: Bearer <your_agent_token>
  ```

---

## 2. 交互环节详解

### 2.1 Agent 注册 (Registration)
**场景**：Agent 首次启动，向 Master 报到并获取身份凭证。

*   **URL**: `POST /api/v1/agent`
*   **方向**: Agent -> Master

**请求数据 (JSON)**
```json
{
  "hostname": "agent-node-01",
  "ip_address": "192.168.1.100",
  "port": 8080,
  "version": "v1.0.0",
  "os": "linux",
  "arch": "amd64",
  "cpu_cores": 4,
  "memory_total": 8589934592,
  "disk_total": 512000000000,
  "capabilities": ["port_scan", "web_scan"], // 核心能力申明
  "tags": ["zone-a", "high-perf"]
}
```

**响应数据 (JSON)**
```json
{
  "code": 200,
  "status": "success",
  "data": {
    "agent_id": "uuid-agent-12345", // Master 分配的唯一ID
    "grpc_token": "eyJhbGciOi...",    // 后续通信用的 Token
    "status": "online"
  }
}
```

---

### 2.2 心跳保活 (Heartbeat)
**场景**：Agent 周期性（默认 30s）发送心跳，汇报存活状态和性能指标。

*   **URL**: `POST /api/v1/agent/heartbeat`
*   **方向**: Agent -> Master

**请求数据 (JSON)**
```json
{
  "agent_id": "uuid-agent-12345",
  "status": "online", // online, busy, offline
  "metrics": {        // 可选，携带性能数据
    "cpu_usage": 15.5,
    "memory_usage": 40.2,
    "running_tasks": 2
  }
}
```

**响应数据 (JSON)**
```json
{
  "code": 200,
  "status": "success",
  "message": "Heartbeat processed"
}
```

---

### 2.3 拉取任务 (Fetch Tasks)
**场景**：Agent 主动向 Master 询问是否有分配给自己的新任务。

*   **URL**: `GET /api/v1/orchestrator/agent/{agent_id}/tasks`
*   **方向**: Agent -> Master

**响应数据 (JSON) - 核心关注点**
Master 返回任务列表。请注意 **`input_target`** 字段，这是我们刚刚升级的"富结构"。

```json
{
  "code": 200,
  "status": "success",
  "data": [
    {
      "task_id": "task-uuid-001",
      "project_id": 101,
      "task_type": "tool",
      "tool_name": "nmap",
      "tool_params": "-sS -p-", // 原始工具参数模板
      
      // 核心变化：这是一个 JSON 字符串，解析后是 Target 对象数组
      "input_target": "[{\"type\":\"ip\",\"value\":\"192.168.1.100\",\"source\":\"manual\",\"meta\":{\"port\":\"8080\",\"service\":\"http\"}}]",
      
      "timeout": 3600
    }
  ]
}
```

**`input_target` 解析后的结构 (`[]Target`)**:
```json
[
  {
    "type": "ip",           // 目标类型: ip, domain, url
    "value": "192.168.1.100", // 扫描目标
    "source": "manual",     // 来源
    "meta": {               // 上下文元数据 (关键!)
      "port": "8080",       // 上一阶段发现的端口
      "service": "http"     // 上一阶段发现的服务
    }
  }
]
```

---

### 2.4 上报任务状态 (Update Status / Result)
**场景**：Agent 开始执行、执行中或执行完成后，汇报状态和结果。

*   **URL**: `POST /api/v1/orchestrator/agent/{id}/tasks/{task_id}/status`
*   **方向**: Agent -> Master

**请求数据 (JSON)**
```json
{
  "status": "completed", // running, completed, failed
  "result": "{\"open_ports\": [80, 443], \"vulnerabilities\": []}", // 任务执行结果摘要(JSON字符串)
  "error_msg": "" // 如果失败，填写错误信息
}
```

**响应数据 (JSON)**
```json
{
  "code": 200,
  "status": "success"
}
```

## 3. 总结与改进点

目前 Master 端已经准备好发送包含 `Meta` 信息的富 `input_target` 数据。

**Agent 端开发需要注意**：
1.  在处理 **拉取任务** 响应时，不要直接把 `input_target` 当作简单字符串。
2.  需要将其反序列化为对象数组。
3.  在构造扫描命令时，优先使用 `meta` 中的信息（如端口），实现智能扫描。