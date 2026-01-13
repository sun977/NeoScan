# 资产与规则管理模块

#### 2.1.2 资产管理模块
- **资产同步**：支持从外部系统同步资产数据
- **资产清单**：维护完整的资产清单信息
- **数据传输**：支持扫描结果向外部系统传输

Asset : 资产管理、策略、漏洞、Web资产、Raw数据清洗。

---

## 3. 指纹规则管理 (Fingerprint Rule Management)

### 3.1 核心架构
本模块负责管理用于资产识别的指纹规则库，支持**混合源 (Hybrid Source)** 模式，即规则既可以来自数据库，也可以来自文件系统。最终统一通过文件系统快照分发给 Agent。

### 3.2 目录结构规范
Master 节点上的规则根目录为 `rules/fingerprint`，结构如下：

```text
rules/fingerprint/
├── system/                  # 系统自带规则 (System Rules)
│   ├── service/             # [Source: DB] Service 指纹 (由 DB 生成)
│   ├── cms/                 # [Source: DB] CMS 指纹 (由 DB 生成)
│   └── os/                  # [Source: File] OS 指纹 (直接文件上传)
│
└── custom/                  # 用户自定义规则 (Custom Rules)
    ├── service/             # [Source: File] 自定义 Service 指纹
    ├── cms/                 # [Source: File] 自定义 CMS 指纹
    └── os/                  # [Source: File] 自定义 OS 指纹
```

### 3.3 数据流与同步策略

#### 3.3.1 规则分类与来源
| 规则类型 | 子类型 | 来源 (Source of Truth) | 同步机制 (Synchronization) |
| :--- | :--- | :--- | :--- |
| **Service** | System | **Database** | Master 需实现 `DB -> JSON File` 的导出逻辑，定期或触发式写入 `system/service` 目录。 |
| | Custom | **File System** | 用户直接上传/管理文件，无需 DB 同步。 |
| **CMS** | System | **Database** | Master 需实现 `DB -> JSON File` 的导出逻辑，定期或触发式写入 `system/cms` 目录。 |
| | Custom | **File System** | 用户直接上传/管理文件，无需 DB 同步。 |
| **OS** | System | **File System** | 直接上传/管理文件，无需 DB 同步 (OS 指纹暂无 DB 表结构)。 |
| | Custom | **File System** | 用户直接上传/管理文件，无需 DB 同步。 |

#### 3.3.2 规则打包与分发
*   **Agent 视角**：Agent 不感知规则来源（DB 或文件），它只通过 `AgentUpdateService` 拉取 `rules/fingerprint` 目录下的所有文件快照。
*   **快照生成**：`AgentUpdateService` 在生成快照前，**不负责**数据同步。
*   **同步时机**：需确保在调用 `BuildSnapshot` 之前，DB 中的最新数据已经刷新到文件系统。

### 3.4 待实现功能 (TODO)
1.  **Rule Exporter (规则导出器)**:
    *   实现从 Service/CMS 指纹表查询数据并生成 JSON 文件的逻辑。
    *   支持全量导出和增量更新（可选）。
2.  **Rule Manager API**:
    *   提供上传接口，允许用户上传 Custom 规则文件。
    *   提供上传接口，允许管理员更新 System OS 规则文件。
3.  **Manual Sync Trigger (手动同步触发)**:
    *   提供 `POST /api/v1/rules/fingerprint/generate` 接口。
    *   **备份机制**: 在生成新文件前，自动将旧文件重命名为备份格式 (e.g., `default_service.json.backup.20260113.18204`)。
    *   **生成逻辑**: 从 DB 读取最新规则 -> 写入标准文件名 (e.g., `default_service.json`) -> 覆盖旧文件。
4.  **Rule Rollback (规则回滚)**:
    *   提供 `POST /api/v1/rules/fingerprint/rollback` 接口。
    *   **功能**: 允许将规则文件回滚到指定的备份版本。
    *   **机制**: 查找最近的（或指定的） `.backup` 文件 -> 覆盖当前标准文件。
