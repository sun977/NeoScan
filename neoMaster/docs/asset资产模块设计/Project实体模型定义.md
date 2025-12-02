# Project 实体模型定义

## 概述

`Project` 是 NeoScan 扫描任务的顶层容器。它定义了"做什么"（关联的工作流）、"什么时候做"（调度策略）以及"做完告诉谁"（通知配置）。

**核心职责**：
1.  **编排**：管理一组 Workflow 的执行顺序（串行/并行）。
2.  **调度**：控制任务的触发时机（Cron/立即/API）。
3.  **生命周期**：维护整个扫描任务的运行状态。

## 模型结构

### 核心字段

| 字段名 | 类型 | 描述 | 索引 |
|--------|------|------|------|
| `id` | uint | 自增主键 | PK |
| `name` | string | 项目唯一标识名（英文） | Unique |
| `display_name` | string | 显示名称（中文友好） | |
| `description` | text | 项目描述 | |
| `status` | string | 当前运行状态 (idle/running/paused/finished/error) | Index |
| `enabled` | bool | 是否启用（总开关） | |
| `schedule_type` | string | 调度类型 (immediate/cron/api/event) | |
| `cron_expr` | string | Cron 表达式 (仅 schedule_type=cron 时有效) | |
| `exec_mode` | string | 工作流执行模式 (sequential/parallel) | |
| `notify_config` | JSON | 通知配置聚合 (收敛所有通知相关字段) | |
| `export_config` | JSON | 结果导出配置 (收敛所有导出相关字段) | |
| `extended_data` | JSON | 扩展数据 (用于存储非结构化元数据) | |
| `tags` | JSON | 标签列表 (用于前端筛选和权限分组) | |
| `last_exec_time` | timestamp | 最后一次执行开始时间 | |
| `last_exec_id` | string | 最后一次执行的任务 ID | |
| `created_by` | uint | 创建者 UserID | |
| `updated_by` | uint | 更新者 UserID | |
| `created_at` | timestamp | 创建时间 | |
| `updated_at` | timestamp | 更新时间 | |
| `deleted_at` | timestamp | 软删除时间 | Index |

### 关联关系

- **HasMany Workflows**: 项目与工作流是一对多关系（或多对多，取决于复用需求）。
    - 建议通过 `ProjectWorkflow` 中间表关联，以支持定义工作流在项目中的执行顺序 `order`。

## 字段详解

### 1. notify_config (通知配置)
*Linus Note: 不要把 email, webhook 摊平在主表里。用 JSON 收敛它们，保持主表整洁。*

```json
{
  "on_success": true,
  "on_failure": true,
  "channels": [
    {
      "type": "email",
      "recipients": ["admin@example.com", "sec@example.com"]
    },
    {
      "type": "webhook",
      "url": "https://hooks.slack.com/services/xxx",
      "secret": "********"
    },
    {
      "type": "lark", // 蓝信/飞书
      "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
    }
  ],
  "template_id": "tpl_daily_report" // 可选：引用通知模板
}
```

### 2. export_config (导出配置)
*Linus Note: 导出逻辑同理，收敛配置。*

```json
{
  "auto_export": true,
  "formats": ["pdf", "excel", "json"],
  "storage": {
    "type": "local", // 或 s3, oss
    "path": "/var/data/reports/{project_name}/{date}/"
  },
  "retention_days": 30
}
```

### 3. exec_mode (执行模式)
定义该项目下关联的多个 Workflow 如何执行：
- `sequential`: **串行执行**。Workflow A 完成后 -> Workflow B。适用于有依赖关系的流程。
- `parallel`: **并行执行**。Workflow A 和 B 同时启动。适用于互不干扰的独立任务（如：同时扫两个不同的网段）。

### 4. status (运行状态)
区分配置状态 (`enabled`) 和 运行时状态 (`status`)：
- `idle`: 空闲，等待调度。
- `queued`: 已进入队列，等待资源。
- `running`: 正在执行中。
- `paused`: 用户手动暂停。
- `error`: 上次执行异常退出。

## 数据库设计 (GORM 示例)

```go
type Project struct {
    gorm.Model
    Name           string         `gorm:"type:varchar(100);uniqueIndex;not null;comment:项目唯一标识"`
    DisplayName    string         `gorm:"type:varchar(200);comment:显示名称"`
    Description    string         `gorm:"type:text;comment:描述"`
    
    // 状态控制
    Status         string         `gorm:"type:varchar(20);default:'idle';index;comment:运行状态"`
    Enabled        bool           `gorm:"default:true;comment:启用开关"`
    
    // 调度配置
    ScheduleType   string         `gorm:"type:varchar(20);default:'immediate';comment:调度类型"`
    CronExpr       string         `gorm:"type:varchar(100);comment:Cron表达式"`
    ExecMode       string         `gorm:"type:varchar(20);default:'sequential';comment:工作流执行模式"`
    
    // JSON 配置块
    NotifyConfig   datatypes.JSON `gorm:"type:json;comment:通知配置"`
    ExportConfig   datatypes.JSON `gorm:"type:json;comment:导出配置"`
    ExtendedData   datatypes.JSON `gorm:"type:json;comment:扩展数据"`
    Tags           datatypes.JSON `gorm:"type:json;comment:标签"`
    
    // 审计信息
    LastExecTime   *time.Time     `gorm:"comment:最后执行时间"`
    LastExecID     string         `gorm:"type:varchar(64);comment:最后执行ID"`
    CreatedBy      uint           `gorm:"index;comment:创建人ID"`
    UpdatedBy      uint           `gorm:"index;comment:更新人ID"`
    
    // 关联
    Workflows      []Workflow     `gorm:"many2many:project_workflows;"`
}

// ProjectWorkflow 关联表，用于定义顺序
type ProjectWorkflow struct {
    ProjectID  uint `gorm:"primaryKey"`
    WorkflowID uint `gorm:"primaryKey"`
    SortOrder  int  `gorm:"default:0;comment:执行顺序"`
    CreatedAt  time.Time
}
```

## 设计哲学

1.  **配置收敛**：将易变的、结构复杂的配置（通知、导出）用 JSON 存储，避免频繁变更表结构。
2.  **状态分离**：明确区分"静态配置开关" (`enabled`) 和 "动态运行状态" (`status`)。
3.  **关联解耦**：通过 `many2many` 关联工作流，复用 Workflow 定义，减少重复配置。
