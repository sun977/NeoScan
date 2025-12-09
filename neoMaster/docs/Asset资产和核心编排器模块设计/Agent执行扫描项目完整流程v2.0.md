# Agent执行扫描项目完整流程 v2.0

## 1. 核心架构变更说明

相较于 v1.0 版本，v2.0 采用了 **"Thin Agent" (瘦客户端)** 架构模式。

- **Master (主脑)**: 承担所有状态维护、工作流编排、数据清洗和决策逻辑。
- **Agent (执行器)**: 变为无状态的纯执行单元，只负责接收单一 `ScanStage` 任务，执行并返回结果，不感知整个 `Project` 的存在。

这种架构使得系统具备了更高的弹性、容错性和可扩展性。

## 2. 完整执行流程概览

整个流程由 Master 端的 **"阶段调度循环" (Stage Scheduling Loop)** 驱动。

```mermaid
sequenceDiagram
    participant User
    participant Master as Master (Brain)
    participant DB as Database
    participant MQ as TaskQueue
    participant Agent as Agent (Worker)
    
    %% 1. 提交项目
    User->>Master: 1. 提交扫描项目 (Project)
    activate Master
    Master->>DB: 1.1 保存 Project (Status: Pending)
    Master->>DB: 1.2 初始化 Workflow (Stage 1 Ready)
    Master-->>User: 返回 ProjectID
    deactivate Master

    %% 2. 调度循环
    loop 阶段调度循环 (直到所有阶段完成)
        activate Master
        %% 2.1 准备阶段
        Master->>Master: 2.1 检查当前待执行阶段 (Current Stage)
        Master->>DB: 2.2 获取上一阶段结果 (作为输入)
        Master->>Master: 2.3 生成原子任务 (ScanStage Task)
        
        %% 2.2 下发任务
        Master->>MQ: 2.4 下发任务 (Task)
        deactivate Master
        
        activate Agent
        MQ->>Agent: 3.1 领取任务
        Note right of Agent: Agent 是无状态的<br/>只负责执行单一命令
        Agent->>Agent: 3.2 执行扫描工具 (Nmap/Nuclei等)
        Agent->>Agent: 3.3 封装结果 (StageResult)
        Agent->>Master: 3.4 返回执行结果
        deactivate Agent
        
        activate Master
        %% 2.3 处理结果
        Master->>Master: 4.1 结果清洗与验证
        Master->>DB: 4.2 结果持久化 (Output Config)
        Master->>Master: 4.3 判定是否存在下一阶段
        
        alt 有下一阶段
            Master->>DB: 更新 Project 状态 (Stage N Done -> Stage N+1 Ready)
        else 无更多阶段
            Master->>DB: 更新 Project 状态 (Completed)
        end
        deactivate Master
    end
    
    Master->>User: 5. 通知/展示最终结果
```

### 逻辑流程图 (Logic Flowchart)

为了更直观地展示 **"阶段调度循环"** 的逻辑判定与数据流向，补充如下流程图：

```mermaid
graph TD
    %% 节点样式定义
    classDef master fill:#e3f2fd,stroke:#1565c0,stroke-width:2px,rx:5,ry:5;
    classDef agent fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px,rx:5,ry:5;
    classDef data fill:#fff3e0,stroke:#ef6c00,stroke-width:2px,rx:5,ry:5;
    classDef startend fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,rx:10,ry:10;

    Start((用户提交项目)):::startend --> Init[1. 初始化项目 & Workflow<br/>Status: Pending]:::master
    Init --> LoopCheck{2. 调度循环:<br/>检查当前待执行阶段?}:::master

    subgraph Master_Core [Master核心调度层]
        LoopCheck -->|存在 Ready 阶段| PrepInput[2.1 准备输入数据<br/>从上一阶段输出提取]:::master
        PrepInput --> MakeTask[2.2 生成 ScanStage 原子任务]:::master
        MakeTask --> Dispatch[2.3 下发任务到队列]:::master
    end

    subgraph Agent_Execution [Agent无状态执行层]
        Dispatch -.->|MQ/HTTP| PickTask[3.1 Agent 领取任务]:::agent
        PickTask --> Exec[3.2 执行扫描工具<br/>Nmap/Masscan/Nuclei]:::agent
        Exec --> Report[3.3 上报 StageResult]:::agent
    end

    subgraph Result_Process [结果处理层]
        Report -.->|HTTP| Clean[4.1 结果清洗与验证]:::master
        Clean --> Persist[(4.2 结果持久化)]:::data
        Persist --> HasNext{4.3 是否存在<br/>下一阶段?}:::master
        
        HasNext -->|Yes| NextState[更新状态:<br/>Current Stage -> Done<br/>Next Stage -> Ready]:::master
        NextState --> LoopCheck
        
        HasNext -->|No| Complete[标记项目为 Completed]:::master
    end

    Complete --> Notify((5. 通知用户)):::startend
```

## 3. 详细执行步骤

### 3.1 项目初始化 (Project Initialization)
用户提交扫描请求后，Master 并不直接将整个请求发给 Agent，而是：
1.  **解析工作流**: 确定该项目需要执行哪些阶段（例如：IP探活 -> 端口扫描 -> Web爬虫）。
2.  **持久化状态**: 在数据库中创建 `Project` 记录，状态置为 `Pending`，并将指针指向 `Stage 1`。

### 3.2 任务分发 (Task Dispatch)
Master 的调度器周期性或事件驱动地检查待执行的 Stage：
1.  **输入准备**: 根据 `TargetPolicy`，从上一阶段的输出（或用户初始输入）中提取当前阶段的目标（如 IP 列表）。
2.  **任务封装**: 将工具配置、目标列表、超时设置等封装为一个独立的 `ScanStage` 对象。
3.  **路由分发**: 将任务推送到任务队列或直接 RPC 调用空闲 Agent。此时可以选择 **任意** Agent，不需要与上一阶段是同一个 Agent。

### 3.3 Agent 执行 (Stateless Execution)
Agent 接收到的是一个自包含的原子任务：
1.  **环境准备**: 下载必要的脚本或插件。
2.  **工具运行**: 调用底层安全工具（Nmap, Masscan, Nuclei 等）。
3.  **结果封装**: 将工具的原始输出（Raw Output）和结构化数据封装为 `StageResult`。
4.  **立即销毁**: 任务完成后，Agent 立即销毁上下文，恢复空闲状态。

### 3.4 结果处理与流转 (Result Processing & Transition)
Master 收到 `StageResult` 后：
1.  **数据清洗**: 根据 `OutputConfig` 过滤无效数据（如去除重复 IP、过滤白名单等）。
2.  **持久化**: 将结果存入数据库，作为 **"中间资产"** 或 **"最终资产"**。
3.  **状态流转**: 
    - 检查当前 Workflow 是否还有后续阶段。
    - 如果有，将当前结果标记为下一阶段的潜在输入源。
    - 更新 Project 进度。

## 4. 数据流转机制 (Data Flow)

数据如何在不同阶段间流转是 v2.0 的核心。

```mermaid
graph LR
    subgraph Stage 1: 网段探活
        Input1[用户输入: 192.168.1.0/24] --> Task1[Nmap Ping Scan]
        Task1 --> Result1[结果: 192.168.1.5, 192.168.1.6]
    end
    
    Result1 -->|Master 提取 & 转换| Input2
    
    subgraph Stage 2: 端口扫描
        Input2[输入: IP List] --> Task2[Masscan/Nmap Port Scan]
        Task2 --> Result2[结果: 1.5:80, 1.6:443]
    end
    
    Result2 -->|Master 提取 & 转换| Input3
    
    subgraph Stage 3: Web扫描
        Input3[输入: http://1.5:80, https://1.6:443] --> Task3[Nuclei Scan]
        Task3 --> Result3[结果: 漏洞列表]
    end
```

## 5. 为什么选择这种架构？(Advantages)

1.  **弹性伸缩 (Scalability)**:
    - 不同的阶段可以由不同的 Agent 执行。
    - 端口扫描阶段任务重，可以动态增加 10 个 Agent 并行处理，而 Web 扫描阶段可能只需要 2 个 Agent。
    
2.  **容错性 (Resilience)**:
    - 如果 Agent A 在执行 Stage 2 时崩溃，Master 只需将该 Stage 任务重新分配给 Agent B，而不需要重跑 Stage 1。
    - v1.0 中 Agent 崩溃会导致整个 Project 失败。

3.  **资源管控 (Resource Control)**:
    - Master 可以精确控制并发度。例如，限制同时进行端口扫描的任务数，防止打崩网络，而让轻量级的探活任务先行。

4.  **逻辑解耦**:
    - Agent 不需要知道业务逻辑（如"扫完端口后要扫Web"），它只需要知道"现在扫这个端口"。这使得 Agent 代码极度简化且稳定。
