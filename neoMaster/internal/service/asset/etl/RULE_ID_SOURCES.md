# 各漏洞来源 rule_id 取值对照表（用于构造 IDAlias）

本文档用于统一 NeoScan 在不同漏洞来源/扫描阶段下的 `rule_id` 取值规则，确保 `IDAlias` 稳定可复现。

推荐 IDAlias canonical 格式：

```
<engine>:<ruleset>:<rule_id>
```

其中 `rule_id` 的取值必须满足：稳定、可复现、具备命名空间（避免跨扫描器撞名）。

## 1. StageResult.ResultType 维度（ETL 输入契约）

| ResultType | 典型来源 | rule_id 取值字段 | engine 建议 | ruleset 建议 | 备注 |
| --- | --- | --- | --- | --- | --- |
| `vuln_finding` | 扫描器漏洞发现（归一化后） | `VulnFindingAttributes.Findings[].ID` | 由阶段工具确定（例如 nuclei/xray/nessus/openvas/neosc） | 对应规则集（例如 nuclei-templates/xray-plugins） | `Findings[].CVE` 作为补充字段，不作为必需 identity。 |
| `poc_scan` | PoC 验证/利用结果 | `PocScanAttributes.PocResults[].PocID` | 由 PoC 执行器确定（poc_scanner 或具体引擎 nuclei/xray） | 对应 PoC 集合（nuclei-templates/custom-pocs） | PoC 验证不应产生“新漏洞 identity”，应复用同一 `IDAlias` 并更新 Verify 字段。 |

## 2. 常见扫描器/规则体系（用于填充 engine/ruleset/rule_id）

| engine | ruleset | rule_id 应取什么 | 示例 IDAlias | 备注 |
| --- | --- | --- | --- | --- |
| `nuclei` | `nuclei-templates` | 模板 `id`（首选），其次为稳定模板路径 | `nuclei:nuclei-templates:CVE-2021-44228` | nuclei 模板天然带命名空间；路径需保证稳定（模板库更新时路径可能变）。 |
| `xray` | `xray-plugins` | 插件/规则 ID（必须稳定） | `xray:xray-plugins:shiro-rememberme` | 需要避免使用泛化名称（例如 sql-injection）；应使用插件唯一 ID。 |
| `nessus` | `nessus-plugins` | Nessus Plugin ID（数字） | `nessus:nessus-plugins:11219` | Plugin ID 稳定且可复现。 |
| `openvas` | `gvmd` | OID（例如 1.3.6.1...） | `openvas:gvmd:1.3.6.1.4.1.25623.1.0.108342` | OID 是稳定标识。 |
| `neosc` | `neosc-rules` | 自研规则 `rule_id`（必须稳定且版本策略明确） | `neosc:neosc-rules:weak-password:ssh` | 建议 rule_id 具备领域维度（例如 weak-password:ssh），避免撞名。 |
| `manual` | `manual-import` | 外部系统/人工导入的漏洞键 | `manual:manual-import:cmdb:VULN-2026-0001` | 必须包含来源命名空间，避免与扫描器规则撞名。 |

## 3. 规范化要求（必须遵守）

### 3.1 rule_id 的稳定性

- rule_id 必须由“规则本身的身份”决定，而不是由“本次扫描实例”决定。
- 禁止把 `task_id`/`stage_id`/时间戳/随机数等写入 rule_id。

### 3.2 命名空间化（避免撞名）

- 规则名称过于泛化时（如 `sql-injection`/`weak-password`），必须通过 engine/ruleset/rule_id 的组合保证全局不撞名。
- 允许在 `rule_id` 内部再加入领域子维度（例如 `weak-password:ssh`）。

### 3.3 证据不进入 identity

- URL path/query/param、命中点、请求响应片段、截图/HTML hash 等属于 Evidence，应落在 `Evidence/Attributes` 聚合。
- 不把这些字段放进 rule_id/IDAlias，避免同一漏洞在同一站点多入口触发导致“多条漏洞记录”。

