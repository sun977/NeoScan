# 待办事项 - PreviousStageProvider 增强后续

## 待办任务
1.  **前端适配**: 更新前端工作流配置页面，支持 `PreviousStageConfig` 的 `stage_status` 字段配置。
2.  **规则编辑器**: 为 `UnwindConfig.Filter` 开发可视化规则编辑器，生成符合 `matcher.MatchRule` 结构的 JSON。
3.  **性能监控**: 在生产环境监控 `Provide` 方法的执行时间，特别是当 `Unwind` 数据量较大时的性能表现。

## 配置指引
### 示例配置
```json
{
  "filter_rules": {
    "stage_name": "port_scan",
    "stage_status": ["completed"]
  },
  "parser_config": {
    "unwind": {
      "path": "attributes",
      "filter": {
        "and": [
          {"field": "state", "operator": "equals", "value": "open"},
          {"field": "service", "operator": "contains", "value": "http"}
        ]
      }
    },
    "generate": {
      "value_template": "{{target_value}}:{{item.port}}"
    }
  }
}
```
