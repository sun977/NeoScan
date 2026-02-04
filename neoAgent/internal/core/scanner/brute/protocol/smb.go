package protocol

import (
	"context"
	"strings"

	"neoagent/internal/core/scanner/brute"

	"github.com/stacktitan/smb/smb"
)

// SMBCracker SMB 协议爆破器
type SMBCracker struct{}

func NewSMBCracker() *SMBCracker {
	return &SMBCracker{}
}

func (c *SMBCracker) Name() string {
	return "smb"
}

func (c *SMBCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

func (c *SMBCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	options := smb.Options{
		Host:        host,
		Port:        port,
		User:        auth.Username,
		Password:    auth.Password,
		Domain:      "", // 默认空域，如果需要支持域爆破，后续可在 Auth 结构体扩展
		Workstation: "",
	}

	// 结果通道
	type result struct {
		success bool
		err     error
	}
	resultChan := make(chan result, 1)

	// 在 goroutine 中执行，配合 select 实现超时控制
	go func() {
		// stacktitan/smb 的 NewSession 是同步阻塞的
		session, err := smb.NewSession(options, false)
		if err != nil {
			resultChan <- result{false, err}
			return
		}
		defer session.Close()

		if session.IsAuthenticated {
			resultChan <- result{true, nil}
		} else {
			resultChan <- result{false, nil} // 鉴权失败
		}
	}()

	select {
	case <-ctx.Done():
		// 上下文超时或取消
		// 注意：这里的 goroutine 可能会泄露，因为 stacktitan/smb 不支持 context 取消
		// 这是一个已知问题，但在短连接爆破场景下通常可接受
		return false, brute.ErrConnectionFailed
	case res := <-resultChan:
		if res.success {
			return true, nil
		}
		return c.handleError(res.err)
	}
}

// handleError 解析 SMB 错误
func (c *SMBCracker) handleError(err error) (bool, error) {
	if err == nil {
		return false, nil // 鉴权失败 (IsAuthenticated == false)
	}

	errMsg := err.Error()

	// 鉴权失败
	// "login failed" 是 qscan 代码中定义的，库本身可能返回不同的错误
	// stacktitan/smb 通常在鉴权失败时返回 error，或者 IsAuthenticated=false
	// 常见的 NTLM 错误:
	// "STATUS_LOGON_FAILURE"
	// "STATUS_WRONG_PASSWORD"
	if strings.Contains(errMsg, "STATUS_LOGON_FAILURE") ||
		strings.Contains(errMsg, "STATUS_WRONG_PASSWORD") ||
		strings.Contains(errMsg, "login failed") {
		return false, nil
	}

	// 连接错误
	// "connection refused"
	// "i/o timeout"
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "EOF") {
		return false, brute.ErrConnectionFailed
	}

	// 默认视为连接失败，避免误报
	return false, brute.ErrConnectionFailed
}
