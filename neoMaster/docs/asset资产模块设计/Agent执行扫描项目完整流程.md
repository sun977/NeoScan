# Agent执行扫描项目完整流程

## 概述

本文档详细描述了Agent接收并执行完整扫描项目的流程。一个扫描项目包含多个按顺序执行的扫描阶段，Agent需要依次执行所有阶段，而非根据扫描类型选择性执行。

## 完整执行流程

### 1. 项目接收与解析

```mermaid
sequenceDiagram
    participant Master
    participant Agent
    participant WorkflowEngine
    
    Master->>Agent: 下发完整扫描项目
    Note over Master,Agent: 包含4个扫描阶段的完整工作流
    Agent->>WorkflowEngine: 解析项目结构
    WorkflowEngine->>Agent: 生成阶段执行计划
```

### 2. 阶段顺序执行流程

```mermaid
graph TD
    A[开始执行项目] --> B[阶段1: 网段扫描]
    B --> C[阶段2: 端口扫描]
    C --> D[阶段3: Web信息爬取]
    D --> E[阶段4: Web漏洞扫描]
    E --> F[项目完成]
```

### 3. 各阶段详细执行过程

#### 阶段1: 网段扫描

```mermaid
graph TD
    A[执行网段扫描] --> B[Nmap探活扫描]
    B --> C[识别存活主机]
    C --> D[生成StageResult]
    D --> E{保存配置}
    E -->|落库| F[保存到StageResult表]
    E -->|不落库| G[仅内存处理]
    F --> H[阶段1完成]
    G --> H
```

#### 阶段2: 端口扫描

```mermaid
graph TD
    A[执行端口扫描] --> B[获取阶段1结果]
    B --> C[Nmap端口扫描]
    C --> D[服务识别与版本检测]
    D --> E{是否为Web服务}
    E -->|是| F[执行HTTP脚本]
    F --> G[提取基础Web信息]
    G --> H[生成AssetWeb记录]
    E -->|否| I[生成AssetService记录]
    I --> J[生成StageResult]
    H --> J
    J --> K{保存配置}
    K -->|落库| L[保存到StageResult表]
    K -->|转换落库| M[转换到最终资产表]
    K -->|提取字段| N[提取并保存到指定表]
    L --> O[阶段2完成]
    M --> O
    N --> O
```

#### 阶段3: Web信息爬取

```mermaid
graph TD
    A[执行Web信息爬取] --> B[获取Web服务列表]
    B --> C[HTTP深度爬取]
    C --> D[分析页面内容]
    D --> E[生成WebDetail记录]
    E --> F[生成StageResult]
    F --> G{保存配置}
    G -->|落库| H[保存到StageResult表]
    G -->|转换落库| I[转换到最终资产表]
    G -->|提取字段| J[提取并保存到指定表]
    H --> K[阶段3完成]
    I --> K
    J --> K
```

#### 阶段4: Web漏洞扫描

```mermaid
graph TD
    A[执行Web漏洞扫描] --> B[获取Web资产列表]
    B --> C[Nuclei漏洞扫描]
    C --> D[漏洞识别与验证]
    D --> E[生成AssetVuln记录]
    E --> F[生成StageResult]
    F --> G{保存配置}
    G -->|落库| H[保存到StageResult表]
    G -->|转换落库| I[转换到最终资产表]
    G -->|提取字段| J[提取并保存到指定表]
    H --> K[阶段4完成]
    I --> K
    J --> K
```

### 4. 完整项目执行流程

```mermaid
sequenceDiagram
    participant Master
    participant Agent
    participant Nmap
    participant Crawler
    participant Nuclei
    
    Master->>Agent: 下发完整扫描项目(4个阶段)
    Note over Master,Agent: 项目包含网段扫描→端口扫描→Web爬取→漏洞扫描
    
    Agent->>Agent: 解析项目工作流
    Agent->>Nmap: 执行阶段1 - 网段扫描
    Nmap-->>Agent: 返回存活主机列表
    Agent->>Master: 阶段1结果回传
    
    Agent->>Nmap: 执行阶段2 - 端口扫描
    Nmap-->>Agent: 返回端口和服务信息
    Agent->>Master: 阶段2结果回传
    
    Agent->>Crawler: 执行阶段3 - Web信息爬取
    Crawler-->>Agent: 返回详细Web信息
    Agent->>Master: 阶段3结果回传
    
    Agent->>Nuclei: 执行阶段4 - Web漏洞扫描
    Nuclei-->>Agent: 返回漏洞信息
    Agent->>Master: 阶段4结果回传
    
    Agent->>Master: 项目执行完成通知
```

### 5. 数据流转与处理

```mermaid
graph TD
    A[Master下发项目] --> B[Agent接收并解析]
    B --> C[阶段1执行]
    C --> D[生成阶段结果1]
    D --> E[根据output_config处理]
    E --> F[回传结果给Master]
    
    F --> G[阶段2执行]
    G --> H[生成阶段结果2]
    H --> I[根据output_config处理]
    I --> J[回传结果给Master]
    
    J --> K[阶段3执行]
    K --> L[生成阶段结果3]
    L --> M[根据output_config处理]
    M --> N[回传结果给Master]
    
    N --> O[阶段4执行]
    O --> P[生成阶段结果4]
    P --> Q[根据output_config处理]
    Q --> R[回传结果给Master]
    
    R --> S[项目完成]
```

## 关键设计要点

### 1. 顺序执行
- 严格按照项目定义的阶段顺序执行
- 每个阶段必须完成后才能进入下一阶段

### 2. 阶段依赖
- 后续阶段可以使用前一阶段的结果作为输入
- 阶段间数据通过StageResult模型传递

### 3. 统一结果处理
- 每个阶段都生成StageResult，并根据output_config进行相应处理
- 支持多种结果处理方式：落库、转换落库、提取字段等

### 4. 灵活配置
- 每个阶段的结果处理方式可通过配置灵活调整
- 支持不同的保存类型和目标表配置

### 5. 完整回溯
- 保持完整的阶段执行链路，便于审计和问题排查
- 通过source_stage_ids维护结果来源信息

## 总结

这种设计确保了扫描项目的完整性和一致性，同时保持了各阶段处理的灵活性。Agent严格按照项目定义的顺序执行所有阶段，确保扫描结果的完整性和准确性。