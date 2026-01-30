Pipeline 全流程编排方案

  ---
  一、核心理念

  Pipeline = 有向无环图（DAG）的扫描任务编排引擎

  典型扫描流程：
  IP Alive → Port Scan → Service Detect → OS Fingerprint → Web Scan → Vuln Scan
     ↓           ↓            ↓               ↓              ↓
    IPs     [IP:Port]   Service Info    OS Info       Web URLs

  ---
  二、设计方案

  方案 A：静态 Pipeline（推荐用于第一阶段）

  特点：预定义常用扫描流程，配置简单

  # configs/pipeline.yaml
  pipelines:
    full_scan:
      stages:
        - type: ip_alive_scan
          params:
            concurrency: 100
          timeout: 5m
        - type: port_scan
          depends_on: ip_alive_scan
          params:
            port_range: "1-65535"
            rate: 1000
          timeout: 10m
        - type: service_scan
          depends_on: port_scan
          params:
            service_detect: true
          timeout: 15m
        - type: os_scan
          depends_on: service_scan
          timeout: 10m
        - type: web_scan
          depends_on: service_scan
          filter: port==443 or port==80
          timeout: 20m
        - type: vuln_scan
          depends_on: web_scan
          timeout: 30m

    quick_scan:
      stages:
        - type: ip_alive_scan
        - type: port_scan
          params:
            port_range: "1-1000"

  数据流转：
  - 每个阶段接收上游的 TaskResult
  - 自动提取目标列表（如 Alive IPs, Open Ports）
  - 传递给下游 Scanner

  优势：
  - 配置即代码，易于理解和修改
  - 适合 80% 的常见扫描场景
  - 实现简单，约 500-800 行核心代码

  劣势：
  - 不够灵活，无法支持复杂条件分支
  - 需要 YAML 解析

  ---
  方案 B：动态 Pipeline Builder（推荐用于第二阶段）

  特点：程序化构建 Pipeline，支持条件分支

  // 使用示例
  pipeline := NewPipeline("full_scan")
  pipeline.
      Stage("alive", IpAliveScanner()).
      Stage("port", PortScanner(), DependentOn("alive")).
      Stage("service", ServiceScanner(), DependentOn("port")).
      Stage("os", OSScanner(), DependentOn("service")).
      Stage("web", WebScanner(), DependentOn("service")).
      If(func(results []*model.TaskResult) bool {
          // 条件：有开放的 Web 端口才执行
          return hasWebPorts(results)
      }).
      Stage("vuln", VulnScanner(), DependentOn("web")).
      Parallel("os", "web").  // OS 和 Web 可以并行执行
      Timeout(30 * time.Minute).
      Retry(3, 5*time.Minute)

  优势：
  - 类型安全，编译期检查
  - 支持复杂的条件逻辑
  - 易于测试

  劣势：
  - 学习曲线稍陡
  - 配置需要重新编译

  ---
  方案 C：基于任务的 Pipeline（最灵活）

  特点：每个阶段是一个独立的 Task，支持任意编排

  type Pipeline struct {
      Name     string
      Stages   []*Stage
      Graph    *DAG  // 有向无环图
  }

  type Stage struct {
      ID          string
      Scanner     scanner.Scanner
      InputFilter  func([]*model.TaskResult) []Target
      OutputMapper func(*model.TaskResult) []Target
      Timeout     time.Duration
      Retry       int
  }

  // 执行流程
  func (p *Pipeline) Run(ctx context.Context, initialTargets []string) ([]*model.TaskResult, error) {
      // 1. 构建 DAG 拓扑
      // 2. 并发执行无依赖的 Stage
      // 3. 收集结果，传递给下游
      // 4. 处理超时和重试
      // 5. 返回最终结果
  }

  优势：
  - 最灵活，支持任意复杂编排
  - 易于可视化（GraphViz）
  - 可扩展性强

  劣势：
  - 实现复杂度最高（约 1500-2000 行代码）
  - 需要处理环检测、死锁等问题

  ---
  三、推荐实施路径

  第一阶段：静态 Pipeline（2-3 周）

  // 实现核心接口
  type PipelineRunner interface {
      Run(ctx context.Context, name string, target string) ([]*model.TaskResult, error)
      ListPipelines() []string
      GetPipeline(name string) *PipelineConfig
  }

  // 配置结构
  type PipelineConfig struct {
      Name    string
      Stages  []StageConfig
  }

  type StageConfig struct {
      Type       model.TaskType
      DependsOn  []string
      Params     map[string]interface{}
      Filter     string  // 简单过滤表达式，如 "port==443"
      Timeout    time.Duration
  }

  CLI 命令：
  # 使用预定义 Pipeline
  ./neoAgent scan run -t 192.168.1.1 -p full_scan

  # 列出所有 Pipeline
  ./neoAgent pipeline list

  # 查看 Pipeline 详情
  ./neoAgent pipeline show full_scan

  ---
  第二阶段：动态 Builder（3-4 周）

  // API 示例
  pipeline := NewPipelineBuilder().
      WithName("custom").
      WithTimeout(30*time.Minute).
      AddStage("alive", alive.Scanner()).
      AddStage("port", port.Scanner()).
      DependsOn("port", "alive").
      Parallel("os", "web").
      Build()

  ---
  第三阶段：高级特性（持续优化）

  1. 条件分支：
  IfAlive("web", func() bool { return hasWebPorts() }).
      Then("vuln").
      Else("nothing")
  2. 结果聚合：
  Aggregate(func(results []*model.TaskResult) *model.TaskResult {
      // 合并多个 Scanner 的结果
  })
  3. 可视化：
  ./neoAgent pipeline visualize full_scan > pipeline.svg
  4. 断点续跑：
  ./neoAgent scan run -t 192.168.1.1 -p full_scan --checkpoint

  ---
  四、关键技术点

  1. 数据流转设计

  // 目标提取器
  type TargetExtractor interface {
      Extract(results []*model.TaskResult) []Target
  }

  // 示例：从 PortScan 结果提取目标
  type PortTargetExtractor struct{}

  func (e *PortTargetExtractor) Extract(results []*model.TaskResult) []Target {
      var targets []Target
      for _, r := range results {
          if portRes, ok := r.Result.(*model.PortServiceResult); ok && portRes.Status == "open" {
              targets = append(targets, Target{
                  IP:   portRes.IP,
                  Port:  portRes.Port,
                  Proto: portRes.Protocol,
              })
          }
      }
      return targets
  }

  2. Stage 依赖管理

  // 依赖图
  type StageDAG struct {
      stages map[string]*Stage
      edges  map[string][]string  // stage -> [depends_on...]
  }

  // 拓扑排序
  func (dag *StageDAG) TopologicalSort() ([]string, error) {
      // Kahn 算法
      // 1. 计算入度
      // 2. 找出入度为 0 的节点
      // 3. 移除边，重复步骤 2
      // 4. 检测环
  }

  3. 并发控制

  // Stage 并发执行
  func (p *Pipeline) RunStage(ctx context.Context, stage *Stage, inputs []*model.TaskResult) error {
      // 1. 提取目标
      targets := p.extractor.Extract(inputs)

      // 2. 创建任务
      tasks := p.createTasks(targets, stage)

      // 3. 并发执行（使用 RunnerManager）
      results, err := p.runner.RunParallel(ctx, tasks, stage.Rate)

      // 4. 缓存结果
      p.cacheResults(stage.ID, results)

      return nil
  }

  4. 错误处理和重试

  // 失败策略
  type FailurePolicy string

  const (
      StopOnError   FailurePolicy = "stop"     // 停止整个 Pipeline
      SkipOnError   FailurePolicy = "skip"     // 跳过该 Stage
      RetryOnError  FailurePolicy = "retry"    // 重试
      ContinueOnError FailurePolicy = "continue" // 继续执行
  )

  ---
  五、与 Master 集成

  // Adapter 模式：将 Master Task 转换为 Pipeline
  type MasterPipelineAdapter struct {
      pipelineRunner PipelineRunner
  }

  func (a *MasterPipelineAdapter) HandleTask(ctx context.Context, task *model.Task) (*model.TaskResult, error) {
      // 1. 从 Master Task 中提取 Pipeline 名称
      pipelineName := task.Params["pipeline"].(string)

      // 2. 运行 Pipeline
      results, err := a.pipelineRunner.Run(ctx, pipelineName, task.Target)

      // 3. 转换结果格式（符合 Master 契约）
      return a.convertResults(results), nil
  }

  ---
  六、推荐优先级
  ┌──────┬────────────────────────┬──────────┐
  │ 阶段 │          内容          │ 时间估算 │
  ├──────┼────────────────────────┼──────────┤
  │ P0   │ 静态 Pipeline 核心框架 │ 2 周     │
  ├──────┼────────────────────────┼──────────┤
  │ P1   │ YAML 配置加载和解析    │ 3 天     │
  ├──────┼────────────────────────┼──────────┤
  │ P1   │ 数据流转和目标提取     │ 3 天     │
  ├──────┼────────────────────────┼──────────┤
  │ P1   │ 依赖图和拓扑排序       │ 3 天     │
  ├──────┼────────────────────────┼──────────┤
  │ P2   │ 并发控制和超时管理     │ 3 天     │
  ├──────┼────────────────────────┼──────────┤
  │ P2   │ 错误处理和重试机制     │ 2 天     │
  ├──────┼────────────────────────┼──────────┤
  │ P3   │ CLI 命令集成           │ 2 天     │
  ├──────┼────────────────────────┼──────────┤
  │ P3   │ 与 Master 适配         │ 3 天     │
  └──────┴────────────────────────┴──────────┘
  ---
  七、参考实现思路

  核心文件结构：
  internal/core/pipeline/
  ├── pipeline.go           # Pipeline 核心接口
  ├── stage.go             # Stage 定义
  ├── dag.go               # 依赖图管理
  ├── runner.go            # Pipeline 执行器
  ├── extractor.go         # 目标提取器
  ├── config_loader.go     # YAML 配置加载
  └── presets.yaml         # 预定义 Pipeline 配置

  ---
  总结

  推荐方案：先实现静态 Pipeline，快速验证数据流转和依赖管理，再逐步扩展到动态 Builder。

  关键点：
  1. 明确 Stage 之间的数据契约
  2. 使用 DAG 管理依赖关系
  3. 并发执行无依赖的 Stage
  4. 支持失败策略和重试
  5. 预定义常用扫描流程

  下一步行动：
  1. 设计 Stage 接口和数据结构
  2. 实现 DAG 拓扑排序
  3. 实现 Pipeline 核心执行逻辑
  4. 编写 YAML 配置和 CLI 命令