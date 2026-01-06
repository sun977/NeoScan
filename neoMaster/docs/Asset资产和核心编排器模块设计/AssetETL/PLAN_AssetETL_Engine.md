# Asset ETL Engine 开发计划 (Phase 4: The Cleaner)

## 1. 核心目标
实现从 `StageResult` (原始扫描结果) 到 `Asset` (标准化资产) 的完整清洗、转换与入库流程。
**原则**: 先做数据处理 (ETL)，后做数据传输 (Communication)，以数据结构驱动通信协议设计。

## 2. 关键里程碑 (Milestones)

| 节点 | 任务名称 | 描述 | 状态 | 预期产出 |
| :--- | :--- | :--- | :--- | :--- |
| **M1** | **框架搭建与流程串联** | 搭建 Processor 主流程，定义数据流转接口。 | 🚧 **待开始** | `processor.go` 能够从 Queue 取数据并调通下游空接口。 |
| **M2** | **工具解析器 (Parser)** | 实现 Nmap/Masscan/HTTP 等工具的输出解析逻辑。 | ⚪ 未开始 | `converter.go` 能将 XML/JSON 字符串转为标准结构体。 |
| **M3** | **指纹标准化 (Fingerprint)** | 实现 CPE 统一映射与服务识别。 | ⚪ 未开始 | `fingerprint.go` 能将 "nginx/1.18" 转为 `cpe:/a:nginx:nginx:1.18`。 |
| **M4** | **资产合并 (Merger)** | **(最核心)** 实现 Host/Port/Web 的 Upsert (更新或插入) 逻辑。 | ⚪ 未开始 | `merger.go` 能正确处理资产的新增、更新和时间戳刷新。 |
| **M5** | **Web 数据处理** | 处理截图、HTML 等大体积非结构化数据。 | ⚪ 未开始 | `web_crawler.go` 能提取 Title 并归档截图。 |
| **M6** | **集成测试 (End-to-End)** | 构造 Mock 数据，验证全链路。 | ⚪ 未开始 | 单元测试通过，数据库中有正确数据。 |

## 3. 详细执行步骤 (Action Items)

### Step 1: 框架搭建 (Processor Loop)
*   **目标**: 让 `Processor` 动起来，跑通 "取数据 -> 转换 -> 处理 -> 存库" 的骨架。
*   **动作**:
    1.  完善 `processor.go`：实现 `Start()` 方法，启动 Goroutine 消费 `ResultQueue`。
    2.  定义 `Parser` 和 `Merger` 的接口签名。
    3.  实现简单的错误处理和日志记录。

### Step 2: 核心组件实现 (逐个击破)

#### 2.1 Converter (解析器)
*   **文件**: `converter.go`
*   **任务**:
    *   实现 `ParseNmapResult(output string)`: 解析 Nmap XML/JSON。
    *   实现 `ParseMasscanResult(output string)`: 解析 Masscan JSON。
    *   实现 `ParseHTTPResult(output string)`: 解析 HTTP 探针结果。
*   **难点**: 不同工具的输出格式差异大，需要统一为中间格式。

#### 2.2 FingerprintMatcher (指纹识别)
*   **文件**: `fingerprint.go`
*   **任务**:
    *   实现简单的正则匹配或字典匹配。
    *   生成标准 CPE 字符串。

#### 2.3 Merger (资产合并 - 重中之重)
*   **文件**: `merger.go`
*   **任务**:
    *   **Host**: 根据 IP 查找，存在则更新 `LastSeen`，不存在则 Insert。
    *   **Port**: 关联 HostID，检查端口状态变更（Open -> Closed?）。
    *   **Web**: 关联 PortID，更新 Web 指纹。
*   **注意**: 这里需要频繁与 DB 交互，要注意性能和事务处理。

### Step 3: 集成验证
*   **动作**:
    *   编写一个 Test Case，构造一个模拟的 Nmap `StageResult`。
    *   手动调用 `Processor.Process(mockResult)`。
    *   断言数据库中是否生成了对应的 `asset_hosts` 和 `asset_ports` 记录。
