package protocol

import (
	"context"
	"fmt"
	"strings"
	"time"

	"neoagent/internal/core/scanner/brute"

	"github.com/redis/go-redis/v9"
)

// RedisCracker 实现 Redis 协议爆破
type RedisCracker struct{}

// NewRedisCracker 创建 Redis 爆破器
func NewRedisCracker() *RedisCracker {
	return &RedisCracker{}
}

// Name 返回协议名称
func (c *RedisCracker) Name() string {
	return "redis"
}

// Mode 返回爆破模式 (通常 Redis 只需要密码)
func (c *RedisCracker) Mode() brute.AuthMode {
	return brute.AuthModeOnlyPass
}

// Check 验证 Redis 凭据
func (c *RedisCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	// 配置 Redis 客户端
	// 注意: Redis 6.0+ 支持 ACL (用户名+密码)，但传统 Redis 只有密码
	// AuthModeOnlyPass 模式下 Username 为空，go-redis 会自动处理
	opts := &redis.Options{
		Addr:     addr,
		Password: auth.Password, // 如果为空字符串，go-redis 不会发送 AUTH 命令
		Username: auth.Username, // 如果为空，兼容旧版
		DB:       0,

		// 关键: 设置超时，快速失败
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		// 禁用重试，避免浪费时间
		MaxRetries: 0,
	}

	client := redis.NewClient(opts)
	defer client.Close()

	// 执行 PING 命令验证连接和认证
	// go-redis 在执行命令时会自动处理连接建立和 AUTH
	// 如果配置了 Password 但不需要密码，或者密码错误，这里会报错

	// 使用传入的 context 进行控制
	cmd := client.Ping(ctx)
	if err := cmd.Err(); err != nil {
		return false, c.handleError(err)
	}

	// 成功 (PONG)
	return true, nil
}

// handleError 将底层错误转换为标准错误
func (c *RedisCracker) handleError(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())

	// 认证失败
	// ERR invalid password
	// WRONGPASS invalid username-password pair
	// NOAUTH Authentication required
	if strings.Contains(msg, "invalid password") ||
		strings.Contains(msg, "wrongpass") ||
		strings.Contains(msg, "noauth") ||
		strings.Contains(msg, "authentication required") {
		return nil // 认证失败，非系统错误
	}

	// 连接/网络错误
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no route to host") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "target machine actively refused") || // Windows
		strings.Contains(msg, "failed to dial") || // go-redis generic dial error
		strings.Contains(msg, "connection pool") || // go-redis connection pool error
		strings.Contains(msg, "connectex") || // Windows connect exception
		strings.Contains(msg, "only one usage of each socket address") || // Windows Bind Error
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "eof") {
		return brute.ErrConnectionFailed
	}

	// 协议错误 (非 Redis 服务)
	// 例如: "redis: invalid response: ..."
	// "reading length: expected '$', got 'H'" (HTTP response)
	if strings.Contains(msg, "invalid response") ||
		strings.Contains(msg, "reading length") ||
		strings.Contains(msg, "expected") {
		return brute.ErrProtocolError
	}

	// 兜底：如果是其他未知错误，倾向于认为是协议或连接问题，避免误报成功
	// 但为了安全起见，如果不确定是否是 Connection 错误，可以归类为 ProtocolError
	return brute.ErrProtocolError
}
