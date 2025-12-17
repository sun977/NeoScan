/**
 * Agent标签管理控制器(基础管理 - 标签管理)
 * 作者: Sun977
 * 日期: 2025-11-07
 * 说明: 标签管理相关的 Handler 方法集中于此，包括：
 * - GetAgentTags（获取标签）
 * - AddAgentTag（添加标签）
 * - RemoveAgentTag（移除标签）
 * - UpdateAgentTags（更新标签列表）
 * 重构策略: 保持原有业务逻辑和返回格式不变，统一成功日志使用 LogBusinessOperation。
 */

package agent

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
)

// GetAgentTags 获取指定Agent的标签列表
// 处理 GET 请求，从路径参数获取 agentID，调用服务层获取标签列表
// 遵循文件风格：规范化客户端信息、参数验证、日志记录、标准化响应
func (h *AgentHandler) GetAgentTags(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogBusinessError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":  "get_agent_tags",
				"option":     "paramValidation",
				"func_name":  "handler.agent.GetAgentTags",
				"user_agent": userAgent,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 调用服务层获取标签列表
	tags, err := h.agentManagerService.GetAgentTags(agentID)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":   "get_agent_tags",
				"option":      "agentManagerService.GetAgentTags",
				"func_name":   "handler.agent.GetAgentTags",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to get agent tags",
			Error:   err.Error(),
		})
		return
	}

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"get_agent_tags",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"获取Agent标签列表成功",
		map[string]interface{}{
			"func_name":  "handler.agent.GetAgentTags",
			"option":     "success",
			"path":       pathUrl,
			"method":     "GET",
			"user_agent": userAgent,
			"agent_id":   agentID,
			"tag_count":  len(tags),
		},
	)

	// 装填响应数据 data
	data := map[string]interface{}{
		"agent_id":  agentID,
		"operation": "get_agent_tags",
		"tags":      tags,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent tags retrieved successfully",
		Data:    data,
	})
}

// AddAgentTag 给指定Agent添加标签
// 处理 POST 请求，从路径获取 agentID，从 body 获取 tag，调用服务层添加标签
// 遵循文件风格：JSON 绑定、验证、日志、响应
func (h *AgentHandler) AddAgentTag(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogBusinessError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "add_agent_tag",
				"option":     "paramValidation",
				"func_name":  "handler.agent.AddAgentTag",
				"user_agent": userAgent,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 解析请求体
	var body struct {
		TagID uint64 `json:"tag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "add_agent_tag",
				"option":     "ShouldBindJSON",
				"func_name":  "handler.agent.AddAgentTag",
				"user_agent": userAgent,
				"agent_id":   agentID,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	// 构建服务请求
	req := &agentModel.AgentTagRequest{
		AgentID: agentID,
		TagID:   body.TagID,
	}

	// 调用服务层添加标签
	err := h.agentManagerService.AddAgentTag(req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":   "add_agent_tag",
				"option":      "agentManagerService.AddAgentTag",
				"func_name":   "handler.agent.AddAgentTag",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"tag_id":      body.TagID,
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to add agent tag",
			Error:   err.Error(),
		})
		return
	}

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"add_agent_tag",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"添加Agent标签成功",
		map[string]interface{}{
			"func_name":  "handler.agent.AddAgentTag",
			"option":     "success",
			"path":       pathUrl,
			"method":     "POST",
			"user_agent": userAgent,
			"agent_id":   agentID,
			"tag_id":     body.TagID,
		},
	)

	// 装填响应数据 data
	data := map[string]interface{}{
		"agent_id":  agentID,
		"operation": "add_agent_tag",
		"tag_id":    body.TagID,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent tag added successfully",
		Data:    data,
	})
}

// RemoveAgentTag 从指定Agent移除标签
// 处理 DELETE 请求，从路径获取 agentID，从 body 或 query 获取 tag，调用服务层移除标签
// 遵循文件风格：验证、日志、响应
func (h *AgentHandler) RemoveAgentTag(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogBusinessError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"DELETE",
			map[string]interface{}{
				"operation":  "remove_agent_tag",
				"option":     "paramValidation",
				"func_name":  "handler.agent.RemoveAgentTag",
				"user_agent": userAgent,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 解析请求体（解析请求 body 中的 tag 值）
	var body struct {
		TagID uint64 `json:"tag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"DELETE",
			map[string]interface{}{
				"operation":  "remove_agent_tag",
				"option":     "ShouldBindJSON",
				"func_name":  "handler.agent.RemoveAgentTag",
				"user_agent": userAgent,
				"agent_id":   agentID,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	// 构建服务请求
	req := &agentModel.AgentTagRequest{
		AgentID: agentID,
		TagID:   body.TagID,
	}

	// 调用服务层移除标签
	err := h.agentManagerService.RemoveAgentTag(req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"DELETE",
			map[string]interface{}{
				"operation":   "remove_agent_tag",
				"option":      "agentManagerService.RemoveAgentTag",
				"func_name":   "handler.agent.RemoveAgentTag",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"tag_id":      body.TagID,
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to remove agent tag",
			Error:   err.Error(),
		})
		return
	}

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"remove_agent_tag",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"移除Agent标签成功",
		map[string]interface{}{
			"func_name":  "handler.agent.RemoveAgentTag",
			"option":     "success",
			"path":       pathUrl,
			"method":     "DELETE",
			"user_agent": userAgent,
			"agent_id":   agentID,
			"tag_id":     body.TagID,
		},
	)

	// 装填响应数据 data
	data := map[string]interface{}{
		"agent_id":  agentID,
		"operation": "remove_agent_tag",
		"tag_id":    body.TagID,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent tag removed successfully",
		Data:    data,
	})
}

// UpdateAgentTags 更新指定Agent的标签列表
func (h *AgentHandler) UpdateAgentTags(c *gin.Context) {
	// 规范化客户端信息（用于统一日志字段）
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID（从路径参数中读取，保持与其它接口一致）
	agentID := c.Param("id")
	if agentID == "" {
		// 按项目日志规范记录业务错误
		logger.LogBusinessError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":  "update_agent_tags",
				"option":     "paramValidation",
				"func_name":  "handler.agent.UpdateAgentTags",
				"user_agent": userAgent,
			},
		)
		// 返回统一错误响应
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 解析请求体（tags为必填）
	// 说明：为了保持与已有Handler风格一致，这里使用局部结构体+binding标签进行校验
	var body struct {
		TagIDs []uint64 `json:"tag_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		// 记录解析错误日志
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":  "update_agent_tags",
				"option":     "ShouldBindJSON",
				"func_name":  "handler.agent.UpdateAgentTags",
				"user_agent": userAgent,
				"agent_id":   agentID,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层更新标签列表（原子更新：使用AddTag/RemoveTag差异更新）
	oldTags, newTags, err := h.agentManagerService.UpdateAgentTags(agentID, body.TagIDs)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		// 记录业务错误日志，包含关键字段
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":   "update_agent_tags",
				"option":      "agentManagerService.UpdateAgentTags",
				"func_name":   "handler.agent.UpdateAgentTags",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"tags_count":  len(body.TagIDs),
				"status_code": statusCode,
			},
		)

		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to update agent tags",
			Error:   err.Error(),
		})
		return
	}

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"update_agent_tags",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"更新Agent标签成功",
		map[string]interface{}{
			"func_name":      "handler.agent.UpdateAgentTags",
			"option":         "success",
			"path":           pathUrl,
			"method":         "PUT",
			"user_agent":     userAgent,
			"agent_id":       agentID,
			"old_tags_count": len(oldTags),
			"new_tags_count": len(newTags),
		},
	)

	// 返回包含agentID、旧标签、新标签的结构，满足你的返回需求
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent tags updated successfully",
		Data: map[string]interface{}{
			"agent_id":  agentID,
			"operation": "update_agent_tags",
			"old_tags":  oldTags,
			"new_tags":  newTags,
		},
	})
}
