# NeoAgent 目录扫描模块功能参数设计

**日期**: 2026-08-14（最近同步代码: 2026-08-22）
**状态**: 已实现（V1），本文档第 3~7 节已对照 `cmd/agent/scan/dir.go` 与 `internal/core/options/scan_dir.go` 校订
**参考**: dirsearch v0.4.3 (https://github.com/maurosoria/dirsearch)

> **实现状态图例**：✅ 已实现并暴露为 CLI flag　🔸 底层已支持但未暴露 CLI flag　❌ 未实现

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

| NeoAgent 参数 | dirsearch 对应 | 级别 | 实现 | 理由 |
|---------------|---------------|:---:|:---:|------|
| `target`（位置参数 / `-t/--target`） | `--url` | ✅ | ✅ | 核心目标，CLI 唯一必须；位置参数优先于 `-t` |
| `--targets-file`/`-l` | `--urls-file` | ⚠️ | ✅ | 多目标批量扫描，逐行读取后依次扫描并汇总结果 |
| `--stdin` | `--stdin` | 🔶 | ❌ | 管道集成场景需要，优先级低，未实现 |

**不做的原因**:
- `--cidr`: NeoAgent 已有 `scan alive -r` 处理 CIDR，目录扫描接收的是已发现的目标
- `--raw`: 原始 HTTP 请求加载不属于目录扫描范畴
- `--nmap-report`: NeoAgent 的数据流是 ServiceScanner 产出，不是独立工具
- `--session`/`--session-id`/`--list-sessions`: NeoAgent 不涉及长时间扫描恢复场景
- `--config`: NeoAgent 通过文件和环境变量管理配置，不需独立配置文件

---

### 3.2 字典配置

| NeoAgent 参数 | dirsearch 对应 | 级别 | 实现 | 理由 |
|---------------|---------------|:---:|:---:|------|
| `--wordlists`/`-w` | `-w` | ✅ | ✅ | 指定字典文件路径，支持逗号分隔多路径 |
| `--category` | — | 🔶 | ✅ | 追加内置技术栈分类字典（逗号分隔，如 `wordpress,php/laravel`），2026-08-22 新增，与 `--wordlists` 并列的字典来源，需用户手动指定，不做自动 TechStack 联动（见设计文档第 10 节） |
| `--extensions`/`-e` | `-e` | ✅ | ✅ | 扩展列表，默认值 `php,asp,html,js,json,bak,git,env` |
| `--force-extensions`/`-f` | `-f` | ⚠️ | ✅ | 对不含 `%EXT%` 的条目也追加扩展名变体 |
| `--exclude-extensions` | — | 🔶 | ❌ | 排除已知静态扩展，未实现 |
| `--prefixes` | — | 🔶 | ✅ | 自定义路径前缀 |
| `--suffixes` | — | 🔶 | ✅ | 自定义路径后缀 |
| `--uppercase`/`-U` | `-U` | 🔶 | ✅ | 词表全部大写 |
| `--lowercase`/`-L` | `-L` | 🔶 | ✅ | 词表全部小写 |
| `--capital`/`-C` | `-C` | 🔶 | ✅ | 词表首字母大写 |

> 注：字典条目仅支持 `%EXT%` 模板 token 展开。`%SUBJECT%`/`%ADMIN_OP%`/`%CRUD_OP%` 等 dirsearch
> 模板系统的其余 9+ 种 token 均未实现（这些 token 正是 `templates/*.txt` 无法接入的根因，见下方
> `--wordlist-categories` 条目说明）。

**不做的原因**:
- `--wordlist-categories`: NeoAgent 内置了 dirsearch 全部字典（`dicc.txt` + `categories/` + `templates/`）；`categories/` 已通过 `--category` 接入（2026-08-22）。`templates/` 仍未接入——2026-08-22 补充审查：并非"与 categories 功能重叠"，而是 `templates/*.txt` 内容依赖 `%ADMIN_OP%`/`%SUBJECT%`/`%CRUD_OP%` 等 9+ 种令牌的笛卡尔积组合展开（对照 dirsearch `wordlist_template.py` 的 `DEFAULT_PLACEHOLDERS`），NeoAgent 的 `dict/template.go` 只实现了 `%EXT%` 一种令牌，接入前置条件（多令牌展开系统）本身不存在，详见设计文档 3.4.1 节
- `--wordlist-backend`: 无 Rust 后端，不需要
- `--wordlist-status`: 调试用，CLI 场景不需要
- `--wordlist-max-size`: NeoAgent 未实现独立的字典条目数硬上限校验（见 7.4 节）
- `--overwrite-extensions`: 使用场景有限，`-f` 已覆盖大部分需求

---

### 3.3 扫描控制

| NeoAgent 参数 | dirsearch 对应 | 级别 | 实现 | 理由 |
|---------------|---------------|:---:|:---:|------|
| `--threads` | `-t` | ✅ | ✅ | 并发线程数（默认 25），同时决定连接池大小（`MaxConns = threads`） |
| `--recursive`/`-r` | `-r` | ✅ | ✅ | 递归扫描，默认开启（见 7.2 节） |
| `--deep-recursive` | — | ✅ | ✅ | 深度递归（命中路径逐级展开父目录） |
| `--force-recursive` | — | ✅ | ✅ | 强制递归（所有命中路径），CLI flag 已于 2026-08-22 补注册，之前仅底层支持、CLI 无法开启 |
| `--max-recursion-depth`/`-R` | `-R` | ⚠️ | ✅ | 最大递归深度（默认 3，范围 0~10，`Validate()` 校验） |
| `--recursion-status` | — | 🔶 | ❌ | 递归触发状态码可配置，未实现；当前固定为 `200/301/302/403`（`isDirectoryStatus()`） |
| `--timeout` | — | ✅ | ✅ | 请求超时（默认 10s，范围 1~300，`Validate()` 校验） |
| `--max-retries` | `--retries` | ⚠️ | ✅ | 最大重试次数（默认 2） |
| `--rate-limit` | `--max-rate` | 🔶 | ✅ | 每秒最大请求数（0=不限） |
| `--delay` | `--delay` | 🔶 | ✅ | 请求间隔（毫秒） |
| `--max-time` | `--max-time` | 🔶 | ✅ | 总扫描时间上限（秒，0=不限） |
| `--target-max-time` | `--target-max-time` | 🔶 | ✅ | 单目标扫描时间上限（秒，0=不限） |

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

| NeoAgent 参数 | dirsearch 对应 | 级别 | 实现 | 理由 |
|---------------|---------------|:---:|:---:|------|
| `--include-status`/`-i` | `-i` | ⚠️ | ✅ | 仅包含的状态码白名单，优先于 `--exclude-status` |
| `--exclude-status`/`-x` | `-x` | ⚠️ | ✅ | 排除的状态码黑名单（默认 `404,500,502,503`） |
| `--exclude-size` | `--exclude-sizes` | 🔶 | ✅ | 排除特定响应体大小（字节，逗号分隔） |
| `--exclude-text` | — | 🔶 | ✅ | 排除响应体关键词，可多次指定 |
| `--exclude-regex` | — | 🔶 | ✅ | 排除响应体正则（仅支持单条表达式） |

**不做的原因**:
- 全部 8 维过滤（`--match-*`/`--filter-*`）：V1 不做，DirScanner 只关心"路径是否存在"
- 通配符检测本身就是一个智能过滤器，不需要用户手动调 8 维规则
- `--auto-calibration`: 通配符检测自动校准，不需要用户干预
- `--exclude-redirect`/`--exclude-response`/`--skip-on-status`: 优先级低，V2 评估

**实际过滤管道**（`engine/filter.go` + `engine/scanner.go`，与最初设计的 3 层基本一致）：

```
第一层：Filter 快速过滤（engine.Filter.Match）
  ├── IncludeStatus 白名单（命中即通过，优先于下方黑名单判断）
  ├── ExcludeStatus 黑名单（默认 404,500,502,503）
  ├── ExcludeSize / ExcludeKeywords / ExcludeRegex（均可选，未配置则跳过）
  └── 注：内置"黑名单路径"结论为不接入（2026-08-22 审查定论）——dict.LoadBlacklist() 加载的
      实际是畸形请求探测 payload，与 ErrorPaths[status][path] 按真实路径查表的方式语义不匹配，
      接入只会做出一个永远不生效的假功能，详见设计文档 5.3 节

第二层：WildcardScanner 通配符检测（两阶段 sync.Once 采样，见第 5 节）

第三层：以上均通过才计为一次 DirHit
```

---

### 3.5 请求配置

| NeoAgent 参数 | dirsearch 对应 | 级别 | 实现 | 理由 |
|---------------|---------------|:---:|:---:|------|
| `--method`/`-m` | `--http-method` | ⚠️ | ✅ | HTTP 方法（默认 GET） |
| `--header`/`-H` | `-H` | 🔶 | ✅ | 自定义请求头，可多次指定 |
| `--follow-redirects`/`-F` | `-F` | 🔶 | ✅ | 跟随重定向 |
| `--user-agent` | — | 🔶 | ✅ | 自定义 User-Agent（优先级低于 `--random-agent`） |
| `--random-agent` | — | 🔶 | ✅ | 从内置 User-Agent 池随机轮换（非每请求随机，见 6.） |
| `--auth` | — | 🔶 | ✅ | 仅支持 Basic Auth（`user:pass`，编码为 Authorization 头） |
| `--headers-file` | — | 🔶 | ✅ | 从文件加载多个请求头，CLI 层与 `-H` 合并后一起下发 |

**不做的原因**:
- `--data`/`--data-file`: 目录扫描是 GET 为主，POST 场景极少
- `--cert-file`/`--key-file`: mTLS 场景在目录扫描中极少
- `--cookie`: 可通过 `--auth` 或 `--header` 替代
- `--request-backend`: 无 Rust 后端，不需要

> 注：`--auth-type`（digest/bearer/ntlm/jwt 等）未实现，仅支持 Basic Auth。

---

### 3.6 网络配置

| NeoAgent 参数 | dirsearch 对应 | 级别 | 实现 | 理由 |
|---------------|---------------|:---:|:---:|------|
| `--proxy`/`-p` | `-p` | ⚠️ | ✅ | 代理 URL（单个字符串，不支持多代理轮换） |
| `--ip` | — | 🔶 | ✅ | 绑定本地出口 IP |
| `--network-interface` | — | 🔶 | ✅ | 绑定网络接口 |

**不做的原因**:
- `--proxies-file`: 代理列表通过配置管理
- `--proxy-auth`: 代理认证通过配置管理
- `--replay-proxy`: NeoAgent 的结果通过 Reporter 输出，不需要重放
- `--tor`: 不属于安全扫描工具范畴
- `--scheme`: NeoAgent 从 ServiceScanner 已获得 URL scheme

---

### 3.7 输出配置

| NeoAgent 参数 | dirsearch 对应 | 级别 | 实现 | 理由 |
|---------------|---------------|:---:|:---:|------|
| `--oj <path>` | — | ✅ | ✅ | JSON 文件输出（复用全局 `OutputOptions`，与其他扫描器一致） |
| `--oc <path>` | — | ✅ | ✅ | CSV 文件输出（复用全局 `OutputOptions`，与其他扫描器一致） |
| `--verbose`/`-v` | `-v` | ⚠️ | ✅ | 详细终端输出 |
| `--quiet`/`-q` | `-q` | ⚠️ | ✅ | 安静模式 |

**不做的原因**:
- `--output-format`/`--output-file`（dirsearch `-O`/`-o`）: NeoAgent 已有全局 `--oj`/`--oc` 统一处理文件输出，引入私有参数只会产生语义重叠和冲突，复用全局参数是唯一正确选择
- `--mysql-url`/`--postgres-url`: NeoAgent 通过 Reporter 接口对接数据库，不直接在 CLI 参数暴露
- `--log`: NeoAgent 统一日志管理
- `--full-url`/`--redirects-history`/`--no-color`/`--disable-cli`: 输出样式调整，优先级低

> 注：`--verbose`/`--quiet` 仅控制 CLI 本地打印详略程度，不会下发到 `task.Params`（Runner 不关心 CLI 端的展示偏好）。

---

## 4. NeoAgent DirScanner 最终参数清单（V1 已实现，与 `cmd/agent/scan/dir.go` 实际 flag 一致）

### 4.1 核心参数（P0）

```bash
neoAgent scan dir <target> [flags]

Flags:
  --extensions, -e string       扩展列表，逗号分隔（默认 php,asp,html,js,json,bak,git,env）
  --wordlists, -w string        字典文件路径（默认使用内置 dirsearch 字典）
  --threads int                 并发线程数（默认 25，1~500）
  --timeout int                 请求超时秒数（默认 10，1~300）
  --recursive, -r                递归扫描（默认已开启，见 7.2）
  --target, -t string            目标 URL（也可用位置参数，位置参数优先）

Global Flags (继承自 scan 父命令):
  --oj string               输出 JSON 文件路径
  --oc string               输出 CSV 文件路径
```

### 4.2 重要参数（P1）

```bash
  --force-extensions, -f          对不含 %EXT% 的词也追加扩展变体
  --exclude-status, -x string     排除的状态码，逗号分隔（默认 404,500,502,503）
  --deep-recursive                深度递归（命中目录时逐级展开父目录并一并扫描）
  --force-recursive                强制递归（对每个命中路径都继续扫描，不限于目录）
  --max-recursion-depth, -R int   最大递归深度（默认 3，范围 0~10）
  --verbose, -v                    输出详细扫描信息
  --quiet, -q                      静默模式，仅输出最终结果
  --header, -H stringArray         自定义请求头 "Key: Value"，可多次指定
  --proxy, -p string               代理地址（如 http://127.0.0.1:8080）
  --follow-redirects, -F           跟随重定向
  --max-retries int                请求失败最大重试次数（默认 2）
  --method, -m string               HTTP 请求方法（默认 GET）
```

> ✅ `--force-recursive`：2026-08-22 已补注 CLI flag，与底层 `DirOptions.ForceRecursive` 打通，现可直接从命令行开启，不再依赖 Pipeline 下发 task 参数。

### 4.3 可选参数（P2）

```bash
  --prefixes string             追加前缀（逗号分隔）
  --suffixes string             追加后缀（逗号分隔）
  --uppercase, -U                字典条目转大写
  --lowercase, -L                字典条目转小写
  --capital, -C                   字典条目首字母大写
  --include-status, -i string    仅包含的状态码（逗号分隔，优先于 --exclude-status）
  --exclude-text stringArray     响应体中排除的关键字，可多次指定
  --exclude-regex string         响应体排除正则
  --exclude-size string          排除的响应体大小（字节，逗号分隔）
  --rate-limit int               每秒最大请求数（0=不限）
  --delay int                    每次请求间隔（毫秒）
  --max-time int                 整个扫描的最长运行时间（秒，0=不限）
  --target-max-time int          单个目标的最长扫描时间（秒，0=不限）
  --headers-file string          从文件加载请求头，每行一条 "Key: Value"
  --random-agent                 使用内置 User-Agent 池随机轮换
  --auth string                  Basic Auth 凭据，格式 user:pass
  --user-agent string            自定义 User-Agent（优先级低于 --random-agent）
  --ip string                    绑定本地出口 IP
  --network-interface string     绑定网络接口
  --targets-file, -l string      批量目标文件，每行一个 URL
```

---

## 5. 参数组合与默认值

### 5.1 默认行为

```bash
# 最小化命令
neoAgent scan dir example.com

# 等价于（与 options.NewDirScanOptions() 默认值一致）
neoAgent scan dir example.com \
  --extensions "php,asp,html,js,json,bak,git,env" \
  --wordlists "" \
  --threads 25 \
  --timeout 10 \
  --max-retries 2 \
  --exclude-status "404,500,502,503" \
  --recursive \
  --max-recursion-depth 3 \
  --method GET
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

| 维度 | dirsearch | NeoAgent DirScanner（实际） | 理由 |
|------|-----------|-------------------|------|
| **定位** | 独立工具 | 原子扫描器（嵌入 Pipeline） | NeoAgent 是安全扫描平台 |
| **参数数量** | 90+ | 6 个核心 + 8 个重要 + 12 个可选 + 2 个全局继承（实际 flag 数） | 最小可用原则 |
| **字典来源** | 文件路径指定 | 内置 dirsearch 全部字典（`go:embed`） | 零配置启动 |
| **递归控制** | 3 种模式 | 3 种模式均实现且均已暴露 CLI flag（2026-08-22 补上 `--force-recursive`） | 基础/深度/强制递归均可直接从 CLI 使用 |
| **过滤能力** | 8 维 + 通配符 | 通配符 + 基础 3 层过滤 | V1 不做 8 维 |
| **字典条目上限** | `wordlist_max_size=500K`，超限报错退出 | 未实现对应硬上限校验 | 见 7.4 节讨论 |
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
- 深度递归保持手动启用（`--deep-recursive`），因为该模式字典膨胀大，需要用户明确授权

**已决定**：递归默认开启（基础递归），深度递归/强制递归需显式启用

> ✅ **已修复**：之前 `cmd/agent/scan/dir.go` 漏注册了 `--force-recursive` flag，导致底层 `DirOptions.ForceRecursive`/`shouldRecursion()` 虽完整但 CLI 无法开启，2026-08-22 已补上该 flag，现可直接从命令行使用，不再需要通过 Pipeline/Master 下发 task 参数。

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

> ⚠️ **与实现的差异**：`--max-entries`/字典总条目数硬上限校验在当前代码中**未实现**——`options.DirScanOptions.Validate()` 只校验了 `MaxRecursionDepth`（0~10）、`Threads`（1~500）、`Timeout`（1~300），没有对字典总条目数设上限。当前完全依赖递归深度限制（3 层默认、最大 10 层）作为唯一防膨胀手段，未形成本节所说的"双重保险"。

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

**实现验证**：与代码完全一致——`options.NewDirScanOptions()` 默认 `MaxRecursionDepth: 3`，`Validate()` 强制 `0~10` 范围校验（用户传参超出范围直接报错拒绝执行，而非静默截断），"自动限制 3 层"体现为默认值而非运行时动态兜底。字典总条目数 500K 的"兜底保护"未实现（见 7.4 节），当前仅有递归深度这一层控制手段。
