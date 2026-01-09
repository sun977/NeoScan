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
