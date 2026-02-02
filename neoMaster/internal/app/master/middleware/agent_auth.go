// agent_auth.go
// 该文件定义 Agent 专属的鉴权中间件，用于处理 Agent 上报接口的鉴权
// 不同于用户系统的鉴权逻辑，此中间件仅用于和 Agent 进行交互，仅需验证 JWT Token
package middleware

import (
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GinAgentAuthMiddleware Agent 鉴权中间件
// 该中间件专用于处理 Agent 上报接口的鉴权，不依赖于用户系统的鉴权逻辑
func (m *MiddlewareManager) GinAgentAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.LogInfo("Agent Auth Middleware Triggered", "", 0, "", c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"func_name": "GinAgentAuthMiddleware",
		})
		// 1. 从 header 提取 Token (Authorization: Bearer <token>)
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, system.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "failed",
				Message: "missing authorization header",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, system.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "failed",
				Message: "invalid authorization format",
			})
			c.Abort()
			return
		}
		token := parts[1]

		logger.LogInfo("Agent Auth Middleware Checking Token", "", 0, "", c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"func_name": "GinAgentAuthMiddleware",
			"token_len": len(token),
			"token_pre": token[:10],
		})

		// 2. 验证 Token
		// Linus: 使用 AgentService 查询 Token，保持层级清晰
		agent, err := m.agentService.GetAgentByToken(token)
		if err != nil {
			logger.LogError(err, "", 0, "", "GinAgentAuthMiddleware", "GetAgentByToken", map[string]interface{}{
				"token": token,
			})
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "internal server error",
			})
			c.Abort()
			return
		}

		if agent == nil {
			c.JSON(http.StatusUnauthorized, system.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "failed",
				Message: "invalid token",
			})
			c.Abort()
			return
		}

		// 3. 将 AgentID 注入上下文
		c.Set("agent_id", agent.AgentID)

		// 4. (可选) 检查 Agent 是否在黑名单/被禁用
		// if agent.Status == agentModel.AgentStatusBlocked { ... }

		c.Next()
	}
}
