# NeoAgent

NeoAgent 是 NeoScan 分布式扫描系统的 Agent 节点，负责执行具体的扫描任务、插件管理和系统监控。Agent 节点通过 http 与 Master 节点通信，接收任务分发并上报执行结果。

Agent 运行模式有两种：
- **集群模式**: Master 节点负责任务分配、结果上报、任务管理，Agent 节点只负责执行任务。
- 单机模式：Agent 充当扫描器，执行任务并返回结果（人工命令行操作 Agent）。

## 核心特性

- **多模块扫描**: 支持资产扫描、Web扫描、POC漏洞扫描、目录扫描、域名扫描、弱口令扫描、代理扫描
- **病毒查杀**: 基于 YARA 引擎的恶意软件检测和隔离处理
- **插件系统**: 支持 Shell、文件管理、系统监控等插件的热插拔和安全执行
- **智能调度**: 自动负载均衡，任务优先级管理，资源限制控制
- **实时通信**: 基于 gRPC 的高性能通信，WebSocket 实时状态推送
- **安全机制**: JWT 认证，TLS 加密，插件沙箱执行，命令白名单

## 技术栈

- **核心框架**: Go 1.22+, Gin (HTTP服务)
- **数据存储**: SQLite (本地数据)
- **扫描工具**: Nmap, Masscan, Nuclei, HTTPx,YARA
- **配置管理**: YAML + 环境变量
- **容器化**: Docker + Docker Compose

## 目录结构

```
neoAgent/
├── cmd/                    # 应用入口
│   └── agent/
│       └── main.go        # Agent主程序
├── internal/              # 内部代码
│   ├── app/              # 应用初始化
│   ├── config/           # 配置管理
│   ├── handler/          # HTTP处理器
│   ├── service/          # 业务服务
│   ├── executor/         # 扫描执行器
│   ├── plugin/           # 插件系统
│   ├── model/            # 数据模型
│   ├── repo/             # 数据访问层
│   └── pkg/              # 公共包
├── configs/              # 配置文件
│   ├── config.yaml       # 默认配置
│   └── nuclei/           # Nuclei模板
├── docker/               # Docker配置
│   ├── Dockerfile        # 生产环境镜像
│   ├── Dockerfile.dev    # 开发环境镜像
│   ├── docker-compose.yaml      # 生产环境编排
│   └── docker-compose.dev.yaml  # 开发环境编排
├── tools/                # 扫描工具
│   ├── nmap/            # Nmap工具
│   ├── masscan/         # Masscan工具
│   ├── nuclei/          # Nuclei工具
│   └── yara/            # YARA规则
├── data/                 # 数据目录
├── logs/                 # 日志目录
├── work/                 # 工作目录
├── temp/                 # 临时目录
├── .env.example         # 环境变量示例
├── .air.toml            # 热重载配置
├── Makefile             # 构建脚本
└── README.md            # 说明文档
```

## 快速开始

### 环境要求

- Go 1.22+
- Docker & Docker Compose (可选)
- Redis 6.0+ (开发环境可选)

### 本地开发

1. **克隆项目**
```bash
git clone <repository-url>
cd NeoScan/neoAgent
```

2. **安装依赖**
```bash
make deps
```

3. **配置环境**
```bash
# 复制环境变量配置
cp .env.example .env

# 编辑配置文件
vim .env
```

4. **运行开发环境**
```bash
# 使用 Air 热重载
make dev

# 或直接运行
make run
```

### Docker 部署

#### 开发环境
```bash
# 启动开发环境
make docker-dev

# 查看日志
docker-compose -f docker/docker-compose.dev.yaml logs -f
```

#### 生产环境
```bash
# 构建并启动
make docker-run

# 查看状态
docker-compose -f docker/docker-compose.yaml ps
```

## 配置说明

### 配置文件优先级
1. 环境变量 (最高优先级)
2. `.env` 文件
3. `config.yaml` 文件 (默认配置)

### 核心配置项

#### 服务器配置
```yaml
server:
  host: "0.0.0.0"          # 监听地址
  port: 8081               # HTTP端口
  grpc_port: 9091          # gRPC端口
  mode: "release"          # 运行模式: debug/release
```

#### Master连接配置
```yaml
master:
  host: "localhost"        # Master地址
  port: 9090              # Master gRPC端口
  tls_enabled: false      # 是否启用TLS
  cert_file: ""           # 证书文件路径
```

#### Agent配置
```yaml
agent:
  id: ""                  # Agent ID (自动生成)
  name: "neoagent-001"    # Agent名称
  region: "default"       # 所属区域
  tags: ["scanner"]       # 标签
```

#### 扫描工具配置
```yaml
executors:
  nmap:
    path: "/usr/bin/nmap"
    timeout: 300
    max_concurrent: 5
  nuclei:
    path: "/usr/bin/nuclei"
    templates_dir: "/opt/nuclei-templates"
    timeout: 600
```

### 环境变量配置

关键环境变量：
```bash
# 基础配置
NEOSCAN_ENV=production
NEOSCAN_LOG_LEVEL=info

# Master连接
NEOSCAN_MASTER_HOST=master.example.com
NEOSCAN_MASTER_PORT=9090
NEOSCAN_MASTER_TLS_ENABLED=true

# Agent配置
NEOSCAN_AGENT_NAME=agent-prod-001
NEOSCAN_AGENT_REGION=us-east-1

# 数据库
NEOSCAN_DB_TYPE=sqlite
NEOSCAN_DB_PATH=./data/agent.db

# Redis
NEOSCAN_REDIS_HOST=redis
NEOSCAN_REDIS_PORT=6379
NEOSCAN_REDIS_PASSWORD=your_password
```

## 扫描模块

### 资产扫描
- **IP存活探测**: ICMP/TCP/UDP探测
- **端口扫描**: TCP/UDP端口扫描，服务识别
- **OS识别**: 操作系统指纹识别
- **服务识别**: 服务版本检测

### Web扫描
- **网页爬虫**: 深度爬取，链接发现
- **指纹识别**: Web应用指纹库匹配
- **CMS检测**: 内容管理系统识别
- **技术栈识别**: 框架、中间件识别

### POC扫描
- **漏洞验证**: 基于Nuclei的POC执行
- **自定义POC**: 支持自定义漏洞检测脚本
- **安全检测**: 常见安全漏洞检测

### 目录扫描
- **目录爆破**: 常见目录和文件发现
- **敏感文件**: 配置文件、备份文件检测
- **状态码分析**: HTTP响应状态分析

### 域名扫描
- **子域名发现**: 多种方式子域名枚举
- **DNS解析**: A/AAAA/CNAME/MX记录查询
- **域名劫持**: 域名安全检测

### 弱口令扫描
- **暴力破解**: SSH/FTP/MySQL/Redis等服务
- **字典攻击**: 内置和自定义字典
- **智能爆破**: 基于目标特征的密码生成

### 代理扫描
- **代理发现**: HTTP/SOCKS代理探测
- **代理验证**: 代理可用性和匿名性测试
- **代理链**: 多级代理检测

## 插件系统

### Shell插件
- **命令执行**: 安全的远程命令执行
- **白名单控制**: 命令白名单机制
- **结果收集**: 命令执行结果收集

### 文件插件
- **文件管理**: 文件上传、下载、删除
- **权限控制**: 文件访问权限管理
- **路径限制**: 文件操作路径限制

### 监控插件
- **系统监控**: CPU、内存、磁盘监控
- **进程监控**: 进程状态和资源使用
- **网络监控**: 网络连接和流量监控

## 病毒查杀

### YARA引擎
- **规则管理**: YARA规则加载和更新
- **文件扫描**: 文件恶意软件检测
- **内存扫描**: 进程内存恶意代码检测

### 隔离处理
- **文件隔离**: 恶意文件安全隔离
- **进程终止**: 恶意进程安全终止
- **日志记录**: 详细的检测和处理日志

## 开发指南

### 构建命令

```bash
# 查看所有可用命令
make help

# 开发流程
make fmt vet test build    # 格式化、检查、测试、构建
make quick                 # 快速开发流程

# 构建
make build                 # 构建当前平台
make build-all            # 构建所有平台
make build-linux          # 构建Linux版本
make build-windows        # 构建Windows版本

# 测试
make test                 # 运行测试
make test-coverage        # 测试覆盖率报告
make lint                 # 代码质量检查

# Docker
make docker-build         # 构建Docker镜像
make docker-run           # 运行生产环境
make docker-dev           # 运行开发环境
make docker-stop          # 停止Docker服务

# 部署
make release              # 创建发布包
make install              # 安装到系统
make ci                   # 完整CI流程
```

### 代码规范

1. **包结构**: 严格遵循 `Controller/Handler → Service → Repository → Database` 层级调用关系
2. **命名规范**: 使用驼峰命名，包名小写，接口名以 `I` 开头
3. **错误处理**: 统一使用 `internal/pkg/logger` 包进行日志记录
4. **类型转换**: 统一使用 `internal/pkg/convert` 包进行类型转换
5. **配置管理**: 敏感信息放入 `.env` 文件，不提交到版本控制

### 日志规范

关键字段说明：
- `path`: Handler层写请求URI路径，Service层写操作名称
- `operation`: 写操作名称，如 `scan`, `plugin_execute`
- `option`: 写具体操作步骤，如 `scanService.ExecuteNmap`
- `func_name`: 写具体函数路径，如 `handler.scan.nmap.Execute`

### 测试规范

- 测试文件放在 `test/日期/` 目录下
- 文件命名格式: `日期_模块名_test.go`
- 覆盖率要求: 单元测试覆盖率 > 80%

## 监控和运维

### 健康检查
```bash
# HTTP健康检查
curl http://localhost:8081/health

# 详细状态
curl http://localhost:8081/status
```

### 日志查看
```bash
# 实时日志
make logs

# Docker日志
docker-compose logs -f neoagent
```

### 性能监控
- **Prometheus指标**: `/metrics` 端点
- **pprof性能分析**: `/debug/pprof/` 端点 (开发模式)
- **系统资源**: CPU、内存、磁盘使用情况

## 安全机制

### 认证和授权
- **JWT认证**: 与Master节点的安全通信
- **TLS加密**: gRPC通信加密
- **API密钥**: 插件和工具的访问控制

### 插件安全
- **沙箱执行**: 插件在受限环境中运行
- **权限控制**: 细粒度的权限管理
- **资源限制**: CPU、内存、网络资源限制
- **命令白名单**: Shell插件命令白名单机制

### 数据安全
- **敏感数据加密**: AES加密存储
- **安全传输**: HTTPS/TLS通信
- **访问日志**: 详细的访问和操作审计

## 故障排除

### 常见问题

1. **连接Master失败**
   - 检查网络连通性
   - 验证Master地址和端口
   - 检查TLS配置

2. **扫描工具不可用**
   - 检查工具安装路径
   - 验证工具可执行权限
   - 查看工具版本兼容性

3. **插件执行失败**
   - 检查插件权限配置
   - 查看沙箱环境设置
   - 验证命令白名单

4. **性能问题**
   - 调整并发数配置
   - 检查资源限制设置
   - 监控系统资源使用

### 日志分析
```bash
# 错误日志
grep "ERROR" logs/agent.log

# 扫描任务日志
grep "scan" logs/agent.log | grep "operation"

# 插件执行日志
grep "plugin" logs/agent.log
```

## 版本信息

- **当前版本**: v1.0.0
- **Go版本**: 1.22+
- **构建时间**: 动态生成
- **Git提交**: 动态获取

查看版本信息：
```bash
make version
./bin/neoAgent --version
```

## 贡献指南

1. **Fork项目** 并创建功能分支
2. **遵循代码规范** 和项目约定
3. **编写测试用例** 确保代码质量
4. **提交Pull Request** 并描述变更内容

## 许可证

本项目采用 MIT 许可证，详见 LICENSE 文件。

## 联系方式

- **作者**: Sun977
- **邮箱**: jiuwei977@foxmail.com
- **项目地址**: [NeoScan Repository]

## 更新日志

### v1.0.0 (2025-01-14)
- 初始版本发布
- 完整的扫描模块实现
- 插件系统和病毒查杀功能
- Docker容器化支持
- 完善的配置管理和日志系统