# 需求对齐文档 - 分层标签体系 (Hierarchical Tagging System)

## 原始需求
用户希望设计一套类似 `/R/安全域/内部服务器区/IDC Level 03` 的分层标签体系，应用于全系统（资产、任务、策略等）。

**核心诉求：**
1.  **独立服务化 (Independent Service)**: 标签系统应作为一个独立的模块或服务存在，提供统一的打标、查询接口。
2.  **通用调用 (Universal Access)**: 不仅限于 Asset ETL 流水线，其他服务（如扫描器、漏洞管理、用户中心）均可调用。
3.  **可插拔的打标器 (Pluggable Taggers)**:
    *   支持自定义打标规则。
    *   支持多种打标引擎（Taggers），例如基于简单规则的匹配、基于复杂表达式的计算，甚至将来基于 AI 的分类。
4.  **CRUD & 树状结构**: 直观的层级展示、重命名与移动。

## 项目上下文
### 技术栈
- 编程语言：Go (Gin)
- 数据库：MySQL (GORM)
- 架构：NeoScan Master (Control Plane)

### 现有架构理解
- 现有系统各模块（Asset, Agent）耦合了部分标签逻辑。
- 需要将标签逻辑剥离，形成独立的 `TagService`。

## 需求理解与架构决策

### 1. 核心理念：标签即服务 (Tagging as a Service)
**Linus 的视角**: "Do one thing and do it well."
标签系统不应该关心它标记的是什么（Asset 还是 User），它只关心：
1.  **Tree**: 标签本身的结构。
2.  **Attachment**: 标签贴在谁身上 (EntityID + EntityType)。
3.  **Rules**: 怎么自动贴上去。

### 2. 核心组件：打标器 (The Taggers)
为了满足"自定义规则"和"多种打标方式"，我们引入 **Tagger** 的概念。
*   **RuleBasedTagger**: 读取数据库中的规则（IP段、正则、关键字），进行匹配。
*   **ExternalTagger**: 允许外部系统通过 API 直接推送标签。
*   **ScriptTagger (Future)**: 允许运行简单的 Lua/CEL 脚本进行动态打标。

### 3. 数据结构设计 (The Data Structure)
保持 **Adjacency List + Materialized Path** 的混合模式，确保查询性能。

## 疑问澄清

### P0级问题
1.  **打标冲突处理**：
    *   多个 Tagger 可能对同一个实体打出冲突的标签（例如一个打 "High Risk"，一个打 "Low Risk"）。
    *   **决策**: 标签系统只负责"贴"，不负责"裁决"。所有标签共存。如果需要互斥（如等级只能有一个），由上层业务逻辑或特定的 Tagger 内部处理。

## 验收标准
### 功能验收
- [ ] **独立性**: 标签模块无任何业务实体的硬编码引用（使用 Interface{} 或 Generic string ID）。
- [ ] **规则引擎**: 支持配置 JSON 格式的复杂规则。
- [ ] **调用灵活**: ETL 流程只需调用 `TagService.AutoTag(entity)` 即可，无需关心内部用了哪个 Tagger。
- [ ] **树操作**: 移动、重命名节点，路径自动更新。

### 质量验收
- [ ] **性能**: 自动打标过程不应阻塞主业务流程（建议异步或高性能同步）。
- [ ] **扩展性**: 新增一种打标规则类型不需要修改数据库结构。
