# 待办事项 - Policy Enforcer Refactor

## 核心优化
- [ ] **添加 LRU 缓存**
  - **优先级**: Low (用户反馈：白名单仅几百条，当前性能足够)
  - **描述**: 目前每次 `Enforce` 调用都会查询数据库。建议在 `AssetPolicyRepository` 或 `PolicyEnforcer` 层添加本地 LRU 缓存（如 go-cache 或 hashicorp/golang-lru）。
  - **配置**: TTL 建议设置为 1-5 分钟，缓存大小限制为 1000 条策略。
  - **收益**: 降低数据库 QPS，提高策略检查速度。

- [ ] **优化 IP Range 匹配算法**
  - **优先级**: Low (用户反馈：数据量小，当前线性遍历足够)
  - **描述**: 目前 IP Range 匹配是线性遍历。如果白名单规则中有成千上万个 IP 段，性能会下降。
  - **方案**: 使用 Trie 树（前缀树）或专门的 IP 匹配库（如 ipcache）来索引所有 IP 规则。
  - **收益**: 将匹配复杂度从 O(N) 降低到 O(1) 或 O(logN)。

## 配置管理
- [ ] **完善 NotifyConfig 和 ExportConfig 的结构化定义**
  - **优先级**: Low
  - **描述**: 测试中发现这些字段在数据库中是 JSON 字符串，但目前代码中多作为 `string` 处理。
  - **方案**: 在 `internal/pkg/models` 中定义具体的结构体，并实现 `Value/Scan` 接口以便 GORM 自动处理 JSON 序列化。

## 监控与告警
- [ ] **添加策略命中指标监控**
  - **优先级**: Medium
  - **描述**: 记录每条白名单或跳过策略的命中次数。
  - **方案**: 集成 Prometheus Metrics，增加 `policy_hit_count{policy_id="xxx", type="whitelist"}` 指标。
  - **收益**: 帮助运维人员了解哪些策略是活跃的，哪些是僵尸策略。
