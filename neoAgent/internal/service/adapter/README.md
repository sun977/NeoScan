# Service Adapter Layer (防腐层/外交官)

## 模块职责
本模块是 Agent 与外部世界（Master）之间的**外交部**。它的核心职责是**协议转换 (Protocol Translation)**。

### 核心功能
1.  **输入清洗**：接收来自 Master 的 `clientModel.Task`（外部协议数据，可能包含脏数据）。
2.  **意图翻译**：将业务意图翻译为技术参数。
    *   例如：Master 下发 "fast_scan"（业务意图），本层将其转换为 `TaskTypePortScan` + `port="top100"`（内核可执行的技术参数）。
3.  **结果封装**：将内核产生的 `model.TaskResult` 转换为符合 Master 通信协议的 JSON 报告。

## 架构设计问答：为什么需要这一层？

**Q: 为什么不直接让 Core 层处理 Master 的任务对象？**
A: **拒绝污染**。Master 的通信协议（JSON 结构、字段命名）属于外部契约，可能会频繁变更。如果让 Core 层直接依赖这些外部模型，一旦协议变更，核心业务逻辑就必须修改。本层作为**防腐层 (Anti-Corruption Layer)**，确保了 Core 层只依赖自己定义的纯净模型 (`internal/core/model`)。

---

### 与 Runner Adapter 的区别
*   **Service Adapter (本层)**：处理 **Master <-> Core** 的边界。解决的是**网络协议与业务模型**的映射问题。
*   **Runner Adapter (`internal/core/runner`)**：处理 **Core <-> Scanner** 的边界。解决的是**通用模型与具体实现**的接口兼容问题。
