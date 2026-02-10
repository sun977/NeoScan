# 指纹识别服务 (Fingerprint Service)

fingerprint 服务：保持纯粹的计算引擎角色，只负责 "Input -> Identify -> Result"，不关心数据库

## 1. 简介
FingerprintService 是 NeoScan 的核心服务之一，负责识别资产的指纹信息。它将分散在各个扫描工具（如 Nmap, Goby, Wappalyzer 等）的指纹数据进行统一清洗、匹配和标准化，最终生成标准的 CPE (Common Platform Enumeration) 标识。

该服务支持**数据库存储**和**文件加载**两种规则管理方式，并提供统一的识别接口。

## 2. 架构决策 (Architecture Decisions)

针对指纹库的**格式策略**与**分发策略**，NeoScan 采用以下核心架构设计：

### 2.1 格式策略：统一结构 (Unified Schema)
我们选择将所有第三方指纹库（EHole, Goby, Wappalyzer等）转换为 NeoScan 定义的**统一内部格式**，而不是让引擎去适配每种第三方格式。

*   **原因 1：性能极致**
    *   Agent 不需要同时运行 5 套不同的匹配引擎（如正则、DOM 分析、关键字等）。
    *   统一格式允许使用高效的算法（如 AC 自动机、预编译正则）进行一次性匹配。
*   **原因 2：维护解耦**
    *   第三方指纹库的数据结构变更（Scheme Change）被隔离在 Master 端的 `converters` 层。
    *   Agent 核心代码无需因指纹库更新而频繁修改。
*   **原因 3：数据清洗**
    *   消除冗余：合并不同来源对同一组件（如 "Shiro"）的定义。
    *   质量控制：在入库前剔除误报高或无效的规则。

### 2.2 分发策略：版本化快照 (Versioned Snapshots)
我们采用 **Master 统一管理 -> 生成快照 -> Agent 拉取 -> 内存匹配** 的策略。

*   **拒绝实时查库**：指纹识别发生在海量 HTTP 响应处理阶段，实时查询 Master 数据库会导致巨大的网络延迟和数据库压力。
*   **拒绝原始文件分发**：直接分发原始 JSON 文件难以进行细粒度的规则控制（如临时禁用某条规则）和版本追踪。
*   **采用内存加载**：Agent 启动或更新时，将指纹库全量加载到内存（Trie 树或 Map），确保匹配速度仅受 CPU 限制，不受 I/O 影响。

## 3. 架构设计

### 3.1 核心组件
*   **Service**: 对外统一接口，负责协调各匹配引擎。
*   **MatchEngine**: 抽象匹配引擎接口，支持扩展不同类型的指纹库。目前内置：
    *   **HTTPEngine**: 基于 HTTP 特征（Header, Body, Title 等）识别 Web 应用/CMS。
    *   **ServiceEngine**: 基于端口 Banner 正则匹配识别基础服务（SSH, MySQL, Redis 等）。
*   **RuleManager**: (集成在 Engine 中) 负责从数据库 (`asset_finger`, `asset_cpe`) 或 JSON 文件加载规则。

### 3.2 数据流
1.  **输入**: `Input` 结构体，包含 IP, Port, Banner, HTTP Headers/Body 等。
2.  **处理**: Service 并行（或串行）调用所有注册的 Engine。
3.  **匹配**: 
    *   `HTTPEngine` 遍历 `AssetFinger` 规则进行特征匹配。
    *   `ServiceEngine` 使用正则匹配 Banner，并提取版本号。
4.  **输出**: `Result` 结构体，包含最佳匹配 (`Best`) 和所有匹配 (`Matches`)。

## 4. 接口定义

### 4.1 Service 接口
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

### 4.2 输入输出模型
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

## 5. 规则文件格式

规则文件默认存放于 `rules/fingerprint/` 目录下，支持 `.json` 格式。
所有规则文件（无论文件名）必须包含顶层的 `type` 字段以区分规则类型。

### 5.1 HTTP 指纹规则 (`type: "http"`)
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

### 5.2 服务/CPE 指纹规则 (`type: "service"`)
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

## 6. 配置与使用

在 `config.yaml` 中配置规则目录：

```yaml
app:
  rules:
    root_path: "rules"
    fingerprint:
      dir: "fingerprint"
```

系统启动时会自动初始化 `FingerprintService` 并加载该目录下的所有规则。
同时，系统也会连接数据库，自动加载 `asset_finger` 和 `asset_cpe` 表中的规则。

## 7. Master-Agent 指纹同步机制

为了实现高效、可控的分布式指纹识别，我们采用 **"Master 管理，Agent 消费"** 的模式。

### 7.1 核心流程
1.  **入库 (Ingest & Manage)**
    *   Master 端的 `converters` 模块将第三方指纹库转换为统一格式，存入数据库 (`FingerprintRules` 表)。
    *   管理员可在 Master 界面对规则进行启用/禁用、修正等操作。

2.  **打包 (Build Snapshot)**
    *   Master 定期或按需（当规则变更时）从数据库生成**指纹库快照 (Snapshot)**。
    *   Snapshot 是一个序列化后的文件（JSON/MsgPack），包含所有当前启用的规则。
    *   为 Snapshot 生成一个全局唯一的 `version_hash` (如 MD5)。

3.  **分发 (Distribute)**
    *   **心跳检测**: Agent 每分钟向 Master 发送心跳时，Master 返回当前最新的 `version_hash`。
    *   **增量/全量更新**: Agent 比较本地 hash 与 Master hash。如果不一致，调用 API 拉取最新的 Snapshot。

4.  **执行 (Execute)**
    *   Agent 下载 Snapshot 后，将其**全量加载到内存**。
    *   扫描过程中，所有指纹匹配均在内存中完成，无网络 IO 开销。

### 7.2 API 设计
*   `GET /api/v1/agent/fingerprint/version`: 获取当前指纹库的版本 Hash。
*   `GET /api/v1/agent/fingerprint/download`: 下载最新的指纹库快照文件。

### 7.3 端云同构
Agent 和 Master 复用同一套核心代码 (`pkg/fingerprint`)。Master 用于规则管理、单点测试和离线治理（存量资产再识别），Agent 用于大规模分布式扫描。

## 8. 指纹生效完整过程

### 8.1 整体架构流程

```
指纹规则文件 (JSON)
    ↓
HTTPEngine.LoadRules()
    ↓
compileRule() - 规则编译
    ↓
CompiledRule - 编译后的规则
    ↓
convertInputToMap() - 输入数据转换
    ↓
matcher.Match() - 匹配器执行
    ↓
FingerprintMatcher.ProcessBatch() - 批量处理
    ↓
识别结果
```

### 8.2 规则编译过程详解

#### 8.2.1 规则定义示例

```json
{
  "name": "WordPress",
  "status_code": "200",
  "url": "/wp-login.php",
  "title": "Log In",
  "match": "regex:wp-.*",
  "enabled": true,
  "source": "system"
}
```

#### 8.2.2 compileRule() 编译逻辑

`compileRule()` 函数将简单的规则字段自动转换为复杂的匹配条件：

```go
func compileRule(rule asset.AssetFinger) CompiledRule {
    var conditions []matcher.MatchRule

    // 1. status_code 字段 → equals 匹配
    if rule.StatusCode != "" {
        conditions = append(conditions, matcher.MatchRule{
            Field:    "status_code",
            Operator: "equals",
            Value:    rule.StatusCode,
        })
    }

    // 2. title 字段 → contains 匹配（忽略大小写）
    if rule.Title != "" {
        conditions = append(conditions, matcher.MatchRule{
            Field:      "title",
            Operator:   "contains",
            Value:      rule.Title,
            IgnoreCase: true,
        })
    }

    // 3. header 字段 → all_headers contains 匹配
    if rule.Header != "" {
        conditions = append(conditions, matcher.MatchRule{
            Field:      "all_headers",
            Operator:   "contains",
            Value:      rule.Header,
            IgnoreCase: true,
        })
    }

    // 4. server 字段 → OR 逻辑匹配
    if rule.Server != "" {
        conditions = append(conditions, matcher.MatchRule{
            Or: []matcher.MatchRule{
                {Field: "server", Operator: "contains", Value: rule.Server, IgnoreCase: true},
                {Field: "all_headers", Operator: "contains", Value: rule.Server, IgnoreCase: true},
            },
        })
    }

    // 5. x_powered_by 字段 → OR 逻辑匹配
    if rule.XPoweredBy != "" {
        conditions = append(conditions, matcher.MatchRule{
            Or: []matcher.MatchRule{
                {Field: "x_powered_by", Operator: "contains", Value: rule.XPoweredBy, IgnoreCase: true},
                {Field: "all_headers", Operator: "contains", Value: rule.XPoweredBy, IgnoreCase: true},
            },
        })
    }

    // 6. body/response/footer/subtitle 字段 → body contains 匹配
    for _, val := range []string{rule.Body, rule.Response, rule.Footer, rule.Subtitle} {
        if val != "" {
            conditions = append(conditions, matcher.MatchRule{
                Field:      "body",
                Operator:   "contains",
                Value:      val,
                IgnoreCase: true,
            })
        }
    }

    // 7. match 字段 → 高级规则或正则
    if rule.Match != "" {
        // 尝试解析为 JSON MatchRule (复杂逻辑)
        if strings.HasPrefix(strings.TrimSpace(rule.Match), "{") {
            var complexRule matcher.MatchRule
            if err := json.Unmarshal([]byte(rule.Match), &complexRule); err == nil {
                conditions = append(conditions, complexRule)
            }
        } else {
            // 默认为正则表达式
            if re, err := regexp.Compile(rule.Match); err == nil {
                conditions = append(conditions, matcher.MatchRule{
                    Field:    "all_response",
                    Operator: "regex",
                    Value:    re,
                })
            }
        }
    }

    // 所有条件通过 AND 连接
    return CompiledRule{
        Original: rule,
        Matcher: matcher.MatchRule{
            And: conditions,
        },
    }
}
```

#### 8.2.3 编译后的规则结构

上述 WordPress 规则编译后变成：

```go
{
    And: [
        {
            Field:    "status_code",
            Operator: "equals",
            Value:    "200"
        },
        {
            Field:      "title",
            Operator:   "contains",
            Value:      "Log In",
            IgnoreCase: true
        },
        {
            Field:    "all_response",
            Operator: "regex",
            Value:    regexp.MustCompile("wp-.*")
        }
    ]
}
```

### 8.3 输入数据转换详解

#### 8.3.1 HTTP 响应示例

假设扫描器访问 `http://target.com/wp-login.php`，得到以下 HTTP 响应：

```http
HTTP/1.1 200 OK
Server: nginx/1.18.0
Content-Type: text/html; charset=UTF-8
X-Powered-By: PHP/7.4.3

<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>WordPress › Log In</title>
    <link rel="stylesheet" href="https://target.com/wp-admin/css/login.min.css">
</head>
<body class="login">
    <h1><a href="https://wordpress.org/">Powered by WordPress</a></h1>
    <form name="loginform" id="loginform" action="/wp-login.php" method="post">
        ...
    </form>
</body>
</html>
```

#### 8.3.2 convertInputToMap() 转换逻辑

```go
func convertInputToMap(input *fingerprint.Input) map[string]interface{} {
    data := make(map[string]interface{})

    // 1. 基础字段
    data["body"] = input.Body  // 原始响应体

    // 2. 提取标题
    data["title"] = extractTitle(input.Body)
    // extractTitle 从 HTML 中提取 <title> 标签内容
    // 返回: "WordPress › Log In"

    // 3. Headers
    data["headers"] = input.Headers

    // 4. 构建 all_headers 字符串
    var allHeadersBuilder strings.Builder
    for k, v := range input.Headers {
        allHeadersBuilder.WriteString(k)
        allHeadersBuilder.WriteString(": ")
        allHeadersBuilder.WriteString(v)
        allHeadersBuilder.WriteString("\n")
    }
    allHeadersStr := allHeadersBuilder.String()
    data["all_headers"] = allHeadersStr
    // 返回: "Server: nginx/1.18.0\nContent-Type: text/html; charset=UTF-8\nX-Powered-By: PHP/7.4.3\n"

    // 5. 特殊 Header 提取（方便快速访问）
    if val, ok := input.Headers["Server"]; ok {
        data["server"] = val
    }
    if val, ok := input.Headers["X-Powered-By"]; ok {
        data["x_powered_by"] = val
    }

    // 6. All Response (Headers + Body)
    data["all_response"] = allHeadersStr + "\n" + input.Body

    return data
}
```

#### 8.3.3 转换后的数据结构

```go
{
    "body": "<!DOCTYPE html>\n<html>\n<head>\n    <meta charset=\"UTF-8\">\n    <title>WordPress › Log In</title>\n    <link rel=\"stylesheet\" href=\"https://target.com/wp-admin/css/login.min.css\">\n</head>\n<body class=\"login\">\n    <h1><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n    <form name=\"loginform\" id=\"loginform\" action=\"/wp-login.php\" method=\"post\">\n        ...\n    </form>\n</body>\n</html>",
    "title": "WordPress › Log In",
    "headers": {
        "Server": "nginx/1.18.0",
        "Content-Type": "text/html; charset=UTF-8",
        "X-Powered-By": "PHP/7.4.3"
    },
    "all_headers": "Server: nginx/1.18.0\nContent-Type: text/html; charset=UTF-8\nX-Powered-By: PHP/7.4.3\n",
    "server": "nginx/1.18.0",
    "x_powered_by": "PHP/7.4.3",
    "all_response": "Server: nginx/1.18.0\nContent-Type: text/html; charset=UTF-8\nX-Powered-By: PHP/7.4.3\n<!DOCTYPE html>\n<html>\n<head>\n    <meta charset=\"UTF-8\">\n    <title>WordPress › Log In</title>\n    <link rel=\"stylesheet\" href=\"https://target.com/wp-admin/css/login.min.css\">\n</head>\n<body class=\"login\">\n    <h1><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n    <form name=\"loginform\" id=\"loginform\" action=\"/wp-login.php\" method=\"post\">\n        ...\n    </form>\n</body>\n</html>"
}
```

### 8.4 匹配过程详解

#### 8.4.1 HTTPEngine.Match() 执行逻辑

```go
func (e *HTTPEngine) Match(input *fingerprint.Input) ([]fingerprint.Match, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    var matches []fingerprint.Match

    // 1. 准备匹配数据
    data := convertInputToMap(input)

    // 2. 遍历所有编译后的规则进行匹配
    for _, rule := range e.rules {
        // 3. 使用匹配器评估数据是否符合规则
        matched, err := matcher.Match(data, rule.Matcher)
        if err != nil {
            continue
        }

        // 4. 匹配成功，创建匹配结果
        if matched {
            matches = append(matches, fingerprint.Match{
                Product:    rule.Original.Name,
                Vendor:     guessVendor(rule.Original.Name),
                Type:       "app",
                CPE:        generateCPE(rule.Original.Name),
                Confidence: 95,
                Source:     "http_engine",
            })
        }
    }

    return matches, nil
}
```

#### 8.4.2 matcher.Match() 匹配逻辑

```go
func Match(data interface{}, rule MatchRule) (bool, error) {
    // 1. 处理逻辑节点 (Branch)
    if len(rule.And) > 0 {
        for _, subRule := range rule.And {
            matched, err := Match(data, subRule)
            if err != nil {
                return false, err
            }
            if !matched {
                return false, nil  // And 只要有一个不匹配，整体就不匹配
            }
        }
        return true, nil  // 所有都匹配
    }

    if len(rule.Or) > 0 {
        for _, subRule := range rule.Or {
            matched, err := Match(data, subRule)
            if err != nil {
                return false, err
            }
            if matched {
                return true, nil  // Or 只要有一个匹配，整体就匹配
            }
        }
        return false, nil  // 所有都不匹配
    }

    // 2. 处理条件节点 (Leaf)
    if rule.Field == "" && rule.Operator == "" {
        return true, nil  // 空规则，默认匹配
    }

    // 3. 获取字段值
    fieldValue, exists := getFieldValue(data, rule.Field)
    if !exists {
        return false, nil
    }

    // 4. 执行具体匹配逻辑
    return evaluateCondition(fieldValue, rule.Operator, rule.Value, rule.IgnoreCase)
}
```

#### 8.4.3 逐条件匹配示例

**条件1：status_code equals "200"**
```go
fieldValue = "200"  // 从 data["status_code"] 获取
operator = "equals"
expected = "200"
matched = ("200" == "200")  // ✅ true
```

**条件2：title contains "Log In"**
```go
fieldValue = "WordPress › Log In"  // 从 data["title"] 获取
operator = "contains"
expected = "Log In"
ignoreCase = true
matched = strings.Contains("WordPress › Log In", "Log In")  // ✅ true
```

**条件3：all_response regex "wp-.*"**
```go
fieldValue = "Server: nginx/1.18.0\nContent-Type: text/html; charset=UTF-8\nX-Powered-By: PHP/7.4.3\n<!DOCTYPE html>\n<html>\n<head>\n    <meta charset=\"UTF-8\">\n    <title>WordPress › Log In</title>\n    <link rel=\"stylesheet\" href=\"https://target.com/wp-admin/css/login.min.css\">\n</head>\n<body class=\"login\">\n    <h1><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n    <form name=\"loginform\" id=\"loginform\" action=\"/wp-login.php\" method=\"post\">\n        ...\n    </form>\n</body>\n</html>"
operator = "regex"
expected = regexp.MustCompile("wp-.*")
matched = regexp.MustCompile("wp-.*").MatchString(fieldValue)  // ✅ true (匹配 "wp-admin", "wp-login.php", "wp-content" 等)
```

#### 8.4.4 最终匹配结果

```go
// 所有条件都匹配
matched = true && true && true  // ✅ true

// 生成匹配结果
matches = append(matches, fingerprint.Match{
    Product:    "WordPress",
    Vendor:     "wordpress",
    Type:       "app",
    CPE:        "cpe:2.3:a:wordpress:wordpress:*:*:*:*:*:*:*:*",
    Confidence: 95,
    Source:     "http_engine",
})
```

### 8.5 字段与匹配逻辑映射表

| 规则字段 | 匹配字段 | 操作符 | 大小写敏感 | 说明 |
|---------|---------|--------|----------|------|
| `status_code` | `status_code` | `equals` | 是 | 精确匹配 HTTP 状态码 |
| `title` | `title` | `contains` | 否 | 包含匹配页面标题（从 Body 提取） |
| `header` | `all_headers` | `contains` | 否 | 包含匹配任意 HTTP 响应头 |
| `server` | `server` 或 `all_headers` | `contains` | 否 | OR 逻辑匹配 Server 头 |
| `x_powered_by` | `x_powered_by` 或 `all_headers` | `contains` | 否 | OR 逻辑匹配 X-Powered-By 头 |
| `body` | `body` | `contains` | 否 | 包含匹配响应体 |
| `response` | `body` | `contains` | 否 | 包含匹配响应体（别名） |
| `footer` | `body` | `contains` | 否 | 包含匹配响应体（别名） |
| `subtitle` | `body` | `contains` | 否 | 包含匹配响应体（别名） |
| `match` | `all_response` | `regex` 或 JSON | 取决于规则 | 正则表达式或复杂规则 |
| `url` | - | - | - | **不参与匹配**，仅用于文档或扫描时指定访问路径 |

### 8.6 支持的匹配操作符

#### 8.6.1 字符串操作符
- `equals`: 精确匹配
- `not_equals`: 精确不匹配
- `contains`: 包含匹配
- `not_contains`: 不包含匹配
- `starts_with`: 以...开头
- `ends_with`: 以...结尾
- `regex`: 正则表达式匹配
- `like`: SQL LIKE 风格匹配（支持 `%` 和 `_`）

#### 8.6.2 集合操作符
- `in`: 在列表中
- `not_in`: 不在列表中
- `list_contains`: 列表包含某个值

#### 8.6.3 数值操作符
- `greater_than`: 大于
- `less_than`: 小于
- `greater_than_or_equal`: 大于等于
- `less_than_or_equal`: 小于等于

#### 8.6.4 存在性操作符
- `exists`: 字段存在
- `is_null`: 字段为空
- `is_not_null`: 字段不为空

#### 8.6.5 网络操作符
- `cidr`: IP 地址在 CIDR 范围内

### 8.7 为什么只需要定义"识别什么"不需要定义"怎么识别"

#### 8.7.1 设计理念

NeoScan 指纹系统采用**声明式规则**设计，用户只需要声明"要识别什么"，系统自动决定"怎么识别"。

#### 8.7.2 核心优势

**1. 简化规则定义**
```json
// 用户只需要这样写
{
  "name": "WordPress",
  "title": "Log In",
  "status_code": "200"
}

// 而不需要这样写
{
  "name": "WordPress",
  "match_logic": [
    {
      "field": "title",
      "operator": "contains",
      "value": "Log In",
      "ignore_case": true,
      "extract_method": "html_parser"
    },
    {
      "field": "status_code",
      "operator": "equals",
      "value": "200",
      "data_type": "string"
    }
  ]
}
```

**2. 引擎自动处理**
- `title` 字段 → 引擎自动知道要提取 HTML 标题
- `header` 字段 → 引擎自动知道要匹配所有响应头
- `server` 字段 → 引擎自动知道要匹配 Server 头（支持大小写不敏感）
- `match` 字段 → 引擎自动识别是正则还是 JSON 规则

**3. 灵活性与可维护性**
- 匹配逻辑集中在引擎中，易于维护和升级
- 规则文件简单直观，易于理解和编写
- 支持从简单字段匹配到复杂正则/JSON 规则的平滑过渡

**4. 性能优化**
- 引擎可以预编译正则表达式
- 引擎可以优化匹配算法（如 AC 自动机）
- 引擎可以缓存匹配结果

#### 8.7.3 实现机制

**规则编译层**
```go
// compileRule 函数负责将简单字段转换为复杂匹配条件
func compileRule(rule asset.AssetFinger) CompiledRule {
    var conditions []matcher.MatchRule

    // 根据字段类型自动选择匹配方式
    if rule.Title != "" {
        // 标题 → contains 匹配（自动处理 HTML 提取）
        conditions = append(conditions, matcher.MatchRule{
            Field:      "title",
            Operator:   "contains",
            Value:      rule.Title,
            IgnoreCase: true,  // 自动添加大小写不敏感
        })
    }

    if rule.Server != "" {
        // Server → OR 逻辑匹配（自动处理多个匹配位置）
        conditions = append(conditions, matcher.MatchRule{
            Or: []matcher.MatchRule{
                {Field: "server", Operator: "contains", Value: rule.Server, IgnoreCase: true},
                {Field: "all_headers", Operator: "contains", Value: rule.Server, IgnoreCase: true},
            },
        })
    }

    // ... 其他字段处理

    // 所有条件通过 AND 连接
    return CompiledRule{
        Original: rule,
        Matcher: matcher.MatchRule{And: conditions},
    }
}
```

**数据转换层**
```go
// convertInputToMap 函数负责将 HTTP 响应转换为统一的数据结构
func convertInputToMap(input *fingerprint.Input) map[string]interface{} {
    data := make(map[string]interface{})

    // 自动提取标题
    data["title"] = extractTitle(input.Body)

    // 自动构建 all_headers
    data["all_headers"] = buildAllHeadersString(input.Headers)

    // 自动提取特殊 Header
    data["server"] = input.Headers["Server"]
    data["x_powered_by"] = input.Headers["X-Powered-By"]

    // ... 其他转换

    return data
}
```

**匹配器层**
```go
// matcher.Match 函数负责执行匹配逻辑
func Match(data interface{}, rule MatchRule) (bool, error) {
    // 自动处理 AND/OR 逻辑
    if len(rule.And) > 0 {
        for _, subRule := range rule.And {
            matched, err := Match(data, subRule)
            if !matched {
                return false, nil
            }
        }
        return true, nil
    }

    // 自动处理各种操作符
    return evaluateCondition(fieldValue, rule.Operator, rule.Value, rule.IgnoreCase)
}
```

#### 8.7.4 对比示例

**传统命令式规则**
```json
{
  "name": "WordPress",
  "rules": [
    {
      "step1": "extract_title_from_html",
      "step2": "check_if_contains",
      "value": "Log In",
      "case_sensitive": false
    },
    {
      "step1": "get_status_code",
      "step2": "check_if_equals",
      "value": "200"
    }
  ]
}
```

**NeoScan 声明式规则**
```json
{
  "name": "WordPress",
  "title": "Log In",
  "status_code": "200"
}
```

**优势对比**：
- 传统方式：用户需要了解 HTML 解析、HTTP 协议、匹配算法等细节
- NeoScan 方式：用户只需要知道"WordPress 的登录页标题是 Log In，状态码是 200"

#### 8.7.5 扩展性

**简单规则 → 复杂规则**
```json
// 简单规则
{
  "name": "SimpleApp",
  "title": "Login"
}

// 复杂规则（使用 match 字段）
{
  "name": "ComplexApp",
  "match": {
    "and": [
      {"field": "title", "operator": "contains", "value": "Login"},
      {"field": "server", "operator": "regex", "value": "nginx.*"},
      {"or": [
        {"field": "status_code", "operator": "equals", "value": "200"},
        {"field": "status_code", "operator": "equals", "value": "301"}
      ]}
    ]
  }
}
```

从简单到复杂，用户可以根据需要选择合适的规则复杂度，而不需要学习新的语法。

#### 8.7.6 规则存储的两种方式

NeoScan 指纹系统支持两种规则存储方式，可以灵活选择：

**方式1：简单字段存储（推荐用于常见场景）**

```json
{
  "name": "WordPress",
  "status_code": "200",
  "title": "Log In",
  "match": "regex:wp-.*"
}
```

存储在数据库的各个字段中：
- `status_code` → "200"
- `title` → "Log In"
- `match` → "regex:wp-.*"

**方式2：完全使用 match 字段存储（推荐用于复杂场景）**

```json
{
  "name": "WordPress",
  "match": {
    "and": [
      {"field": "status_code", "operator": "equals", "value": "200"},
      {"field": "title", "operator": "contains", "value": "Log In", "ignore_case": true},
      {"field": "all_response", "operator": "regex", "value": "wp-.*"}
    ]
  }
}
```

存储在数据库中：
- `status_code` → "" (空)
- `title` → "" (空)
- `match` → `{"and": [...]}` (JSON字符串)

**两种方式的对比**

| 特性 | 简单字段方式 | match 字段方式 |
|------|------------|---------------|
| **规则文件** | 简单直观，易于理解 | 相对复杂，需要了解 JSON 语法 |
| **灵活性** | 有限，只能使用预定义字段和操作符 | 完全灵活，支持所有操作符和逻辑组合 |
| **逻辑组合** | 只能 AND 组合 | 支持 AND/OR 嵌套组合 |
| **高级操作符** | 不支持（如 cidr, list_contains） | 完全支持 |
| **适用场景** | 常见的简单匹配 | 复杂逻辑匹配 |
| **学习成本** | 低 | 中等 |
| **维护性** | 高（直观易懂） | 中等（需要理解 JSON 规则） |

**compileRule 函数的统一处理**

```go
func compileRule(rule asset.AssetFinger) CompiledRule {
    var conditions []matcher.MatchRule

    // 1. 处理简单字段（status_code, title, header等）
    if rule.StatusCode != "" {
        conditions = append(conditions, matcher.MatchRule{
            Field:    "status_code",
            Operator: "equals",
            Value:    rule.StatusCode,
        })
    }

    if rule.Title != "" {
        conditions = append(conditions, matcher.MatchRule{
            Field:      "title",
            Operator:   "contains",
            Value:      rule.Title,
            IgnoreCase: true,
        })
    }

    // ... 其他简单字段处理

    // 2. 处理 match 字段（复杂规则）
    if rule.Match != "" {
        // 尝试解析为 JSON MatchRule
        if strings.HasPrefix(strings.TrimSpace(rule.Match), "{") {
            var complexRule matcher.MatchRule
            if err := json.Unmarshal([]byte(rule.Match), &complexRule); err == nil {
                conditions = append(conditions, complexRule)
            }
        } else {
            // 默认为正则表达式
            if re, err := regexp.Compile(rule.Match); err == nil {
                conditions = append(conditions, matcher.MatchRule{
                    Field:    "all_response",
                    Operator: "regex",
                    Value:    re,
                })
            }
        }
    }

    // 所有条件通过 AND 连接
    return CompiledRule{
        Original: rule,
        Matcher: matcher.MatchRule{
            And: conditions,
        },
    }
}
```

**混合使用（最佳实践）**

```json
{
  "name": "WordPress",
  "status_code": "200",        // 简单字段
  "title": "Log In",            // 简单字段
  "match": "regex:wp-.*"        // match 字段（正则）
}
```

**实际应用建议**

1. **简单场景**：使用简单字段存储
   ```json
   {
     "name": "SimpleApp",
     "title": "Login"
   }
   ```

2. **复杂场景**：使用 match 字段存储
   ```json
   {
     "name": "ComplexApp",
     "match": {
       "and": [
         {"field": "title", "operator": "contains", "value": "Login"},
         {"field": "server", "operator": "regex", "value": "nginx.*"},
         {"or": [
           {"field": "status_code", "operator": "equals", "value": "200"},
           {"field": "status_code", "operator": "equals", "value": "301"}
         ]}
       ]
     }
   }
   ```

3. **混合场景**：简单字段 + match 字段
   ```json
   {
     "name": "WordPress",
     "status_code": "200",
     "title": "Log In",
     "match": "regex:wp-.*"
   }
   ```

**数据库存储示例**

**当前设计（混合存储）**

```sql
CREATE TABLE `asset_finger` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '指纹名称',
  `status_code` varchar(50) DEFAULT '' COMMENT 'HTTP状态码',
  `url` varchar(500) DEFAULT '' COMMENT 'URL路径',
  `title` varchar(255) COMMENT '网页标题',
  `subtitle` varchar(255) COMMENT '网页副标题',
  `footer` varchar(255) COMMENT '网页页脚',
  `header` varchar(255) DEFAULT '' COMMENT 'HTTP响应头',
  `response` varchar(1000) DEFAULT '' COMMENT 'HTTP响应内容',
  `server` varchar(500) DEFAULT '' COMMENT 'Server头',
  `x_powered_by` varchar(255) DEFAULT '' COMMENT 'X-Powered-By头',
  `body` varchar(255) COMMENT 'HTTP响应体',
  `match` varchar(255) COMMENT '匹配模式(如正则或JSON规则)',
  `enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `source` varchar(20) DEFAULT 'system' COMMENT '来源(system/custom)',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**未来优化（完全使用 match 字段）**

如果未来想统一使用 match 字段，可以：

```sql
CREATE TABLE `asset_finger_v2` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '指纹名称',
  `match_rule` text NOT NULL COMMENT 'JSON格式的匹配规则',
  `enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `source` varchar(20) DEFAULT 'system' COMMENT '来源(system/custom)',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

存储示例：
```json
{
  "name": "WordPress",
  "match_rule": "{\"and\":[{\"field\":\"status_code\",\"operator\":\"equals\",\"value\":\"200\"},{\"field\":\"title\",\"operator\":\"contains\",\"value\":\"Log In\",\"ignore_case\":true},{\"field\":\"all_response\",\"operator\":\"regex\",\"value\":\"wp-.*\"}]}"
}
```

**关键要点**

1. **是的，所有简单规则都可以写成 match 字段的 JSON 形式**
2. **当前设计支持两种方式混合使用**：
   - 简单字段：适合常见简单场景
   - match 字段：适合复杂场景
3. **compileRule 函数会自动处理两种方式**：
   - 简单字段 → 自动转换为匹配条件
   - match 字段 → 直接使用（JSON）或编译为正则
4. **建议**：
   - 简单规则使用简单字段（更直观）
   - 复杂规则使用 match 字段（更灵活）
   - 可以混合使用（最佳实践）

这种设计既保持了简单场景的易用性，又提供了复杂场景的灵活性！

### 8.8 完整匹配流程图

```
┌─────────────────────────────────────────────────────────────┐
│  1. 规则文件加载                                       │
│     example_neoscan_finger_cpe_rules.json                │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  2. HTTPEngine.LoadRules(path)                          │
│     - 读取 JSON 文件                                       │
│     - 解析规则结构                                         │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  3. compileRule(rule) - 规则编译                      │
│     - status_code → equals 匹配                          │
│     - title → contains 匹配                              │
│     - header → all_headers contains 匹配                  │
│     - match → regex 或 JSON 规则                          │
│     - 所有条件通过 AND 连接                                │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  4. CompiledRule - 编译后的规则                        │
│     { And: [条件1, 条件2, 条件3, ...] }                │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  5. HTTPEngine.Match(input) - 执行匹配                 │
│     - 遍历所有编译后的规则                                 │
│     - 对每个规则调用 matcher.Match()                       │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  6. convertInputToMap(input) - 输入数据转换             │
│     - 提取 HTML 标题                                       │
│     - 构建 all_headers 字符串                             │
│     - 提取特殊 Header                                      │
│     - 构建 all_response 字符串                              │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  7. matcher.Match(data, rule.Matcher) - 匹配器执行       │
│     - 处理 AND/OR 逻辑                                  │
│     - 获取字段值                                         │
│     - 执行具体匹配逻辑                                     │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  8. evaluateCondition(fieldValue, operator, value)        │
│     - equals, contains, regex, in, greater_than 等          │
│     - 返回匹配结果                                         │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  9. 匹配结果                                            │
│     - Product: "WordPress"                               │
│     - Vendor: "wordpress"                                │
│     - CPE: "cpe:2.3:a:wordpress:wordpress:*:*:*:*:*:*:*:*" │
│     - Confidence: 95                                      │
└─────────────────────────────────────────────────────────────┘
```

### 8.9 关键设计亮点总结

1. **规则与匹配逻辑分离**：规则文件只定义"匹配什么"，匹配逻辑由引擎实现
2. **自动规则编译**：将简单的字段定义转换为复杂的匹配条件
3. **灵活的匹配器**：支持多种操作符和逻辑组合（AND/OR）
4. **数据结构转换**：将 HTTP 响应转换为统一的 map 结构
5. **多引擎支持**：HTTP 引擎、Service 引擎等可以并存
6. **正则表达式支持**：`match` 字段支持正则表达式匹配
7. **JSON 规则支持**：`match` 字段支持 JSON 格式的复杂规则
8. **声明式设计**：用户只需要声明"要识别什么"，系统自动决定"怎么识别"
9. **性能优化**：预编译正则、优化匹配算法、缓存匹配结果
10. **可维护性**：匹配逻辑集中在引擎中，易于维护和升级
