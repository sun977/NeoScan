# Asset ETL Engine

负责资产数据的清洗、转换和入库。

## 职责
1. 消费 ResultQueue 中的 StageResult
2. 数据清洗与标准化 (Parser/Normalizer)
3. 资产合并与入库 (Merger) - 核心 Upsert 逻辑

## 错误处理策略 (Error Handling Strategy)

为了保证数据的一致性和系统的稳定性，ETL 引擎采用了分级错误处理机制。

### 1. 错误分类 (Classification)
我们将错误分为两类：
- **瞬时错误 (Transient Error)**: 由于网络抖动、数据库锁、连接超时等原因导致的临时性故障。这类错误可以通过重试解决。
- **持久错误 (Persistent Error)**: 由于数据格式错误、约束冲突（非并发导致）、代码逻辑错误等原因导致的永久性故障。这类错误重试无效。

### 2. 重试机制 (Retry Policy)
针对 **瞬时错误**，采用 **指数退避 (Exponential Backoff)** 策略进行重试：
- 最大重试次数: 3次
- 退避时间: 100ms -> 200ms -> 400ms
- 目的: 避免在数据库压力大时加剧负载 (Thundering Herd)。

### 3. 死信队列 (Dead Letter Queue)
针对 **持久错误** 或 **重试耗尽** 的任务，系统将其降级为"死信"，记录到 `asset_etl_errors` 表中。
- **记录内容**: 原始数据 (RawData)、错误堆栈、错误阶段。
- **后续处理**: 提供重放接口 (Replay API)，允许在修复代码或数据后重新处理死信。

### 4. 数据完整性
- **事务性**: 尽可能保证单个 Asset Bundle 的入库是原子的（虽然目前 Merger 尚未完全事务化，但通过幂等性设计减少了影响）。
- **无数据丢失**: 所有失败的任务都会被持久化，确保没有任何资产数据因为程序错误而静默丢失。
