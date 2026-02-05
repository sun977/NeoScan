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

#### 核心架构特性
- **无状态执行**: 不维护任务上下文，执行完即销毁
- **Factory模式**: 统一能力构建入口，确保配置一致性
- **QoS自适应限流**: 基于RTT估算的动态并发控制
- **跨平台支持**: Linux/Windows/macOS原生支持
- **零外部依赖**: 纯Go实现，无需安装额外工具

#### 扫描引擎详细能力

##### 1. IP存活扫描 (ip_alive_scan) ✅ 已完成
**核心能力**:
- 支持ARP/ICMP/TCP Connect三种探测协议
- 自动策略选择（同网段优先ARP，跨网段ICMP+TCP）
- 支持手动指定协议开关
- 内置QoS自适应限流（初始200，最大5000并发）
- 支持TTL猜测操作系统
- 支持主机名反向解析

**参数**:
- `enable_arp` (bool): 启用ARP探测
- `enable_icmp` (bool): 启用ICMP探测
- `enable_tcp` (bool): 启用TCP Connect探测
- `tcp_ports` ([]int): TCP探测端口列表（默认[22,23,80,139,512,443,445,3389]）
- `concurrency` (int): 并发数（默认1000）
- `resolve_hostname` (bool): 解析主机名（默认false）
- `timeout`: 默认1小时

##### 2. 端口服务扫描 (port_scan) ✅ 已完成
**核心能力**:
- 基于Gonmap引擎的端口扫描
- 支持Nmap服务指纹识别（内置规则库）
- 支持服务探测（可选）
- 支持CPE指纹识别
- 内置QoS自适应限流（初始100，最大2000并发）
- 动态超时控制（基于RTT估算）

**参数**:
- `port` (string): 端口范围（如"80,443,1000-2000"、"top100"、"top1000"）
- `rate` (int): 扫描速率/并发数（默认1000）
- `service_detect` (bool): 启用服务识别（默认true）
- `timeout`: 默认1小时

##### 3. 操作系统扫描 (os_scan) ✅ 已完成
**核心能力**:
- 多引擎并发竞速识别
- TTL引擎：基于TTL值快速猜测
- Nmap Stack引擎：基于Nmap OS指纹库深度识别
- Service Banner引擎：基于服务Banner识别
- 支持三种模式：fast/deep/auto
- 动态超时控制（基于RTT估算）

**参数**:
- `mode` (string): 扫描模式
  - `fast`: 仅TTL快速识别
  - `deep`: Nmap深度识别
  - `auto`: 混合模式（默认）
- `timeout`: 默认30分钟

##### 4. 弱口令爆破 (brute_force) ✅ 已完成
**核心能力**:
- 支持15+种协议爆破：SSH, RDP, MySQL, Redis, FTP, Telnet, SNMP, SMB, PostgreSQL, Oracle, MSSQL, MongoDB, Elasticsearch, Clickhouse等
- 内置字典管理器
- 支持自定义用户名/密码字典
- 支持字典文件路径
- 全局QoS自适应限流（初始50，最大200并发）
- 支持找到即停或继续爆破
- 单次尝试超时3秒

**参数**:
- `service` (string): 目标服务（必须）
- `port` (string): 目标端口（支持多个端口逗号分隔）
- `users` (string): 用户名（支持文件路径或逗号分隔）
- `pass` (string): 密码（支持文件路径或逗号分隔）
- `stop_on_success` (bool): 找到即停（默认true）
- `timeout`: 默认1小时

##### 5. Web综合扫描 (web_scan) 🟡 部分完成
**核心能力**: Web指纹识别、目录枚举、敏感文件检测

**参数**:
- `ports` (string): Web端口（默认"80,443"）
- `path` (string): 扫描路径（默认"/"）
- `method` (string): HTTP方法（默认"GET"）
- `timeout`: 默认30分钟

##### 6. 目录扫描 (dir_scan) 🟡 部分完成
**核心能力**: 基于字典的目录爆破

**参数**:
- `dict` (string): 字典路径
- `extensions` (string): 文件扩展名
- `threads` (int): 线程数（默认10）
- `timeout`: 默认2小时

##### 7. 漏洞扫描 (vuln_scan) 🟡 部分完成
**核心能力**: 集成Nuclei引擎进行漏洞验证

**参数**:
- `templates` (string): Nuclei模板路径
- `severity` (string): 漏洞等级（默认"medium,high,critical"）
- `timeout`: 默认1小时

##### 8. 子域名扫描 (subdomain) 🟡 部分完成
**核心能力**: 字典爆破/被动收集子域名

**参数**:
- `dict` (string): 字典路径
- `threads` (int): 线程数（默认10）
- `timeout`: 默认1小时

##### 9. 代理服务 (proxy) 🟡 部分完成
**核心能力**: SOCKS5/HTTP代理服务器、端口转发

**参数**:
- `mode` (string): 代理模式（socks5/http/port_forward，默认socks5）
- `listen` (string): 监听地址（默认":1080"）
- `auth` (string): 认证信息（格式user:pass）
- `forward` (string): 转发目标（仅port_forward模式需要）
- `timeout`: 0（无超时）

##### 10. 原始命令执行 (raw_cmd) 🔴 未实现
**用途**: 执行原始命令，用于特殊场景

#### Pipeline编排能力 ✅ 已完成
**核心能力**:
- 自动化编排器（AutoRunner）
- 漏斗式扫描流程：存活→端口→服务→OS→Web→漏洞
- 目标生成器（支持IP/CIDR/域名）
- 上下文传递（各阶段间共享数据）
- 支持阶段依赖和条件执行

**Pipeline参数**:
- `target` (string): 扫描目标（必须）
- `concurrency` (int): 并发数（默认10）
- `port_range` (string): 端口范围（默认"top1000"）
- `show_summary` (bool): 显示摘要
- `enable_brute` (bool): 启用爆破（默认false）
- `brute_users` (string): 爆破用户名
- `brute_pass` (string): 爆破密码

#### 网络基础设施
- **统一网络连接层 (Dialer)**: 全局超时控制、代理支持（SOCKS5）、连接复用
- **Raw Socket能力 (NetRaw)**: Linux完整支持，Windows/macOS降级处理
- **QoS自适应限流**: RTT估算器、自适应限流器、动态超时控制

#### 结果输出能力
- **Console**: 控制台输出
- **CSV**: 文件输出
- **JSON**: 可扩展格式

#### 完成度总结
| 能力模块 | 完成度 | 说明 |
|---------|--------|------|
| IP存活扫描 | ✅ 100% | 完整实现，支持多协议、QoS |
| 端口服务扫描 | ✅ 100% | 完整实现，基于Gonmap引擎 |
| 操作系统扫描 | ✅ 100% | 完整实现，多引擎竞速 |
| 弱口令爆破 | ✅ 100% | 完整实现，支持15+协议 |
| Web扫描 | 🟡 50% | 参数已定义，扫描器待实现 |
| 目录扫描 | 🟡 50% | 参数已定义，扫描器待实现 |
| 漏洞扫描 | 🟡 50% | 参数已定义，扫描器待实现 |
| 子域名扫描 | 🟡 50% | 参数已定义，扫描器待实现 |
| 代理服务 | 🟡 50% | 参数已定义，扫描器待实现 |
| 原始命令 | 🔴 0% | 未实现 |
| Pipeline编排 | ✅ 100% | 完整实现 |
| 网络基础设施 | ✅ 100% | 完整实现 |
| Factory模式 | ✅ 100% | 完整实现 |

**总体完成度**: 约 **75%**

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