## Agent模型设计

基于文档中的Agent管理模块需求，建议采用如下数据模型设计：

```go
// 1. Agent基础信息 - 相对静态，注册时确定
type Agent struct {
    // 基本信息
    ID           string    `json:"id" gorm:"primaryKey"`
    AgentID      string    `json:"agent_id" gorm:"index"`
    Hostname     string    `json:"hostname" gorm:"not null"`
    IPAddress    string    `json:"ip_address" gorm:"not null"`
    Port         int       `json:"port" gorm:"default:5772"`
    Version      string    `json:"version"`
    Status       string    `json:"status" gorm:"default:offline"` // online, offline, error, maintenance
    
    // 静态系统信息
    OS           string    `json:"os"`
    Arch         string    `json:"arch"`
    CPUCores     int       `json:"cpu_cores"`
    MemoryTotal  int64     `json:"memory_total"`
    DiskTotal    int64     `json:"disk_total"`

    // 能力和标签
    Capabilities []string  `json:"capabilities" gorm:"type:json"`  // 表示agent支持的功能模块
    Tags         []string  `json:"tags" gorm:"type:json"`

    // 安全认证字段
    Token        string    `json:"token"`          // 用于通信的Token
    TokenExpiry  time.Time `json:"token_expiry"`   // Token过期时间

    // 时间戳
    ResultLatestTime *time.Time `json:"result_latest_time"` // 最新的返回结果时间
    LastHeartbeat time.Time `json:"last_heartbeat"`
    RegisteredAt  time.Time `json:"registered_at"`
    UpdatedAt     time.Time `json:"updated_at"`
    
    // 扩展字段
    Remark           string     `json:"remark"`             // 备注信息
}

// 2. Agent配置 - 独立管理，支持版本控制
type AgentConfig struct {
    ID                  string    `json:"id" gorm:"primaryKey"`
    AgentID             string    `json:"agent_id" gorm:"index"`
    Version             int       `json:"version" gorm:"default:1"`
    HeartbeatInterval   int            `json:"heartbeat_interval"`    // 心跳间隔
    TaskPollInterval    int            `json:"task_poll_interval"`     // 任务轮询间隔
    MaxConcurrentTasks  int            `json:"max_concurrent_tasks"`   // 最大并发任务数
    PluginConfig        map[string]interface{} `json:"plugin_config" gorm:"type:json"` // 插件配置
    LogLevel            string         `json:"log_level"`              // 日志级别
    ScanTimeout         int            `json:"scan_timeout"`           // 扫描超时时间
    TokenExpiryDuration int            `json:"token_expiry_duration"`  // Token过期时间（秒）
    TokenNeverExpire    bool           `json:"token_never_expire"`     // Token是否永不过期 true 表示永不过期
    IsActive            bool      `json:"is_active" gorm:"default:true"`
    CreatedAt           time.Time `json:"created_at"`
    UpdatedAt           time.Time `json:"updated_at"`
}

// 3. Agent负载信息 - 高频更新，独立存储
type AgentMetrics struct {
    ID                string    `json:"id" gorm:"primaryKey"`
    AgentID           string    `json:"agent_id" gorm:"index"`
    CPUUsage          float64   `json:"cpu_usage"`
    MemoryUsage       float64   `json:"memory_usage"`
    DiskUsage         float64   `json:"disk_usage"`
    NetworkBytesSent  int64     `json:"network_bytes_sent"`
    NetworkBytesRecv  int64     `json:"network_bytes_recv"`
    ActiveConnections int       `json:"active_connections"`
    RunningTasks      int       `json:"running_tasks"`
    CompletedTasks    int       `json:"completed_tasks"`
    FailedTasks       int       `json:"failed_tasks"`
    // 新增字段
    WorkStatus string    `json:"work_status"` // 工作状态：空闲/工作中/异常
    ScanStatus string    `json:"scan_status"` // 扫描阶段：空闲/IP探活/快速扫描/端口扫描/漏洞扫描等
    PluginStatus map[string]interface{} `json:"plugin_status"` // 插件状态信息 key: 插件名称, value: 插件状态详情【第三方工具都可以使用这一个字段】
    Timestamp         time.Time `json:"timestamp" gorm:"index"`
}

// 统一的度量接口
type MetricsCollector interface {
    GetCPUUsage() float64
    GetMemoryUsage() float64
    GetDiskUsage() float64
    GetNetworkStats() (sent, recv int64)
    GetTaskStats() (running, completed, failed int)
}

// Agent实现度量收集
func (a *Agent) CollectMetrics() *AgentMetrics {
    return &AgentMetrics{
        ID:      generateMetricsID(),
        AgentID: a.ID,
        // ... 统一的度量收集逻辑
        Timestamp: time.Now(),
    }
}

// 4. Agent分组
type AgentGroup struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    Name        string    `json:"name" gorm:"not null"`
    Description string    `json:"description"`
    Tags        []string  `json:"tags" gorm:"type:json"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type AgentGroupMember struct {
    AgentID   string    `json:"agent_id" gorm:"primaryKey"`
    GroupID   string    `json:"group_id" gorm:"primaryKey"`
    JoinedAt  time.Time `json:"joined_at"`
}

// Agent版本信息
type AgentVersion struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    Version     string    `json:"version" gorm:"not null"`
    ReleaseDate time.Time `json:"release_date"`
    Changelog   string    `json:"changelog"`
    DownloadURL string    `json:"download_url"`
    IsActive    bool      `json:"is_active"`
    IsLatest    bool      `json:"is_latest"
}
```


## 架构设计原则与优化说明

### 1. 数据分离策略 - "好品味"设计原则

本设计遵循Linus Torvalds的"好品味"原则，按数据生命周期进行分离：

#### 1.1 三层数据分离架构
```
Agent (静态基础信息) ← 相对稳定，注册时确定
    ↓
AgentConfig (配置信息) ← 中等频率变更，支持版本控制  
    ↓
AgentMetrics (动态负载) ← 高频更新，独立存储优化
```

**设计优势：**
- **消除数据冲突**：不同更新频率的数据完全分离，避免锁竞争
- **优化存储性能**：高频数据独立存储，可采用时序数据库优化
- **简化缓存策略**：静态数据可长期缓存，动态数据短期缓存
- **降低复杂度**：每个模型职责单一，避免"上帝对象"反模式

#### 1.2 数据关系设计
- **Agent ↔ AgentConfig**：一对多关系，支持配置版本历史
- **Agent ↔ AgentMetrics**：一对多关系，支持历史监控数据
- **Agent ↔ AgentGroupMember**：多对多关系，灵活分组管理

### 2. 核心功能支持与实现策略

#### 2.1 Agent注册与发现
- **唯一标识**：使用`ID`作为主键，`AgentID`作为业务标识
- **网络发现**：通过`Hostname`、`IPAddress`、`Port`实现网络层发现
- **能力匹配**：通过`Capabilities`数组支持功能匹配和任务分配
- **标签分类**：通过`Tags`数组支持灵活的分类和筛选

#### 2.2 状态监控与健康检查
- **实时状态**：`Status`字段记录当前状态（online/offline/error/maintenance）
- **心跳机制**：`LastHeartbeat`字段跟踪最后活跃时间
- **状态转换**：支持状态机模式，确保状态转换的合法性
- **异常检测**：基于心跳超时自动检测Agent异常

#### 2.3 配置管理与版本控制
- **版本化配置**：`AgentConfig.Version`支持配置版本管理
- **增量推送**：只推送变更的配置项，减少网络开销
- **回滚机制**：保留历史配置版本，支持快速回滚
- **配置验证**：推送前验证配置合法性，避免Agent异常

#### 2.4 负载监控与调度优化
- **实时指标**：`AgentMetrics`收集CPU、内存、磁盘、网络等关键指标
- **任务统计**：跟踪运行中、已完成、失败的任务数量
- **负载均衡**：基于实时负载数据进行智能任务分配
- **历史分析**：保留历史监控数据，支持性能趋势分析

### 3. 扩展性与可维护性设计

#### 3.1 接口抽象化
```go
// MetricsCollector接口支持不同类型的监控实现
type MetricsCollector interface {
    GetCPUUsage() float64
    GetMemoryUsage() float64
    GetDiskUsage() float64
    GetNetworkStats() (sent, recv int64)
    GetTaskStats() (running, completed, failed int)
}
```

#### 3.2 插件化架构
- **能力扩展**：通过`Capabilities`声明支持的扫描类型
- **插件配置**：`PluginConfig`支持动态插件配置
- **热插拔**：支持运行时加载/卸载插件，无需重启Agent

#### 3.3 分组管理策略
- **灵活分组**：`AgentGroupMember`支持Agent的多维度分组
- **标签继承**：分组标签可继承到成员Agent
- **批量操作**：支持按分组进行批量配置推送和任务分配

### 4. 性能优化考虑

#### 4.1 数据库优化
- **索引策略**：为`AgentID`、`Timestamp`等高频查询字段建立索引
- **分区存储**：`AgentMetrics`按时间分区，提高查询性能
- **数据清理**：定期清理过期的监控数据，控制存储增长

#### 4.2 缓存策略
- **多层缓存**：Agent基础信息使用Redis缓存，配置信息使用本地缓存
- **缓存更新**：配置变更时主动失效相关缓存
- **缓存预热**：系统启动时预加载活跃Agent信息

#### 4.3 网络优化
- **批量操作**：支持批量Agent注册和状态更新
- **压缩传输**：监控数据传输使用gzip压缩
- **连接复用**：使用HTTP/2或gRPC提高连接效率

### 5. 安全性与可靠性

#### 5.1 安全机制
- **身份验证**：Agent注册需要提供有效的认证凭据
- **通信加密**：所有Agent通信使用TLS加密
- **权限控制**：基于Agent能力和标签进行细粒度权限控制

#### 5.2 容错设计
- **心跳容错**：支持网络抖动的心跳容错机制
- **配置回滚**：配置推送失败时自动回滚到上一版本
- **故障隔离**：单个Agent故障不影响其他Agent的正常运行

### 6. 与现有系统集成

#### 6.1 模型规范对齐
- **命名规范**：遵循项目现有的驼峰命名规范
- **字段标签**：统一使用`json`和`gorm`标签
- **时间管理**：使用`time.Time`类型，与现有User、Role模型保持一致
- **业务方法**：参考User模型添加`IsActive()`、`HasCapability()`等业务方法

#### 6.2 数据库集成
- **表名规范**：添加`TableName()`方法指定表名
- **外键约束**：合理设置外键约束，保证数据一致性
- **迁移脚本**：提供完整的数据库迁移脚本

#### 6.3 API集成
- **RESTful设计**：遵循项目现有的API设计规范
- **错误处理**：使用项目统一的错误处理机制
- **日志记录**：使用logrus记录关键操作日志

该模型设计不仅解决了当前Agent管理的核心需求，更重要的是建立了一个可扩展、高性能、易维护的架构基础，为后续功能扩展提供了坚实的技术保障。