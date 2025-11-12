/*
 * @author: sun977
 * @date: 2025.11.12
 * @description: 通用的工具包
 * @func:
 */

package utils

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ContextKey 类型用于标准上下文键的定义，避免使用裸字符串造成键冲突
type ContextKey string

// ContextKeyClientIP 标准上下文中存储客户端IP的统一键
const ContextKeyClientIP ContextKey = "client_ip"

// getCurrentUserIDFromGinContext 从 Gin 上下文中提取当前用户ID
// 用于从Gin上下文提取当前用户ID，如果不存在则返回0，轻校验
// 适用范围：service 层以上获取当前 userID 使用
// 来源：user_id 最初是JWT中间件写入Gin上下文 GinJWTAuthMiddleware() 中
func getCurrentUserIDFromGinContext(c *gin.Context) uint {
	if v, ok := c.Get("user_id"); ok {
		if id, ok2 := v.(uint); ok2 {
			return id
		}
	}
	return 0
}

// GetClientIPFromContext 从标准上下文读取客户端IP（统一键）
// 用于从标准上下文提取当前用户IP，如果不存在则返回0，轻校验
// 适用范围：service 层以下获取当前 clientIP 使用
// 来源：clientIPKey(定义常量名) 最初是logging中间件写入标准上下文 GinLoggingMiddleware() 中
// 用法示例：ip := utils.GetClientIPFromContext(ctx)
// 说明：
// - 使用 ContextKeyClientIP 作为唯一键，保证读写一致，跨包可用
// - 如果不存在或类型不匹配，返回空字符串
func GetClientIPFromContext(ctx context.Context) string {
	// 用于替换如下老旧调用 --- TODO 后续有空再说
	// type clientIPKeyType struct{}
	// clientIP, _ := ctx.Value(clientIPKeyType{}).(string)
	v := ctx.Value(ContextKeyClientIP)
	if ip, ok := v.(string); ok {
		return ip
	}
	return ""
}
