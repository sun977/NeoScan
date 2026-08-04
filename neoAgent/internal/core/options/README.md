# 任务参数解析与校验 (options)

- **版本**: v1.0
- **日期**: 2026-08-04

## 作用

`options` 包是 CLI 与核心任务模型 (`internal/core/model.Task`) 之间的**唯一合法转换层**。

每一种扫描能力都对应一个 `XxxOptions` 结构体，统一实现 `types.go` 中定义的 `TaskOption` 接口：

```go
type TaskOption interface {
	Validate() error      // 校验参数合法性（含默认值兜底、必填项检查）
	ToTask() *model.Task  // 转换为核心任务模型，供 RunnerManager 执行
}
```

存在这一层的意义：
*   **类型安全**：CLI Flag 直接绑定到结构体字段，避免 Runner 侧用 `map[string]interface{}` 做类型断言时的隐式失败。
*   **默认值集中管理**：所有默认值（如端口、并发数、超时时间）都在 `NewXxxOptions()`/`Validate()` 中定义一次，不会散落在 CLI 命令代码里。
*   **CLI 与 Master 行为对齐**：`ToTask()` 产出的都是同一个 `model.Task`，无论指令来自命令行还是 Master 下发，最终都走 `RunnerManager.Execute()` 这一条路径。

## 目录结构

| 文件 | 对应指令 | TaskType | 说明 |
| :--- | :--- | :--- | :--- |
| `types.go` | - | - | 定义 `TaskOption` 接口，所有 `XxxOptions` 的公共契约 |
| `output.go` | 通用 | - | `OutputOptions`，定义 `-oc`/`-oj` 等结果输出参数，供各 Options 组合复用 |
| `scan_alive.go` | `scan alive` | `ip_alive_scan` | IP 存活扫描（ICMP/ARP/TCP） |
| `scan_port_service.go` | `scan port` | `port_scan` | 端口扫描 + 可选深度服务识别 |
| `scan_os.go` | `scan os` | `os_scan` | 操作系统识别 |
| `scan_web.go` | `scan web` | `web_scan` | Web 综合扫描（指纹/爬取/截图） |
| `scan_brute.go` | `scan brute` | `brute_force` | 弱口令爆破，含按 `service` 推断默认端口的逻辑 |
| `scan_dir.go` | `scan dir` | `dir_scan` | 目录扫描参数定义（⚠️ 底层 `DirScanner` 未实现，见下文） |
| `scan_vuln.go` | `scan vuln` | `vuln_scan` | 漏洞扫描参数定义（⚠️ 底层 `VulnScanner` 未实现，见下文） |
| `scan_subdomain.go` | `scan subdomain` | `subdomain` | 子域名扫描参数定义（⚠️ 底层 `SubdomainScanner` 未实现，见下文） |
| `proxy.go` | `neoAgent proxy` | `proxy` | 代理/端口转发参数定义（⚠️ 底层 Runner 未接入，见下文） |
| `scan_run.go` | `scan run` | - | `ScanRunOptions`，**不实现 `TaskOption` 接口**，是自动化全流程扫描（Pipeline）专用的参数容器，一次运行内部会编排出多个子任务，因此没有单一的 `ToTask()` |

> ⚠️ **未实现能力的提示**：`scan_dir.go`/`scan_vuln.go`/`scan_subdomain.go`/`proxy.go` 中的 `Options` 结构体和 `Validate()` 逻辑是完整的，但对应的 Scanner/Runner 尚未实现，CLI 命令会在参数校验通过后直接报错 `xxx not implemented yet`（`proxy` 命令目前只会打印 Task JSON）。完整的实现状态与参数表格见 [`docs/Agent指令集规范.md`](../../../docs/Agent指令集规范.md) 第 3 章。

## 如何使用（新增一个扫描指令的标准流程）

以新增 `scan_xxx.go` 为例：

1.  **定义结构体**：字段与 CLI Flag 一一对应，只保留该指令需要的参数。
2.  **实现 `NewXxxOptions()`**：给出合理默认值（超时、并发数、模式等）。
3.  **实现 `Validate() error`**：校验必填项，需要"缺省值推断"的逻辑（如 `scan_brute.go` 按 `service` 推断默认端口）也放在这里，而不是放在 CLI 命令或 `ToTask()` 里。
4.  **实现 `ToTask() *model.Task`**：调用 `model.NewTask(TaskType, target)`，把字段写入 `task.Params`；如需支持结果输出，调用 `o.Output.ApplyToParams(task.Params)`。
5.  **CLI 侧接入**（`cmd/agent/scan/xxx.go`）：
    ```go
    opts := options.NewXxxOptions()
    // ... 绑定 Flags 到 opts 字段 ...
    if err := opts.Validate(); err != nil {
        return err
    }
    task := opts.ToTask()
    manager := runner.NewRunnerManager()
    results, err := manager.Execute(ctx, task)
    ```

更完整的接入流程（含 Scanner 实现、Factory 注册、Runner 注册）参见 [`docs/原子扫描器开发指南[定稿].md`](../../../docs/原子扫描器开发指南[定稿].md)。

## 相关文档
*   [`docs/Agent指令集规范.md`](../../../docs/Agent指令集规范.md)：每个 TaskType 的完整参数表格与实现状态。
*   [`docs/原子扫描器开发指南[定稿].md`](../../../docs/原子扫描器开发指南[定稿].md)：新增原子扫描能力的完整 5 步流程。
