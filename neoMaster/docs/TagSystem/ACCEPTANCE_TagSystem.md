# 验收文档 - 标签系统 (Tag System)

## 任务执行记录

### 任务3: 规则传播与回溯 (Propagation & Backfill)
**执行状态**: 已完成
**完成时间**: 2025-12-16

#### 1. 代码实现
- **LocalAgent 更新**:
  - 更新 `TagPropagationPayload` 结构，增加 `RuleID` 和 `TagIDs` 字段。
  - 增加 `syncEntityTags` 方法，用于在 `processAssetHost/Web/Network` 后同步 `sys_entity_tags` 表。
  - 支持 `add` 和 `remove` 操作。
- **TagService 更新**:
  - 更新 `SubmitPropagationTask` 签名，支持 `action` 参数。
  - 在 `CreateRule`, `UpdateRule`, `DeleteRule` 中触发传播任务。
  - 修正了 Payload 构造逻辑。
- **OpenAPI 更新**:
  - 补充了 `DELETE /api/v1/tags/rules/{id}` 接口定义。

#### 2. 验证结果
- **编译检查**: `go build` 成功。
- **逻辑验证**:
  - 规则创建/更新时，会生成 `sys_tag_propagation` 任务，Action 为 `add`。
  - 规则删除时，会生成 `sys_tag_propagation` 任务，Action 为 `remove`。
  - LocalAgent 执行任务时，会批量扫描资产，匹配规则。
  - 匹配成功后，更新资产表的 `tags` 字段 (JSON)。
  - 同时在 `sys_entity_tags` 表中插入或删除记录，记录 `RuleID` 和 `Source='auto'`。

### 任务4: 性能优化与缓存 (Performance & Cache)
**执行状态**: 已完成
**完成时间**: 2025-12-18

#### 1. 代码实现
- **MatchRuleCache**: 
  - 引入 `MatchRuleCache` 结构体，使用 `sync.RWMutex` 保护内存中的规则映射。
  - 实现了 `ReloadMatchRules` 方法，将 JSON 解析逻辑前置，避免每次匹配时的 O(N*M) 解析开销。
  - 在 `NewTagService` 初始化时自动加载规则。
- **自动刷新**:
  - 在 `CreateRule`, `UpdateRule`, `DeleteRule` 操作成功后，自动调用 `ReloadMatchRules` 刷新缓存。
- **日志规范化**:
  - 全面替换 `fmt.Printf` 为 `internal/pkg/logger`，确保生产环境日志的可观测性。

#### 2. 验证结果
- **性能**: 自动打标 (`AutoTag`) 直接使用预解析的 `CachedRule`，消除了 JSON 解析瓶颈。
- **一致性**: 规则变更后缓存即时更新，无需重启服务。

### 任务5: 层级标签与查询优化 (Hierarchical Tags)
**执行状态**: 已完成
**完成时间**: 2025-12-18

#### 1. 代码实现
- **方案选择**: 采用 **Scheme 2 (Materialized Path)**。
- **路径维护**:
  - `CreateTag` 自动计算 `Path` (例如 `/1/5/`)。
  - `GetTagsByIDs` 自动根据 `Path` 填充 `FullPathName` (例如 `OS/Linux/Ubuntu`)。
- **查询优化**:
  - `GetEntityIDsByTagIDs` 优化：利用 `path LIKE 'prefix%'` 一次性查出所有子标签 ID，代替递归查询。
  - 实现了子标签的自动包含查询 (查 "Linux" 会自动包含 "Ubuntu" 的资产)。
- **级联删除**:
  - `DeleteTag` 增加 `force` 参数。
  - `force=true` 时，利用 `Path` 快速查找并删除所有后代标签及关联规则、实体关联。

#### 2. 验证结果
- **单元测试**: `TestAgentTagRefactor` 和 `TestAgentTaskSupport` 全部通过。
- **功能验证**: 级联删除逻辑正确，子标签查询逻辑覆盖预期。

### 任务6: Agent 任务支持集成 (Agent Task Support)
**执行状态**: 已完成
**完成时间**: 2025-12-18

#### 1. 代码实现
- **接口对齐**:
  - 修复 `MockAgentRepo` 缺失的方法 (`AddTaskSupport`, `GetAllScanTypes`, `HasTaskSupport`)。
  - 修复 `DeleteTag` 签名变更导致的调用错误。
- **测试通过**:
  - `20251217_Agent_TaskSupport_test.go` 测试通过。

### 任务7: UpdateTag 字段限制 (Restricted Fields)
**执行状态**: 已完成
**完成时间**: 2025-12-18

#### 1. 代码实现
- **限制修改**: 
  - 修改 `UpdateTag` 方法，仅允许更新 `Name`, `Color`, `Category`, `Description`。
  - 使用 `r.db.Model(tag).Select(...).Updates(tag)` 明确指定字段。
  - 严禁修改 `ParentID`, `Path`, `Level` 等结构性字段，防止数据不一致。

#### 2. 验证结果
- **安全性**: 即使上层业务逻辑错误地修改了 ParentID，底层 Repo 也会忽略该修改，保护树结构完整性。
- **测试**: 现有测试通过 (现有测试未强依赖 ParentID 修改)。

### 任务8: 标签移动与层级维护 (MoveTag Safety)
**执行状态**: 已完成
**完成时间**: 2025-12-18

#### 1. 代码实现
- **专用接口**: 新增 `MoveTag` 接口，专门用于处理标签树结构的调整。
- **事务保障**: 
  - 使用 `gorm.DB.Transaction` 确保操作原子性。
  - 使用 `clause.Locking{Strength: "UPDATE"}` 对目标记录加锁，防止竞态条件。
  - 自动更新所有后代节点的 `Path` 和 `Level`。
  - **循环检测**: 严格检查目标父节点是否为当前节点的后代，防止死循环。

#### 2. 验证结果
- **测试通过**: `20251218_Tag_Move_test.go` 覆盖了正常移动、移动到根节点、移动到后代(预期失败)等场景。
- **数据一致性**: 验证了移动后所有后代节点的 Path 均正确更新。

### 任务9: TagService 测试用例 (Test Cases)
**执行状态**: 已完成
**完成时间**: 2025-12-18

#### 1. 代码实现
- **测试文件**: 创建 `test/20251218/20251218_TagService_test.go`。
- **覆盖范围**:
  - **CRUD**: 创建、更新、删除标签，验证 Path 自动计算。
  - **AutoTag**: 验证规则创建、缓存自动刷新、规则匹配及标签应用。
  - **SyncEntityTags**: 验证实体标签同步逻辑。
- **环境**: 使用 `neoscan_dev` 本地开发库进行集成测试。

#### 2. 验证结果
- **执行通过**: 所有测试用例均通过。
- **缺陷修复**: 修复了 AutoTag 测试中 Operator 使用 "eq" 导致的不匹配问题，修正为 "equals"。

## 待办事项 (Pending)
无核心阻塞问题。

## 遗留问题 (Known Issues)
无。

