# Qscan 并发模型在 NeoScan 中的借鉴与取舍

## 1. 核心定位差异

在参考 Qscan 的并发代码时，必须首先明确两者架构定位的本质区别，否则会陷入战术上的盲目照搬。

| 特性 | Qscan (现有参考) | NeoScan (本项目) |
| :--- | :--- | :--- |
| **系统定位** | **单兵作战工具** (Standalone Tool) | **分布式指挥系统** (Distributed Brain) |
| **运行环境** | 本地单机运行 | Master (云端) + Agents (分布节点) |
| **并发对象** | **本地线程** (直接执行 Socket/IO) | **远程任务** (分发指令 & 状态管理) |
| **瓶颈所在** | 本地 CPU/内存/带宽 | 调度吞吐量、DB I/O、Agent 集群总容量 |
| **适用场景** | 适合 **NeoScan Agent** 内部实现 | 适合 **NeoScan Master** 调度逻辑 |

---

## 2. 值得借鉴的设计 (The Good Parts)

Qscan 的 Worker Pool 模式非常适合 **NeoScan Agent** 端的内部实现，因为 Agent 本质上就是在执行具体的扫描任务。

### 2.1 `sync.WaitGroup` 优雅退出
- **Qscan 实践**: 使用 `wg.Add(1)` 和 `wg.Done()` 确保所有子协程完成后主程序才退出。
- **NeoScan 应用**: 
    - **Agent 端**: 接收到 Master 的 `Stop` 指令或进程关闭时，必须等待当前正在扫描的端口/指纹任务完成，避免产生脏数据。
    - **Master 端**: 服务重启时，等待正在处理的 HTTP 请求返回。

### 2.2 `sync.Map` 状态追踪
- **Qscan 实践**: 维护一个 `JobsList` (sync.Map) 记录正在运行的任务，用于日志和状态检查。
- **NeoScan 应用**: 
    - **Agent 端**: 需要维护 `RunningTasks` Map，记录 TaskID -> Context，以便随时可以 Cancel 掉某个特定的扫描任务。

### 2.3 泛型接口解耦
- **Qscan 实践**: Pool 接收 `interface{}` 类型参数，具体的 Worker 再进行断言处理。
- **NeoScan 应用**: 
    - **Agent 端**: 扫描器应该设计为通用执行器，无论是 Nmap 扫描还是 GoPOC 扫描，统一封装为 `TaskPayload` 接口，由 Worker 统一调度。

---

## 3. 不适合 Master 的设计 (The Bad Parts for Distributed System)

以下设计在单机工具中是优点，但在分布式调度器 (Master) 中是**致命缺陷**。

### 3.1 无缓冲 Channel (阻塞式生产)
- **Qscan 模式**: `make(chan interface{})`。生产者 (分发者) 会被阻塞，直到有 Worker 空闲。
- **Master 风险**: Master 是 HTTP Server，**绝对不能被阻塞**。如果所有 Agent 都忙，Master 不应卡在 `channel <- task` 等待，而应将任务存入数据库或优先级队列，并立即响应 HTTP 成功。

### 3.2 简单的线程数限制 (Thread Counting)
- **Qscan 模式**: `Threads = 200` 意味着并发 200 个连接。
- **Master 风险**: Master 管理的是远程资源。限制 Master 开 200 个 Goroutine 毫无意义（分发任务只需几毫秒）。Master 需要的是 **"权重限制" (Weighted Semaphores)** —— 限制的是 Agent 的负载总分（如 Nmap 占 10 分，Ping 占 1 分），而不是协程数。

### 3.3 缺乏优先级 (FIFO)
- **Qscan 模式**: 任务先进先出。
- **Master 风险**: 生产环境中，高危漏洞应急扫描必须能**插队**到普通的资产盘点任务前面。简单的 Channel 队列无法实现插队。

---

## 4. 架构决策结论

### 对于 NeoScan Agent (执行器)
*   **建议**: **完全参考 Qscan 的 `lib/pool` 实现**。
*   **理由**: Agent 就是一个高性能的单机执行单元，需要精细控制本地的 Goroutine 数量防止系统崩溃。Worker Pool 模型是最优解。

### 对于 NeoScan Master (大脑)
*   **建议**: **放弃 Worker Pool，采用 "优先级队列 + 异步调度循环"**。
*   **理由**: Master 需要非阻塞的高吞吐调度、复杂的优先级控制和基于权重的负载均衡。
*   **参考文档**: 见同目录下的 `NeoScan纯内存并发调度器设计方案.md`。

---
> **Linus 的注脚**: 
> "Good code is about data structures." 
> 在 Agent 端，关注的是如何高效利用 CPU (Worker Pool); 
> 在 Master 端，关注的是如何高效组织数据 (Priority Queue)。
> 别搞混了。
