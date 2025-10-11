# ScanConfig 模块说明文档

## 模块概述

`scan_config` 模块是 NeoScan 系统的**配置管理核心**，负责统一管理所有扫描相关的配置和策略。该模块遵循 Linus Torvalds 的"好品味"设计原则，消除了传统扫描系统中的特殊情况，用统一的配置模型解决所有扫描场景。

## 核心设计理念

### 1. "好品味"设计原则
- **消除特殊情况**：用统一的 ProjectConfig 替代"工程方案"和"非工程方案"的分离
- **数据结构优先**：先设计好数据模型，代码逻辑自然简洁
- **线性化处理**：避免复杂的条件分支，用规则引擎统一处理

### 2. 严格分层架构
```
Handler (HTTP接口) → Service (业务逻辑) → Repository (数据访问) → Database
```

### 3. 实用主义实现
- 解决扫描配置管理的实际痛点
- 向后兼容现有 Agent 和扫描执行逻辑
- 渐进式增强，可逐步迁移现有配置

## 四大核心组件

### 1. 项目配置管理 (ProjectConfig)
**文件**: `project_config_handler.go`

**核心功能**:
- 统一的扫描项目配置管理
- 目标范围管理 (target_scope)
- 扫描频率控制 (scan_frequency)
- 并发数限制 (max_concurrent)
- 超时控制 (timeout_second)
- 通知配置 (notify_emails)

**主要作用**:
- 消除"24小时不间断"和"周期性扫描"的特殊情况
- 用统一的 cron 表达式处理所有调度需求
- 提供项目级别的配置模板和复用机制

### 2. 工作流管理 (WorkflowConfig)
**文件**: `workflow_handler.go`

**核心功能**:
- 扫描工作流编排和执行控制
- 执行模式 (execution_mode: sequential/parallel)
- 调度配置 (schedule_config)
- 阶段定义 (stages: 端口扫描→服务识别→漏洞扫描)
- 全局配置 (global_config)

**主要作用**:
- 将复杂的扫描流程标准化为可配置的工作流
- 支持串行和并行执行模式
- 提供工作流模板，降低配置复杂度

### 3. 扫描工具管理 (ScanTool)
**文件**: `scan_tool_handler.go`

**核心功能**:
- 第三方扫描工具统一管理
- 工具类型 (tool_type: port_scan/vuln_scan/web_scan)
- 可执行路径 (executable_path)
- 默认参数 (default_args)
- 健康状态监控 (is_active)

**主要作用**:
- 统一管理 nmap、nuclei、masscan、gobuster 等扫描工具
- 提供工具健康检查和状态监控
- 支持工具的安装、卸载和版本管理
- 工具配置的标准化和复用

### 4. 扫描规则引擎 (ScanRule)
**文件**: `scan_rule_handler.go` + `rule_engine_handler.go`

**核心功能**:
- 智能规则系统
- 规则类型 (rule_type: filter/validation/transform/alert)
- 条件表达式 (condition: "severity >= high")
- 执行动作 (action: include/exclude/alert/timeout)
- 适用工具 (applicable_tools)

**主要作用**:
- 提供灵活的扫描策略配置（白名单、黑名单、超时控制）
- 智能结果过滤和告警机制
- 支持规则的导入导出和模板化
- 基于条件表达式的动态规则执行

## API 接口设计

### 项目配置管理 API
```
GET    /api/v1/scan-config/projects          # 获取项目配置列表
GET    /api/v1/scan-config/projects/:id      # 获取项目配置详情
POST   /api/v1/scan-config/projects          # 创建项目配置
PUT    /api/v1/scan-config/projects/:id      # 更新项目配置
DELETE /api/v1/scan-config/projects/:id      # 删除项目配置
POST   /api/v1/scan-config/projects/:id/enable   # 启用项目配置
POST   /api/v1/scan-config/projects/:id/disable  # 禁用项目配置
```

### 工作流管理 API
```
GET    /api/v1/scan-config/workflows         # 获取工作流列表
POST   /api/v1/scan-config/workflows/:id/execute  # 执行工作流
POST   /api/v1/scan-config/workflows/:id/stop     # 停止工作流
GET    /api/v1/scan-config/workflows/:id/status   # 获取工作流状态
```

### 扫描工具管理 API
```
GET    /api/v1/scan-config/tools             # 获取扫描工具列表
POST   /api/v1/scan-config/tools/:id/install     # 安装扫描工具
GET    /api/v1/scan-config/tools/:id/health      # 健康检查
```

### 扫描规则管理 API
```
GET    /api/v1/scan-config/rules             # 获取扫描规则列表
POST   /api/v1/scan-config/rules/import      # 导入扫描规则
POST   /api/v1/scan-config/rule-engine/rules/:id/execute  # 执行规则
```

## 实际应用场景

### 内网安全扫描
```json
{
  "project": "内网资产扫描",
  "workflow": "端口发现 → 服务识别 → 漏洞扫描",
  "schedule": "每天凌晨2点执行",
  "tools": ["nmap", "nuclei"],
  "rules": ["过滤常用端口", "高危漏洞告警"]
}
```

### Web应用扫描
```json
{
  "project": "Web安全检测", 
  "workflow": "目录扫描 || 漏洞扫描",
  "schedule": "手动触发",
  "tools": ["gobuster", "nuclei"],
  "rules": ["敏感路径检测", "SQL注入告警"]
}
```

## 与其他模块的关系

**scan_config 是整个 NeoScan 系统的"大脑"：**
- **Agent 模块**：接收 scan_config 推送的扫描任务配置
- **执行器模块**：根据 scan_config 的工具配置执行具体扫描
- **结果处理**：按照 scan_config 的规则处理和存储扫描结果
- **通知系统**：根据 scan_config 的通知配置发送告警

## 技术特性

### 配置热重载
- 支持配置的动态更新，无需重启服务
- 基于文件监听和缓存机制实现

### 模板复用
- 提供丰富的配置模板库
- 支持自定义模板的创建和分享

### 权限控制
- 基于 JWT + RBAC 的权限管理
- 支持细粒度的操作权限控制

### 性能优化
- 数据库索引优化
- 缓存机制减少数据库查询
- 异步处理提高响应速度

## 开发规范

### 代码规范
- 严格遵循 Controller → Service → Repository → Database 的分层架构
- 使用统一的错误处理和日志记录
- 遵循项目的命名规范和代码风格

### 测试要求
- 单元测试覆盖率 > 80%
- 集成测试覆盖主要业务流程
- 性能测试确保响应时间 < 100ms

### 安全要求
- 输入验证和输出编码
- 敏感信息使用 .env 文件管理
- 定期进行安全扫描和代码审查

---

**作者**: Linus Torvalds (AI Assistant)  
**日期**: 2025.10.11  
**版本**: v1.0