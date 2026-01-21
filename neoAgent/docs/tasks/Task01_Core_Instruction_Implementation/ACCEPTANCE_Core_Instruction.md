# 验收文档 - [NeoAgent 核心指令集建设]

## 执行结果
### 功能完成度
- [x] **指令集对齐**：所有原子能力均已定义。
- [x] **模型更新**：`model.TaskType` 支持 Asset, Port, Web, Dir, Vuln, Subdomain, Proxy。
- [x] **CLI 框架**：基于 Cobra 的多级命令结构已就绪。
- [x] **参数解析**：实现了 Options 模式，支持 Flag 绑定、校验和模型转换。
- [x] **Proxy 指令**：支持 mode, listen, auth, forward 参数。
- [x] **Scan 指令**：支持 6 种扫描类型的子命令。

### 验证记录
1. **编译测试**：`go build` 成功。
2. **Help 验证**：
   - `neoAgent --help`: 显示 proxy, scan, server 命令。
   - `neoAgent proxy --help`: 显示代理参数。
   - `neoAgent scan --help`: 显示 asset, port, web 等子命令。

## 质量评估
- **代码规范**：遵循 Go 项目结构，分离了 cmd 和 internal 逻辑。
- **扩展性**：新增扫描类型只需添加 Options 和 Command，无需修改核心逻辑。
- **一致性**：CLI 参数与 Cluster JSON 协议通过 `model.Task` 统一。

## 遗留问题 / 后续计划 (TODO)
1. **Runner 实现**：目前命令只是打印 JSON，需要实现真正的扫描和代理逻辑 (Phase 3)。
2. **Web Server 模式**：`neoAgent server` 目前只是简单的启动逻辑，需要确保其能处理来自 Master 的这些新类型任务。
