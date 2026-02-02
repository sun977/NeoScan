# Master-Agent 数据传递契约 (TaskSupport)

## 1. 概述
本契约定义了 NeoScan Master 与 Agent 之间关于 `TaskSupport` (任务支持/能力) 的标准枚举值。
Agent 在注册 (`RegisterAgent`) 时，必须在 `task_support` 字段中上传以下标准标识符 (Key/Name)。
Master 将根据这些标识符将 Agent 绑定到对应的系统标签，从而实现基于能力的任务调度。

## 2. 核心原则
1.  **唯一配置源**：Master 数据库中的 `scan_types` 表是唯一真理来源。
2.  **Fail Fast**：Agent 上传未定义的 `task_support` 将导致注册失败。
3.  **字符串标识**：Master 与 Agent 交互仅使用字符串标识符 (Name)，不使用数据库 ID。

## 3. 标准 TaskSupport 枚举

| 标识符 (Key) | 名称 | 描述 | 类别 |
| :--- | :--- | :--- | :--- |
| `ipAliveScan` | IP探活扫描 | ICMP/ARP/TCP Ping 探测 | network |
| `fastPortScan` | 快速端口扫描 | Top 1000 端口扫描 | network |
| `fullPortScan` | 全量端口扫描 | 1-65535 端口扫描 | network |
| `serviceScan` | 服务扫描 | 端口服务指纹识别 | service |
| `vulnScan` | 漏洞扫描 | 通用漏洞扫描 (CVE/Exploit) | security |
| `pocScan` | POC扫描 | 基于 POC 脚本的高精度验证 | security |
| `webScan` | Web扫描 | Web 爬虫与漏洞扫描 | web |
| `passScan` | 弱密码扫描 | 常见服务弱口令爆破 | security |
| `proxyScan` | 代理探测 | HTTP/SOCKS 代理服务探测 | network |
| `dirScan` | 目录扫描 | Web 目录/文件爆破 | web |
| `subDomainScan` | 子域名扫描 | 子域名枚举与探测 | web |
| `apiScan` | API扫描 | Swagger/GraphQL API 资产发现 | web |
| `fileScan` | 文件扫描 | Webshell 查杀与病毒扫描 | file |
| `otherScan` | 其他扫描 | 自定义脚本或扩展扫描 | custom |

## 4. Agent 配置示例 (`agent.yaml`)

```yaml
agent:
  # ... 其他配置
  # Agent 支持的扫描能力列表 (必须与上述枚举严格匹配)
  task_support:
    - "ipAliveScan"
    - "fastPortScan"
    - "webScan"
```

## 5. Master 校验逻辑
Master 在接收到 Agent 注册请求时：
1. 读取 `task_support` 列表。
2. 在 `scan_types` 表中查询对应的 `name`。
3. 如果任一 `task_support` 无法匹配，或者匹配到的 `tag_id` 无效，拒绝注册。
4. 校验通过后，自动将 Agent 绑定到对应的系统标签。
