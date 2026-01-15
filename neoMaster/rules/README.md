# 规则目录

这里存放的各种规则是为了让 Agent 主动拉取更新使用
获取方式： master生成规则文件，供Agent下载
加密传输：1.身份鉴权，只有已认证的Agent可以下载 2.HTTPS安全传输
规则加密：在Master端加密规则文件，然后Agent下载后解密使用

需要在配置文件中添加加密密钥
security:
  # Agent 通信与数据安全配置
  agent:
    token_secret: "your-agent-token-secret-here"      # 身份鉴权：用于验证 Agent 身份
    rule_encryption_key: "your-encryption-key-here"   # 规则加密：用于加密规则文件 (AES等)

## 规则类型

- 指纹识别规则
  - cms CMS指纹
  - os OS指纹
  - service 服务指纹
- POC文件（yaml格式）


## 文件目录说明【作废】
rules/
├── fingerprint/
│   ├── system/
│   │   ├── cms/
│   │   │   ├── default_cms.json
│   │   │   └── ...
│   │   ├── os/
│   │   │   ├── default_os.json
│   │   │   └── ...
│   │   └── service/
│   │       ├── default_service.json
│   │       └── ...
│   └── custom/
│       ├── cms/
│       │   ├── user_defined_cms.json
│       │   └── ...
│       ├── os/
│       │   ├── user_defined_os.json
│       │   └── ...
│       └── service/
│           ├── user_defined_service.json
│           └── ...
├── poc/
│   ├── system/
│   │   ├── default_poc_1.yaml
│   │   ├── default_poc_2.yaml
│   │   └── ...
│   └── custom/
│       ├── user_poc_1.yaml
│       ├── user_poc_2.yaml
│       └── ...
└── README.md
