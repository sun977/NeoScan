# 共识文档 - PreviousStageProvider 增强

## 核心共识
我们将在 `PreviousStageProvider` 中引入更严格的状态控制和更强大的数据过滤能力，以支持复杂的编排逻辑。

## 需求规格
1.  **StageStatus 控制**:
    - **字段**: `PreviousStageConfig.StageStatus []string`
    - **逻辑**: 在查询 `StageResult` 之前，先查询 `AgentTask` 表。
    - **流程**:
        1. 解析目标 `StageID`。
        2. 如果配置了 `StageStatus`，查询 `AgentTask` 表，获取 `WorkflowID` + `StageID` 下且 `Status` 在列表中的所有 `AgentID`。
        3. 如果没有符合条件的 Agent，直接返回空。
        4. 查询 `StageResult` 时，增加 `AgentID IN (?)` 的条件。

2.  **高级数据过滤**:
    - **字段**: `UnwindConfig.Filter` 类型变更为 `matcher.MatchRule` (引用 `neomaster/internal/pkg/matcher`)。
    - **逻辑**:
        1. 使用 `gjson` 展开数组。
        2. 对每个元素，将其转换为 `map[string]interface{}` (或直接对 `gjson.Result` 进行包装以适配 Matcher，但 Matcher 目前接受 `map` 或 struct，转 map 更通用)。
        3. 调用 `matcher.Match(data, rule)`。
        4. 仅保留 `Match` 返回 `true` 的元素。

## 技术实现方案
### 数据结构变更
```go
// internal/service/orchestrator/policy/provider_previous_stage.go

import "neomaster/internal/pkg/matcher"

type PreviousStageConfig struct {
    ResultType  []string `json:"result_type"`
    StageName   string   `json:"stage_name"`
    StageStatus []string `json:"stage_status"` // 新增
}

type UnwindConfig struct {
    Path   string            `json:"path"`
    Filter matcher.MatchRule `json:"filter"` // 变更: map -> MatchRule
}
```

### 核心逻辑变更
1.  **resolveTargetStage** 保持不变 (主要负责找 ID)。
2.  **Provide**:
    - 增加 `resolveValidAgents(stageID, statusList)` 步骤。
    - 修改 `StageResult` 查询构建。
3.  **processResult**:
    - 解析 JSON 元素为 map。
    - 调用 `matcher.Match`。

## 任务边界
- 仅修改 Provider 逻辑，不修改数据库 Schema。
- 假设 `matcher` 包已稳定可用。

## 验收标准
1.  配置 `stage_status: ["completed"]` 能过滤掉 running/failed 任务的结果。
2.  配置复杂的 nested filter 能正确筛选数据。
