# NeoScan Agent节点功能设计说明 v1.0

## 1. 概述

### 1.1 Agent节点定位
NeoScan Agent节点是分布式扫描系统的执行单元，负责接收Master节点下发的扫描任务，执行具体的安全扫描工作，并将扫描结果回传给Master节点。Agent节点采用轻量化设计，支持模块化架构和插件扩展，具备高度的灵活性和可扩展性。

### 1.2 核心特性
- **轻量化架构**：最小化资源占用，适合大规模部署
- **模块化设计**：功能模块独立，支持热插拔
- **插件扩展**：标准化插件接口，支持自定义功能扩展
- **智能调度**：支持任务优先级和资源管理
- **安全隔离**：插件沙箱执行，保障系统安全
- **实时通信**：与Master节点保持实时连接和状态同步

### 1.3 技术架构
- **开发语言**：Go语言
- **Web框架**：Gin框架
- **通信协议**：gRPC + RESTful API
- **本地存储**：SQLite（轻量级数据库）
- **缓存系统**：本地内存缓存
- **消息队列**：RabbitMQ客户端
- **容器化**：Docker容器部署

## 2. 系统架构设计

### 2.1 目录结构
```
neoAgent/
├── cmd/                          # 应用程序入口
│   └── agent/
│       └── main.go              # Agent主程序入口
├── internal/                     # 内部代码，不对外暴露
│   ├── app/                     # 应用程序核心
│   │   ├── agent.go            # Agent主服务
│   │   └── config.go           # 配置管理
│   ├── handler/                 # HTTP处理器
│   │   ├── health.go           # 健康检查接口
│   │   ├── config.go           # 配置管理接口
│   │   ├── task.go             # 任务管理接口
│   │   ├── plugin.go           # 插件管理接口
│   │   └── metrics.go          # 系统指标接口
│   ├── service/                 # 业务逻辑层
│   │   ├── task.go             # 任务管理服务
│   │   ├── config.go           # 配置管理服务
│   │   ├── plugin.go           # 插件管理服务
│   │   └── metrics.go          # 系统监控服务
│   ├── repository/              # 数据访问层
│   │   ├── sqlite/             # SQLite数据库
│   │   ├── cache/              # 本地缓存
│   │   └── file/               # 文件存储
│   ├── model/                   # 数据模型
│   │   ├── task.go             # 任务模型
│   │   ├── config.go           # 配置模型
│   │   ├── result.go           # 扫描结果模型
│   │   └── plugin.go           # 插件模型
│   ├── grpc/                    # gRPC通信
│   │   ├── client/             # gRPC客户端
│   │   ├── server/             # gRPC服务端
│   │   └── interceptor/        # 拦截器
│   └── modules/                 # 功能模块
│       ├── scan/               # 扫描模块
│       ├── virusKill/          # 病毒查杀模块
│       ├── core/               # 核心模块
│       ├── logs/               # 日志模块
│       ├── plugins/            # 插件模块
│       └── runner/             # 运行调度模块
├── pkg/                         # 公共库
│   ├── logger/                 # 日志库
│   ├── utils/                  # 工具库
│   └── security/               # 安全库
├── configs/                     # 配置文件
│   ├── agent.yaml             # Agent配置
│   ├── database.yaml          # 数据库配置
│   ├── grpc.yaml              # gRPC配置
│   ├── security.yaml          # 安全配置
│   ├── logging.yaml           # 日志配置
│   ├── plugins.yaml           # 插件配置
│   └── scan/                  # 扫描模块配置
│       ├── asset_scan.yaml    # 资产扫描配置
│       ├── web_scan.yaml      # Web扫描配置
│       ├── poc_scan.yaml      # POC扫描配置
│       ├── dir_scan.yaml      # 目录扫描配置
│       ├── domain_scan.yaml   # 域名扫描配置
│       ├── weakpwd_scan.yaml  # 弱口令扫描配置
│       └── proxy_scan.yaml    # 代理扫描配置
├── scripts/                     # 脚本文件
│   ├── build.sh               # 构建脚本
│   ├── deploy.sh              # 部署脚本
│   ├── start.sh               # 启动脚本
│   └── stop.sh                # 停止脚本
├── docs/                        # 文档
├── tests/                       # 测试文件
├── logs/                        # 日志文件
├── storage/                     # 存储目录
├── Dockerfile                   # Docker构建文件
└── docker-compose.yml          # Docker编排文件
```

### 2.2 分层架构

#### 2.2.1 Handler层（HTTP处理层）
- **职责**：处理HTTP请求，参数验证，响应格式化
- **组件**：健康检查、配置管理、任务管理、插件管理、系统指标

#### 2.2.2 Service层（业务逻辑层）
- **职责**：业务逻辑处理，模块协调，状态管理
- **组件**：任务服务、配置服务、插件服务、监控服务

#### 2.2.3 Repository层（数据访问层）
- **职责**：数据持久化，缓存管理，文件操作
- **组件**：SQLite数据库、本地缓存、文件存储

#### 2.2.4 Model层（数据模型层）
- **职责**：数据结构定义，业务实体建模
- **组件**：任务模型、配置模型、结果模型、插件模型

## 3. 核心功能模块设计

### 3.1 scan扫描模块

#### 3.1.1 资产扫描功能（asset_scan）
**功能描述**：对目标网络进行资产发现和识别

**核心能力**：
- **IP存活探测**：使用ICMP、TCP、UDP等协议探测IP存活状态
- **全端口扫描**：支持1-65535端口的全面扫描
- **服务识别**：识别端口对应的服务类型和版本信息
- **操作系统识别**：通过指纹识别技术识别目标系统类型
- **网络拓扑发现**：分析网络结构和设备连接关系

**集成工具**：
- **Nmap**：网络发现和安全审计的标准工具
- **Masscan**：高速端口扫描工具，适合大规模网络扫描
- **Fscan**：内网综合扫描工具，支持多种协议探测

**配置参数**：
```yaml
asset_scan:
  enabled: true
  scan_threads: 100          # 扫描线程数
  timeout: 30                # 超时时间（秒）
  port_range: "1-65535"      # 端口范围
  scan_rate: 1000            # 扫描速率（包/秒）
  ping_scan: true            # 是否进行ping扫描
  service_detection: true    # 是否进行服务识别
  os_detection: true         # 是否进行操作系统识别
  tools:
    nmap:
      enabled: true
      path: "/usr/bin/nmap"
      options: "-sS -sV -O"
    masscan:
      enabled: true
      path: "/usr/bin/masscan"
      options: "--rate=1000"
```

#### 3.1.2 Web扫描功能（web_scan）
**功能描述**：对Web应用进行深度扫描和分析

**核心能力**：
- **网站信息识别**：识别Web服务器、框架、CMS等技术栈
- **网页爬取分析**：深度爬取网站内容，发现隐藏页面和接口
- **指纹识别**：基于HTTP响应头、页面特征进行技术栈识别
- **动态内容分析**：使用ChromeDriver分析JavaScript渲染的动态内容
- **敏感信息发现**：识别页面中的敏感信息泄露

**爬取策略**：
- **传统爬虫**：
  - 页面标题（Title）提取
  - 服务横幅（Banner）获取
  - Meta标签和关键字分析
  - 技术栈特征识别
- **ChromeDriver模拟访问**：
  - 动态内容渲染
  - 页面截图生成
  - DOM结构分析
  - 网络请求监控

**配置参数**：
```yaml
web_scan:
  enabled: true
  max_depth: 3               # 爬取深度
  max_pages: 1000            # 最大页面数
  request_delay: 1           # 请求间隔（秒）
  timeout: 30                # 请求超时（秒）
  user_agents:               # User-Agent列表
    - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  chrome_driver:
    enabled: true
    path: "/usr/bin/chromedriver"
    headless: true
    screenshot: true
```

#### 3.1.3 POC漏洞扫描功能（poc_scan）
**功能描述**：基于POC（Proof of Concept）进行漏洞验证

**核心能力**：
- **POC验证**：自动化执行漏洞概念验证脚本
- **漏洞检测**：基于规则库进行已知漏洞检测
- **自定义POC**：支持用户自定义POC脚本
- **漏洞评级**：根据CVSS标准进行漏洞风险评级

**集成工具**：
- **Nuclei**：基于YAML模板的漏洞扫描器
- **Xray**：被动式安全评估工具

**配置参数**：
```yaml
poc_scan:
  enabled: true
  concurrent_tasks: 10       # 并发任务数
  timeout: 60                # POC执行超时（秒）
  severity_filter:           # 严重级别过滤
    - "critical"
    - "high"
    - "medium"
  tools:
    nuclei:
      enabled: true
      path: "/usr/bin/nuclei"
      templates_path: "/opt/nuclei-templates"
    xray:
      enabled: true
      path: "/usr/bin/xray"
```

#### 3.1.4 目录扫描功能（dir_scan）
**功能描述**：发现Web应用的隐藏目录和敏感文件

**核心能力**：
- **目录爆破**：使用字典进行目录和文件名爆破
- **敏感文件检测**：识别备份文件、配置文件等敏感信息
- **状态码分析**：分析HTTP响应状态码，识别有效路径
- **内容长度分析**：通过响应内容长度判断页面有效性

**配置参数**：
```yaml
dir_scan:
  enabled: true
  threads: 50                # 扫描线程数
  timeout: 10                # 请求超时（秒）
  dictionary_path: "/opt/dict/dir.txt"  # 字典文件路径
  extensions:                # 文件扩展名
    - ".php"
    - ".asp"
    - ".jsp"
    - ".txt"
  status_codes:              # 有效状态码
    - 200
    - 301
    - 302
    - 403
```

#### 3.1.5 域名扫描功能（domain_scan）
**功能描述**：进行域名相关的信息收集和分析

**核心能力**：
- **子域名发现**：通过多种方式枚举子域名
- **域名解析**：获取域名对应的IP地址
- **DNS记录查询**：查询A、AAAA、CNAME、MX等DNS记录
- **域名劫持检测**：检测域名是否存在劫持风险

**发现方式**：
- 字典爆破
- DNS区域传输
- 证书透明度日志
- 搜索引擎查询

**配置参数**：
```yaml
domain_scan:
  enabled: true
  threads: 20                # 扫描线程数
  timeout: 5                 # DNS查询超时（秒）
  subdomain_dict: "/opt/dict/subdomain.txt"  # 子域名字典
  dns_servers:               # DNS服务器列表
    - "8.8.8.8"
    - "114.114.114.114"
  methods:                   # 发现方法
    - "brute_force"
    - "certificate_transparency"
    - "search_engine"
```

#### 3.1.6 弱口令扫描功能（weakpwd_scan）
**功能描述**：对常见服务进行弱口令检测

**核心能力**：
- **服务爆破**：支持SSH、FTP、MySQL、Redis等服务的密码爆破
- **字典管理**：支持自定义用户名和密码字典
- **智能爆破**：根据服务特点调整爆破策略
- **防护绕过**：支持验证码识别、频率限制绕过等技术

**支持服务**：
- SSH (22)
- FTP (21)
- Telnet (23)
- MySQL (3306)
- Redis (6379)
- MongoDB (27017)
- RDP (3389)
- SMB (445)

**配置参数**：
```yaml
weakpwd_scan:
  enabled: true
  threads: 10                # 爆破线程数
  timeout: 10                # 连接超时（秒）
  delay: 1                   # 爆破间隔（秒）
  username_dict: "/opt/dict/username.txt"
  password_dict: "/opt/dict/password.txt"
  services:
    ssh:
      enabled: true
      port: 22
    mysql:
      enabled: true
      port: 3306
    redis:
      enabled: true
      port: 6379
```

#### 3.1.7 代理扫描功能（proxy_scan）
**功能描述**：发现和验证网络代理服务

**核心能力**：
- **代理发现**：扫描常见代理端口，发现代理服务
- **代理验证**：验证代理服务的可用性和匿名性
- **代理类型识别**：识别HTTP、HTTPS、SOCKS4、SOCKS5等代理类型
- **匿名性测试**：测试代理的匿名程度

**配置参数**：
```yaml
proxy_scan:
  enabled: true
  threads: 50                # 扫描线程数
  timeout: 10                # 连接超时（秒）
  proxy_ports:               # 常见代理端口
    - 8080
    - 3128
    - 1080
    - 8888
  test_url: "http://httpbin.org/ip"  # 代理测试URL
  proxy_types:               # 支持的代理类型
    - "http"
    - "https"
    - "socks4"
    - "socks5"
```

### 3.2 virusKill病毒查杀模块

**功能描述**：基于YARA规则进行恶意软件检测和查杀

**核心能力**：
- **病毒检测**：使用YARA规则引擎进行恶意软件检测
- **规则管理**：支持自定义检测规则的加载和更新
- **文件扫描**：对指定目录或文件进行病毒扫描
- **实时监控**：监控文件系统变化，实时检测新增文件
- **隔离处理**：发现恶意文件后进行隔离或删除处理

**检测类型**：
- 木马病毒
- 蠕虫病毒
- 后门程序
- 恶意脚本
- 可疑文件

**配置参数**：
```yaml
virus_kill:
  enabled: false              # 默认禁用，需要时启用
  scan_threads: 4             # 扫描线程数
  yara_rules_path: "/opt/yara/rules"  # YARA规则目录
  scan_paths:                 # 扫描路径
    - "/tmp"
    - "/var/www"
    - "/home"
  exclude_paths:              # 排除路径
    - "/proc"
    - "/sys"
  file_size_limit: 100        # 文件大小限制（MB）
  quarantine_path: "/opt/quarantine"  # 隔离目录
  actions:
    on_detection: "quarantine"  # 检测到病毒时的动作：quarantine/delete/log
    notify_master: true        # 是否通知Master节点
```

### 3.3 core核心模块

**功能描述**：提供Agent节点的基础运行框架和核心服务

**核心组件**：

#### 3.3.1 基础服务
- **服务启动**：Agent节点的启动和初始化流程
- **配置加载**：从配置文件加载系统配置
- **依赖注入**：管理各模块间的依赖关系
- **生命周期管理**：控制各组件的启动、运行、停止

#### 3.3.2 API接口服务
- **RESTful API**：提供HTTP接口服务
- **健康检查**：系统健康状态检查接口
- **配置管理**：配置查询和更新接口
- **任务管理**：任务状态查询和控制接口
- **系统指标**：性能指标和监控数据接口

#### 3.3.3 数据库管理
- **SQLite管理**：本地轻量级数据库管理
- **数据模型**：定义任务、配置、结果等数据模型
- **数据迁移**：数据库结构升级和数据迁移
- **数据备份**：本地数据备份和恢复

#### 3.3.4 配置管理
- **配置加载**：从YAML文件加载配置
- **配置验证**：配置参数合法性验证
- **配置热更新**：支持配置的动态更新
- **配置同步**：与Master节点的配置同步

### 3.4 grpc通信模块

**功能描述**：负责与Master节点的gRPC通信

**核心功能**：

#### 3.4.1 Master通信
- **节点注册**：向Master节点注册Agent身份
- **心跳保持**：定期发送心跳包维持连接
- **任务接收**：接收Master下发的扫描任务
- **状态上报**：向Master上报任务执行状态
- **结果回传**：将扫描结果发送给Master

#### 3.4.2 认证服务
- **身份认证**：基于证书或Token的身份认证
- **连接加密**：TLS加密保护通信安全
- **权限验证**：验证操作权限

#### 3.4.3 数据传输
- **流式传输**：支持大文件的流式传输
- **压缩传输**：数据压缩减少网络带宽
- **断点续传**：支持传输中断后的续传
- **传输监控**：监控传输状态和进度

**gRPC服务定义**：
```protobuf
service AgentService {
  // 节点注册
  rpc RegisterAgent(RegisterRequest) returns (RegisterResponse);
  
  // 心跳保持
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  
  // 接收任务
  rpc ReceiveTask(TaskRequest) returns (TaskResponse);
  
  // 上报状态
  rpc ReportStatus(StatusRequest) returns (StatusResponse);
  
  // 上传结果
  rpc UploadResult(stream ResultRequest) returns (ResultResponse);
  
  // 配置更新
  rpc UpdateConfig(ConfigRequest) returns (ConfigResponse);
  
  // 插件控制
  rpc ControlPlugin(PluginRequest) returns (PluginResponse);
}
```

### 3.5 logs日志模块

**功能描述**：负责日志收集、管理和上报

**核心功能**：

#### 3.5.1 日志收集
- **模块日志**：收集各功能模块的运行日志
- **系统日志**：收集系统级别的事件日志
- **错误日志**：收集异常和错误信息
- **审计日志**：记录关键操作的审计信息

#### 3.5.2 结果上报
- **RabbitMQ客户端**：通过消息队列向Master上报扫描结果
- **批量上报**：支持批量发送提高效率
- **重试机制**：失败重试和死信队列处理
- **消息确认**：确保消息可靠传输

#### 3.5.3 日志管理
- **日志轮转**：按大小或时间进行日志文件轮转
- **日志压缩**：历史日志文件压缩存储
- **日志清理**：定期清理过期日志文件
- **日志查询**：提供日志查询和过滤功能

**配置参数**：
```yaml
logging:
  level: "info"               # 日志级别：debug/info/warn/error
  format: "json"              # 日志格式：json/text
  output: "file"              # 输出方式：file/console/both
  file_path: "./logs/agent.log"  # 日志文件路径
  max_size: 100               # 单个日志文件最大大小（MB）
  max_backups: 10             # 保留的日志文件数量
  max_age: 30                 # 日志文件保留天数
  compress: true              # 是否压缩历史日志
  rabbitmq:
    enabled: true
    host: "localhost"
    port: 5672
    username: "guest"
    password: "guest"
    exchange: "neoscan.results"
    routing_key: "agent.results"
```

### 3.6 plugins扩展插件模块

**功能描述**：管理和执行各种功能扩展插件

**核心功能**：

#### 3.6.1 插件管理
- **插件加载**：动态加载和卸载插件
- **插件注册**：插件注册和发现机制
- **生命周期管理**：控制插件的启动、运行、停止
- **依赖管理**：处理插件间的依赖关系

#### 3.6.2 热加载支持
- **动态加载**：运行时加载新插件
- **热更新**：在线更新插件版本
- **配置重载**：插件配置的动态更新
- **状态迁移**：插件更新时的状态保持

#### 3.6.3 插件接口
- **标准接口**：定义统一的插件开发接口
- **事件系统**：基于事件的插件通信机制
- **数据共享**：插件间的数据共享机制
- **API调用**：插件调用系统API的接口

#### 3.6.4 内置插件

##### 3.6.4.1 shell-plugin（系统命令插件）
**功能描述**：执行Linux系统命令并返回结果

**核心能力**：
- **命令执行**：支持常用Linux命令执行
- **结果返回**：实时返回命令执行结果
- **超时控制**：命令执行超时控制
- **权限限制**：严格的命令执行权限控制

**支持命令**：
- 系统信息：`uname`, `whoami`, `id`, `uptime`
- 进程管理：`ps`, `top`, `kill`, `pgrep`
- 网络状态：`netstat`, `ss`, `lsof`, `iptables`
- 文件系统：`ls`, `df`, `du`, `find`
- 系统监控：`free`, `vmstat`, `iostat`

**安全机制**：
- **命令白名单**：只允许执行预定义的安全命令
- **参数过滤**：过滤危险参数和特殊字符
- **权限检查**：检查命令执行权限
- **审计日志**：记录所有命令执行日志

**配置参数**：
```yaml
shell_plugin:
  enabled: true
  timeout: 30                 # 命令执行超时（秒）
  user_privilege: "limited"   # 执行权限：limited/normal/elevated
  allowed_commands:           # 允许执行的命令白名单
    - "ls"
    - "ps"
    - "netstat"
    - "df"
    - "free"
    - "uname"
    - "whoami"
  forbidden_patterns:         # 禁止的命令模式
    - "rm -rf"
    - "dd if="
    - "mkfs"
    - "format"
  audit_log: true             # 是否记录审计日志
```

##### 3.6.4.2 file-plugin（文件操作插件）
**功能描述**：提供文件和目录的操作功能

**核心能力**：
- **文件上传/下载**：支持文件的上传和下载
- **文件管理**：文件和目录的创建、删除、移动
- **文件查看**：文件内容的查看和编辑
- **权限管理**：文件权限的查看和修改

**操作类型**：
- **文件操作**：创建、删除、复制、移动、重命名
- **目录操作**：创建、删除、遍历、权限设置
- **内容操作**：读取、写入、追加、搜索
- **属性操作**：权限、所有者、时间戳修改

**安全机制**：
- **路径限制**：限制可访问的文件路径
- **大小限制**：限制上传文件的大小
- **类型检查**：检查文件类型和扩展名
- **病毒扫描**：上传文件的病毒扫描

**配置参数**：
```yaml
file_plugin:
  enabled: false             # 默认禁用，需要时启用
  max_file_size: 100          # 最大文件大小（MB）
  allowed_paths:              # 允许访问的路径
    - "/tmp"
    - "/var/log"
    - "/opt/neoscan"
  forbidden_paths:            # 禁止访问的路径
    - "/etc"
    - "/root"
    - "/home"
  allowed_extensions:         # 允许的文件扩展名
    - ".txt"
    - ".log"
    - ".conf"
  virus_scan: true            # 是否进行病毒扫描
  audit_log: true             # 是否记录审计日志
```

##### 3.6.4.3 monitor-plugin（系统监控插件）
**功能描述**：收集Agent主机的系统资源信息

**核心能力**：
- **CPU监控**：CPU使用率、负载、核心数等信息
- **内存监控**：内存使用率、可用内存、交换分区等
- **磁盘监控**：磁盘使用率、I/O统计、挂载点信息
- **网络监控**：网络连接状态、流量统计、接口信息
- **进程监控**：进程列表、资源占用、状态信息

**监控指标**：
- **系统指标**：
  - CPU使用率（用户态、内核态、空闲）
  - 内存使用率（已用、可用、缓存、交换）
  - 磁盘使用率（已用空间、可用空间、I/O统计）
  - 网络流量（接收、发送、错误、丢包）
- **进程指标**：
  - 进程数量、状态分布
  - Top进程的CPU和内存占用
  - 僵尸进程检测
- **服务指标**：
  - 关键服务状态
  - 端口监听状态
  - 服务响应时间

**配置参数**：
```yaml
monitor_plugin:
  enabled: true
  interval: 60                # 监控间隔（秒）
  metrics:                    # 监控指标
    - "cpu"
    - "memory"
    - "disk"
    - "network"
    - "process"
  cpu:
    collect_per_cpu: false    # 是否收集每个CPU核心的数据
  memory:
    include_swap: true        # 是否包含交换分区信息
  disk:
    include_all_mounts: false # 是否包含所有挂载点
    exclude_types:            # 排除的文件系统类型
      - "tmpfs"
      - "devtmpfs"
  network:
    include_loopback: false   # 是否包含回环接口
  process:
    top_count: 10             # Top进程数量
    include_threads: false    # 是否包含线程信息
```

#### 3.6.5 插件安全机制

##### 3.6.5.1 权限控制
- **执行权限**：基于角色的插件执行权限控制
- **资源限制**：限制插件使用的系统资源
- **API权限**：控制插件可调用的系统API
- **网络权限**：限制插件的网络访问权限

##### 3.6.5.2 沙箱执行
- **进程隔离**：插件在独立进程中执行
- **文件系统隔离**：限制插件的文件系统访问
- **网络隔离**：控制插件的网络访问
- **资源限制**：限制CPU、内存、磁盘使用

##### 3.6.5.3 操作审计
- **执行日志**：记录插件的所有执行操作
- **资源使用**：监控插件的资源使用情况
- **异常检测**：检测插件的异常行为
- **安全告警**：发现安全威胁时及时告警

### 3.7 runner运行调度模块

**功能描述**：统一调度和管理各功能模块的执行

**核心功能**：

#### 3.7.1 任务调度
- **任务队列**：管理待执行的任务队列
- **优先级调度**：基于优先级的任务调度算法
- **资源调度**：根据系统资源情况调度任务
- **并发控制**：控制同时执行的任务数量

#### 3.7.2 计划任务
- **Crontab支持**：支持类似crontab的定时任务
- **周期任务**：支持周期性执行的任务
- **延迟任务**：支持延迟执行的任务
- **条件触发**：基于条件的任务触发机制

#### 3.7.3 执行管理
- **执行顺序**：管理模块间的执行顺序
- **依赖关系**：处理模块间的依赖关系
- **状态跟踪**：跟踪任务的执行状态
- **异常处理**：处理任务执行中的异常情况

**配置参数**：
```yaml
runner:
  max_concurrent_tasks: 5     # 最大并发任务数
  task_timeout: 3600          # 任务超时时间（秒）
  retry_count: 3              # 失败重试次数
  retry_delay: 60             # 重试间隔（秒）
  scheduler:
    type: "priority"          # 调度算法：priority/fifo/round_robin
    check_interval: 10        # 调度检查间隔（秒）
  resource_limits:
    cpu_percent: 80           # CPU使用率限制
    memory_percent: 80        # 内存使用率限制
    disk_io_mbps: 100         # 磁盘I/O限制（MB/s）
```

## 4. API接口设计

### 4.1 RESTful API接口

#### 4.1.1 健康检查接口

**GET /api/v1/health** - 处理文件：`internal/handler/http/health.go`
- **功能**：检查Agent节点健康状态
- **响应**：
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "1.0.0",
  "uptime": 3600,
  "agent_id": "agent-001",
  "master_connected": true,
  "modules": {
    "scan": "running",
    "virusKill": "stopped",
    "plugins": "running"
  }
}
```

#### 4.1.2 配置管理接口

**GET /api/v1/config** - 处理文件：`internal/handler/http/config.go`
- **功能**：获取当前配置信息
- **响应**：
```json
{
  "config_version": "v1.2.3",
  "last_updated": "2024-01-01T12:00:00Z",
  "modules": {
    "scan": {
      "asset_scan": {
        "enabled": true,
        "threads": 100
      }
    }
  }
}
```

**PUT /api/v1/config** - 处理文件：`internal/handler/http/config.go`
- **功能**：更新配置信息
- **请求体**：
```json
{
  "config_version": "v1.2.4",
  "modules": {
    "scan": {
      "asset_scan": {
        "enabled": true,
        "threads": 150
      }
    }
  }
}
```

#### 4.1.3 任务管理接口

**GET /api/v1/tasks** - 处理文件：`internal/handler/http/task.go`
- **功能**：获取任务列表
- **响应**：
```json
{
  "tasks": [
    {
      "task_id": "task-001",
      "task_type": "asset_scan",
      "status": "running",
      "progress": 65,
      "start_time": "2024-01-01T10:00:00Z",
      "estimated_completion": "2024-01-01T12:30:00Z"
    }
  ]
}
```

**GET /api/v1/tasks/{task_id}** - 处理文件：`internal/handler/http/task.go`
- **功能**：获取特定任务详情
- **响应**：
```json
{
  "task_id": "task-001",
  "task_type": "asset_scan",
  "status": "running",
  "progress": 65,
  "start_time": "2024-01-01T10:00:00Z",
  "estimated_completion": "2024-01-01T12:30:00Z",
  "target_info": {
    "ip_range": "192.168.1.0/24",
    "port_range": "1-65535"
  },
  "results": {
    "total_hosts": 254,
    "alive_hosts": 45,
    "scanned_hosts": 30
  }
}
```

**POST /api/v1/tasks/{task_id}/control** - 处理文件：`internal/handler/http/task.go`
- **功能**：控制任务执行（暂停/恢复/取消）
- **请求体**：
```json
{
  "action": "pause"  // pause/resume/cancel
}
```

#### 4.1.4 插件管理接口

**GET /api/v1/plugins** - 处理文件：`internal/handler/http/plugin.go`
- **功能**：获取插件列表
- **响应**：
```json
{
  "plugins": [
    {
      "name": "shell-plugin",
      "version": "1.0.0",
      "status": "enabled",
      "description": "系统命令执行插件"
    },
    {
      "name": "file-plugin",
      "version": "1.0.0",
      "status": "disabled",
      "description": "文件操作插件"
    }
  ]
}
```

**POST /api/v1/plugins/{plugin_name}/control** - 处理文件：`internal/handler/http/plugin.go`
- **功能**：控制插件状态（启用/禁用）
- **请求体**：
```json
{
  "action": "enable"  // enable/disable
}
```

**POST /api/v1/plugins/{plugin_name}/execute** - 处理文件：`internal/handler/http/plugin.go`
- **功能**：执行插件功能
- **请求体**：
```json
{
  "command": "ls -la /tmp",
  "parameters": {
    "timeout": 30
  }
}
```

#### 4.1.5 系统指标接口

**GET /api/v1/metrics** - 处理文件：`internal/handler/http/metrics.go`
- **功能**：获取系统监控指标
- **响应**：
```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "system": {
    "cpu_percent": 45.2,
    "memory_percent": 62.8,
    "disk_usage_percent": 23.1,
    "network_bandwidth_mbps": 15.6
  },
  "tasks": {
    "running_count": 2,
    "completed_today": 12,
    "failed_today": 1
  },
  "modules": {
    "scan": {
      "status": "running",
      "active_scans": 2
    },
    "plugins": {
      "enabled_count": 2,
      "total_count": 4
    }
  }
}
```

### 4.2 gRPC接口定义

#### 4.2.1 Agent服务接口

**gRPC服务定义** (`internal/grpc/agent_service.go`)

```protobuf
// Agent节点服务定义
service AgentService {
  // 节点注册 - 处理文件: internal/handler/grpc/register.go
  rpc RegisterAgent(RegisterRequest) returns (RegisterResponse);
  
  // 心跳保持 - 处理文件: internal/handler/grpc/heartbeat.go
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  
  // 接收任务 - 处理文件: internal/handler/grpc/task.go
  rpc ReceiveTask(TaskRequest) returns (TaskResponse);
  
  // 上报状态 - 处理文件: internal/handler/grpc/status.go
  rpc ReportStatus(StatusRequest) returns (StatusResponse);
  
  // 上传结果 - 处理文件: internal/handler/grpc/result.go
  rpc UploadResult(stream ResultRequest) returns (ResultResponse);
  
  // 配置更新 - 处理文件: internal/handler/grpc/config.go
  rpc UpdateConfig(ConfigRequest) returns (ConfigResponse);
  
  // 插件控制 - 处理文件: internal/handler/grpc/plugin.go
  rpc ControlPlugin(PluginRequest) returns (PluginResponse);
}

// 注册请求
message RegisterRequest {
  string agent_id = 1;
  string version = 2;
  string hostname = 3;
  string ip_address = 4;
  SystemInfo system_info = 5;
  repeated string capabilities = 6;
}

// 注册响应
message RegisterResponse {
  bool success = 1;
  string message = 2;
  string master_id = 3;
  AgentConfig config = 4;
}

// 心跳请求
message HeartbeatRequest {
  string agent_id = 1;
  int64 timestamp = 2;
  AgentStatus status = 3;
  ResourceUsage resource_usage = 4;
}

// 心跳响应
message HeartbeatResponse {
  bool success = 1;
  int64 server_timestamp = 2;
  repeated string commands = 3;
}

// 任务请求
message TaskRequest {
  string task_id = 1;
  string task_type = 2;
  string target = 3;
  map<string, string> parameters = 4;
  int32 priority = 5;
  int64 timeout = 6;
}

// 任务响应
message TaskResponse {
  bool accepted = 1;
  string message = 2;
  int64 estimated_duration = 3;
}

// 状态上报请求
message StatusRequest {
  string agent_id = 1;
  repeated TaskStatus task_status = 2;
  ResourceUsage resource_usage = 3;
  repeated ModuleStatus module_status = 4;
}

// 状态上报响应
message StatusResponse {
  bool success = 1;
  string message = 2;
}

// 结果上传请求
message ResultRequest {
  string task_id = 1;
  string result_type = 2;
  bytes data = 3;
  bool is_final = 4;
}

// 结果上传响应
message ResultResponse {
  bool success = 1;
  string message = 2;
}

// 配置更新请求
message ConfigRequest {
  string config_version = 1;
  string config_data = 2;
  repeated string modules = 3;
}

// 配置更新响应
message ConfigResponse {
  bool success = 1;
  string message = 2;
  repeated string applied_modules = 3;
}

// 插件控制请求
message PluginRequest {
  string plugin_name = 1;
  string action = 2;  // enable/disable/execute
  map<string, string> parameters = 3;
}

// 插件控制响应
message PluginResponse {
  bool success = 1;
  string message = 2;
  string result = 3;
}
```

#### 4.2.2 数据模型定义

```protobuf
// 系统信息
message SystemInfo {
  string os = 1;
  string arch = 2;
  string kernel_version = 3;
  int32 cpu_cores = 4;
  int64 total_memory = 5;
  int64 total_disk = 6;
}

// Agent状态
message AgentStatus {
  string status = 1;  // idle/busy/error/offline
  int32 active_tasks = 2;
  int64 uptime = 3;
  string last_error = 4;
}

// 资源使用情况
message ResourceUsage {
  float cpu_percent = 1;
  float memory_percent = 2;
  float disk_usage_percent = 3;
  float network_bandwidth_mbps = 4;
  int32 open_files = 5;
  int32 network_connections = 6;
}

// 任务状态
message TaskStatus {
  string task_id = 1;
  string task_type = 2;
  string status = 3;  // running/paused/completed/failed
  int32 progress = 4;
  int64 start_time = 5;
  int64 estimated_completion = 6;
  string error_message = 7;
}

// 模块状态
message ModuleStatus {
  string module_name = 1;
  string status = 2;  // running/stopped/error
  string version = 3;
  int64 last_activity = 4;
  map<string, string> metrics = 5;
}

// Agent配置
message AgentConfig {
  string config_version = 1;
  map<string, string> global_config = 2;
  repeated ModuleConfig module_configs = 3;
  repeated PluginConfig plugin_configs = 4;
}

// 模块配置
message ModuleConfig {
  string module_name = 1;
  bool enabled = 2;
  map<string, string> parameters = 3;
}

// 插件配置
message PluginConfig {
  string plugin_name = 1;
  bool enabled = 2;
  map<string, string> parameters = 3;
  repeated string permissions = 4;
}
```

## 5. 数据模型设计

### 5.1 任务模型（Task）

```go
// Task 扫描任务模型
type Task struct {
    ID                string            `json:"id" db:"id"`
    Type              string            `json:"type" db:"type"`                           // 任务类型
    Status            string            `json:"status" db:"status"`                       // 任务状态
    Priority          int               `json:"priority" db:"priority"`                   // 优先级
    Target            string            `json:"target" db:"target"`                       // 扫描目标
    Parameters        map[string]string `json:"parameters" db:"parameters"`               // 任务参数
    Progress          int               `json:"progress" db:"progress"`                   // 执行进度
    StartTime         time.Time         `json:"start_time" db:"start_time"`               // 开始时间
    EndTime           *time.Time        `json:"end_time,omitempty" db:"end_time"`         // 结束时间
    EstimatedDuration int64             `json:"estimated_duration" db:"estimated_duration"` // 预估执行时间
    ActualDuration    int64             `json:"actual_duration" db:"actual_duration"`     // 实际执行时间
    ErrorMessage      string            `json:"error_message,omitempty" db:"error_message"` // 错误信息
    ResultCount       int               `json:"result_count" db:"result_count"`           // 结果数量
    CreatedAt         time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt         time.Time         `json:"updated_at" db:"updated_at"`
}

// TaskResult 任务结果模型
type TaskResult struct {
    ID       string                 `json:"id" db:"id"`
    TaskID   string                 `json:"task_id" db:"task_id"`
    Type     string                 `json:"type" db:"type"`         // 结果类型
    Data     map[string]interface{} `json:"data" db:"data"`         // 结果数据
    Severity string                 `json:"severity" db:"severity"` // 严重程度
    CreateAt time.Time              `json:"created_at" db:"created_at"`
}
```

### 5.2 配置模型（Config）

```go
// AgentConfig Agent配置模型
type AgentConfig struct {
    ID            string                 `json:"id" db:"id"`
    Version       string                 `json:"version" db:"version"`
    GlobalConfig  map[string]interface{} `json:"global_config" db:"global_config"`
    ModuleConfigs []ModuleConfig         `json:"module_configs" db:"module_configs"`
    PluginConfigs []PluginConfig         `json:"plugin_configs" db:"plugin_configs"`
    LastUpdated   time.Time              `json:"last_updated" db:"last_updated"`
    AppliedAt     *time.Time             `json:"applied_at,omitempty" db:"applied_at"`
}

// ModuleConfig 模块配置模型
type ModuleConfig struct {
    ModuleName string                 `json:"module_name" db:"module_name"`
    Enabled    bool                   `json:"enabled" db:"enabled"`
    Parameters map[string]interface{} `json:"parameters" db:"parameters"`
    Version    string                 `json:"version" db:"version"`
}

// PluginConfig 插件配置模型
type PluginConfig struct {
    PluginName  string                 `json:"plugin_name" db:"plugin_name"`
    Enabled     bool                   `json:"enabled" db:"enabled"`
    Parameters  map[string]interface{} `json:"parameters" db:"parameters"`
    Permissions []string               `json:"permissions" db:"permissions"`
    Version     string                 `json:"version" db:"version"`
}
```

### 5.3 系统状态模型（Status）

```go
// AgentStatus Agent状态模型
type AgentStatus struct {
    AgentID       string        `json:"agent_id" db:"agent_id"`
    Status        string        `json:"status" db:"status"`               // idle/busy/error/offline
    ActiveTasks   int           `json:"active_tasks" db:"active_tasks"`   // 活跃任务数
    Uptime        int64         `json:"uptime" db:"uptime"`               // 运行时间
    LastHeartbeat time.Time     `json:"last_heartbeat" db:"last_heartbeat"` // 最后心跳时间
    ResourceUsage ResourceUsage `json:"resource_usage" db:"resource_usage"` // 资源使用情况
    LastError     string        `json:"last_error,omitempty" db:"last_error"` // 最后错误
    UpdatedAt     time.Time     `json:"updated_at" db:"updated_at"`
}

// ResourceUsage 资源使用情况模型
type ResourceUsage struct {
    CPUPercent           float64 `json:"cpu_percent" db:"cpu_percent"`
    MemoryPercent        float64 `json:"memory_percent" db:"memory_percent"`
    DiskUsagePercent     float64 `json:"disk_usage_percent" db:"disk_usage_percent"`
    NetworkBandwidthMbps float64 `json:"network_bandwidth_mbps" db:"network_bandwidth_mbps"`
    OpenFiles            int     `json:"open_files" db:"open_files"`
    NetworkConnections   int     `json:"network_connections" db:"network_connections"`
    Timestamp            time.Time `json:"timestamp" db:"timestamp"`
}

// SystemInfo 系统信息模型
type SystemInfo struct {
    OS            string `json:"os" db:"os"`
    Arch          string `json:"arch" db:"arch"`
    KernelVersion string `json:"kernel_version" db:"kernel_version"`
    CPUCores      int    `json:"cpu_cores" db:"cpu_cores"`
    TotalMemory   int64  `json:"total_memory" db:"total_memory"`
    TotalDisk     int64  `json:"total_disk" db:"total_disk"`
    Hostname      string `json:"hostname" db:"hostname"`
    IPAddress     string `json:"ip_address" db:"ip_address"`
}
```

### 5.4 插件模型（Plugin）

```go
// Plugin 插件模型
type Plugin struct {
    Name        string                 `json:"name" db:"name"`
    Version     string                 `json:"version" db:"version"`
    Status      string                 `json:"status" db:"status"`         // enabled/disabled/error
    Type        string                 `json:"type" db:"type"`             // builtin/custom
    Description string                 `json:"description" db:"description"`
    Author      string                 `json:"author" db:"author"`
    Config      map[string]interface{} `json:"config" db:"config"`
    Permissions []string               `json:"permissions" db:"permissions"`
    InstallPath string                 `json:"install_path" db:"install_path"`
    LastActivity time.Time             `json:"last_activity" db:"last_activity"`
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// PluginExecution 插件执行记录模型
type PluginExecution struct {
    ID          string                 `json:"id" db:"id"`
    PluginName  string                 `json:"plugin_name" db:"plugin_name"`
    Command     string                 `json:"command" db:"command"`
    Parameters  map[string]interface{} `json:"parameters" db:"parameters"`
    Result      string                 `json:"result" db:"result"`
    Status      string                 `json:"status" db:"status"`         // success/failed/timeout
    Duration    int64                  `json:"duration" db:"duration"`     // 执行时间（毫秒）
    ErrorMessage string                `json:"error_message,omitempty" db:"error_message"`
    ExecutedAt  time.Time              `json:"executed_at" db:"executed_at"`
}
```

### 5.5 扫描结果模型（ScanResult）

```go
// ScanResult 扫描结果基础模型
type ScanResult struct {
    ID          string                 `json:"id" db:"id"`
    TaskID      string                 `json:"task_id" db:"task_id"`
    ScanType    string                 `json:"scan_type" db:"scan_type"`     // asset/web/poc/dir/domain/weakpwd/proxy/virus
    Target      string                 `json:"target" db:"target"`
    Status      string                 `json:"status" db:"status"`           // success/failed/partial
    Data        map[string]interface{} `json:"data" db:"data"`
    Severity    string                 `json:"severity" db:"severity"`       // critical/high/medium/low/info
    Confidence  float64                `json:"confidence" db:"confidence"`   // 置信度 0-1
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// AssetScanResult 资产扫描结果模型
type AssetScanResult struct {
    ScanResult
    IP          string   `json:"ip" db:"ip"`
    Hostname    string   `json:"hostname,omitempty" db:"hostname"`
    OS          string   `json:"os,omitempty" db:"os"`
    OpenPorts   []Port   `json:"open_ports" db:"open_ports"`
    Services    []Service `json:"services" db:"services"`
    IsAlive     bool     `json:"is_alive" db:"is_alive"`
}

// Port 端口信息模型
type Port struct {
    Number   int    `json:"number" db:"number"`
    Protocol string `json:"protocol" db:"protocol"` // tcp/udp
    State    string `json:"state" db:"state"`       // open/closed/filtered
    Service  string `json:"service,omitempty" db:"service"`
    Version  string `json:"version,omitempty" db:"version"`
    Banner   string `json:"banner,omitempty" db:"banner"`
}

// Service 服务信息模型
type Service struct {
    Name     string `json:"name" db:"name"`
    Version  string `json:"version,omitempty" db:"version"`
    Product  string `json:"product,omitempty" db:"product"`
    CPE      string `json:"cpe,omitempty" db:"cpe"`
    Port     int    `json:"port" db:"port"`
    Protocol string `json:"protocol" db:"protocol"`
}
```

## 6. 安全设计

### 6.1 认证与授权

#### 6.1.1 Agent身份认证
- **证书认证**：使用TLS客户端证书进行Agent身份认证
- **Token认证**：基于JWT Token的身份验证机制
- **双向认证**：Agent和Master之间的双向身份验证
- **证书轮换**：定期更新和轮换认证证书

#### 6.1.2 API访问控制
- **接口权限**：基于角色的API接口访问控制
- **IP白名单**：限制API访问的IP地址范围
- **频率限制**：API调用频率限制和防护
- **请求签名**：关键API请求的数字签名验证

### 6.2 数据安全

#### 6.2.1 传输加密
- **TLS加密**：所有网络通信使用TLS 1.3加密
- **证书管理**：自动化的TLS证书管理和更新
- **加密算法**：使用强加密算法（AES-256、RSA-4096）
- **完整性校验**：数据传输完整性校验

#### 6.2.2 存储加密
- **数据库加密**：敏感数据的数据库字段加密
- **文件加密**：配置文件和日志文件的加密存储
- **密钥管理**：安全的密钥生成、存储和轮换
- **数据脱敏**：日志中敏感信息的脱敏处理

### 6.3 插件安全

#### 6.3.1 沙箱隔离
- **进程隔离**：插件在独立进程中运行
- **文件系统隔离**：限制插件的文件系统访问权限
- **网络隔离**：控制插件的网络访问权限
- **资源限制**：限制插件的CPU、内存、磁盘使用

#### 6.3.2 权限控制
- **最小权限原则**：插件仅获得必要的最小权限
- **权限审计**：记录插件的所有权限使用情况
- **动态权限**：根据需要动态调整插件权限
- **权限撤销**：及时撤销不再需要的权限

### 6.4 系统安全

#### 6.4.1 系统加固
- **最小化安装**：仅安装必要的系统组件
- **安全配置**：系统和服务的安全配置
- **补丁管理**：及时安装安全补丁和更新
- **防火墙配置**：严格的网络防火墙规则

#### 6.4.2 监控告警
- **异常检测**：实时检测系统异常行为
- **入侵检测**：检测潜在的安全威胁
- **日志监控**：安全事件的日志监控和分析
- **告警机制**：及时的安全告警和响应

## 7. 性能优化

### 7.1 并发处理

#### 7.1.1 Goroutine池
- **工作池模式**：使用Goroutine池处理并发任务
- **动态调整**：根据系统负载动态调整池大小
- **任务队列**：高效的任务队列管理
- **负载均衡**：任务在Goroutine间的负载均衡

#### 7.1.2 资源管理
- **连接池**：数据库和网络连接池管理
- **内存管理**：高效的内存分配和回收
- **文件句柄**：文件句柄的合理使用和释放
- **系统资源**：CPU、内存、磁盘I/O的优化使用

### 7.2 缓存策略

#### 7.2.1 本地缓存
- **内存缓存**：热点数据的内存缓存
- **LRU策略**：最近最少使用的缓存淘汰策略
- **缓存预热**：系统启动时的缓存预热
- **缓存更新**：数据变更时的缓存同步更新

#### 7.2.2 数据缓存
- **查询缓存**：数据库查询结果缓存
- **配置缓存**：系统配置的缓存机制
- **结果缓存**：扫描结果的临时缓存
- **缓存失效**：基于TTL的缓存失效机制

### 7.3 网络优化

#### 7.3.1 连接优化
- **连接复用**：HTTP和gRPC连接复用
- **连接池**：网络连接池管理
- **超时控制**：合理的网络超时设置
- **重试机制**：网络请求的智能重试

#### 7.3.2 数据传输
- **数据压缩**：传输数据的压缩处理
- **批量传输**：批量数据传输优化
- **流式传输**：大文件的流式传输
- **断点续传**：支持传输中断后的续传

## 8. 监控与日志

### 8.1 系统监控

#### 8.1.1 性能指标
- **系统指标**：CPU、内存、磁盘、网络使用率
- **应用指标**：任务执行数量、成功率、响应时间
- **业务指标**：扫描目标数量、发现漏洞数量
- **自定义指标**：业务相关的自定义监控指标

#### 8.1.2 健康检查
- **服务健康**：各服务模块的健康状态检查
- **依赖检查**：外部依赖服务的可用性检查
- **资源检查**：系统资源的可用性检查
- **功能检查**：核心功能的可用性检查

### 8.2 日志管理

#### 8.2.1 日志分类
- **应用日志**：应用程序运行日志
- **访问日志**：API访问日志
- **错误日志**：系统错误和异常日志
- **审计日志**：安全相关的审计日志
- **性能日志**：性能监控相关日志

#### 8.2.2 日志处理
- **结构化日志**：使用JSON格式的结构化日志
- **日志级别**：DEBUG、INFO、WARN、ERROR、FATAL
- **日志轮转**：按大小和时间的日志文件轮转
- **日志压缩**：历史日志文件的压缩存储
- **日志清理**：过期日志文件的自动清理

### 8.3 告警机制

#### 8.3.1 告警规则
- **阈值告警**：基于指标阈值的告警
- **趋势告警**：基于指标变化趋势的告警
- **异常告警**：基于异常检测的告警
- **业务告警**：基于业务规则的告警

#### 8.3.2 告警处理
- **告警分级**：不同级别的告警处理策略
- **告警聚合**：相似告警的聚合处理
- **告警抑制**：避免告警风暴的抑制机制
- **告警恢复**：告警恢复的自动通知

## 9. 部署配置

### 9.1 Docker容器化

#### 9.1.1 Dockerfile配置
```dockerfile
# 多阶段构建
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agent ./cmd/agent

# 运行时镜像
FROM alpine:latest

# 安装必要工具
RUN apk --no-cache add ca-certificates nmap masscan

# 创建用户
RUN addgroup -g 1000 neoscan && \
    adduser -D -s /bin/sh -u 1000 -G neoscan neoscan

# 设置工作目录
WORKDIR /opt/neoscan

# 复制应用文件
COPY --from=builder /app/agent .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/scripts ./scripts

# 创建必要目录
RUN mkdir -p logs storage data && \
    chown -R neoscan:neoscan /opt/neoscan

# 切换用户
USER neoscan

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./agent health || exit 1

# 启动命令
CMD ["./agent", "start"]
```

#### 9.1.2 Docker Compose配置
```yaml
version: '3.8'

services:
  neoscan-agent:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: neoscan-agent
    hostname: neoscan-agent
    restart: unless-stopped
    environment:
      - AGENT_ID=agent-001
      - MASTER_HOST=neoscan-master
      - MASTER_PORT=9090
      - LOG_LEVEL=info
      - CONFIG_PATH=/opt/neoscan/configs
    volumes:
      - ./configs:/opt/neoscan/configs:ro
      - ./logs:/opt/neoscan/logs
      - ./storage:/opt/neoscan/storage
      - ./data:/opt/neoscan/data
    ports:
      - "8080:8080"  # HTTP API
      - "9091:9090"  # gRPC
    networks:
      - neoscan-network
    depends_on:
      - neoscan-master
    healthcheck:
      test: ["CMD", "./agent", "health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G

networks:
  neoscan-network:
    external: true

volumes:
  agent-data:
    driver: local
  agent-logs:
    driver: local
```

### 9.2 配置文件

#### 9.2.1 主配置文件（agent.yaml）
```yaml
# Agent基础配置
agent:
  id: "agent-001"
  name: "NeoScan Agent Node 1"
  version: "1.0.0"
  description: "NeoScan分布式扫描Agent节点"
  
# Master连接配置
master:
  host: "neoscan-master"
  port: 9090
  tls_enabled: true
  cert_file: "/opt/neoscan/certs/client.crt"
  key_file: "/opt/neoscan/certs/client.key"
  ca_file: "/opt/neoscan/certs/ca.crt"
  
# HTTP服务配置
http:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  
# gRPC服务配置
grpc:
  host: "0.0.0.0"
  port: 9090
  tls_enabled: true
  cert_file: "/opt/neoscan/certs/server.crt"
  key_file: "/opt/neoscan/certs/server.key"
  
# 系统配置
system:
  max_concurrent_tasks: 5
  task_timeout: 3600
  heartbeat_interval: 30
  resource_check_interval: 60
  
# 安全配置
security:
  enable_auth: true
  jwt_secret: "your-jwt-secret"
  token_expire: 3600
  max_login_attempts: 5
  lockout_duration: 300
```

#### 9.2.2 数据库配置（database.yaml）
```yaml
# SQLite配置
sqlite:
  path: "/opt/neoscan/data/agent.db"
  max_open_conns: 10
  max_idle_conns: 5
  conn_max_lifetime: 3600
  
# 缓存配置
cache:
  type: "memory"  # memory/redis
  max_size: 1000
  ttl: 3600
  cleanup_interval: 600
```

### 9.3 启动脚本

#### 9.3.1 启动脚本（start.sh）
```bash
#!/bin/bash

# NeoScan Agent启动脚本

set -e

# 配置变量
APP_NAME="neoscan-agent"
APP_DIR="/opt/neoscan"
CONFIG_DIR="$APP_DIR/configs"
LOG_DIR="$APP_DIR/logs"
PID_FILE="$APP_DIR/$APP_NAME.pid"

# 检查配置文件
if [ ! -f "$CONFIG_DIR/agent.yaml" ]; then
    echo "错误: 配置文件不存在: $CONFIG_DIR/agent.yaml"
    exit 1
fi

# 创建必要目录
mkdir -p "$LOG_DIR"
mkdir -p "$APP_DIR/data"
mkdir -p "$APP_DIR/storage"

# 检查是否已经运行
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        echo "$APP_NAME 已经在运行 (PID: $PID)"
        exit 1
    else
        rm -f "$PID_FILE"
    fi
fi

# 启动应用
echo "启动 $APP_NAME..."
cd "$APP_DIR"
nohup ./agent start > "$LOG_DIR/startup.log" 2>&1 &
echo $! > "$PID_FILE"

# 等待启动
sleep 3

# 检查启动状态
if ps -p $(cat "$PID_FILE") > /dev/null 2>&1; then
    echo "$APP_NAME 启动成功 (PID: $(cat "$PID_FILE"))"
else
    echo "$APP_NAME 启动失败"
    rm -f "$PID_FILE"
    exit 1
fi
```

#### 9.3.2 停止脚本（stop.sh）
```bash
#!/bin/bash

# NeoScan Agent停止脚本

set -e

# 配置变量
APP_NAME="neoscan-agent"
APP_DIR="/opt/neoscan"
PID_FILE="$APP_DIR/$APP_NAME.pid"

# 检查PID文件
if [ ! -f "$PID_FILE" ]; then
    echo "$APP_NAME 未运行"
    exit 0
fi

# 获取PID
PID=$(cat "$PID_FILE")

# 检查进程是否存在
if ! ps -p "$PID" > /dev/null 2>&1; then
    echo "$APP_NAME 进程不存在 (PID: $PID)"
    rm -f "$PID_FILE"
    exit 0
fi

# 优雅停止
echo "停止 $APP_NAME (PID: $PID)..."
kill -TERM "$PID"

# 等待进程结束
for i in {1..30}; do
    if ! ps -p "$PID" > /dev/null 2>&1; then
        echo "$APP_NAME 已停止"
        rm -f "$PID_FILE"
        exit 0
    fi
    sleep 1
done

# 强制停止
echo "强制停止 $APP_NAME..."
kill -KILL "$PID"
sleep 2

if ! ps -p "$PID" > /dev/null 2>&1; then
    echo "$APP_NAME 已强制停止"
    rm -f "$PID_FILE"
else
    echo "无法停止 $APP_NAME"
    exit 1
fi
```

## 10. 系统优势与特点

### 10.1 技术优势

#### 10.1.1 轻量化设计
- **资源占用小**：最小化的系统资源占用
- **快速部署**：简单的部署和配置流程
- **高效运行**：优化的算法和数据结构
- **低延迟**：快速的响应和处理能力

#### 10.1.2 模块化架构
- **松耦合设计**：模块间低耦合，高内聚
- **热插拔支持**：模块的动态加载和卸载
- **独立开发**：模块可独立开发和测试
- **易于维护**：清晰的模块边界和接口

#### 10.1.3 高可扩展性
- **水平扩展**：支持Agent节点的水平扩展
- **插件系统**：丰富的插件扩展机制
- **API开放**：完善的API接口体系
- **标准化接口**：统一的开发接口规范

### 10.2 功能特点

#### 10.2.1 全面的扫描能力
- **多类型扫描**：支持资产、Web、POC、目录等多种扫描
- **工具集成**：集成主流安全扫描工具
- **自定义扩展**：支持自定义扫描模块
- **智能识别**：基于指纹的智能识别技术

#### 10.2.2 智能任务调度
- **负载均衡**：智能的任务分发和负载均衡
- **优先级调度**：基于优先级的任务调度
- **资源感知**：基于资源使用情况的调度决策
- **故障恢复**：任务执行失败的自动恢复

#### 10.2.3 安全可靠
- **多层安全**：从网络到应用的多层安全防护
- **权限控制**：细粒度的权限控制机制
- **审计追踪**：完整的操作审计和追踪
- **数据保护**：敏感数据的加密保护

### 10.3 运维友好

#### 10.3.1 易于部署
- **容器化部署**：基于Docker的容器化部署
- **自动化配置**：自动化的配置管理
- **一键启停**：简单的启动和停止操作
- **健康检查**：自动的健康状态检查

#### 10.3.2 监控完善
- **实时监控**：实时的系统状态监控
- **性能指标**：丰富的性能监控指标
- **告警机制**：及时的异常告警
- **日志管理**：完善的日志管理体系

#### 10.3.3 维护便捷
- **远程管理**：支持远程配置和管理
- **在线更新**：支持在线配置更新
- **故障诊断**：完善的故障诊断工具
- **性能调优**：灵活的性能调优选项

## 11. 应用场景

### 11.1 企业安全
- **内网安全扫描**：企业内网的安全扫描和评估
- **资产发现**：网络资产的自动发现和清点
- **漏洞管理**：漏洞的发现、评估和管理
- **合规检查**：安全合规性检查和审计

### 11.2 渗透测试
- **信息收集**：目标系统的信息收集
- **漏洞发现**：安全漏洞的发现和验证
- **权限提升**：系统权限的提升测试
- **横向移动**：网络内部的横向移动测试

### 11.3 安全运营
- **持续监控**：安全状态的持续监控
- **威胁检测**：安全威胁的实时检测
- **应急响应**：安全事件的应急响应
- **风险评估**：安全风险的评估和管理

### 11.4 开发测试
- **安全测试**：应用程序的安全测试
- **代码审计**：源代码的安全审计
- **配置检查**：系统配置的安全检查
- **依赖扫描**：第三方依赖的安全扫描

---

**文档版本**：v1.0  
**创建日期**：2025年1月21日  
**最后更新**：2025年1月21日  
**文档状态**：正式版  
**适用范围**：NeoScan Agent节点开发和部署

> 注：本文档详细描述了NeoScan Agent节点的功能设计、技术架构、接口规范和部署配置，为开发团队提供了完整的技术指导和实现参考。