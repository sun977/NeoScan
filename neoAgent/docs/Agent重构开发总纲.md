# NeoAgent 重构与开发总纲 v1.1

## 1. 重构愿景

打造一个**独立、快速、自包含**的安全扫描 Agent。
它既是分布式集群中的忠实 Worker，也是单机环境下强大的扫描利器。
它遵循 "Native First" 原则，追求极致的性能与部署体验。

---

## 2. 架构蓝图

### 2.1 目录结构重组
```text
neoAgent/
├── cmd/agent/
│   ├── main.go          # Cobra Root Command (入口)
│   ├── server.go        # Server Mode (Cluster Worker)
│   └── scan.go          # CLI Mode (Standalone Scanner)
├── internal/
│   ├── core/            # 核心业务层 (无 HTTP 依赖)
│   │   ├── scanner/     # 扫描引擎 (IpAlive, PortService, Vuln...)
│   │   ├── runner/      # 并发调度器
│   │   └── reporter/    # 结果上报接口
│   ├── service/         # [应用] Server模式业务逻辑 (Worker Logic)
│   │   ├── communication/ # Master 通信
│   │   ├── adapter/       # 协议适配与 DTO
│   │   └── task/          # Worker 主循环
│   └── pkg/             # 通用工具包
```

### 2.2 核心设计原则
1.  **解耦**: Core Service 不感知 HTTP/CLI 上下文。
2.  **并发**: 基于 Channel 信号量的分层并发控制。
3.  **能力**: 
    - **Host Discovery**: 原生 ICMP/ARP/TCP Connect。
    - **Port/Service**: 仿写 Nmap (Gonmap) 逻辑，实现端口发现+服务识别+OS识别。
    - **Fingerprint**: 混合模式 (Built-in Nmap Probes + Dynamic Master Rules)。
4.  **安全**: 基于 Token + CA Hash 的注册机制。

---

## 3. 开发阶段规划 (The Roadmap)

### 阶段一：核心解耦 (Core Decoupling) —— **Foundation**
**目标**: 将业务逻辑从 HTTP Server 中剥离，建立独立的 Core Service。
**状态**: 🟢 **已完成**

- [x] **1.1 目录结构调整**: 创建 `internal/core`，迁移相关代码。
- [x] **1.2 任务模型统一**: 定义通用的 `Task` 和 `TaskResult` 结构体，消除 Web 依赖。
- [x] **1.3 核心接口定义**: 定义 `Scanner`, `Runner`, `Reporter` 接口。
- [x] **1.4 依赖清理**: 确保 `internal/core` 不引用 `gin` 或 `net/http` (作为 Server)。

### 阶段二：CLI 改造 (CLI Transformation) —— **Interaction**
**目标**: 引入 Cobra，实现命令行入口和参数解析。
**状态**: 🟢 **已完成**

- [x] **2.1 引入 Cobra**: 重写 `cmd/agent/main.go`。
- [x] **2.2 实现 Server 命令**: 将原 `main` 逻辑封装进 `server` 子命令（保持默认行为）。
- [x] **2.3 实现 Scan 命令**: 开发 `scan` 子命令，实现 Flags 到 `Task` 的映射。
- [x] **2.4 结果输出**: 实现 `ConsoleReporter`，支持表格和 JSON 输出。
- [x] **2.5 参数优化**: 实现简写参数 (`-r`, `-p`) 和隐式 TCP 扫描逻辑。

### 阶段三：原生能力建设 (Native Capabilities) —— **Power**
**目标**: 逐步替换/实现原生扫描能力，摆脱外部依赖。
**状态**: 🟢 **已完成** (所有原生扫描模块已交付)

- [x] **3.1 并发框架**: 实现 `internal/core/runner` (Semaphore + WaitGroup)。
- [x] **3.2 主机发现**: 实现原生的 ICMP/ARP/TCP Connect (`IpAliveScanner`)。
- [x] **3.3 端口服务扫描**: 移植 Gonmap 逻辑，实现 `PortServiceScanner`。
    - [x] 探针管理 (Probe Management)
    - [x] 扫描引擎 (Scan Engine)
    - [x] 指纹匹配 (Match Engine)
- [x] **3.4 指纹规则管理**: 实现混合规则加载机制 (Embed + Dynamic)。
- [x] **3.5 OS 识别**: 实现基于 TCP/IP 栈 (Nmap) 和 服务 Banner 的 OS 识别。
- [x] **3.6 核心网络库**: 重构 `internal/core/lib/network`，支持 Proxy/Timeout 统一管理。
- [x] **3.7 基础爆破 (Brute Force)**:
    - [x] **Infrastructure**: Cracker 接口定义, 字典管理 (内置+动态), 并发调度器 (Global+Serial)。
    - [x] **Protocols**: 
        - [x] SSH, RDP (Native), SMB, Telnet, FTP, SNMP
        - [x] MySQL, Postgres, MSSQL, Oracle (SID+Auth), Mongo, Redis, ClickHouse, ES
    - [x] **Integration**: 注册到 RunnerManager。
    - [x] **CLI**: 实现 `scan brute` 子命令，支持多端口解析与全量模式 (已国际化)。
    - [x] **Refactor**: 引入 Global Scanner Factory，统一 Brute Scanner 实例化逻辑 (Phase 1)。
- [x] **3.8 高级并发优化**:
    - [x] 引入自适应速率控制 (Adaptive Rate Limiting - AIMD)。
    - [x] 实现 `RttEstimator` 动态调整超时 (RFC 6298)。
    - [x] 全面集成 QoS: `PortServiceScanner`, `IpAliveScanner`, `OSScanner` 均已接入动态超时与流控。

### 阶段四：编排与集成 (Orchestration & Integration) —— **Automation**
**目标**: 实现单机全流程扫描与集群接入。
**状态**: 🟢 **已完成**

- [x] **4.1 全流程编排 (Scan Orchestration)**:
    - [x] 实现 `PipelineRunner` 串联各个 Scanner (v1.0 线性流程)。
    - [x] 实现 `scan run` 命令，支持 `--auto` 和 Pipeline Mode。
    - [x] **重构：并行分发 (Phase 2 Upgrade)**:
        - [x] 实现 `ServiceDispatcher`: 基于端口服务结果进行任务分发 (Web/Vuln/Brute)。
        - [x] 实现 `PipelineRunner` 并行化: 
            - Phase 1 (Sequential): Alive -> Port -> Service
            - Phase 2 (Parallel): Web + Vuln (High Priority) -> Brute (Low Priority)
        - [x] 实现优先级控制: 确保 Vuln 任务完成后再触发 Brute 任务 (针对同一 Target)。
- [x] **4.2 集群接入增强 (Cluster Adapter)**:
    - [x] **Step 1**: 创建 `internal/model/adapter`，固化数据契约 (Payload DTO)。
    - [x] **Step 2**: 完善 `internal/service/adapter`，实现双向协议转换。
    - [x] **Step 3**: 完善 `internal/service/client`，实现主动 HTTP 通信。
    - [x] **Step 4**: 重构 `internal/service/task`，实现 Worker 主循环。
    - [x] **Step 5**: 更新 `cmd/agent/server.go` (实际为 `app.go`) 入口。
    - [x] **Step 6**: 统一 Server 模式扫描器初始化逻辑 (`setup/core.go` -> `RunnerManager` Factory)。
- [x] **4.3 高级能力集成**:
    - [x] 实现 Web 指纹识别。
    - [x] Web Scanner 完善与集成。
    - [x] **稳定性加固**: Linux 平台下的权限修复与 Segfault 防护。

### 阶段五：深度探测与漏洞扫描 (Deep Recon & Vulnerability Scanning) —— **Exploitation**
**目标**: 补齐 Web 深度信息收集、真正的漏洞扫描引擎以及主动暴露面试探能力。
**状态**: 🏃 **进行中**

- [ ] **5.1 Web 扫描深度增强 (Web Crawler & Passive Analyzer)** — 🏃 **进行中，详细拆分见** [`docs/爬虫/Web爬虫与被动分析器实施文档-v1.0.md`](./爬虫/Web爬虫与被动分析器实施文档-v1.0.md)：
    - [x] **Sprint 0**：依赖引入 (`goquery`) + `crawler` 包骨架 + `WebResult` 新增字段 (`Depth/Forms/Params/Leaks`)。
    - [x] **Sprint 1**：`crawler` 核心 BFS 爬取引擎（并发 worker、去重、深度/页数上限，`-race` 检测通过，无死锁）。
    - [ ] **Sprint 2**：攻击面提取 `extract.go`（链接/表单/URL 参数，基于 `goquery`）。
    - [ ] **Sprint 3**：被动泄露检测 `leak.go`（AK/SK、JWT、内网 IP 正则规则 + 脱敏）。
    - [ ] **Sprint 4**：`web_scanner.go` 收口重构（`fallbackScan` → `fallbackFetch` + `buildWebResult` 统一收口，零功能回归）。
    - [ ] **Sprint 5**：三处接入点改造 (`scan_web.go`/`scan_run.go`/`dispatcher.go`/`task_to_core.go`) + 自动决策 (`decideCrawlDepth`) + 按需浏览器升级 (`escalateIfNeeded`) + 端到端联调。
- [ ] **5.2 漏洞扫描原子能力落地 (Vuln Scanner)**:
    - [ ] 封装 Nuclei 执行引擎 (`internal/core/scanner/vuln`)。
    - [ ] 根据 WebScanner 识别出的技术栈动态过滤 Nuclei 模板。
    - [ ] 真实落地 "Vuln First" 调度策略，打通全流程。
- [ ] **5.3 目录与子域名扫描 (Dir & Subdomain Scanner)**:
    - [ ] 实现 DirScanner，基于高并发 HTTP 请求进行敏感路径/备份文件爆破。
    - [ ] 实现 SubdomainScanner，基于并发 DNS 解析进行子域名枚举。
    - [ ] 完善对应 CLI 命令的底层调用联调。

---

## 4. 质量控制 (Quality Control)

- **Unit Test**: 核心扫描逻辑覆盖率 > 80%。
- **Benchmark**: 端口扫描速度不低于 `fscan` 水平。
- **Lint**: 通过 `golangci-lint` 检查。
- **No Global State**: 严禁在 Core 中使用全局变量（Logger 除外）。
