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

	// 2. 生成文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("neoscan_fingerprint_rules_%s.json", timestamp)

	// 3. 记录审计日志
	logger.LogBusinessOperation("export_fingerprint_rules", 0, "", clientIP, requestID, "success", "export fingerprint rules", map[string]interface{}{
		"filename":  filename,
		"size":      len(data),
		"timestamp": logger.NowFormatted(),
	})

	// 4. 返回文件流
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", data)
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

	// 3. 调用 Manager 导入数据
	// 默认 overwrite=true (根据需求可改为参数控制)
	overwrite := c.Query("overwrite") == "true"
	if err := h.ruleManager.ImportRules(c.Request.Context(), data, overwrite); err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to import rules", err, requestID, clientIP, urlPath, "ImportRules")
		return
	}

	// 4. 记录审计日志
	logger.LogBusinessOperation("import_fingerprint_rules", 0, "", clientIP, requestID, "success", "import fingerprint rules", map[string]interface{}{
		"filename":  file.Filename,
		"size":      len(data),
		"overwrite": overwrite,
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
