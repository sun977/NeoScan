/**
 * Agent 分组管理控制器(基础管理 - 分组管理)
 * 作者: Sun977
 * 日期: 2025-11-07
 * 说明: 分组管理相关的 Handler 方法集中于此，包括：
 * - GetAgentGroupList（获取分组列表）
 * - CreateAgentGroup（创建分组）
 * - UpdateAgentGroup（更新分组）
 * - DeleteAgentGroup（删除分组）
 * - AddAgentToGroup（将Agent添加到分组）
 * - RemoveAgentFromGroup（从分组中移除Agent）
 * - GetAgentsInGroup （获取分组中的Agent列表）
 * - SetAgentGroupStatus （设置分组状态 - 激活 禁用分组）
 * 重构策略: 保持原有业务逻辑和返回格式不变，统一成功日志使用 LogBusinessOperation。
 */

// agentManageGroup.GET("/groups", r.agentGetGroupsPlaceholder)                        // ✅ 获取Agent分组列表 [Master端查询分组表]
// agentManageGroup.POST("/groups", r.agentCreateGroupPlaceholder)                     // ✅ 创建Agent分组 [Master端创建分组记录]
// agentManageGroup.PUT("/groups/:group_id", r.agentUpdateGroupPlaceholder)            // ✅ 更新Agent分组 [Master端更新分组信息]
// agentManageGroup.DELETE("/groups/:group_id", r.agentDeleteGroupPlaceholder)         // ✅ 删除Agent分组 [Master端删除分组及关联]
// agentManageGroup.POST("/:id/groups", r.agentAddToGroupPlaceholder)                  // ✅ 将Agent添加到分组 [Master端更新Agent分组关系]
// agentManageGroup.DELETE("/:id/groups/:group_id", r.agentRemoveFromGroupPlaceholder) // ✅ 从分组中移除Agent [Master端删除分组关系]
package agent

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
)

// CreateAgentGroup 创建分组
func (h *AgentHandler) CreateAgentGroup(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c) // 调用 utils 工具包直接从Gin上下文提取当前用户ID，如果不存在则返回0
	xRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	method := c.Request.Method

	var req agentModel.AgentGroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 记录参数绑定失败，同时输出当前用户ID用于审计
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "create_agent_group",
			"option":    "bind_json",
			"func_name": "handler.agent.CreateAgentGroup",
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "请求体解析失败",
			Error:   err.Error(),
		})
		return
	}

	resp, err := h.agentManagerService.CreateAgentGroup(&req)
	if err != nil {
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation":  "create_agent_group",
			"option":     "service_call.CreateAgentGroup",
			"func_name":  "handler.agent.CreateAgentGroup",
			"group_id":   req.GroupID,
			"group_name": req.Name,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "创建分组失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "ok",
		Data:    resp,
	})

	logger.LogBusinessOperation(
		"create_agent_group",
		currentUserID,
		"",
		clientIP,
		xRequestID,
		"success",
		"Agent分组创建成功",
		map[string]interface{}{
			"func_name":  "handler.agent.CreateAgentGroup",
			"option":     "result.success",
			"path":       pathUrl,
			"group_id":   resp.GroupID,
			"group_name": resp.Name,
		},
	)
}

// UpdateAgentGroup 更新分组信息
func (h *AgentHandler) UpdateAgentGroup(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c) // 调用 utils 工具包直接从Gin上下文提取当前用户ID，如果不存在则返回0
	xRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	method := c.Request.Method
	groupID := c.Param("group_id")

	var req agentModel.AgentGroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "update_agent_group",
			"option":    "bind_json",
			"func_name": "handler.agent.UpdateAgentGroup",
			"group_id":  groupID,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "请求体解析失败",
			Error:   err.Error(),
		})
		return
	}

	resp, err := h.agentManagerService.UpdateAgentGroup(groupID, &req)
	if err != nil {
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation":  "update_agent_group",
			"option":     "service_call.UpdateAgentGroup",
			"func_name":  "handler.agent.UpdateAgentGroup",
			"group_id":   groupID,
			"group_name": req.Name,
			"path":       pathUrl,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "更新分组失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "更新成功",
		Data:    resp,
	})

	logger.LogBusinessOperation(
		"update_agent_group",
		currentUserID,
		"",
		clientIP,
		xRequestID,
		"success",
		"Agent分组更新成功",
		map[string]interface{}{
			"func_name":  "handler.agent.UpdateAgentGroup",
			"option":     "result.success",
			"group_id":   groupID,
			"group_name": req.Name,
			"path":       pathUrl,
		},
	)
}

// DeleteAgentGroup 删除分组
func (h *AgentHandler) DeleteAgentGroup(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c) // 调用 utils 工具包直接从Gin上下文提取当前用户ID，如果不存在则返回0
	xRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	method := c.Request.Method
	groupID := c.Param("group_id")

	// 调用service层删除分组 - 服务层默认把该分组下所有成员迁移至默认分组
	if err := h.agentManagerService.DeleteAgentGroup(groupID); err != nil {
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "delete_agent_group",
			"option":    "service_call.DeleteAgentGroup",
			"func_name": "handler.agent.DeleteAgentGroup",
			"group_id":  groupID,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "删除分组失败",
			Error:   err.Error(),
			Data:    map[string]interface{}{"group_id": groupID},
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "删除分组成功,组成员已迁移至默认分组",
		Data:    map[string]interface{}{"group_id": groupID},
	})

	logger.LogBusinessOperation(
		"delete_agent_group",
		currentUserID,
		"",
		clientIP,
		xRequestID,
		"success",
		"Agent分组删除成功",
		map[string]interface{}{
			"func_name": "handler.agent.DeleteAgentGroup",
			"option":    "result.success",
			"group_id":  groupID,
			"path":      pathUrl,
		},
	)
}

// AddAgentToGroup 将Agent添加到分组
func (h *AgentHandler) AddAgentToGroup(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c) // 调用 utils 工具包直接从Gin上下文提取当前用户ID，如果不存在则返回0
	xRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	method := c.Request.Method
	agentID := c.Param("id")

	// 分组ID在请求体中
	var body struct {
		GroupID string `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.GroupID == "" {
		err := fmt.Errorf("group_id不能为空")
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "add_agent_to_group",
			"option":    "bind_json",
			"func_name": "handler.agent.AddAgentToGroup",
			"agent_id":  agentID,
			"group_id":  body.GroupID,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "请求体解析失败或缺少group_id",
			Error:   err.Error(),
		})
		return
	}

	req := agentModel.AgentGroupMemberRequest{AgentID: agentID, GroupID: body.GroupID}
	if err := h.agentManagerService.AddAgentToGroup(&req); err != nil {
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "add_agent_to_group",
			"option":    "service_call.AddAgentToGroup",
			"func_name": "handler.agent.AddAgentToGroup",
			"agent_id":  agentID,
			"group_id":  body.GroupID,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "添加Agent到分组失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "添加Agent到分组成功",
		Data: map[string]interface{}{
			"agent_id": agentID,
			"group_id": body.GroupID,
		},
	})

	logger.LogBusinessOperation(
		"add_agent_to_group",
		currentUserID,
		"",
		clientIP,
		xRequestID,
		"success",
		"Agent添加到分组成功",
		map[string]interface{}{
			"func_name": "handler.agent.AddAgentToGroup",
			"option":    "result.success",
			"group_id":  body.GroupID,
			"agent_id":  agentID,
			"path":      pathUrl,
		},
	)
}

// RemoveAgentFromGroup 从分组移除Agent
// :id/groups/:group_id
func (h *AgentHandler) RemoveAgentFromGroup(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c) // 调用 utils 工具包直接从Gin上下文提取当前用户ID，如果不存在则返回0
	xRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	method := c.Request.Method
	agentID := c.Param("id")       // AgentID在URL中
	groupID := c.Param("group_id") // 分组ID在URL中

	req := agentModel.AgentGroupMemberRequest{AgentID: agentID, GroupID: groupID}
	if err := h.agentManagerService.RemoveAgentFromGroup(&req); err != nil {
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "remove_agent_from_group",
			"option":    "service_call.RemoveAgentFromGroup",
			"func_name": "handler.agent.RemoveAgentFromGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "从分组移除Agent失败",
			Error:   err.Error(),
			Data:    map[string]interface{}{"agent_id": agentID, "group_id": groupID},
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "从分组移除Agent成功",
		Data: map[string]interface{}{
			"agent_id": agentID,
			"group_id": groupID,
		},
	})

	logger.LogBusinessOperation(
		"remove_agent_from_group",
		currentUserID,
		"",
		clientIP,
		xRequestID,
		"success",
		"Agent从分组移除成功",
		map[string]interface{}{
			"func_name": "handler.agent.RemoveAgentFromGroup",
			"option":    "result.success",
			"group_id":  groupID,
			"agent_id":  agentID,
			"path":      pathUrl,
		},
	)
}

// GetAgentsInGroup 获取分组成员（分页形参）
func (h *AgentHandler) GetAgentsInGroup(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c) // 调用 utils 工具包直接从Gin上下文提取当前用户ID，如果不存在则返回0
	xRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	method := c.Request.Method
	groupID := c.Query("group_id")                   // 分组ID在查询参数中
	pageStr := c.DefaultQuery("page", "1")           // 分页参数，默认第1页
	pageSizeStr := c.DefaultQuery("page_size", "10") //	分页参数，默认每页10条
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	infos, err := h.agentManagerService.GetAgentsInGroup(page, pageSize, groupID)
	if err != nil {
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "get_group_members",
			"option":    "service_call.GetAgentsInGroup",
			"func_name": "handler.agent.GetAgentsInGroup",
			"group_id":  groupID,
			"page":      page,
			"page_size": pageSize,
			"path":      pathUrl,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "获取分组成员失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "ok",
		Data: map[string]interface{}{
			"members":   infos,
			"page":      page,
			"page_size": pageSize,
		},
	})

	logger.LogBusinessOperation(
		"get_group_members",
		currentUserID,
		"",
		clientIP,
		xRequestID,
		"success",
		"分组成员获取成功",
		map[string]interface{}{
			"func_name": "handler.agent.GetAgentsInGroup",
			"option":    "result.success",
			"group_id":  groupID,
			"page":      page,
			"page_size": pageSize,
			"count":     len(infos),
		},
	)
}

// GetAgentGroupList 获取分组列表（分页与筛选）
// 说明：从查询参数读取 page/page_size/tags/status/keywords，调用 Service 层形参方法，返回统一分页响应
func (h *AgentHandler) GetAgentGroupList(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c) // 调用 utils 工具包直接从Gin上下文提取当前用户ID，如果不存在则返回0
	xRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	method := c.Request.Method

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	statusStr := c.DefaultQuery("status", "")
	keywords := c.DefaultQuery("keywords", "")
	// 标签过滤参数处理 - 支持逗号分隔的标签值
	// 例如: tags=2,7 或 tags=2&tags=7 两种格式都支持
	tags := c.QueryArray("tags") // 支持 ?tags=a&tags=b

	status := -1
	if statusStr == "0" || statusStr == "1" {
		s, _ := strconv.Atoi(statusStr)
		status = s
	}

	// 调用 Service
	resp, err := h.agentManagerService.GetAgentGroupList(page, pageSize, tags, status, keywords)
	if err != nil {
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "get_agent_groups",
			"option":    "service_call.GetAgentGroupList",
			"func_name": "handler.agent.GetAgentGroupList",
			"page":      page,
			"page_size": pageSize,
			"status":    status,
			"keywords":  keywords,
			"tags":      tags,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "获取分组列表失败",
			Error:   err.Error(),
		})
		return
	}

	// 适配统一分页响应（用 system.PaginationResponse 包装）
	total := resp.Pagination.Total
	totalPages := resp.Pagination.TotalPages
	hasNext := page < totalPages
	hasPrev := page > 1

	// 组装分页响应数据
	data := map[string]interface{}{
		"groups": resp.Groups,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "ok",
		Data: system.PaginationResponse{
			Total:       total,
			Page:        page,
			PageSize:    pageSize,
			TotalPages:  totalPages,
			HasNext:     hasNext,
			HasPrevious: hasPrev,
			Data:        data,
		},
	})

	logger.LogBusinessOperation(
		"get_agent_groups",
		currentUserID,
		"",
		clientIP,
		xRequestID,
		"success",
		"Agent分组列表获取成功",
		map[string]interface{}{
			"func_name": "handler.agent.GetAgentGroupList",
			"option":    "result.success",
			"path":      pathUrl,
			"page":      page,
			"page_size": pageSize,
			"status":    status,
			"keywords":  keywords,
			"tags":      tags,
			"count":     len(resp.Groups),
			"total":     total,
		},
	)
}

// SetAgentGroupStatus 设置分组状态（激活/禁用）
// 说明：从查询参数读取 group_id；从请求体(JSON)读取 status，不再从query中获取；
//
//	完成后调用 Service 层方法并返回统一响应。
func (h *AgentHandler) SetAgentGroupStatus(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c) // 调用 utils 工具包直接从Gin上下文提取当前用户ID，如果不存在则返回0
	xRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	method := c.Request.Method
	// 路由已定义为 "/groups/:group_id/status"，因此这里从路径参数读取 group_id
	groupID := c.Param("group_id")
	// 从请求体中读取 status 字段；该字段为必填，类型为整数，取值限制为 0 或 1
	var body struct {
		Status int `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		// 请求体解析失败或缺少必要的status字段
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "set_group_status",
			"option":    "bind_json",
			"func_name": "handler.agent.SetAgentGroupStatus",
			"group_id":  groupID,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "请求体解析失败或缺少status",
			Error:   err.Error(),
		})
		return
	}

	// 校验 group_id 是否存在（保持与原有逻辑一致）
	if groupID == "" {
		err := fmt.Errorf("group_id不能为空")
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "set_group_status",
			"option":    "parameter_validation",
			"func_name": "handler.agent.SetAgentGroupStatus",
			"group_id":  groupID,
			"status":    body.Status,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "参数校验失败",
			Error:   err.Error(),
		})
		return
	}

	// 校验 status 合法取值范围（仅允许 0 或 1）
	status := body.Status
	if status != 0 && status != 1 {
		err := fmt.Errorf("status必须为0或1")
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "set_group_status",
			"option":    "parameter_validation",
			"func_name": "handler.agent.SetAgentGroupStatus",
			"group_id":  groupID,
			"status":    status,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "参数status不合法",
			Error:   err.Error(),
		})
		return
	}

	if err := h.agentManagerService.SetAgentGroupStatus(groupID, status); err != nil {
		logger.LogBusinessError(err, xRequestID, currentUserID, clientIP, pathUrl, method, map[string]interface{}{
			"operation": "set_group_status",
			"option":    "service_call.SetGroupStatus",
			"func_name": "handler.agent.SetAgentGroupStatus",
			"group_id":  groupID,
			"status":    status,
		})
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "设置分组状态失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "分组状态设置成功",
		Data: map[string]interface{}{
			"group_id":   groupID,
			"new_status": status,
		},
	})

	logger.LogBusinessOperation(
		"set_group_status",
		currentUserID,
		"",
		clientIP,
		xRequestID,
		"success",
		"设置分组状态成功",
		map[string]interface{}{
			"func_name": "handler.agent.SetAgentGroupStatus",
			"option":    "result.success",
			"group_id":  groupID,
			"status":    status,
			"path":      pathUrl,
		},
	)
}
