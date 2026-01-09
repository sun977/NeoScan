// AgentUpdateHandler 处理 Agent 主动拉取规则更新请求
package agent

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"neomaster/internal/config"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	"neomaster/internal/service/agent_update"
)

type AgentUpdateHandler struct {
	cfg *config.Config
}

func NewAgentUpdateHandler(cfg *config.Config) *AgentUpdateHandler {
	return &AgentUpdateHandler{cfg: cfg}
}

func (h *AgentUpdateHandler) GetFingerprintVersion(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	rulePath := ""
	if h.cfg != nil {
		rulePath = strings.TrimSpace(h.cfg.Fingerprint.RulePath)
	}

	info, err := agent_update.GetFingerprintSnapshotInfo(c.Request.Context(), rulePath)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "agent_fingerprint_sync",
			"option":     "GetFingerprintVersion",
			"func_name":  "handler.agent.agent_update.GetFingerprintVersion",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "get fingerprint snapshot version failed",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("agent_fingerprint_version", 0, "agent", clientIP, requestID, "success", "get fingerprint snapshot version", map[string]interface{}{
		"path":         urlPath,
		"operation":    "agent_fingerprint_sync",
		"option":       "GetFingerprintVersion",
		"func_name":    "handler.agent.agent_update.GetFingerprintVersion",
		"version_hash": info.VersionHash,
		"file_count":   info.FileCount,
		"rule_path":    info.RulePath,
		"timestamp":    logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "ok",
		Data:    info,
	})
}

func (h *AgentUpdateHandler) DownloadFingerprintSnapshot(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	rulePath := ""
	if h.cfg != nil {
		rulePath = strings.TrimSpace(h.cfg.Fingerprint.RulePath)
	}

	snapshot, err := agent_update.BuildFingerprintSnapshot(c.Request.Context(), rulePath)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "agent_fingerprint_sync",
			"option":     "DownloadFingerprintSnapshot",
			"func_name":  "handler.agent.agent_update.DownloadFingerprintSnapshot",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "download fingerprint snapshot failed",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("agent_fingerprint_download", 0, "agent", clientIP, requestID, "success", "download fingerprint snapshot", map[string]interface{}{
		"path":         urlPath,
		"operation":    "agent_fingerprint_sync",
		"option":       "DownloadFingerprintSnapshot",
		"func_name":    "handler.agent.agent_update.DownloadFingerprintSnapshot",
		"version_hash": snapshot.VersionHash,
		"file_count":   snapshot.FileCount,
		"rule_path":    snapshot.RulePath,
		"timestamp":    logger.NowFormatted(),
	})

	c.Header("Content-Type", snapshot.ContentType)
	c.Header("Content-Disposition", "attachment; filename=\""+snapshot.FileName+"\"")
	c.Data(http.StatusOK, snapshot.ContentType, snapshot.Bytes)
}
