# NeoScan Master 节点功能设计说明

> 基于 NeoScan 产品需求文档 v1.0 和目录结构设计 pkg-v4.0
> Master 节点采用单Master架构，负责整个扫描系统的统一管理和调度
> 设计日期：2025年8月

## 1. Master 节点核心功能概述

NeoScan Master 节点是整个分布式扫描系统的控制中心，负责：

- **集中管理**：统一管理所有 Agent 节点和扫描任务
- **智能调度**：基于负载均衡的任务分发和调度
- **数据汇聚**：收集和处理所有扫描结果数据
- **用户界面**：提供 Web 管理界面和 API 接口
- **监控预警**：实时监控系统状态和安全威胁
- **报告生成**：自动生成各类扫描和分析报告

## 2. 系统架构设计

### 2.1 分层架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Web 前端界面层                             │
├─────────────────────────────────────────────────────────────┤
│                    HTTP API 接口层                           │
├─────────────────────────────────────────────────────────────┤
│                    业务逻辑服务层                             │
├─────────────────────────────────────────────────────────────┤
│                    数据访问仓储层                             │
├─────────────────────────────────────────────────────────────┤
│              数据存储层 (MySQL + Redis + 文件)                │
└─────────────────────────────────────────────────────────────┘
```

Controller/Handler → Service → Repository → Database
     ↓                ↓           ↓
  路由处理        业务逻辑    数据访问
  参数解析        参数验证    SQL操作
  响应格式化      业务规则    事务管理


### 2.2 通信协议

- **HTTP/HTTPS**：Web 界面和 RESTful API
- **gRPC**：与 Agent 节点的高性能通信
- **WebSocket**：实时数据推送和双向通信
- **RabbitMQ**：异步消息队列和任务分发

## 3. 核心功能模块

### 3.1 认证授权模块

#### 功能描述
- 用户身份认证和会话管理
- 基于角色的访问控制 (RBAC)
- JWT Token 管理
- 安全审计日志

#### 核心服务
- **JWT 令牌管理** (`internal/service/auth/jwt.go`)
  - Token 生成、验证、刷新
  - 支持访问令牌和刷新令牌机制
  - Token 黑名单管理

- **RBAC 权限控制** (`internal/service/auth/rbac.go`)
  - 角色定义和权限分配
  - 细粒度的功能权限控制
  - 动态权限验证

- **会话管理** (`internal/service/auth/session.go`)
  - 用户会话状态跟踪
  - 会话超时和清理
  - 多设备登录控制

### 3.2 Agent 管理模块

#### 功能描述
- Agent 节点注册和发现
- Agent 状态监控和健康检查
- Agent 配置推送和管理
- Agent 负载监控和调度

#### 核心服务
- **Agent 节点管理** (`internal/service/agent/manager.go`)
  - Agent 注册、注销、更新
  - Agent 分组和标签管理
  - Agent 版本管理和升级

- **状态监控服务** (`internal/service/agent/monitor.go`)
  - 实时健康状态检查
  - 性能指标收集
  - 异常状态告警

- **配置推送服务** (`internal/service/agent/config_push.go`)
  - 配置文件统一管理
  - 配置变更推送
  - 配置版本控制

- **负载监控服务** (`internal/service/agent/load_monitor.go`)
  - CPU、内存、网络使用率监控
  - 任务执行负载统计
  - 负载均衡决策支持

### 3.3 任务管理模块

#### 功能描述
- 扫描任务创建和配置
- 智能任务调度和分发
- 任务执行监控和控制
- 扫描结果收集和处理

#### 核心服务
- **任务调度服务** (`internal/service/task/scheduler.go`)
  - 任务优先级管理
  - 基于负载的智能调度
  - 任务依赖关系处理
  - 定时任务和周期任务

- **任务分发服务** (`internal/service/task/dispatcher.go`)
  - 任务分解和分片
  - Agent 选择和分配
  - 任务状态同步
  - 失败重试机制

- **任务监控服务** (`internal/service/task/monitor.go`)
  - 实时任务状态跟踪
  - 任务执行进度监控
  - 异常任务检测和处理
  - 任务性能统计

- **结果处理服务** (`internal/service/task/result.go`)
  - 扫描结果收集和聚合
  - 结果数据清洗和标准化
  - 漏洞数据去重和关联
  - 结果存储和索引

### 3.4 资产管理模块

#### 功能描述
- 资产信息统一管理
- 资产发现和同步
- 资产清单维护
- 资产变更跟踪

#### 核心服务
- **资产同步服务** (`internal/service/asset/sync.go`)
  - 外部资产系统对接
  - 资产信息同步和更新
  - 数据一致性保证

- **资产发现服务** (`internal/service/asset/discovery.go`)
  - 网络资产自动发现
  - 服务和端口识别
  - 资产指纹识别

- **资产清单管理** (`internal/service/asset/inventory.go`)
  - 资产分类和标签
  - 资产生命周期管理
  - 资产关系图谱

### 3.5 监控预警模块

#### 功能描述
- 漏洞监控和威胁情报
- 实时告警和通知
- 安全态势感知
- 监控面板和仪表板

#### 核心服务
- **漏洞监控服务** (`internal/service/monitor/vuln_monitor.go`)
  - 新漏洞监控和推送
  - 漏洞影响面分析
  - 漏洞优先级评估

- **GitHub 爬虫服务** (`internal/service/monitor/github_crawler.go`)
  - GitHub 漏洞信息爬取
  - PoC 代码收集和分析
  - 威胁情报更新

- **告警服务** (`internal/service/monitor/alert.go`)
  - 多维度告警规则
  - 告警级别和优先级
  - 告警收敛和去重

### 3.6 插件管理模块

#### 功能描述
- 扫描插件统一管理
- 插件安装和更新
- 插件执行和控制
- 插件安全和隔离

#### 核心服务
- **插件管理器** (`internal/service/plugin/manager.go`)
  - 插件注册和发现
  - 插件版本管理
  - 插件依赖处理

- **插件执行器** (`internal/service/plugin/executor.go`)
  - 插件运行环境管理
  - 插件执行监控
  - 执行结果处理

- **插件安装器** (`internal/service/plugin/installer.go`)
  - 插件包下载和验证
  - 插件安装和卸载
  - 插件更新管理

- **插件安全控制** (`internal/service/plugin/security.go`)
  - 插件权限控制
  - 沙箱隔离机制
  - 恶意插件检测

### 3.7 通知服务模块

#### 功能描述
- 多渠道消息通知
- 通知模板管理
- 通知规则配置
- 通知历史记录

#### 核心服务
- **蓝信通知** (`internal/service/notification/lanxin.go`)
- **SEC 通知** (`internal/service/notification/sec.go`)
- **邮件通知** (`internal/service/notification/email.go`)
- **Webhook 通知** (`internal/service/notification/webhook.go`)

### 3.8 报告管理模块

#### 功能描述
- 多类型报告生成
- 报告模板管理
- 报告导出和分发
- 报告数据分析

#### 核心服务
- **报告生成器** (`internal/service/report/generator.go`)
  - 多格式报告生成 (PDF, Word, Excel, HTML)
  - 报告数据聚合和分析
  - 图表和可视化生成

- **报告模板管理** (`internal/service/report/template.go`)
  - 模板设计和编辑
  - 模板版本控制
  - 自定义模板支持

- **报告导出服务** (`internal/service/report/export.go`)
  - 多种格式导出
  - 报告加密和签名
  - 报告分发和推送

## 4. API 接口设计

### 4.1 认证接口

#### 用户登录
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password",
  "captcha": "1234"
}

Response:
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 3600,
    "user_info": {
      "id": 1,
      "username": "admin",
      "role": "administrator",
      "permissions": ["user:read", "task:create"]
    }
  }
}
```

#### 用户登出
```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "登出成功"
}
```

#### 刷新Token
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}

Response:
{
  "code": 200,
  "message": "Token刷新成功",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 3600
  }
}
```

### 4.2 Agent 管理接口

#### Agent 注册
```http
POST /api/v1/agents/register
Content-Type: application/json

{
  "agent_id": "agent-001",
  "hostname": "scan-node-01",
  "ip_address": "192.168.1.100",
  "port": 8080,
  "version": "1.0.0",
  "capabilities": ["port_scan", "vuln_scan", "web_scan"],
  "tags": ["production", "internal"]
}

Response:
{
  "code": 200,
  "message": "Agent注册成功",
  "data": {
    "agent_id": "agent-001",
    "status": "active",
    "registered_at": "2025-01-01T10:00:00Z"
  }
}
```

#### Agent 状态查询
```http
GET /api/v1/agents/{agent_id}/status
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "agent_id": "agent-001",
    "status": "online",
    "last_heartbeat": "2025-01-01T10:30:00Z",
    "system_info": {
      "cpu_usage": 25.5,
      "memory_usage": 60.2,
      "disk_usage": 45.8,
      "network_io": {
        "bytes_sent": 1024000,
        "bytes_recv": 2048000
      }
    },
    "running_tasks": 3,
    "completed_tasks": 156
  }
}
```

#### Agent 配置管理
```http
GET /api/v1/agents/{agent_id}/config
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "agent_id": "agent-001",
    "config": {
      "scan_timeout": 3600,
      "max_concurrent_tasks": 5,
      "log_level": "info",
      "plugins": {
        "nmap": {
          "enabled": true,
          "timeout": 300
        },
        "nuclei": {
          "enabled": true,
          "templates_path": "/opt/nuclei-templates"
        }
      }
    },
    "version": "1.2.3",
    "updated_at": "2025-01-01T09:00:00Z"
  }
}

PUT /api/v1/agents/{agent_id}/config
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "config": {
    "scan_timeout": 7200,
    "max_concurrent_tasks": 8,
    "log_level": "debug"
  }
}

Response:
{
  "code": 200,
  "message": "配置更新成功"
}
```

#### Agent 远程控制
```http
POST /api/v1/agents/{agent_id}/control
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "action": "restart",
  "params": {
    "force": false,
    "delay": 30
  }
}

Response:
{
  "code": 200,
  "message": "控制命令已发送",
  "data": {
    "command_id": "cmd-12345",
    "status": "pending"
  }
}
```

### 4.3 任务管理接口

#### 创建扫描任务
```http
POST /api/v1/tasks
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "name": "Web应用安全扫描",
  "description": "对目标Web应用进行全面安全扫描",
  "type": "web_scan",
  "targets": [
    {
      "type": "url",
      "value": "https://example.com"
    },
    {
      "type": "ip_range",
      "value": "192.168.1.1-192.168.1.100"
    }
  ],
  "scan_config": {
    "plugins": ["nuclei", "sqlmap", "xss_scanner"],
    "intensity": "medium",
    "timeout": 3600,
    "concurrent_limit": 10
  },
  "schedule": {
    "type": "immediate",
    "start_time": "2025-01-01T14:00:00Z"
  },
  "notification": {
    "enabled": true,
    "channels": ["email", "webhook"],
    "recipients": ["admin@example.com"]
  }
}

Response:
{
  "code": 200,
  "message": "任务创建成功",
  "data": {
    "task_id": "task-67890",
    "status": "pending",
    "created_at": "2025-01-01T13:30:00Z",
    "estimated_duration": 1800
  }
}
```

#### 任务列表查询
```http
GET /api/v1/tasks?page=1&size=20&status=running&type=web_scan
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "total": 156,
    "page": 1,
    "size": 20,
    "tasks": [
      {
        "task_id": "task-67890",
        "name": "Web应用安全扫描",
        "type": "web_scan",
        "status": "running",
        "progress": 45.5,
        "created_at": "2025-01-01T13:30:00Z",
        "started_at": "2025-01-01T14:00:00Z",
        "estimated_completion": "2025-01-01T14:30:00Z",
        "assigned_agents": ["agent-001", "agent-002"],
        "target_count": 50,
        "completed_targets": 23
      }
    ]
  }
}
```

#### 任务状态查询
```http
GET /api/v1/tasks/{task_id}/status
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "task_id": "task-67890",
    "status": "running",
    "progress": 65.8,
    "started_at": "2025-01-01T14:00:00Z",
    "estimated_completion": "2025-01-01T14:25:00Z",
    "statistics": {
      "total_targets": 50,
      "completed_targets": 33,
      "failed_targets": 2,
      "vulnerabilities_found": 15,
      "high_risk_vulns": 3,
      "medium_risk_vulns": 8,
      "low_risk_vulns": 4
    },
    "agent_status": [
      {
        "agent_id": "agent-001",
        "status": "running",
        "assigned_targets": 25,
        "completed_targets": 18
      },
      {
        "agent_id": "agent-002",
        "status": "running",
        "assigned_targets": 25,
        "completed_targets": 15
      }
    ]
  }
}
```

#### 任务控制
```http
POST /api/v1/tasks/{task_id}/control
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "action": "pause",
  "reason": "系统维护"
}

Response:
{
  "code": 200,
  "message": "任务已暂停",
  "data": {
    "task_id": "task-67890",
    "status": "paused",
    "action_time": "2025-01-01T14:15:00Z"
  }
}
```

### 4.4 资产管理接口

#### 资产导入
```http
POST /api/v1/assets/import
Content-Type: multipart/form-data
Authorization: Bearer <access_token>

# 文件上传或JSON数据
{
  "source": "manual",
  "format": "csv",
  "assets": [
    {
      "ip": "192.168.1.100",
      "hostname": "web-server-01",
      "type": "server",
      "os": "Linux",
      "services": ["http", "https", "ssh"],
      "tags": ["production", "web"]
    }
  ]
}

Response:
{
  "code": 200,
  "message": "资产导入成功",
  "data": {
    "import_id": "import-12345",
    "total_count": 100,
    "success_count": 95,
    "failed_count": 5,
    "failed_items": [
      {
        "line": 10,
        "error": "IP地址格式错误"
      }
    ]
  }
}
```

#### 资产导出
```http
GET /api/v1/assets/export?format=csv&filter=type:server
Authorization: Bearer <access_token>

Response:
# CSV文件下载或JSON数据
{
  "code": 200,
  "message": "导出成功",
  "data": {
    "download_url": "/downloads/assets_20250101.csv",
    "expires_at": "2025-01-01T18:00:00Z",
    "total_count": 500
  }
}
```

#### 资产同步
```http
POST /api/v1/assets/sync
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "source_type": "cmdb",
  "source_config": {
    "api_url": "https://cmdb.example.com/api",
    "api_key": "your-api-key",
    "sync_interval": 3600
  },
  "mapping_rules": {
    "ip_field": "management_ip",
    "hostname_field": "device_name",
    "type_field": "device_type"
  }
}

Response:
{
  "code": 200,
  "message": "同步任务已启动",
  "data": {
    "sync_id": "sync-54321",
    "status": "running",
    "started_at": "2025-01-01T15:00:00Z"
  }
}
```

#### 资产列表
```http
GET /api/v1/assets?page=1&size=20&type=server&tag=production
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "total": 500,
    "page": 1,
    "size": 20,
    "assets": [
      {
        "id": "asset-001",
        "ip": "192.168.1.100",
        "hostname": "web-server-01",
        "type": "server",
        "os": "Linux Ubuntu 20.04",
        "services": [
          {
            "port": 80,
            "protocol": "tcp",
            "service": "http",
            "version": "Apache 2.4.41"
          },
          {
            "port": 443,
            "protocol": "tcp",
            "service": "https",
            "version": "Apache 2.4.41"
          }
        ],
        "vulnerabilities": {
          "high": 2,
          "medium": 5,
          "low": 8
        },
        "last_scan": "2025-01-01T12:00:00Z",
        "tags": ["production", "web"],
        "created_at": "2024-12-01T10:00:00Z",
        "updated_at": "2025-01-01T12:00:00Z"
      }
    ]
  }
}
```

### 4.5 监控预警接口

#### 漏洞监控任务
```http
POST /api/v1/monitor/vulnerabilities
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "name": "高危漏洞监控",
  "description": "监控新发布的高危漏洞",
  "rules": {
    "severity": ["high", "critical"],
    "keywords": ["RCE", "SQL注入", "XSS"],
    "sources": ["nvd", "github", "exploit-db"],
    "asset_filter": {
      "tags": ["production"],
      "types": ["web", "server"]
    }
  },
  "notification": {
    "enabled": true,
    "channels": ["email", "lanxin"],
    "recipients": ["security@example.com"]
  },
  "schedule": {
    "interval": 3600,
    "enabled": true
  }
}

Response:
{
  "code": 200,
  "message": "监控任务创建成功",
  "data": {
    "monitor_id": "monitor-001",
    "status": "active",
    "created_at": "2025-01-01T16:00:00Z"
  }
}
```

#### 预警列表
```http
GET /api/v1/monitor/alerts?page=1&size=20&severity=high&status=unread
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "total": 25,
    "page": 1,
    "size": 20,
    "alerts": [
      {
        "alert_id": "alert-001",
        "title": "发现新的Apache Log4j RCE漏洞",
        "severity": "critical",
        "type": "vulnerability",
        "description": "Apache Log4j 2.x版本存在远程代码执行漏洞",
        "cve_id": "CVE-2021-44228",
        "cvss_score": 9.8,
        "affected_assets": 15,
        "status": "unread",
        "source": "nvd",
        "created_at": "2025-01-01T16:30:00Z",
        "details": {
          "poc_available": true,
          "exploit_public": true,
          "patch_available": true
        }
      }
    ]
  }
}
```

#### 监控面板
```http
GET /api/v1/monitor/dashboard
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "summary": {
      "total_assets": 500,
      "online_agents": 8,
      "running_tasks": 12,
      "active_alerts": 25
    },
    "vulnerability_stats": {
      "critical": 5,
      "high": 18,
      "medium": 45,
      "low": 120,
      "total": 188
    },
    "recent_scans": [
      {
        "task_id": "task-001",
        "name": "Web应用扫描",
        "status": "completed",
        "vulnerabilities_found": 12,
        "completed_at": "2025-01-01T15:30:00Z"
      }
    ],
    "system_health": {
      "master_status": "healthy",
      "database_status": "healthy",
      "queue_status": "healthy",
      "storage_usage": 65.5
    }
  }
}
```

### 4.6 插件管理接口

#### 插件列表
```http
GET /api/v1/plugins?category=scanner&status=enabled
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "total": 50,
    "plugins": [
      {
        "plugin_id": "nmap",
        "name": "Nmap端口扫描器",
        "version": "7.94",
        "category": "scanner",
        "type": "port_scan",
        "status": "enabled",
        "description": "网络发现和安全审计工具",
        "author": "Nmap Project",
        "capabilities": ["port_scan", "service_detection", "os_detection"],
        "config_schema": {
          "timeout": {
            "type": "integer",
            "default": 300,
            "description": "扫描超时时间(秒)"
          },
          "scan_type": {
            "type": "string",
            "enum": ["tcp", "udp", "syn"],
            "default": "tcp",
            "description": "扫描类型"
          }
        },
        "installed_at": "2024-12-01T10:00:00Z",
        "last_used": "2025-01-01T14:30:00Z"
      }
    ]
  }
}
```

#### 插件远程控制
```http
POST /api/v1/plugins/{plugin_id}/control
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "action": "enable",
  "target_agents": ["agent-001", "agent-002"],
  "config": {
    "timeout": 600,
    "scan_type": "syn"
  }
}

Response:
{
  "code": 200,
  "message": "插件控制命令已发送",
  "data": {
    "command_id": "cmd-plugin-001",
    "affected_agents": 2,
    "status": "pending"
  }
}
```

#### 插件安装
```http
POST /api/v1/plugins/install
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "plugin_package": "nuclei-v2.9.15.zip",
  "source": "official",
  "target_agents": ["agent-001", "agent-002"],
  "auto_enable": true,
  "config": {
    "templates_path": "/opt/nuclei-templates",
    "update_templates": true
  }
}

Response:
{
  "code": 200,
  "message": "插件安装任务已创建",
  "data": {
    "install_id": "install-001",
    "status": "pending",
    "target_agents": 2,
    "estimated_duration": 300
  }
}
```

#### 插件执行
```http
POST /api/v1/plugins/{plugin_id}/execute
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "targets": ["192.168.1.100", "https://example.com"],
  "config": {
    "timeout": 300,
    "threads": 10,
    "output_format": "json"
  },
  "agent_id": "agent-001"
}

Response:
{
  "code": 200,
  "message": "插件执行任务已创建",
  "data": {
    "execution_id": "exec-001",
    "status": "running",
    "started_at": "2025-01-01T17:00:00Z",
    "estimated_completion": "2025-01-01T17:05:00Z"
  }
}
```

### 4.7 报告管理接口

#### 报告生成
```http
POST /api/v1/reports/generate
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "name": "月度安全扫描报告",
  "type": "vulnerability_report",
  "template_id": "template-001",
  "data_source": {
    "type": "task_results",
    "task_ids": ["task-001", "task-002", "task-003"],
    "date_range": {
      "start": "2024-12-01T00:00:00Z",
      "end": "2024-12-31T23:59:59Z"
    }
  },
  "format": "pdf",
  "options": {
    "include_charts": true,
    "include_raw_data": false,
    "language": "zh-CN"
  },
  "recipients": ["manager@example.com"]
}

Response:
{
  "code": 200,
  "message": "报告生成任务已创建",
  "data": {
    "report_id": "report-001",
    "status": "generating",
    "estimated_completion": "2025-01-01T17:30:00Z"
  }
}
```

#### 报告导出
```http
GET /api/v1/reports/{report_id}/export?format=pdf
Authorization: Bearer <access_token>

Response:
# PDF文件下载或重定向到下载链接
{
  "code": 200,
  "message": "报告导出成功",
  "data": {
    "download_url": "/downloads/report_001.pdf",
    "expires_at": "2025-01-02T17:00:00Z",
    "file_size": 2048576
  }
}
```

#### 报告模板
```http
GET /api/v1/reports/templates
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "templates": [
      {
        "template_id": "template-001",
        "name": "标准漏洞报告模板",
        "type": "vulnerability_report",
        "description": "包含漏洞详情、风险评估和修复建议",
        "version": "1.2.0",
        "supported_formats": ["pdf", "word", "html"],
        "created_at": "2024-11-01T10:00:00Z",
        "updated_at": "2024-12-15T14:30:00Z"
      }
    ]
  }
}

POST /api/v1/reports/templates
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "name": "自定义扫描报告",
  "type": "custom_report",
  "description": "自定义格式的扫描报告模板",
  "template_content": "<html>...</html>",
  "supported_formats": ["html", "pdf"]
}

Response:
{
  "code": 200,
  "message": "模板创建成功",
  "data": {
    "template_id": "template-002",
    "version": "1.0.0"
  }
}
```

### 4.8 系统管理接口

#### 用户管理
```http
GET /api/v1/system/users?page=1&size=20&role=operator
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "total": 15,
    "page": 1,
    "size": 20,
    "users": [
      {
        "user_id": 1,
        "username": "operator01",
        "email": "operator01@example.com",
        "role": "operator",
        "status": "active",
        "last_login": "2025-01-01T16:45:00Z",
        "created_at": "2024-11-01T10:00:00Z",
        "permissions": ["task:read", "task:create", "asset:read"]
      }
    ]
  }
}

POST /api/v1/system/users
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "SecurePassword123!",
  "role": "viewer",
  "permissions": ["task:read", "asset:read"]
}

Response:
{
  "code": 200,
  "message": "用户创建成功",
  "data": {
    "user_id": 16,
    "username": "newuser",
    "status": "active"
  }
}
```

#### 角色管理
```http
GET /api/v1/system/roles
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "roles": [
      {
        "role_id": 1,
        "name": "administrator",
        "display_name": "系统管理员",
        "description": "拥有系统所有权限",
        "permissions": ["*:*"],
        "user_count": 2,
        "created_at": "2024-10-01T10:00:00Z"
      },
      {
        "role_id": 2,
        "name": "operator",
        "display_name": "操作员",
        "description": "可以创建和管理扫描任务",
        "permissions": [
          "task:*",
          "asset:read",
          "agent:read",
          "report:read"
        ],
        "user_count": 8,
        "created_at": "2024-10-01T10:00:00Z"
      }
    ]
  }
}
```

#### 系统配置
```http
GET /api/v1/system/config
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "system": {
      "name": "NeoScan安全扫描平台",
      "version": "1.0.0",
      "timezone": "Asia/Shanghai",
      "language": "zh-CN"
    },
    "security": {
      "session_timeout": 3600,
      "password_policy": {
        "min_length": 8,
        "require_uppercase": true,
        "require_lowercase": true,
        "require_numbers": true,
        "require_symbols": true
      },
      "login_attempts": {
        "max_attempts": 5,
        "lockout_duration": 1800
      }
    },
    "notification": {
      "email": {
        "enabled": true,
        "smtp_server": "smtp.example.com",
        "smtp_port": 587,
        "use_tls": true
      },
      "webhook": {
        "enabled": true,
        "timeout": 30
      }
    }
  }
}

PUT /api/v1/system/config
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "security": {
    "session_timeout": 7200,
    "password_policy": {
      "min_length": 10
    }
  }
}

Response:
{
  "code": 200,
  "message": "配置更新成功"
}
```

#### 审计日志
```http
GET /api/v1/system/audit?page=1&size=50&action=login&start_time=2025-01-01T00:00:00Z
Authorization: Bearer <access_token>

Response:
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "total": 1250,
    "page": 1,
    "size": 50,
    "logs": [
      {
        "log_id": "audit-001",
        "timestamp": "2025-01-01T16:45:30Z",
        "user_id": 1,
        "username": "admin",
        "action": "login",
        "resource": "system",
        "ip_address": "192.168.1.50",
        "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "status": "success",
        "details": {
          "login_method": "password",
          "session_id": "sess-12345"
        }
      },
      {
        "log_id": "audit-002",
        "timestamp": "2025-01-01T16:50:15Z",
        "user_id": 1,
        "username": "admin",
        "action": "create",
        "resource": "task",
        "resource_id": "task-67890",
        "ip_address": "192.168.1.50",
        "status": "success",
        "details": {
          "task_name": "Web应用安全扫描",
          "task_type": "web_scan",
          "target_count": 50
        }
      }
    ]
  }
}
```

## 5. gRPC 服务接口

### 5.1 Agent 通信服务

```protobuf
// Agent通信服务定义
service AgentService {
  // Agent注册
  rpc RegisterAgent(RegisterAgentRequest) returns (RegisterAgentResponse);
  
  // 心跳检测
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  
  // 获取任务
  rpc GetTasks(GetTasksRequest) returns (GetTasksResponse);
  
  // 上报任务状态
  rpc ReportTaskStatus(ReportTaskStatusRequest) returns (ReportTaskStatusResponse);
  
  // 上报扫描结果
  rpc ReportScanResult(ReportScanResultRequest) returns (ReportScanResultResponse);
  
  // 接收配置更新
  rpc ReceiveConfigUpdate(ReceiveConfigUpdateRequest) returns (ReceiveConfigUpdateResponse);
}

// Agent注册请求
message RegisterAgentRequest {
  string agent_id = 1;
  string hostname = 2;
  string ip_address = 3;
  int32 port = 4;
  string version = 5;
  repeated string capabilities = 6;
  repeated string tags = 7;
  SystemInfo system_info = 8;
}

// 系统信息
message SystemInfo {
  string os = 1;
  string arch = 2;
  int32 cpu_cores = 3;
  int64 memory_total = 4;
  int64 disk_total = 5;
}

// Agent注册响应
message RegisterAgentResponse {
  bool success = 1;
  string message = 2;
  AgentConfig config = 3;
}

// Agent配置
message AgentConfig {
  int32 heartbeat_interval = 1;
  int32 task_poll_interval = 2;
  int32 max_concurrent_tasks = 3;
  map<string, string> plugin_config = 4;
}

// 心跳请求
message HeartbeatRequest {
  string agent_id = 1;
  AgentStatus status = 2;
  repeated TaskStatus running_tasks = 3;
}

// Agent状态
message AgentStatus {
  double cpu_usage = 1;
  double memory_usage = 2;
  double disk_usage = 3;
  int64 network_bytes_sent = 4;
  int64 network_bytes_recv = 5;
  int32 active_connections = 6;
}

// 任务状态
message TaskStatus {
  string task_id = 1;
  string status = 2;  // running, completed, failed, paused
  double progress = 3;
  string error_message = 4;
  int64 started_at = 5;
  int64 updated_at = 6;
}

// 心跳响应
message HeartbeatResponse {
  bool success = 1;
  repeated string commands = 2;  // 待执行的命令
  AgentConfig updated_config = 3;
}
```

### 5.2 任务管理服务

```protobuf
// 任务管理服务定义
service TaskService {
  // 分发任务
  rpc DistributeTask(DistributeTaskRequest) returns (DistributeTaskResponse);
  
  // 取消任务
  rpc CancelTask(CancelTaskRequest) returns (CancelTaskResponse);
  
  // 暂停任务
  rpc PauseTask(PauseTaskRequest) returns (PauseTaskResponse);
  
  // 恢复任务
  rpc ResumeTask(ResumeTaskRequest) returns (ResumeTaskResponse);
}

// 分发任务请求
message DistributeTaskRequest {
  string task_id = 1;
  string agent_id = 2;
  TaskConfig config = 3;
  repeated Target targets = 4;
}

// 任务配置
message TaskConfig {
  string task_type = 1;  // port_scan, vuln_scan, web_scan
  repeated string plugins = 2;
  map<string, string> plugin_config = 3;
  int32 timeout = 4;
  int32 concurrent_limit = 5;
  string output_format = 6;
}

// 扫描目标
message Target {
  string type = 1;  // ip, url, domain, ip_range
  string value = 2;
  map<string, string> metadata = 3;
}

// 分发任务响应
message DistributeTaskResponse {
  bool success = 1;
  string message = 2;
  string execution_id = 3;
}
```

### 5.3 配置管理服务

```protobuf
// 配置管理服务定义
service ConfigService {
  // 推送配置
  rpc PushConfig(PushConfigRequest) returns (PushConfigResponse);
  
  // 获取配置
  rpc GetConfig(GetConfigRequest) returns (GetConfigResponse);
  
  // 验证配置
  rpc ValidateConfig(ValidateConfigRequest) returns (ValidateConfigResponse);
}

// 推送配置请求
message PushConfigRequest {
  string agent_id = 1;
  string config_version = 2;
  map<string, string> config_data = 3;
  bool force_update = 4;
}

// 推送配置响应
message PushConfigResponse {
  bool success = 1;
  string message = 2;
  string applied_version = 3;
}
```

### 5.4 插件控制服务

```protobuf
// 插件控制服务定义
service PluginService {
  // 安装插件
  rpc InstallPlugin(InstallPluginRequest) returns (InstallPluginResponse);
  
  // 卸载插件
  rpc UninstallPlugin(UninstallPluginRequest) returns (UninstallPluginResponse);
  
  // 启用插件
  rpc EnablePlugin(EnablePluginRequest) returns (EnablePluginResponse);
  
  // 禁用插件
  rpc DisablePlugin(DisablePluginRequest) returns (DisablePluginResponse);
  
  // 更新插件
  rpc UpdatePlugin(UpdatePluginRequest) returns (UpdatePluginResponse);
}

// 安装插件请求
message InstallPluginRequest {
  string agent_id = 1;
  string plugin_id = 2;
  string plugin_version = 3;
  string download_url = 4;
  string checksum = 5;
  map<string, string> install_config = 6;
}

// 安装插件响应
message InstallPluginResponse {
  bool success = 1;
  string message = 2;
  string installed_version = 3;
}
```

## 6. WebSocket 实时通信

### 6.1 连接管理

```go
// WebSocket连接管理器
type Hub struct {
    clients    map[*Client]bool     // 已连接的客户端
    broadcast  chan []byte          // 广播消息通道
    register   chan *Client         // 客户端注册通道
    unregister chan *Client         // 客户端注销通道
}

// 客户端连接
type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte
    userID   string
    channels []string  // 订阅的频道
}
```

### 6.2 消息类型

```json
// 任务状态更新
{
  "type": "task_status_update",
  "data": {
    "task_id": "task-67890",
    "status": "running",
    "progress": 75.5,
    "agent_id": "agent-001",
    "updated_at": "2025-01-01T17:15:00Z"
  }
}

// Agent状态变更
{
  "type": "agent_status_change",
  "data": {
    "agent_id": "agent-001",
    "status": "offline",
    "last_seen": "2025-01-01T17:10:00Z",
    "reason": "connection_timeout"
  }
}

// 新漏洞告警
{
  "type": "vulnerability_alert",
  "data": {
    "alert_id": "alert-001",
    "severity": "critical",
    "title": "发现新的RCE漏洞",
    "affected_assets": 15,
    "cve_id": "CVE-2025-0001"
  }
}

// 系统通知
{
  "type": "system_notification",
  "data": {
    "level": "info",
    "title": "系统维护通知",
    "message": "系统将于今晚22:00进行维护，预计持续2小时",
    "timestamp": "2025-01-01T17:20:00Z"
  }
}
```

## 7. 数据模型设计

### 7.1 Agent 模型

```go
// Agent节点模型
type Agent struct {
    ID           string    `json:"id" gorm:"primaryKey"`
    Hostname     string    `json:"hostname" gorm:"not null"`
    IPAddress    string    `json:"ip_address" gorm:"not null"`
    Port         int       `json:"port" gorm:"default:8080"`
    Version      string    `json:"version"`
    Status       string    `json:"status" gorm:"default:offline"` // online, offline, error
    Capabilities []string  `json:"capabilities" gorm:"type:json"`
    Tags         []string  `json:"tags" gorm:"type:json"`
    SystemInfo   SystemInfo `json:"system_info" gorm:"embedded"`
    LastHeartbeat time.Time `json:"last_heartbeat"`
    RegisteredAt time.Time `json:"registered_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// 系统信息
type SystemInfo struct {
    OS          string  `json:"os"`
    Arch        string  `json:"arch"`
    CPUCores    int     `json:"cpu_cores"`
    MemoryTotal int64   `json:"memory_total"`
    DiskTotal   int64   `json:"disk_total"`
    CPUUsage    float64 `json:"cpu_usage"`
    MemoryUsage float64 `json:"memory_usage"`
    DiskUsage   float64 `json:"disk_usage"`
}
```

### 7.2 任务模型

```go
// 扫描任务模型
type Task struct {
    ID              string    `json:"id" gorm:"primaryKey"`
    Name            string    `json:"name" gorm:"not null"`
    Description     string    `json:"description"`
    Type            string    `json:"type" gorm:"not null"` // port_scan, vuln_scan, web_scan
    Status          string    `json:"status" gorm:"default:pending"` // pending, running, completed, failed, paused
    Priority        int       `json:"priority" gorm:"default:5"`
    Progress        float64   `json:"progress" gorm:"default:0"`
    CreatedBy       string    `json:"created_by"`
    AssignedAgents  []string  `json:"assigned_agents" gorm:"type:json"`
    Targets         []Target  `json:"targets" gorm:"type:json"`
    ScanConfig      ScanConfig `json:"scan_config" gorm:"embedded"`
    Schedule        Schedule  `json:"schedule" gorm:"embedded"`
    Notification    NotificationConfig `json:"notification" gorm:"embedded"`
    CreatedAt       time.Time `json:"created_at"`
    StartedAt       *time.Time `json:"started_at"`
    CompletedAt     *time.Time `json:"completed_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

// 扫描目标
type Target struct {
    Type     string            `json:"type"` // ip, url, domain, ip_range
    Value    string            `json:"value"`
    Metadata map[string]string `json:"metadata"`
}

// 扫描配置
type ScanConfig struct {
    Plugins         []string          `json:"plugins"`
    PluginConfig    map[string]string `json:"plugin_config"`
    Intensity       string            `json:"intensity"` // low, medium, high
    Timeout         int               `json:"timeout"`
    ConcurrentLimit int               `json:"concurrent_limit"`
    OutputFormat    string            `json:"output_format"`
}

// 调度配置
type Schedule struct {
    Type      string     `json:"type"` // immediate, scheduled, recurring
    StartTime *time.Time `json:"start_time"`
    CronExpr  string     `json:"cron_expr"`
    Enabled   bool       `json:"enabled"`
}

// 通知配置
type NotificationConfig struct {
    Enabled    bool     `json:"enabled"`
    Channels   []string `json:"channels"`
    Recipients []string `json:"recipients"`
}
```

### 7.3 资产模型【后续优化】

```go
// 资产模型
type Asset struct {
    ID             string            `json:"id" gorm:"primaryKey"`
    IP             string            `json:"ip" gorm:"index"`
    Hostname       string            `json:"hostname"`
    Type           string            `json:"type"` // server, workstation, network_device, web_app
    OS             string            `json:"os"`
    OSVersion      string            `json:"os_version"`
    Services       []Service         `json:"services" gorm:"type:json"`
    Vulnerabilities VulnSummary      `json:"vulnerabilities" gorm:"embedded"`
    Tags           []string          `json:"tags" gorm:"type:json"`
    Metadata       map[string]string `json:"metadata" gorm:"type:json"`
    LastScan       *time.Time        `json:"last_scan"`
    CreatedAt      time.Time         `json:"created_at"`
    UpdatedAt      time.Time         `json:"updated_at"`
}

// 服务信息
type Service struct {
    Port     int    `json:"port"`
    Protocol string `json:"protocol"`
    Service  string `json:"service"`
    Version  string `json:"version"`
    Banner   string `json:"banner"`
}

// 漏洞统计
type VulnSummary struct {
    Critical int `json:"critical"`
    High     int `json:"high"`
    Medium   int `json:"medium"`
    Low      int `json:"low"`
    Total    int `json:"total"`
}
```

### 7.4 漏洞模型【后续优化】

```go
// 漏洞模型
type Vulnerability struct {
    ID          string            `json:"id" gorm:"primaryKey"`
    TaskID      string            `json:"task_id" gorm:"index"`
    AssetID     string            `json:"asset_id" gorm:"index"`
    PluginID    string            `json:"plugin_id"`
    Name        string            `json:"name" gorm:"not null"`
    Description string            `json:"description"`
    Severity    string            `json:"severity"` // critical, high, medium, low, info
    CVEID       string            `json:"cve_id" gorm:"index"`
    CVSSScore   float64           `json:"cvss_score"`
    CVSSVector  string            `json:"cvss_vector"`
    Category    string            `json:"category"`
    Target      string            `json:"target"`
    Port        int               `json:"port"`
    Protocol    string            `json:"protocol"`
    Evidence    string            `json:"evidence"`
    Solution    string            `json:"solution"`
    References  []string          `json:"references" gorm:"type:json"`
    Tags        []string          `json:"tags" gorm:"type:json"`
    Status      string            `json:"status" gorm:"default:open"` // open, fixed, false_positive, accepted
    RiskScore   float64           `json:"risk_score"`
    Metadata    map[string]string `json:"metadata" gorm:"type:json"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}
```

### 7.5 用户和权限模型

```go
// 用户模型
type User struct {
    ID          uint      `json:"id" gorm:"primaryKey"`
    Username    string    `json:"username" gorm:"uniqueIndex;not null"`
    Email       string    `json:"email" gorm:"uniqueIndex"`
    Password    string    `json:"-" gorm:"not null"`
    Role        string    `json:"role" gorm:"not null"`
    Status      string    `json:"status" gorm:"default:active"` // active, inactive, locked
    Permissions []string  `json:"permissions" gorm:"type:json"`
    LastLogin   *time.Time `json:"last_login"`
    LoginCount  int       `json:"login_count" gorm:"default:0"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// 角色模型
type Role struct {
    ID          uint     `json:"id" gorm:"primaryKey"`
    Name        string   `json:"name" gorm:"uniqueIndex;not null"`
    DisplayName string   `json:"display_name"`
    Description string   `json:"description"`
    Permissions []string `json:"permissions" gorm:"type:json"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// 权限模型
type Permission struct {
    ID          uint   `json:"id" gorm:"primaryKey"`
    Resource    string `json:"resource" gorm:"not null"` // task, agent, asset, user, system
    Action      string `json:"action" gorm:"not null"`   // create, read, update, delete, execute
    Description string `json:"description"`
}
```

## 8. 安全设计

### 8.1 身份认证

- **多因素认证**：支持密码、验证码、LDAP等多种认证方式
- **JWT Token**：使用 RS256 算法签名，支持 Token 刷新和黑名单
- **会话管理**：会话超时控制，异地登录检测
- **密码策略**：强密码要求，密码历史记录，定期更换

### 8.2 访问控制

- **RBAC 模型**：基于角色的访问控制，支持细粒度权限
- **API 鉴权**：所有 API 接口都需要有效的 Token 和权限验证
- **资源隔离**：不同用户只能访问授权的资源
- **操作审计**：记录所有用户操作和系统事件

### 8.3 数据安全

- **数据加密**：敏感数据存储加密，传输过程 TLS 加密
- **数据脱敏**：日志和报告中的敏感信息脱敏处理
- **备份安全**：数据备份加密存储，定期恢复测试
- **数据清理**：过期数据自动清理，符合数据保护法规

### 8.4 网络安全

- **网络隔离**：Master 和 Agent 之间使用专用网络通道
- **防火墙规则**：严格的入站和出站规则配置
- **DDoS 防护**：API 限流和异常流量检测
- **入侵检测**：实时监控异常访问和攻击行为

## 9. 性能优化

### 9.1 并发处理

- **Goroutine 池**：控制并发数量，避免资源耗尽
- **任务队列**：使用 RabbitMQ 实现异步任务处理
- **连接池**：数据库和 Redis 连接池优化
- **负载均衡**：智能任务分发和 Agent 负载均衡

### 9.2 缓存策略

- **Redis 缓存**：热点数据缓存，减少数据库查询
- **本地缓存**：配置信息和元数据本地缓存
- **CDN 加速**：静态资源和报告文件 CDN 分发
- **缓存预热**：系统启动时预加载常用数据

### 9.3 数据库优化

- **索引优化**：合理设计数据库索引，提高查询性能
- **分库分表**：大数据量表的水平分割
- **读写分离**：主从复制，读写分离提高并发能力
- **查询优化**：SQL 查询优化，避免全表扫描

### 9.4 监控和调优

- **性能监控**：实时监控系统性能指标
- **慢查询分析**：数据库慢查询监控和优化
- **内存管理**：Go 程序内存使用监控和优化
- **性能测试**：定期进行压力测试和性能调优

## 10. 部署和运维

### 10.1 容器化部署

```dockerfile
# Master 节点 Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o neoscan-master ./cmd/master

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/neoscan-master .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/web ./web

EXPOSE 8080 9090 9091
CMD ["./neoscan-master"]
```

### 10.2 Docker Compose 配置

```yaml
version: '3.8'

services:
  neoscan-master:
    build: .
    ports:
      - "8080:8080"   # HTTP API
      - "9090:9090"   # gRPC
      - "9091:9091"   # WebSocket
    environment:
      - DB_HOST=mysql
      - REDIS_HOST=redis
      - RABBITMQ_HOST=rabbitmq
    depends_on:
      - mysql
      - redis
      - rabbitmq
    volumes:
      - ./configs:/app/configs
      - ./logs:/app/logs
      - ./data:/app/data

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: neoscan123
      MYSQL_DATABASE: neoscan
    volumes:
      - mysql_data:/var/lib/mysql
      - ./scripts/init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "3306:3306"

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"

  rabbitmq:
    image: rabbitmq:3-management
    environment:
      RABBITMQ_DEFAULT_USER: neoscan
      RABBITMQ_DEFAULT_PASS: neoscan123
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq
    ports:
      - "5672:5672"
      - "15672:15672"

volumes:
  mysql_data:
  redis_data:
  rabbitmq_data:
```

### 10.3 Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: neoscan-master
  namespace: neoscan
spec:
  replicas: 2
  selector:
    matchLabels:
      app: neoscan-master
  template:
    metadata:
      labels:
        app: neoscan-master
    spec:
      containers:
      - name: neoscan-master
        image: neoscan/master:latest
        ports:
        - containerPort: 8080
        - containerPort: 9090
        - containerPort: 9091
        env:
        - name: DB_HOST
          value: "mysql-service"
        - name: REDIS_HOST
          value: "redis-service"
        - name: RABBITMQ_HOST
          value: "rabbitmq-service"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: neoscan-master-service
  namespace: neoscan
spec:
  selector:
    app: neoscan-master
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: grpc
    port: 9090
    targetPort: 9090
  - name: websocket
    port: 9091
    targetPort: 9091
  type: LoadBalancer
```

### 10.4 监控配置

```yaml
# Prometheus 配置
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'neoscan-master'
    static_configs:
      - targets: ['neoscan-master:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'neoscan-agents'
    consul_sd_configs:
      - server: 'consul:8500'
        services: ['neoscan-agent']
```

## 11. 故障处理和恢复

### 11.1 高可用设计

- **主备切换**：Master 节点主备部署，自动故障切换
- **数据备份**：定时数据备份，多地域存储
- **服务降级**：关键服务故障时的降级策略
- **熔断机制**：防止级联故障的熔断保护

### 11.2 故障监控

- **健康检查**：定期检查各组件健康状态
- **告警机制**：故障自动告警和通知
- **日志监控**：实时日志分析和异常检测
- **性能监控**：系统性能指标监控和预警

### 11.3 恢复策略

- **自动恢复**：服务自动重启和故障自愈
- **数据恢复**：数据损坏时的快速恢复机制
- **业务恢复**：关键业务的快速恢复流程
- **灾难恢复**：灾难情况下的完整恢复方案

## 12. 扩展性设计

### 12.1 水平扩展

- **无状态设计**：Master 节点无状态，支持水平扩展
- **负载均衡**：多个 Master 实例的负载均衡
- **数据分片**：大数据量的分片存储和处理
- **微服务架构**：模块化设计，独立扩展

### 12.2 插件扩展

- **插件接口**：标准化的插件开发接口
- **动态加载**：插件的动态安装和卸载
- **沙箱隔离**：插件运行的安全隔离
- **版本管理**：插件版本控制和兼容性管理

### 12.3 API 扩展

- **版本控制**：API 版本管理和向后兼容
- **自定义接口**：支持用户自定义 API 接口
- **第三方集成**：与外部系统的集成接口
- **Webhook 支持**：事件驱动的 Webhook 机制

## 13. 总结

NeoScan Master 节点作为整个分布式扫描系统的核心控制中心，承担着系统管理、任务调度、数据处理等关键职责。通过模块化的架构设计、完善的 API 接口、强大的安全机制和高性能的技术实现，为用户提供了一个功能完整、安全可靠、易于扩展的安全扫描管理平台。

### 13.1 核心优势

1. **统一管理**：集中管理所有 Agent 节点和扫描任务
2. **智能调度**：基于负载均衡的智能任务分发
3. **实时监控**：全方位的系统监控和告警机制
4. **安全可靠**：完善的安全机制和高可用设计
5. **易于扩展**：模块化架构支持功能和性能扩展

### 13.2 技术特色

1. **多协议支持**：HTTP、gRPC、WebSocket 等多种通信协议
2. **插件化架构**：灵活的插件管理和扩展机制
3. **容器化部署**：支持 Docker 和 Kubernetes 部署
4. **微服务设计**：松耦合的微服务架构
5. **云原生支持**：适配云原生环境的设计理念

### 13.3 应用场景

1. **企业安全**：大型企业的安全扫描和漏洞管理
2. **合规检查**：满足各种安全合规要求的扫描
3. **持续监控**：7x24 小时的安全态势监控
4. **应急响应**：安全事件的快速响应和处置
5. **风险评估**：全面的安全风险评估和报告

通过本设计文档的详细说明，开发团队可以清晰地了解 Master 节点的功能要求、技术架构和实现方案，为后续的开发工作提供了完整的技术指导。