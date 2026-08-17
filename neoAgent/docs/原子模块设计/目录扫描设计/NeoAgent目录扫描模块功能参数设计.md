# NeoAgent 目录扫描模块功能参数设计

**日期**: 2026-08-14
**状态**: 设计讨论阶段
**参考**: dirsearch v0.4.3 (https://github.com/maurosoria/dirsearch)

---

## 1. dirsearch 完整参数参考

### 1.1 目标输入（Mandatory）

| 参数 | 简写 | 说明 |
|------|------|------|
| `--url` | `-u` | 目标 URL（可多次指定） |
| `--urls-file` | `-l` | URL 列表文件 |
| `--stdin` | — | 从 STDIN 读取 URL |
| `--cidr` | — | CIDR 格式目标 |
| `--raw` | — | 从文件加载原始 HTTP 请求 |
| `--nmap-report` | — | 从 nmap 报告加载目标 |
| `--session` | `-s` | 会话文件（恢复扫描） |
| `--session-id` | — | 会话 ID（恢复扫描） |
| `--config` | — | 配置文件路径 |

### 1.2 字典配置（Dictionary Settings）

| 参数 | 简写 | 说明 |
|------|------|------|
| `--wordlists` | `-w` | 字典文件或目录（逗号分隔） |
| `--wordlist-categories` | — | 分类字典名称（common/conf/web/all） |
| `--wordlist-backend` | — | 字典后端：auto/python/native |
| `--wordlist-status` | — | 显示字典状态并退出 |
| `--wordlist-max-size` | — | 最大字典条目数（默认 500K） |
| `--extensions` | `-e` | 扩展列表（php,asp） |
| `--force-extensions` | `-f` | 对所有词追加扩展 |
| `--overwrite-extensions` | — | 覆盖已有扩展 |
| `--exclude-extensions` | — | 排除的扩展（css,js） |
| `--prefixes` | — | 路径前缀（admin_, /） |
| `--suffixes` | — | 路径后缀（%, .bak） |
| `--uppercase` | `-U` | 词表大写 |
| `--lowercase` | `-L` | 词表小写 |
| `--capital` | `-C` | 词表首字母大写 |

### 1.3 通用设置（General Settings）

| 参数 | 简写 | 说明 |
|------|------|------|
| `--threads` | `-t` | 线程数 |
| `--list-sessions` | — | 列出所有会话 |
| `--sessions-dir` | — | 会话目录 |
| `--async` | `-a` | 启用异步模式 |
| `--sync` / `--no-async` | — | 禁用异步 |
| `--recursive` | `-r` | 递归扫描 |
| `--deep-recursive` | — | 深度递归（每级目录） |
| `--force-recursive` | — | 强制递归（所有命中路径） |
| `--max-recursion-depth` | `-R` | 最大递归深度 |
| `--recursion-status` | — | 递归触发状态码 |
| `--filter-threshold` | — | 重复响应过滤阈值 |
| `--subdirs` | — | 扫描指定子目录 |
| `--exclude-subdirs` | — | 排除子目录 |
| `--include-status` | `-i` | 包含的状态码白名单 |
| `--exclude-status` | `-x` | 排除的状态码黑名单 |
| `--exclude-sizes` | — | 排除的响应大小 |
| `--exclude-text` | — | 排除的关键词（可多次） |
| `--exclude-regex` | — | 排除的正则表达式 |
| `--exclude-redirect` | — | 排除匹配的重定向 |
| `--exclude-response` | — | 排除与某页面相似的响应 |
| `--skip-on-status` | — | 命中某状态码跳过目标 |
| `--min-response-size` | — | 最小响应长度 |
| `--max-response-size` | — | 最大响应长度 |
| `--max-time` | — | 总扫描时间（秒） |
| `--target-max-time` | — | 单目标最大时间（秒） |
| `--exit-on-error` | — | 出错时退出 |

### 1.4 高级过滤（Advanced Filtering）— 8 维匹配器/过滤器

| 参数 | 简写 | 说明 |
|------|------|------|
| `--auto-calibration` | — | 强制通配符校准 |
| `--matcher-mode` / `--mmode` | — | 匹配器逻辑：and/or |
| `--filter-mode` / `--fmode` | — | 过滤器逻辑：and/or |
| `--match-status` / `--mc` | — | 匹配状态码 |
| `--filter-status` / `--fc` | — | 过滤状态码 |
| `--match-size` / `--ms` | — | 匹配响应大小 |
| `--filter-size` / `--fs` | — | 过滤响应大小 |
| `--match-words` / `--mw` | — | 匹配词数 |
| `--filter-words` / `--fw` | — | 过滤词数 |
| `--match-lines` / `--ml` | — | 匹配行数 |
| `--filter-lines` / `--fl` | — | 过滤行数 |
| `--match-regex` / `--mr` | — | 匹配响应体正则 |
| `--filter-regex` / `--fr` | — | 过滤响应体正则 |
| `--match-header` | — | 匹配响应头文本（可多次） |
| `--filter-header` | — | 过滤响应头文本（可多次） |
| `--match-header-regex` | — | 匹配响应头正则 |
| `--filter-header-regex` | — | 过滤响应头正则 |
| `--match-time` / `--mt` | — | 匹配响应时间（>100 或 <100） |
| `--filter-time` / `--ft` | — | 过滤响应时间 |

### 1.5 请求配置（Request Settings）

| 参数 | 简写 | 说明 |
|------|------|------|
| `--http-method` | `-m` | HTTP 方法（默认 GET） |
| `--request-backend` | — | 请求后端：python/native |
| `--data` | `-d` | HTTP 请求数据 |
| `--data-file` | — | HTTP 请求数据文件 |
| `--header` | `-H` | 请求头（可多次） |
| `--headers-file` | — | 请求头文件 |
| `--follow-redirects` | `-F` | 跟随重定向 |
| `--random-agent` | — | 随机 User-Agent |
| `--auth` | — | 认证凭证（user:password） |
| `--auth-type` | — | 认证类型（basic/digest/bearer/ntlm/jwt） |
| `--cert-file` | — | 客户端证书文件 |
| `--key-file` | — | 客户端证书私钥 |
| `--user-agent` | — | User-Agent |
| `--cookie` | — | Cookie |

### 1.6 连接配置（Connection Settings）

| 参数 | 简写 | 说明 |
|------|------|------|
| `--timeout` | — | 连接超时（秒） |
| `--delay` | — | 请求间隔（秒） |
| `--proxy` | `-p` | 代理 URL（可多次） |
| `--proxies-file` | — | 代理列表文件 |
| `--proxy-auth` | — | 代理认证凭证 |
| `--replay-proxy` | — | 发现路径后重放代理 |
| `--tor` | — | 使用 Tor 网络 |
| `--scheme` | — | 协议（http/https） |
| `--max-rate` | — | 每秒最大请求数 |
| `--retries` | — | 失败重试次数 |
| `--ip` | — | 指定服务器 IP |
| `--interface` | — | 指定网络接口 |

### 1.7 高级设置（Advanced Settings）

| 参数 | 简写 | 说明 |
|------|------|------|
| `--crawl` | — | 从响应中爬取新路径 |

### 1.8 视图设置（View Settings）

| 参数 | 简写 | 说明 |
|------|------|------|
| `--full-url` | — | 输出完整 URL |
| `--redirects-history` | — | 显示重定向历史 |
| `--no-color` | — | 关闭颜色 |
| `--quiet-mode` | `-q` | 安静模式 |
| `--disable-cli` | — | 关闭 CLI 输出 |
| `--verbose` | `-v` | 详细输出（响应时间/Content-Type） |

### 1.9 输出设置（Output Settings）

| 参数 | 简写 | 说明 |
|------|------|------|
| `--output-formats` | `-O` | 报告格式（simple/plain/json/xml/md/csv/html/sqlite） |
| `--output-file` | `-o` | 输出文件路径 |
| `--mysql-url` | — | MySQL 输出 |
| `--postgres-url` | — | PostgreSQL 输出 |
| `--log` | — | 日志文件 |

### 1.10 dirsearch 参数统计

| 类别 | 参数数量 | 说明 |
|------|---------|------|
| 目标输入 | 9 | 核心是 `-u`，其他为辅助输入 |
| 字典配置 | 15 | 目录扫描核心能力 |
| 通用设置 | 25 | 扫描行为控制 |
| 高级过滤 | 20 | 8 维匹配器/过滤器 + 逻辑 |
| 请求配置 | 16 | HTTP 请求细节 |
| 连接配置 | 12 | 网络层控制 |
| 高级设置 | 1 | 爬虫 |
| 视图设置 | 6 | 输出展示 |
| 输出设置 | 5 | 报告格式 |
| **合计** | **109** | 实际独立参数约 90+（含简写别名） |

---

## 2. NeoAgent 目录扫描功能取舍分析

### 2.1 决策原则

NeoAgent 目录扫描器不是独立的 dirsearch 工具，而是一个**原子扫描器**，需要遵循：

1. **最小可行参数集**：CLI 单点扫描需要核心参数，Pipeline 编排场景下参数由上游传递
2. **不重复造轮子**：能用 NeoAgent 现有能力的（QoS/Reporter/RunnerManager），不重新实现
3. **渐进式交付**：V1 只做核心能力，高级功能 V2 按需补充
4. **复用 dirsearch 字典**：字典文件直接复用，零成本验证质量，但 NeoAgent 有自己的参数设计哲学

### 2.2 参数取舍总览

```
Legend: ✅ P0 必须    ⚠️ P1 应该有    🔶 P2 可以做    ❌ 不做
```

---

## 3. 功能参数详细设计

### 3.1 目标输入

| NeoAgent 参数 | dirsearch 对应 | 级别 | 理由 |
|---------------|---------------|:---:|------|
| `target`（位置参数） | `--url` | ✅ | 核心目标，CLI 唯一必须 |
| `--targets-file` | `--urls-file` | ⚠️ | 多目标批量扫描需要 |
| `--stdin` | `--stdin` | 🔶 | 管道集成场景需要，优先级低 |

**不做的原因**:
- `--cidr`: NeoAgent 已有 `scan alive -r` 处理 CIDR，目录扫描接收的是已发现的目标
- `--raw`: 原始 HTTP 请求加载不属于目录扫描范畴
- `--nmap-report`: NeoAgent 的数据流是 ServiceScanner 产出，不是独立工具
- `--session`/`--session-id`/`--list-sessions`: NeoAgent 不涉及长时间扫描恢复场景
- `--config`: NeoAgent 通过文件和环境变量管理配置，不需独立配置文件

---

### 3.2 字典配置

| NeoAgent 参数 | dirsearch 对应 | 级别 | 理由 |
|---------------|---------------|:---:|------|
| `--wordlists` | `-w` | ✅ | 指定字典文件路径 |
| `--extensions` | `-e` | ✅ | 扩展列表 |
| `--force-extensions` | `-f` | ⚠️ | 扩展模式切换 |
| `--exclude-extensions` | — | 🔶 | 排除已知静态扩展 |
| `--prefixes` | — | 🔶 | 自定义路径前缀 |
| `--suffixes` | — | 🔶 | 自定义路径后缀 |
| `--uppercase` | `-U` | 🔶 | 词表全部大写 |
| `--lowercase` | `-L` | 🔶 | 词表全部小写 |
| `--capital` | `-C` | 🔶 | 词表首字母大写 |

**不做的原因**:
- `--wordlist-categories`: NeoAgent 内置了 dirsearch 全部字典（`dicc.txt` + `categories/` + `templates/`），无需用户选择
- `--wordlist-backend`: 无 Rust 后端，不需要
- `--wordlist-status`: 调试用，CLI 场景不需要
- `--wordlist-max-size`: NeoAgent 通过 `--max-entries` 控制，与 Dir 扫描无直接关系
- `--overwrite-extensions`: 使用场景有限，`-f` 已覆盖大部分需求

---

### 3.3 扫描控制

| NeoAgent 参数 | dirsearch 对应 | 级别 | 理由 |
|---------------|---------------|:---:|------|
| `--threads` | `-t` | ✅ | 并发线程数（默认 25） |
| `--recursive` | `-r` | ✅ | 递归扫描 |
| `--deep-recursive` | — | ✅ | 深度递归 |
| `--force-recursive` | — | ⚠️ | 强制递归 |
| `--max-recursion-depth` | `-R` | ⚠️ | 最大递归深度（默认 3） |
| `--recursion-status` | — | 🔶 | 递归触发状态码 |
| `--timeout` | — | ✅ | 请求超时（默认 10s） |
| `--max-retries` | `--retries` | ⚠️ | 最大重试次数（默认 2） |
| `--rate-limit` | `--max-rate` | 🔶 | 每秒最大请求数 |
| `--delay` | `--delay` | 🔶 | 请求间隔 |
| `--max-time` | `--max-time` | 🔶 | 总扫描时间上限（秒） |
| `--target-max-time` | `--target-max-time` | 🔶 | 单目标扫描时间上限（秒） |

> **`--max-recursion-depth` 是什么**：控制递归扫描的层数上限，防止字典指数级膨胀。
>
> **为什么需要**：递归扫描的字典膨胀是指数级的：
> ```
> 第 1 层：50K 词（用户字典）
> 第 2 层：20 个命中目录 × 50K = 1,000K
> 第 3 层：400 个命中 × 50K = 20,000K
> 第 4 层：8,000 个命中 × 50K = 400,000K ← 失控！
> ```
>
> **举例**：目标 `example.com`，字典有 `api`，开了递归：
> ```
> 第 1 层：扫描 api → 命中 /api/，追加子路径
> 第 2 层：扫描 api/admin, api/login... → 命中 /api/admin/，追加子路径
> 第 3 层：扫描 api/admin/users... → 命中 /api/admin/users/
> 第 4 层：如果继续 → 400,000K 请求 ❌
> ```
> 限制 `--max-recursion-depth 3` 后，第 3 层扫完停止 ✅
>
> **类比**：`-r` = 打开递归开关，`--deep-recursive` = 选递归模式，`--max-recursion-depth` = 限定扫多深

**不做的原因**：
- `--async`/`--sync`: Go 原生 goroutine 并发，不需要切换模型
- `--filter-threshold`: NeoAgent 通配符检测已内置动态学习，不需要手动调阈值
- `--subdirs`: NeoAgent 通过 `scan run` 全流程编排控制，不在此参数化
- `--exit-on-error`: Pipeline 场景中某个目标失败不应阻塞整个扫描

---

### 3.4 响应过滤

| NeoAgent 参数 | dirsearch 对应 | 级别 | 理由 |
|---------------|---------------|:---:|------|
| `--include-status` | `-i` | ⚠️ | 要显示的状态码白名单 |
| `--exclude-status` | `-x` | ⚠️ | 要排除的状态码黑名单 |
| `--exclude-size` | `--exclude-sizes` | 🔶 | 排除特定大小 |
| `--exclude-text` | — | 🔶 | 排除关键词 |
| `--exclude-regex` | — | 🔶 | 排除正则 |

**不做的原因**:
- 全部 8 维过滤（`--match-*`/`--filter-*`）：V1 不做，DirScanner 只关心"路径是否存在"
- 通配符检测本身就是一个智能过滤器，不需要用户手动调 8 维规则
- `--auto-calibration`: 通配符检测自动校准，不需要用户干预
- `--exclude-redirect`/`--exclude-response`/`--skip-on-status`: 优先级低，V2 评估

**核心过滤设计**：DirScanner 的过滤管道比 dirsearch 简化为 3 层：

```
第一层：快速过滤
  ├── 排除状态码（可配置）
  ├── 排除大小（可配置）
  └── 黑名单路径（内置 400/403/500 特定路径）

第二层：通配符检测（动态学习，自动校准）

第三层：（可选）关键词/正则排除
```

---

### 3.5 请求配置

| NeoAgent 参数 | dirsearch 对应 | 级别 | 理由 |
|---------------|---------------|:---:|------|
| `--method` | `--http-method` | ⚠️ | HTTP 方法（默认 GET） |
| `--header` | `-H` | 🔶 | 自定义请求头 |
| `--follow-redirects` | `-F` | 🔶 | 跟随重定向 |
| `--user-agent` | — | 🔶 | 自定义 User-Agent |
| `--random-agent` | — | 🔶 | 随机 User-Agent |
| `--auth` | — | 🔶 | 认证凭证 |
| `--headers-file` | — | 🔶 | 从文件加载多个请求头（登录态/多头场景） |

**不做的原因**:
- `--data`/`--data-file`: 目录扫描是 GET 为主，POST 场景极少
- `--cert-file`/`--key-file`: mTLS 场景在目录扫描中极少
- `--cookie`: 可通过 `--auth` 或 `--header` 替代
- `--request-backend`: 无 Rust 后端，不需要

---

### 3.6 网络配置

| NeoAgent 参数 | dirsearch 对应 | 级别 | 理由 |
|---------------|---------------|:---:|------|
| `--proxy` | `-p` | ⚠️ | 代理 URL |
| `--ip` | — | 🔶 | 指定服务器 IP |
| `--network-interface` | — | 🔶 | 指定网络接口 |

**不做的原因**:
- `--proxies-file`: 代理列表通过配置管理
- `--proxy-auth`: 代理认证通过配置管理
- `--replay-proxy`: NeoAgent 的结果通过 Reporter 输出，不需要重放
- `--tor`: 不属于安全扫描工具范畴
- `--scheme`: NeoAgent 从 ServiceScanner 已获得 URL scheme

---

### 3.7 输出配置

| NeoAgent 参数 | dirsearch 对应 | 级别 | 理由 |
|---------------|---------------|:---:|------|
| `--oj <path>` | — | ✅ | JSON 文件输出（复用全局 `OutputOptions`，与其他扫描器一致） |
| `--oc <path>` | — | ✅ | CSV 文件输出（复用全局 `OutputOptions`，与其他扫描器一致） |
| `--verbose` | `-v` | ⚠️ | 详细终端输出（响应时间/Content-Type） |
| `--quiet` | `-q` | ⚠️ | 安静模式 |

**不做的原因**:
- `--output-format`/`--output-file`（dirsearch `-O`/`-o`）: NeoAgent 已有全局 `--oj`/`--oc` 统一处理文件输出，引入私有参数只会产生语义重叠和冲突，复用全局参数是唯一正确选择
- `--mysql-url`/`--postgres-url`: NeoAgent 通过 Reporter 接口对接数据库，不直接在 CLI 参数暴露
- `--log`: NeoAgent 统一日志管理
- `--full-url`/`--redirects-history`/`--no-color`/`--disable-cli`: 输出样式调整，优先级低

---

## 4. NeoAgent DirScanner 最终参数清单（V1）

### 4.1 核心参数（P0 - 必须实现）

```bash
neoAgent scan dir <target> [flags]

Flags:
  --extensions string       扩展列表，逗号分隔（如 php,asp,html）
  --wordlists string        字典文件路径（默认使用内置 dirsearch 字典）
  --threads int             并发线程数（默认 25）
  --timeout int             请求超时秒数（默认 10）
  --recursive               启用递归扫描

Global Flags (继承自 scan 父命令):
  --oj string               输出 JSON 文件路径
  --oc string               输出 CSV 文件路径
```

### 4.2 重要参数（P1 - 应该有）

```bash
  --force-extensions        对所有词追加扩展
  --exclude-status string   排除的状态码，逗号分隔（默认 404,500,502,503）
  --deep-recursive          启用深度递归（每级目录）
  --max-recursion-depth int 最大递归深度（默认 3）
  --verbose                 详细输出
  --quiet                   安静模式
  --header string           自定义请求头
  --proxy string            代理 URL
  --follow-redirects        跟随重定向
  --max-retries int         最大重试次数（默认 2）
```

### 4.3 可选参数（P2 - 可以做）

```bash
  --exclude-extensions string   排除的扩展
  --prefixes string             路径前缀
  --suffixes string             路径后缀
  --uppercase                   词表全部大写
  --lowercase                   词表全部小写
  --capital                     词表首字母大写
  --include-status string       要显示的状态码
  --exclude-text string         排除的关键词
  --exclude-regex string        排除的正则
  --rate-limit int              每秒最大请求数
  --max-time int                总扫描时间上限（秒）
  --target-max-time int         单目标扫描时间上限（秒）
  --headers-file string         请求头文件（一行一个，格式: Name: Value）
  --random-agent                随机 User-Agent
  --auth string                 认证凭证
```

---

## 5. 参数组合与默认值

### 5.1 默认行为

```bash
# 最小化命令
neoAgent scan dir example.com

# 等价于
neoAgent scan dir example.com \
  --extensions "" \
  --wordlists "builtin" \
  --threads 25 \
  --timeout 10 \
  --max-retries 2 \
  --exclude-status "404,500,502,503" \
  --recursive \
  --max-recursion-depth 3
```

### 5.2 典型使用场景

```bash
# 场景 1: 快速扫描（Pipeline 场景）
neoAgent scan dir 10.0.0.1 -p 80 --oj result.json

# 场景 2: 深度扫描（带递归 + 指定扩展）
neoAgent scan dir example.com -e php,asp,jsp,html --recursive --force-extensions -v

# 场景 3: 自定义字典
neoAgent scan dir example.com -w /path/to/custom.txt --exclude-status "404"

# 场景 4: 通过代理扫描
neoAgent scan dir example.com --proxy http://127.0.0.1:8080 -t 50
```

---

## 6. 与 dirsearch 的设计差异

| 维度 | dirsearch | NeoAgent DirScanner | 理由 |
|------|-----------|-------------------|------|
| **定位** | 独立工具 | 原子扫描器（嵌入 Pipeline） | NeoAgent 是安全扫描平台 |
| **参数数量** | 90+ | V1: 5 个核心 + 7 个重要 + 13 个可选 + 2 个全局继承 | 最小可用原则 |
| **字典来源** | 文件路径指定 | 内置 dirsearch 全部字典 | 零配置启动 |
| **递归控制** | 3 种模式 | 3 种模式全部保留 | 都有实用价值 |
| **过滤能力** | 8 维 + 通配符 | 通配符 + 基础 3 层过滤 | V1 不做 8 维 |
| **会话恢复** | 完整支持 | 不支持 | Pipeline 编排不依赖 |
| **报告输出** | 8 种格式 | 2 种文件格式（JSON/CSV），复用全局 `--oj`/`--oc` | NeoAgent 全局 OutputOptions 统一 |
| **请求后端** | Python/Rust 双后端 | Go 原生 | Go 并发模型不同 |
| **并发模型** | threading/async/rust | goroutine + AdaptiveLimiter | 复用 NeoAgent QoS |

---

## 7. 待讨论问题

### 7.1 扩展列表默认值

dirsearch 的 `--extensions` 默认是**空**（需要用户指定），否则只扫不带扩展的路径。

NeoAgent 决策（接受建议）：

- **CLI 场景**：提供合理默认值（如 `php,asp,html,js,json,bak,git,env`），用户无感即可使用
- **Pipeline 场景**：通过 `--extensions` 显式传参或由 Master 下发配置，**不依赖 WebScanner/ApiScanner 等上游扫描器的 TechStack 产出**（DirScanner 是独立原子扫描器，现阶段不耦合其他扫描器结果）

> 注：未来 Master 编排场景下，Master 可根据目标域名特征（如 `.cn` 后缀倾向 PHP、`.com` 倾向 Node.js）智能选择扩展，下发到 DirScanner。但这属于编排层决策，不是 DirScanner 自身耦合。

**已决定**：需要默认扩展列表

### 7.2 递归扫描默认值

dirsearch 默认**不递归**（需要 `-r` 启用）。

NeoAgent 决策（接受建议）：

- 默认开启基础递归（`--recursive`），因为 ServiceScanner 识别出的 Web 端口大概率有隐藏路径
- 通过 `--max-recursion-depth` 控制深度上限
- 深度递归和强制递归保持手动启用（`--deep-recursive` / `--force-recursive`），因为这两种模式字典膨胀大，需要用户明确授权

**已决定**：递归默认开启（基础递归 + 深度递归），强制递归需显式启用

### 7.3 通配符检测默认行为

> **通配符检测是什么**：CDN/WAF 对不存在的路径统一返回同一个 404 页面（有 Logo、导航栏等丰富内容），导致大量误报。通配符检测通过采样学习"不存在路径长什么样"，然后排除所有"长得像不存在路径"的响应。
>
> **原理**：随机选 2 个已知不存在的路径 → 提取两个响应的"共同词"（静态模式） → 后续每个命中路径与样本比对 → 相似度超过阈值（长度差异 ≤ 35%、相似度 ≥ 0.9）则判定为通配符响应，跳过。
>
> **dirsearch 默认行为**：自动启动（Scanner + DynamicContentParser），用户通过 `--auto-calibration` 提前触发加速学习。

NeoAgent 决策：

- **默认开启**通配符检测（与 dirsearch 一致），不需要用户额外参数
- **采样阈值**：统一 **5 次**（dirsearch 普通 8 次 / auto_calibration 3 次，取折中）
- **指纹生命周期**：**整个扫描周期**（每个目标一个 Scanner 实例，目标 IP:Port 相同则 CDN 返回的 404 页面一致，无需每路径重新采样）

**已决定**：默认开启，阈值 5，生命周期为整个扫描周期

### 7.4 字典条目数上限

> **为什么需要**：递归扫描会导致字典膨胀。内置 dirsearch 全部字典约 50K-100K 词，开启递归后每个命中目录都会追加新的子路径词，字典条目呈指数级增长。
>
> **膨胀举例**：
> - 50K 词 × 不递归 = 50K 请求 ✅
> - 50K 词 × 基础递归 3 层 × 20 个命中目录 ≈ 200K 请求 ✅
> - 50K 词 × 深度递归 3 层 × 20 个命中目录 × 更深嵌套 ≈ 2M+ 请求 ❌
>
> **dirsearch 对策**：`wordlist_max_size = 500,000`，生成字典时超过则直接停止并报错退出。

NeoAgent 决策：

- **默认 `--max-entries = 500,000`**（与 dirsearch 一致）
- **必须启用递归深度自动限制**：即使用户只开了 `--recursive` 没开 `--max-recursion-depth`，也要默认限制为 3 层深度

**已决定**：默认 500K；递归必须限制深度，即使用户只开 `-r` 也要自动限制 3 层

### 7.5 递归深度自动限制策略（新增）

> **问题**：用户开了 `--recursive`（基础递归），但不知道 `--max-recursion-depth`，会扫到多深？
>
> **风险**：深度递归的字典膨胀是指数级的：
> ```
> 第 1 层：50K 词 → 扫出 20 个命中目录
> 第 2 层：20 个命中 × 50K = 1,000K
> 第 3 层：400 个命中 × 50K = 20,000K  ← 如果继续第 4 层
> 第 4 层：8,000 个命中 × 50K = 400,000K  ← 4 千万次请求，直接失控
> ```
>
> **dirsearch 行为**：`--max-recursion-depth` 不传值时**没有默认值**，理论上可以无限递归下去（靠 `wordlist_max_size` 兜底）。但实际使用中很多人开了递归就忘了设深度限制，跑出去扫了几百万次才靠 max_size 截断。
>
> **安全设计**：递归深度必须有上限，不能依赖用户自觉。

NeoAgent 决策：

- **自动限制策略**：即使用户只传了 `-r`（基础递归），未传 `--max-recursion-depth`，也自动限制为 **3 层**
- **用户显式设置时覆盖**：用户传 `--max-recursion-depth 5` 时按用户设置执行
- **理由**：3 层递归已覆盖 95% 的安全扫描场景（`/admin/` → `/admin/login/` → `/admin/login/redirect`），再深的路径通常是业务逻辑路径，不是安全关注点
- **与 `--wordlist-max-size` 形成双重保险**：3 层是主动控制，500K 是兜底保护

**已决定**：递归深度默认 3 层，用户可覆盖；`--recursive` 和 `--max-recursion-depth` 互为补充，缺一不可失控
