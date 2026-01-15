# 指纹规则库说明

本目录存放指纹识别服务的规则文件。NeoScan 的指纹识别服务支持 **数据库存储** 和 **文件加载** 两种方式，并将所有规则统一转换为内部标准格式进行匹配。

### 指纹规则的两种用途
1. Master内部加载规则：指纹识别服务（fingerprint service）从 rules/fingerprint 目录加载规则文件用于资产识别，这个是在Master内部运行时使用的。
2. Agent 下载规则：Master将rules/fingerprint目录下的规则文件打包成ZIP快照，供Agent下载使用。
这两个功能使用的是同一个目录，但有不同的用途：
- Master内部使用：通过LoadRules方法加载rules/fingerprint目录下的规则文件到内存中，用于二次离线指纹识别
- Agent下载使用：通过BuildSnapshot方法将DB中的规则存入 rules/fingerprint 目录，然后打包成ZIP快照，供Agent下载使用
所以 rules/fingerprint 既是Master内部加载规则的源目录，也是生成Agent下载快照的源目录。

3. 文件规则加载机制
- 路径解析：使用 GetFingerprintRulePath() 获取规则路径，默认是 rules/fingerprint
- 文件遍历：通过 listRuleFiles() 递归遍历指定目录下的所有文件
- 格式识别：每个引擎（如 HTTPEngine 和 ServiceEngine）会根据文件的 type 字段判断是否为自己处理的类型
- 规则加载：引擎根据 type 字段决定加载方式：
   - type: "http" → HTTPEngine 加载为HTTP指纹规则
   - type: "service" → ServiceEngine 加载为CPE服务规则
4. 目录结构与规则加载的关系
- 路径映射：rules/fingerprint 目录及其子目录（system/cms, system/service, custom/等）中的所有JSON文件都会被遍历加载
- 类型区分：通过文件内容中的 type 字段区分规则类型，而不是目录路径
- 引擎选择：不同类型的规则会被相应的引擎加载（HTTPEngine vs ServiceEngine）


## 1. HTTP 指纹规则 (HTTP Fingerprints)

HTTP 指纹主要用于识别 Web 应用、CMS、框架等。
引擎会自动将 Goby 等第三方格式转换为内部的 `AssetFinger` 结构。

### 1.1 Goby 兼容格式示例 (`goby.json`)
支持 Goby 的 JSON 格式 (v1 格式)，`rule` 字段为逻辑表达式字符串。

```json
{
  "rule": [
    {
      "name": "Apache-Tomcat",
      "level": "critical",
      "soft_hard": "2",
      "product": "Apache-Tomcat",
      "company": "Apache",
      "category": "Server",
      "rule": "body=\"Apache Tomcat\" || header=\"Apache-Coyote\" || title=\"Tomcat Admin\""
    },
    {
      "name": "ThinkPHP",
      "product": "ThinkPHP",
      "rule": "header=\"ThinkPHP\" || body=\"ThinkPHP\""
    }
  ]
}
```

### 1.2 内部通用格式示例 (`custom.json`)
直接对应数据库 `asset_finger` 表结构。
文件结构包含 `samples` 数组，每个元素包含 `name` 和 `rule` 对象。
类型 type: http (HTTP fingerprint) 区分为 HTTP 指纹。

```json
{
  "name": "Custom Rules",
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
    },
    {
      "name": "Nginx-Server",
      "rule": {
        "header": "Server: nginx",
        "match": "contains" 
      }
    },
    {
      "name": "WordPress",
      "rule": {
        "status_code": "200",
        "url": "",
        "title": "Just another WordPress site",
        "subtitle": "",
        "footer": "",
        "header": "",
        "response": "",
        "server": "",
        "x_powered_by": "",
        "match": "contains"
      }
    }
  ]
}
```

**字段说明 (AssetFinger):**
- `name`: 指纹名称 (通常由外层传入，也可在 rule 中定义)
- `url`: 请求路径 (默认为 /)
- `body`: 响应体包含的关键字
- `header`: 响应头包含的关键字
- `title`: 网页标题关键字
- `status_code`: 期望的 HTTP 状态码
- `match`: 匹配模式 (目前主要支持 contains)
- `server`: Server 头关键字
- `x_powered_by`: X-Powered-By 头关键字

---

## 2. CPE 服务指纹规则 (Service/CPE Fingerprints)

CPE 指纹主要用于识别端口服务协议、版本号等 (类似 Nmap)。
对应数据库 `asset_cpe` 表结构。

### 2.1 CPE 规则示例 (`services.json`)
类型 type: service (Service/CPE fingerprint) 区分为服务指纹。

```json
{
  "name": "Service Rules",
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
      "match_str": "(?i)Apache/([\\d\\.]+)",
      "vendor": "apache",
      "product": "http_server",
      "part": "a",
      "cpe": "cpe:2.3:a:apache:http_server:$1:*:*:*:*:*:*:*"
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

**字段说明 (AssetCPE):**
- `match_str`: 匹配 Banner 的正则表达式 (支持捕获组)
- `cpe`: 标准 CPE 标识符模板 (使用 `$1`, `$2` 引用正则捕获组)
- `vendor`: 厂商名称
- `product`: 产品名称
- `part`: 类型 (a: Application, o: OS, h: Hardware)
