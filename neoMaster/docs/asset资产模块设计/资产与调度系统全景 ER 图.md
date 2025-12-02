# 资产与调度系统全景 ER 图

## 图例说明
- **[Table]** : 需要在数据库中建立物理表。
- **[Struct]** : 仅作为代码中的 Go Struct 或 JSON 对象，不需要建表。
- **实线** : 强关联 (Foreign Key)。
- **虚线** : 逻辑关联 (ID Reference)。

```mermaid
erDiagram
    %% ========================================================
    %% 1. 调度域 (Scheduling Domain) - Master 核心配置
    %% ========================================================
    Project ||--o{ ProjectWorkflow : "包含"
    Workflow ||--o{ ProjectWorkflow : "被引用"
    Workflow ||--o{ ScanStage : "定义"
    
    Project {
        uint id PK "[Table]"
        string name
        string status
        json notify_config
        json export_config
    }

    Workflow {
        uint id PK "[Table]"
        string name
        string exec_mode
        json policy_config
        json global_vars
    }

    ProjectWorkflow {
        uint project_id PK,FK "[Table] 关联表"
        uint workflow_id PK,FK
        int sort_order
    }

    ScanStage {
        uint id PK "[Table]"
        uint workflow_id FK
        string stage_type
        string tool_name
        json target_policy
        json output_config
    }

    TargetProvider {
        string source_type "[Struct] 逻辑实体"
        string target_type
        json filter_rules
    }
    ScanStage ..> TargetProvider : "包含(target_policy)"

    %% ========================================================
    %% 2. 执行域 (Execution Domain) - 任务与结果
    %% ========================================================
    ScanStage ||--o{ StageResult : "产生"
    WorkflowStats ||--|| Workflow : "统计"

    StageResult {
        uint id PK "[Table] 海量数据"
        uint stage_id FK
        uint workflow_id FK
        string result_type
        string target_value
        json attributes
        json evidence
    }

    WorkflowStats {
        uint workflow_id PK,FK "[Table] 热数据"
        int total_execs
        int success_execs
    }

    %% ========================================================
    %% 3. 资产域 (Asset Domain) - 结果沉淀
    %% ========================================================
    StageResult }|..|{ AssetHost : "ETL转换"
    
    AssetHost ||--o{ AssetService : "开放端口"
    AssetHost ||--o{ AssetWeb : "Web服务"
    AssetService ||--o{ AssetVuln : "存在漏洞"
    AssetWeb ||--o{ AssetVuln : "存在漏洞"

    AssetHost {
        uint id PK "[Table]"
        string ip
        string os
        json tags
        datetime last_seen_at
    }

    AssetService {
        uint id PK "[Table]"
        uint host_id FK
        int port
        string service_name
        string version
    }

    AssetWeb {
        uint id PK "[Table]"
        uint host_id FK
        string domain
        string url
        json tech_stack
    }

    AssetVuln {
        uint id PK "[Table]"
        string target_type
        uint target_ref_id
        string cve_id
        string severity
    }

    %% ========================================================
    %% 4. 外部导入域 (Import Domain) - 资产来源 B
    %% ========================================================
    RawAsset ||--o{ RawAssetNetwork : "规范化"
    RawAssetNetwork ||--o{ AssetNetwork : "拆分/实例化"
    AssetNetwork }|..|{ ScanStage : "作为扫描目标"

    RawAsset {
        uint id PK "[Table] 原始导入记录"
        string source_type
        json payload
    }

    RawAssetNetwork {
        uint id PK "[Table] 待确认网段"
        string cidr
        string status
    }

    AssetNetwork {
        uint id PK "[Table] 正式网段资产"
        string cidr
        string zone
    }

    AssetNetworkScan {
        uint id PK "[Table] 网段扫描记录"
        uint network_id FK
        uint agent_id
        string scan_status
        int round
        datetime started_at
        datetime finished_at
    }
    
    AssetNetwork ||--o{ AssetNetworkScan : "记录历史"

    %% ========================================================
    %% 5. 辅助域 (Auxiliary Domain)
    %% ========================================================
    AssetWhitelist {
        uint id PK "[Table]"
        string target_value
        string scope
    }
    
    AssetSkipPolicy {
        uint id PK "[Table]"
        string condition
        string action
    }
```

## 实体类型清单

### 1. 必须建表 (Physical Tables)
这些实体承载核心业务数据，必须持久化。

- **Project**: 项目主表。
- **Workflow**: 工作流定义表。
- **ProjectWorkflow**: 项目与工作流的多对多关联表。
- **ScanStage**: 扫描阶段定义表。
- **StageResult**: 扫描结果表（日志型，数据量大）。
- **WorkflowStats**: 运行时统计表（读写分离优化）。
- **AssetHost / Service / Web / Vuln**: 最终资产表（业务核心）。
- **RawAsset / RawAssetNetwork / AssetNetwork**: 外部导入与网段管理表。
- **AssetNetworkScan**: 网段扫描历史记录表。
- **AssetWhitelist / SkipPolicy**: 全局策略表。

### 2. 逻辑实体 (Logical Structs)
这些实体仅存在于代码逻辑或 JSON 字段中，**不需要**单独建表。

- **TargetProvider**: 实际上是 `ScanStage.target_policy` 字段解析后的逻辑对象。
- **WorkflowPolicy**: 实际上是 `Workflow.policy_config` 字段对应的 Go Struct。
- **NotifyConfig**: 实际上是 `Project.notify_config` 字段对应的 Go Struct。

## 数据流转视图

1.  **配置流**：用户创建 `Project` -> 关联 `Workflow` -> 定义 `ScanStage`。
2.  **执行流**：Master 读取 `ScanStage` -> 解析 `TargetProvider` -> 下发任务 -> Agent 返回 `StageResult`。
3.  **数据流**：Master 收到 `StageResult` -> (根据 OutputConfig) -> 清洗并 Upsert 到 `AssetHost/Service/...` 表。
