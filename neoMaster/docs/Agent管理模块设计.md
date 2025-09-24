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
    Capabilities []string  `json:"capabilities" gorm:"type:json"`
    Tags         []string  `json:"tags" gorm:"type:json"`

    // 时间戳
    LastHeartbeat time.Time `json:"last_heartbeat"`
    RegisteredAt  time.Time `json:"registered_at"`
    UpdatedAt     time.Time `json:"updated_at"`
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

// Agent分组
type AgentGroupMember struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    Name        string    `json:"name" gorm:"not null"`
    Description string    `json:"description"`
    AgentID   string    `json:"agent_id"`
    GroupID   string    `json:"group_id"`
    Tags        []string  `json:"tags" gorm:"type:json"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
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


## 模型设计要点说明

### 1. 核心功能支持

该模型设计充分考虑了Agent管理模块的各项功能需求：

- **注册和发现**：通过 [ID](file://C:\Users\root\Desktop\code\PythonCode\NeoScan\NeoScan\neoMaster\debug\test_convert_utils.go#L18-L18)、`Hostname`、`IPAddress` 等基本信息实现Agent的唯一标识和发现
- **状态监控**：通过 [Status](file://C:\Users\root\Desktop\code\PythonCode\NeoScan\NeoScan\neoMaster\internal\model\user.go#L25-L25) 和 `LastHeartbeat` 字段跟踪Agent的实时状态
- **配置管理**：通过 `AgentConfig` 结构统一管理Agent配置，并支持配置推送和版本控制
- **负载监控**：通过 `LoadInfo` 结构收集CPU、内存、网络等负载信息，支持负载均衡决策

### 2. 扩展性考虑

- **标签系统**：支持通过标签对Agent进行分类管理，便于分组和筛选
- **能力声明**：通过 `Capabilities` 字段声明Agent支持的功能，便于任务分配
- **版本管理**：独立的 `AgentVersion` 模型支持版本管理和升级
- **分组管理**：通过 `AgentGroup` 模型支持Agent分组管理

### 3. 实用性增强

- **性能指标**：包含详细的系统和负载信息，便于监控和调度决策
- **配置灵活性**：支持插件配置、日志级别、超时时间等可配置项
- **任务统计**：跟踪运行中、已完成和失败的任务数量，便于负载评估
- **时间追踪**：记录注册时间、心跳时间和更新时间，便于状态管理

该模型设计既满足了当前Agent管理模块的功能需求，又具备良好的扩展性，能够适应未来可能的功能扩展需求。