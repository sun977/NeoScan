
# ETL 漏洞幂等性：演进对话与决策记录

## 1. 我们在解决什么问题

### 1.1 现象

- 同一资产目标（target）在并发处理/重试/重复上报时，可能出现“同一个漏洞被写入多条记录”的情况。

### 1.2 影响

- 资产闭环被污染：漏洞数量虚高、状态分裂、证据分散。
- 排查困难：问题表现为“看起来偶发”，但本质上是并发竞态的必然结果。

### 1.3 误区澄清：ID 主键唯一 ≠ 业务幂等

- `asset_vulns.id` 的唯一性只保证“每一行都有不同的物理编号”。
- 幂等性要保证的是“同一个 target 上的同一个漏洞（同一业务身份）只能有一条记录”。

并发下经典竞态：

- Worker A：查不到（认为不存在）→ Insert
- Worker B：也查不到（认为不存在）→ Insert

两个 Insert 都能成功，因为它们写入的是两行不同的 `id`。这就是“代码层查重无法在并发下形成保证”的根源。

## 2. 我们做了哪些思考

### 2.1 核心原则（数据结构优先）

- 幂等性不是“写对了 if/else”，而是“业务身份 + 数据库约束”让重复写入在物理层面不可能。

### 2.2 一对多关系不与唯一约束冲突

- 设计上同一个 target 允许存在多个漏洞（1:N）。
- 唯一约束约束的是：同一个 target 下，“同一个漏洞身份”不能重复。
- 换句话说：允许 N 个不同漏洞（不同 identity），禁止同一漏洞 identity 落多行。

### 2.3 为什么不再使用 UUID

- 随机 UUID（例如 UUIDv4）会导致每次发现都是新 identity，天然破坏幂等。
- 只有“确定性 identity”（由稳定维度规范化得到）才有意义。
- 本次决策：不强制使用 UUID，改用可读、可规范化、可索引的 `IDAlias` 结构。

## 3. 阶段性想法与演进

### 3.1 阶段性约束（当前可落地的方向）

- `CVE` 作为可选字段：有则填，无则为空。
- `IDAlias` 作为必须字段：用于唯一身份标识（业务 identity）。

结论：这个方向正确，因为它解决了“无 CVE 漏洞无法稳定标识”的现实问题。

### 3.2 但仅靠“IDAlias 必填”仍不足以彻底幂等

即便 `IDAlias` 完美，只靠“先查再写”的应用逻辑仍会在并发下写出重复记录。

彻底幂等需要两个条件同时满足：

1. `IDAlias` 的语义必须稳定且具备命名空间（避免撞名/漂移）。
2. 数据库必须对“业务唯一性”提供兜底约束（防并发竞态）。

## 4. 最终决定（本次讨论的结论）

### 4.1 统一使用 IDAlias 作为漏洞业务身份

- `IDAlias` 必须存在且稳定。
- `CVE` 保留为可选增强字段，不参与“必需 identity”。

### 4.2 IDAlias 推荐结构（可读、可规范化）

推荐 canonical 格式：

```
<engine>:<ruleset>:<rule_id>
```

说明：

- `engine`：漏洞来源引擎/扫描器（nuclei/xray/nessus/openvas/neosc 等）
- `ruleset`：规则集或家族（nuclei-templates/xray-plugins/neosc-rules 等）
- `rule_id`：规则的稳定 ID（模板 id、插件 id、OID、自研 rule_id 等）

举例：

- `nuclei:nuclei-templates:CVE-2021-44228`
- `xray:xray-plugins:shiro-rememberme`
- `openvas:gvmd:1.3.6.1.4.1.25623.1.0.108342`
- `neosc:neosc-rules:weak-password:ssh`

### 4.3 幂等性彻底落地策略（工程落地目标）

- 最终幂等性目标：同一 target 下，同一 `IDAlias` 只能存在一条漏洞记录。
- 存储层兜底：使用数据库唯一约束表达业务唯一性，而不是依赖应用层查重。

约束语义（推荐）：

- `(target_type, target_ref_id, id_alias)` 作为业务唯一键
- `cve` 仅作为补充字段，用于检索/统计/展示

### 4.4 对 Evidence/Attributes 的定位（避免 identity 膨胀）

- 触发路径、参数点、请求/响应片段、扫描时间、task_id/stage_id 等属于“证据”，应进入 `Evidence/Attributes` 聚合存储。
- 不把 URL path/query/param 等放进 `IDAlias`，否则同一漏洞在同一站点不同入口会变成多条漏洞记录（数量爆炸且不可运营）。

## 5. 待办（下一步演进）

- 明确每种来源（ResultType/Scanner）的 `rule_id` 取值规则，保证 `IDAlias` 真正稳定。
- 在数据库层补齐业务唯一约束前，需要先审计并清理历史重复数据（按 target + id_alias 归并证据）。
