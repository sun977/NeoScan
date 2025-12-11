# PreviousStageProvider 实现方案设计

## 1. 核心目标
`PreviousStageProvider` 的职责是将**上一阶段的扫描结果 (`StageResult`)** 转换为**当前阶段的扫描目标 (`Target`)**。这是工作流（Workflow）实现阶段间数据流转的关键组件。

它需要解决的核心问题是：**如何从非结构化或半结构化的 `StageResult` 中提取、过滤、展开并重组数据？**

---

## 2. 输入与输出

### 输入 (TargetSourceConfig)
从 `TargetSourceConfig` 中我们能获取以下配置：
*   **SourceValue**: 指定来源阶段的名称（如 "stage_1"）或相对引用（"prev"）。
*   **FilterRules**: 过滤规则（如只取 "port_scan" 类型的 Result）。
*   **Unwind**: 数组展开规则（如将 `attributes.ports` 数组拆成多个 Target）。
*   **Generate**: 目标生成模板（如 `{{target_value}}:{{item.port}}`）。

### 输出 ([]Target)
一个标准化的目标列表，供当前 Stage 使用。

---

## 3. 功能需求拆解

根据设计文档，该模块需要实现以下 4 个核心步骤：

### 3.1 结果查询 (Query)
*   **功能**: 从数据库的 `stage_results` 表中查询符合条件的记录。
*   **关键点**:
    *   需要注入 `gorm.DB` 实例。
    *   需要从 Context 中获取当前的 `WorkflowID` 和 `ProjectID`（这需要调度器在调用 `Provide` 前将这些上下文注入 `ctx`）。
    *   解析 `SourceValue`：如果是 "prev"，需要根据当前 Stage 的 Order 找到前一个 Stage 的 ID。
    *   应用 `FilterRules`：在 SQL 层面过滤 `result_type` 等字段。

### 3.2 JSON 展开 (Unwind)
*   **功能**: 处理 "一对多" 场景。例如，上一阶段发现了一个 IP (TargetValue="1.1.1.1")，其 `attributes` 里包含 10 个开放端口。我们需要将这 1 条记录展开为 10 个潜在目标。
*   **实现方案**:
    *   使用 `tidwall/gjson` 库（高性能 JSON 解析）。
    *   根据 `Unwind.Path` (如 `attributes.ports`) 提取数组。
    *   遍历数组，对每个元素应用 `Unwind.Filter` (如 `item.state == "open"`）。

### 3.3 变量渲染 (Render)
*   **功能**: 将提取的数据填充到 `Generate` 模板中。
*   **支持的变量**:
    *   `{{target_value}}`: 原始结果的目标值 (如 IP)。
    *   `{{item}}`: 展开后的当前元素 (如果是基本类型)。
    *   `{{item.field}}`: 展开后的对象字段 (如 `item.port`)。
    *   `{{attributes.field}}`: 原始结果的属性字段。
*   **实现方案**:
    *   简单的字符串替换（`strings.Replace`）或使用轻量级模板引擎（如 `fasttemplate` 或 `text/template`）。鉴于性能要求，推荐简单的正则替换或 `strings.Replace`。

### 3.4 目标封装 (Wrap)
*   **功能**: 生成最终的 `Target` 对象。
*   **关键点**:
    *   设置 `Type` (如 "ip_port")。
    *   设置 `Value` (渲染后的字符串，如 "1.1.1.1:80")。
    *   提取 Meta 数据 (如 `protocol: http`)。

---

## 4. 数据结构增强建议

为了支持上述功能，我们需要在 `TargetSourceConfig` 中补充结构体定义（目前是 `json.RawMessage`，建议定义辅助结构体用于解析）：

```go
type PreviousStageConfig struct {
    ResultType []string `json:"result_type"` // 过滤 ResultType
    StageName  string   `json:"stage_name"`  // 指定 StageName，为空则默认 "prev"
}

type UnwindConfig struct {
    Path   string                 `json:"path"`   // e.g., "attributes.ports"
    Filter map[string]interface{} `json:"filter"` // e.g., {"state": "open"}
}

type GenerateConfig struct {
    Type          string            `json:"type"`           // e.g., "ip_port"
    ValueTemplate string            `json:"value_template"` // e.g., "{{target_value}}:{{item.port}}"
    MetaMap       map[string]string `json:"meta_map"`       // 元数据映射
}
```

---

## 5. 实现步骤规划

### 5.1 Context 注入 (关键步骤)

**位置**: `neoMaster/internal/service/orchestrator/core/scheduler/engine.go` -> `processProject` 方法

**说明**: 
目前的 `ctx` 仅包含基础的取消信号，不包含业务上下文。必须在调用 `ResolveTargets` 之前，将当前阶段的 `WorkflowID` 和 `StageOrder` 注入到 Context 中。

**代码修改逻辑**:
在 `s.targetProvider.ResolveTargets` 调用前 (约 line 313)，添加以下 Context 注入逻辑：

```go
// ... 
// Case C: 初始启动 或 上一个任务完成 -> 寻找下一个 Stage
nextStage, err := s.findNextStage(ctx, project, lastTask)
// ...

// [新增] 注入 Workflow 和 Stage 上下文信息
// 定义 Context Key 类型以避免冲突 (建议在 common 包中定义)
type contextKey string
const (
    CtxKeyWorkflowID   contextKey = "workflow_id"
    CtxKeyCurrentStage contextKey = "current_stage_order"
)

// 注入值
ctx = context.WithValue(ctx, CtxKeyWorkflowID, nextStage.WorkflowID)
ctx = context.WithValue(ctx, CtxKeyCurrentStage, nextStage.StageOrder)

// 2. 使用 TargetProvider 解析最终目标 (应用 TargetPolicy)
resolvedTargetObjs, err := s.targetProvider.ResolveTargets(ctx, nextStage.TargetPolicy, seedTargets)
// ...
```

### 5.2 Provider 实现逻辑

1.  **依赖注入**: 修改 `NewTargetProvider`，将 `gorm.DB` 传递给 `PreviousStageProvider`。
2.  **获取上下文**: 在 `Provide` 方法中从 `ctx` 提取 `workflow_id` 和 `current_stage_order`。
3.  **查询逻辑**: 实现 `findStageResults(ctx, workflowID, stageID)`。
4.  **处理逻辑**: 实现 `processResult(result, config)`，包含 Unwind 和 Render。
5.  **集成**: 在 `Provide` 方法中串联上述逻辑。

这个方案既满足了灵活性（通过 JSON 配置定义提取规则），又保证了性能（数据库过滤 + 高效 JSON 解析）。
