package protocol

import (
	"context"
	"fmt"
	"strings"
	"time"

	"neoagent/internal/core/scanner/brute"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouseCracker ClickHouse 协议爆破器
type ClickHouseCracker struct{}

func NewClickHouseCracker() *ClickHouseCracker {
	return &ClickHouseCracker{}
}

func (c *ClickHouseCracker) Name() string {
	return "clickhouse"
}

func (c *ClickHouseCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

func (c *ClickHouseCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// 使用 native TCP 协议
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", host, port)},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: auth.Username,
			Password: auth.Password,
		},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: "neoAgent", Version: "1.0"},
			},
		},
		DialTimeout: 3 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		// 禁用 debug 日志
		Debug: false,
	})

	if err != nil {
		// Open 这里的错误通常是配置错误
		return false, fmt.Errorf("invalid config: %w", err)
	}

	// 必须使用带有超时的 Context 进行 Ping，否则可能被卡住
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := conn.Ping(pingCtx); err != nil {
		return c.handleError(err)
	}

	return true, nil
}

// handleError 解析 ClickHouse 错误
func (c *ClickHouseCracker) handleError(err error) (bool, error) {
	if err == nil {
		return true, nil
	}

	errMsg := err.Error()

	// 错误码参考: https://github.com/ClickHouse/ClickHouse/blob/master/src/Common/ErrorCodes.cpp
	
	// Code: 516. DB::Exception: Authentication failed
	// Code: 192. DB::Exception: Unknown user
	if strings.Contains(errMsg, "code: 516") || 
	   strings.Contains(errMsg, "code: 192") ||
	   strings.Contains(errMsg, "Authentication failed") ||
	   strings.Contains(errMsg, "Unknown user") {
		return false, nil // 密码或用户名错误
	}

	// Code: 210. DB::Exception: Connection refused
	// Code: 209. DB::Exception: Socket is not ready
	// i/o timeout
	if strings.Contains(errMsg, "connection refused") ||
	   strings.Contains(errMsg, "i/o timeout") ||
	   strings.Contains(errMsg, "network is unreachable") ||
	   strings.Contains(errMsg, "context deadline exceeded") {
		return false, brute.ErrConnectionFailed
	}

	// 其他未知错误，记录日志
	return false, brute.ErrConnectionFailed
}
