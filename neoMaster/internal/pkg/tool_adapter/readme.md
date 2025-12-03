├── pkg/                      # [通用库] - 嵌入基础设施
│   ├── tool_adapter/         # [新增] 工具适配层 (Infrastructure)   # 这个包在agent端也需要有
│   │   ├── factory/          # 命令生成工厂 (Command Builder)
│   │   ├── parser/           # 结果解析器接口 (Result Parser)
│   │   └── registry/         # 工具注册与版本管理