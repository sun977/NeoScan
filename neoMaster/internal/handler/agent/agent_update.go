// Agent 规则更新控制器
package agent

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
)

func (h *AgentHandler) GetFingerprintVersion(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.agentUpdateService == nil {
		err := errors.New("agent update service is nil")
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "agent_fingerprint_sync",
			"option":     "GetFingerprintVersion",
			"func_name":  "handler.agent.GetFingerprintVersion",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "agent update service not initialized",
			Error:   err.Error(),
		})
		return
	}

	info, err := h.agentUpdateService.GetFingerprintSnapshotInfo(c.Request.Context())
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "agent_fingerprint_sync",
			"option":     "GetFingerprintVersion",
			"func_name":  "handler.agent.GetFingerprintVersion",
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
		"func_name":    "handler.agent.GetFingerprintVersion",
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

func (h *AgentHandler) DownloadFingerprintSnapshot(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.agentUpdateService == nil {
		err := errors.New("agent update service is nil")
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "agent_fingerprint_sync",
			"option":     "DownloadFingerprintSnapshot",
			"func_name":  "handler.agent.DownloadFingerprintSnapshot",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "agent update service not initialized",
			Error:   err.Error(),
		})
		return
	}

	snapshot, err := h.agentUpdateService.BuildFingerprintSnapshot(c.Request.Context())
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "agent_fingerprint_sync",
			"option":     "DownloadFingerprintSnapshot",
			"func_name":  "handler.agent.DownloadFingerprintSnapshot",
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
		"func_name":    "handler.agent.DownloadFingerprintSnapshot",
		"version_hash": snapshot.VersionHash,
		"file_count":   snapshot.FileCount,
		"rule_path":    snapshot.RulePath,
		"timestamp":    logger.NowFormatted(),
	})

	c.Header("Content-Type", snapshot.ContentType)
	c.Header("Content-Disposition", "attachment; filename=\""+snapshot.FileName+"\"")
	c.Data(http.StatusOK, snapshot.ContentType, snapshot.Bytes)
}
