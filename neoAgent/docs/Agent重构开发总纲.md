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
│   ├── server/          # HTTP/gRPC 相关 (原 handler/router/middleware)
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
**状态**: 🟡 **进行中**

- [x] **3.1 并发框架**: 实现 `internal/core/runner` (Semaphore + WaitGroup)。
- [x] **3.2 主机发现**: 实现原生的 ICMP/ARP/TCP Connect (`IpAliveScanner`)。
- [x] **3.3 端口服务扫描**: 移植 Gonmap 逻辑，实现 `PortServiceScanner`。
    - [x] 探针管理 (Probe Management)
    - [x] 扫描引擎 (Scan Engine)
    - [x] 指纹匹配 (Match Engine)
- [x] **3.4 指纹规则管理**: 实现混合规则加载机制 (Embed + Dynamic)。
- [ ] **3.5 基础爆破**: 实现 SSH/MySQL/Redis 的原生爆破。
- [ ] **3.6 高级并发优化 (Advanced Concurrency)**:
    - [ ] 引入自适应速率控制 (Adaptive Rate Limiting)。
    - [ ] 实现 `RttEstimator` 动态调整超时。
    - [ ] 重构 `Runner` 接口，支持优先级队列和任务分片。

### 阶段四：集群接入增强 (Cluster Enhancement) —— **Connection**
**目标**: 实现安全的注册和通信机制。
**状态**: ⚪ **待开始**

- [ ] **4.1 注册流程**: 实现 `join` 命令和 Token 握手逻辑。
- [ ] **4.2 凭证管理**: 实现 API Key / 证书的安全存储与加载。
- [ ] **4.3 通信升级**: 确保所有 Master 通信都使用 mTLS 或 API Key 认证。

### 阶段五：高级能力集成 (Advanced Integration) —— **Ecological**
**目标**: 集成 Nuclei 等重型工具。
**状态**: ⚪ **待开始**

- [ ] **5.1 Nuclei 集成**: 嵌入 Nuclei 库或实现二进制自动下载/调用 wrapper。
- [ ] **5.2 浏览器扫描**: 集成 `chromedp` 实现截图和 DOM 解析。
- [ ] **5.3 插件系统**: 完善对 Nmap/Hydra 等可选工具的调用接口。

---

## 4. 质量控制 (Quality Control)

- **Unit Test**: 核心扫描逻辑覆盖率 > 80%。
- **Benchmark**: 端口扫描速度不低于 `fscan` 水平。
- **Lint**: 通过 `golangci-lint` 检查。
- **No Global State**: 严禁在 Core 中使用全局变量（Logger 除外）。

---

## 5. 当前进度 (Current Status)

| 阶段 | 状态 | 负责人 | 备注 |
| :--- | :--- | :--- | :--- |
| **阶段一：核心解耦** | 🟢 已完成 | Linus & User | 架构基石已稳固 |
| **阶段二：CLI 改造** | 🟢 已完成 | Linus & User | 交互体验优化完毕 |
| **阶段三：原生能力** | 🟡 进行中 | Linus & User | 正在攻坚 Gonmap 移植 |
| **阶段四：集群接入** | ⚪ 待开始 | - | - |
| **阶段五：高级能力** | ⚪ 待开始 | - | - |
