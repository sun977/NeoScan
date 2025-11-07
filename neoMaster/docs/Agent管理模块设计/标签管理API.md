# 标签管理 API 说明

本文档描述 Master 侧针对 Agent 标签的管理接口，包括获取、添加、删除与整体覆盖更新。所有接口均遵循项目既定的处理流程与日志规范（internal/pkg/logger），并在 Handler 层进行参数校验与统一响应封装（internal/model/system.APIResponse）。

## 路由与方法

- GET    /api/v1/agents/:id/tags
  - 功能：获取指定 Agent 的标签列表
  - 成功响应：200，`{"code":200,"status":"success","message":"...","data":{"tags":["web","db"]}}`
  - 失败响应：404/500，`{"code":404|500,"status":"failed","message":"...","error":"..."}`

- POST   /api/v1/agents/:id/tags
  - 功能：为指定 Agent 添加单个标签
  - 请求体：`{"tag":"web"}`
  - 成功响应：200，`{"code":200,"status":"success","message":"..."}`
  - 失败响应：400/500，`{"code":400|500,"status":"failed","message":"...","error":"..."}`

- DELETE /api/v1/agents/:id/tags
  - 功能：从指定 Agent 删除单个标签
  - 请求体：`{"tag":"web"}`
  - 成功响应：200，`{"code":200,"status":"success","message":"..."}`
  - 失败响应：400/500，`{"code":400|500,"status":"failed","message":"...","error":"..."}`

- PUT    /api/v1/agents/:id/tags
  - 功能：原子性地覆盖更新指定 Agent 的标签列表（差异计算后执行 AddTag/RemoveTag）
  - 请求体：`{"tags":["db","cache"]}`
  - 成功响应：200，返回旧/新标签，示例：
    ```json
    {
      "code": 200,
      "status": "success",
      "message": "Agent tags updated successfully",
      "data": {
        "agent_id": "agent-1",
        "old_tags": ["web", "db"],
        "new_tags": ["db", "cache"]
      }
    }
    ```
  - 失败响应：400/500，`{"code":400|500,"status":"failed","message":"...","error":"..."}`

## 处理流程说明（PUT /agents/:id/tags）

1. 路径参数校验：从 `:id` 获取 agentID，缺失则返回 400。
2. 请求体绑定：绑定 `{"tags": ["..."]}`，缺失或格式错误返回 400。
3. 调用服务层：`AgentManagerService.UpdateAgentTags(agentID, tags)`，服务层内部：
   - 获取当前标签 `oldTags`；
   - 计算差异：`toAdd = new - old`，`toRemove = old - new`；
   - 依次调用仓储层 `AddTag/RemoveTag` 执行更新；
4. 日志记录：按规范填充 `operation/option/func_name/agent_id/old_tags/new_tags` 等关键字段；
5. 统一响应：返回 `agent_id/old_tags/new_tags`。

## 设计与实现对齐

- 框架：Gin
- 依赖管理：Go Modules
- ORM：GORM（MySQL 驱动）
- 日志：internal/pkg/logger
- 层级调用关系：Handler → Service → Repository → Database
- 响应封装：internal/model/system.APIResponse

## 测试说明

- 测试文件位置：`neoMaster/test/20251107/20251107_Agent_UpdateAgentTags_test.go`
- 覆盖用例：
  - 成功更新（返回旧/新标签）
  - 缺少 agentID（返回 400）
  - 无效 JSON（返回 400）

## 注意事项

- 请求头需设置 `Content-Type: application/json`。
- 标签列表会在服务层进行去重（保持输入顺序），空字符串将被忽略。
- 后续可在服务层增加标签合法性校验（IsValidTagByName/IsValidTagId），与仓储层规则对齐。