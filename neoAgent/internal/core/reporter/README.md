# 结果上报与输出 (Reporter)

本模块负责将扫描任务的结果 (`model.TaskResult`) 输出到不同的目的地（控制台、文件、Master 节点等）。

## 当前状态 (Phase 3)

目前的实现处于 **"新老交替"** 的过渡阶段，存在一定的代码不一致性，这是为了快速推进原子能力建设而做出的妥协。

### 1. 存在的组件
*   **`interface.go`**: 定义了 `Reporter` 接口和 `TabularData` 接口。
*   **`console.go`**: 实现了 `ConsoleReporter`，用于在终端打印漂亮的表格。
*   **`csv.go`**: 提供了 `SaveCsvResult` 静态方法，用于导出 CSV 文件。

### 2. 遗留问题 (Technical Debt)
在 `cmd/agent/scan/` 下的各个子命令中，输出逻辑并不统一：
*   **JSON 输出**: 直接使用了本地定义的 helper 函数 `saveJsonResult` (位于 `alive.go`)。
*   **CSV 输出**: 调用了 `reporter.SaveCsvResult`。
*   **Console 输出**: 调用了 `reporter.NewConsoleReporter()`。

这种分散的调用方式不符合 "Good Taste"，但在现阶段避免了过度封装带来的复杂性。

*   **控制台无流式进度输出（2026-08-04 发现）**：`ConsoleReporter.PrintResults` 只能"给全量数据、画一张完整表"，不支持逐条追加渲染。而 `WebScanner.Run()` 内部用 `sync.WaitGroup` 并发扫描所有端口，`wg.Wait()` 阻塞到全部端口（含每个端口各自同步执行的 BFS 深度爬取）都完成后才 `return`，`cmd/agent/scan/web.go` 才能拿到完整结果调用 `PrintResults` 渲染一次表格。两者叠加的结果：`--crawl=true` 时用户会经历一段长时间的黑屏等待，然后表格整个"跳出来"，中途没有任何逐行反馈。根因是整条链路（`Run` → `runOnePort` → `PrintResults`）压根不存在流式/增量的结果通道，不是渲染卡顿导致的观感问题。

## 未来规划 (Phase 4: Orchestration)

在 Phase 4 (全流程编排) 阶段，我们将引入统一的 **Output Pipeline** 来偿还技术债务。

### 重构目标
建立一个统一配置驱动的 `OutputManager`，彻底解耦 Scanner 和 Output。

### 设计草图

```go
// 统一配置
type OutputConfig struct {
    JSONFile string // --oj
    CSVFile  string // --oc
    Console  bool   // default true
    Webhook  string // --webhook
}

// 统一调用
func (m *OutputManager) Handle(result *model.TaskResult) {
    if m.config.JSONFile != "" {
        m.jsonReporter.Report(result)
    }
    if m.config.CSVFile != "" {
        m.csvReporter.Report(result)
    }
    // ...
}
```

### 待办事项
1.  [ ] 实现 `JsonReporter` (替代 `saveJsonResult`)。
2.  [ ] 实现 `FileReporter` (支持 TXT/HTML 等其他格式)。
3.  [ ] 实现 `OutputManager` 或 `MultiReporter` 的高级封装，支持流式写入和文件轮转。
4.  [ ] 将 `cmd/agent/scan/*.go` 中的输出逻辑全部收敛到 `OutputManager`。
5.  [ ] `ConsoleReporter` 补充流式/增量渲染能力（如按 `*model.TaskResult` 逐条 append 一行，而非等全量结果collect 完再 `Render()` 一次），前提是 `WebScanner.Run()` 等 Scanner 先对外暴露单条结果的回调/channel（当前 `Run()` 是同步阻塞到全部完成才返回，见上方"遗留问题"）。
