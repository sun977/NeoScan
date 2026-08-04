# 日志模块 (Logger)

基于 `logrus` 封装的全局日志管理器，供 Agent 全部模块（CLI 扫描命令、Server 模式、Core Scanner 等）统一调用。

## 1. 核心组件

*   **`logger.go`**：`LoggerManager` 定义与初始化（`InitLogger`/`UpdateConfig`），以及一组可直接调用的全局便捷方法（`Debug/Info/Warn/Error/Fatal` 及其 `xxxf` 格式化版本、`WithField/WithFields`）。
*   **`formatter.go`**：自定义时间戳格式化、`LogType` 分类枚举，以及一批针对具体业务场景的结构化日志方法（`LogAccessRequest`/`LogBusinessOperation`/`LogSystemEvent`/`LogScanOperation` 等）。

## 2. 初始化

任何模块使用日志前，必须先调用一次 `InitLogger` 完成初始化，未初始化时全局便捷方法（`logger.Info` 等）会静默跳过（`LoggerInstance == nil` 直接 return，不会 panic，但也不会输出任何内容）：

```go
cfg := &config.LogConfig{
    Level:  "info",   // debug/info/warn/error/fatal，非法值会退化为 info 并打一条 Warn
    Format: "text",   // text（彩色，适合终端）/ json（适合日志采集）
    Output: "stdout",  // stdout / stderr / file
    Caller: false,     // 是否在日志里附带调用者文件名+行号
}
if _, err := logger.InitLogger(cfg); err != nil {
    // 初始化失败：多半是 Format/Output 传了非法值，或 Output=file 时 FilePath 为空
}
```

CLI 场景（`cmd/agent/root.go` 的 `initCLILogger`）在此基础上做了一层映射：`--log-level` 未显式传入时默认使用 `"fatal"`（避免单机扫描时刷屏），显式传入才使用用户指定的级别；同时会联动设置 `pterm` 的 `EnableDebugMessages()/DisableDebugMessages()`，用于控制 `pterm` 侧（区别于 logrus）自身的 Debug 输出开关。

## 3. 输出到文件与日志轮转

`Output: "file"` 时必须提供 `FilePath`，日志轮转基于 `lumberjack`，支持 `MaxSize`(MB)/`MaxBackups`/`MaxAge`(天)/`Compress`。有一个隐藏行为需要注意：**`Level: "debug"` 时会自动同时输出到 `os.Stdout` 和文件**（`io.MultiWriter`），其他级别只写文件，这是为了方便开发环境调试，配置文件中不需要额外开关。

## 4. 日常调用方式

绝大多数场景直接用包级别便捷函数即可，不需要关心 `LoggerInstance` 内部细节：

```go
logger.Debugf("[WebScanner] fetch %s cost=%v", url, elapsed)
logger.Infof("[WebScanner] Loaded %d fingerprint rules from %s", len(rules), path)
logger.Warnf("[WebScanner] Screenshot failed: %v", err)
logger.Errorf("[WebScanner] PANIC RECOVERED: %v", err)
```

需要附加结构化字段（会体现在 JSON 格式输出里，text 格式下以 `key=value` 追加在行尾）时用 `WithField`/`WithFields`：

```go
logger.WithFields(logrus.Fields{"task_id": taskID, "target": target}).Info("scan started")
```

`formatter.go` 里的一批 `LogXxx` 函数（`LogAccessRequest`/`LogBusinessOperation`/`LogAuditOperation`/`LogScanOperation` 等）是更"重"的结构化封装，各自绑定固定的字段 Schema 和 `LogType`，主要给需要统一日志格式做采集分析的场景使用（例如 HTTP 访问日志、审计日志）；如果只是想在业务代码里打一行排查用的日志，直接用上面的包级别便捷函数即可，不必套用这些封装。

## 5. 已知问题 (2026-08-04 发现)

### 5.1 `--log-level=debug` 开关本身生效，但业务代码普遍没有埋 Debug 日志

以 `neoAgent scan web --log-level=debug` 为例，实测跑完整个扫描（含 `--crawl=true`）不会输出任何一条 `DEBU[...]` 日志。**排查结论：这不是日志模块的 bug**，`InitLogger` 对 `Level: "debug"` 的处理是正确的，`logrus.SetLevel(logrus.DebugLevel)` 生效后 `logger.Debug/Debugf` 是能正常打印的。真正的原因是**核心扫描逻辑压根没有调用 `logger.Debugf`**——以 `internal/core/scanner/web/web_scanner.go` 为例，全文只有 `Infof/Warnf/Errorf` 调用，没有一处 `Debugf`。也就是说日志开关这把锁是开着的，但门后面根本没放东西。

同类问题在别的 Scanner 包里大概率也存在，不是 `web` 包独有。

**待办**：在各 Scanner 的关键路径（尤其是 `web_scanner.go` 的 `runOnePort`：协议猜测/http-https 双发选优过程、go-rod 各阶段耗时、BFS 逐页爬取进度、指纹匹配细节）补充 `logger.Debugf` 埋点，让 `--log-level=debug` 真正能输出有诊断价值的信息，而不是一个不会触发任何效果的空开关。

### 5.2 CLI 场景下 `pterm` 与 `logrus` 是两条独立且脱节的日志通道

`cmd/agent/root.go` 的 `initCLILogger` 会同时操作 `logrus`（通过 `InitLogger`）和 `pterm`（`EnableDebugMessages`/`DisableDebugMessages`/`Info.WithWriter(io.Discard)`），两者互不感知。目前 `web_scanner.go` 等核心扫描代码只调用 `logger.Xxx`（logrus 侧），从未调用过 `pterm.Debug` 之类的方法，因此 `initCLILogger` 里针对 `pterm` 的 debug 开关代码目前是不会被触发的死代码路径。如果后续要在 CLI 展示层（区别于日志采集层）加彩色的调试提示，需要明确这两套机制各自的职责边界，避免继续两头各写一半、谁都没真正用起来。
