# 待办事项 - [NeoAgent 核心指令集建设]

## 紧急待办
1. **Runner 接口定义**：定义统一的 `TaskRunner` 接口，用于执行 `model.Task`。
2. **Proxy 核心实现**：实现 Socks5/HTTP 代理服务器逻辑。
3. **扫描器集成**：集成 Masscan/Nmap/Nuclei 等工具的执行逻辑。

## 配置项
- 检查 `configs/config.yaml`，确保包含 Proxy 和 Scan 相关的默认配置（如默认字典路径、默认超时时间等）。

## 建议
- 下一个任务建议为：**Phase 3: 原生能力建设 - 实现基础并发Runner**。
