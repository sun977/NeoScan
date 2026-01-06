# 指纹识别服务 (Fingerprint Service)

## 1. 简介
FingerprintService 是 NeoScan 的核心服务之一，负责识别资产的指纹信息。它将分散在各个扫描工具（如 Nmap, Goby, Wappalyzer 等）的指纹数据进行统一清洗、匹配和标准化，最终生成标准的 CPE (Common Platform Enumeration) 标识。

该服务支持**数据库存储**和**文件加载**两种规则管理方式，并提供统一的识别接口。

## 2. 架构设计

### 2.1 核心组件
*   **Service**: 对外统一接口，负责协调各匹配引擎。
*   **MatchEngine**: 抽象匹配引擎接口，支持扩展不同类型的指纹库。目前内置：
    *   **HTTPEngine**: 基于 HTTP 特征（Header, Body, Title 等）识别 Web 应用/CMS。
    *   **ServiceEngine**: 基于端口 Banner 正则匹配识别基础服务（SSH, MySQL, Redis 等）。
*   **RuleManager**: (集成在 Engine 中) 负责从数据库 (`asset_finger`, `asset_cpe`) 或 JSON 文件加载规则。

### 2.2 数据流
1.  **输入**: `Input` 结构体，包含 IP, Port, Banner, HTTP Headers/Body 等。
2.  **处理**: Service 并行（或串行）调用所有注册的 Engine。
3.  **匹配**: 
    *   `HTTPEngine` 遍历 `AssetFinger` 规则进行特征匹配。
    *   `ServiceEngine` 使用正则匹配 Banner，并提取版本号。
4.  **输出**: `Result` 结构体，包含最佳匹配 (`Best`) 和所有匹配 (`Matches`)。

## 3. 接口定义

### 3.1 Service 接口
```go
type Service interface {
    // Identify 统一识别入口
    Identify(ctx context.Context, input *Input) (*Result, error)
    
    // LoadRules 从指定目录加载规则文件 (同时也会加载数据库规则)
    LoadRules(dir string) error
    
    // GetStats 获取规则库统计信息
    GetStats() map[string]int
}
```

### 3.2 输入输出模型
```go
// Input 识别输入
type Input struct {
    Target   string            // IP or Domain
    Port     int               // Port number
    Protocol string            // tcp, udp, http
    Banner   string            // Raw service banner
    Headers  map[string]string // HTTP Headers
    Body     string            // HTTP Body
}

// Result 识别结果
type Result struct {
    Matches []Match // 所有命中结果
    Best    *Match  // 优先级最高的最佳结果
}

type Match struct {
    Product    string // 产品名称 (e.g., Apache Tomcat)
    Version    string // 版本号 (e.g., 9.0.1)
    Vendor     string // 厂商 (e.g., Apache)
    CPE        string // 标准化 CPE 2.3 (e.g., cpe:2.3:a:apache:tomcat:9.0.1:...)
    Type       string // 类型 (a:应用, o:操作系统, h:硬件)
    Confidence int    // 置信度 (0-100)
    Source     string // 来源引擎
}
```

## 4. 规则文件格式

规则文件默认存放于 `rules/fingerprint/` 目录下，支持 `.json` 格式。
所有规则文件（无论文件名）必须包含顶层的 `type` 字段以区分规则类型。

### 4.1 HTTP 指纹规则 (`type: "http"`)
用于 `HTTPEngine`。支持 Goby 格式（自动转换）和 NeoScan 原生格式。

**原生格式示例:**
```json
{
  "name": "Custom HTTP Rules",
  "version": "1.0",
  "type": "http",
  "samples": [
    {
      "name": "Spring-Boot-Actuator",
      "rule": {
        "url": "/actuator/health",
        "body": "{\"status\":\"UP\"}",
        "status_code": "200",
        "match": "contains"
      }
    }
  ]
}
```

**Goby 兼容格式:**
直接使用 Goby 的 JSON 导出格式，引擎会自动识别并转换。

### 4.2 服务/CPE 指纹规则 (`type: "service"`)
用于 `ServiceEngine`。基于正则表达式匹配 Banner。

**示例:**
```json
{
  "name": "Service Rules",
  "version": "1.0",
  "type": "service",
  "samples": [
    {
      "match_str": "(?i)Apache/([\\d\\.]+)",
      "vendor": "apache",
      "product": "http_server",
      "part": "a",
      "cpe": "cpe:2.3:a:apache:http_server:$1:*:*:*:*:*:*:*"
    }
  ]
}
```
*   `match_str`: 正则表达式，支持捕获组。
*   `cpe`: CPE 模板，使用 `$1`, `$2` 引用正则捕获的版本号等信息。

## 5. 配置与使用

在 `config.yaml` 中配置规则路径：

```yaml
fingerprint:
  rule_path: "rules/fingerprint"
```

系统启动时会自动初始化 `FingerprintService` 并加载该目录下的所有规则。
同时，系统也会连接数据库，自动加载 `asset_finger` 和 `asset_cpe` 表中的规则。
