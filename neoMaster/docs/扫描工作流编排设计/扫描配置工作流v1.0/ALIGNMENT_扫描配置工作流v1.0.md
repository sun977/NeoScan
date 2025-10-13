# 需求对齐文档 - 扫描配置工作流v1.0

## 原始需求

基于用户提供的扫描配置核心流程设计，需要实现一个完整的扫描配置工作流系统，包含以下核心概念：

1. **工程方案**：支持24小时不间断和周期性扫描的完整工作流
2. **非工程方案**：单独执行的临时扫描任务
3. **高级功能**：基于AI工作流的可视化节点拖拽自定义配置

核心功能模块包括：
- 时间周期管理（24小时不间断/周期性）
- 数据源管理（系统数据库/API/第三方数据库）
- 扫描工作流（工具配置/扫描逻辑/结果处理）
- 扫描策略（白名单/跳过规则/超时/代理）
- 结果存放管理
- 通知系统

## 项目上下文分析

### 技术栈
- **编程语言**：Go 1.21+
- **Web框架**：Gin
- **ORM框架**：GORM
- **数据库**：MySQL 8.0+
- **缓存**：Redis
- **日志**：Logrus
- **配置管理**：Viper + 热重载机制
- **通信协议**：gRPC (Agent通信) + HTTP/WebSocket (API)

### 现有架构理解

#### 分层架构模式
```
Controller/Handler → Service → Repository → Database
```

#### 核心模块分析
1. **Agent管理模块**：已实现Agent注册、状态监控、配置推送
2. **认证授权模块**：JWT + RBAC权限控制
3. **配置管理模块**：支持热重载的多环境配置
4. **数据模型**：
   - Agent相关：Agent、AgentConfig、AgentMetrics、AgentTaskAssignment
   - 扫描类型：AgentScanType枚举（11种扫描类型）
   - 任务状态：AgentTaskStatus枚举

#### 现有扫描能力
系统已定义11种扫描类型：
- ipAliveScan（IP探活）
- fastPortScan（快速端口扫描）
- fullPortScan（全量端口扫描）
- serviceScan（服务扫描）
- vulnScan（漏洞扫描）
- pocScan（POC扫描）
- webScan（Web扫描）
- passScan（弱密码扫描）
- proxyScan（代理扫描）
- dirScan（目录扫描）
- subDomainScan（子域名扫描）
- apiScan（API扫描）

## 需求理解

### 功能边界

**包含功能：**
- [x] 扫描工程方案管理（CRUD操作）
- [x] 时间调度系统（立即/定时/周期执行）
- [x] 多数据源集成（数据库/API/文件/手动）
- [x] 扫描工作流编排（阶段化扫描流程）
- [x] 扫描策略配置（白名单/黑名单/跳过规则）
- [x] 结果存储管理（多种存储方式）
- [x] 通知系统集成（多渠道通知）
- [x] 非工程方案支持（临时扫描）
- [x] 可视化工作流编辑器（拖拽式配置）
- [x] 配置模板管理（预设方案）

**明确不包含（Out of Scope）：**
- [x] Agent节点的具体扫描工具实现
- [x] 扫描结果的详细分析和报告生成
- [x] 第三方系统的具体对接实现
- [x] 前端UI界面的详细设计

### 核心数据结构设计

基于Linus的"好品味"原则，数据结构应该消除特殊情况，让代码更简洁：

#### 1. 统一的配置表示
所有配置（工程方案/非工程方案/高级工作流）都使用统一的JSON结构，避免多套数据模型。

#### 2. 扫描阶段的线性化
将复杂的扫描依赖关系转换为简单的有序阶段列表，消除循环依赖的特殊处理。

#### 3. 策略规则的统一化
白名单、黑名单、跳过规则使用统一的规则引擎，避免多套判断逻辑。

## 疑问澄清

### P0级问题（必须澄清）

1. **数据源easeData概念**
   - 背景：用户提到"数据源数据集概念，easeData"
   - 影响：影响数据源抽象设计
   - 建议方案：设计统一的数据源接口，支持多种数据源类型

2. **AI工作流节点拖拽实现方式**
   - 背景：需要支持可视化的工作流编辑
   - 影响：影响前后端架构设计
   - 建议方案：后端提供工作流节点定义API，前端使用流程图库实现拖拽

3. **扫描结果处理的具体需求**
   - 背景：扫描结果需要进行处理和转换
   - 影响：影响结果处理器的设计
   - 建议方案：设计插件化的结果处理器架构

### P1级问题（建议澄清）

1. **通知系统的具体渠道**
   - 当前系统已支持蓝信、SEC、邮件、Webhook
   - 是否需要扩展其他通知渠道？

2. **代理配置的安全性**
   - 代理配置中的敏感信息如何安全存储？
   - 建议使用加密存储和环境变量管理

## 技术方案建议

### 1. 数据结构优化
```go
// 统一的扫描配置结构
type ScanProject struct {
    BaseModel
    Name        string          `json:"name"`
    Type        ProjectType     `json:"type"` // project/adhoc/workflow
    Config      ProjectConfig   `json:"config"`
    Status      ProjectStatus   `json:"status"`
}

// 统一的配置结构
type ProjectConfig struct {
    Schedule    ScheduleConfig  `json:"schedule"`
    DataSource  DataSourceConfig `json:"data_source"`
    Workflow    WorkflowConfig  `json:"workflow"`
    Strategy    StrategyConfig  `json:"strategy"`
    Storage     StorageConfig   `json:"storage"`
    Notification NotificationConfig `json:"notification"`
}
```

### 2. 简化的调度系统
使用cron表达式统一处理所有时间调度，消除多套调度逻辑：
```go
type ScheduleConfig struct {
    Type        string `json:"type"` // immediate/scheduled/recurring
    CronExpr    string `json:"cron_expr"`
    StartTime   time.Time `json:"start_time"`
    EndTime     *time.Time `json:"end_time"`
    Enabled     bool   `json:"enabled"`
}
```

### 3. 统一的规则引擎
```go
type Rule struct {
    Condition   string `json:"condition"` // 使用表达式语言
    Action      string `json:"action"`    // allow/deny/skip
    Priority    int    `json:"priority"`
}
```

## 验收标准

### 功能验收
- [x] 标准1：能够创建、编辑、删除扫描工程方案
- [x] 标准2：支持立即执行、定时执行、周期执行三种调度模式
- [x] 标准3：支持至少3种数据源类型（数据库、API、文件）
- [x] 标准4：能够配置完整的扫描工作流（工具链、参数、依赖关系）
- [x] 标准5：支持白名单、黑名单、跳过规则配置
- [x] 标准6：能够配置多种通知方式
- [x] 标准7：支持非工程方案的临时扫描
- [x] 标准8：提供工作流可视化配置接口

### 质量验收
- [x] 单元测试覆盖率 > 80%
- [x] API响应时间 < 500ms
- [x] 支持并发配置操作
- [x] 配置数据完整性验证
- [x] 敏感信息加密存储

### 安全验收
- [x] 配置操作需要权限验证
- [x] 敏感配置信息加密存储
- [x] 输入参数严格验证
- [x] SQL注入防护
- [x] XSS攻击防护

## 设计原则

### Linus式设计思考

1. **"好品味"原则**
   - 消除特殊情况：统一配置结构，避免多套处理逻辑
   - 简化数据结构：使用JSON统一表示所有配置类型
   - 线性化处理：将复杂依赖关系转换为简单的有序列表

2. **"Never break userspace"原则**
   - 向后兼容：新版本配置格式兼容旧版本
   - 渐进式迁移：支持配置格式的平滑升级
   - API稳定性：保持API接口的向后兼容

3. **实用主义原则**
   - 解决实际问题：专注于扫描配置的核心需求
   - 避免过度设计：不实现不必要的复杂功能
   - 性能优先：优化配置加载和执行性能

4. **简洁执念**
   - 函数职责单一：每个函数只做一件事
   - 避免深层嵌套：配置结构扁平化设计
   - 清晰的命名：使用明确的变量和函数名

## 下一步行动

1. 确认以上需求理解是否准确
2. 澄清P0级问题的具体需求
3. 进入架构设计阶段，详细设计系统架构和数据模型