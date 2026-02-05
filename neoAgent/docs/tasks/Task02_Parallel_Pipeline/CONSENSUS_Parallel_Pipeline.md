# 需求共识文档 - 全流程并行编排升级 (Phase 4.1)

## 1. 核心共识
我们一致同意将 NeoAgent 的扫描流程从 **线性串行** 升级为 **两阶段并行** 架构，以解决性能瓶颈并提高扫描安全性。

## 2. 需求描述
### 2.1 目标
在保持 "IP 级并发" 的基础上，引入 **ServiceDispatcher**，实现针对开放服务的深度扫描任务（Web/Vuln/Brute）的并行分发与执行。

### 2.2 流程定义
**Phase 1: Discovery (串行)**
- Alive Scan (ARP/ICMP/TCP)
- Port Scan (TCP Connect)
- Service Scan (Version Detection)
- OS Scan (Fingerprinting)

**Phase 2: Assessment (并行 + 优先级)**
- **Step 2.1 (High Priority)**: 并行执行 Web Scan + Vuln Scan。
- **Step 2.2 (Low Priority)**: 待 Step 2.1 完成后，执行 Brute Force。

## 3. 技术方案
### 3.1 ServiceDispatcher
- **职责**: 接收 `PipelineContext`，根据 `Services` Map 生成任务列表。
- **策略**:
    - `http/https` -> 生成 WebScan Task, VulnScan Task
    - `ssh/rdp/mysql...` -> 生成 BruteScan Task, VulnScan Task
    - `unknown` -> 忽略

### 3.2 Parallel AutoRunner
- **改造点**: `executePipeline` 方法。
- **并发控制**: 使用 `sync.WaitGroup` 管理 Phase 2 的子任务。
- **错误处理**: 子任务失败不影响整体流程，但在 Report 中体现。

### 3.3 Global Factory 集成
- 所有的 Scanner 实例（包括 Web/Vuln）必须通过 `internal/core/factory` 创建。

## 4. 验收标准
1.  **并行性验证**: 只有 Web/Vuln 任务在同一时间段内运行。
2.  **顺序性验证**: Brute 任务必须在 Web/Vuln 结束后开始。
3.  **结果完整性**: 最终 Report 包含所有阶段的扫描结果。
4.  **无回归**: 原有的 Alive/Port/OS 扫描功能不受影响。

## 5. 风险控制
- **资源竞争**: 并行扫描可能会增加 CPU/内存开销，需监控资源使用。
- **死锁风险**: 确保 WaitGroup 正确 Done，防止 Goroutine 泄漏。
