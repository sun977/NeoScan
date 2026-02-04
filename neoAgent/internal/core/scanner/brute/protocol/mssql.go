package protocol

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"neoagent/internal/core/scanner/brute"

	_ "github.com/denisenkom/go-mssqldb"
)

// MSSQLCracker MSSQL 协议爆破器
type MSSQLCracker struct{}

func NewMSSQLCracker() *MSSQLCracker {
	return &MSSQLCracker{}
}

func (c *MSSQLCracker) Name() string {
	return "mssql"
}

func (c *MSSQLCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

func (c *MSSQLCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// 构建连接字符串
	// server=%s;user id=%s;password=%s;port=%v;encrypt=disable;timeout=%v
	// 使用 URL 对象构建以处理特殊字符转义
	query := url.Values{}
	query.Add("database", "master") // 默认连接 master 库
	query.Add("encrypt", "disable") // 默认禁用加密，兼容性更好
	// connection timeout 是建立连接的超时时间
	// 但在 Check 内部，我们更依赖 ctx 的超时控制
	query.Add("connection timeout", "3")

	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(auth.Username, auth.Password),
		Host:     fmt.Sprintf("%s:%d", host, port),
		RawQuery: query.Encode(),
	}

	// sql.Open 不会立即建立连接，所以这里 err 极少发生
	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		return false, fmt.Errorf("invalid dsn: %w", err)
	}
	defer db.Close()

	// 强制设置连接参数，防止连接池副作用
	db.SetConnMaxLifetime(5 * time.Second)
	db.SetMaxIdleConns(0)

	// 使用 PingContext 进行连接检测
	// Context 的超时由外部控制
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return c.handleError(err)
	}

	return true, nil
}

// handleError 解析 MSSQL 错误
func (c *MSSQLCracker) handleError(err error) (bool, error) {
	if err == nil {
		return true, nil
	}

	errMsg := err.Error()

	// 鉴权失败
	// Error: 18456, Severity: 14, State: 1. Login failed for user 'sa'.
	if strings.Contains(errMsg, "Login failed") ||
		strings.Contains(errMsg, "Login failed for user") {
		return false, nil // 密码错误
	}

	// 连接错误
	// "connection refused"
	// "i/o timeout"
	// "pre-login handshake failed"
	// "The connection is closed"
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "pre-login handshake failed") ||
		strings.Contains(errMsg, "The connection is closed") ||
		strings.Contains(errMsg, "context deadline exceeded") {
		return false, brute.ErrConnectionFailed
	}

	// 其他错误，默认作为连接错误处理
	return false, brute.ErrConnectionFailed
}
