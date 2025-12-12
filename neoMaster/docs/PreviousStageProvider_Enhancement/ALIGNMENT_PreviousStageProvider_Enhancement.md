# 需求对齐文档 - PreviousStageProvider 增强

## 原始需求
P1 任务 ：增强 PreviousStageProvider 。
1. 修改 PreviousStageConfig 增加 StageStatus 。
2. 升级 UnwindConfig 的 Filter 为复杂结构。
3. 实现 Provide 中的高级过滤逻辑。

## 项目上下文
### 技术栈
- 编程语言：Go
- 框架：Gin, Gorm
- 核心组件：PreviousStageProvider, Matcher Engine
- 数据模型：StageResult, AgentTask, ScanStage

### 现有架构理解
- **PreviousStageProvider**: 负责从数据库查询上一阶段的扫描结果，并将其转换为当前阶段的输入目标。
- **PreviousStageConfig**: 目前包含 `ResultType` 和 `StageName`。
- **UnwindConfig**: 目前包含 `Path` 和 `Filter` (简单 map)。
- **Matcher Engine**: 位于 `internal/pkg/matcher`，支持复杂的嵌套逻辑规则 (AND/OR, Regex, Range 等)。
- **数据流**: `Provide` 方法查询 `StageResult` -> `processResult` 展开和过滤 -> 返回 `[]Target`。

## 需求理解
### 功能边界
**包含功能：**
1.  **StageStatus 过滤**:
    - 在 `PreviousStageConfig` 中增加 `StageStatus` 字段 (字符串数组)。
    - 在 `Provide` 方法中，不仅要找到 `ScanStage`，还需要验证该 Stage 对应的执行任务 (`AgentTask`) 状态是否符合要求。
    - *假设*: 如果未指定 Status，默认不限制（或者默认只取 completed？需确认，暂定默认不限制）。
    - *注意*: `StageResult` 表没有 Status 字段，需要关联查询 `AgentTask` 表来获取状态。

2.  **复杂 Filter 支持**:
    - 将 `UnwindConfig` 中的 `Filter map[string]interface{}` 升级为 `Filter *matcher.MatchRule` (或兼容结构)。
    - 利用 `internal/pkg/matcher` 包的能力。

3.  **高级过滤逻辑**:
    - 在 `processResult` 中，使用 `matcher.Match(item, rule)` 替代原有的简单 key-value 遍历比较。
    - 支持对展开后的 JSON 对象进行复杂逻辑判断。

**明确不包含（Out of Scope）：**
- 修改 `StageResult` 表结构（不增加 Status 字段）。
- 修改 `AgentTask` 表结构。
- 跨项目或跨工作流的查询（限制在当前 Context）。

## 疑问澄清
### P0级问题（必须澄清）
1.  **StageStatus 的数据源**:
    - 背景: `StageResult` 表没有 status。
    - 假设: 需要查询 `AgentTask` 表，通过 `WorkflowID`, `StageID` (可能还有 `ProjectID`) 找到对应的任务，并检查其 `Status`。
    - 风险: 如果一个 Stage 有多个 AgentTask (分布式执行)，如何判断？
    - *策略*: 只要有任意一个 AgentTask 满足状态且产生了 Result？或者必须所有？
    - *简化假设*: 通常一个 Stage 在一次 Workflow 执行中对应一个 AgentTask (或者一组)。`PreviousStageProvider` 主要关注结果。如果配置了 `StageStatus: ["completed"]`，则只有当产生该结果的 Task 状态为 completed 时才采纳？
    - *实际操作*: 我们可以先查询符合条件的 `AgentTask`，获取其 `StageID` (如果我们是按名查找) 或者确认其状态。
    - *更正*: `StageResult` 有 `AgentID`。我们可以检查产生该 Result 的 `AgentID` 在 `AgentTask` 中的状态。
    - **问题深度解析与规避策略**:
        - **风险识别**: 数据流 (`StageResult`) 与控制流 (`AgentTask`) 分离导致潜在的 "脏读" (Dirty Read) 和 "竞态条件" (Race Condition)。如果 Provider 只读取 Result 而不检查 Task 状态，可能会读取到正在运行任务产生的不完整数据，或者失败任务产生的错误数据。
        - **规避策略**: 采用 "AgentID 桥接策略"。
            1.  **Check Status**: 先查询 `AgentTask` 表，获取指定 Stage 下状态为 `completed` (或配置状态) 的所有合法 `AgentID`。
            2.  **Fetch Data**: 在查询 `StageResult` 表时，强制增加 `WHERE agent_id IN (...)` 过滤条件。
        - **效果**: 确保 "只见结果，如见其人"。只有任务状态可靠，其产出的数据才被视为有效。这消除了数据不一致性，符合 "Fail Fast" 和 "Good Taste" 的设计原则。

2.  **UnwindConfig.Filter 的兼容性**:
    - 背景: 原有 `Filter` 是 `map[string]interface{}`。
    - 建议: 将 `Filter` 字段类型改为 `interface{}`，自定义 Unmarshal 逻辑，或者引入新字段 `MatchRule` 并废弃旧字段。
    - *决策*: 为了代码清晰，建议直接修改 `Filter` 类型为 `matcher.MatchRule` (如果可以破坏兼容性) 或者 `ComplexFilter`。
    - *指令*: "升级 UnwindConfig 的 Filter 为复杂结构"。
    - *方案*: 直接替换类型。因为这是一个新项目/内部项目，且用户不仅是用户也是开发者 (Linus Persona)，我们可以重构。直接将 `Filter` 类型改为 `matcher.MatchRule`。

## 验收标准
### 功能验收
- [ ] `PreviousStageConfig` 支持配置 `stage_status` (e.g. `["completed", "running"]`)。
- [ ] 如果指定了 `stage_status`，只有对应状态的任务产生的结果会被返回。
- [ ] `UnwindConfig` 支持配置复杂的 JSON 规则 (e.g. nested AND/OR)。
- [ ] `Provide` 方法能正确解析并执行复杂规则过滤，剔除不匹配的数据。
- [ ] 单元测试或集成测试证明复杂规则有效。

### 质量验收
- [ ] 代码通过编译。
- [ ] 不破坏现有简单过滤场景（简单 KV 也是一种 Rule）。
- [ ] 错误处理完善（Rule 解析失败等）。
