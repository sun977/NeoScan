# scan run 全流程处理逻辑

> **版本**: v1.1
> **更新日期**: 2026-08-14
> **状态**: 核心流程已落地，部分细节过期（已标注）

---

## 修改记录

| 日期 | 版本 | 变更说明 |
|------|------|---------|
| 2026-08-14 | v1.1 | 标注过期内容，Web Scan 已实现，补充 Vuln/Dir/API Scanner 状态 |

---

本文档描述了 `scan run` 命令的完整执行流程，包括当前已实现的模块和未来规划的功能。

## 流程图

```mermaid
flowchart TD
    %% 样式定义
    classDef planned stroke-dasharray: 5 5,fill:#f9f9f9,stroke:#999;
    classDef implemented fill:#e1f5fe,stroke:#01579b;
    classDef decision fill:#fff9c4,stroke:#fbc02d;
    classDef startend fill:#e0e0e0,stroke:#333;
    classDef skipped fill:#ffebee,stroke:#c62828;

    Start((开始)):::startend --> Init[初始化选项 & 目标解析]:::implemented
    Init -->|"--target"| TargetGen[生成目标 IP 列表]:::implemented
    TargetGen --> LoopStart{并发循环处理每个 IP}:::decision
    
    %% 并发控制
    LoopStart -->|"--concurrency"| Phase1_Start[Phase 1: 基础信息收集]:::implemented

    subgraph Phase1 [Phase 1: 基础发现]
        direction TB
        Phase1_Start --> AliveScan[Alive Scan<br/>存活检测]:::implemented
        AliveScan --> IsAlive{目标存活?}:::decision
        IsAlive -- "No" --> SkipIP[跳过该 IP]:::implemented
        IsAlive -- "Yes" --> PortScan[Port Scan<br/>端口扫描]:::implemented
        
        PortScan -->|"--port"| HasPorts{发现开放端口?}:::decision
        HasPorts -- "No" --> SkipIP
        HasPorts -- "Yes" --> ServiceScan[Service Scan<br/>服务指纹识别]:::implemented
        
        ServiceScan --> OSScan[OS Scan<br/>操作系统识别]:::implemented
        OSScan --> Phase1_End[Phase 1 完成]:::implemented
    end

    Phase1_End --> Phase2_Start[Phase 2: 深度分析分发器]:::implemented

    subgraph Phase2 ["Phase 2: 深度分析 (Dispatcher)"]
        direction TB
        Phase2_Start --> HighPriority[高优先级任务组]:::implemented
        
        %% 高优先级并行
        subgraph HighPri ["Web & 漏洞 (并行)"]
            direction LR
            IsWeb{是 Web 服务?}:::decision -. "Yes" .-> WebScan[Web Scan<br/>Web 指纹/爬虫<br/>CDN 检测]:::implemented
            HasVulnSvc{有潜在漏洞服务?}:::decision -. "Yes" .-> VulnScan[Vuln Scan<br/>POC 验证]:::planned
        end
        
        HighPriority --> HighPri
        HighPri --> WaitHigh[等待高优先级任务完成]:::implemented
        
        WaitHigh --> LowPriority[低优先级任务组]:::implemented
        
        LowPriority --> CheckBrute{启用爆破?}:::decision
        CheckBrute -- "No (--brute=false)" --> Phase2_End[Phase 2 完成]:::implemented
        CheckBrute -- "Yes (--brute=true)" --> BruteScan[Brute Force<br/>弱口令爆破]:::implemented
        
        BruteScan -->|"--users / --pass"| BruteResult[生成爆破结果]:::implemented
        BruteResult --> Phase2_End
    end

    Phase2_End --> Collect[收集单机结果]:::implemented
    Collect --> LoopEnd{所有 IP 完成?}:::decision
    
    LoopEnd -- "No" --> LoopStart
    LoopEnd -- "Yes" --> CheckSummary{显示汇总?}:::decision
    
    CheckSummary -- "No" --> End((结束)):::startend
    CheckSummary -- "Yes (--summary)" --> PrintReport[打印最终汇总报告]:::implemented
    PrintReport --> End

    %% 连接线标注
    SkipIP --> Collect

    %% 新增：未接入 Pipeline 的扫描器
    subgraph "未来接入 (Pending)"
        direction LR
        Pending1[Dir Scanner<br/>目录/路径爆破]:::skipped
        Pending2[API Scanner<br/>JS 接口提取]:::skipped
    end
```

> **变更说明**:
> - Web Scan 已从 `planned` 更新为 `implemented`（2026-08 已实现）
> - ~~Vuln Scan 仍为 `planned`~~（实际未实现，Phase 5.2 规划中）
> - Dir Scanner 和 API Scanner **未接入** Pipeline（见 `internal/core/pipeline/README.md`）

## 关键流程说明

### Phase 1: 基础发现 (已完全实现)
- **Alive Scan**: 使用 ICMP/ARP/TCP Ping 检测主机存活。不存活的主机直接跳过。
- **Port Scan**: 扫描开放端口，受 `--port` 参数控制范围。
- **Service Scan**: 基于 nmap-service-probes 指纹库识别服务版本。
- **OS Scan**: 基于 TTL 和指纹推断操作系统。

### Phase 2: 深度分析 (部分实现)
- **Dispatcher**: 负责根据服务类型分发后续任务。
- **Web Scan (已实现)**: 
  - 针对 Web 服务进行指纹识别、爬虫、CDN 边缘节点检测。
  - **状态**: ✅ 已完全实现（2026-08）
- **Vuln Scan (~~规划中~~ 未实现)**: ~~针对 Web 服务进行 POC 漏洞验证~~
  - **状态**: ❌ **未实现**（Phase 5.2 规划中，代码中标记 `// TODO`）
- **Brute Force (已实现)**: 
  - 受 `--brute` 参数控制开启。
  - 针对 SSH, FTP, Telnet, MySQL, Postgres, Redis 等服务进行弱口令检测。
  - 支持 `--users` 和 `--pass` 自定义字典。
  - **状态**: ✅ 已实现
- **Dir Scanner (~~规划中~~ 未接入)**: ~~目录/路径扫描~~
  - **状态**: 📝 **未接入 Pipeline**（仅 CLI 独立命令 `scan dir`，详见 `docs/目录扫描/`）
- **API Scanner (~~规划中~~ 未接入)**: ~~JS 接口提取~~
  - **状态**: 📝 **未接入 Pipeline**（仅 CLI 独立命令 `scan api`，详见 `docs/API 扫描-js提取/`）

## 参数影响总结

| 参数 | 影响阶段 | 说明 | 状态 |
| :--- | :--- | :--- | :---: |
| `--target` | 初始化 | 决定扫描目标的数量和范围 | ✅ |
| `--concurrency` | 循环控制 | 决定同时进行 Phase 1 的 IP 数量 | ✅ |
| `--port` | Phase 1 | 决定端口扫描的覆盖范围 | ✅ |
| ~~`--vuln`~~ | Phase 2 | ~~决定是否开启漏洞扫描~~ | ❌ **未实现** |
| `--brute` | Phase 2 | **关键开关**，决定是否进行弱口令爆破 | ✅ |
| `--users/--pass` | Phase 2 | 仅在爆破开启时生效，覆盖默认字典 | ✅ |
| `--web` | Phase 2a | 开启 Web 指纹识别/爬虫/CDN 检测 | ✅ |
| ~~`--dir`~~ | Phase 2a | ~~开启目录/路径扫描~~ | 📝 **未接入 Pipeline** |
| ~~`--api`~~ | Phase 2b | ~~开启 API 接口提取~~ | 📝 **未接入 Pipeline** |
| `--summary` | 结束阶段 | 决定是否输出统计报告 | ✅ |
