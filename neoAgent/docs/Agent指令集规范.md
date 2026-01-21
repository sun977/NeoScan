# NeoAgent 指令集规范 (Instruction Set Architecture) v1.0

## 1. 概述

NeoAgent 作为一个多模态执行单元，需要统一处理来自不同源头的指令：
1.  **Cluster Source**: 来自 Master 节点的 JSON 格式指令 (通过 HTTP/gRPC 下发)。
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

### 3.1 资产扫描 (asset_scan)
**描述**: 基础的主机发现和端口扫描。

| 参数 (JSON) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target` | `--target, -t` | string | **Yes** | - | 目标 (IP/CIDR/Domain) |
| `port_range` | `--port, -p` | string | No | "top1000" | 端口范围 (80,443,1-1000) |
| `rate` | `--rate` | int | No | 1000 | 发包速率 (PPS) |
| `ping` | `--ping` | bool | No | true | 是否先进行 Ping 存活检测 |
| `tech_detect` | `--tech-detect` | bool | No | true | 是否进行服务指纹识别 |

### 3.2 端口扫描 (port_scan)
**描述**: 仅执行端口扫描，不进行其他检测。

| 参数 (JSON) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target` | `--target, -t` | string | **Yes** | - | 目标 IP/CIDR |
| `port_range` | `--port, -p` | string | No | "top1000" | 端口范围 |
| `rate` | `--rate` | int | No | 1000 | 速率 |

### 3.3 Web 扫描 (web_scan)
**描述**: 针对 HTTP/HTTPS 服务的深度扫描（指纹、爬虫）。

| 参数 (JSON) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target` | `--target, -t` | string | **Yes** | - | URL 或 Domain |
| `spider` | `--spider` | bool | No | false | 是否开启爬虫 |
| `headless` | `--headless` | bool | No | false | 是否使用浏览器渲染 |

### 3.4 目录扫描 (dir_scan)
**描述**: Web 目录爆破。

| 参数 (JSON) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target` | `--target, -t` | string | **Yes** | - | 目标 URL |
| `wordlist` | `--wordlist` | string | No | "default" | 字典文件路径或内置字典名 |
| `extensions` | `--ext` | string | No | "php,jsp,asp" | 文件扩展名 |

### 3.5 漏洞扫描 (vuln_scan)
**描述**: 调用 Nuclei 等工具进行 POC 验证。

| 参数 (JSON) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `target` | `--target, -t` | string | **Yes** | - | 目标 |
| `templates` | `--templates` | []string | No | ["cve"] | 指定扫描模板目录/标签 |
| `severity` | `--severity` | []string | No | ["critical","high"] | 漏洞等级过滤 |

### 3.6 子域名扫描 (subdomain)
**描述**: 子域名枚举。

| 参数 (JSON) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `domain` | `--domain, -d` | string | **Yes** | - | 根域名 |
| `brute` | `--brute` | bool | No | false | 是否启用暴力枚举 |

### 3.7 穿透代理 (proxy)
**描述**: 开启 SOCKS5/HTTP 代理服务或端口转发。

| 参数 (JSON) | CLI Flag | 类型 | 必选 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `mode` | `--mode` | string | **Yes** | "socks5" | 模式: socks5, http, port_forward |
| `listen` | `--listen, -l` | string | **Yes** | ":1080" | 监听地址 |
| `auth` | `--auth` | string | No | - | 认证信息 (user:pass) |
| `forward` | `--forward, -f` | string | No | - | 转发目标 (仅 port_forward 模式) |

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
