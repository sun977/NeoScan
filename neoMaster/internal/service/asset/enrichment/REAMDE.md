# 资产丰富化服务 (Asset Enrichment Service)

本模块负责对已入库的基础资产数据进行"后处理"（Post-Processing），通过指纹识别、标签推导等手段，丰富资产的属性和上下文信息，提升资产价值。(目前借助指纹识别服务 (Fingerprint Service) 提供资产丰富功能)

## 核心职责

1.  **指纹再识别 (Re-identification)**: 
    - 针对 `AssetService` 中有 Banner 但无 Product/Version 的服务，调用指纹库进行二次识别。
    - 解决扫描工具（如 Masscan）识别能力弱的问题。

2.  **自动打标签 (Auto-Tagging)**:
    - 基于识别出的属性（如 `Product: Nginx`），自动推导并应用业务标签（如 `Tag: WebServer`）。
    - 确保资产分类的自动化和一致性。

## 架构决策：标签推导的归属

在 NeoScan 系统中，关于"自动打标签"这一动作，存在三个潜在的执行者。为了保持架构清晰，我们对其职责边界进行了明确划分：

| 组件 | 角色 | 适用场景 | 交互模式 |
| :--- | :--- | :--- | :--- |
| **Asset Enrichment**<br>(本模块) | **实时/流式执行者** | **增量数据的即时处理**。<br>例如：扫描任务结束 -> 识别出新指纹 -> 立即推导标签。 | 直接调用 TagService 接口，同步执行。 |
| **Local Agent**<br>(Orchestrator) | **批量/全量执行者** | **规则变更后的存量清洗**。<br>例如：用户修改了打标规则 -> 需要对 100万 存量资产重新计算标签。 | 消费 System Task，异步批量执行 (Batch Job)。 |
| **Tag System**<br>(Service Layer) | **逻辑承载者** | **提供能力，不直接触发**。<br>负责管理规则、提供匹配逻辑和 CRUD 接口。 | 被上层调用。 |

### 为什么 Enrichment 负责实时打标？

1.  **时效性**: 指纹识别刚完成时，上下文（Product, Version, Banner）是最鲜活的。此时立即打标效率最高，无需等待后台任务轮询。
2.  **避免惊群**: 扫描任务是持续流式的，如果每个资产都生成一个系统任务给 Local Agent，会导致任务队列爆炸。Local Agent 更适合处理"一次性、大规模"的维护任务。
3.  **闭环**: "识别 -> 属性更新 -> 标签更新" 是一个完整的丰富化闭环。

## 模块结构

- `FingerprintMatcher`: 核心逻辑组件。
  - 1. 从数据库拉取待识别服务。
  - 2. 调用 `FingerprintService` 进行识别。
  - 3. 更新资产属性 (`Product`, `Version`)。
  - 4. **(Key Step)** 调用 `TagService.AutoTag` 进行标签联动。
