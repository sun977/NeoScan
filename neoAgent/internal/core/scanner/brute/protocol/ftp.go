package protocol

import (
	"context"
	"fmt"
	"time"

	"neoagent/internal/core/scanner/brute"

	"github.com/jlaffaye/ftp"
)

// FTPCracker FTP 协议爆破器
type FTPCracker struct{}

func NewFTPCracker() *FTPCracker {
	return &FTPCracker{}
}

func (c *FTPCracker) Name() string {
	return "ftp"
}

func (c *FTPCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

func (c *FTPCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// 使用 DialTimeout 建立连接
	// 注意：jlaffaye/ftp 的 DialTimeout 只控制 TCP 连接超时
	addr := fmt.Sprintf("%s:%d", host, port)

	// 连接超时 (通常 5s)
	// 由于 Check 外部已有 ctx 控制，这里的 Timeout 应该小于 ctx 的 Deadline
	// 但 ftp 库不支持 ctx，只能用 Timeout
	conn, err := ftp.DialTimeout(addr, 5*time.Second)
	if err != nil {
		return false, brute.ErrConnectionFailed
	}
	defer conn.Quit()

	// 登录
	if err := conn.Login(auth.Username, auth.Password); err != nil {
		return c.handleError(err)
	}

	// 登出 (Quit 在 defer 中也会调用，但 Login 成功后显式 Logout 是好习惯)
	conn.Logout()

	return true, nil
}

// handleError 解析 FTP 错误
func (c *FTPCracker) handleError(err error) (bool, error) {
	if err == nil {
		return true, nil
	}

	errMsg := err.Error()

	// 530 Login incorrect.
	// 530 Not logged in.
	if len(errMsg) >= 3 && errMsg[:3] == "530" {
		return false, nil // 密码错误
	}

	// 421 Service not available, closing control connection.
	// 421 Too many connections (8) from this IP
	if len(errMsg) >= 3 && errMsg[:3] == "421" {
		return false, brute.ErrConnectionFailed
	}

	// 网络错误
	if len(errMsg) >= 3 && errMsg[:3] == "EOF" {
		return false, brute.ErrConnectionFailed
	}

	// 其他未知错误，记录日志
	return false, err
}
