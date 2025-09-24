## 扫描结果接收结构体设计

### 1. 通用扫描结果结构体

```go
// 通用扫描结果
type ScanResult struct {
    ID          string                 `json:"id" gorm:"primaryKey"`
    TaskID      string                 `json:"task_id" gorm:"index"`
    AgentID     string                 `json:"agent_id" gorm:"index"`
    Target      string                 `json:"target"`                     // 扫描目标
    PluginName  string                 `json:"plugin_name"`               // 使用的插件/工具名称
    PluginID    string                 `json:"plugin_id"`                 // 插件ID
    StartTime   time.Time              `json:"start_time"`
    EndTime     time.Time              `json:"end_time"`
    Status      string                 `json:"status"`                    // success, failed, partial
    RawResult   interface{}            `json:"raw_result" gorm:"-"`       // 原始结果数据
    ResultData  map[string]interface{} `json:"result_data" gorm:"type:json"` // 标准化结果数据
    Metadata    map[string]interface{} `json:"metadata" gorm:"type:json"`    // 元数据
    Error       string                 `json:"error"`                     // 错误信息
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// 漏洞信息结构体
type Vulnerability struct {
    ID            string            `json:"id" gorm:"primaryKey"`
    ScanResultID  string            `json:"scan_result_id" gorm:"index"`
    Name          string            `json:"name"`
    Description   string            `json:"description"`
    Severity      string            `json:"severity"` // critical, high, medium, low, info
    CVEID         string            `json:"cve_id"`
    CVSSScore     float64           `json:"cvss_score"`
    Reference     []string          `json:"reference" gorm:"type:json"`
    Solution      string            `json:"solution"`
    Target        string            `json:"target"`
    Port          int               `json:"port"`
    Protocol      string            `json:"protocol"`
    Evidence      string            `json:"evidence"` // 漏洞证据
    Metadata      map[string]interface{} `json:"metadata" gorm:"type:json"`
    CreatedAt     time.Time         `json:"created_at"`
}

// 端口扫描结果
type PortScanResult struct {
    ID           string    `json:"id" gorm:"primaryKey"`
    ScanResultID string    `json:"scan_result_id" gorm:"index"`
    Host         string    `json:"host"`
    Port         int       `json:"port"`
    Protocol     string    `json:"protocol"`
    Service      string    `json:"service"`
    Version      string    `json:"version"`
    State        string    `json:"state"` // open, closed, filtered
    Banner       string    `json:"banner"`
    CreatedAt    time.Time `json:"created_at"`
}

// Web扫描结果
type WebScanResult struct {
    ID            string            `json:"id" gorm:"primaryKey"`
    ScanResultID  string            `json:"scan_result_id" gorm:"index"`
    URL           string            `json:"url"`
    Title         string            `json:"title"`
    StatusCode    int               `json:"status_code"`
    Technologies  []string          `json:"technologies" gorm:"type:json"`
    Headers       map[string]string `json:"headers" gorm:"type:json"`
    Forms         []WebForm         `json:"forms" gorm:"type:json"`
    Links         []string          `json:"links" gorm:"type:json"`
    Screenshots   []string          `json:"screenshots" gorm:"type:json"` // 截图URL列表
    Metadata      map[string]interface{} `json:"metadata" gorm:"type:json"`
    CreatedAt     time.Time         `json:"created_at"`
}

// Web表单信息
type WebForm struct {
    Action   string            `json:"action"`
    Method   string            `json:"method"`
    Inputs   []WebFormInput    `json:"inputs" gorm:"type:json"`
}

// Web表单输入项
type WebFormInput struct {
    Name  string `json:"name"`
    Type  string `json:"type"`
    Value string `json:"value"`
}
```


### 2. gRPC服务中的扫描结果定义

根据文档中的gRPC定义，可以补充以下内容：

```go
// 上报扫描结果请求
message ReportScanResultRequest {
  string agent_id = 1;
  string task_id = 2;
  string plugin_name = 3;
  string target = 4;
  bytes raw_result = 5;  // 原始结果数据（字节流）
  ScanResultData result_data = 6;  // 结构化结果数据
  string status = 7;  // success, failed, partial
  string error_message = 8;
  int64 start_time = 9;
  int64 end_time = 10;
  map<string, string> metadata = 11;
}

// 结构化扫描结果数据
message ScanResultData {
  oneof result_type {
    PortScanData port_scan = 1;
    WebScanData web_scan = 2;
    VulnScanData vuln_scan = 3;
    GenericScanData generic = 4;
  }
}

// 端口扫描数据
message PortScanData {
  repeated PortInfo ports = 1;
}

// 端口信息
message PortInfo {
  int32 port = 1;
  string protocol = 2;
  string service = 3;
  string version = 4;
  string state = 5;
  string banner = 6;
}

// Web扫描数据
message WebScanData {
  string url = 1;
  int32 status_code = 2;
  string title = 3;
  map<string, string> headers = 4;
  repeated string technologies = 5;
  repeated WebFormInfo forms = 6;
}

// Web表单信息
message WebFormInfo {
  string action = 1;
  string method = 2;
  repeated WebInputInfo inputs = 3;
}

// Web输入信息
message WebInputInfo {
  string name = 1;
  string type = 2;
  string value = 3;
}

// 漏洞扫描数据
message VulnScanData {
  repeated VulnerabilityInfo vulnerabilities = 1;
}

// 漏洞信息
message VulnerabilityInfo {
  string name = 1;
  string description = 2;
  string severity = 3;
  string cve_id = 4;
  double cvss_score = 5;
  string solution = 6;
  repeated string references = 7;
  string evidence = 8;
}

// 通用扫描数据
message GenericScanData {
  map<string, string> data = 1;
}

// 上报扫描结果响应
message ReportScanResultResponse {
  bool success = 1;
  string message = 2;
  string result_id = 3;
}
```


### 3. 设计说明

这种设计具有以下优势：

1. **灵活性**：支持不同工具返回的不同格式结果
    - 通过`RawResult`字段保存原始数据
    - 通过`ResultData`字段保存标准化后的数据

2. **可扩展性**：
    - 定义了多种特定类型的扫描结果结构体
    - 支持添加新的扫描类型

3. **标准化**：
    - 提供了统一的结果存储格式
    - 便于后续的数据分析和处理

4. **元数据支持**：
    - 通过`Metadata`字段可以存储额外的上下文信息
    - 便于追踪和调试

5. **错误处理**：
    - 包含了错误信息字段
    - 支持部分成功的结果上报

通过这种设计，系统可以接收和处理来自不同扫描工具（如nmap、nuclei、xray等）的多样化结果，同时保持数据的一致性和可管理性。