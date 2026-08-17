# Brute Scanner Module

- **版本**: v1.1
- **日期**: 2026-08-04

## 简介
本模块是 NeoAgent 的核心扫描能力之一，负责执行**弱口令爆破**任务。
它设计为一个高并发、可扩展、协议无关的框架，支持通过插件方式轻松添加新的协议支持。

## 核心设计
*   **接口抽象 (`cracker.go`)**: 定义了 `Cracker` 接口，所有具体的协议爆破逻辑都只需实现 `Check` 方法。
*   **统一调度 (`scanner.go`)**: `BruteScanner` 负责任务分发、并发控制 (基于 QoS AdaptiveLimiter)、字典生成和结果收集。
*   **字典管理 (`dict.go`)**: 支持内置 Top100 弱口令、用户自定义字典以及基于规则的动态字典生成。

## 目录结构说明

| 文件/目录 | 说明 |
| :--- | :--- |
| `scanner.go` | **核心调度器**。实现了 `Runner` 接口（`Name()`/`Run()`），负责管理并发、调用 Cracker、处理结果。 |
| `cracker.go` | **接口定义**。定义了 `Cracker` 接口和 `AuthMode` (UserPass/OnlyPass/None)。 |
| `dict.go` | **字典管理器**。负责组合用户名和密码，生成待测试的凭据列表。 |
| `protocol/` | **协议实现层**。包含所有具体协议的 Cracker 实现（当前 15 种，见下文）。 |
| `DESIGN.md` | **设计文档**。详细记录了模块的架构设计、数据流和接口规范。 |
| `*_test.go` | 单元测试文件。 |

## 支持的协议
目前已内置支持以下协议的弱口令爆破：
*   **系统服务**: SSH, RDP, SMB, Telnet, FTP, SNMP
*   **数据库**: MySQL, PostgreSQL, MSSQL, Oracle（含 Oracle SID 字典爆破）, MongoDB, Redis, ClickHouse, Elasticsearch

## 如何添加新协议支持
1.  在 `protocol/` 目录下创建一个新的 `.go` 文件 (e.g., `protocol/pop3.go`)。
2.  实现 `brute.Cracker` 接口：
    *   `Name()`: 返回协议名称 (e.g., "pop3")。
    *   `Mode()`: 返回认证模式 (e.g., `AuthModeUserPass`)。
    *   `Check()`: 实现具体的连接和认证逻辑。
3.  **在 `internal/core/factory/brute_factory.go` 的 `NewFullBruteScanner()` 中注册新的 Cracker**（这是唯一正确的注册位置）：
    ```go
    s.RegisterCracker(protocol.NewPOP3Cracker())
    ```
    > ⚠️ 注意：不要在 `internal/core/runner/manager.go` 中直接调用 `RegisterCracker`。
    > `manager.go` 只会调用 `factory.NewFullBruteScanner()` 拿到一个已注册好全部协议的 `BruteScanner` 实例，
    > 本身不感知具体协议。CLI（`neoAgent scan brute`）和 Master（`brute_force` TaskType）两条入口
    > 共用这一份工厂产出的实例，只在 `brute_factory.go` 一处注册可以保证两条入口的能力集合永远一致；
    > 如果在别处也调用 `RegisterCracker`，会造成"CLI 能用、Master 不能用"或反之的不一致问题。
4.  如需在 CLI 侧暴露默认端口等参数，同步更新 `internal/core/options/scan_brute.go` 中的 `defaultBrutePorts` 表。

## 依赖说明
本模块尽量减少外部依赖：
*   大部分协议使用 Go 标准库或成熟的开源驱动 (如 `go-sql-driver/mysql`, `lib/pq`, `crypto/ssh`)。
*   RDP 协议使用内部移植的纯 Go 实现 (`protocol/rdp/`)，无需 CGO。

## ⚠️ 已知问题：`rules/passwords/` 字典目录当前未生效

`neoAgent/rules/passwords/` 下存放了按协议分类的用户名/密码字典文件（`ssh/user.txt`、`mysql/pass.txt` 等共 13 个协议目录），但**当前代码完全没有读取这些文件**。

`dict.go` 的字典来源是硬编码在 Go 源码里的两个数组：

```go
var DefaultTopUsers = []string{"root", "admin", "user", ...}
var DefaultTopPasswords = []string{"123456", "password", ...}
```

用户只能通过 Task `Params["users"]`/`Params["passwords"]` 传参来覆盖，无法直接通过修改 `rules/passwords/` 下的文件来影响行为。

**后续优化方向**：将字典外部化，让 `rules/passwords/` 成为真正的可配置字典源：
- 启动时扫描 `rules/passwords/<protocol>/user.txt` 和 `pass.txt`，按协议加载对应字典
- 加载逻辑参考 `rules/fingerprint/web/web_fingerprints.json` 的多路径 fallback 范式（`os.ReadFile` 运行时加载，免重新编译）
- 内置 Top 字典作为兜底默认值，外部文件存在时优先使用

详见 `rules/README.md` 第三节"后续优化方向"。

## 相关文档
*   [`DESIGN.md`](./DESIGN.md)：架构设计、数据流、调度器逻辑与开发进度。
*   [`../../../../docs/Agent指令集规范.md`](../../../../docs/Agent指令集规范.md)：3.5 节记录了 `brute_force` 指令的完整 CLI/Task 参数映射。
*   [`../../../../rules/README.md`](../../../../rules/README.md)：记录了 `rules/` 目录下所有规则文件的实际调用状态，含 `passwords/` 当前未生效的详细说明。
