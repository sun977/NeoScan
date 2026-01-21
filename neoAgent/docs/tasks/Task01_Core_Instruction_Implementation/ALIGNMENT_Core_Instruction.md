# 需求对齐文档 - [NeoAgent 核心指令集建设]

## 原始需求
1. **统一 CLI 设计**：采用 Cobra 风格的子命令结构 (`neoAgent scan [type]`)。
2. **原子能力整合**：参考 `sbscan` 的功能，将扫描能力和代理穿透能力作为 Agent 的原子能力。
3. **指令集规范**：明确定义 Agent 支持的所有指令及其参数，确保 CLI 和 Cluster 模式的统一。

## 项目上下文
### 技术栈
- **编程语言**：Go (Golang)
- **CLI 框架**：Cobra
- **核心模型**：`internal/core/model/task.go` (Task 结构体)
- **文档规范**：`docs/Agent指令集规范.md`

### 现有架构理解
- **Agent 模式**：NeoAgent 支持两种模式：
    - **Cluster Mode**：作为 Worker 连接 Master，接收 JSON 指令。
    - **CLI Mode**：作为独立工具，通过命令行参数执行任务。
- **指令映射**：CLI 参数和 Master JSON 指令需要映射到统一的 `model.Task` 结构。
- **核心能力**：
    - 扫描类：资产、端口、Web、目录、漏洞、子域名。
    - 代理类：Socks5、HTTP、端口转发。

## 需求理解
### 功能边界
**包含功能：**
- [x] **指令集定义**：在 `Agent指令集规范.md` 中完成所有原子指令的定义。
- [x] **Task 模型更新**：在 `model/task.go` 中支持所有新的 `TaskType`。
- [ ] **CLI 实现**：实现 `neoAgent scan` 和 `neoAgent proxy` 的 Cobra 命令结构。
- [ ] **参数解析**：实现将 CLI Flag 解析为 `model.Task` 的逻辑。
- [ ] **Proxy 实现**：实现基础的 Proxy 启动逻辑（仅框架，具体网络逻辑后续实现）。

**明确不包含（Out of Scope）：**
- [ ] **具体的扫描逻辑实现**：本次任务只负责“指令层”和“调度层”的通畅，不涉及具体的端口扫描算法、Web 爬虫实现等（这些是后续的 Runner 实现任务）。
- [ ] **Master 端适配**：本次专注于 Agent 端的指令接收和模型定义。

## 疑问澄清
### P0级问题（必须澄清）
1. **Proxy 任务的生命周期**
   - **背景**：扫描任务通常是短周期的，有明确的开始和结束。Proxy 任务通常是长周期的（Daemon）。
   - **问题**：`model.Task` 是否足以描述 Proxy 任务？是否需要 `IsDaemon` 字段或特殊的超时处理？
   - **建议方案**：在 `Params` 中通过 `timeout=0` 或不设置 timeout 来表示无限期运行。Agent 内部调度器需要能区分处理。

2. **指令参数的验证**
   - **背景**：CLI 可以利用 Cobra 的验证机制，Master JSON 需要另外的验证。
   - **建议方案**：在 `model` 层实现统一的 `Validate()` 方法，无论来源如何，都通过此方法验证。

## 验收标准
### 功能验收
- [ ] **文档完备**：`Agent指令集规范.md` 包含所有原子能力（Asset, Port, Web, Dir, Vuln, Subdomain, Proxy）。
- [ ] **模型支持**：`model.TaskType` 包含所有对应的常量。
- [ ] **CLI 可用**：
    - 运行 `neoAgent scan port --help` 能看到正确参数。
    - 运行 `neoAgent proxy --help` 能看到正确参数。
- [ ] **参数映射**：
    - CLI 输入 `neoAgent scan port -t 1.1.1.1 -p 80` 能正确转换为 `Task{Type: "port_scan", Target: "1.1.1.1", PortRange: "80"}`。

### 质量验收
- [ ] 代码符合 Go 规范。
- [ ] 无魔术字符串，统一使用常量。
- [ ] 目录结构清晰，命令代码与业务逻辑解耦。
