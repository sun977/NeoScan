# Executor Architecture Design (Fusion Style)

## 1. 核心理念 (Core Philosophy)

**"Executor is Muscle, Adapter is Brain."**

本模块采用 **Brain & Muscle Fusion** 架构，将原有的 `executor` (Process Manager) 与 `tool_adapter` (Protocol Translator) 进行有机融合，旨在解决代码分裂问题，实现高内聚、低耦合的工具集成。

*   **Executor (Muscle)**: 负责运行时生命周期管理（启动、停止、超时、资源限制）。它不知道工具的具体业务逻辑，只负责"跑命令"。
*   **Adapter (Brain)**: 负责业务逻辑转换（参数拼接、结果解析）。它不知道进程如何管理，只负责"思考"。

## 2. 目录结构 (Directory Structure)

```text
internal/executor/
├── base/                   # [Existing] 基础接口 (Executor, Task)
├── core/                   # [New] 核心适配能力 (原 tool_adapter 的精华)
│   ├── command.go          # CommandBuilder 接口
│   ├── parser.go           # Parser 接口
│   └── types.go            # ToolScanResult (统一输出结构)
├── masscan/                # [Existing] 具体工具实现
│   ├── executor.go         # [Run] 运行时 (Start/Stop)
│   ├── adapter.go          # [Brain] Masscan 适配逻辑 (实现 core 接口)
│   └── masscan_test.go
├── nmap/
│   ├── executor.go
│   ├── adapter.go          # [Brain] Nmap 专用的解析逻辑
│   └── xml_models.go       # [Model] XML 结构定义
└── manager/                # [Existing] 执行器管理器
```

## 3. 核心组件 (Core Components)

### 3.1 Executor Core (`core` package)

定义了所有 Adapter 必须遵守的协议。

*   **`CommandBuilder`**: `Build(target, config) -> (cmd, args)`
    *   将抽象的任务配置转换为具体的 CLI 命令。
*   **`Parser`**: `Parse(output) -> ToolScanResult`
    *   将工具的原始输出（XML/JSON/Text）转换为标准化的中间结果。
*   **`ToolScanResult`**: 标准化的资产数据结构 (Host, Port, Web, Vuln)。

### 3.2 Tool Implementation (e.g., `masscan` package)

每个工具包都是自包含的 (Self-Contained)，包含该工具的所有逻辑。

*   **`adapter.go`**: 实现 `core.CommandBuilder` 和 `core.Parser`。
*   **`executor.go`**: 实现 `base.Executor`，内部持有 `adapter` 实例。

## 4. 交互流程 (Interaction Flow)

```go
// 伪代码示例

func (e *MasscanExecutor) Start() error {
    // 1. Brain: 构建命令
    cmdStr, args, err := e.adapter.Build(e.target, e.config)
    if err != nil {
        return err
    }
    
    // 2. Muscle: 执行命令 (Process Management)
    cmd := exec.CommandContext(ctx, cmdStr, args...)
    output, err := cmd.CombinedOutput()
    
    // 3. Brain: 解析结果
    result, err := e.adapter.Parse(string(output))
    if err != nil {
        return err
    }
    
    // 4. Report: 上报标准化结果
    e.report(result)
    return nil
}
```

## 5. 优势 (Benefits)

1.  **高内聚 (High Cohesion)**: 工具的所有逻辑（执行+解析）都在同一个包内，修改方便。
2.  **单一职责 (SRP)**: Executor 专注进程管理，Adapter 专注协议转换。
3.  **标准化 (Standardization)**: 所有工具输出都转换为 `ToolScanResult`，简化了上层 ETL 处理。
4.  **可测试性 (Testability)**: Adapter 是纯函数式的（无副作用），极易进行单元测试。

## 6. 迁移指南 (Migration Guide)

对于现有的 Executor：
1.  在同级目录下创建 `adapter.go`。
2.  将 `executor.go` 中的命令拼接逻辑移动到 `adapter.Build`。
3.  将 `executor.go` 中的结果解析逻辑（及相关 Struct）移动到 `adapter.Parse`。
4.  在 `executor.go` 中调用 `adapter` 的方法。
