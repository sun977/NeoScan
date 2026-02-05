# Core - 核心扫描引擎

## 概述

`core` 是 NeoAgent 的**核心扫描引擎**，包含了所有原子扫描能力、任务调度、结果输出和网络基础设施。它是一个**完全独立、可复用**的模块，不依赖任何上层应用逻辑（如 CLI 或 Server），可以被任何需要扫描能力的场景直接调用。

**设计原则**：
- **原子性**: 每个扫描器都是独立的原子能力，可单独使用
- **可组合**: 通过 Pipeline 可以灵活组合多个扫描器（仅单机模式CLI使用）
- **统一构建**: 通过 Factory 模式统一管理能力的实例化与配置
- **跨平台**: 核心能力支持 Linux/Windows/macOS
- **高性能**: 原生 Go 实现，零外部依赖

---

## 目录结构

```
core/
├── factory/              # [New] 全局能力工厂 (统一实例化入口)
│   ├── alive_factory.go
│   ├── port_factory.go
│   ├── os_factory.go
│   └── brute_factory.go
│
├── lib/                  # 底层网络基础设施
│   └── network/
│       ├── dialer/       # 统一网络连接层（代理、超时控制）
│       └── netraw/       # Raw Socket 和数据包构建（Linux Only）
│
├── model/                # 核心数据模型
│   ├── task.go           # 任务定义
│   ├── result_types.go   # 结果类型
│   └── 说明.md           # 模型说明
│
├── options/              # 任务参数解析与校验
│   ├── types.go          # 基础类型定义
│   ├── output.go         # 输出配置
│   ├── proxy.go          # 代理配置
│   └── scan_*.go         # 各模块参数定义
│
├── pipeline/             # 核心编排模块
│   ├── pipeline.go       # Pipeline 定义
│   ├── auto_runner.go    # 自动化编排器
│   └── target.go         # 目标生成器
│
├── reporter/             # 结果上报与输出
│   ├── interface.go      # Reporter 接口定义
│   ├── console.go        # 控制台输出
│   └── csv.go            # CSV 文件输出
│
├── runner/               # 任务调度与执行器
│   ├── interface.go      # Scanner 接口定义
│   └── manager.go        # 执行器管理
│
└── scanner/              # 具体扫描能力实现
    ├── interface.go      # Scanner 统一接口
    ├── alive/            # IP 存活扫描
    ├── port_service/     # 端口服务扫描 (Gonmap)
    ├── os/               # OS 识别 (多引擎)
    └── brute/            # 弱口令爆破 (15+ 协议)
```

---

## 核心模块详解

### 1. 全局能力工厂 (`factory/`)

**职责**：作为 Agent 核心能力的**唯一构建入口**，负责实例化所有 Scanner，并注入全局统一的配置（如 QoS 策略、超时默认值）。

**核心价值**：
- **Single Source of Truth**: 消除 `RunnerManager` (CLI/Server) 和 `AutoRunner` (Pipeline) 之间的初始化逻辑重复。
- **一致性**: 确保无论在何种运行模式下，获取的扫描能力配置完全一致。
- **解耦**: 使用者无需关心 Scanner 的复杂依赖（如字典加载、引擎注册）。

**文档**: [factory/README.md](./factory/README.md)

### 2. 底层网络基础设施 (`lib/network/`)

#### 2.1 Dialer - 统一网络连接层
**职责**：提供全局超时控制、代理支持（SOCKS5）和连接复用。所有 TCP/UDP 连接必须通过此模块发起。

**文档**: [dialer/README.md](./lib/network/dialer/README.md)

#### 2.2 NetRaw - Raw Socket 和数据包构建
**职责**：提供跨平台的 Raw Socket 操作和数据包构建能力（Linux 完整支持，Windows/macOS 降级）。

**文档**: [netraw/README.md](./lib/network/netraw/README.md)

### 3. 核心数据模型 (`model/`)

**职责**：定义核心扫描引擎使用的通用数据模型 (`Task`, `TaskResult`)，实现核心层与应用层的解耦。

**文档**: [model/说明.md](./model/说明.md)

### 4. 任务参数解析 (`options/`)

**职责**：定义和解析各种扫描类型的强类型参数，提供统一的校验逻辑。

### 5. 核心编排模块 (`pipeline/`)

**职责**：将独立的扫描能力串联成有逻辑、高效的执行流。

**核心组件**：
- `AutoRunner`: 自动化编排器，通过 **Factory** 获取扫描能力。
- `PipelineContext`: 上下文对象，在各个扫描阶段间传递数据。

**扫描流程（漏斗式过滤）**：
```
输入目标 → 存活扫描 → 端口扫描 → 服务识别 → OS 识别 → 最终报告
```

**文档**: [pipeline/README.md](./pipeline/README.md)

### 6. 结果上报与输出 (`reporter/`)

**职责**：将扫描任务结果输出到不同目的地（Console, CSV, JSON）。

**文档**: [reporter/README.md](./reporter/README.md)

### 7. 任务调度与执行器 (`runner/`)

**职责**：统一的任务执行入口，管理所有扫描器的生命周期。

**核心接口**：
- `Scanner`: 扫描器统一接口
- `RunnerManager`: 执行器管理器，通过 **Factory** 注册所有可用 Runner。

**文档**: [runner/README.md](./runner/README.md)

### 8. 具体扫描能力实现 (`scanner/`)

#### 8.1 IP 存活扫描 (`alive/`)
支持 ARP/ICMP/TCP 探测，自动策略选择。
**文档**: [alive/README.md](./scanner/alive/README.md)

#### 8.2 端口服务扫描 (`port_service/`)
基于 **Gonmap** 引擎，支持 Nmap 规则库的服务指纹识别。
**文档**: [port_service/README.md](./scanner/port_service/README.md)

#### 8.3 OS 识别 (`os/`)
多引擎并发竞速（TTL, Nmap Stack, Service Banner）。
**文档**: [os/README.md](./scanner/os/README.md)

#### 8.4 弱口令爆破 (`brute/`)
支持 SSH, RDP, MySQL, Redis 等 15+ 种协议的高并发爆破，内置自适应限流 (QoS)。
**文档**: [brute/README.md](./scanner/brute/README.md)

---

## 架构设计

### 分层架构 (Layered Architecture)

引入 Factory 层后的架构视图：

```
┌─────────────────────────────────────────┐
│         Application Layer              │
│  (CLI / Server / Custom Integration)    │
└──────────────┬──────────────────────┘
               │ Request
┌──────────────▼──────────────────────┐
│         Runner Layer                  │
│  (Task Scheduling & Execution)       │
└──────────────┬──────────────────────┘
               │ Create (via Factory)
┌──────────────▼──────────────────────┐
│         Factory Layer                 │ <--- [New] Construction Center
│  (Unified Instantiation & Config)    │
└──────────────┬──────────────────────┘
               │ Produce
┌──────────────▼──────────────────────┐
│         Scanner Layer                 │
│  (Atomic Scan Capabilities)          │
└──────────────┬──────────────────────┘
               │ Use
┌──────────────▼──────────────────────┐
│         Network Layer                 │
│  (Dialer / Raw Socket)             │
└───────────────────────────────────────┘
```

### 数据流转

```
Options → Task → Runner → (Factory) → Scanner → Result → Reporter → Output
   ↓        ↓       ↓          ↓         ↓         ↓         ↓          ↓
 参数    任务    调度      构建      执行     结果     上报      输出
```

---

## 扩展指南

### 添加新的扫描器

1.  **实现核心逻辑**: 在 `scanner/` 下创建新目录，实现 `runner.Scanner` 接口。
2.  **定义参数**: 在 `options/` 下添加参数定义。
3.  **创建工厂方法**: 在 `factory/` 下新建工厂文件（如 `web_factory.go`），实现 `NewWebScanner()`。
4.  **注册能力**:
    *   在 `runner/manager.go` 中调用工厂方法注册到 `RunnerManager`。
    *   在 `pipeline/auto_runner.go` 中调用工厂方法添加到 `AutoRunner`。

---

## 版本历史

- **v1.0.0**: 初始版本，实现核心扫描能力
- **v1.1.0**: 添加 Pipeline 编排
- **v1.2.0**: 集成 Gonmap 引擎
- **v1.3.0**: 添加 OS 多引擎识别
- **v1.4.0**: **引入 Global Factory**，统一能力构建；集成 Brute 爆破模块

---

## 许可证

MIT License
