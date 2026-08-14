# model — 核心层模型

> 核心层模型不应直接暴露给外部，必须通过通信层进行转换。
>
> 数据流向：核心层 → App 层 → 通信层

---

## 文件结构

```
model/
  ├── task.go             # 任务定义：TaskType、TaskStatus、Task、TaskResult
  ├── result_network.go   # 网络层扫描结果：IpAliveResult、PortServiceResult、OsInfo
  ├── result_web.go       # Web 扫描结果：WebResult、FormInfo、LeakInfo、EdgeComponent
  ├── result_api.go       # API 扫描结果：APIInfo、ApiResult
  └── result_vuln.go      # 攻击面结果：VulnResult、BruteResult、BruteResults
```

### 各文件职责边界

| 文件 | 包含类型 | 说明 |
|---|---|---|
| `task.go` | `TaskType`、`TaskStatus`、`Task`、`TaskResult` | 任务定义，所有扫描模块共用 |
| `result_network.go` | `IpAliveResult`、`PortServiceResult`、`OsInfo` | 同属网络层扫描产物，强相关 |
| `result_web.go` | `WebResult`、`FormInfo`、`LeakInfo`、`EdgeComponent` | `WebResult` 嵌套后三者，强耦合 |
| `result_api.go` | `APIInfo`、`ApiResult` | 与 `WebResult` 完全独立，不做指纹/截图/CDN识别 |
| `result_vuln.go` | `VulnResult`、`BruteResult`、`BruteResults` | 同属攻击面发现产物，`BruteResults` 是 `BruteResult` 的集合 |

---

## 变更记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v0.2 | 2026-08-14 | 将 `result_types.go`（308行）按业务域拆分为四个独立文件，同一 `package model` 无破坏性 |
| v0.1 | — | 初始结构：`task.go` + `result_types.go` |
