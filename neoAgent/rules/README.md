# rules 目录调用现状说明

> 最后更新：2026-08-03（commit `0e8a82bc`）
> 维护人：请在每次改动 `rules/` 目录结构或对应加载逻辑后，同步更新本文档的"最后更新"信息。

本文档记录 `neoAgent/rules/` 下各规则文件/目录**是否**、以及**如何**被 agent 侧代码实际调用，避免出现"改了规则文件但线上行为毫无变化"的排查陷阱。

## 一、调用现状总表

| 规则路径 | 是否被调用 | 调用方式 | 关键代码位置 |
|---|:---:|---|---|
| `fingerprint/web/web_fingerprints.json` | 🟢 是 | 运行时 `os.ReadFile`，多路径 fallback 依次尝试，**支持免编译热更新** | `internal/core/scanner/web/web_scanner.go` `ensureInit()` |
| `fingerprint/nmap/nmap-os-db` | 🟡 间接 | **不是**直接读取此文件；真正生效的是 `internal/core/scanner/os/nmap_stack/nmap-os-db` 的物理拷贝，经 `go:embed` 编译进二进制 | `internal/core/scanner/os/nmap_stack/rules.go` |
| `fingerprint/nmap/nmap-service-probes.bak` | 🟡 间接 | 同上，真正生效的是 `internal/core/scanner/port_service/nmap_service/nmap-service-probes` 的物理拷贝，经 `go:embed` 编译进二进制 | `internal/core/scanner/port_service/nmap_service/rules.go` |
| `fingerprint/nmap/nmap-services` | 🔴 否 | 无任何代码读取；Top100 端口表是硬编码整数数组 | `internal/core/scanner/port_service/nmap_service/port_lists.go` |
| `passwords/**/*.txt`（mysql/ssh/redis/mongodb/... 共 13 个协议目录） | 🔴 否 | 无任何代码读取；弱口令字典硬编码在 Go 源码里 | `internal/core/scanner/brute/dict.go`（`DefaultTopUsers` / `DefaultTopPasswords`） |
| `edge/cdn.json` | 🔴 否 | 无任何代码读取，仅停留在方案文档设计阶段 | 参见 `docs/爬虫/Web扫描CDN识别实施文档.md` |
| `poc/poc文件.md` | 🔴 否 | 空文件，无独立 POC 执行引擎消费此目录 | — |

图例：🟢 直接生效　🟡 间接生效（存在同步风险）　🔴 完全无效（死文件）

## 二、各项细节

### 2.1 🟢 `fingerprint/web/web_fingerprints.json`

`WebScanner.ensureInit()` 启动时按以下顺序尝试读取，命中第一个存在的文件即生效：

```549:557:internal/core/scanner/web/web_scanner.go
paths := []string{
    "rules/fingerprint/web/web_fingerprints.json",
    "../rules/fingerprint/web/web_fingerprints.json",
    "../../rules/fingerprint/web/web_fingerprints.json",
    "../../../rules/fingerprint/web/web_fingerprints.json",
    "../../../../rules/fingerprint/web/web_fingerprints.json",
    "../../../../../rules/fingerprint/web/web_fingerprints.json", // For unit tests in internal/core/scanner/web
    "neoAgent/rules/fingerprint/web/web_fingerprints.json",
}
```

加载后通过 `s.fpEngine.Reload(rules)` 注入指纹匹配引擎。单元测试 `web_scanner_test.go` 显式依赖此文件被正确加载（断言 Nginx/PHP 指纹可识别）。**这是目前唯一真正做到"改配置不用重新编译"的规则文件。**

### 2.2 🟡 `fingerprint/nmap/nmap-os-db` 与 `nmap-service-probes.bak`

这两个文件在 `internal/` 下各有一份**物理拷贝**（文件名去掉了 `.bak` 后缀），通过 `go:embed` 编译进二进制：

```1:6:internal/core/scanner/os/nmap_stack/rules.go
package nmap_stack

import _ "embed"

//go:embed nmap-os-db
var NmapOSDB string
```

```1:6:internal/core/scanner/port_service/nmap_service/rules.go
package nmap_service

import _ "embed"

//go:embed nmap-service-probes
var NmapServiceProbes string
```

**风险**：`go:embed` 不支持 `../` 跨目录引用，因此这两份数据库文件必须物理复制到各自 package 目录下才能被 embed。这意味着 `rules/fingerprint/nmap/` 下的原始文件与 `internal/.../nmap_stack/`、`internal/.../nmap_service/` 下的 embed 副本是**三份独立文件，没有任何自动同步机制**。修改 `rules/fingerprint/nmap/nmap-os-db` **不会**对运行时产生任何影响，必须同步修改 embed 副本并重新编译才能生效。

此外，`port_service_scanner.go` 中还留有一段运行时 fallback 逻辑：

```54:59:internal/core/scanner/port_service/port_service_scanner.go
// Fallback: 尝试加载外部规则文件 (仅开发环境或特殊配置)
paths := []string{
    "rules/fingerprint/nmap-service-probes",
    "../rules/fingerprint/nmap-service-probes",
    "../../rules/fingerprint/nmap-service-probes", // 针对 test 目录
}
```

这段代码是**死代码**：
1. `embed` 分支永远优先命中并直接 `return`，fallback 分支实际不可达；
2. 即便可达，路径 `rules/fingerprint/nmap-service-probes` 本身也是错的——仓库里实际路径是 `rules/fingerprint/nmap/nmap-service-probes.bak`（多一层 `nmap/` 子目录，且带 `.bak` 后缀），这个路径在仓库历史上从未存在过对应文件。

### 2.3 🔴 `fingerprint/nmap/nmap-services`

无任何代码读取。Top100 端口列表在 `port_lists.go` 中手写硬编码：

```1:6:internal/core/scanner/port_service/nmap_service/port_lists.go
package nmap_service

// Nmap Top 100 Ports (Based on Nmap's nmap-services frequency)
var Top100Ports = []int{
    80, 23, 443, 21, 22, 25, 3389, 110, 445, 139,
    ...
```

### 2.4 🔴 `passwords/**`

爆破模块 `internal/core/scanner/brute/dict.go` 的字典来源是硬编码常量，用户可通过任务参数 `users`/`passwords` 覆盖，但代码里**从未**读取过 `passwords/` 目录下任何一个 `pass.txt`/`user.txt` 文件：

```7:19:internal/core/scanner/brute/dict.go
var DefaultTopUsers = []string{
    "root", "admin", "user", "test", "guest",
    "postgres", "mysql", "oracle", "weblogic",
    "administrator", "service", "system",
}
var DefaultTopPasswords = []string{
    "123456", "password", "12345678", "123456789", "12345", "123",
    "root", "admin", "test", "111111", "1234567",
    "%user%", "%user%123", "%user%@123", "123%user%",
}
```

`passwords/` 下 clickhouse、elasticsearch、ftp、memcached、mongodb、mssql、mysql、oracle、postgres、rdp、redis、smb、snmp、ssh 共 13 个协议子目录，全部是孤儿文件。

### 2.5 🔴 `edge/cdn.json`

仅在 `docs/爬虫/Web扫描CDN识别实施文档.md`、`Web扫描CDN识别方案.md` 中作为设计方案被提及，`internal/` 下没有任何代码读取此文件。CDN 识别能力尚未落地。

### 2.6 🔴 `poc/poc文件.md`

空文件。`internal/` 中搜索 "poc" 仅在个别 model 字段名（如 `internal/model/adapter/vuln.go`）中出现，没有独立的 POC 加载/执行引擎。

## 三、后续优化方向

1. **消灭"看得见摸不着"的规则文件**：`passwords/**`、`edge/cdn.json`、`poc/poc文件.md` 三类文件目前 100% 不生效，属于误导性的死文件。二选一：
   - 要么尽快把对应能力（字典外部化加载、CDN 识别、POC 引擎）实现出来，让目录名副其实；
   - 要么在能力真正排期落地之前直接删除，避免维护者误以为改文件就能改行为，浪费排查时间。

2. **打通 nmap 规则的单一数据源**：现状是 `rules/fingerprint/nmap/` 原始文件与两个 `internal/.../rules.go` 里的 embed 副本三份数据不同步。建议二选一：
   - 在构建流程（`Makefile`/CI）中增加一步，编译前自动从 `rules/fingerprint/nmap/` 同步覆盖到各 embed 目录，保证 `rules/` 才是唯一可信源；
   - 或者干脆删除 `rules/fingerprint/nmap/` 下的原始文件，在其 README 中直接注明"真实数据源见 `internal/core/scanner/os/nmap_stack/nmap-os-db` 与 `internal/core/scanner/port_service/nmap_service/nmap-service-probes`"，避免留两份造成混淆。

3. **清理 `port_service_scanner.go` 中不可达的死代码分支**：`ensureInit()` 里 embed 分支必定优先命中并 `return`，后面基于 `rules/fingerprint/nmap-service-probes` 路径的 fallback 逻辑永远执行不到，且路径本身也是错的。要么删除，要么改造成真正有意义的"外部规则覆盖 embed 默认值"语义（此时需要先修正路径，并调整判断顺序为"先查外部文件，查不到再退回 embed"）。

4. **把 `web_fingerprints.json` 的加载方式作为标准范式推广**：这是目前唯一验证过、且有单测覆盖的"外部化配置 + 运行时加载"范式。后续如果要让弱口令字典、CDN 网段等规则真正可配置，应参考它的路径查找 + 单测校验方式，而不是重新发明一套。

5. **补齐单测覆盖**：`fingerprint/nmap/*`、`passwords/*`、`edge/cdn.json` 目前均无类似 `web_scanner_test.go` 那样"断言规则确实被加载生效"的测试。规则一旦接入真实加载逻辑，应同步补充测试，防止未来再次退化成摆设文件。
