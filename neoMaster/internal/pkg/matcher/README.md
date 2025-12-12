# Matcher 通用匹配引擎

## 1. 简介
`matcher` 是一个高性能、无状态的通用规则匹配引擎。它负责评估一组条件（Condition）针对给定的数据（Data）是否成立。
该引擎仅负责 "布尔判定"（Match or Not Match），不涉及任何业务动作（如过滤、报警、阻断）。

## 2. 核心数据结构

为了支持复杂的嵌套逻辑（如 `(A AND B) OR (C AND D)`），我们采用递归结构设计。

### 2.1 MatchRule (匹配规则)
`MatchRule` 是规则树的基本单元，它既可以是单纯的**条件节点（Leaf）**，也可以是包含子规则的**逻辑节点（Branch）**。

```go
type MatchRule struct {
    // --- 逻辑节点 (Branch) ---
    // 逻辑节点包含子规则列表。
    // 如果设置了 And，则所有子规则都必须匹配。
    // 如果设置了 Or，则任意子规则匹配即可。
    // 注意：同一层级 And 和 Or 不应同时存在于同一个 Rule 对象中（如果同时存在，优先处理 And 或报错，视实现而定）。
    And []MatchRule `json:"and,omitempty"`
    Or  []MatchRule `json:"or,omitempty"`

    // --- 条件节点 (Leaf) ---
    // 当 And/Or 为空时，该节点被视为条件节点，必须包含以下字段：
    Field    string      `json:"field,omitempty"`    // 待匹配字段名 (支持点号访问嵌套字段，如 "meta.os")
    Operator string      `json:"operator,omitempty"` // 操作符
    Value    interface{} `json:"value,omitempty"`    // 目标值
}
```

## 3. 支持的操作符 (Operators)

目前支持 17 种操作符：

| 操作符 | 说明 | 适用类型 | 示例 |
| :--- | :--- | :--- | :--- |
| `equals` | 等于 | Any | `field == value` |
| `not_equals` | 不等于 | Any | `field != value` |
| `contains` | 包含 | String, List | `"hello" contains "he"` |
| `not_contains` | 不包含 | String, List | `"hello" !contains "x"` |
| `starts_with` | 以...开头 | String | `"server-01" starts_with "server"` |
| `ends_with` | 以...结尾 | String | `"image.png" ends_with ".png"` |
| `greater_than` | 大于 | Number | `count > 10` |
| `less_than` | 小于 | Number | `count < 10` |
| `greater_than_or_equal` | 大于等于 | Number | `count >= 10` |
| `less_than_or_equal` | 小于等于 | Number | `count <= 10` |
| `in` | 在列表中 | String/Number -> List | `"admin" in ["admin", "root"]` |
| `not_in` | 不在列表中 | String/Number -> List | `"guest" not_in ["admin", "root"]` |
| `is_null` | 为空 | Any | Field 不存在或值为 nil |
| `is_not_null` | 不为空 | Any | Field 存在且值不为 nil |
| `regex` | 正则匹配 | String | `ip regex "^192\.168\..*"` |
| `like` | 模糊匹配 | String | `name like "test_%"` (支持 % 和 _) |
| `exists` | 存在 | Any | 字段 Key 存在 (无论值是否为空) |
| `cidr` | IP网段匹配 | String (IP) | `"192.168.1.5" cidr "192.168.1.0/24"` |

## 4. 使用示例

### 4.1 JSON 配置示例

#### 基础示例
```json
{
  "and": [
    {
      "field": "device_type",
      "operator": "equals",
      "value": "honeypot"
    },
    {
      "field": "os",
      "operator": "contains",
      "value": "linux"
    }
  ]
}
```

#### 复杂嵌套示例 (混合逻辑)
支持在列表中混合使用 "条件节点" 和 "逻辑节点"。
```json
 { 
   "and": [{ 
     "field": "sourceProcessName", 
     "operator": "contains", 
     "value": "HaozipSvc.exe" 
   }, { 
     "and": [{ 
       "field": "destinationProcessName", 
       "operator": "equals", 
       "value": "C:\\Windows\\System32\\lsass.exe" 
     }] 
   }, { 
     "or": [{ 
       "field": "filePath", 
       "operator": "contains", 
       "value": "NT AUTHORITY\\SYSTEM" 
     }] 
   }] 
 }
```

### 4.2 Go 调用示例

```go
import "neomaster/internal/pkg/matcher"

// 1. 准备数据
data := map[string]interface{}{
    "device_type": "honeypot",
    "os":          "ubuntu linux",
    "port_count":  20,
    "port_open":   []int{80, 443},
}

// 2. 定义规则 (通常从 JSON 反序列化)
rule := matcher.MatchRule{
    And: []matcher.MatchRule{
        {Field: "device_type", Operator: "equals", Value: "honeypot"},
        {Field: "os", Operator: "contains", Value: "linux"},
        {
            Or: []matcher.MatchRule{
                {Field: "port_count", Operator: "greater_than", Value: 1000}, // False
                {Field: "port_open", Operator: "contains", Value: 80},       // True
            },
        },
    },
}

// 3. 执行匹配
matched, err := matcher.Match(data, rule)
if err != nil {
    // 处理错误
}

if matched {
    fmt.Println("Target matches the rules!")
}
```

## 5. 工作逻辑与流程
### 5.1 流程图
```mermaid
graph TD
    A["开始: Match(data, rule)"] --> B{"检查节点类型"}

    B -->|"逻辑节点(AND)"| C["遍历 And 子规则列表"]
    C --> D{"评估子规则 Match(data, subRule)"}
    D -->|"失败 (err != nil)"| E["返回 false, err"]
    D -->|"不匹配 (false)"| F["返回 false (AND 短路)"]
    D -->|"匹配 (true)"| G{"是否还有下一个子规则?"}
    G -->|"是"| C
    G -->|"否"| H["返回 true (AND 全部匹配)"]

    B -->|"逻辑节点(OR)"| I["遍历 Or 子规则列表"]
    I --> J{"评估子规则 Match(data, subRule)"}
    J -->|"失败 (err != nil)"| K["返回 false, err"]
    J -->|"匹配 (true)"| L["返回 true (OR 短路)"]
    J -->|"不匹配 (false)"| M{"是否还有下一个子规则?"}
    M -->|"是"| I
    M -->|"否"| N["返回 false (OR 全部不匹配)"]

    B -->|"条件节点(Leaf)"| O["获取字段值 GetValue(data, field)"]
    O --> P{"检查获取结果"}
    P -->|"字段不存在 / 类型错误"| Q["返回 false (Fail Safe)"]
    P -->|"获取成功"| R["执行操作符比较 evaluateCondition"]
    R --> S{"比较结果"}
    S -->|"错误 (err != nil)"| T["返回 false, err"]
    S -->|"成功"| U["返回 true / false"]
```

### 5.2 核心流程说明
1.  **递归评估**：
    - 引擎首先判断当前规则是**逻辑节点**（And/Or）还是**条件节点**（Leaf）。
    - 如果是逻辑节点，则递归调用 `Match` 函数处理子规则。
    - 如果是条件节点，则直接执行字段值提取和比较。

2.  **短路逻辑 (Short-circuit)**：
    - **AND**：一旦遇到一个子规则不匹配（false），立即返回 false，不再计算后续规则。
    - **OR**：一旦遇到一个子规则匹配（true），立即返回 true，不再计算后续规则。
    - 这种机制在处理复杂规则时能显著提高性能。

3.  **字段值提取 (Field Extraction)**：
    - 支持点号（`.`）语法访问嵌套字段（如 `meta.os`）。
    - 能够自动处理 `map[string]interface{}` 和 `struct` 类型。
    - 如果路径中的任何中间节点不存在或类型不符，视为字段不存在。

4.  **类型兼容性**：
    - 数值比较时，引擎会自动处理不同数值类型（如 `int` vs `float64`）以及字符串形式的数字（如 `"100"` vs `100`）。
    - 确保比较操作尽可能符合直觉，减少因类型不匹配导致的误报。

## 6. 设计原则
1.  **Fail Safe**: 如果字段不存在或类型不匹配，默认返回 false 并报错（可配置 Strict Mode）。
2.  **Stateless**: 引擎本身不存储任何状态，完全由输入决定输出。
3.  **Recursion**: 支持任意深度的嵌套逻辑。
4.  **Performance**: 针对高频操作符进行优化。
