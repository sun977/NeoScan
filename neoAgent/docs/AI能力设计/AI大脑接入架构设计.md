# NeoAgent AI 大脑接入架构设计

> **版本**：v1.0.0
> **日期**：2026-08-14
> **状态**：📝 设计阶段（未实现）
> **作者**：架构设计

---

## 1. 背景与目标

### 1.1 现状

当前 NeoAgent 的扫描编排逻辑是**硬编码的线性流程**，由 `internal/core/pipeline/auto_runner.go` 固定实现：

```
Alive → Port → Service → OS → ServiceDispatcher(Web / Brute)
```

`ServiceDispatcher` 的决策逻辑（是否扫 Web、是否爆破哪个协议）全部写死在 `dispatcher.go` 的条件分支里。这种模式在已知场景下高效，但**无法动态调整策略、无法应对复杂未知目标**。

### 1.2 目标

引入 AI 作为扫描编排的**决策大脑**，使其能够：

1. **自主决定**每一步扫什么：看完 Port 结果再决定是否扫 Web，看完 Web 结果再决定是否爆破。
2. **动态组合能力**：基于上一步结果，从可用的原子 Scanner 中自由选择下一个行动。
3. **渐进式深入**：发现 Redis 未授权 → AI 决定进一步尝试写入；发现登录页 → AI 决定启用深度爬取+截图。
4. **零破坏现有逻辑**：现有 `scan run`、`scan web` 等命令一行代码不动。

---

## 2. 核心设计思路

### 2.1 AI 的角色定位

AI 是一个**新的编排者（Orchestrator）**，与现有的 `AutoRunner` 平级，不是 `AutoRunner` 的子模块。

```
┌─────────────────────────────────────────────────┐
│                  cmd/agent/                       │
│                                                   │
│  scan/run.go          ai/root.go                  │
│  （硬编码编排者）       （AI 编排者）  ← 新增      │
│        │                    │                     │
│        └────────┬───────────┘                     │
│                 ↓                                 │
│         internal/core/factory/                    │
│         （Scanner 能力工厂，双方共用）              │
└─────────────────────────────────────────────────┘
```

两条路径共用同一套底层能力（`factory/` + `runner/` + `scanner/`），互不干扰。

### 2.2 AI 工作模式：ReAct 循环

AI 采用 **ReAct（Reasoning + Acting）** 模式运行：

```
┌─────────────────────────────────────────────────┐
│                  ReAct Loop                       │
│                                                   │
│  Observation (观察)                               │
│    └─ 当前 PipelineContext（IP、端口、服务、结果） │
│         │                                         │
│         ▼                                         │
│  Reasoning (思考)                                 │
│    └─ LLM 分析结果，决定下一个行动                │
│         │                                         │
│         ▼                                         │
│  Action (行动)                                    │
│    └─ 调用某个 Scanner Tool（alive/port/web/...） │
│         │                                         │
│         ▼                                         │
│  Observation (观察新结果) ──────────────────────┐ │
│                                                 │ │
│  [终止条件]                                     └─┘│
│    - LLM 输出 "DONE" 指令                         │
│    - 达到最大轮次上限（防止无限循环）               │
│    - context 超时                                 │
└─────────────────────────────────────────────────┘
```

---

## 3. 目录结构规划

```
neoAgent/
├── cmd/agent/
│   ├── scan/              # 现有：硬编码编排模式（不动）
│   └── ai/                # 新增：AI 驱动编排模式
│       └── root.go        # `scan ai` 子命令入口，承载 ReAct 循环逻辑
│
└── internal/
    └── pkg/
        └── ai/            # 新增：AI 能力工具包（纯工具，无编排逻辑）
            ├── client.go  # LLM HTTP 客户端（支持 OpenAI / Ollama 接口）
            └── tools.go   # 将 Scanner 能力包装为 LLM Tool Calling JSON Schema
```

### 3.1 关键原则

- `internal/pkg/ai/` **只做两件事**：与 LLM 通信、定义 Tool Schema。它不知道 Scanner 的存在，不持有任何 Scanner 实例。
- `cmd/agent/ai/root.go` **负责编排**：持有 Scanner 实例，执行 ReAct 循环，将 LLM 的 Tool Call 映射到真实的 Scanner 调用。
- `internal/core/` **永远是被调用方**：AI 通过 `factory/` 获取 Scanner，和 CLI 模式完全一致，不为 AI 新增任何接口。

---

## 4. 模块详细设计

### 4.1 `internal/pkg/ai/client.go`

**职责**：封装与 LLM API 的 HTTP 通信，屏蔽模型差异。

```go
// LLMClient LLM 通信客户端接口
type LLMClient interface {
    // Chat 发送消息列表，返回 LLM 的回复（可能包含 ToolCall）
    Chat(ctx context.Context, messages []Message) (*Response, error)
}

// Message 对话消息
type Message struct {
    Role    string // "system" | "user" | "assistant" | "tool"
    Content string
    // ToolCallID 仅 role=tool 时有效，关联 LLM 发起的 ToolCall
    ToolCallID string
}

// Response LLM 回复
type Response struct {
    Content   string     // 文字回复（当 LLM 直接回答时）
    ToolCalls []ToolCall // 工具调用请求（当 LLM 决定行动时）
    Done      bool       // LLM 是否表示任务完成
}

// ToolCall LLM 发起的工具调用
type ToolCall struct {
    ID        string          // 调用 ID
    Name      string          // 工具名（对应 Scanner 名）
    Arguments json.RawMessage // 参数（JSON）
}
```

**支持的后端**：
- OpenAI 兼容接口（OpenAI、Azure OpenAI、本地 Ollama、自部署 vLLM 等）
- 通过配置文件切换，代码层面只有一套实现

**配置项**（`configs/config.yaml` 新增节）：
```yaml
ai:
  enabled: false              # 总开关
  provider: openai            # openai | ollama
  base_url: "https://api.openai.com/v1"
  api_key: "${AI_API_KEY}"    # 支持环境变量
  model: "gpt-4o"
  max_rounds: 20              # ReAct 循环最大轮次
  timeout: 300s               # 单次 AI 决策超时
```

---

### 4.2 `internal/pkg/ai/tools.go`

**职责**：将 Scanner 的能力定义为 LLM 能理解的 **Tool（函数）描述**，供 LLM 在 Tool Calling 时选择。

工具清单（对应现有 Scanner 能力）：

| Tool 名 | 对应 Scanner | 关键参数 |
|---------|-------------|---------|
| `scan_alive` | `alive.IpAliveScanner` | `target` (IP/CIDR) |
| `scan_port` | `port_service.PortServiceScanner` | `target`, `port_range`, `service_detect` |
| `scan_os` | `os.Scanner` | `target` |
| `scan_web` | `web.WebScanner` | `target`, `port`, `crawl`, `crawl_depth`, `screenshot` |
| `scan_api` | `api.ApiScanner` | `target`, `port` |
| `scan_brute` | `brute.BruteScanner` | `target`, `port`, `service`, `users`, `passwords` |
| `scan_vuln` | `vuln.Scanner`（未来） | `target`, `port`, `tech_stack` |
| `finish` | —（内置） | `summary`（扫描结论） |

每个 Tool 的 JSON Schema 示例（`scan_web`）：
```json
{
  "type": "function",
  "function": {
    "name": "scan_web",
    "description": "对指定目标的 HTTP/HTTPS 端口进行 Web 综合扫描，包括指纹识别、CDN 检测、表单提取、被动泄露检测，可选截图和深度爬取。",
    "parameters": {
      "type": "object",
      "properties": {
        "target": { "type": "string", "description": "目标 IP 或域名" },
        "port":   { "type": "integer", "description": "目标端口号" },
        "crawl":  { "type": "boolean", "description": "是否启用 BFS 深度爬取", "default": false },
        "crawl_depth": { "type": "integer", "description": "爬取深度（1-5）", "default": 2 },
        "screenshot": { "type": "boolean", "description": "是否截图", "default": false }
      },
      "required": ["target", "port"]
    }
  }
}
```

---

### 4.3 `cmd/agent/ai/root.go`

**职责**：`scan ai` 子命令入口，承载完整的 ReAct 循环逻辑。

**主流程**：

```go
// 伪代码，展示 ReAct 循环结构
func runAI(ctx context.Context, target string) error {
    // 1. 初始化 LLM 客户端和所有 Scanner
    client := ai.NewLLMClient(cfg.AI)
    tools  := ai.BuildTools()         // 生成 Tool Schema 列表
    scannerMap := buildScannerMap()   // name -> Scanner 实例

    // 2. 初始化 PipelineContext（结果累积容器）
    pCtx := pipeline.NewPipelineContext(target)

    // 3. 构建初始 System Prompt
    messages := []ai.Message{
        {Role: "system", Content: buildSystemPrompt()},
        {Role: "user",   Content: fmt.Sprintf("请对目标 %s 进行安全扫描评估。", target)},
    }

    // 4. ReAct 循环
    for round := 0; round < cfg.AI.MaxRounds; round++ {
        resp, err := client.Chat(ctx, messages, tools)
        if err != nil || resp.Done { break }

        if len(resp.ToolCalls) == 0 {
            // LLM 直接回复文字，说明它认为任务完成或需要用户反馈
            break
        }

        // 执行 LLM 请求的工具调用
        for _, tc := range resp.ToolCalls {
            result := executeTool(ctx, tc, pCtx, scannerMap)
            // 将结果反馈回对话历史
            messages = append(messages,
                ai.Message{Role: "assistant", ToolCalls: []ai.ToolCall{tc}},
                ai.Message{Role: "tool", ToolCallID: tc.ID, Content: result},
            )
        }
    }

    // 5. 输出最终结果
    printFinalReport(pCtx)
    return nil
}
```

**System Prompt 设计要点**：
- 告知 AI 当前目标、可用工具列表及其能力边界
- 明确扫描优先级策略（存活→端口→服务→评估）
- 要求 AI 每次只发起一组相关 Tool Call，避免无效并发
- 当认为信息足够时调用 `finish` 工具，防止无限探测

---

## 5. 数据流转图

```
用户输入 (target)
      │
      ▼
cmd/agent/ai/root.go
      │  初始化
      ├──────────────────────────────────────────┐
      │                                          │
      ▼                                          ▼
ai.LLMClient                          buildScannerMap()
(pkg/ai/client.go)                    (factory.NewXxxScanner())
      │                                          │
      │◄─────────── ReAct 循环 ─────────────────►│
      │                                          │
      │  1. Chat(messages, tools)                │
      │     LLM 返回 ToolCall{name, args}         │
      │                                          │
      │  2. executeTool(name, args)              │
      │     → scanner.Run(task)                  │
      │     → 更新 PipelineContext               │
      │                                          │
      │  3. 将结果序列化为字符串追加到 messages   │
      │     继续下一轮 Chat                       │
      │                                          │
      ▼                                          │
PipelineContext (结果累积) ◄────────────────────┘
      │
      ▼
Reporter (最终输出)
```

---

## 6. 与现有架构的边界

### 6.1 不能越过的边界

| 边界 | 原因 |
|------|------|
| `internal/core/` 中不能引入 `pkg/ai/` | `core` 是零外部依赖的纯扫描引擎，AI 是上层消费者 |
| `pkg/ai/` 中不能持有 Scanner 实例 | `pkg` 是工具包，不做业务编排 |
| 现有 `scan/run.go` 不能因 AI 而修改 | 向后兼容是铁律，AI 是新增能力，不是改造 |

### 6.2 共享的基础设施

| 基础设施 | 共享方式 |
|---------|---------|
| `core/factory/` | AI 编排者通过 Factory 获取 Scanner，与 CLI 模式完全一致 |
| `core/model/` | `PipelineContext`、`TaskResult` 等数据结构共用 |
| `core/pipeline/PipelineContext` | AI 模式复用同一个 Context 结构累积结果 |
| `config/` | AI 配置节新增到现有 `config.yaml`，由现有配置系统加载 |
| `pkg/logger/` | AI 模块复用现有日志基础设施 |

---

## 7. 扩展性设计

### 7.1 多模型支持

`ai.LLMClient` 是接口，不同后端（OpenAI / Ollama / Anthropic）各自实现，通过配置文件的 `provider` 字段选择，对上层编排逻辑透明。

### 7.2 Tool 动态注册

`BuildTools()` 函数接收一个 Scanner 注册表，未来新增 `scan_vuln`、`scan_dir`、`scan_subdomain` 时，只需在注册表中添加对应条目，ReAct 循环无需修改。

### 7.3 与 Master 集群模式的结合（远期）

当前设计仅针对 CLI 单机模式（`cmd/agent/ai/`）。远期可以在 `internal/service/` 层新增 `ai_orchestrator.go`，让 Master 下发 "AI 模式任务"，Agent 在集群 Worker 模式下同样走 ReAct 循环，而 Scanner 调用路径完全不变。

---

## 8. 开发实施顺序（建议）

当决定实施时，建议按以下顺序推进，每步可独立验证：

| 步骤 | 内容 | 验证方式 |
|------|------|---------|
| Step 1 | 实现 `pkg/ai/client.go`，支持 OpenAI 兼容接口 | 单元测试：发送消息，验证 JSON 解析 |
| Step 2 | 实现 `pkg/ai/tools.go`，定义全部 Tool Schema | 单元测试：验证 JSON Schema 合法性 |
| Step 3 | 实现 `cmd/agent/ai/root.go` 骨架，先不接真实 Scanner，Mock 工具返回固定结果 | 端到端测试：验证 ReAct 循环逻辑正确 |
| Step 4 | 接入真实 Scanner，替换 Mock | 集成测试：对测试目标执行完整 AI 扫描 |
| Step 5 | 完善 System Prompt，调优 AI 决策质量 | 人工评估：与 `scan run` 结果对比 |
| Step 6 | 配置文件集成，支持 `ai.enabled = false` 降级 | 回归测试：确保关闭 AI 时现有命令不受影响 |

---

## 9. 风险与约束

| 风险 | 缓解措施 |
|------|---------|
| LLM 网络不可用 | `ai.enabled = false` 时完全降级为现有模式，无任何影响 |
| LLM 产生无意义的 Tool Call（如重复扫同一个端口） | ReAct 循环维护已执行 Tool Call 去重表；设置 `max_rounds` 硬上限 |
| Token 消耗超预期 | 每轮只传递增量结果（非全量对话历史）；结果摘要化后再喂给 LLM |
| AI 模型幻觉导致错误参数 | Tool Schema 严格定义参数类型和枚举值；`executeTool` 层做参数校验，非法参数直接返回错误信息给 LLM |
| 扫描时间不可控 | 每个 Tool Call 受现有 Scanner 的 `context` 超时约束；AI 总超时由 `ai.timeout` 控制 |

---

## 10. 参考

- 现有编排层：`internal/core/pipeline/auto_runner.go`、`dispatcher.go`
- 现有能力工厂：`internal/core/factory/`
- 现有扫描模型：`internal/core/model/task.go`、`result_types.go`
- 现有上下文结构：`internal/core/pipeline/pipeline.go`（`PipelineContext`）
- 架构总纲：`docs/Agent重构开发总纲.md`（阶段五之后的规划方向）
