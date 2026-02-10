# Web 指纹 vs 服务器指纹对比

本文档详细对比了 NeoScan 系统中 Web 指纹和服务器指纹的区别，帮助您理解两种指纹的使用场景和编写方法。

## 目录

- [核心区别](#核心区别)
- [详细对比](#详细对比)
- [使用场景](#使用场景)
- [服务器指纹编写指南](#服务器指纹编写指南)
- [实际示例](#实际示例)
- [最佳实践](#最佳实践)

## 核心区别

### 识别对象

| 指纹类型 | 识别对象 | 示例 |
|---------|---------|------|
| **Web 指纹** | Web 应用、CMS、框架 | WordPress、Joomla、Spring Boot、Django |
| **服务器指纹** | 服务器、服务、数据库 | Nginx、Apache、OpenSSH、MySQL、Redis |

### 输入数据

| 指纹类型 | 输入数据 | 数据来源 |
|---------|---------|---------|
| **Web 指纹** | HTTP 响应（状态码、响应头、响应体） | HTTP 请求 |
| **服务器指纹** | Banner 信息（服务返回的文本） | 服务连接 |

### 规则格式

| 指纹类型 | 规则格式 | 字段数量 |
|---------|---------|---------|
| **Web 指纹** | 简单字段 + match 字段 | 12 个字段 |
| **服务器指纹** | 单一 JSON 格式 | 5 个字段 |

### 匹配方式

| 指纹类型 | 匹配方式 | 逻辑组合 |
|---------|---------|---------|
| **Web 指纹** | 简单字段、正则表达式、JSON 规则 | 支持 AND/OR |
| **服务器指纹** | 仅正则表达式 | 不支持 |

### 返回结果

| 指纹类型 | Product | Vendor | Type | CPE | Version |
|---------|---------|--------|------|-----|---------|
| **Web 指纹** | `name` 字段 | 自动猜测 | 固定 `"app"` | 自动生成 | 不提取 |
| **服务器指纹** | `product` 字段 | 明确指定 | `part` 字段 | 精确控制 | 自动提取 |

## 详细对比

### 1. 输入数据对比

#### Web 指纹输入

```go
input := &fingerprint.Input{
    Target:     "example.com",
    Port:       80,
    Protocol:   "http",
    Headers: map[string]string{
        "Server":       "nginx/1.18.0",
        "X-Powered-By": "PHP/8.0",
    },
    Body: "<html><title>WordPress › Log In</title>...wp-content...</html>",
}
```

**特点**：
- 需要完整的 HTTP 响应
- 包含状态码、响应头、响应体
- 数据丰富，信息量大

#### 服务器指纹输入

```go
input := &fingerprint.Input{
    Target:   "example.com",
    Port:     22,
    Protocol: "ssh",
    Banner:   "SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5",
}
```

**特点**：
- 只需要 Banner 信息
- 数据简单，信息量小
- 专注于服务标识

### 2. 规则格式对比

#### Web 指纹规则

**格式1：简单字段**

```json
{
  "name": "WordPress",
  "status_code": "200",
  "title": "Log In",
  "body": "wp-content",
  "enabled": true,
  "source": "custom"
}
```

**格式2：match 字段**

```json
{
  "name": "WordPress",
  "match": {
    "and": [
      {"field": "status_code", "operator": "equals", "value": "200"},
      {"field": "title", "operator": "contains", "value": "Log In"},
      {"field": "body", "operator": "regex", "value": "wp-.*"}
    ]
  },
  "enabled": true,
  "source": "custom"
}
```

**特点**：
- 支持两种格式
- 字段丰富（12 个）
- 支持逻辑组合

#### 服务器指纹规则

```json
{
  "match_str": "(?i)^SSH-[\\d\\.]+-OpenSSH_([\\w\\.]+)",
  "vendor": "openbsd",
  "product": "openssh",
  "part": "a",
  "cpe": "cpe:2.3:a:openbsd:openssh:$1:*:*:*:*:*:*:*"
}
```

**特点**：
- 单一格式
- 字段精简（5 个）
- 支持占位符

### 3. 匹配方式对比

#### Web 指纹匹配

**方式1：简单字段匹配**

```go
// 直接匹配字段值
if input.StatusCode == "200" &&
   strings.Contains(input.Title, "Log In") &&
   strings.Contains(input.Body, "wp-content") {
    // 匹配成功
}
```

**方式2：正则表达式匹配**

```go
// 使用 match 字段的正则表达式
regex := regexp.MustCompile(`wp-.*`)
if regex.MatchString(input.Body) {
    // 匹配成功
}
```

**方式3：JSON 规则匹配**

```go
// 支持复杂的逻辑组合
match := &Match{
    And: []Match{
        {Field: "status_code", Operator: "equals", Value: "200"},
        {Field: "title", Operator: "contains", Value: "Log In"},
        {Or: []Match{
            {Field: "body", Operator: "contains", Value: "wp-content"},
            {Field: "body", Operator: "contains", Value: "wp-includes"},
        }}
    }
}
```

**特点**：
- 支持多种匹配方式
- 支持逻辑组合（AND/OR）
- 支持嵌套逻辑
- 灵活性高

#### 服务器指纹匹配

```go
// 编译正则表达式
regex := regexp.MustCompile(`(?i)^SSH-[\d\.]+-OpenSSH_([\w\.]+)`)

// 匹配 Banner
submatches := regex.FindStringSubmatch("SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5")

// 提取版本号
version := submatches[1]  // "8.2p1"

// 填充 CPE
cpe := "cpe:2.3:a:openbsd:openssh:$1:*:*:*:*:*:*:*"
cpe = strings.ReplaceAll(cpe, "$1", "8.2p1")
```

**特点**：
- 只能使用正则表达式
- 通过捕获组提取信息
- 占位符替换生成 CPE
- 简单直接

### 4. 返回结果对比

#### Web 指纹返回

```go
{
    Product:    "WordPress",
    Vendor:     "wordpress",
    Type:       "app",
    CPE:        "cpe:2.3:a:wordpress:wordpress:*:*:*:*:*:*:*:*",
    Version:    "",
    Confidence: 95,
    Source:     "http_engine"
}
```

**特点**：
- Product 来自 `name` 字段
- Vendor 自动猜测（小写）
- Type 固定为 `"app"`
- CPE 自动生成（模板格式）
- Version 不提取
- Confidence 较高（95）

#### 服务器指纹返回

```go
{
    Product:    "openssh",
    Vendor:     "openbsd",
    Type:       "a",
    CPE:        "cpe:2.3:a:openbsd:openssh:8.2p1:*:*:*:*:*:*:*",
    Version:    "8.2p1",
    Confidence: 90,
    Source:     "service_banner"
}
```

**特点**：
- Product 来自 `product` 字段
- Vendor 明确指定
- Type 来自 `part` 字段
- CPE 精确控制（占位符）
- Version 自动提取
- Confidence 略低（90）

### 5. 占位符机制对比

#### Web 指纹

**不支持占位符**

```json
{
  "name": "WordPress",
  "match": {
    "field": "body",
    "operator": "regex",
    "value": "wp-([\\d\\.]+)"
  }
}
```

**说明**：
- 虽然可以使用正则表达式捕获组
- 但不能使用 $1 占位符
- 版本号需要通过其他方式提取

#### 服务器指纹

**支持占位符**

```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**使用方式**：
```go
// 正则匹配
submatches := regex.FindStringSubmatch("nginx/1.18.0")
// 返回: ["nginx/1.18.0", "1.18.0"]

// 占位符替换
cpe := "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
cpe = strings.ReplaceAll(cpe, "$1", "1.18.0")
// 结果: "cpe:2.3:a:f5:nginx:1.18.0:*:*:*:*:*:*:*"
```

## 使用场景

### Web 指纹适用场景

✅ **推荐使用**：
- 识别 Web 应用（WordPress、Joomla、Drupal）
- 识别 Web 框架（Spring Boot、Django、Flask）
- 识别 CMS 系统（DedeCMS、Typecho）
- 识别编程语言（PHP、ASP.NET、Python）
- 识别中间件（Cloudflare、Nginx反向代理）
- 需要复杂的逻辑组合
- 需要匹配多个字段

❌ **不推荐使用**：
- 识别服务器软件（Nginx、Apache）
- 识别服务（OpenSSH、FTP）
- 识别数据库（MySQL、Redis）

### 服务器指纹适用场景

✅ **推荐使用**：
- 识别 Web 服务器（Nginx、Apache、IIS）
- 识别 SSH 服务（OpenSSH）
- 识别 FTP 服务（ProFTPD、vsftpd）
- 识别数据库（MySQL、PostgreSQL、Redis）
- 识别邮件服务（Postfix、Sendmail）
- 需要精确的版本号
- 需要标准化的 CPE

❌ **不推荐使用**：
- 识别 Web 应用（WordPress、Joomla）
- 识别 Web 框架（Spring Boot、Django）
- 需要复杂的逻辑组合
- 需要匹配多个字段

## 服务器指纹编写指南

### 规则文件格式

#### 格式1：数组格式（向后兼容）

```json
[
  {
    "match_str": "(?i)^SSH-[\\d\\.]+-OpenSSH_([\\w\\.]+)",
    "vendor": "openbsd",
    "product": "openssh",
    "part": "a",
    "cpe": "cpe:2.3:a:openbsd:openssh:$1:*:*:*:*:*:*:*"
  },
  {
    "match_str": "(?i)nginx/([\\d\\.]+)",
    "vendor": "f5",
    "product": "nginx",
    "part": "a",
    "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
  }
]
```

#### 格式2：标准格式（推荐）

```json
{
  "name": "Service CPE Rules",
  "version": "1.0",
  "type": "service",
  "samples": [
    {
      "match_str": "(?i)^SSH-[\\d\\.]+-OpenSSH_([\\w\\.]+)",
      "vendor": "openbsd",
      "product": "openssh",
      "part": "a",
      "cpe": "cpe:2.3:a:openbsd:openssh:$1:*:*:*:*:*:*:*"
    },
    {
      "match_str": "(?i)nginx/([\\d\\.]+)",
      "vendor": "f5",
      "product": "nginx",
      "part": "a",
      "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
    }
  ]
}
```

### 规则字段说明

| 字段名 | 类型 | 必填 | 说明 | 示例 |
|-------|------|------|------|------|
| `match_str` | string | 是 | 正则表达式，用于匹配 Banner | `"(?i)nginx/([\\d\\.]+)"` |
| `vendor` | string | 是 | 供应商名称 | `"f5"` |
| `product` | string | 是 | 产品名称 | `"nginx"` |
| `part` | string | 是 | CPE 组件类型 | `"a"` |
| `cpe` | string | 否 | 目标 CPE，可含占位符 | `"cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"` |

### CPE 组件类型（part）

| 值 | 说明 | 示例 |
|----|------|------|
| `a` | 应用程序 | nginx, Apache, OpenSSH |
| `o` | 操作系统 | Ubuntu, Debian, Windows |
| `h` | 硬件 | Cisco, Juniper |

### CPE 格式说明

CPE (Common Platform Enumeration) 是一种标准化的平台标识格式。

**格式**：
```
cpe:2.3:part:vendor:product:version:update:edition:language:sw_edition:target_sw:target_hw:other
```

**示例**：
```
cpe:2.3:a:f5:nginx:1.18.0:*:*:*:*:*:*:*
```

**说明**：
- `cpe:2.3` - CPE 版本
- `a` - 应用程序
- `f5` - 供应商
- `nginx` - 产品
- `1.18.0` - 版本号
- `*` - 其他字段使用通配符

### 正则表达式技巧

#### 1. 忽略大小写

使用 `(?i)` 标志忽略大小写。

```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)"
}
```

#### 2. 匹配开头

使用 `^` 匹配字符串开头。

```json
{
  "match_str": "^SSH-[\\d\\.]+-OpenSSH_([\\w\\.]+)"
}
```

#### 3. 匹配数字

使用 `\\d` 匹配数字。

```json
{
  "match_str": "nginx/([\\d\\.]+)"
}
```

#### 4. 匹配单词字符

使用 `\\w` 匹配单词字符。

```json
{
  "match_str": "OpenSSH_([\\w\\.]+)"
}
```

#### 5. 匹配点号

使用 `\\.` 匹配点号。

```json
{
  "match_str": "nginx/([\\d\\.]+)"
}
```

### 占位符使用

#### 基本用法

```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**说明**：
- `$1` 表示第一个捕获组
- 系统会自动替换为匹配到的版本号

#### 多个占位符

```json
{
  "match_str": "(?i)nginx/([\\d]+)\\.([\\d]+)\\.([\\d]+)",
  "cpe": "cpe:2.3:a:f5:nginx:$1.$2.$3:*:*:*:*:*:*:*"
}
```

**说明**：
- `$1` 表示第一个捕获组（主版本号）
- `$2` 表示第二个捕获组（次版本号）
- `$3` 表示第三个捕获组（修订版本号）

#### 版本号提取

```go
// 系统自动提取第一个捕获组作为版本号
version := submatches[1]  // "1.18.0"
```

## 实际示例

### 示例1：识别 Nginx

#### Banner 信息

```
HTTP/1.1 200 OK
Server: nginx/1.18.0
```

#### 规则

```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

#### 匹配过程

1. **编译正则表达式**
   ```go
   regex := regexp.MustCompile(`(?i)nginx/([\d\.]+)`)
   ```

2. **匹配 Banner**
   ```go
   submatches := regex.FindStringSubmatch("nginx/1.18.0")
   // 返回: ["nginx/1.18.0", "1.18.0"]
   ```

3. **提取版本号**
   ```go
   version := submatches[1]  // "1.18.0"
   ```

4. **填充 CPE**
   ```go
   cpe := "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
   cpe = strings.ReplaceAll(cpe, "$1", "1.18.0")
   // 结果: "cpe:2.3:a:f5:nginx:1.18.0:*:*:*:*:*:*:*"
   ```

5. **生成匹配结果**
   ```go
   {
     Product: "nginx",
     Vendor: "f5",
     Type: "a",
     CPE: "cpe:2.3:a:f5:nginx:1.18.0:*:*:*:*:*:*:*",
     Version: "1.18.0",
     Confidence: 90,
     Source: "service_banner"
   }
   ```

### 示例2：识别 OpenSSH

#### Banner 信息

```
SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5
```

#### 规则

```json
{
  "match_str": "(?i)^SSH-[\\d\\.]+-OpenSSH_([\\w\\.]+)",
  "vendor": "openbsd",
  "product": "openssh",
  "part": "a",
  "cpe": "cpe:2.3:a:openbsd:openssh:$1:*:*:*:*:*:*:*"
}
```

#### 匹配过程

1. **编译正则表达式**
   ```go
   regex := regexp.MustCompile(`(?i)^SSH-[\d\.]+-OpenSSH_([\w\.]+)`)
   ```

2. **匹配 Banner**
   ```go
   submatches := regex.FindStringSubmatch("SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5")
   // 返回: ["SSH-2.0-OpenSSH_8.2p1", "8.2p1"]
   ```

3. **提取版本号**
   ```go
   version := submatches[1]  // "8.2p1"
   ```

4. **填充 CPE**
   ```go
   cpe := "cpe:2.3:a:openbsd:openssh:$1:*:*:*:*:*:*:*"
   cpe = strings.ReplaceAll(cpe, "$1", "8.2p1")
   // 结果: "cpe:2.3:a:openbsd:openssh:8.2p1:*:*:*:*:*:*:*"
   ```

5. **生成匹配结果**
   ```go
   {
     Product: "openssh",
     Vendor: "openbsd",
     Type: "a",
     CPE: "cpe:2.3:a:openbsd:openssh:8.2p1:*:*:*:*:*:*:*",
     Version: "8.2p1",
     Confidence: 90,
     Source: "service_banner"
   }
   ```

### 示例3：识别 Apache

#### Banner 信息

```
HTTP/1.1 200 OK
Server: Apache/2.4.41 (Ubuntu)
```

#### 规则

```json
{
  "match_str": "(?i)Apache/([\\d\\.]+)",
  "vendor": "apache",
  "product": "http_server",
  "part": "a",
  "cpe": "cpe:2.3:a:apache:http_server:$1:*:*:*:*:*:*:*"
}
```

#### 匹配过程

1. **编译正则表达式**
   ```go
   regex := regexp.MustCompile(`(?i)Apache/([\d\.]+)`)
   ```

2. **匹配 Banner**
   ```go
   submatches := regex.FindStringSubmatch("Apache/2.4.41 (Ubuntu)")
   // 返回: ["Apache/2.4.41", "2.4.41"]
   ```

3. **提取版本号**
   ```go
   version := submatches[1]  // "2.4.41"
   ```

4. **填充 CPE**
   ```go
   cpe := "cpe:2.3:a:apache:http_server:$1:*:*:*:*:*:*:*"
   cpe = strings.ReplaceAll(cpe, "$1", "2.4.41")
   // 结果: "cpe:2.3:a:apache:http_server:2.4.41:*:*:*:*:*:*:*"
   ```

5. **生成匹配结果**
   ```go
   {
     Product: "http_server",
     Vendor: "apache",
     Type: "a",
     CPE: "cpe:2.3:a:apache:http_server:2.4.41:*:*:*:*:*:*:*",
     Version: "2.4.41",
     Confidence: 90,
     Source: "service_banner"
   }
   ```

### 示例4：识别 MySQL

#### Banner 信息

```
5.7.33-0ubuntu0.18.04.1
```

#### 规则

```json
{
  "match_str": "([\\d\\.]+)",
  "vendor": "oracle",
  "product": "mysql",
  "part": "a",
  "cpe": "cpe:2.3:a:oracle:mysql:$1:*:*:*:*:*:*:*"
}
```

#### 匹配过程

1. **编译正则表达式**
   ```go
   regex := regexp.MustCompile(`([\d\.]+)`)
   ```

2. **匹配 Banner**
   ```go
   submatches := regex.FindStringSubmatch("5.7.33-0ubuntu0.18.04.1")
   // 返回: ["5.7.33", "5.7.33"]
   ```

3. **提取版本号**
   ```go
   version := submatches[1]  // "5.7.33"
   ```

4. **填充 CPE**
   ```go
   cpe := "cpe:2.3:a:oracle:mysql:$1:*:*:*:*:*:*:*"
   cpe = strings.ReplaceAll(cpe, "$1", "5.7.33")
   // 结果: "cpe:2.3:a:oracle:mysql:5.7.33:*:*:*:*:*:*:*"
   ```

5. **生成匹配结果**
   ```go
   {
     Product: "mysql",
     Vendor: "oracle",
     Type: "a",
     CPE: "cpe:2.3:a:oracle:mysql:5.7.33:*:*:*:*:*:*:*",
     Version: "5.7.33",
     Confidence: 90,
     Source: "service_banner"
   }
   ```

## 最佳实践

### 规则设计原则

#### 1. 选择独特的特征

**原则**：选择服务独有的特征，避免与其他服务混淆。

**示例**：

❌ **不好的做法**：
```json
{
  "match_str": "Server",
  "vendor": "unknown",
  "product": "unknown",
  "part": "a",
  "cpe": "cpe:2.3:a:unknown:unknown:$1:*:*:*:*:*:*:*"
}
```

**问题**："Server" 太通用，很多服务都有这个字符串。

✅ **好的做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**优点**："nginx/" 更独特，减少误报。

#### 2. 提取版本号

**原则**：使用正则表达式捕获组提取版本号。

**示例**：

❌ **不好的做法**：
```json
{
  "match_str": "nginx",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:*:*:*:*:*:*:*:*"
}
```

**问题**：没有提取版本号，CPE 不精确。

✅ **好的做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**优点**：提取版本号，CPE 更精确。

#### 3. 使用占位符

**原则**：使用占位符自动填充 CPE。

**示例**：

❌ **不好的做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:*:*:*:*:*:*:*:*"
}
```

**问题**：没有使用占位符，CPE 不精确。

✅ **好的做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**优点**：使用占位符，自动填充版本号。

#### 4. 忽略大小写

**原则**：使用 `(?i)` 标志忽略大小写。

**示例**：

❌ **不好的做法**：
```json
{
  "match_str": "nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**问题**：无法匹配 "NGINX" 或 "Nginx"。

✅ **好的做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**优点**：忽略大小写，匹配更全面。

### 规则编写技巧

#### 1. 从简单开始

**步骤**：
1. 先写一个简单的规则
2. 测试规则是否匹配
3. 逐步完善规则

**示例**：

**步骤1**：简单规则
```json
{
  "match_str": "nginx",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:*:*:*:*:*:*:*:*"
}
```

**步骤2**：添加版本号
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**步骤3**：优化正则表达式
```json
{
  "match_str": "(?i)^Server: nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

#### 2. 测试规则

**方法**：
1. 使用真实 Banner 测试
2. 检查匹配结果
3. 验证 CPE 生成

**示例**：

```bash
# 测试 Nginx 规则
Banner: "nginx/1.18.0"
Expected: Product=nginx, Version=1.18.0, CPE=cpe:2.3:a:f5:nginx:1.18.0:*:*:*:*:*:*:*
```

#### 3. 参考现有规则

**方法**：
1. 查看系统默认规则
2. 参考类似服务的规则
3. 修改适配目标服务

**示例**：

**参考 Nginx 规则**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**修改为 Apache 规则**：
```json
{
  "match_str": "(?i)Apache/([\\d\\.]+)",
  "vendor": "apache",
  "product": "http_server",
  "part": "a",
  "cpe": "cpe:2.3:a:apache:http_server:$1:*:*:*:*:*:*:*"
}
```

### 常见错误

#### 1. 使用过于通用的特征

❌ **错误示例**：
```json
{
  "match_str": "Server",
  "vendor": "unknown",
  "product": "unknown",
  "part": "a",
  "cpe": "cpe:2.3:a:unknown:unknown:$1:*:*:*:*:*:*:*"
}
```

**问题**："Server" 太通用，会匹配很多服务。

✅ **正确做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

#### 2. 忽略大小写

❌ **错误示例**：
```json
{
  "match_str": "nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**问题**：无法匹配 "NGINX" 或 "Nginx"。

✅ **正确做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

#### 3. 正则表达式错误

❌ **错误示例**：
```json
{
  "match_str": "nginx/([\d\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**问题**：JSON 中需要双反斜杠。

✅ **正确做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

#### 4. 忘记提取版本号

❌ **错误示例**：
```json
{
  "match_str": "(?i)nginx",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:*:*:*:*:*:*:*:*"
}
```

**问题**：没有提取版本号，CPE 不精确。

✅ **正确做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

#### 5. CPE 格式错误

❌ **错误示例**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

**问题**：CPE 格式错误，缺少 `part` 字段。

✅ **正确做法**：
```json
{
  "match_str": "(?i)nginx/([\\d\\.]+)",
  "vendor": "f5",
  "product": "nginx",
  "part": "a",
  "cpe": "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*"
}
```

## 总结

### Web 指纹 vs 服务器指纹

| 特性 | Web 指纹 | 服务器指纹 |
|------|---------|-----------|
| **识别对象** | Web 应用、CMS、框架 | 服务器、服务、数据库 |
| **输入数据** | HTTP 响应 | Banner 信息 |
| **规则格式** | 简单字段 + match 字段 | 单一 JSON 格式 |
| **字段数量** | 12 个字段 | 5 个字段 |
| **匹配方式** | 简单字段、正则表达式、JSON 规则 | 仅正则表达式 |
| **逻辑组合** | 支持 AND/OR | 不支持 |
| **占位符** | 不支持 | 支持（$1, $2） |
| **Product** | `name` 字段 | `product` 字段 |
| **Vendor** | 自动猜测 | 明确指定 |
| **Type** | 固定 `"app"` | `part` 字段 |
| **CPE** | 自动生成 | 精确控制 |
| **Version** | 不提取 | 自动提取 |
| **Confidence** | 95 | 90 |

### 使用建议

| 识别对象 | 推荐使用 |
|---------|---------|
| Nginx、Apache、IIS | 服务器指纹 |
| OpenSSH、FTP、MySQL | 服务器指纹 |
| WordPress、Joomla | Web 指纹 |
| Spring Boot、Django | Web 指纹 |
| Redis、PostgreSQL | 服务器指纹 |

### 下一步

- 📖 [快速开始](./00_快速开始.md) - 了解 Web 指纹规则
- 📖 [服务器指纹识别原理](./07_服务器指纹识别原理.md) - 深入了解服务器指纹
- ✍️ [简单规则编写](./02_简单规则编写.md) - 学习编写 Web 指纹规则
- 🚀 [高级规则编写](./03_高级规则编写.md) - 学习编写高级 Web 指纹规则
