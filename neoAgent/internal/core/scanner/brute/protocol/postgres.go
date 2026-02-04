package protocol

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"neoagent/internal/core/scanner/brute"

	"github.com/lib/pq"
)

// PostgresCracker PostgreSQL 协议爆破器
type PostgresCracker struct{}

func NewPostgresCracker() *PostgresCracker {
	return &PostgresCracker{}
}

func (c *PostgresCracker) Name() string {
	return "postgres"
}

func (c *PostgresCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

func (c *PostgresCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// 构造连接字符串
	// sslmode=disable 禁用 SSL，提高爆破速度并兼容旧版
	// connect_timeout 控制连接建立超时
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable&connect_timeout=3",
		auth.Username, auth.Password, host, port)

	// sql.Open 不会立即建立连接，只校验参数
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		// DSN 格式错误，通常不会发生
		return false, fmt.Errorf("invalid dsn: %w", err)
	}
	defer db.Close()

	// 关键：PingContext 才会真正建立连接
	// 使用传入的 ctx 控制整体超时
	if err := db.PingContext(ctx); err != nil {
		return c.handleError(err)
	}

	return true, nil
}

// handleError 解析 PostgreSQL 错误
func (c *PostgresCracker) handleError(err error) (bool, error) {
	if err == nil {
		return true, nil
	}

	// 检查是否为 lib/pq 的 Error 类型
	if pqErr, ok := err.(*pq.Error); ok {
		// PostgreSQL Error Codes: https://www.postgresql.org/docs/current/errcodes-appendix.html
		switch pqErr.Code {
		case "28P01": // invalid_password (认证失败)
			return false, nil // 明确的密码错误，无需返回 error
		case "28000": // invalid_authorization_specification
			return false, nil
		case "53300": // too_many_connections (稍后重试)
			return false, brute.ErrConnectionFailed
		}
	}

	// 检查网络错误
	errMsg := err.Error()
	if strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no such host") {
		return false, brute.ErrConnectionFailed
	}

	// 其他未知错误，记录日志以便排查
	return false, err
}
