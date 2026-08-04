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

*   **这不是 Web 扫描器的个例，是全部原子扫描能力共同的技术债（2026-08-04 补充调研）**：逐一核对了 `internal/core/runner/interface.go` 定义的统一 `Runner` 接口——

    ```go
    type Runner interface {
        Name() model.TaskType
        Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error)
    }
    ```

    `alive`/`port_service`/`os`/`web`/`brute`/`vuln`/`dir`/`subdomain` 全部 8 个 Scanner 无一例外实现的都是"一次调用、同步阻塞、跑完才返回全量 slice"这一个契约。再核对 `cmd/agent/scan/` 下 `web.go`/`port_service.go`/`os.go`/`brute.go`/`alive.go` 五个已接入 Console 输出的子命令，**全部调用的是 `ConsoleReporter.PrintResults(results)` 这个"先聚合、再一次性渲染"的批量 helper，没有一个用到 `Reporter.Report(ctx, result)` 这个本来就是单条设计的流式接口**。也就是说：
    1. `Reporter.Report` 接口层面早就具备流式能力，只是从未被任何 CLI 命令实际使用；
    2. 真正的病根在更底层——`Runner.Run()` 本身就是同步阻塞契约，只要它不返回，`cmd/agent/scan/*.go` 手里就没有数据可以提前打印，输出层怎么改都是治标不治本；
    3. 因此"控制台无流式输出"不是 Web 模块孤立的 bug，而是贯穿全部 8 个原子扫描能力的架构级技术债，根治需要动 `Runner` 接口这个被大范围复用的核心抽象，波及 `AutoRunner`（全流程编排）、`Dispatcher`（Master 集群任务分发）、全部 Scanner 实现方，影响面是全局性的，不适合在推进具体原子能力开发的过程中顺手改掉。

*   **优先级决策**：明确将这项改造列为**独立的架构级任务，推迟到当前规划内的各原子能力（Phase 5.2 Vuln Scanner、Phase 5.3 Dir/Subdomain Scanner 等）开发完成之后再统一解决**，不在推进具体模块开发的过程中顺带"顺手"改掉。原因：
    1. 这是一个跨越全部 Scanner 的地基级改动，一旦启动就必须一次性想清楚新接口长什么样、怎么兼容 Master 集群上报协议、怎么兼容 `AutoRunner` 依赖"全量结果做漏斗判断"的现有逻辑，动到一半搁置的代价很高；
    2. 等 Phase 5.2/5.3 落地后，`Runner` 会有更多实现方（Vuln/Dir/Subdomain），到时候一次性设计流式契约，能覆盖到的场景更完整，不用等新模块上线后再补一轮改造；
    3. 当前的体验缺陷是"CLI 单点调试时要多等一会儿"，不是数据错误或功能缺失，严重程度可以接受延后处理；短期内如果某个具体模块（如 Web 扫描）确实等不了，可以先用不改 `Runner` 接口的局部方案缓解（见下方临时方案），不需要为了这一个体验优化提前打乱整体开发节奏。
    4. **临时缓解方案（不改 `Runner` 接口，影响面仅限单个 Scanner）**：给需要进度反馈的 Scanner（如 `WebScanner`）新增一个可选的进度回调字段（如 `SetProgressCallback`），零值 `nil` 安全、不影响任何现有调用方；`Run()` 内部在汇总结果的循环里判空调用一下即可。这不是"根治"，只是在正式收口 `Runner` 接口之前，给个别等不及的场景开的一个小口子，`Reporter`/`ConsoleReporter`/`PrintResults`/其余 7 个 Scanner 都不需要动。

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
6.  [ ] **【根治方案，排期在 Phase 5.2/5.3 之后】重新设计 `Runner` 接口支持流式/回调式返回**，让全部 8 个原子扫描能力（而不只是 Web）都具备"边扫边输出"的能力，同步梳理 `AutoRunner`/`Dispatcher`/Master 集群上报路径如何适配新契约。这是本文件当前记录的最高优先级架构级技术债，具体决策依据见上方"这不是 Web 扫描器的个例"与"优先级决策"两节。
