# 需求对齐文档 - Scheduler_Implementation

## 原始需求
在完成 Orchestrator API（项目/工作流/阶段管理）后，接下来的核心任务是实现 **调度引擎 (Scheduler Engine)**，负责将静态的配置（Project/Workflow）转化为动态的执行任务（ScanStage -> AgentTask），并管理 Master 与 Agent 之间的交互。

## 项目上下文
### 现有架构
- **Master-Agent 模型**: "Thin Agent, Smart Master"。Agent 无状态，只执行单一任务；Master 负责状态流转和调度。
- **Orchestrator API**: 已完成，提供 Project, Workflow, ScanStage 的 CRUD。
- **Agent Service**: `internal/service/agent/task.go` 存在但方法均未实现（TODO）。
- **Database**: MySQL (GORM)。

### 核心文档参考
- `docs/Asset资产和核心编排器模块设计/Master-Agent交互与多阶段调度.md`

## 需求理解
### 核心功能
1. **任务调度 (Scheduler Loop)**:
   - 监控 `Project` 状态 (Idle -> Running)。
   - 生成/获取当前待执行的 `ScanStage`。
   - 将 `ScanStage` 转换为 `AgentTask`。
   - 选择合适的 Agent 并分发任务。
2. **任务分发 (Dispatcher)**:
   - 实现 `AgentTaskService.AssignTask`。
   - 通过 gRPC/HTTP (需确认 Agent 端协议) 下发任务。
3. **结果处理 (Result Processor)**:
   - 接收 Agent 返回的 `StageResult`。
   - 解析结果并入库。
   - 根据 `target_policy` 和 `output_config` 生成下一阶段的任务输入。
4. **状态机 (State Machine)**:
   - 管理 Project (Running/Paused/Finished) 和 Workflow (Stage 1 -> Stage 2) 的状态流转。

### 疑问澄清
#### P0级问题
1. **Master-Agent 通讯协议**:
   - Agent 端目前暴露的是 gRPC 还是 HTTP 接口？
   - `internal/service/agent/grpc` 目录存在，暗示使用 gRPC。需确认 proto 定义。
2. **Agent 选址策略**:
   - 如何选择 Agent？(随机？轮询？基于负载？基于标签？)
   - 初步建议：基于 **Tags** 匹配 + **随机/轮询**。

## 验收标准
### 功能验收
- [ ] 能启动一个 Project，状态变为 Running。
- [ ] Scheduler 能自动识别第一个 ScanStage 并生成 Task。
- [ ] 能成功调用 Agent 接口下发任务。
- [ ] 能接收并处理 Agent 的模拟结果。
- [ ] 能根据结果触发下一阶段任务。

### 质量验收
- [ ] 单元测试覆盖核心调度逻辑。
- [ ] 状态机流转无死锁。
- [ ] 异常处理（Agent 离线、任务超时）有基本机制。
