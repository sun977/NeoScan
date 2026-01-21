# 设计文档 - [NeoAgent 核心指令集建设]

## 架构概览

### 整体架构图
```mermaid
graph TD
    User[用户 (CLI)] --> Cobra[Cobra Commands]
    Master[Master (JSON)] --> Listener[Task Listener]
    
    subgraph "Interface Layer"
        Cobra -->|Flags| Options[Options Structs]
        Listener -->|JSON| Options
    end
    
    subgraph "Core Domain"
        Options -->|ToTask()| Task[model.Task]
        Task -->|Validate()| ValidTask[Validated Task]
    end
    
    ValidTask --> Runner[Task Runner (Future)]
```

## 模块设计

### 1. Command Layer (`cmd/agent`)
负责处理用户输入，展示帮助信息，调用 Options 层进行数据转换。
- `cmd/agent/scan/*.go`: 负责扫描相关子命令。
- `cmd/agent/proxy/*.go`: 负责代理相关子命令。

### 2. Options Layer (`internal/core/options`)
负责定义强类型的参数结构，处理默认值，进行基础校验，并转换为核心 Task 模型。

#### 结构体定义示例
```go
type PortScanOptions struct {
    Target    string
    PortRange string
    Rate      int
    // ...
}

func (o *PortScanOptions) Validate() error { ... }
func (o *PortScanOptions) ToTask() *model.Task { ... }
```

### 3. Model Layer (`internal/core/model`)
核心领域模型，保持纯净，不依赖 CLI 或 HTTP 框架。

## 接口契约
### CLI 接口
- `neoAgent scan asset -t <target> [--os-detect]`
- `neoAgent scan port -t <target> -p <ports> [--rate <rate>]`
- `neoAgent proxy --mode <mode> --listen <addr> [--auth <user:pass>]`

### 内部接口 (Go)
```go
// internal/core/options/interface.go
type TaskOption interface {
    Validate() error
    ToTask() *model.Task
}
```

## 异常处理策略
- **Flag 解析错误**：Cobra 自动处理，打印 Usage。
- **参数校验错误**：`Validate()` 返回 error，CLI 层捕获并打印错误日志（不打印堆栈）。
- **模型转换错误**：属于内部逻辑错误，应 Log Error 并退出。
