# 需求对齐文档 - Orchestrator_API_Implementation

## 原始需求
完成 Asset资产和核心编排器模块设计 中的 Orchestrator（编排器）部分。
目前 Backend (Model/Repo/Service) 已实现，但缺少 Frontend API (Handler/Router)。
需要实现一套 RESTful API 来管理扫描项目、工作流、扫描阶段和工具模板。

## 项目上下文
### 技术栈
- 编程语言：Go 1.20+
- Web框架：Gin
- 数据库：MySQL (GORM)
- 依赖注入：手动注入 (Setup层)

### 现有架构理解
- **Models**: `internal/model/orchestrator` (Project, Workflow, ScanStage, ScanToolTemplate) 已定义。
- **Services**: `internal/service/orchestrator` 逻辑已实现。
- **Routers**: `internal/app/master/router/orchestrator_routers.go` 存在但为空。
- **Handlers**: `internal/handler/orchestrator` 目录不存在 (仅有 `orchestrator_drop`)。

## 需求理解
### 功能边界
**包含功能：**
1.  **Project Management**:
    -   CRUD 操作
    -   状态变更 (Enable/Disable)
    -   关联 Workflows (Add/Remove/Reorder)
2.  **Workflow Management**:
    -   CRUD 操作
    -   关联 ScanStages
3.  **ScanStage Management**:
    -   CRUD 操作
    -   定义阶段行为 (关联 Tool, 参数覆写)
4.  **ScanToolTemplate Management**:
    -   CRUD 操作 (管理 Nmap, Masscan 等基础模板)
5.  **Router Registration**:
    -   在 `orchestrator_routers.go` 中注册上述路由。
6.  **Dependency Injection**:
    -   在 `setup` 包中初始化所有 Handler 并注入 Router。

**明确不包含（Out of Scope）：**
-   **实际调度逻辑**: 本次任务只负责**配置管理 API**。真正的任务调度（Cron/立即执行）和 Agent 分发逻辑属于后续 "Dispatch" 任务，虽然 Project 字段里有 Cron，但本次只负责存取，不负责触发。
-   **Agent 交互**: 不包含与 Agent 的 GRPC/HTTP 通信。

## 疑问澄清
### P0级问题
无明显歧义。现有 Service 层签名清晰，直接透传调用即可。

## 验收标准
### 功能验收
- [ ] **Project**: 能创建包含 Cron 表达式和 JSON 配置的项目。
- [ ] **Workflow**: 能创建工作流并正确关联到 Project。
- [ ] **Stage**: 能定义扫描阶段并关联到 Workflow。
- [ ] **API**: 所有接口返回统一的 `system.APIResponse` 格式。
- [ ] **Validation**: 输入参数缺失或非法时返回 400。

### 质量验收
- [ ] 单元测试覆盖率 > 80% (针对 Router/Handler 层)。
- [ ] 遵循 `internal/pkg/logger` 日志规范。
- [ ] 代码无 Linter 错误。
