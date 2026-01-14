# Asset ETL Engine

负责资产数据的清洗、转换和入库。

## 职责
1. 消费 ResultQueue 中的 StageResult
2. 数据清洗与标准化 (Parser/Normalizer)
3. 资产合并与入库 (Merger) - 核心 Upsert 逻辑

## 支持的扫描结果类型 (Result Types)

| Result Type | 映射状态 | 说明 |
| :--- | :--- | :--- |
| `ip_alive` | ✅ 已支持 | 基础主机信息 (IP, OS) |
| `fast_port_scan` | ✅ 已支持 | 端口开放情况 |
| `full_port_scan` | ✅ 已支持 | 端口开放情况 |
| `service_fingerprint` | ✅ 已支持 | 服务指纹 (Product, Version, CPE) |
| `vuln_finding` | ✅ 已支持 | 通用漏洞 (CVE, Severity) |
| `poc_scan` | ✅ 已支持 | PoC 验证结果 (High Confidence) |
| `web_endpoint` | ✅ 已支持 | Web 站点信息 (Title, Headers, TechStack) |
| `password_audit` | ✅ 已支持 | 弱口令审计 (SSH, MySQL, etc.) |
| `proxy_detection` | ⏭️ 跳过 | 待专门表设计 |
| `directory_scan` | ⏭️ 跳过 | 待专门表设计 |
| `subdomain_discovery` | ⏭️ 跳过 | 待专门表设计 |
| `api_discovery` | ⏭️ 跳过 | 待专门表设计 |
| `file_discovery` | ⏭️ 跳过 | 待专门表设计 |
| `other_scan` | ⏭️ 跳过 | 待专门表设计 |

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
- **后续处理**: 
    - **API 重放**: 支持通过 API (`POST /api/v1/asset/etl/errors/replay`) 触发重放。
    - **CLI 重放**: 支持通过命令行工具触发重放。
    - **逻辑复用**: 重放操作会将原始 `StageResult` 重新投递到 `ResultQueue`，完全复用现有的 Mapper/Merger/Retry 逻辑。

### 4. 数据完整性
- **事务性**: 尽可能保证单个 Asset Bundle 的入库是原子的（虽然目前 Merger 尚未完全事务化，但通过幂等性设计减少了影响）。
- **无数据丢失**: 所有失败的任务都会被持久化，确保没有任何资产数据因为程序错误而静默丢失。
