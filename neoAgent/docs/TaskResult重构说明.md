# TaskResult 重构与最佳实践指南

## 1. 背景与问题

当前的 `TaskResult` 结构体定义如下：

```go
type TaskResult struct {
    // ... 其他字段
    Result map[string]interface{} `json:"result"` // 执行结果
    // ...
}
```

**存在的问题**:
1.  **弱类型**: `map[string]interface{}` 缺乏编译时检查，容易导致字段名拼写错误（如 `ip` 写成 `Ip`）或类型错误（如 `port` 存成字符串）。
2.  **契约丢失**: 开发者在编写具体的 Executor 时，不知道 Master 期望什么样的 JSON 结构，容易导致 Agent 与 Master 协议不一致。
3.  **维护困难**: 随着扫描类型增加，`Result` 的内容变得不可控，难以维护和重构。

---

## 2. 重构目标

在**不修改核心通信协议**（即 `TaskResult` 结构体保持不变）的前提下，通过**代码规范**和**DTO (Data Transfer Object)** 模式，实现强类型约束和契约对齐。

---

## 3. 实施方案：Payload DTO 模式

建议在 `internal/model/payloads` 包（或各 Executor 内部）定义具体的 Payload 结构体。

**重要提示**: 根据 `Master-Agent 数据契约`，`TaskResult.Result` 顶层必须包含 `attributes` 和 `evidence`。以下定义的 DTO 仅对应 `attributes` 字段的内容。

### 3.1 定义 Payload 结构体 (对应 attributes)

根据 `Master_Agent_Data_Contract.md` 文档，为每种 Result Type 定义对应的 Go Struct。

**示例：端口扫描 Payload**

```go
package payloads

// PortScanResult 对应 Master 的 port_scan 契约
type PortScanResult struct {
    Ports   []PortInfo        `json:"ports"`
    Summary PortScanSummary   `json:"summary,omitempty"`
}

type PortInfo struct {
    IP          string `json:"ip"`
    Port        int    `json:"port"`          // 必须是 int
    Proto       string `json:"proto"`         // tcp/udp
    State       string `json:"state"`         // open/closed
    ServiceHint string `json:"service_hint"`  // 可选
    Banner      string `json:"banner"`        // 可选
}

type PortScanSummary struct {
    OpenCount    int    `json:"open_count"`
    ScanStrategy string `json:"scan_strategy"`
    ElapsedMs    int64  `json:"elapsed_ms"`
}
```

### 3.2 在 Executor 中使用

Executor 不应直接操作 `map[string]interface{}`，而应先填充 DTO，最后再转换。

**示例代码**:

```go
func (e *NmapExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // 1. 执行 Nmap 扫描
    nmapOutput := runNmap(...) 

    // 2. 转换为强类型的 Payload DTO
    payload := payloads.PortScanResult{
        Ports: make([]payloads.PortInfo, 0),
    }

    for _, host := range nmapOutput.Hosts {
        for _, port := range host.Ports {
            payload.Ports = append(payload.Ports, payloads.PortInfo{
                IP:    host.IP,
                Port:  port.ID,
                Proto: port.Protocol,
                State: port.State,
            })
        }
    }
    
    // 3. 构造 Top-Level Result
    // 契约要求：Result 必须包含 attributes 和 evidence 两个字段
    
    // 序列化 attributes
    var attributesMap map[string]interface{}
    attrBytes, _ := json.Marshal(payload)
    _ = json.Unmarshal(attrBytes, &attributesMap)

    // 构造顶层 Map
    topLevelResult := map[string]interface{}{
        "attributes": attributesMap,
        "evidence": map[string]interface{}{
            "raw_output": nmapOutput.RawXML, // 示例：保留原始输出
            // "screenshots": ...
        },
    }

    // 4. 构造 TaskResult
    return &TaskResult{
        TaskID: task.ID,
        Status: "completed",
        Result: topLevelResult, // 此时 Result 严格符合契约 (attributes + evidence)
    }, nil
}
```

---

## 4. 校验与测试

### 4.1 单元测试校验
在 Executor 的单元测试中，应验证生成的 JSON 是否符合 Schema。

```go
func TestNmapExecutor_Output(t *testing.T) {
    // ... mock execution ...
    result, _ := executor.Execute(...)
    
    // 1. 验证顶层结构
    attributes, ok := result.Result["attributes"].(map[string]interface{})
    assert.True(t, ok, "attributes field missing or invalid")
    
    _, ok = result.Result["evidence"].(map[string]interface{})
    assert.True(t, ok, "evidence field missing or invalid")

    // 2. 验证 attributes 内容
    ports, ok := attributes["ports"].([]interface{})
    assert.True(t, ok)
    assert.NotEmpty(t, ports)
    
    firstPort := ports[0].(map[string]interface{})
    assert.Equal(t, 80, int(firstPort["port"].(float64))) // JSON unmarshal number 默认为 float64
}
```

### 4.2 契约测试
建议建立一个专门的 `contract_test.go`，引入 Master 端的契约定义（如果可能），或者硬编码契约 JSON，验证 DTO 的序列化结果是否与契约一致。

---

## 5. 常见问题规避

1.  **空数组问题**: Master 期望数组字段（如 `ports`）即使为空也应存在（`[]`），而不是 `null` 或字段缺失。初始化切片时请使用 `make([]T, 0)` 而非 `var s []T`。
2.  **时间格式**: 统一使用 RFC3339 (`2006-01-02T15:04:05Z07:00`)。
3.  **枚举值**: `state` 字段不要返回 `Open` 或 `OPEN`，必须是小写 `open`，与 Master 枚举对齐。

---

## 6. 总结

通过引入 **Payload DTO** 层，我们将：
1.  **编译时安全**: 字段名和类型的错误将在编译期被发现。
2.  **契约显性化**: 代码即文档，DTO 结构体直接反映了通信契约。
3.  **解耦**: Executor 的内部逻辑与 Master 的通信格式解耦，DTO 负责适配。
