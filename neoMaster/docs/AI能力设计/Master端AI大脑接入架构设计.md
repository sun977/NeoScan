# NeoScan Master 端 AI 大脑接入架构设计

> **版本**：v1.0.0
> **日期**：2026-08-14
> **状态**：📝 设计阶段（未实现）
> **关联文档**：
> - `neoAgent/docs/AI能力设计/AI大脑接入架构设计.md`（Agent 端 AI 设计）
> - `docs/Asset资产和核心编排器模块设计/Scheduler_Engine_Architecture_v2.0.md`
> - `docs/Asset资产和核心编排器模块设计/Master-Agent交互与多阶段调度.md`

---

## 1. 为什么 Master 端更需要 AI

### 1.1 现有架构中 Master 承担的职责

读完代码，Master 当前做的事情是：

```
Project (用户配置工作流 + 目标)
    ↓
SchedulerService (DAG 调度引擎，每 10 秒轮询)
    ↓  findNextStages() → 基于 Predecessors 依赖判定
TaskGenerator (生成 AgentTask 写入 DB)
    ↓  PolicyEnforcer.Enforce() → 合规校验
TaskDispatcher (Agent 来拉取时分发)
    ↓
Agent 执行，结果回传
    ↓
ResultIngestor → ResultQueue (MemoryQueue)
    ↓
ETL Processor (Mapper → Merger → 写入 Asset 表)
```

**关键观察**：

1. **工作流是静态的**：`Workflow` + `ScanStage` + `Predecessors` 在项目创建时由人工配置好，调度引擎只是机械地执行这张 DAG 图，**无法在运行时根据扫描发现动态调整**。
2. **`findNextStages()` 是纯机械判断**：只要前置依赖 `status == "finished"` 就启动下一 Stage，完全不看上一 Stage 扫出了什么内容。
3. **ETL 后的资产数据无人分析**：`StageResult.Attributes` 写入 DB 后就沉睡了，没有任何组件主动分析它并产出安全洞察。
4. **用户交互门槛高**：用户必须提前配置好完整的 `Workflow → ScanStage` DAG，才能发起扫描，对不懂安全的运营人员不友好。

### 1.2 AI 能在哪里发挥作用

基于对代码的完整阅读，AI 在 Master 端有**三个高价值切入点**，从易到难排列：

| 切入点 | 介入位置 | 价值 | 破坏性 |
|--------|---------|------|--------|
| **A. 结果分析**（Analysis） | ETL 完成后，消费 `StageResult.Attributes` | 从结果中提炼洞察、漏洞研判 | 零破坏，纯新增 |
| **B. 工作流生成**（Workflow Generation） | Project 创建时，AI 自动生成 `Workflow + ScanStage` DAG | 降低使用门槛 | 零破坏，纯新增 |
| **C. 动态调度介入**（Dynamic Orchestration） | `findNextStages()` 的决策过程 | 根据扫描结果动态调整后续路径 | 需改造 Scheduler，风险最高 |

**本文档重点设计 A 和 B，C 作为远期方向描述。**

---

## 2. 切入点 A：结果分析（AI Analysis）

### 2.1 问题定义

当前 ETL 流程结束后，`StageResult` 被写入 `stage_results` 表，`AssetHost`/`AssetWeb`/`AssetVuln` 等资产被写入资产表，但**没有任何组件将这些分散的数据汇总成"这个项目发现了什么，哪些需要重点关注"**。

分析师需要手动查询多张表、人工关联漏洞与资产，工作量巨大且容易遗漏。

### 2.2 方案设计

在 ETL Processor 的下游，增加一个 **AI 分析器（AIAnalyzer）**，当一个项目的所有 Stage 执行完毕（`Project.Status = "finished"`）时，触发异步 AI 分析。

#### 数据流

```
Project Status → "finished"
        ↓
SchedulerService.ProcessProject() 感知到完成
        ↓
AIAnalyzer.TriggerAnalysis(projectID)  ← 新增，异步触发
        ↓
从 DB 聚合该项目的所有 StageResult.Attributes
        ↓
LLM.Chat(systemPrompt + 结构化扫描摘要)
        ↓
AIReport { 风险摘要 / 攻击路径 / 优先级列表 / 修复建议 }
        ↓
写入 ai_analysis_reports 表（新增）
        ↓
通过现有 NotifyService 推送给相关人员
```

#### 接入位置（精确到代码）

```go
// internal/service/orchestrator/core/scheduler/engine.go
// ProcessProject() 中，在标记 finished 之后插入一行 Hook：

// 已有代码：
project.Status = "finished"
s.projectRepo.UpdateProject(ctx, project)

// 新增：异步触发 AI 分析（非阻塞，不影响调度器主循环）
if s.aiAnalyzer != nil {
    go s.aiAnalyzer.TriggerAnalysis(context.Background(), uint64(project.ID))
}
```

`schedulerService` 增加一个可选字段 `aiAnalyzer AIAnalyzer`，通过依赖注入传入（`nil` = 禁用），**对现有代码零侵入**。

#### 新增模块位置

```
neoMaster/internal/service/
└── ai/                          # 新增：AI 能力服务包
    ├── README.md
    ├── client.go                # LLM HTTP 客户端（OpenAI 兼容接口）
    ├── analyzer.go              # AI 分析器（消费 StageResult，产出报告）
    └── prompt/
        ├── scan_analysis.go     # 扫描结果分析 Prompt 模板
        └── vuln_report.go       # 漏洞研判 Prompt 模板

neoMaster/internal/model/
└── ai/                          # 新增：AI 相关数据模型
    └── analysis_report.go       # AIAnalysisReport 表结构
```

#### AI 分析器接口定义

```go
// internal/service/ai/analyzer.go

// AIAnalyzer AI 分析器接口
type AIAnalyzer interface {
    // TriggerAnalysis 异步触发对某个项目的 AI 分析
    // 由 SchedulerService 在项目完成时调用
    TriggerAnalysis(ctx context.Context, projectID uint64)
}

// AIReport AI 分析报告
type AIReport struct {
    ProjectID     uint64    `json:"project_id"`
    RiskSummary   string    `json:"risk_summary"`   // 风险全景摘要（自然语言）
    KeyFindings   []Finding `json:"key_findings"`   // 关键发现列表（按优先级排序）
    AttackPaths   []string  `json:"attack_paths"`   // 推演的攻击路径
    Remediation   string    `json:"remediation"`    // 整体修复建议
    TokensUsed    int       `json:"tokens_used"`    // 消耗 Token 数（用于成本监控）
    GeneratedAt   time.Time `json:"generated_at"`
}

// Finding 单条关键发现
type Finding struct {
    Severity    string `json:"severity"`    // critical/high/medium/low
    Target      string `json:"target"`      // 受影响的资产（IP/URL）
    Description string `json:"description"` // 描述
    Confidence  string `json:"confidence"`  // high/medium/low（是否可能是误报）
}
```

#### Prompt 设计要点

AI 收到的输入是结构化的扫描摘要，而不是原始 JSON 堆（避免 Token 浪费）：

```
你是一名资深安全工程师，请分析以下自动化安全扫描结果并给出专业评估。

【扫描项目】{{ project.name }}
【扫描目标】{{ target_scope }}
【扫描时间】{{ start_time }} ~ {{ end_time }}

【存活主机（{{ alive_count }}台）】
{{ alive_hosts_summary }}

【开放端口与服务（Top 20）】
{{ port_service_summary }}

【Web 资产（{{ web_count }}个）】
{{ web_assets_summary }}

【漏洞发现（{{ vuln_count }}个）】
{{ vuln_findings_summary }}

【弱口令（{{ brute_count }}个）】
{{ brute_results_summary }}

请输出：
1. 整体风险评级（Critical/High/Medium/Low）及理由
2. 前 5 条最高风险发现（按实际危害排序，注意排除误报）
3. 是否存在可利用的攻击路径（如 RCE + 内网横向）
4. 3 条最优先修复建议
```

---

## 3. 切入点 B：工作流生成（AI Workflow Builder）

### 3.1 问题定义

当前用户必须手动创建 `Project → Workflow → ScanStage[]` 的完整配置，包括设置每个 Stage 的 `StageType`、`ToolName`、`ToolParams`、`TargetPolicy`、`Predecessors`（DAG 依赖）等十多个字段，门槛极高。

### 3.2 方案设计

提供 **AI 工作流生成器（AIWorkflowBuilder）**：用户只需用自然语言描述意图，AI 自动生成完整的 `Workflow + ScanStage[]` JSON 配置，通过现有的 `WorkflowService` 写入 DB，用户确认后直接运行。

#### 交互流程

```
用户输入（自然语言）：
  "对 192.168.1.0/24 做一次完整安全评估，
   先探活，然后扫端口，发现 Web 服务的再做 Web 扫描，
   发现数据库服务的做弱口令爆破。跳过 192.168.1.1。"

        ↓  POST /api/v1/ai/workflow/generate

AI Workflow Builder
  → LLM 理解意图，生成 DAG 配置

        ↓

返回预览 JSON：
{
  "workflow": { "name": "ai-gen-20260814", "display_name": "AI 生成：完整安全评估" },
  "stages": [
    { "stage_name": "存活扫描", "stage_type": "ipAliveScan", "predecessors": [] },
    { "stage_name": "端口扫描", "stage_type": "serviceScan", "predecessors": [<存活扫描 ID>],
      "target_policy": { "target_sources": [{"source_type": "previous_stage"}] } },
    { "stage_name": "Web 扫描", "stage_type": "webScan",    "predecessors": [<端口扫描 ID>],
      "target_policy": { "skip_enabled": true, "skip_conditions": [...] } },
    { "stage_name": "弱口令扫描", "stage_type": "passScan", "predecessors": [<端口扫描 ID>] }
  ]
}

        ↓  用户确认

POST /api/v1/workflows（已有接口）写入 DB
POST /api/v1/projects（已有接口）关联并启动
```

#### 接入位置（新增 API Handler）

```
neoMaster/internal/handler/ai/
└── workflow_builder.go   # POST /api/v1/ai/workflow/generate
                          # 调用 AIWorkflowBuilder，返回预览配置

neoMaster/internal/service/ai/
└── workflow_builder.go   # AI 工作流生成器服务实现
```

#### LLM 输出格式约束

LLM 必须输出严格符合现有模型的 JSON Schema，对应 `model/orchestrator/scan_stage.go` 中的字段定义。需在 System Prompt 中提供完整的 Schema 文档和 `StageType` 枚举值列表：

```
可用 StageType（必须精确使用以下枚举值）：
- ipAliveScan：IP 存活探测
- fastPortScan：快速端口扫描（Top 100）
- fullPortScan：全量端口扫描（1-65535）
- serviceScan：服务版本识别（指定端口深度指纹）
- webScan：Web 综合扫描
- apiScan：API 接口发现
- vulnScan：漏洞扫描（Nuclei）
- pocScan：POC 精准验证
- passScan：弱口令爆破
- dirScan：目录爆破
- subDomainScan：子域名枚举

Predecessors 使用占位符 "${stage_name}"，系统在保存时自动解析为真实 ID。
```

---

## 4. 切入点 C：动态调度介入（远期）

> **状态**：远期规划，当前不实施，此处仅记录思路。

### 4.1 当前瓶颈

`findNextStages()` 的判定逻辑：
```go
if status, exists := stageStatus[predID]; !exists || status != "finished" {
    dependenciesResolved = false
}
```

这是**纯状态驱动**，只检查"跑完了没有"，不看"跑出了什么"。

### 4.2 动态调度的设计思路

在调度器中引入一个可选的 **`StageRouter`** 接口：

```go
// StageRouter 动态阶段路由器（可选组件）
// 当 findNextStages 找到 Ready 的 Stage 时，由 Router 决定是否真正启动、
// 以及是否动态插入新的临时 Stage
type StageRouter interface {
    // Route 基于当前 Project 的结果，决定下一步行动
    // 返回值：
    //   - approved：是否批准启动 readyStages 中的每个 Stage
    //   - injected：AI 临时注入的额外 Stage（如发现 Redis 自动注入未授权检测 Stage）
    Route(ctx context.Context, project *Project, readyStages []*ScanStage) (approved []*ScanStage, injected []*ScanStage, err error)
}
```

`SchedulerService` 默认 `stageRouter = nil`（禁用动态路由），AI 实现为一个具体的 `AIStageRouter`，通过配置开关启用。

### 4.3 为什么这条路最难

动态注入的 Stage 是**临时的、非持久化的**（或需要持久化），这会导致：
- `findNextStages()` 的 DAG 图在运行时发生变化，难以确保幂等性
- 临时 Stage 的重试、超时、失败处理逻辑需要全套跟进
- 对 LLM 的决策质量要求极高，一个错误的 Tool Call 可能导致对未授权目标的扫描

**结论**：先把 A（分析）和 B（工作流生成）做好，积累足够的数据和使用经验后，再考虑 C。

---

## 5. 模块目录结构（最终规划）

```
neoMaster/internal/
├── service/
│   └── ai/                          # 新增：AI 能力服务包
│       ├── client.go                # LLM 客户端（OpenAI 兼容，支持 env 切换模型）
│       ├── analyzer.go              # AI 分析器（项目完成后触发，产出 AIReport）
│       ├── workflow_builder.go      # AI 工作流生成器（自然语言 → DAG JSON）
│       └── prompt/
│           ├── scan_analysis.go     # 分析类 Prompt 模板
│           └── workflow_gen.go      # 工作流生成类 Prompt 模板（含 Schema 约束）
│
├── handler/
│   └── ai/                          # 新增：AI 相关 HTTP Handler
│       ├── analysis_handler.go      # GET /api/v1/projects/:id/ai-report
│       └── workflow_builder.go      # POST /api/v1/ai/workflow/generate
│
└── model/
    └── ai/                          # 新增：AI 相关数据模型
        └── analysis_report.go       # AIAnalysisReport（写入 DB 的分析结果）
```

---

## 6. LLM 客户端设计

与 Agent 端设计保持一致，共享同一套接口规范，但**各自独立实现**（Agent 端在 `neoagent/internal/pkg/ai/`，Master 端在 `neomaster/internal/service/ai/`）。

```go
// internal/service/ai/client.go

// LLMClient LLM 通信客户端接口
type LLMClient interface {
    // Chat 发送消息，返回回复（可能包含 ToolCall，用于 B 方案）
    Chat(ctx context.Context, messages []Message, tools []Tool) (*Response, error)
}

// Message 对话消息
type Message struct {
    Role       string `json:"role"`        // system | user | assistant | tool
    Content    string `json:"content"`
    ToolCallID string `json:"tool_call_id,omitempty"` // role=tool 时有效
}

// Response LLM 回复
type Response struct {
    Content   string     `json:"content"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
```

**配置项**（新增到 `configs/config.yaml`）：
```yaml
ai:
  enabled: false                    # 总开关，false 时所有 AI 组件静默
  provider: "openai"                # openai | ollama
  base_url: "https://api.openai.com/v1"
  api_key: "${AI_API_KEY}"
  model: "gpt-4o"
  analysis_model: "gpt-4o"         # 分析任务用的模型（可与 workflow_gen 不同）
  workflow_gen_model: "gpt-4o"     # 工作流生成任务用的模型
  max_tokens: 4096
  timeout: "120s"
```

---

## 7. 与现有架构的边界

### 7.1 绝对不能碰的边界

| 约束 | 原因 |
|------|------|
| `SchedulerService.findNextStages()` 的核心 DAG 逻辑不改动（切入点 C 除外） | 调度引擎是整个 Master 的心脏，任何改动都是高风险操作 |
| `ETL Processor` 的 Mapper/Merger 流程不改动 | ETL 是数据完整性的保障，AI 消费的是 ETL 产出，不能混入 ETL 链路 |
| `AgentTask` 的数据结构不改动 | Agent 侧硬依赖此结构，改动即破坏现有 Agent |
| `PolicyEnforcer` 的合规校验不绕过 | 安全红线，即便是 AI 动态生成的任务也必须过策略校验 |

### 7.2 共享基础设施

| 基础设施 | 使用方式 |
|---------|---------|
| `internal/pkg/logger/` | AI 模块复用现有日志基础设施 |
| `internal/service/notify/` | AI 报告生成后，通过现有 NotifyService 推送 |
| `handler/orchestrator/workflow_handler.go` | AI 生成的工作流配置，调用已有的 Workflow 创建接口写入 DB |
| `internal/config/` | AI 配置节挂到现有配置结构，由现有 Viper 加载 |

---

## 8. 开发实施顺序（建议）

### Phase 1：结果分析（切入点 A）

| 步骤 | 内容 | 验证方式 |
|------|------|---------|
| Step 1 | 实现 `service/ai/client.go`，支持 OpenAI Chat API | 单元测试：Mock HTTP，验证请求/响应解析 |
| Step 2 | 实现 `model/ai/analysis_report.go` + DB 迁移 | 运行 migrate，确认表创建成功 |
| Step 3 | 实现 `service/ai/analyzer.go`，硬编码 Prompt，先不接真实 LLM | 单元测试：Mock LLM Client，验证摘要聚合逻辑 |
| Step 4 | 在 `scheduler/engine.go` 中注入 `AIAnalyzer`（可选，nil = 禁用） | 集成测试：跑完一个项目，验证 AI 分析被触发 |
| Step 5 | 实现 `handler/ai/analysis_handler.go`：`GET /api/v1/projects/:id/ai-report` | 接口测试 |
| Step 6 | Prompt 调优：用真实扫描数据评估输出质量 | 人工评估：覆盖率、误报率、建议可操作性 |

### Phase 2：工作流生成（切入点 B）

| 步骤 | 内容 | 验证方式 |
|------|------|---------|
| Step 1 | 实现 `service/ai/workflow_builder.go`，定义 Prompt + JSON Schema 约束 | 单元测试：Mock LLM，验证生成的 JSON 可被现有 WorkflowService 接受 |
| Step 2 | 实现 `handler/ai/workflow_builder.go`：`POST /api/v1/ai/workflow/generate` | 接口测试：各类自然语言输入的健壮性 |
| Step 3 | 前端接入（超出本文档范围） | - |

---

## 9. 风险与约束

| 风险 | 缓解措施 |
|------|---------|
| `ai.enabled = false` 时现有功能零影响 | `AIAnalyzer` 和 `AIWorkflowBuilder` 均以接口形式注入，`nil` 时相关代码路径直接跳过 |
| AI 生成工作流配置格式错误 | Prompt 中嵌入完整 JSON Schema；`WorkflowService` 在写入前执行现有的 Validate 逻辑；生成结果返回给前端预览，用户确认后才写入 |
| LLM 分析报告产生误报/幻觉 | 报告中增加 `confidence` 字段；Prompt 中明确要求 AI 标注不确定项；报告定位为"辅助参考"而非"权威结论" |
| Token 消耗不可控 | 聚合摘要而非原始 JSON 作为 LLM 输入；设置 `max_tokens` 上限；记录每次 `tokens_used` 到 `analysis_reports` 表用于成本追踪 |
| 并发多个项目同时完成导致 LLM 请求洪峰 | `TriggerAnalysis` 使用独立 Goroutine Pool 控制并发；加入请求限流（令牌桶）|
| AI 生成的工作流绕过安全策略 | 无论工作流来源（人工/AI），进入 `generateTasksForStage` 后都必须经过 `PolicyEnforcer.Enforce()`，不设豁免 |

---

## 10. 参考

- 调度引擎：`internal/service/orchestrator/core/scheduler/engine.go`
- 工作流模型：`internal/model/orchestrator/workflow.go`、`scan_stage.go`
- Stage 结果模型：`internal/model/orchestrator/stage_result.go`
- ETL 处理器：`internal/service/asset/etl/processor.go`
- 通知服务：`internal/service/notify/`
- Agent 端 AI 设计：`neoAgent/docs/AI能力设计/AI大脑接入架构设计.md`
