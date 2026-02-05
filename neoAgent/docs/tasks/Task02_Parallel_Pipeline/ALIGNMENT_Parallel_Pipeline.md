# 需求对齐文档 - 全流程并行编排升级 (Phase 4.1)

## 原始需求
参考 `docs/Agent重构开发总纲.md` 中的 4.1 节：
- **重构：并行分发 (Phase 2 Upgrade)**
    - 实现 `ServiceDispatcher`: 基于端口服务结果进行任务分发 (Web/Vuln/Brute)。
    - 实现 `PipelineRunner` 并行化:
        - Phase 1 (Sequential): Alive -> Port -> Service
        - Phase 2 (Parallel): Web + Vuln (High Priority) -> Brute (Low Priority)
    - 实现优先级控制: 确保 Vuln 任务完成后再触发 Brute 任务 (针对同一 Target)。

## 项目上下文
### 技术栈
- 编程语言：Go 1.23+
- 核心框架：NeoScan Agent Core
- 并发模型：Go Routine + Channel + WaitGroup + Semaphore

### 现有架构理解
- **AutoRunner**: 目前采用 "IP级并发" 的线性流程。每个 IP 一个 Goroutine，内部顺序执行 Alive -> Port -> Service -> OS。
- **瓶颈**: 
    - "Head-of-Line Blocking": 耗时的服务识别或 OS 识别会阻塞当前 IP 的后续处理。
    - 资源利用率低: 网络 IO (Port Scan) 和 CPU 计算 (Brute) 混合在同一个线性流中，难以最大化利用资源。
- **Dispatcher**: 代码中已预留 `dispatcher` 字段，但目前仅为 Placeholder。

## 需求理解
### 核心目标
将现有的 `AutoRunner` 升级为支持 **Stage-Decoupled Pipeline (分阶段解耦流水线)** 的架构，或者至少实现 **Phase 2 的并行化**。

### 功能边界
**包含功能：**
1.  **ServiceDispatcher**: 
    - 接收 `PipelineContext` (包含 Open Ports 和 Service Info)。
    - 根据服务类型 (e.g. `http`, `ssh`, `mysql`) 生成对应的子任务 (WebScan, BruteScan, VulnScan)。
2.  **Parallel Execution Engine**:
    - 在 Phase 2 阶段，并行执行生成的子任务。
    - 支持优先级控制：Web/Vuln 优先执行，Brute 后置执行（防止账号锁定影响其他探测）。
3.  **集成 Global Factory**:
    - 使用 Factory 创建 Brute/Web/Vuln Scanner 实例。

**明确不包含（Out of Scope）：**
- 复杂的 DAG 任务调度（保持两阶段模型：Discovery -> Assessment）。
- 跨主机的关联分析。
- 分布式调度（仅限单机并发）。

## 疑问澄清
### P0级问题（必须澄清）
1.  **并发模型选择**
    - **Option A (Current Enhanced)**: 保持 IP 级并发，但在 IP 内部，当进入 Phase 2 时，启动多个 Goroutine 并行处理 Web/Vuln/Brute。
    - **Option B (Fully Decoupled)**: 拆分为多个全局 Worker Pool (DiscoveryPool, AssessmentPool)，通过 Channel 传递。
    - **Linus 决策**: 考虑到代码复杂度和向后兼容性，**Option A** 是更稳健的第一步。它能解决主要瓶颈，且不需要重写整个 Pipeline 架构。我们先做 Option A。

2.  **优先级控制实现**
    - 如何确保 Vuln 优先于 Brute？
    - **方案**: 在 Phase 2 中，先启动 Web/Vuln 任务，使用 `WaitGroup` 等待它们完成（或部分完成），再启动 Brute。或者并行启动，但在 Brute 内部增加延迟？
    - **Linus 决策**: 严格的顺序是 `(Web|Vuln) -> Wait -> Brute`。虽然牺牲了一点并发度，但安全性（防封禁）更重要。Brute 永远是最后手段 (Last Resort)。

## 验收标准
### 功能验收
- [ ] `scan run` 命令能正确触发并行扫描。
- [ ] 识别出 HTTP 服务后，自动触发 WebScan (模拟/空跑)。
- [ ] 识别出 SSH/MySQL 服务后，自动触发 BruteScan。
- [ ] 日志显示 Web/Vuln 任务在 Brute 任务之前启动/完成。
- [ ] 最终报告包含所有阶段的扫描结果。

### 质量验收
- [ ] 单元测试覆盖 Dispatcher 逻辑。
- [ ] 无数据竞争 (Data Race)。
- [ ] 资源泄漏检查 (Goroutine Leak)。
