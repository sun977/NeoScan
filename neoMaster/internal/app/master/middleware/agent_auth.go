// agent_auth.go
// 该文件定义 Agent 专属的鉴权中间件，用于处理 Agent 上报接口的鉴权
// 不同于用户系统的鉴权逻辑，此中间件仅用于和 Agent 进行交互，仅需验证 JWT Token
package middleware

import (
	"github.com/gin-gonic/gin"
)

// GinAgentAuthMiddleware Agent 鉴权中间件
// 该中间件专用于处理 Agent 上报接口的鉴权，不依赖于用户系统的鉴权逻辑
func (m *MiddlewareManager) GinAgentAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现具体的鉴权逻辑
		// 1. 从 header 提取 Token (Authorization: Bearer <token>)
		// 2. 解析并验证 AgentClaims
		// 3. (可选) 检查 Agent 是否在黑名单/被禁用
		// 4. 将 AgentID 注入上下文 c.Set("agent_id", claims.AgentID)

		// 占位逻辑：目前直接放行，后续完善
		c.Next()
	}
}
