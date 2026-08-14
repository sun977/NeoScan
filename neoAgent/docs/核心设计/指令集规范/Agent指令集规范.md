# NeoAgent 指令集规范 (Instruction Set Architecture)

- **版本**: v1.4
- **日期**: 2026-08-07
- **作者**: sun977
- **状态**: 正式生效

### 更新记录

| 版本 | 日期 | 变更内容 |
| :--- | :--- | :--- |
| v1.0 | - | 初版发布：定义控制/任务/系统三类指令集与 CLI-Cluster 映射原则 |
| v1.1 | 2026-08-04 | 按真实代码修正 3.3 节 Web 扫描指令：移除代码中不存在的 `spider`/`headless` 字段，补充真实字段 `ports`/`path`/`method`/`crawl`/`crawl_depth`/`screenshot`，并标注 TaskType 取值与 Master `api_scan` 的映射关系 |
| v1.2 | 2026-08-04 | 全面核对代码，根据实际开发状态重写 3.1-3.9节：删除代码中不存在的 `asset_scan`；修正 3.1/3.2 为真实存在的 `port_scan`（合并了原文档中重复的端口扫描与服务识别），并标注 Master 通道 `ip_alive_scan` 参数透传缺陷；新增 `ip_alive_scan`、`os_scan`、`brute_force` 三节（已实现但旧版文档从未收录）；3.6-3.9（`dir_scan`/`vuln_scan`/`subdomain`/`proxy`）恢复参数表格但按代码中 `options.*Options`/CLI Flag 逐字核实修正字段名与默认值，并在标题与正文中明确标注为**未实现**，避免误导用户认为可用 |
| v1.3 | 2026-08-04 | 修复 3.5 节记录的技术债：新增 `internal/core/options/scan_brute.go`（`BruteScanOptions`），将 `cmd/agent/scan/brute.go` 中散落的端口默认值推断逻辑收敛进 `Validate()`，`brute` 命令改为标准的 `NewXxxOptions → Validate → ToTask` 三段式，与其余已实现扫描指令的组织方式统一；`BruteScanner.Run()` 接口保持不变 |
| v1.4 | 2026-08-07 | `api_scan` 从 `web_scan` 的 `mode:"api"` 参数独立成自己的 TaskType（`model.TaskTypeApiScan`）与 `ApiScanner`，新增独立的 3.4 节（骨架版本，JS 接口提取逻辑待 `Web-JS接口提取实施文档.md` 补充），原 3.4-3.9 节顺延为 3.5-3.10；同步移除 3.3 节中“Master 下发的 `api_scan` 也会被翻译成同一个 TaskType”的过时描述 |

---

## 1. 概述

NeoAgent 作为一个多模态执行单元，需要统一处理来自不同源头的指令：
1.  **Cluster Source**: 来自 Master 节点的 JSON 格式指令 (通过 HTTP 下发)。
2.  **CLI Source**: 来自用户命令行的 Flags 参数。

本规范旨在建立一套**统一的指令集**，确保无论指令来自何处，最终都能映射到相同的内部逻辑 (`internal/core/model.Task`)。

---

## 2. 指令分类

| 类别 | 来源 | 说明 | 示例 |
| :--- | :--- | :--- | :--- |
| **控制指令 (Control)** | Cluster | 影响 Agent 自身状态，不产生扫描结果。 | 心跳、配置更新、重启 |
| **任务指令 (Task)** | Cluster/CLI | 触发扫描业务，产生标准化的 TaskResult。 | 端口扫描、Web 指纹 |
| **系统指令 (System)** | Cluster/CLI | 获取底层系统信息或执行 Raw 命令。 | 资源监控、Shell 执行 |

---

## 3. 任务指令集 (Task Instructions)

这是核心业务指令，Cluster 和 CLI 必须严格对齐。

### 3.1 IP 存活扫描 (ip_alive_scan)
**描述**: 仅进行主机存活探测（ICMP/ARP/TCP Connect），不扫描端口服务，可根据 ICMP TTL 粗略估算操作系统。

**TaskType**: `ip_alive_scan`（`model.TaskTypeIpAliveScan`）。CLI 命令：`neoAgent scan alive`。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target`（写入 `Task.Target`） | `--target, -t` | string | **Yes** | - | 目标 IP/CIDR |
| `enable_arp` | `--arp, -A` | bool | No | `false` | 使用 ARP 探测（仅同广播域有效） |
| `enable_icmp` | `--icmp, -I` | bool | No | `false` | 使用 ICMP 探测（可根据 TTL 估算 OS） |
| `enable_tcp` | `--tcp, -T` | bool | No | `false` | 使用 TCP Full Connect 探测 |
| `tcp_ports` | `--tcp-ports, -p` | []int | No | `[22,23,80,139,512,443,445,3389]` | TCP 探测端口列表 |
| `concurrency` | `--concurrency, -c` | int | No | `1000` | 并发数 |
| `resolve_hostname` | `--resolve-hostname, -r` | bool | No | `false` | 是否启用 DNS PTR 反向解析 |

**说明**: 三个协议开关（`enable_arp`/`enable_icmp`/`enable_tcp`）默认全部为 `false` 代表自动模式（Agent 根据是否同网段智能选择探测协议）；一旦手动指定任意一个开关即进入手动模式。只有 ICMP 会携带 TTL，因此只有 ICMP 探测才可能返回 OS 估算结果。

**⚠️ 已知缺陷（Master 通道）**: `internal/service/adapter/task_to_core.go` 中 Master 的 `ip_alive_scan` 翻译逻辑只写入了 `ping`/`port` 两个字段，而 `AliveScanner.parseOptions` 实际只读取 `enable_arp`/`enable_icmp`/`enable_tcp`/`tcp_ports`/`concurrency`/`resolve_hostname`。这意味着 **Master 下发的存活扫描任务当前无法透传探测协议参数**，只会走三个开关全 `false` 的默认（自动）模式；CLI 直接调用不受影响。修复需同步改造 `task_to_core.go`，不在本次文档修订范围内，此处仅如实记录。

### 3.2 端口/服务扫描 (port_scan)
**描述**: 端口扫描，`service_detect=true` 时会附加深度服务指纹识别（等价于旧文档中"资产扫描"与"服务识别"的合并版本，两者底层是同一个 `PortServiceScanner`，仅通过参数区分）。

**TaskType**: `port_scan`（`model.TaskTypePortScan`）。CLI 命令：`neoAgent scan port`。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target`（写入 `Task.Target`） | `--target, -t` | string | **Yes** | - | 目标 IP/CIDR |
| `port_range`（写入 `Task.PortRange`） | `--port, -p` | string | **Yes** | - | 端口范围（如 `80,443,1-1000`），无默认值，必须显式指定 |
| `rate` | `--rate, -r` | int | No | `1000` | 扫描速率/并发数 |
| `service_detect` | `--service-detect, -s` | bool | No | `true` | 是否启用深度服务版本识别 |

**说明**: 旧版文档中的 `asset_scan`（含 `ping`/`tech_detect` 字段）在代码中从未存在，已删除；`rate` 的真实语义是并发数而非发包速率（PPS），已按代码注释修正。

**Master 通道的细分类型**: `internal/service/adapter/task_to_core.go` 中 Master 还会下发 `fast_port_scan`（top100 端口，不做服务识别）、`full_port_scan`（1-65535 全端口，不做服务识别）、`service_scan`（独立 TaskType，值为 `service_scan`，强制 `service_detect=true`）三种细分指令，均由同一个 `PortServiceScanner` 处理（`service_scan` 通过 `ServiceRunner` 适配器注册），只是参数预设不同；CLI 没有与之一一对应的独立子命令，只能通过 `--port`/`--service-detect` 组合手动达到相同效果。

### 3.3 Web 扫描 (web_scan)
**描述**: 针对 HTTP/HTTPS 服务的综合扫描（指纹识别、深度爬行、截图）。

**TaskType**: `web_scan`（`model.TaskTypeWebScan`）。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target`（写入 `Task.Target`） | `--target, -t` | string | **Yes** | - | 目标 URL/IP |
| `port_range`（写入 `Task.PortRange`） | `--ports, -p` | string | No | `"80,443"` | 端口范围，支持 `"80,443"`/`"1-100"`/`"top100"` 等写法，多端口并发探测 |
| `path` | `--path` | string | No | `"/"` | 扫描路径 |
| `method` | `--method, -m` | string | No | `"GET"` | HTTP 方法 |
| `crawl` | `--crawl` | string | No | `"auto"` | `auto`(默认，交给 WebScanner 自动判断) / `true` / `false`，是否开启深度爬取（BFS） |
| `crawl_depth` | `--crawl-depth` | int | No | `2` | 爬取深度，仅 `crawl=true` 时生效 |
| `screenshot` | `--screenshot` | bool | No | `false` | 是否对页面截图（依赖无头浏览器） |
| `protocol`（仅 Master 下发时使用） | - | string | No | - | 协议提示 `http`/`https`，CLI 目前不传递，由 WebScanner 自行推断 |

**说明**: 这里列出的是 `internal/core/options/scan_web.go` 中 `WebScanOptions.ToTask()` 真实写入 `Task.Params` 的字段，与 `cmd/agent/scan/web.go` 的 CLI Flag 一一对应。旧版本文档中的 `spider`/`headless` 字段为早期设计草稿，实际代码中从未实现，已在 v1.1 修正。

### 3.4 API 扫描 (api_scan)

> 本章节参数表为骨架阶段的最小版本，`max_files` 参数与风险提示文案见
> `docs/API扫描-js提取/Web-JS接口提取实施文档.md` 第二节，实施完成后需要合并补充到本节。

**描述**: 从抓取到的页面（含可选深度爬取子页面）中提取接口调用地址（JS 接口提取等）。

**TaskType**: `api_scan`（`model.TaskTypeApiScan`），与 `web_scan` 是完全独立的 TaskType，不再共享同一段处理逻辑，不再借道 `web_scan` 的 `mode:"api"` 参数。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target`（写入 `Task.Target`） | `--target, -t` | string | **Yes** | - | 目标 URL/IP |
| `port_range`（写入 `Task.PortRange`） | `--ports, -p` | string | No | `"80,443"` | 目标端口范围 |
| `crawl` | `--crawl` | string | No | `"auto"` | `auto`(默认，骨架阶段等价于不爬) / `true` / `false`，是否深度爬取子页面 |
| `crawl_depth` | `--crawl-depth` | int | No | `2` | 深度爬取层数，仅 `crawl=true` 时生效 |

**现状**: 参数已按 `internal/core/options/scan_api.go` 中 `ApiScanOptions.ToTask()` 完整定义并通过 `RunnerManager` 正确调度到 `ApiScanner`，但当前为骨架版本——`ApiScanner.Run()` 只组装 `URL`/`Depth` 两个定位字段，真正的 JS 接口提取逻辑（`APIs`/`APIsTruncated` 字段、`max_files` 参数）由 `docs/API扫描-js提取/Web-JS接口提取实施文档.md` 补充实现。

### 3.5 操作系统识别 (os_scan)
**描述**: 通过 TCP/IP 协议栈指纹识别目标操作系统类型。

**TaskType**: `os_scan`（`model.TaskTypeOsScan`）。CLI 命令：`neoAgent scan os`。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target`（写入 `Task.Target`） | `--target, -t` | string | **Yes** | - | 目标 IP |
| `mode` | `--mode, -m` | string | No | `"auto"` | `fast`（仅 TTL 估算）/ `deep`（Nmap OS 指纹库 + 服务 Banner）/ `auto`（混合模式） |

### 3.6 弱口令爆破 (brute_force)
**描述**: 针对指定服务进行弱口令爆破，内置 Top100 弱口令字典，支持自定义用户名/密码列表。当前已注册的协议：SSH、MySQL、Redis、PostgreSQL、FTP、MongoDB、ClickHouse、SMB、MSSQL、Oracle、Oracle SID、Telnet、Elasticsearch、SNMP、RDP。

**TaskType**: `brute_force`（`model.TaskTypeBrute`）。CLI 命令：`neoAgent scan brute`。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target`（写入 `Task.Target`） | `--target, -t` | string | **Yes** | - | 目标 IP |
| `port_range`（写入 `Task.PortRange`） | `--port, -p` | string | No | 按 `service` 自动填充常见端口（如 ssh→22，mysql→3306） | 目标端口；服务无默认端口映射时必须显式指定 |
| `service` | `--service, -s` | string | **Yes** | - | 服务名称（ssh/mysql/redis/postgres/ftp/mongo/clickhouse/smb/mssql/oracle/oracle-sid/telnet/elasticsearch/snmp） |
| `users` | `--users, -u` | []string | No | 内置字典 | 自定义用户名，逗号分隔或文件路径 |
| `passwords` | `--pass` | []string | No | 内置 Top100 字典 | 自定义密码，逗号分隔或文件路径 |
| `stop_on_success` | `--all, -a`（取反） | bool | No | `true` | 是否找到一个有效凭据后即停止；`--all` 会将其置为 `false`（尝试所有凭据） |

**说明**: CLI 已改造为 `internal/core/options/scan_brute.go` 中的 `BruteScanOptions`（`NewBruteScanOptions` → `Validate` → `ToTask` 三段式），与其余已实现扫描指令的组织方式统一。按 `service` 推断默认端口的逻辑已收敛进 `Validate()`，不再散落在 CLI 命令里。`BruteScanner.Run()` 本身不受影响，仍从 `task.Params` 读取参数，接口无变化。

### 3.7 目录扫描 (dir_scan) —— **未实现**
**描述**: Web 目录爆破（设计目标，尚未落地）。

**TaskType**: `dir_scan`（`model.TaskTypeDirScan`）。CLI 命令：`neoAgent scan dir`。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target`（写入 `Task.Target`） | `--target, -t` | string | **Yes** | - | 目标 URL |
| `dict` | `--dict, -d` | string | No | - | 字典文件路径 |
| `extensions` | `--extensions, -e` | string | No | - | 文件后缀（如 `php,jsp`） |
| `threads` | `--threads` | int | No | `10` | 并发线程数 |

**现状**: 参数已按 `options.DirScanOptions` 完整定义并通过 `Validate()` 校验，但 `internal/core/scanner` 下没有 `DirScanner` 实现，`RunnerManager` 也未注册对应 Runner。`cmd/agent/scan/dir.go` 在校验参数通过后会直接 `return fmt.Errorf("dir scan not implemented yet")`，即上表参数目前**不会触发任何真实扫描**。

### 3.8 漏洞扫描 (vuln_scan) —— **未实现**
**描述**: 调用 POC/模板进行漏洞验证（设计目标，尚未落地）。

**TaskType**: `vuln_scan`（`model.TaskTypeVulnScan`）。CLI 命令：`neoAgent scan vuln`。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target`（写入 `Task.Target`） | `--target, -t` | string | **Yes** | - | 扫描目标 |
| `templates` | `--templates` | string | No | - | 模板路径 |
| `severity` | `--severity` | string | No | `"medium,high,critical"` | 漏洞等级过滤（逗号分隔字符串，非数组） |

**现状**: 参数已按 `options.VulnScanOptions` 定义，但底层无 `VulnScanner` 实现。`cmd/agent/scan/vuln.go` 校验参数后直接 `return fmt.Errorf("vuln scan not implemented yet")`，上表参数目前**不会触发任何真实扫描**。Master 侧 `task_to_core.go` 中的 `poc_scan`/`weak_pass_scan` 也会被一并映射到这个未实现的 `vuln_scan`。

### 3.9 子域名扫描 (subdomain) —— **未实现**
**描述**: 子域名枚举（设计目标，尚未落地）。

**TaskType**: `subdomain`（`model.TaskTypeSubdomain`）。CLI 命令：`neoAgent scan subdomain`。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `domain`（写入 `Task.Target`） | `--domain, -d` | string | **Yes** | - | 根域名 |
| `dict` | `--dict` | string | No | - | 字典文件路径 |
| `threads` | `--threads` | int | No | `10` | 并发线程数 |

**现状**: 参数已按 `options.SubdomainScanOptions` 定义，但底层无 `SubdomainScanner` 实现。`cmd/agent/scan/subdomain.go` 校验参数后直接 `return fmt.Errorf("subdomain scan not implemented yet")`，上表参数目前**不会触发任何真实扫描**。

### 3.10 穿透代理 (proxy) —— **未实现**
**描述**: 开启 SOCKS5/HTTP 代理服务或端口转发（设计目标，尚未落地）。

**TaskType**: `proxy`（`model.TaskTypeProxy`）。CLI 命令：`neoAgent proxy`（注意：不在 `scan` 子命令组下，是独立顶级命令）。

| 参数 (Task.Params) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `mode` | `--mode` | string | No | `"socks5"` | 模式：`socks5`/`http`/`port_forward` |
| `-`（写入 `Task.Target`，取值同 `listen`） | `--listen, -l` | string | No | `":1080"` | 监听地址 |
| `auth` | `--auth` | string | No | - | 认证信息 `user:pass`，`port_forward` 模式外可选 |
| `forward` | `--forward, -f` | string | No | - | 转发目标，仅 `port_forward` 模式下必填（`Validate()` 会强制校验） |

**现状**: 参数已按 `options.ProxyOptions` 完整定义并通过 `Validate()` 校验，但 `cmd/agent/proxy/root.go` 的 `RunE` 只会把 `ToTask()` 的结果打印成 JSON（源码标注 `TODO: 调用真正的 Runner`），`RunnerManager` 中也没有注册 `TaskTypeProxy` 对应的 Runner。上表参数目前**不会启动任何真实代理服务**。

---

## 4. 控制指令集 (Control Instructions)

此类指令通常只存在于 Cluster 模式，用于 Master 管理 Agent。

### 4.1 配置更新 (config_update)
**Payload**:
```json
{
  "concurrent_limit": 500,
  "log_level": "debug"
}
```

### 4.2 自身升级 (self_upgrade)
**Payload**:
```json
{
  "version": "v1.2.0",
  "download_url": "https://master/dl/agent_v1.2.0",
  "checksum": "sha256:..."
}
```

---

## 5. Raw 指令集 (Advanced)

允许用户透传参数给底层工具 (如 Nmap/Nuclei)。

### 5.1 工具透传 (raw_exec)
**CLI 用法**: `neoAgent scan raw --tool <name> -- <args>`

| 参数 (JSON) | CLI Flag | 类型 | 说明 |
| :--- | :--- | :--- | :--- |
| `tool` | `--tool` | string | 工具名称 (nmap, nuclei) |
| `args` | `--` 后的参数 | []string | 透传参数列表 |

**Cluster Payload**:
```json
{
  "type": "raw_exec",
  "payload": {
    "tool": "nmap",
    "args": ["-sS", "-Pn", "-p80", "1.1.1.1"]
  }
}
```
*注意*: Cluster 模式下应严格限制 Raw 指令的权限，防止 RCE 风险。

---

## 6. 实现映射 (Implementation Mapping)

为了实现统一处理，我们需要在 `internal/server/handler` (Cluster) 和 `cmd/agent` (CLI) 之间共享一套 **Command Parser**。

1.  **CLI**: `Flags` -> `Core Task`
2.  **Cluster**: `JSON` -> `Core Task`

**核心原则**: 所有的扫描逻辑，最终都必须转换为 `internal/core/model.Task` 结构体，丢给 Runner 执行。
