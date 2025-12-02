# Workflow 实体模型定义

## 概述

`Workflow`（工作流）是 NeoScan 的核心执行单元。它定义了一组有序的扫描阶段（ScanStage），描述了"如何"完成一次完整的扫描任务。

**核心职责**：
1.  **编排**：定义 ScanStage 的执行顺序（DAG 或 线性链）。
2.  **上下文**：管理在不同 Stage 之间传递的全局变量和环境配置。
3.  **策略**：定义错误处理、超时控制和通知策略。

**与 Project 的关系**：
- `Project` 是**调度器**（When & Who），`Workflow` 是**执行蓝图**（How）。
- 一个 Project 可以包含多个 Workflow。
- Project 负责触发 Workflow，Workflow 负责触发 Stage。

## 模型结构

### 核心字段

| 字段名 | 类型 | 描述 | 索引 |
|--------|------|------|------|
| `id` | uint | 自增主键 | PK |
| `name` | string | 工作流唯一标识名 | Unique |
| `display_name` | string | 显示名称 | |
| `version` | string | 版本号 (v1.0.0) | |
| `description` | text | 描述 | |
| `enabled` | bool | 启用状态 | |
| `exec_mode` | string | 阶段执行模式 (sequential/parallel/dag) | |
| `global_vars` | JSON | 全局变量定义 (供所有 Stage 引用) | |
| `policy_config` | JSON | 执行策略配置 (超时/重试/通知) | |
| `tags` | JSON | 标签列表 | |
| `created_by` | uint | 创建者 ID | |
| `updated_by` | uint | 更新者 ID | |
| `created_at` | timestamp | 创建时间 | |
| `updated_at` | timestamp | 更新时间 | |
| `deleted_at` | timestamp | 软删除时间 | Index |

### 关联关系

- **BelongsTo Project**: (可选) 如果设计为多对多复用，则不在此处存储 `project_id`，而是通过中间表。*鉴于你的设计图是 1:N，这里保留 ProjectID 字段，但推荐使用中间表实现更灵活的复用。*
- **HasMany ScanStages**: 一个工作流包含多个扫描阶段。

## 字段详解

### 1. exec_mode (执行模式)
定义内部 Stage 的流转方式：
- `sequential`: **线性执行**。Stage 1 -> Stage 2 -> Stage 3。最常用。
- `parallel`: **全并行**。所有 Stage 同时启动。
- `dag`: **有向无环图**。基于 Stage 间的依赖关系执行（高级特性，暂不推荐 MVP 阶段实现）。

### 2. global_vars (全局变量)
定义在整个工作流生命周期内有效的变量，Stage 可以通过 `${var_name}` 语法引用。

```json
{
  "target_corp": "ExampleCorp",
  "scan_depth": "deep",
  "proxy_server": "socks5://1.2.3.4:1080"
}
```

### 3. policy_config (策略配置)
收敛所有运行时控制策略：

```json
{
  "timeout_seconds": 3600,      // 整个工作流的总超时
  "max_retries": 3,             // 失败重试次数
  "continue_on_error": false,   // 某个 Stage 失败是否继续执行后续 Stage
  "notify": {                   // 工作流级别的通知（覆盖 Project 级别或作为补充）
    "on_start": false,
    "on_success": true,
    "on_failure": true
  }
}
```

### 4. 运行时统计 (Stateless Design)
**注意**：`execution_count`, `success_count` 等统计字段**不应设计在 Workflow 主表中**。
原因：
1.  **高频写**：每次执行都要更新，会导致配置表被频繁锁定，影响读取。
2.  **单一职责**：Workflow 表是"配置"，不应包含"状态"。

**建议方案**：
创建一个独立的 `WorkflowStats` 表，用于缓存高频访问的统计数据。

### 5. WorkflowStats (运行时统计)

**设计初衷**：分离冷热数据。`Workflow` 表是配置（冷数据，读多写少），`WorkflowStats` 表是状态（热数据，写多读多）。

| 字段名 | 类型 | 描述 | 索引 |
|--------|------|------|------|
| `workflow_id` | uint | 关联的工作流 ID | PK |
| `total_execs` | int | 总执行次数 | |
| `success_execs` | int | 成功次数 | |
| `failed_execs` | int | 失败次数 | |
| `avg_duration_ms` | int | 平均执行耗时 (毫秒) | |
| `last_exec_id` | string | 最后一次执行 ID | |
| `last_exec_status` | string | 最后一次执行状态 | |
| `last_exec_time` | timestamp | 最后一次执行时间 | |
| `updated_at` | timestamp | 统计更新时间 | |

**更新策略**：
- **异步更新**：每次 WorkflowExecution 结束时，通过消息队列或后台任务更新此表。
- **定期校准**：每周运行一次聚合查询，校准此表数据，消除计数器漂移。

## 数据库设计 (GORM 示例)

```go
type Workflow struct {
    gorm.Model
    Name          string         `gorm:"type:varchar(100);uniqueIndex;not null;comment:工作流标识"`
    DisplayName   string         `gorm:"type:varchar(200);comment:显示名称"`
    Version       string         `gorm:"type:varchar(20);default:'v1.0.0';comment:版本号"`
    Description   string         `gorm:"type:text;comment:描述"`
    
    Enabled       bool           `gorm:"default:true;comment:启用开关"`
    ExecMode      string         `gorm:"type:varchar(20);default:'sequential';comment:阶段执行模式"`
    
    // 配置 JSON
    GlobalVars    datatypes.JSON `gorm:"type:json;comment:全局变量"`
    PolicyConfig  datatypes.JSON `gorm:"type:json;comment:执行策略"`
    Tags          datatypes.JSON `gorm:"type:json;comment:标签"`
    
    // 审计
    CreatedBy     uint           `gorm:"index"`
    UpdatedBy     uint           `gorm:"index"`
    
    // 关联
    Stages        []ScanStage    `gorm:"foreignKey:WorkflowID"`
    // 如果是多对多，则不需要 ProjectID；如果是 1:N，则需要
    // ProjectID  uint           `gorm:"index"` 
}

// WorkflowPolicy 定义 (嵌入在 Workflow.PolicyConfig 中)
type WorkflowPolicy struct {
    TimeoutSeconds  int  `json:"timeout_seconds,omitempty"`   // 全局超时(秒)
    MaxRetries      int  `json:"max_retries,omitempty"`       // 失败重试次数
    ContinueOnError bool `json:"continue_on_error,omitempty"` // 出错是否继续
    
    // 通知策略 (覆盖 Project 级配置)
    Notify *NotifyPolicy `json:"notify,omitempty"`
}

type NotifyPolicy struct {
    OnStart   bool `json:"on_start"`
    OnSuccess bool `json:"on_success"`
    OnFailure bool `json:"on_failure"`
}
```

## 模型关系图谱 (Mental Model)

```mermaid
graph TD
    Project[Project (调度/编排)] -->|1:N| ProjectWorkflow
    ProjectWorkflow -->|N:1| Workflow[Workflow (蓝图)]
    Workflow -->|1:N| ScanStage[ScanStage (原子任务)]
    
    ScanStage -->|Config| TargetProvider[TargetProvider]
    ScanStage -->|Output| StageResult[StageResult]
    
    subgraph "运行时 (Runtime)"
        ProjectExecution --> WorkflowExecution
        WorkflowExecution --> StageExecution
    end
```

## 设计哲学

1.  **职责单一**：Workflow 不管调度，只管执行逻辑。调度是 Project 的事。
2.  **无状态**：配置表与运行时状态表（Stats/Execution）严格分离。
3.  **策略收敛**：将复杂的控制逻辑收敛到 JSON 字段，保持 Schema 稳定。
