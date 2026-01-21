# NeoAgent CLI 架构决策说明 (Cobra vs goflags)

## 1. 决策背景

NeoAgent 需要从一个单纯的 "Worker 进程" 进化为一个多模态的 "安全平台工具"。
我们面临两个主要的技术选型方向：
1.  **Cobra** (spf13): Go 语言 CLI 的工业标准 (Kubernetes/Docker/Hugo)。
2.  **goflags** (ProjectDiscovery): 安全工具圈的流行选择 (Nuclei/Subfinder)。

**最终决策**: **采用 Cobra**。

---

## 2. 为什么选择 Cobra？

### 2.1 架构清晰度 (Architectural Clarity)
NeoAgent 的核心特征是**多模式 (Multi-Mode)**：
- **Server Mode**: 启动 HTTP 服务，长连接 Master，作为 Worker。
- **Scan Mode**: 单机运行，一次性任务，作为 Scanner。
- **Join Mode**: 握手注册，证书交换。

这种差异巨大的行为模式，非常适合 Cobra 的 **"Command-Based"** 架构：
```bash
neoAgent server [flags]  # 启动服务
neoAgent scan [flags]    # 扫描任务
neoAgent join [flags]    # 集群注册
```
而 `goflags` 更适合单一职责的工具，通过 Flag 组合功能。如果在 `goflags` 中实现多模式，会导致 `main` 函数充斥着大量的 `if/else` 判断逻辑，代码极难维护。

### 2.2 扩展性与规范性 (Extensibility & Standard)
- **子命令嵌套**: Cobra 天生支持 `noun verb` 结构（如 `neoAgent config show`），方便未来扩展管理功能。
- **自动文档**: Cobra 可以自动生成 Man Pages 和 Markdown 文档，甚至 Shell 补全脚本。对于一个企业级 Agent，这是必须的。
- **代码组织**: Cobra 强制将每个命令的逻辑封装在独立的文件中 (`cmd/server.go`, `cmd/scan.go`)，避免了 `main.go` 的臃肿。

### 2.3 参数透传支持 (Dash Dash Support)
虽然 `goflags` 在参数解析上更灵活，但 Cobra 对标准 Unix `--` (Dash Dash) 模式支持良好，这对于未来支持 "Raw Command Wrapper" 至关重要。

---

## 3. 未来复杂场景预演与应对

### 3.1 场景：复杂参数透传 (The Wrapper Scenario)
**挑战**: 用户希望 Agent 作为一个 Wrapper，执行极其复杂的 Nmap 命令，而不经过 Agent 的标准化参数层。
**示例**:
```bash
neoAgent scan raw --tool nmap -- -sS -sV --script "vuln,safe" -p- --min-rate 1000 192.168.1.1
```
**应对策略**:
- 利用 Cobra 的 `DisableFlagParsing: true` 特性（针对特定子命令）。
- 在 `scan raw` 子命令中，捕获 `--` 后的所有参数，直接透传给底层 Executor。
- **价值点**: 即使透传，Agent 依然负责**资源限制 (Cgroups)**、**日志收集**、**结果回传**。

### 3.2 场景：配置文件与 Flag 的优先级 (Config vs Flag)
**挑战**: 用户既在 `config.yaml` 里配了代理，又在命令行里指定了 `--proxy`。
**应对策略**:
- 使用 `Viper` (Cobra 的黄金搭档) 进行配置绑定。
- 优先级明确：`Flag > Env > Config File > Default`。
- Cobra + Viper 可以自动处理这种层级覆盖，无需手写逻辑。

### 3.3 场景：交互式配置 (Interactive Setup)
**挑战**: `neoAgent join` 需要用户输入 Token，用户可能不想明文写在历史记录里。
**应对策略**:
- Cobra 的 `Run` 函数中可以轻松集成 `survey` 或 `promptui` 库。
- 如果检测到 Flag 为空且由 TTY 启动，自动进入交互模式询问用户。

---

## 4. 结论

选择 Cobra 不是为了"通用"，而是为了**"可治理性"**。
NeoAgent 的目标是成为一个**平台 (Platform)**，而不仅仅是一个脚本 (Script)。Cobra 提供的结构化框架能支撑 Agent 在未来数年内的功能演进，而不会崩塌成一堆混乱的 Flags。
