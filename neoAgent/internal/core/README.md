# Core - 核心扫描引擎

## 概述

`core` 是 NeoAgent 的**核心扫描引擎**，包含了所有原子扫描能力、任务调度、结果输出和网络基础设施。它是一个**完全独立、可复用**的模块，不依赖任何上层应用逻辑（如 CLI 或 Server），可以被任何需要扫描能力的场景直接调用。

**设计原则**：
- **原子性**: 每个扫描器都是独立的原子能力，可单独使用
- **可组合**: 通过 Pipeline 可以灵活组合多个扫描器（仅单机模式CLI使用）
- **跨平台**: 核心能力支持 Linux/Windows/macOS
- **高性能**: 原生 Go 实现，零外部依赖

---

## 目录结构

```
core/
├── lib/                    # 底层网络基础设施
│   └── network/
│       ├── dialer/         # 统一网络连接层（代理、超时控制）
│       └── netraw/        # Raw Socket 和数据包构建（Linux Only）
│
├── model/                 # 核心数据模型
│   ├── task.go           # 任务定义
│   ├── result_types.go   # 结果类型
│   └── 说明.md           # 模型说明
│
├── options/              # 任务参数解析与校验
│   ├── types.go         # 基础类型定义
│   ├── output.go        # 输出配置
│   ├── proxy.go         # 代理配置
│   ├── scan_alive.go    # 存活扫描参数
│   ├── scan_port_service.go  # 端口扫描参数
│   ├── scan_os.go       # OS扫描参数
│   ├── scan_web.go      # Web扫描参数
│   ├── scan_vuln.go     # 漏洞扫描参数
│   ├── scan_dir.go      # 目录扫描参数
│   └── scan_subdomain.go  # 子域名扫描参数
│
├── pipeline/            # 核心编排模块
│   ├── pipeline.go      # Pipeline 定义
│   ├── auto_runner.go   # 自动化编排器
│   └── target.go       # 目标生成器
│
├── reporter/            # 结果上报与输出
│   ├── interface.go     # Reporter 接口定义
│   ├── console.go       # 控制台输出
│   └── csv.go          # CSV 文件输出
│
├── runner/              # 任务调度与执行器
│   ├── interface.go     # Scanner 接口定义
│   └── manager.go      # 执行器管理
│
└── scanner/             # 具体扫描能力实现
    ├── interface.go     # Scanner 统一接口
    ├── alive/          # IP 存活扫描
    │   ├── alive.go
    │   ├── icmp.go
    │   ├── tcp_connect.go
    │   ├── arp_linux.go
    │   ├── arp_windows.go
    │   ├── arp_darwin.go
    │   ├── prober.go
    │   └── probe_result.go
    ├── port_service/   # 端口服务扫描
    │   ├── port_service_scanner.go
    │   └── nmap_service/  # Gonmap 引擎
    │       ├── engine.go
    │       ├── parser.go
    │       ├── rules.go
    │       ├── types.go
    │       └── port_lists.go
    └── os/             # OS 识别
        ├── os_scanner.go
        ├── ttl_engine.go
        ├── service_engine.go
        └── nmap_stack/  # Nmap 协议栈指纹
            ├── engine.go
            ├── probes.go
            ├── parser.go
            ├── matcher.go
            └── rules.go
```

---

## 核心模块详解

### 1. 底层网络基础设施 (`lib/network/`)

#### 1.1 Dialer - 统一网络连接层
**职责**：提供全局超时控制、代理支持（SOCKS5）和连接复用

**核心功能**：
- 统一的超时控制（默认 2 秒）
- 透明代理支持（SOCKS5）
- 全局单例管理

**使用场景**：所有 TCP/UDP 连接必须通过此模块发起

**文档**: [dialer/README.md](./lib/network/dialer/README.md)

#### 1.2 NetRaw - Raw Socket 和数据包构建
**职责**：提供跨平台的 Raw Socket 操作和数据包构建能力

**核心功能**：
- Raw Socket 操作（Linux 完整支持，Windows/macOS 降级）
- IP/TCP 数据包构建
- 校验和计算

**使用场景**：OS 协议栈指纹识别、SYN 扫描等高级扫描

**文档**: [netraw/README.md](./lib/network/netraw/README.md)

---

### 2. 核心数据模型 (`model/`)

**职责**：定义核心扫描引擎使用的数据模型

**核心模型**：
- `Task`: 扫描任务定义
- `TaskResult`: 扫描结果
- `TaskType`: 任务类型枚举

**设计原则**：
- 核心层模型不应直接暴露给外部
- 通过通信层进行转换（DTO 模式）

**文档**: [model/说明.md](./model/说明.md)

---

### 3. 任务参数解析 (`options/`)

**职责**：定义和解析各种扫描类型的参数

**支持的扫描类型**：
- 存活扫描（Alive Scan）
- 端口服务扫描（Port Service Scan）
- OS 识别（OS Scan）
- Web 扫描（Web Scan）
- 漏洞扫描（Vuln Scan）
- 目录扫描（Dir Scan）
- 子域名扫描（Subdomain Scan）

**设计特点**：
- 强类型参数定义
- 支持参数校验
- 统一的参数解析接口

---

### 4. 核心编排模块 (`pipeline/`)

**职责**：将独立的扫描能力串联成有逻辑、高效的执行流

**核心组件**：
- `AutoRunner`: 自动化编排器
- `PipelineContext`: 上下文对象，在各个扫描阶段间传递数据
- `TargetGenerator`: 目标生成器，支持 CIDR、IP 列表等

**扫描流程（漏斗式过滤）**：
```
输入目标 → 存活扫描 → 端口扫描 → 服务识别 → OS 识别 → 最终报告
```

**设计决策**：
- 将 Port Scan 和 Service Scan 分离以提高效率
- 采用 IP 级并发模型
- 支持快速失败策略

**文档**: [pipeline/README.md](./pipeline/README.md)

---

### 5. 结果上报与输出 (`reporter/`)

**职责**：将扫描任务结果输出到不同目的地

**当前实现**：
- `ConsoleReporter`: 控制台表格输出
- `CsvReporter`: CSV 文件输出

**未来规划**：
- `JsonReporter`: JSON 文件输出
- `OutputManager`: 统一输出管理器

**文档**: [reporter/README.md](./reporter/README.md)

---

### 6. 任务调度与执行器 (`runner/`)

**职责**：统一的任务执行入口，管理所有扫描器的生命周期

**核心接口**：
- `Scanner`: 扫描器统一接口
- `RunnerManager`: 执行器管理器

**设计特点**：
- CLI 和 Server 模式共用同一个 Runner
- 支持动态并发控制
- 统一的任务调度逻辑

**文档**: [runner/README.md](./runner/README.md)

---

### 7. 具体扫描能力实现 (`scanner/`)

#### 7.1 IP 存活扫描 (`alive/`)

**职责**：探测目标 IP 是否存活

**支持协议**：
- **ARP**: 二层探测，仅限局域网，速度极快
- **ICMP**: 系统调用 ping，可获取 TTL/OS 信息
- **TCP**: TCP Connect 探测，可穿透防火墙

**扫描模式**：
- **Auto Mode**: 自动判断目标类型，选择最优策略
- **Manual Mode**: 用户手动指定协议

**文档**: [alive/README.md](./scanner/alive/README.md)

#### 7.2 端口服务扫描 (`port_service/`)

**职责**：端口发现和服务指纹识别

**核心技术**：
- 基于 **Gonmap** 引擎（移植自 Qscan）
- 完整解析 Nmap 的 `nmap-service-probes` 规则库
- 支持服务版本、操作系统、设备类型识别

**扫描流程**：
1. TCP Connect 端口发现
2. Gonmap 引擎指纹识别
3. 结果聚合

**文档**: [port_service/README.md](./scanner/port_service/README.md)

#### 7.3 OS 识别 (`os/`)

**职责**：多维度操作系统识别

**多引擎并发竞速架构**：
- **TTL Engine**: 基于 ICMP TTL 值推断（快速但精度低）
- **Nmap Stack Engine**: 协议栈指纹识别（Linux Only，精度高）
- **Service Engine**: 服务 Banner 推断（通用，补充识别）

**跨平台兼容性**：
- **Linux**: 完整支持所有引擎
- **Windows/macOS**: 降级使用 TTL Engine 和 Service Engine

**文档**: [os/README.md](./scanner/os/README.md)

---

## 架构设计

### 分层架构

```
┌─────────────────────────────────────────┐
│         Application Layer              │
│  (CLI / Server / Custom Integration) │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│         Runner Layer                  │
│  (Task Scheduling & Execution)       │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│         Scanner Layer                 │
│  (Atomic Scan Capabilities)          │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│         Network Layer                 │
│  (Dialer / Raw Socket)             │
└───────────────────────────────────────┘
```

### 数据流转

```
Options → Task → Runner → Scanner → Result → Reporter → Output
   ↓        ↓       ↓        ↓         ↓         ↓          ↓
 参数    任务    调度    执行     结果     上报      输出
```

---

## 使用示例

### 示例 1: 单独使用存活扫描器

```go
import (
    "neoagent/internal/core/scanner/alive"
    "neoagent/internal/core/model"
    "neoagent/internal/core/options"
)

// 创建扫描器
scanner := alive.NewIpAliveScanner()

// 构建任务
task := &model.Task{
    Type:   model.TaskTypeIpAliveScan,
    Target: "192.168.1.0/24",
    Params: options.NewAliveScanOptions(
        options.WithConcurrency(1000),
        options.WithICMP(true),
    ),
}

// 执行扫描
results, err := scanner.Run(context.Background(), task)
```

### 示例 2: 使用 Pipeline 自动化编排

```go
import (
    "neoagent/internal/core/pipeline"
    "neoagent/internal/core/options"
)

// 创建 Pipeline
runner := pipeline.NewAutoRunner()

// 配置选项
opts := &options.ScanRunOptions{
    Target:      "192.168.1.0/24",
    Concurrency:  100,
    PortRange:    "1-1000",
    ServiceScan:  true,
    OSScan:       true,
}

// 执行自动化扫描
results, err := runner.Run(context.Background(), opts)
```

---

## 性能特性

### 并发控制
- **IP 级并发**: 每个 IP 独立处理，互不阻塞
- **信号量机制**: 精确控制并发度
- **动态调整**: 支持运行时调整并发数

### 优化策略
- **漏斗式过滤**: 尽早剔除无效目标
- **快速失败**: 超时立即跳过
- **连接复用**: Dialer 层统一管理连接

### 跨平台性能
- **Linux**: 完整性能（Raw Socket 支持）
- **Windows**: 应用层扫描（Connect Scan）
- **macOS**: 降级扫描（ICMP + TCP）

---

## 依赖关系

### 外部依赖
- **无外部依赖**: 纯 Go 实现，零 CGO
- **标准库**: 仅使用 Go 标准库

### 内部依赖
```
scanner → lib/network
scanner → model
runner  → scanner
pipeline → runner
reporter → model
options → (独立)
```

---

## 扩展指南

### 添加新的扫描器

1. 在 `scanner/` 下创建新目录
2. 实现 `runner.Scanner` 接口
3. 在 `options/` 下添加参数定义
4. 在 `runner/manager.go` 中注册

### 添加新的输出格式

1. 在 `reporter/` 下创建新文件
2. 实现 `reporter.Reporter` 接口
3. 在 `OutputManager` 中注册

---

## 最佳实践

1. **统一使用 Dialer**: 所有网络连接必须通过 `dialer` 模块
2. **支持 Context**: 所有扫描操作必须支持 `context.Context`
3. **错误处理**: 统一使用 `logger` 包记录日志
4. **并发安全**: 所有共享状态必须使用 `sync` 包保护
5. **资源清理**: 使用 `defer` 确保资源释放

---

## 版本历史

- **v1.0.0**: 初始版本，实现核心扫描能力
- **v1.1.0**: 添加 Pipeline 编排
- **v1.2.0**: 集成 Gonmap 引擎
- **v1.3.0**: 添加 OS 多引擎识别

---

## 贡献指南

1. 遵循现有代码风格
2. 添加单元测试
3. 更新相关文档
4. 确保跨平台兼容性

---

## 许可证

MIT License
