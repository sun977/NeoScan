# Agent 标签与能力体系重构设计方案

> **Author**: Linus (AI Assistant)  
> **Date**: 2025-12-16  
> **Status**: Draft / Proposal  
> **Philosophy**: "Bad programmers worry about the code. Good programmers worry about data structures and their relationships."

## 1. 现状与问题 (The Bad Taste)

当前 `Agent` 模块的设计在数据结构上存在严重的“品味问题”，主要体现在 `tags` 和 `capabilities` 两个字段的处理上。

### 1.1 当前设计
- **Tags**: 在 `agent` 表中使用 `json` 类型字段存储标签 ID 列表 (e.g., `["1", "5"]`)。
- **Capabilities**: 在 `agent` 表中使用 `json` 类型字段存储扫描类型 ID 列表 (e.g., `["2", "3"]`)，对应 `agent_scan_types` 表。

### 1.2 核心问题 (Critical Issues)

1.  **丧失数据完整性 (No Referential Integrity)**
    - JSON 数组中的 ID 只是纯文本，数据库无法通过外键约束（FK）来保证这些 ID 的有效性。
    - 如果在 `SysTag` 或 `ScanType` 表中删除了某条记录，`agent` 表中的数据就会变成“悬空指针”，导致**数据腐烂 (Data Rot)**。

2.  **反向查询噩梦 (The Reverse Lookup Nightmare)**
    - 无法高效回答“找出所有支持 PortScan 的在线 Agent”这类基础调度问题。
    - 必须使用低效的 `JSON_CONTAINS` 或字符串匹配，导致数据库索引失效，全表扫描不可避免。

3.  **系统不一致性 (Inconsistency)**
    - **资产 (Asset)** 模块正确使用了 `SysEntityTag` (Many-to-Many 关联表) 来管理标签。
    - **节点 (Agent)** 模块却使用 JSON 字段。
    - 这种“精神分裂”的设计增加了系统的认知负担和维护成本。

4.  **概念混淆**
    - `Capabilities` 本质上是 Agent 的一种属性（Attribute）或标签（Tag），却被特殊对待，开辟了独立的字段，导致无法复用通用的标签筛选、搜索和规则引擎能力。

---

## 2. 重构目标 (The Good Taste)

1.  **统一模型**：将 Agent 视为系统中的一种标准 **实体 (Entity)**，完全复用新设计的通用标签系统。
2.  **单一真理源 (Single Source of Truth)**：`agent_scan_types` 表定义能力，`sys_tags` 表作为其投影，确保定义与筛选分离但同步。
3.  **强一致性**：利用关系型数据库的关联表特性，保证数据的完整性和查询的高效性。
4.  **通用接口**：前端和调度器使用统一的标签查询接口（Tag Search API）来筛选 Agent，无需为 Capabilities 编写特殊逻辑。

---

## 3. 详细重构方案

### 3.1 数据库层变更 (Schema Changes)

#### A. 清理 Agent 表
- **删除** `agent` 表中的 `tags` 字段。
- **删除** `agent` 表中的 `capabilities` 字段。

#### B. 扩展 Agent Scan Types 表
在 `agent_scan_types` 表中建立与标签系统的**硬链接**。

```go
type ScanType struct {
    ID    uint64
    Name  string // e.g., "PortScan"
    // ... 其他执行参数 ...
    
    // 新增：关联的系统标签ID
    // 这建立了 "能力定义" -> "标签筛选" 的 O(1) 映射
    TagID uint64 `gorm:"uniqueIndex"` 
}
```

#### C. 复用 SysEntityTag 表
直接使用现有的 `sys_entity_tags` 表存储 Agent 的标签和能力。
- `EntityType`: 固定为 `"agent"`
- `EntityID`: Agent 的 UUID
- `TagID`: 对应普通标签或能力标签的 ID
- `Source`: 
    - `'manual'` (手动打标)
    - `'agent_report'` (Agent 心跳上报的能力)

### 3.2 标签树结构设计 (The Skeleton)

我们需要在系统初始化时预设标签树的**骨架**，但**叶子节点**（具体能力）应由代码动态维护。

**预设骨架 (SQL/Init)**:
```text
ROOT (ID: 1)
└── System (Category: 'system')
    └── Capability (Category: 'system', Path: '/1/2/')
```

### 3.3 镜像同步机制 (The Mirroring Mechanism)

为了解决“能力定义在 ScanType 表，筛选在 Tag 表”的问题，建立自动同步机制。

**启动引导逻辑 (Bootstrap Logic)**:
当 Master 启动时：
1.  读取 `agent_scan_types` 表的所有记录。
2.  检查 `/System/Capability/` 路径下是否存在同名标签。
3.  **不存在则自动创建**，并将生成的 `TagID` 回写到 `agent_scan_types.tag_id`。
4.  **锁定标签**：设置生成的标签 `IsSystem = true` (防止用户手动删除)。

### 3.4 运行时数据流 (Runtime Flow)

**场景：Agent 心跳上报能力**

1.  **Agent 上报**: 
    `Heartbeat { AgentID: "xyz", Capabilities: ["PortScan", "WebScan"] }` (这里上报的是能力名称或 ScanType ID)
    
2.  **Master 处理**:
    - 查询 `agent_scan_types` 表，根据上报的能力找到对应的记录。
    - 提取这些记录中的 `TagID` 字段。
    - 得到 `TargetTagIDs = [101, 102]`。

3.  **调用标签服务**:
    ```go
    // 使用全量调和逻辑 (Full Reconciliation)
    // 确保 Agent 失去某种能力时，对应的标签也会被移除
    tagService.SyncEntityTags(ctx, "agent", agentID, TargetTagIDs, "agent_report")
    ```

---

## 4. 迁移策略 (Migration Strategy)

为了在不破坏现有数据的前提下完成重构，建议按以下步骤操作：

1.  **Pre-Flight**: 备份数据库。
2.  **Schema Upgrade**: 
    - 向 `agent_scan_types` 添加 `tag_id` 列。
    - 确保 `sys_tags` 中存在 `/System/Capability/` 骨架。
3.  **Data Migration (Script)**:
    - 遍历 `agent_scan_types`，为每个类型创建对应的 `SysTag` 并回填 `tag_id`。
    - 遍历 `agent` 表：
        - 解析 `tags` JSON，插入 `sys_entity_tags` (Source='manual')。
        - 解析 `capabilities` JSON，映射到对应的 `TagID`，插入 `sys_entity_tags` (Source='agent_report')。
4.  **Code Switch**: 部署新代码（移除旧字段的读写，启用新的心跳处理逻辑）。
5.  **Cleanup**: 删除 `agent` 表中的旧字段。

---

## 5. 最终效果 (The Outcome)

通过此次重构，我们达成：
- **统一的搜索接口**：`SELECT * FROM agents WHERE tag_id IN (...)` 即可筛选任意属性或能力的 Agent。
- **干净的数据库**：没有非结构化的 JSON 字段，所有关系均由 FK 约束。
- **自动化维护**：新增扫描能力只需在 `agent_scan_types` 插入一条记录，标签系统自动感知，无需人工干预。
