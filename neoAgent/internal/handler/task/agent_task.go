/**
 * Agent任务处理器
 * @author: sun977
 * @date: 2025.10.21
 * @description: 处理Master端发送的任务管理HTTP请求
 * @func: 占位符实现，待后续完善
 */
package task

import (
	"net/http"
	"strconv"
	"time"

	"neoagent/internal/model/base"
	serviceTask "neoagent/internal/service/task"

	"github.com/gin-gonic/gin"
)

// AgentTaskHandler Agent任务处理器接口
type AgentTaskHandler interface {
	// ==================== Agent任务管理（🔴 响应Master端命令） ====================
	GetTaskList(c *gin.Context) // 获取Agent任务列表 [响应Master端GET /:id/tasks]
	CreateTask(c *gin.Context)  // 创建新任务 [响应Master端POST /:id/tasks]
	GetTask(c *gin.Context)     // 获取特定任务信息 [响应Master端GET /:id/tasks/:task_id]
	DeleteTask(c *gin.Context)  // 删除任务 [响应Master端DELETE /:id/tasks/:task_id]

	// ==================== 任务执行控制 ====================
	StartTask(c *gin.Context)     // 启动任务执行
	StopTask(c *gin.Context)      // 停止任务执行
	PauseTask(c *gin.Context)     // 暂停任务执行
	ResumeTask(c *gin.Context)    // 恢复任务执行
	GetTaskStatus(c *gin.Context) // 获取任务执行状态
	CancelTask(c *gin.Context)    // 取消任务执行

	// ==================== 任务查询和监控 ====================
	ListTasks(c *gin.Context)       // 列出所有任务
	GetTaskProgress(c *gin.Context) // 获取任务进度
	GetTaskLogs(c *gin.Context)     // 获取任务日志（复数形式）

	// ==================== 任务配置管理 ====================
	UpdateTaskConfig(c *gin.Context)   // 更新任务配置
	UpdateTaskPriority(c *gin.Context) // 更新任务优先级

	// ==================== 任务队列管理 ====================
	GetTaskQueue(c *gin.Context)   // 获取任务队列
	ClearTaskQueue(c *gin.Context) // 清空任务队列

	// ==================== 任务统计监控 ====================
	GetTaskStats(c *gin.Context)   // 获取任务统计信息
	GetTaskMetrics(c *gin.Context) // 获取任务指标信息

	// ==================== 任务结果管理 ====================
	GetTaskResult(c *gin.Context) // 获取任务执行结果
	GetTaskLog(c *gin.Context)    // 获取任务执行日志（单数形式）
	CleanupTask(c *gin.Context)   // 清理任务资源
}

// agentTaskHandler Agent任务处理器实现
type agentTaskHandler struct {
	taskService serviceTask.AgentTaskService
}

// NewAgentTaskHandler 创建Agent任务处理器实例
func NewAgentTaskHandler(taskService serviceTask.AgentTaskService) AgentTaskHandler {
	return &agentTaskHandler{
		taskService: taskService,
	}
}

// ==================== Agent任务管理处理器实现 ====================

// GetTaskList 获取Agent任务列表
// @Summary 获取任务列表
// @Description 获取Agent上的所有任务列表
// @Tags Agent任务
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param status query string false "任务状态过滤"
// @Success 200 {object} map[string]interface{} "任务列表获取成功"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks [get]
func (h *agentTaskHandler) GetTaskList(c *gin.Context) {
	// 解析查询参数
	page := parseIntParam(c, "page", 1)
	size := parseIntParam(c, "size", 10)
	status := c.Query("status")

	// 调用任务服务获取任务列表
	tasks, err := h.taskService.GetTaskList(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "获取任务列表失败",
			Error:   err.Error(),
		})
		return
	}

	// 简单的内存分页（因为Service目前返回所有）
	// 实际生产中应该在Service/Repo层做分页
	total := len(tasks)
	start := (page - 1) * size
	end := start + size
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedTasks := tasks[start:end]

	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "任务列表获取成功",
		Data: map[string]interface{}{
			"tasks": pagedTasks,
			"pagination": map[string]interface{}{
				"page":       page,
				"size":       size,
				"total":      total,
				"total_page": (total + size - 1) / size,
			},
			"filter": map[string]interface{}{
				"status": status,
			},
		},
	})
}

// CreateTask 创建新任务
// @Summary 创建新任务
// @Description 在Agent上创建新的执行任务
// @Tags Agent任务
// @Accept json
// @Produce json
// @Param task body map[string]interface{} true "任务数据"
// @Success 201 {object} map[string]interface{} "任务创建成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks [post]
func (h *agentTaskHandler) CreateTask(c *gin.Context) {
	var taskData map[string]interface{}
	if err := c.ShouldBindJSON(&taskData); err != nil {
		c.JSON(http.StatusBadRequest, base.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "任务数据格式错误",
			Error:   err.Error(),
		})
		return
	}

	// 验证必需字段
	taskName, exists := taskData["name"].(string)
	if !exists || taskName == "" {
		c.JSON(http.StatusBadRequest, base.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "任务名称不能为空",
		})
		return
	}

	taskType, exists := taskData["type"].(string)
	if !exists || taskType == "" {
		c.JSON(http.StatusBadRequest, base.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "任务类型不能为空",
		})
		return
	}

	// TODO: 实现任务创建处理逻辑
	// 1. 验证任务数据有效性
	// 2. 调用任务服务创建任务
	// 3. 返回创建的任务信息

	// 占位符实现
	taskID := "task-" + strconv.FormatInt(time.Now().Unix(), 10)
	c.JSON(http.StatusCreated, base.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "CreateTask处理器待实现 - 需要实现任务创建处理逻辑",
		Data: map[string]interface{}{
			"task_id":    taskID,
			"name":       taskName,
			"type":       taskType,
			"status":     "created",
			"created_at": time.Now(),
		},
	})
}

// GetTask 获取特定任务信息
// @Summary 获取任务信息
// @Description 获取指定任务的详细信息
// @Tags Agent任务
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务信息获取成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id} [get]
func (h *agentTaskHandler) GetTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, base.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "任务不存在或获取失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "任务信息获取成功",
		Data:    task,
	})
}

// DeleteTask 删除任务
// @Summary 删除任务
// @Description 删除指定的任务
// @Tags Agent任务
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务删除成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id} [delete]
func (h *agentTaskHandler) DeleteTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	// TODO: 实现任务删除处理逻辑
	// 1. 验证任务是否可以删除
	// 2. 调用任务服务删除任务
	// 3. 返回删除结果

	// 占位符实现
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "DeleteTask处理器待实现 - 需要实现任务删除处理逻辑",
		Data: map[string]interface{}{
			"task_id":       taskID,
			"delete_status": "completed",
			"deleted_at":    time.Now(),
		},
	})
}

// ==================== 任务执行控制处理器实现 ====================

// StartTask 启动任务执行
// @Summary 启动任务执行
// @Description 启动指定任务的执行
// @Tags Agent任务
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务启动成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/start [post]
func (h *agentTaskHandler) StartTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	// TODO: 实现任务启动处理逻辑
	// 1. 验证任务状态是否可启动
	// 2. 调用任务服务启动任务
	// 3. 返回启动结果

	// 占位符实现
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "StartTask处理器待实现 - 需要实现任务启动处理逻辑",
		Data: map[string]interface{}{
			"task_id":      taskID,
			"start_status": "started",
			"started_at":   time.Now(),
		},
	})
}

// StopTask 停止任务执行
// @Summary 停止任务执行
// @Description 停止指定任务的执行
// @Tags Agent任务
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务停止成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/stop [post]
func (h *agentTaskHandler) StopTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	if err := h.taskService.StopTask(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, base.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "停止任务失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "任务停止信号已发送",
		Data: map[string]interface{}{
			"task_id":     taskID,
			"stop_status": "stopping",
			"stopped_at":  time.Now(),
		},
	})
}

// PauseTask 暂停任务执行
// @Summary 暂停任务执行
// @Description 暂停指定任务的执行
// @Tags Agent任务
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务暂停成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/pause [post]
func (h *agentTaskHandler) PauseTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	// TODO: 实现任务暂停处理逻辑
	// 1. 验证任务状态是否可暂停
	// 2. 调用任务服务暂停任务
	// 3. 返回暂停结果

	// 占位符实现
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "PauseTask处理器待实现 - 需要实现任务暂停处理逻辑",
		Data: map[string]interface{}{
			"task_id":      taskID,
			"pause_status": "paused",
			"paused_at":    time.Now(),
		},
	})
}

// ResumeTask 恢复任务执行
// @Summary 恢复任务执行
// @Description 恢复指定任务的执行
// @Tags Agent任务
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务恢复成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/resume [post]
func (h *agentTaskHandler) ResumeTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	// TODO: 实现任务恢复处理逻辑
	// 1. 验证任务状态是否可恢复
	// 2. 调用任务服务恢复任务
	// 3. 返回恢复结果

	// 占位符实现
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "ResumeTask处理器待实现 - 需要实现任务恢复处理逻辑",
		Data: map[string]interface{}{
			"task_id":       taskID,
			"resume_status": "resumed",
			"resumed_at":    time.Now(),
		},
	})
}

// GetTaskStatus 获取任务执行状态
// @Summary 获取任务执行状态
// @Description 获取指定任务的执行状态和进度
// @Tags Agent任务
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务状态获取成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/status [get]
func (h *agentTaskHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	status, err := h.taskService.GetTaskStatus(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, base.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "获取任务状态失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "任务状态获取成功",
		Data:    status,
	})
}

// ==================== 任务结果管理处理器实现 ====================

// GetTaskResult 获取任务执行结果
// @Summary 获取任务执行结果
// @Description 获取指定任务的执行结果
// @Tags Agent任务
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务结果获取成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/result [get]
func (h *agentTaskHandler) GetTaskResult(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	result, err := h.taskService.GetTaskResult(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, base.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "获取任务结果失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "任务结果获取成功",
		Data:    result,
	})
}

// GetTaskLog 获取任务执行日志
// @Summary 获取任务执行日志
// @Description 获取指定任务的执行日志
// @Tags Agent任务
// @Produce json
// @Param task_id path string true "任务ID"
// @Param lines query int false "日志行数" default(100)
// @Param level query string false "日志级别过滤"
// @Success 200 {object} map[string]interface{} "任务日志获取成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/log [get]
func (h *agentTaskHandler) GetTaskLog(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	logs, err := h.taskService.GetTaskLog(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, base.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "获取任务日志失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "任务日志获取成功",
		Data:    logs,
	})
}

// CleanupTask 清理任务资源
// @Summary 清理任务资源
// @Description 清理指定任务的相关资源
// @Tags Agent任务
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务资源清理成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/cleanup [post]
func (h *agentTaskHandler) CleanupTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	// TODO: 实现任务资源清理处理逻辑
	// 1. 调用任务服务清理任务资源
	// 2. 返回清理结果

	// 占位符实现
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "CleanupTask处理器待实现 - 需要实现任务资源清理处理逻辑",
		Data: gin.H{
			"task_id":        taskID,
			"cleanup_status": "completed",
			"cleaned_at":     time.Now(),
			"resources_freed": gin.H{
				"temp_files": "50MB",
				"memory":     "256MB",
				"processes":  3,
			},
		},
	})
}

// ==================== 任务执行控制扩展实现 ====================

// CancelTask 取消任务执行
// @Summary 取消任务执行
// @Description 取消正在执行或等待执行的任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务取消成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/cancel [post]
func (h *agentTaskHandler) CancelTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if !validateRequiredParam(c, "任务ID", taskID) {
		return
	}

	if err := h.taskService.StopTask(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, base.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "取消任务失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "任务取消信号已发送",
		Data: map[string]interface{}{
			"task_id":       taskID,
			"cancel_status": "cancelling",
			"cancelled_at":  time.Now(),
		},
	})
}

// ==================== 任务查询和监控实现 ====================

// ListTasks 列出所有任务
// @Summary 列出所有任务
// @Description 获取所有任务的列表信息
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param status query string false "任务状态过滤"
// @Param limit query int false "返回数量限制"
// @Param offset query int false "偏移量"
// @Success 200 {object} map[string]interface{} "任务列表"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/list [get]
func (h *agentTaskHandler) ListTasks(c *gin.Context) {
	// TODO: 实现任务列表获取逻辑
	status := c.Query("status")
	limit := parseIntParam(c, "limit", 10)
	offset := parseIntParam(c, "offset", 0)

	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "ListTasks处理器待实现",
		Data: map[string]interface{}{
			"tasks": []map[string]interface{}{
				{
					"task_id":    "task-001",
					"name":       "扫描任务1",
					"status":     "running",
					"progress":   45.5,
					"created_at": "2024-01-01T10:00:00Z",
				},
				{
					"task_id":    "task-002",
					"name":       "扫描任务2",
					"status":     "completed",
					"progress":   100.0,
					"created_at": "2024-01-01T09:00:00Z",
				},
			},
			"total":  2,
			"limit":  limit,
			"offset": offset,
			"filter": map[string]interface{}{
				"status": status,
			},
		},
	})
}

// GetTaskProgress 获取任务进度
// @Summary 获取任务进度
// @Description 获取指定任务的执行进度信息
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务进度信息"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/progress [get]
func (h *agentTaskHandler) GetTaskProgress(c *gin.Context) {
	// TODO: 实现任务进度获取逻辑
	taskID := c.Param("task_id")
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "GetTaskProgress处理器待实现",
		Data: map[string]interface{}{
			"task_id":          taskID,
			"progress_percent": 67.5,
			"current_step":     "端口扫描",
			"total_steps":      5,
			"completed_steps":  3,
			"estimated_time":   "15分钟",
			"start_time":       "2024-01-01T10:00:00Z",
			"last_update":      time.Now(),
		},
	})
}

// GetTaskLogs 获取任务日志（复数形式）
// @Summary 获取任务日志
// @Description 获取指定任务的执行日志信息
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param level query string false "日志级别过滤"
// @Param limit query int false "返回数量限制"
// @Success 200 {object} map[string]interface{} "任务日志信息"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/logs [get]
func (h *agentTaskHandler) GetTaskLogs(c *gin.Context) {
	// TODO: 实现任务日志获取逻辑
	taskID := c.Param("task_id")
	level := c.Query("level")
	limit := parseIntParam(c, "limit", 100)

	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "GetTaskLogs处理器待实现",
		Data: map[string]interface{}{
			"task_id": taskID,
			"logs": []map[string]interface{}{
				{
					"timestamp": "2024-01-01T10:00:01Z",
					"level":     "INFO",
					"message":   "任务开始执行",
				},
				{
					"timestamp": "2024-01-01T10:00:05Z",
					"level":     "INFO",
					"message":   "开始端口扫描",
				},
				{
					"timestamp": "2024-01-01T10:00:10Z",
					"level":     "WARN",
					"message":   "发现开放端口: 80, 443",
				},
			},
			"total": 3,
			"limit": limit,
			"filter": map[string]interface{}{
				"level": level,
			},
		},
	})
}

// ==================== 任务配置管理实现 ====================

// UpdateTaskConfig 更新任务配置
// @Summary 更新任务配置
// @Description 更新指定任务的配置参数
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param config body map[string]interface{} true "任务配置"
// @Success 200 {object} map[string]interface{} "配置更新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/config [put]
func (h *agentTaskHandler) UpdateTaskConfig(c *gin.Context) {
	// TODO: 实现任务配置更新逻辑
	taskID := c.Param("task_id")
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "UpdateTaskConfig处理器待实现",
		Data: map[string]interface{}{
			"task_id":    taskID,
			"updated_at": time.Now(),
			"config": map[string]interface{}{
				"timeout":     300,
				"retry_count": 3,
				"priority":    "high",
			},
		},
	})
}

// UpdateTaskPriority 更新任务优先级
// @Summary 更新任务优先级
// @Description 更新指定任务的执行优先级
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param priority body map[string]interface{} true "优先级信息"
// @Success 200 {object} map[string]interface{} "优先级更新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "任务不存在"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/{task_id}/priority [put]
func (h *agentTaskHandler) UpdateTaskPriority(c *gin.Context) {
	// TODO: 实现任务优先级更新逻辑
	taskID := c.Param("task_id")
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "UpdateTaskPriority处理器待实现",
		Data: map[string]interface{}{
			"task_id":    taskID,
			"priority":   "high",
			"updated_at": time.Now(),
		},
	})
}

// ==================== 任务队列管理实现 ====================

// GetTaskQueue 获取任务队列
// @Summary 获取任务队列
// @Description 获取当前任务队列状态和排队任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "任务队列信息"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/queue [get]
func (h *agentTaskHandler) GetTaskQueue(c *gin.Context) {
	// TODO: 实现任务队列获取逻辑
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "GetTaskQueue处理器待实现",
		Data: map[string]interface{}{
			"queue_size":     3,
			"running_tasks":  2,
			"pending_tasks":  1,
			"max_concurrent": 5,
			"queue": []map[string]interface{}{
				{
					"task_id":  "task-003",
					"priority": "high",
					"status":   "pending",
					"position": 1,
				},
			},
		},
	})
}

// ClearTaskQueue 清空任务队列
// @Summary 清空任务队列
// @Description 清空所有等待中的任务队列
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "队列清空成功"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/queue/clear [post]
func (h *agentTaskHandler) ClearTaskQueue(c *gin.Context) {
	// TODO: 实现任务队列清空逻辑
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "ClearTaskQueue处理器待实现",
		Data: map[string]interface{}{
			"cleared_tasks": 3,
			"cleared_at":    time.Now(),
		},
	})
}

// ==================== 任务统计监控实现 ====================

// GetTaskStats 获取任务统计信息
// @Summary 获取任务统计信息
// @Description 获取任务执行的统计数据
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "任务统计信息"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/stats [get]
func (h *agentTaskHandler) GetTaskStats(c *gin.Context) {
	// TODO: 实现任务统计获取逻辑
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "GetTaskStats处理器待实现",
		Data: map[string]interface{}{
			"total_tasks":     156,
			"completed_tasks": 142,
			"failed_tasks":    8,
			"running_tasks":   2,
			"pending_tasks":   4,
			"success_rate":    91.0,
			"avg_duration":    "5m30s",
			"last_24h": map[string]interface{}{
				"completed": 25,
				"failed":    2,
				"created":   28,
			},
		},
	})
}

// GetTaskMetrics 获取任务指标信息
// @Summary 获取任务指标信息
// @Description 获取任务执行的详细指标和性能数据
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "任务指标信息"
// @Failure 500 {object} map[string]interface{} "内部服务器错误"
// @Router /agent/tasks/metrics [get]
func (h *agentTaskHandler) GetTaskMetrics(c *gin.Context) {
	// TODO: 实现任务指标获取逻辑
	c.JSON(http.StatusOK, base.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "GetTaskMetrics处理器待实现",
		Data: map[string]interface{}{
			"metrics": map[string]interface{}{
				"total_tasks":     100,
				"running_tasks":   5,
				"completed_tasks": 85,
				"failed_tasks":    10,
				"avg_duration":    "2m30s",
				"success_rate":    85.0,
				"cpu_usage":       "15%",
				"memory_usage":    "256MB",
				"disk_io":         "10MB/s",
				"network_io":      "5MB/s",
			},
			"period": "last_24h",
		},
	})
}

// ==================== 辅助函数 ====================

// parseIntParam 解析整数参数
func parseIntParam(c *gin.Context, paramName string, defaultValue int) int {
	if paramStr := c.Query(paramName); paramStr != "" {
		if value, err := strconv.Atoi(paramStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// validateRequiredParam 验证必需参数
func validateRequiredParam(c *gin.Context, paramName, paramValue string) bool {
	if paramValue == "" {
		c.JSON(http.StatusBadRequest, base.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: paramName + "不能为空",
		})
		return false
	}
	return true
}
