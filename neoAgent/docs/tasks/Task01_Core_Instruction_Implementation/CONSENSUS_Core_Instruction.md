# 共识文档 - [NeoAgent 核心指令集建设]

## 最终需求描述
构建 NeoAgent 的核心指令集架构，确保 CLI 和 Cluster 模式共享同一套任务模型和参数规范。实现对扫描能力（Asset, Port, Web, Dir, Vuln, Subdomain）和代理能力（Proxy）的统一抽象。

## 技术实现方案
### 1. 统一任务模型
- **位置**：`internal/core/model/task.go`
- **策略**：
    - 使用 `TaskType` 区分原子能力。
    - 使用 `Params` (map[string]interface{}) 存储特定指令的参数。
    - 约定 `Timeout <= 0` 代表长运行任务（如 Proxy）。

### 2. CLI 架构 (Cobra)
- **Root** (`neoAgent`)
    - **Scan** (`neoAgent scan`)
        - `asset`: 资产扫描
        - `port`: 端口扫描
        - `web`: Web 综合扫描
        - `dir`: 目录扫描
        - `vuln`: 漏洞扫描
        - `subdomain`: 子域名扫描
    - **Proxy** (`neoAgent proxy`)
        - 支持 `--mode` (socks5, http, port_forward)
    - **Service** (`neoAgent service`) - *预留给 Master 模式*

### 3. 参数解析层
- 实现 `Options` 结构体绑定 CLI Flags。
- 实现 `Options.ToTask()` 方法将 Flags 转换为 `model.Task`。
- 实现 `model.Task.Validate()` 方法进行统一校验。

## 任务边界
- **In Scope**:
    - Cobra 命令定义 (Command 结构)。
    - Flag 定义与绑定。
    - Task 模型转换逻辑。
    - Proxy 指令定义。
- **Out of Scope**:
    - 实际的网络扫描逻辑 (TCP/UDP 连接等)。
    - 实际的 Proxy 转发逻辑。

## 验收标准
1. `neoAgent scan [type] --help` 显示正确的帮助信息和参数。
2. `neoAgent proxy --help` 显示正确的代理参数。
3. 能够通过 CLI 构造出合法的 `model.Task` 对象并打印（作为 Debug 验证）。
4. 代码结构清晰，符合 `cmd -> options -> model` 的数据流。
