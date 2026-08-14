# 核心层模型存放处

核心层模型存放处，这些模型不应该直接暴露给外部，而是通过通信层进行转换。

核心层 > App 层 > 通信层

---

## 目录结构（当前 vs 规划）

### 当前结构

```
model/
  ├── task.go           # 任务定义：TaskType、TaskStatus、Task、TaskResult
  └── result_types.go   # 所有扫描结果类型（8种，308行，持续膨胀中）
```

### 规划结构（差分拆分方案）

```
model/
  ├── task.go           # 不动，任务定义保持原位
  ├── result_network.go # IpAliveResult、PortServiceResult、OsInfo
  ├── result_web.go     # WebResult、FormInfo、LeakInfo、EdgeComponent
  ├── result_api.go     # APIInfo、ApiResult
  ├── result_vuln.go    # VulnResult、BruteResult、BruteResults
  └── result_types.go   # 拆分完成后删除此文件
```

### 各文件职责边界

| 文件 | 包含类型 | 拆分理由 |
|---|---|---|
| `result_network.go` | `IpAliveResult`、`PortServiceResult`、`OsInfo` | 同属网络层扫描产物，强相关 |
| `result_web.go` | `WebResult`、`FormInfo`、`LeakInfo`、`EdgeComponent` | `WebResult` 嵌套后三者，强耦合 |
| `result_api.go` | `APIInfo`、`ApiResult` | 注释已声明"与 WebResult 完全独立"，天然独立文件 |
| `result_vuln.go` | `VulnResult`、`BruteResult`、`BruteResults` | 同属攻击面发现产物，`BruteResults` 是 `BruteResult` 的集合 |

### 重构注意事项

- 所有文件均在同一 `package model` 下，**调用方零改动**，无破坏性
- `result_types.go` 在四个文件全部迁移完成后统一删除，不要分批删
- 每次只迁移一个文件对应的类型，迁完即可编译验证，降低出错风险