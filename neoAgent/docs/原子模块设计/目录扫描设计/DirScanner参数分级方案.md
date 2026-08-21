# DirScanner 参数分级方案

**日期**: 2025-01-20
**状态**: 已完成
**关联文档**: [NeoAgent目录扫描模块功能参数设计.md](./NeoAgent目录扫描模块功能参数设计.md)、[目录扫描模块设计文档.md](./目录扫描模块设计文档.md)、[DirScanner开发实施文档.md](./DirScanner开发实施文档.md)

---

## 1. 背景与动机

### 1.1 问题

DirScanner 当前注册了 **40+ 个 CLI flag**，`--help` 输出一屏放不下，用户找不到核心参数。

但参数多本身不是错——dirscan 确实需要这些能力。问题是**所有参数不分主次地堆在一起**。

### 1.2 方案

不砍参数，改为**分级展示**：

- `neoagent scan dir --help`：只展示 **P0 常用参数**（~15 个）
- `neoagent scan dir --help-all`：展示 **全部参数**（含 P1/P2 高级参数）

底层能力全部保留，CLI flag 全部保留，`task.Params` key 全部保留。只是 `--help` 默认隐藏高级参数。

### 1.3 原则

1. **不删任何参数**：所有 CLI flag、结构体字段、task.Params key 保持不变
2. **不破坏任何现有功能**：`--help` 隐藏的参数仍然可用，只是不显示在帮助里
3. **cobra 原生支持**：用 `Flag.Hidden = true` 实现，不需要自定义帮助模板

---

## 2. 参数分级

### 2.1 P0 常用参数（`--help` 展示，15 个）

用户 90% 场景下需要的参数：

| CLI flag | task.Params key | 说明 |
|----------|----------------|------|
| `target` / 位置参数 | — | 目标 URL |
| `-l` / `--targets-file` | — | 批量目标文件 |
| `-w` / `--wordlists` | `wordlists` | 自定义字典 |
| `-e` / `--extensions` | `extensions` | 扩展名列表 |
| `--category` | `category` | 技术栈分类字典 |
| `-f` / `--force-extensions` | `force_extensions` | 强制追加扩展名 |
| `--threads` | `threads` | 并发线程数 |
| `--timeout` | `timeout` | 请求超时 |
| `-r` / `--recursive` | `recursive` | 递归扫描 |
| `-R` / `--max-recursion-depth` | `max_recursion_depth` | 最大递归深度 |
| `-x` / `--exclude-status` | `exclude_status` | 排除状态码 |
| `-i` / `--include-status` | `include_status` | 仅包含状态码 |
| `-H` / `--header` | `headers` | 自定义请求头 |
| `-m` / `--method` | `method` | HTTP 方法 |
| `-v` / `--verbose` | — | 详细输出 |

### 2.2 P1 高级参数（`--help-all` 展示，`--help` 隐藏，18 个）

高级用户和编排层需要的参数：

| CLI flag | task.Params key | 说明 | 隐藏理由 |
|----------|----------------|------|---------|
| `-U` / `--uppercase` | `uppercase` | 字典条目转大写 | 字典变换是高级场景 |
| `-L` / `--lowercase` | `lowercase` | 字典条目转小写 | 同上 |
| `-C` / `--capital` | `capital` | 字典条目首字母大写 | 同上 |
| `--prefixes` | `prefixes` | 追加前缀 | 同上 |
| `--suffixes` | `suffixes` | 追加后缀 | 同上 |
| `--deep-recursive` | `deep_recursive` | 深度递归 | 会导致请求量爆炸 |
| `--force-recursive` | `force_recursive` | 强制递归 | 同上 |
| `--rate-limit` | `rate_limit` | 每秒最大请求数 | QoS 已自适应 |
| `--delay` | `delay` | 请求间隔 | 和 rate-limit 重叠 |
| `--max-time` | `max_time` | 总扫描时间上限 | 高级控制 |
| `--target-max-time` | `target_max_time` | 单目标时间上限 | 和 max-time 重叠 |
| `--max-retries` | `max_retries` | 最大重试次数 | 默认值已够用 |
| `--exclude-text` | `exclude_text` | 响应体关键词排除 | 通配符检测已覆盖大部分 |
| `--exclude-regex` | `exclude_regex` | 响应体正则排除 | 同上 |
| `--exclude-size` | `exclude_size` | 响应体大小排除 | 同上 |
| `--random-agent` | `random_agent` | 随机 UA 轮换 | 高级反封禁 |
| `-F` / `--follow-redirects` | `follow_redirects` | 跟随重定向 | 默认不跟随已够用 |
| `-p` / `--proxy` | `proxy` | 代理地址 | 将由 proxy 通用服务统一实现 |

### 2.3 P2 边缘参数（`--help-all` 展示，`--help` 隐藏，5 个）

极少使用，但底层保留：

| CLI flag | task.Params key | 说明 | 隐藏理由 |
|----------|----------------|------|---------|
| `--user-agent` | `user_agent` | 自定义 UA | 用 `--header` 或 `--random-agent` 替代 |
| `--auth` | `auth` | Basic Auth | 用 `--header` 替代 |
| `--headers-file` | — | 从文件加载请求头 | 用 shell 脚本 + `-H` 替代 |
| `--ip` | `ip` | 绑定本地出口 IP | 网络层配置，非扫描器职责 |
| `--network-interface` | `network_interface` | 绑定网络接口 | 同上 |

---

## 3. 实现方案

### 3.1 cobra Hidden flag + 自定义 --help-all

cobra 原生支持 `Flag.Hidden = true`，隐藏的 flag：
- `--help` 不显示
- 仍然可以正常使用（`--proxy http://...` 照常工作）

`--help-all` **不是 cobra/pflag 内置能力**，需自定义实现：
1. 注册 `--help-all` bool flag（自身也 hidden）
2. 在 `RunE` 最前面拦截，如果 `helpAll == true`，临时重置所有 flag 的 `Hidden = false`，调用 `FlagUsages()` 输出全部 flag，然后 return
3. 输出后恢复 hidden 状态

### 3.2 代码变更

只改一个文件：`cmd/agent/scan/dir.go`：
1. 对 P1/P2 共 23 个 flag 追加 `flags.MarkHidden("xxx")`
2. 注册 `--help-all` bool flag（自身也 hidden）
3. 在 `RunE` 最前面拦截 `helpAll`，调用 `printAllFlags()` 输出全部 flag
4. 新增 `printAllFlags()` 函数：临时重置 `Hidden = false`，调用 `FlagUsages()` 输出

> **不需要改的文件**：
> - `internal/core/options/scan_dir.go` — 结构体字段、`Validate()`、`ToTask()` 全部不变
> - `internal/core/scanner/dir/dir_scanner.go` — `DirOptions`、`parseDirOptions()` 全部不变
> - `internal/core/scanner/dir/dict/dictionary.go` — 不变
> - `internal/core/scanner/dir/engine/` — 全部不变

### 3.3 变更量

| 文件 | 改动量 |
|------|--------|
| `cmd/agent/scan/dir.go` | 追加 23 行 `MarkHidden` + 注册 `--help-all` flag + `printAllFlags()` 函数 |
| 其他文件 | **零改动** |

---

## 4. 规则文件清理

参数分级不影响规则文件清理。以下文件仍然需要删除（原因与参数分级无关，是功能本身从未接入主流程）：

### 4.1 删除的规则文件

| 文件路径 | 删除理由 |
|---------|---------|
| `wordlists/templates/` 全部 7 个文件 | 依赖多令牌笛卡尔积展开系统，Go 侧从未实现，`LoadTemplateWordlist()` 从未接入主流程（Task 5.5 审查定论） |
| `wordlists/blacklist/` 全部 3 个文件 | 数据是畸形请求探测 payload，与 `ErrorPaths[status][path]` 查表方式语义不匹配（Task 5.4 审查定论） |

### 4.2 删除后需同步清理的代码

| 文件 | 清理内容 |
|------|---------|
| `internal/core/scanner/dir/dict/wordlists.go` | 删除 `LoadTemplateWordlist()` 函数、`LoadBlacklist()` 函数、`ErrBlacklistNotFound` |
| `internal/core/scanner/dir/dict/wordlists_test.go` | 删除对应测试函数 |
| `internal/core/scanner/dir/engine/filter.go` | 删除 `FilterConfig.ErrorPaths` 字段、`AddErrorPath()` 方法 |

### 4.3 保留的规则文件

| 文件/目录 | 保留理由 |
|----------|---------|
| `wordlists/dicc.txt` | 主字典 |
| `wordlists/categories/` 全部子目录 | 技术栈分类字典 |
| `wordlists/user-agents.txt` | UA 列表，`--random-agent` 活跃使用 |

---

## 5. 预期效果

### 5.1 `neoagent scan dir --help`

```
Flags:
  -e, --extensions string       文件扩展名列表（默认 php,asp,html,js,json,bak,git,env）
  -w, --wordlists string        自定义字典文件路径
      --category string         追加技术栈分类字典（逗号分隔）
  -f, --force-extensions        对不含 %EXT% 的条目也追加扩展变体
      --threads int             并发线程数（默认 25）
      --timeout int             请求超时秒数（默认 10）
  -r, --recursive               递归扫描（默认开启）
  -R, --max-recursion-depth int 最大递归深度（默认 3）
  -x, --exclude-status string   排除的状态码（默认 404,500,502,503）
  -i, --include-status string   仅包含的状态码
  -H, --header stringArray      自定义请求头 "Key: Value"
  -m, --method string           HTTP 请求方法（默认 GET）
  -l, --targets-file string     批量目标文件
  -v, --verbose                 详细输出

Global Flags:
      --oj string               输出 JSON 文件路径
      --oc string               输出 CSV 文件路径
  -q, --quiet                   静默模式

Use "neoagent scan dir --help-all" to see all available flags.
```

### 5.2 `neoagent scan dir --help-all`

展示全部 40+ flag，包括 hidden 的 P1/P2 参数。

---

## 6. 与瘦身方案的对比

| 维度 | 瘦身方案（已废弃） | 分级方案（当前） |
|------|-------------------|----------------|
| 砍除参数 | 13 个 | 0 个 |
| `--help` 展示参数 | 27 个 | 15 个 |
| `--help-all` 展示参数 | — | 38 个 |
| 代码改动文件数 | 4 个 | 1 个 |
| 代码改动行数 | ~100 行（删+改） | ~50 行（追加 MarkHidden + printAllFlags） |
| 破坏现有功能风险 | 中（要改结构体） | 零（只改展示） |
| task.Params 一致性 | 需同步删 key | 无需改动 |
| 底层字段 | 需决定保留/删除 | 全部保留 |
