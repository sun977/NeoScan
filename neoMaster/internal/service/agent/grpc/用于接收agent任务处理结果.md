## 通信协议分离
- HTTP接口 ：用于master对agent的管理(注册，认证，删除等)和任务调度
- gRPC接口 ：主要用于agent的回传大量的扫描结果给master

## gRPC拦截器的作用范围
gRPC拦截器主要用于：

1. 验证Agent节点的合法性 - grpcToken
2. master端用于接收agent的任务扫描结果

两者使用不同的协议栈，各自有独立的认证机制
- 用户登录 → HTTP接口 + HTTP中间件
- Agent通信 → gRPC接口 + gRPC拦截器

## 两者分工
控制面 (HTTP)          数据面 (gRPC)
- Agent认证注册        - 扫描结果上报
- 任务下发             - 系统指标上报
- 配置下发             - 大量数据传输
- 状态查询             - 实时数据流

## 任务分发流程
Orchestrator -> HTTP API -> Agent (Agent的生命周期管理、任务下发)
Agent -> gRPC -> Orchestrator (仅限扫描结果上报)

## 将来扩展
1. 支持master对agent二进制文件的生成，生成的时候可以选择agent具备的功能模块
2. agent支持扫描数据 syslog 外发功能
