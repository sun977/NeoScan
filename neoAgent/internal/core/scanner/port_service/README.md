# Port Service Scanner (端口服务扫描器)

## 模块概述
本模块 (`internal/core/scanner/port_service`) 提供了强大的端口发现与服务指纹识别能力。它基于 **TCP Connect** 进行存活探测，并集成工业级指纹识别引擎 **Gonmap** (改编自 Qscan) 来解析 Nmap 的 `nmap-service-probes` 规则库，从而实现对服务版本、操作系统、设备类型等信息的精确识别。

## 技术原理：借鉴 Qscan 的 Gonmap 引擎

为了获得最成熟、最准确的服务识别能力，我们没有重复造轮子，而是借鉴并移植了 **Qscan** 项目中优秀的 `gonmap` 模块，并针对 NeoAgent 的架构进行了深度的适配与优化。

### 为什么选择 Gonmap (Qscan)?
1.  **纯 Go 实现**: 完全脱离了对 Nmap 二进制文件的依赖，实现了 Native Go 的解析与执行。
2.  **成熟稳定**: Qscan 的指纹引擎经过了大量实战检验，逻辑严密，对 Nmap 规则（Probe, Match, SoftMatch）的支持非常完善。
3.  **高性能**: 相比于简单的正则匹配，Gonmap 实现了完整的探针调度逻辑（排序、依赖、回退），效率极高。

### 我们做了哪些改造?
我们并未直接拷贝代码，而是进行了以下关键改造以适应 NeoAgent 的架构：
1.  **依赖清洗**: 移除了 Qscan 原有的日志库 (`gologger`)、工具库等外部依赖，替换为 NeoAgent 内部的 `logger` 和标准库，保持核心纯净。
2.  **上下文集成**: 改造了核心 `Scan` 接口，全面支持 Go `context.Context`，实现了扫描任务的可超时、可取消。
3.  **网络层抽象**: 替换了底层的网络调用，对接 NeoAgent 的 `internal/core/lib/network/dialer`，从而透明地获得了 **SOCKS5 代理支持** 和全局流量控制能力。
4.  **目录重构**: 将代码结构调整为符合 Clean Architecture 的形式，Parser、Model、Engine 职责分明。

## 核心组件

### 1. PortServiceScanner (`port_service_scanner.go`)
- **角色**: 调度器 / 扫描器入口。
- **职责**:
    - 实现 `runner.Scanner` 接口，接入任务调度系统。
    - 执行端口发现（TCP Connect）。
    - 调度 `gonmap.Engine` 对开放端口进行指纹识别。
    - 负责资源管理（信号量并发控制）。

### 2. Gonmap Engine (`gonmap/engine.go`)
- **角色**: 指纹识别核心引擎。
- **职责**:
    - **规则加载**: 解析 `nmap-service-probes` 文件，构建探针树。
    - **探针调度**: 根据端口号和稀有度（Rarity）智能选择探测包序列。
    - **交互执行**: 发送探测包（Probes），读取响应。
    - **指纹匹配**: 使用预编译的正则规则匹配响应，提取服务信息。

### 3. Parser & Types (`gonmap/parser.go`, `gonmap/types.go`)
- **角色**: 规则解析器与数据模型。
- **职责**:
    - 解析 Nmap 复杂的指令集 (`Probe`, `match`, `softmatch`, `ports`, `sslports`, `rarity`, `fallback`)。
    - 定义标准化的指纹数据结构 (`FingerPrint`)。

## 扫描流程

1.  **任务接收**: Scanner 接收 `Task`，解析目标 IP 和端口范围。
2.  **规则初始化**: 首次运行时，懒加载 `nmap-service-probes` 规则库。
3.  **并发扫描**:
    - 使用 Semaphore 控制并发度（默认 CLI 参数或内部默认值）。
    - 对每个目标端口发起 TCP Connect。
4.  **服务识别 (Service Discovery)**:
    - 如果端口开放且开启了 `-s` (Service Detect)：
    - 调用 `gonmapEngine.Scan(ctx, ip, port)`。
    - 引擎根据端口优选探针（如 80 端口优先发 HTTP Get，22 端口优先等 Banner）。
    - 匹配响应，返回 `FingerPrint`。
5.  **结果聚合**: 将端口状态和服务信息封装为 `model.TaskResult` 返回。

## 配置与规则

- **规则文件**: 目前支持从 `rules/fingerprint/nmap-service-probes` 加载。
- **并发控制**: 支持通过 Task 参数 `rate` 动态调整扫描并发度。

## 局限性

- **UDP 支持**: 目前 Gonmap 移植版主要专注于 TCP 协议的服务识别。
- **NSE 脚本**: 不支持 Nmap 的 Lua 脚本引擎（这是极其复杂且独立的子系统）。
