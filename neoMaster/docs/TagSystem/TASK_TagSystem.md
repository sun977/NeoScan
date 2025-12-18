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

### 任务4: 性能优化与缓存 (Performance & Cache)
#### 输入契约
- 现有 AutoTag 逻辑
- 性能瓶颈分析 (JSON 解析)
#### 输出契约
- MatchRuleCache 内存缓存
- 规则预解析逻辑
- 自动刷新机制 (ReloadMatchRules)
- 日志规范化 (Logger)
#### 状态
- [x] 已完成

### 任务5: 层级标签体系 (Hierarchical Tags)
#### 输入契约
- 标签层级需求 (Path/Tree)
- 查询需求 (子树查询)
#### 输出契约
- Materialized Path 方案实现
- 路径计算与维护
- 级联删除 (Cascading Delete)
- 子标签包含查询优化
#### 状态
- [x] 已完成

### 任务6: Agent 集成与测试 (Agent Integration)
#### 输入契约
- AgentRepository 接口
- 现有测试用例
#### 输出契约
- 接口对齐 (Mock 修复)
- 集成测试通过 (20251217_Agent_TaskSupport_test.go)
- API 完善 (DeleteTag 参数调整)
#### 状态
- [x] 已完成
