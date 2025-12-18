// agent_jwt.go
// 该文件定义 Agent 专属的 JWT 相关内容，包括 Claims 结构体和 JWT 管理器
package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// AgentClaims 定义 Agent 专属的 JWT Claims
// 区别于用户系统的 Claims，这里只包含 Agent 及其宿主机的身份信息
type AgentClaims struct {
	AgentID  string `json:"agent_id"` // Agent UUID
	Hostname string `json:"hostname"` // 机器主机名
	jwt.RegisteredClaims
}

// TODO: 后续实现 AgentJWTManager
// type AgentJWTManager struct {
//     secretKey []byte
//     ...
// }
