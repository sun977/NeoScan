Orchestrator : 项目、工作流、阶段、工具模板、结果存储、统计
scheduler目录下的模块是NeoScan扫描编排系统的核心调度引擎，包含以下模块：

## 模块作用

Scheduler 目录
- engine.go：调度引擎主服务，负责定时任务管理、项目状态跟踪、阶段流转控制
- generator.go：任务生成器，将扫描阶段配置和目标列表转换为具体的Agent任务
TaskDispatcher 目录
- dispatcher.go：任务分发器，负责将待执行任务分发给合适的Agent
- agent_task.go：Agent任务服务，处理Agent的任务获取、状态更新等操作

### 1. engine.go - 调度引擎主服务
- **职责**：负责定时触发和项目级流程控制
- **核心功能**：
    - 定时任务管理：检查并触发配置了Cron表达式的定时扫描项目
    - 项目状态跟踪：监控运行中项目的进度和状态
    - 阶段流转控制：确保项目按预定义的工作流顺序执行各个阶段
    - 任务生成：根据阶段配置和目标生成具体可执行的任务
    - 策略执行：在任务执行前进行安全和合规性检查

### 2. generator.go - 任务生成器
- **职责**：将扫描阶段配置和目标列表转换为具体的Agent任务
- **核心功能**：
    - 根据扫描阶段配置生成任务
    - 将目标列表按分块大小分割成多个批次
    - 构造任务参数和策略快照
    - 生成可下发给Agent的任务对象

### 3. README.md - 模块说明
- 简要说明调度器的核心组件和职责

## 与TargetProvider和PolicyEnforcer的配合

### TargetProvider配合
- 在 engine.go 的 generateTasksForStage方法中：
    - 获取项目种子目标（从Project.TargetScope）
    - 使用TargetProvider的 ResolveTargets 方法解析最终目标
    - 应用 ScanStage.TargetPolicy 进行目标转换和过滤
    - 将解析后的目标传递给任务生成器

### PolicyEnforcer配合
- 在 engine.go 的 generateTasksForStage 方法中：
    - 在任务保存到数据库前，调用PolicyEnforcer的 Enforce 方法
    - 对生成的任务进行最终安全校验
    - 如果校验不通过，任务状态设置为"failed"并添加错误信息

## 工作流程

```mermaid
graph TD
    A[启动调度引擎] --> B[定时轮询运行中项目]
    B --> C{获取运行中项目}
    C --> D[检查项目任务状态]
    D --> E{上一任务是否失败?}
    E -->|是| F[暂停项目,设置为error状态]
    E -->|否| G[查找下一阶段]
    G --> H{是否有下一阶段?}
    H -->|否| I[项目完成,设置为finished状态]
    H -->|是| J[获取种子目标]
    J --> K[TargetProvider解析目标]
    K --> L[应用TargetPolicy过滤]
    L --> M[任务生成器生成任务]
    M --> N[PolicyEnforcer策略校验]
    N --> O{任务是否合规?}
    O -->|否| P[任务标记为failed,添加错误信息]
    O -->|是| Q[保存任务到数据库]
    Q --> R[等待Agent执行]
    R --> S[检查任务超时]
    S --> D
```


## 详细流程图

```mermaid
graph TB
    subgraph "调度器主循环"
        A[Start - 启动调度引擎] --> B[loop - 定时轮询]
        B --> C[checkScheduledProjects - 检查定时任务]
        B --> D[checkTaskTimeouts - 检查任务超时]
        B --> E[GetRunningProjects - 获取运行中项目]
    end

    subgraph "项目处理流程"
        E --> F[ProcessProject - 处理单个项目]
        F --> G[HasRunningTasks - 检查是否有运行中任务]
        G --> H{上一任务失败?}
        H -->|是| I[设置项目为error状态]
        H -->|否| J[findNextStages - 查找下一阶段]
        J --> K{有下一阶段?}
        K -->|否| L{有运行中任务?}
        L -->|是| M[继续等待]
        L -->|否| N[设置项目为finished状态]
        K -->|是| O[generateTasksForStage - 为阶段生成任务]
    end

    subgraph "任务生成流程"
        O --> P[获取种子目标]
        P --> Q[TargetProvider.ResolveTargets - 解析目标]
        Q --> R[应用TargetPolicy过滤]
        R --> S[TaskGenerator.GenerateTasks - 生成任务]
        S --> T[PolicyEnforcer.Enforce - 策略校验]
        T --> U{任务合规?}
        U -->|否| V[任务状态设为failed]
        U -->|是| W[保存任务到数据库]
    end

    subgraph "TargetProvider内部流程"
        Q --> Q1[fetchTargetsFromSources - 从各种源获取目标]
        Q1 --> Q2[applyWhitelist - 应用白名单过滤]
        Q2 --> Q3[applySkipRule - 应用跳过规则]
        Q3 --> Q4[deduplicateTargets - 去重]
    end

    subgraph "PolicyEnforcer内部流程"
        T --> T1[ScopeValidator - 作用域校验]
        T1 --> T2[WhitelistChecker - 全局白名单校验]
        T2 --> T3[SkipLogicEvaluator - 全局跳过策略校验]
    end
```


## 关键配合点

1. **目标解析**：调度器通过TargetProvider将多种来源的目标统一为标准格式
2. **策略过滤**：TargetProvider应用局部策略（阶段级）进行初步过滤
3. **最终校验**：PolicyEnforcer对生成的任务进行全局策略校验
4. **任务生成**：任务生成器将过滤后的目标转换为可执行的Agent任务
5. **流程控制**：调度器控制整个工作流的执行顺序和状态流转

## 补充-工作流程
```mermaid
graph TD
    subgraph "调度阶段"
        A[Scheduler.Start] --> B[定时轮询]
        B --> C[获取运行中项目]
        C --> D[查找下一阶段]
        D --> E[获取种子目标]
        E --> F[TargetProvider.ResolveTargets]
        F --> G[应用TargetPolicy过滤]
        G --> H[TaskGenerator.GenerateTasks]
        H --> I[PolicyEnforcer.Enforce]
        I --> J{任务合规?}
        J -->|是| K[保存任务到数据库]
        J -->|否| L[任务标记为failed]
    end

    subgraph "分发阶段"
        M[Agent.FetchTasks] --> N[TaskDispatcher.Dispatch]
        N --> O[获取待执行任务]
        O --> P[ResourceAllocator.CanExecute]
        P --> Q{Agent可执行?}
        Q -->|是| R[PolicyEnforcer.Enforce]
        Q -->|否| S[跳过此任务]
        R --> T{任务合规?}
        T -->|是| U[ClaimTask原子操作]
        T -->|否| V[任务标记为failed]
        U --> W[任务分发给Agent]
    end

    K --> O
    L --> X[等待重试或结束]
    V --> Y[避免重复分发]
    S --> Z[继续寻找其他任务]
```

## 补充-协作详细流程
```mermaid
graph TB
    subgraph "调度器层"
        A[Project.Start] --> B[Scheduler.Engine]
        B --> C[获取种子目标]
        C --> D[TargetProvider]
        D --> E[解析多源目标]
        E --> F[应用局部策略过滤]
        F --> G[返回统一Target对象]
        G --> H[TaskGenerator]
        H --> I[生成AgentTask对象]
        I --> J[PolicyEnforcer]
        J --> K[全局策略校验]
        K --> L[任务入库-Pending状态]
    end

    subgraph "分发器层"
        M[Agent.RequestTasks] --> N[TaskDispatcher]
        N --> O[获取Pending任务]
        O --> P[ResourceAllocator]
        P --> Q[能力匹配检查]
        Q --> R[PolicyEnforcer]
        R --> S[最终合规检查]
        S --> T[ClaimTask原子操作]
        T --> U[任务分发给Agent]
    end

    subgraph "TargetProvider内部"
        D --> D1[注册多种Provider]
        D1 --> D2[file,db,api,manual等]
        D2 --> D3[并发获取目标]
        D3 --> D4[白名单过滤]
        D4 --> D5[跳过规则过滤]
        D5 --> D6[目标去重]
    end

    subgraph "PolicyEnforcer内部"
        J --> J1[ScopeValidator]
        J1 --> J2[WhitelistChecker]
        J2 --> J3[SkipLogicEvaluator]
        R --> J1
    end

    L --> O
    F --> H
    G --> H
    I --> J
    K --> L
    S --> T

```

## 配合点
- 目标生成：调度器使用TargetProvider从多种来源获取并标准化目标
- 策略过滤：调度器应用局部策略过滤目标，生成任务后进行全局策略校验
- 任务分发：分发器在分发前再次进行策略校验，确保任务合规
- 资源匹配：分发器结合资源分配器，确保任务分发给合适的Agent
- 状态管理：两层策略校验确保只有合规任务才能执行
- 这种设计实现了分层控制：调度器负责任务生成和初步校验，分发器负责运行时分发和最终校验，确保了系统的安全性和可靠性
