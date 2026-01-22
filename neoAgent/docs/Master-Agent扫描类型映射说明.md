# Master-Agent 扫描类型映射说明

本文档详细说明了 Master 定义的**业务扫描场景 (Intent)** 如何映射到 Agent 的**原子执行能力 (Capability)**。

## 核心设计哲学
- **Master (Brain)**: 负责编排，定义"做什么"（例如：进行一次快速体检）。
- **Agent (Muscle)**: 负责执行，定义"怎么做"（例如：测量血压，参数为左臂）。

Agent 遵循 KISS 原则，只提供最基础的原子能力，通过参数组合来满足复杂的业务需求。

## 类型映射表

| Master 业务场景 (AgentScanType) | Agent 原子能力 (TaskType) | 参数配置 (Params) | 说明 |
| :--- | :--- | :--- | :--- |
| **`ipAliveScan`**<br>(IP探活) | `asset_scan` | `ping: true`<br>`port: ""` | 仅进行 ICMP/ARP 探测，不扫端口 |
| **`fastPortScan`**<br>(快速端口扫描) | `asset_scan` | `ping: false`<br>`port: "top100"`<br>`os_detect: false` | 扫描 Top100 端口，不进行深度识别 |
| **`fullPortScan`**<br>(全量端口扫描) | `port_scan` | `port: "1-65535"`<br>`service_detect: true` | 全端口扫描 + 服务指纹识别 |
| **`serviceScan`**<br>(服务识别) | `port_scan` | `port: "custom"`<br>`service_detect: true` | 针对特定端口进行深度指纹识别 |
| **`vulnScan`**<br>(漏洞扫描) | `vuln_scan` | `templates: "cves"`<br>`severity: "critical,high"` | 使用 Nuclei 进行通用漏洞扫描 |
| **`pocScan`**<br>(POC扫描) | `vuln_scan` | `templates: "custom_pocs"` | 使用指定的 POC 模板进行精确扫描 |
| **`passScan`**<br>(弱口令扫描) | `vuln_scan` | `templates: "weak_passwords"` | 使用弱口令爆破模板 |
| **`webScan`**<br>(Web扫描) | `web_scan` | `crawl: true`<br>`method: "GET"` | 包含爬虫、指纹识别的综合 Web 扫描 |
| **`apiScan`**<br>(API扫描) | `web_scan` | `mode: "api"`<br>`path: "/api/v1"` | 针对 API 接口的特定扫描 |
| **`dirScan`**<br>(目录扫描) | `dir_scan` | `dict: "common.txt"` | 目录爆破 |
| **`subDomainScan`**<br>(子域名扫描) | `subdomain` | `mode: "brute"` | 子域名枚举 |
| **`proxyScan`**<br>(代理探测) | `port_scan` | `port: "1080,8080"`<br>`scripts: "proxy_check"` | 探测目标是否开放代理服务 |
| **`fileScan`**<br>(文件扫描) | `raw_cmd` | `cmd: "yara ..."` | **慎用**：仅在本地模式或受控环境下使用 |
| **`otherScan`**<br>(其他扫描) | `raw_cmd` | `cmd: "custom_script"` | 执行自定义脚本 |

## 特殊说明：代理服务 (Proxy)
- **Agent 端 `TaskTypeProxy`**: 指的是 Agent **自身启动一个代理服务**（如 Socks5 Server），供 Master 或其他节点作为跳板使用。
- **Master 端 `AgentScanTypeProxyScan`**: 指的是 **扫描目标主机是否开放了代理服务**。映射为 `port_scan` 或 `vuln_scan`。

## 任务转换逻辑 (Task Compiler)
Master 在下发任务前，需要经过一个 `Compiler` 层：

```go
func Compile(intent AgentScanType, target string) model.Task {
    switch intent {
    case AgentScanTypeIpAliveScan:
        return model.Task{
            Type: model.TaskTypeAssetScan,
            Params: map[string]interface{}{"ping": true},
        }
    case AgentScanTypeFullPortScan:
        return model.Task{
            Type: model.TaskTypePortScan,
            Params: map[string]interface{}{"port": "1-65535", "service_detect": true},
        }
    // ... 其他映射
    }
}
```
