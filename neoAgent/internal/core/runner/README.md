# Runner & Execution Layer (执行局/调度器)

## 模块职责
本模块是 Agent 的**任务调度核心**，同时也是通用任务模型与具体扫描工具之间的**适配中心**。

### 核心组件
1.  **RunnerManager**：调度总管。负责管理所有注册的 Runner，根据任务类型分发任务。
2.  **Runner 接口**：统一的执行契约。所有扫描能力必须实现此接口才能被调度。
3.  **Internal Adapters (`adapter_*.go`)**：具体扫描器的适配器。

## 适配器设计 (`adapter_*.go`)

在 `runner` 包中，你会看到如 `adapter_os.go` 或 `adapter_service.go` 这样的文件。它们是**接口适配器 (Interface Adapter)**。

### 为什么需要这些适配器？
底层的扫描器（如 `scanner/os`, `scanner/port`）是**特种兵**，它们应该是纯粹的、独立的工具，不应该知道 "Task" 系统的存在。

*   **Scanner 的视角**：我只接受 `ip`, `port`, `timeout` 这样具体的参数，我不知道什么是 `model.Task`，也不想去解析 `map[string]interface{}`。
*   **Runner 的视角**：我只处理通用的 `model.Task`。

**适配器的工作流**：
1.  **参数解包 (Unpacking)**：从通用的 `Task.Params` 中提取具体扫描器需要的参数（如从 map 中提取 "service_detect=true"）。
2.  **调用转发 (Delegation)**：调用底层 Scanner 的纯函数接口。
3.  **结果封装 (Wrapping)**：将 Scanner 返回的具体结构体（如 `OsInfo`）包装成通用的 `TaskResult`。

---

## 架构设计问答：为什么不复用 Service Adapter？

**Q: 既然 Service 层已经做了一次适配，为什么这里还要再做一次？**
A: **这是两道完全不同的防线**。

1.  **Service Adapter (`internal/service/adapter`)**：是**外交官**。它保护 Core 层不受**外部网络协议**变化的影响。
2.  **Runner Adapter (本层)**：是**翻译官**。它保护底层 Scanner 不受**上层任务模型**变化的影响（依赖倒置原则）。

如果去掉这一层，底层的 Scanner 就必须依赖上层的 `model.Task` 定义，导致架构分层混乱（底层依赖高层）。保持这一层适配，使得 Scanner 可以作为独立的库被复用，而完全不需要知道 Agent 任务系统的存在。
