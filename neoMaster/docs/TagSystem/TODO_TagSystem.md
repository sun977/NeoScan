# 待办事项 - 标签系统 (Tag System)

## 后续迭代规划 (Future Roadmap)

### 优先级 P1 (High) - 稳定性与一致性
- [ ] **UpdateTag 路径级联更新**: 
  - **背景**: 目前修改标签的 `ParentID` 不会自动更新子节点的 `Path`，可能导致层级关系断裂。
  - **方案**: 引入 `MoveTag` 专用接口，使用事务 + 递归/批量 SQL 更新所有后代节点的 Path。
  - **现状**: 已在 `UpdateTag` 中禁用 `ParentID` 修改以规避风险。

### 优先级 P2 (Medium) - 可观测性
- [ ] **监控指标 (Metrics)**:
  - 添加 `tag_match_duration_seconds` 直方图 (Prometheus)，监控自动打标耗时分布。
  - 添加 `tag_cache_reload_total` 计数器，监控规则重载发生的频率。

### 优先级 P3 (Low) - 性能极致优化
- [ ] **AutoTag 批量接口**:
  - 扩展 `AutoTag` 支持批量实体输入，减少函数调用开销，适用于大规模资产导入场景。
- [ ] **缓存细粒度锁**:
  - 如果规则数量达到百万级，考虑将全局 `RWMutex` 拆分为基于 `EntityType` 的分片锁，减少锁竞争。
