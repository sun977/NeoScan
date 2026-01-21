# Master-Agent 数据契约 (Data Contract)

## 1. 概述

本文档定义了 NeoAgent 上报任务结果时必须遵守的数据格式契约。
Master 端的编排器（Orchestrator）和 ETL 引擎强依赖于此结构进行数据清洗、入库和下一阶段的任务调度。

**严禁事项**:
1.  严禁随意更改 JSON 字段名称（大小写敏感）。
2.  严禁缺失关键字段（如 `ip`, `port` 等）。
3.  严禁更改数据类型（如将 `port` 从 `int` 改为 `string`）。

所有 Agent 的 `TaskResult.Result` 必须严格符合以下 JSON Schema。

**重要变更**：`TaskResult.Result` 不再直接作为 `attributes`，而是包含 `attributes` 和 `evidence` 两个顶层字段。

---

## 2. 顶层结构 (Root Structure)

Agent 上报的 `TaskResult.Result` 必须包含以下两个字段：

```json
{
  "attributes": {
    // 结构化结果 (具体结构见下文)
  },
  "evidence": {
    // 原始证据 (如工具原始输出、截图Base64等)
    "raw_output": "...",
    "screenshots": [...]
  }
}
```

*   `attributes`: (必选) 供 Master 进行逻辑判断、数据清洗和入库。
*   `evidence`: (可选) 供审计和人工复查，Master 会将其归档。

---

## 3. 结果类型定义 (Attributes Definitions)

以下定义的是 `TaskResult.Result["attributes"]` 的内部结构。

### 3.1 探活扫描 (ip_alive)

*   **对应工具**: ICMP Ping, ARP Scan
*   **Master 接收字段**: `attributes`

```json
{
  "hosts": [
    {
      "ip": "192.168.1.10",          // [必须] 存活IP
      "rtt": 0.45,                   // [可选] 响应时间(ms)
      "ttl": 64,                     // [可选] TTL值
      "hostname": "host-10.local",   // [可选] 主机名
      "os": "Linux 5.x",             // [可选] 操作系统猜测
      "mac": "00:11:22:33:44:55"     // [可选] MAC地址
    }
  ],
  "summary": {
    "alive_count": 1,
    "total_scanned": 256,
    "elapsed_ms": 1500
  }
}
```

### 3.2 端口扫描 (port_scan)

*   **对应工具**: Nmap, Masscan
*   **适用类型**: `fast_port_scan` (快速), `full_port_scan` (全量)
*   **Master 接收字段**: `attributes`

```json
{
  "ports": [
    {
      "ip": "192.168.1.10",          // [必须] 目标IP
      "port": 80,                    // [必须] 端口号 (int)
      "proto": "tcp",                // [必须] 协议 (tcp/udp)
      "state": "open",               // [必须] 状态 (open/closed/filtered)
      "service_hint": "http",        // [可选] 服务猜测 (nmap-service-probes)
      "banner": "nginx/1.18"         // [可选] 端口Banner
    }
  ],
  "summary": {
    "open_count": 1,
    "scan_strategy": "top-1000",     // [可选] 扫描策略
    "elapsed_ms": 1234
  }
}
```

### 3.3 服务指纹识别 (service_fingerprint)

*   **对应工具**: Nmap (-sV), Goby
*   **Master 接收字段**: `attributes`

```json
{
  "services": [
    {
      "ip": "192.168.1.10",
      "port": 80,
      "proto": "tcp",
      "name": "Apache httpd",                         // [必须] 服务名称
      "version": "2.4.41",                            // [可选] 版本号
      "product": "Apache httpd",                      // [可选] 产品名
      "os_type": "Linux",                             // [可选] 操作系统类型
      "cpe": "cpe:/a:apache:http_server:2.4.41",      // [重要] CPE标识
      "confidence": 10                                // [可选] 置信度 (0-10)
    }
  ]
}
```

### 3.4 Web 端点发现 (web_endpoint)

*   **对应工具**: Nuclei, HTTPX, FingerprintHub
*   **Master 接收字段**: `attributes`

```json
{
  "endpoints": [
    {
      "url": "https://example.com/api",               // [必须] 完整URL
      "ip": "1.2.3.4",                                // [必须] 解析IP
      "port": 443,                                    // [必须] 端口
      "title": "API Documentation",                   // [可选] 网页标题
      "status_code": 200,                             // [必须] HTTP状态码
      "content_length": 1024,                         // [可选] 响应长度
      "headers": {                                    // [可选] 关键响应头
        "Server": "Nginx", 
        "X-Powered-By": "Express"
      },
      "tech_stack": ["Node.js", "Express", "Nginx"],  // [重要] 识别到的技术栈
      "screenshot": "base64_encoded...",              // [可选] 截图
      "favicon": "base64_encoded..."                  // [可选] Favicon
    }
  ]
}
```

### 3.5 漏洞发现 (vuln_finding)

*   **对应工具**: Nuclei, Xray, Nessus
*   **Master 接收字段**: `attributes`

```json
{
  "findings": [
    {
      "ip": "192.168.1.10",
      "port": 80,                                     // [可选] 关联端口
      "url": "http://192.168.1.10/login",             // [可选] 关联URL
      "id": "CVE-2021-1234",                          // [必须] 漏洞唯一ID (或PluginID)
      "name": "Apache Log4j RCE",                     // [必须] 漏洞名称
      "cve": "CVE-2021-1234",                         // [可选] CVE编号
      "severity": "high",                             // [必须] 严重等级 (critical/high/medium/low/info)
      "confidence": "high",                           // [可选] 置信度
      "description": "Remote Code Execution...",      // [可选] 描述
      "solution": "Upgrade to version x.y.z",         // [可选] 修复建议
      "evidence_ref": "ref-uuid-123"                  // [可选] 关联到 evidence 字段的引用
    }
  ]
}
```

### 3.6 PoC 验证 (poc_scan)

*   **对应工具**: Custom PoC Runner
*   **Master 接收字段**: `attributes`

```json
{
  "poc_results": [
    {
      "ip": "192.168.1.10",
      "poc_id": "CVE-2021-1234#poc1",                 // [必须] PoC ID
      "target": "https://example.com",                // [必须] 验证目标
      "status": "confirmed",                          // [必须] 状态 (confirmed/not_vulnerable/failed)
      "severity": "high",                             // [可选] 严重等级
      "payload": "user=${jndi:ldap://...}",           // [可选] 使用的Payload
      "response_snapshot": "..."                      // [可选] 响应快照
    }
  ]
}
```

### 3.7 密码审计 (password_audit)

*   **对应工具**: Hydra, Medusa
*   **Master 接收字段**: `attributes`

```json
{
  "accounts": [
    {
      "host": "example.com",                          // [必须] 目标主机
      "port": 22,                                     // [必须] 端口
      "service": "ssh",                               // [必须] 服务名
      "username": "admin",                            // [必须] 用户名
      "password": "admin123",                         // [必须] 密码
      "success": true,                                // [必须] 是否成功
      "root_access": false                            // [可选] 是否由Root权限
    }
  ],
  "policy": {
    "max_attempts": 3
  }
}
```

### 3.8 域名/子域发现 (subdomain_discovery)

*   **对应工具**: Subfinder, OneForAll
*   **Master 接收字段**: `attributes`

```json
{
  "subdomains": [
    {
      "host": "api.example.com",                      // [必须] 子域名
      "ip": "203.0.113.10",                           // [可选] 解析IP
      "source": "crt.sh",                             // [可选] 来源
      "cname": "example.herokuapp.com",               // [可选] CNAME记录
      "is_wildcard": false                            // [可选] 是否泛解析
    }
  ]
}
```

### 3.9 代理检测 (proxy_detection)

*   **对应工具**: Custom Proxy Checker
*   **Master 接收字段**: `attributes`

```json
{
  "proxies": [
    {
      "ip": "1.2.3.4",                                // [必须] 代理IP
      "port": 8080,                                   // [必须] 端口
      "type": "http",                                 // [必须] 类型 (http/socks4/socks5)
      "open": true,                                   // [必须] 是否开放
      "auth_required": false                          // [可选] 是否需要认证
    }
  ]
}
```

### 3.10 目录扫描 (directory_scan)

*   **对应工具**: Gobuster, Dirsearch, Feroxbuster
*   **Master 接收字段**: `attributes`

```json
{
  "paths": [
    {
      "url": "https://example.com/.git",              // [必须] 完整URL
      "status": 200,                                  // [必须] HTTP状态码
      "length": 1024,                                 // [可选] 响应长度
      "sensitive": true                               // [可选] 是否敏感文件
    }
  ]
}
```

### 3.11 API 发现 (api_discovery)

*   **对应工具**: Kiterunner, Custom Scripts
*   **Master 接收字段**: `attributes`

```json
{
  "apis": [
    {
      "method": "GET",                                // [必须] HTTP方法
      "path": "/v1/users",                            // [必须] API路径
      "status": 200,                                  // [必须] 状态码
      "auth_required": true                           // [可选] 是否需要认证
    }
  ],
  "spec": {
    "format": "OpenAPI",                              // [可选] 规范格式
    "version": "3.0"                                  // [可选] 版本
  }
}
```

### 3.12 文件发现 (file_discovery)

*   **对应工具**: Custom Scripts
*   **Master 接收字段**: `attributes`

```json
{
  "files": [
    {
      "url": "https://example.com/backup.zip",        // [必须] 文件URL
      "path": "/backup.zip",                          // [可选] 相对路径
      "size": 1048576,                                // [可选] 文件大小(bytes)
      "mime": "application/zip",                      // [可选] MIME类型
      "sensitive": true                               // [可选] 是否敏感
    }
  ]
}
```

### 3.13 其他扫描 (other_scan)

*   **对应工具**: Custom Tools
*   **Master 接收字段**: `attributes`

```json
{
  "summary": "Custom scan output",                    // [可选] 摘要
  "data": {                                           // [可选] 任意结构化数据
    "key": "value", 
    "note": "free-form data"
  }
}
```

---

## 4. 字段类型规范

*   **IP 地址**: 必须是标准 IPv4 或 IPv6 字符串。
*   **Port**: 必须是整数 (Integer)。
*   **Status/Severity**: 必须使用全小写枚举值 (如 `open`, `high`)。
*   **Time**: 推荐使用 ISO 8601 格式字符串或 Unix 时间戳。
*   **Arrays**: 即使只有一个结果，也必须包装在数组中 (如 `ports: []`)，方便 Master 统一处理。
