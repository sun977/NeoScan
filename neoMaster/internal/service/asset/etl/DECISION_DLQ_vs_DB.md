# 为什么选择 "Database-as-a-Queue" 而非 MQ 死信队列

## 1. 核心定义

*   **当前实现 (Soft DLQ)**: 使用关系型数据库表 (`asset_etl_errors`) 存储处理失败的消息和上下文，状态由字段 (`status`) 控制。
*   **传统死信队列 (MQ DLQ)**: RabbitMQ/Kafka 的原生 Dead Letter Exchange。消息处理失败后被路由到一个独立的队列，通常是二进制或序列化后的 Blob。

## 2. 决策依据 (Rationale)

### 2.1 可观测性 (Observability) —— 核心痛点
ETL (数据清洗) 场景中，失败通常意味着数据格式异常或代码逻辑缺陷。
*   **DB 方案优势**: 
    *   **透明化**: SQL 是最强大的调试工具。可以直接查询特定项目、特定错误类型的失败记录。
    *   **上下文**: 错误记录与业务实体 (`ProjectID`, `TaskID`) 强关联，一眼可见影响范围。
*   **MQ DLQ 劣势**: 
    *   **黑盒**: 消息通常是 Base64 编码的二进制。排查需要消费、解码、打印，流程繁琐。

### 2.2 修复与重试 (Fix & Replay) —— 灵活性
*   **DB 方案优势**:
    *   **原地修正**: 支持直接修改 `raw_data` 并重置状态为 `new`，实现单条数据的修复重试。
    *   **选择性重试**: 可以编写 SQL 或脚本，只重试特定类型的错误（如"网络超时"），忽略"数据结构错误"。
*   **MQ DLQ 劣势**:
    *   **笨重**: 通常只能全量 "Shovel" (铲回) 原队列。如果代码 Bug 未修，只会死循环。

### 2.3 架构复杂度 (Complexity) —— 实用主义
*   **DB 方案优势**:
    *   **复用设施**: 直接使用现有的 MySQL，无需维护额外的 MQ 集群配置。
    *   **持久化**: 不受 MQ 消息过期 (TTL) 限制，证据永久保留。
*   **MQ DLQ 劣势**:
    *   **运维成本**: 引入了新的运维对象和故障点。

## 3. 性能考量

虽然 DB 吞吐量不如 MQ，但在 NeoScan 的 ETL 场景下：
1.  **错误率预期**: 正常情况下错误率应极低 (<1%)。
2.  **写压力**: 即使在大规模扫描下，错误记录的 INSERT 频率远未达到 MySQL 瓶颈。
3.  **兜底策略**: 如果预见极端错误风暴 (Error Storm)，可在代码层增加简单的 Circuit Breaker (熔断器) 保护数据库。

## 4. 结论

> "Bad programmers worry about the code (MQ mechanics). Good programmers worry about data structures (observability)."

在需要深度分析错误原因的 ETL 场景下，**数据库表提供的可观测性和操作灵活性优于盲目的死信队列**。
