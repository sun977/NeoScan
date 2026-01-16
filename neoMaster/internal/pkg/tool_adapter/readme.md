├── pkg/                      # [通用库] - 嵌入基础设施
│   ├── tool_adapter/         # [新增] 工具适配层 (Infrastructure)   # 这个包在agent端也需要有
│   │   ├── command/          # [重构] 命令生成模块 (原 factory)
│   │   │   ├── interface.go       # CommandBuilder 接口
│   │   │   ├── template.go        # [New] 通用模板构建器 (Go Template)
│   │   │   └── nmap.go            # [New] Nmap 专用构建器
│   │   ├── parser/           # 结果解析模块
│   │   │   ├── interface.go       # Parser 接口
│   │   │   ├── nmap_xml.go        # [New] Nmap XML 解析器
│   │   │   └── masscan_json.go    # [New] Masscan JSON 解析器
│   │   ├── registry/         # 注册中心
│   │   │   └── registry.go        # 工具注册表
│   │   └── models/           # [New] 中间数据模型
│   │       └── result.go          # 统一的解析结果结构
│   │   └── readme.md

# Tool Adapter Layer Design (Linus Style)

## 1. 核心理念 (Core Philosophy)

**"Good programmers worry about data structures."**

本模块旨在实现 Master/Agent 与具体安全工具的**完全解耦**。Master 只需知道 "我要跑 Nmap"，而不需要知道 Nmap 的具体参数拼接细节或 XML 输出格式。所有的脏活累活（参数拼接、输出解析）都由 Adapter 层封装。

## 2. 架构设计 (Architecture)

### 2.1 Command Builder (命令构建)

我们采用 **"Template First" (模板优先)** 策略。
绝大多数工具（Masscan, Nuclei, HTTPX）的命令格式是固定的，只是参数不同。

*   **TemplateCommandBuilder**:
    *   使用 Go Template 语法定义命令模式。
    *   例如: `masscan -p {{.Ports}} {{.Target}} --rate {{.Rate}} -oJ {{.OutputFile}}`
    *   **优点**: 配置即代码，灵活性极高。

*   **CustomCommandBuilder**:
    *   针对 Nmap 等参数逻辑极其复杂（依赖 OS、依赖网络接口、互斥参数多）的工具，允许编写专门的 Go 代码进行构建。

### 2.2 Result Parser (结果解析)

**"Type safety is good."**

解析器的输出不再是弱类型的 `map[string]interface{}`，而是标准化的 **Intermediate Result Structure (中间结果结构)**。这层结构作为工具原始输出与 NeoScan 业务模型 (`StageResult`) 之间的桥梁。

#### Standardized Result Model (`models/result.go`)

```go
package models

// ToolScanResult 工具执行的标准化中间结果
// Parser 的职责是将 XML/JSON/Text 转换为这个结构
type ToolScanResult struct {
    ToolName  string    `json:"tool_name"`
    StartTime int64     `json:"start_time"`
    EndTime   int64     `json:"end_time"`
    Status    string    `json:"status"` // success, failed
    
    // 原始输出 (可选，用于 Debug)
    RawOutput string    `json:"raw_output,omitempty"`

    // --- 标准化资产数据 (Flattened) ---
    // Parser 必须尽力将结果映射到以下切片中
    
    Hosts []HostInfo    `json:"hosts,omitempty"` // 存活主机
    Ports []PortInfo    `json:"ports,omitempty"` // 开放端口
    Webs  []WebInfo     `json:"webs,omitempty"`  // Web 服务
    Vulns []VulnInfo    `json:"vulns,omitempty"` // 漏洞/风险
}

type HostInfo struct {
    IP       string `json:"ip"`
    Hostname string `json:"hostname"`
    OS       string `json:"os"`
}

type PortInfo struct {
    IP      string `json:"ip"`
    Port    int    `json:"port"`
    Proto   string `json:"proto"`   // tcp/udp
    Service string `json:"service"` // http, ssh
    Product string `json:"product"` // nginx
    Version string `json:"version"` // 1.14.2
    Banner  string `json:"banner"`
}

// ... WebInfo, VulnInfo ...
```

### 2.3 Registry (注册中心)

采用 **自注册 (Self-Registration)** 模式。

```go
// 在工具对应的 init() 中自动注册
func init() {
    registry.Register("nmap", 
        &NmapCommandBuilder{}, 
        &NmapXMLParser{},
    )
}
```

## 3. 交互流程 (Interaction Flow)

1.  **Task Dispatch**: Agent 收到任务，包含 `ToolName` ("nmap") 和 `Params` (Map)。
2.  **Command Build**: Agent 调用 `registry.GetBuilder("nmap").Build(Params)` 得到命令行。
3.  **Execution**: Agent 执行命令，捕获 Stdout/Stderr 或读取 Output File。
4.  **Parsing**: Agent 调用 `registry.GetParser("nmap").Parse(Output)`。
    *   Parser 将 Nmap XML 解析为 `models.ToolScanResult`。
5.  **Reporting**: Agent 将 `ToolScanResult` 序列化为 JSON，通过 `StageResult.Attributes` 上报给 Master。
6.  **ETL**: Master 的 ETL 模块读取 `StageResult`，直接映射到 `Asset` 表 (因为结构已经标准化了，Mapper 逻辑会变得非常简单)。

## 4. 实施路线 (Roadmap)

1.  **Phase 1**: 定义 `models` 包和标准化结构。
2.  **Phase 2**: 实现 `TemplateCommandBuilder` 和 `Registry`。
3.  **Phase 3**: 实现 `NmapParser` (XML -> ToolScanResult) 和 `MasscanParser`。
4.  **Phase 4**: 在 Agent 端集成。
