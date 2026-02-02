# Master-Agent 数据传递契约

## 1. 概述
本文档定义了 NeoScan 系统中 Master 与 Agent 之间通信的数据格式和约束，确保双方对数据结构和枚举值的理解一致。

## 2. 任务支持 (TaskSupport)
Agent 在注册时通过 `task_support` 字段上报其支持的扫描能力。Master 根据此列表将 Agent 映射到对应的系统标签 (System Tags)。

**约束原则**:
- Agent 必须严格使用以下定义的字符串作为能力标识。
- Master 必须在数据库 (`scan_types` 表) 中预置以下类型。
- 任何未定义的类型将被 Master 拒绝 (注册失败)。

### 2.1 标准扫描能力
以下是系统目前支持的标准扫描能力：

| 标识符 (Key) | 名称 | 描述 | 备注 |
| :--- | :--- | :--- | :--- |
| `ipAliveScan` | IP探活扫描 | ICMP/ARP/TCP Ping 探测 | 基础能力 |
| `fastPortScan` | 快速端口扫描 | Top 1000 端口扫描 | |
| `fullPortScan` | 全量端口扫描 | 1-65535 端口扫描 | |
| `serviceScan` | 服务识别 | 端口服务指纹识别 | |
| `vulnScan` | 漏洞扫描 | 基于插件的漏洞检测 | |
| `pocScan` | POC扫描 | 特定漏洞验证 | |
| `webScan` | Web扫描 | 爬虫与Web漏洞扫描 | |
| `passScan` | 弱口令扫描 | 服务弱口令爆破 | |
| `proxyScan` | 代理探测 | 代理服务发现 | |
| `dirScan` | 目录扫描 | Web目录爆破 | |
| `subDomainScan` | 子域名扫描 | 子域名挖掘 | |
| `apiScan` | API扫描 | API接口发现与安全检测 | |
| `fileScan` | 文件扫描 | 敏感文件与恶意软件检测 | |
| `otherScan` | 其他扫描 | 自定义脚本或工具 | |

### 2.2 配置来源
- **Agent**: 应在 `internal/app/agent/app.go` 或配置文件中硬编码使用上述 Key。
- **Master**: 应在 `cmd/migrate/main.go` (种子数据) 和 `internal/model/agent/agent.go` (常量定义) 中维护一致的定义。

## 3. 注册接口契约

**接口**: `POST /api/v1/agent/register`

**请求体 (JSON)**:
```json
{
  "hostname": "string",
  "ip_address": "string",
  "port": 8080,
  "version": "1.0.0",
  "os": "linux",
  "arch": "amd64",
  "cpu_cores": 4,
  "memory_total": 8589934592,
  "disk_total": 107374182400,
  "task_support": [
    "ipAliveScan",
    "fastPortScan"
  ],
  "tags": ["custom_tag"],
  "token_secret": "string"
}
```

**响应体 (JSON)**:
```json
{
  "code": 200,
  "status": "registered",
  "data": {
    "agent_id": "agent_xxxxxxxx",
    "token": "token_xxxxxxxx",
    "token_expiry": "2026-01-01T00:00:00Z"
  }
}
```
