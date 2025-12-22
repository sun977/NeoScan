# 需求对齐文档 - Policy Enforcer 重构

## 原始需求
完善 `PolicyEnforcer` 模块，将目前的硬编码（Hardcoded）白名单逻辑替换为基于数据库的实时查询，并实现缺失的跳过策略（Skip Policy）逻辑。

## 项目上下文
### 现状
- `PolicyEnforcer` 目前是一个半成品。
- 白名单检查逻辑 (`isWhitelisted`) 仅包含硬编码的示例数据 (`127.0.0.1` 等)。
- 跳过策略逻辑 (`SkipLogicEvaluator`) 完全未实现。
- 已有 `AssetPolicyRepository` 用于 CRUD 操作，但缺乏专门供 Enforcer 使用的高效查询接口。

### 技术约束
- **实时性要求**: 白名单检查必须实时查库（Real-time DB Lookup），不使用缓存，以确保安全性优先。
- **架构一致性**: 必须通过 Repository 模式访问数据库，禁止在 Service 层直接操作 DB。
- **依赖注入**: 需要调整 `SchedulerService` 和 `PolicyEnforcer` 的初始化流程以注入新的依赖。

## 需求理解

### 功能边界
**包含功能 (In Scope):**
1.  **Repository 扩展**: 在 `AssetPolicyRepository` 中添加获取所有**已启用**白名单和跳过策略的方法。
2.  **白名单逻辑实装**:
    - 从 DB 获取白名单规则。
    - 实现 IP/CIDR 匹配。
    - 实现域名后缀匹配。
    - 实现精确字符串匹配。
3.  **跳过策略实装**:
    - 解析 `ConditionRules` JSON 配置。
    - 实现基于**时间窗口** (Time Window) 的跳过逻辑。
    - 实现基于**环境标签** (Env Tags) 的跳过逻辑。
4.  **依赖注入**: 更新 `NewSchedulerService` 和 `NewPolicyEnforcer`。

**明确不包含 (Out of Scope):**
1.  **缓存机制**: 暂时不实现 Redis 或内存缓存（根据用户决策）。
2.  **复杂的跳过规则**: 仅支持时间窗和环境标签，暂不支持复杂的自定义脚本规则。

## 疑问澄清
### 已解决的决策点
1.  **查询策略**: 确认使用**实时查库**，不使用缓存。
    - *原因*: 安全兜底功能，正确性优于性能。当前 Master 节点负载允许实时查询。

## 验收标准
### 功能验收
- [ ] **白名单阻断**: 当目标 IP/域名存在于 `asset_whitelists` 表且 `enabled=true` 时，任务必须被拦截（Status=failed, ErrorMsg="whitelisted"）。
- [ ] **正常放行**: 当目标不在白名单中时，任务正常通过。
- [ ] **时间窗跳过**: 当当前时间处于 `asset_skip_policies` 定义的禁止时间段内，任务被跳过。
- [ ] **环境标签跳过**: 当项目标签命中 `asset_skip_policies` 定义的禁止标签时，任务被跳过。
- [ ] **无硬编码**: 代码中不再包含 `127.0.0.1` 等硬编码字符串。

### 质量验收
- [ ] 单元测试覆盖核心匹配逻辑（CIDR、Domain、TimeWindow）。
- [ ] 集成测试验证 DB 数据生效。
