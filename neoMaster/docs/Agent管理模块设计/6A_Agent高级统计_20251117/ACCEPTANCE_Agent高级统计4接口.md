# 验收记录 - Agent高级统计4接口

## 构建与接线
- 构建命令：`go build -o neoMaster.exe ./cmd/master` → 通过
- 路由接线：/agent/statistics|load-balance|performance|capacity → Handler 成功

## 单元测试
- 仅针对本功能的基础测试：`go test -v ./test/20251117`
- 说明：项目历史测试包含尚未对齐的接口签名，建议按目录运行本次新增测试

## 手工验证建议
- /agent/statistics?window_seconds=300 应返回在线/离线及分布统计
- /agent/statistics?group_id=G-001 应仅统计该分组成员
- /agent/load-balance?top_n=3 应返回 TopBusy/TopIdle 列表
- /agent/load-balance?group_id=G-001&top_n=3 仅统计该分组
- /agent/performance?top_n=5 应返回 Top CPU/Mem/Net/Failed 列表
- /agent/performance?group_id=G-001&top_n=5 仅统计该分组
- /agent/capacity?cpu_threshold=85&memory_threshold=85&disk_threshold=90 应返回过载明细与建议
- /agent/capacity?group_id=G-001&cpu_threshold=85 仅统计该分组

## 限制说明
- 单快照模型，不提供时序趋势；“趋势”相关诉求需要引入历史表或时序库