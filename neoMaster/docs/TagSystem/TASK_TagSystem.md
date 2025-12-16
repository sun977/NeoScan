# 任务拆分文档 - 标签系统 (Tag System)

## 任务列表

### 任务1: 基础 CRUD 接口与数据库模型
#### 输入契约
- 数据库连接
- 现有项目架构
#### 输出契约
- 数据库表结构 (sys_tags, sys_match_rules, sys_entity_tags)
- Repository 层代码
- Service 层基础 CRUD 代码
- Handler 层代码
#### 状态
- [x] 已完成

### 任务2: 自动打标逻辑 (AutoTag)
#### 输入契约
- 资产数据 (Host, Web, Network)
- 匹配规则
#### 输出契约
- Matcher 匹配库
- AutoTag 核心逻辑
#### 状态
- [x] 已完成

### 任务3: 规则传播与回溯 (Propagation & Backfill)
#### 输入契约
- LocalAgent (原 SystemWorker)
- 任务调度机制
#### 输出契约
- System Task 定义 (Payload)
- LocalAgent 标签传播执行逻辑 (Host, Web, Network)
- 规则变更时的任务触发机制
- sys_entity_tags 表的自动同步
#### 状态
- [x] 已完成

### 任务4: API 完善与集成
#### 输入契约
- OpenAPI 文档
- 前端需求
#### 输出契约
- 完整的 API 接口
- 单元测试
#### 状态
- [ ] 进行中
