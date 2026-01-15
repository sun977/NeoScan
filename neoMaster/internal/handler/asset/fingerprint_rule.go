// 实现指纹规则管理 API (Export/Import)
package asset

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	"neomaster/internal/service/fingerprint"
	"neomaster/internal/service/fingerprint/converters"
)

// FingerprintRuleHandler 指纹规则管理控制器
type FingerprintRuleHandler struct {
	ruleManager *fingerprint.RuleManager
}

// NewFingerprintRuleHandler 创建控制器实例
func NewFingerprintRuleHandler(ruleManager *fingerprint.RuleManager) *FingerprintRuleHandler {
	return &FingerprintRuleHandler{ruleManager: ruleManager}
}

// ExportRules 导出规则 (Admin)
// GET /api/v1/asset/fingerprint/rules/export
func (h *FingerprintRuleHandler) ExportRules(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.ruleManager == nil {
		h.handleError(c, http.StatusInternalServerError, "rule manager not initialized", nil, requestID, clientIP, urlPath, "ExportRules")
		return
	}

	// 1. 调用 Manager 导出数据
	data, err := h.ruleManager.ExportRules(c.Request.Context())
	if err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to export rules", err, requestID, clientIP, urlPath, "ExportRules")
		return
	}

	// 2. 计算签名 (SHA256)
	signature := h.ruleManager.CalculateSignature(data)

	// 3. 生成文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("neoscan_fingerprint_rules_%s.json", timestamp)

	// 4. 记录审计日志
	logger.LogBusinessOperation("export_fingerprint_rules", 0, "", clientIP, requestID, "success", "export fingerprint rules", map[string]interface{}{
		"filename":  filename,
		"size":      len(data),
		"signature": signature,
		"timestamp": logger.NowFormatted(),
	})

	// 5. 返回文件流和签名头
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("X-Content-Signature", signature) // 自定义头返回签名
	c.Data(http.StatusOK, "application/json", data)
}

// PublishRules 发布规则 (将数据库中的规则同步到磁盘文件，供 Agent 下载)
// POST /api/v1/asset/fingerprint/rules/publish
func (h *FingerprintRuleHandler) PublishRules(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.ruleManager == nil {
		h.handleError(c, http.StatusInternalServerError, "rule manager not initialized", nil, requestID, clientIP, urlPath, "PublishRules")
		return
	}

	// 1. 调用 Manager 执行发布逻辑 (DB -> Disk)
	// 这个操作会：
	// 1. 从 DB 读取最新规则
	// 2. 生成 JSON 文件并覆盖磁盘上的规则文件
	// 3. 更新文件 mtime，触发 AgentUpdateService 的缓存失效
	if err := h.ruleManager.PublishRulesToDisk(c.Request.Context()); err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to publish rules", err, requestID, clientIP, urlPath, "PublishRules")
		return
	}

	logger.LogBusinessOperation("publish_fingerprint_rules", 0, "", clientIP, requestID, "success", "publish fingerprint rules to disk", map[string]interface{}{
		"timestamp": logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Rules published successfully. Agents will receive updates shortly.",
	})
}

// ImportRules 导入规则 (Admin)
// POST /api/v1/asset/fingerprint/rules/import
func (h *FingerprintRuleHandler) ImportRules(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.ruleManager == nil {
		h.handleError(c, http.StatusInternalServerError, "rule manager not initialized", nil, requestID, clientIP, urlPath, "ImportRules")
		return
	}

	// 1. 获取上传文件
	file, err := c.FormFile("file")
	if err != nil {
		h.handleError(c, http.StatusBadRequest, "failed to get uploaded file", err, requestID, clientIP, urlPath, "ImportRules")
		return
	}

	// 2. 读取文件内容
	src, err := file.Open()
	if err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to open uploaded file", err, requestID, clientIP, urlPath, "ImportRules")
		return
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to read uploaded file", err, requestID, clientIP, urlPath, "ImportRules")
		return
	}

	// 3. 获取预期签名 (可选)
	// 如果前端或客户端提供了签名头，我们必须验证
	// 如果是 Multipart Form，可能在 Form 字段里？或者 Header 里？
	// 这里支持从 Header "X-Content-Signature" 读取
	expectedSignature := c.GetHeader("X-Content-Signature")

	// 4. 调用 Manager 导入数据
	// 默认 overwrite=true (根据需求可改为参数控制)
	overwrite := c.Query("overwrite") == "true"

	// 从 Query 中获取 Source，默认为 "custom" (API 导入默认为自定义)
	// Admin 也可以指定 source=system 用于恢复系统规则
	source := c.DefaultQuery("source", "custom")

	// 默认格式为 StandardJSON，可以通过查询参数 format 指定 (e.g. ?format=goby)
	formatStr := c.DefaultQuery("format", "standard")
	var format converters.ConverterType
	switch formatStr {
	case "goby":
		format = converters.TypeGoby
	case "ehole":
		format = converters.TypeEHole
	default:
		format = converters.TypeStandard
	}

	if err := h.ruleManager.ImportRules(c.Request.Context(), data, overwrite, expectedSignature, source, format); err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to import rules", err, requestID, clientIP, urlPath, "ImportRules")
		return
	}

	// 5. 记录审计日志
	logger.LogBusinessOperation("import_fingerprint_rules", 0, "", clientIP, requestID, "success", "import fingerprint rules", map[string]interface{}{
		"filename":  file.Filename,
		"size":      len(data),
		"overwrite": overwrite,
		"source":    source,
		"signature": expectedSignature, // Log what was provided
		"timestamp": logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "rules imported successfully",
	})
}

// GetVersion 获取规则库版本信息 (Admin)
// GET /api/v1/asset/fingerprint/rules/version
func (h *FingerprintRuleHandler) GetVersion(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.ruleManager == nil {
		h.handleError(c, http.StatusInternalServerError, "rule manager not initialized", nil, requestID, clientIP, urlPath, "GetVersion")
		return
	}

	stats, err := h.ruleManager.GetRuleStats(c.Request.Context())
	if err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to get rule stats", err, requestID, clientIP, urlPath, "GetVersion")
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "success",
		Data:    stats,
	})
}

// ListBackups 获取规则库备份列表
// GET /api/v1/asset/fingerprint/rules/backups
func (h *FingerprintRuleHandler) ListBackups(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.ruleManager == nil {
		h.handleError(c, http.StatusInternalServerError, "rule manager not initialized", nil, requestID, clientIP, urlPath, "ListBackups")
		return
	}

	backups, err := h.ruleManager.ListBackups()
	if err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to list backups", err, requestID, clientIP, urlPath, "ListBackups")
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "success",
		Data:    backups,
	})
}

// RollbackRules 回滚规则库到指定备份
// POST /api/v1/asset/fingerprint/rules/rollback
func (h *FingerprintRuleHandler) RollbackRules(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.ruleManager == nil {
		h.handleError(c, http.StatusInternalServerError, "rule manager not initialized", nil, requestID, clientIP, urlPath, "RollbackRules")
		return
	}

	var req struct {
		Filename string `json:"filename" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, http.StatusBadRequest, "invalid request body", err, requestID, clientIP, urlPath, "RollbackRules")
		return
	}

	if err := h.ruleManager.Rollback(c.Request.Context(), req.Filename); err != nil {
		h.handleError(c, http.StatusInternalServerError, "rollback failed", err, requestID, clientIP, urlPath, "RollbackRules")
		return
	}

	// 记录审计日志
	logger.LogBusinessOperation("rollback_fingerprint_rules", 0, "", clientIP, requestID, "success", "rollback fingerprint rules", map[string]interface{}{
		"filename":  req.Filename,
		"timestamp": logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "rollback successful",
	})
}

// handleError 统一错误处理
func (h *FingerprintRuleHandler) handleError(c *gin.Context, code int, msg string, err error, requestID, clientIP, path, option string) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	logger.LogBusinessError(err, requestID, 0, clientIP, path, c.Request.Method, map[string]interface{}{
		"operation": "fingerprint_rule_management",
		"option":    option,
		"error":     errMsg,
		"message":   msg,
	})

	c.JSON(code, system.APIResponse{
		Code:    code,
		Status:  "failed",
		Message: msg,
		Error:   errMsg,
	})
}
