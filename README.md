# NeoScan

NeoScan 是一个综合性自动扫描信息收集系统，采用分布式架构设计，支持多种扫描类型和智能任务调度，主要用于网络安全评估、资产发现和漏洞检测。

## 项目架构

### 核心特性
- **分布式架构**: Master-Agent 架构，支持多节点协同工作
- **Thin Agent 架构**: 采用 "瘦客户端" 模式，Agent 无状态，Master 统一编排
- **HTTP/WebSocket 通信**: 全面采用标准 HTTP 协议，摒弃复杂的 gRPC，降低运维成本
- **智能调度**: Master 负责全生命周期管理，支持复杂的 ScanStage 调度循环
- **多模块扫描**: 资产扫描、Web扫描、POC漏洞扫描、目录扫描、域名扫描、弱口令扫描、代理扫描
- **插件系统**: 支持热插拔的插件架构，内置Shell、文件、监控插件
- **实时监控**: WebSocket实时通信，任务状态实时监控
- **安全机制**: JWT + RBAC认证，TLS加密通信，插件沙箱执行

### 技术栈
- **后端技术**: Go 1.25+, Gin (HTTP), MySQL 8.0+, SQLite, Redis 6.0+, RabbitMQ 3.8+
- **前端技术**: Vue.js 3 + Element Plus
- **部署技术**: Docker + Docker Compose, Nginx

## 目录结构

```
NeoScan/
├── neoMaster/              # Master节点
│   ├── cmd/               # 应用入口
│   ├── internal/          # 内部代码
│   ├── web/               # 前端代码
│   ├── configs/           # 配置文件
│   └── ...                # 其他目录
├── neoAgent/               # Agent节点
│   ├── cmd/               # 应用入口
│   ├── internal/          # 内部代码
│   ├── tools/             # 扫描工具
│   └── ...                # 其他目录
└── sdds/                  # 系统设计文档
```

## 核心功能

### Master节点功能 (Brain)
1. **工作流编排**: 基于 DAG 的多阶段扫描任务调度 (Stage Scheduling)
2. **任务管理**: 原子任务 (ScanStage) 的生成、分发、回收
3. **数据清洗**: 负责将 Agent 返回的原始数据转换为标准资产
4. **节点管理**: Agent 节点注册、状态监控、配置推送
5. **资产管理**: 资产发现、同步、清单管理、导入导出
6. **监控预警**: 漏洞监控、GitHub爬取、告警通知
7. **插件管理**: 插件安装、远程控制、安全执行
8. **用户管理**: 用户认证、权限控制、审计日志
9. **报告管理**: 报告生成、模板管理、数据导出
10. **通知系统**: 蓝信、SEC、邮件、Webhook通知

### Agent节点功能 (Worker)
1. **无状态执行**: 不维护任务上下文，执行完即销毁
2. **扫描引擎**:
   - 资产扫描: IP存活探测、端口扫描、服务识别、OS识别
   - Web扫描: 网页爬虫、指纹识别、CMS检测、技术栈识别
   - POC扫描: 漏洞验证、POC执行、安全检测
   - 目录扫描: 目录爆破、敏感文件检测
   - 域名扫描: 子域名发现、DNS解析
   - 弱口令扫描: 暴力破解、服务认证
   - 代理扫描: 代理发现、代理验证

3. **病毒查杀模块**: YARA引擎、恶意软件检测、隔离处理

4. **插件系统**:
   - Shell插件: 命令执行、白名单控制
   - 文件插件: 文件管理、上传下载、权限控制
   - 监控插件: 系统监控、进程监控、网络监控

## 部署要求

### 系统要求
- **Master节点**: 4核CPU, 8GB内存, 100GB存储
- **Agent节点**: 2核CPU, 4GB内存, 50GB存储
- **依赖服务**: MySQL 8.0+, Redis 6.0+, RabbitMQ 3.8+

### Docker部署
项目支持Docker Compose部署，包含Master节点、Agent节点以及依赖服务。

## 安全机制

- **认证和授权**: JWT Token无状态认证，RBAC基于角色的访问控制
- **数据安全**: HTTPS/TLS加密通信，敏感数据AES加密存储
- **插件安全**: 沙箱执行，权限控制，命令白名单，资源限制


## 贡献指南
- **架构设计**: Sun977
- **功能实现**: Sun977


## 联系方式
- **邮箱**: jiuwei977@foxmail.com


## 开发指南

### 项目构建
```bash
# Master节点构建
cd neoMaster
go build -o neoMaster

# Agent节点构建
cd neoAgent
go build -o neoAgent
```

### 代码规范
- 遵循Go官方代码规范
- RESTful风格API设计
- 结构化日志记录
- 高测试覆盖率

## 文档

更多详细信息请参考[sdds](sdds)目录下的系统设计文档:
- [项目概览](sdds/00.NeoScan项目概览.md)
- [产品需求文档](sdds/01.NeoScan产品需求文档v2.0.md)
- [目录结构设计](sdds/02.NeoScan目录结构设计v6.0.md)
- [Master详细实现](neoMaster/docs)
- [Agent详细实现](neoAgent/docs)


**注： 项目处于开发阶段，功能完善度较低，后续会慢慢补充完善(请勿剽窃设计用于商业用途)。**

![sponsor](https://vps.town/static/images/sponsor.png)
**感谢vps.town提供的服务器赞助与支持**