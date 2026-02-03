package protocol

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"neoagent/internal/core/scanner/brute"

	"golang.org/x/crypto/ssh"
)

// SSHCracker 实现 SSH 协议爆破
type SSHCracker struct{}

// NewSSHCracker 创建 SSH 爆破器
func NewSSHCracker() *SSHCracker {
	return &SSHCracker{}
}

// Name 返回协议名称
func (c *SSHCracker) Name() string {
	return "ssh"
}

// Mode 返回爆破模式 (需要用户名和密码)
func (c *SSHCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

// Check 验证 SSH 凭据
func (c *SSHCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	config := &ssh.ClientConfig{
		User: auth.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(auth.Password),
		},
		// 必须忽略 HostKey 检查，否则无法连接未知主机
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// 设置连接超时
		Timeout: 3 * time.Second, // 默认 3 秒连接超时
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	// 使用 DialTimeout 进行连接，但我们也需要尊重传入的 context
	// 由于 ssh.Dial 内部使用 net.DialTimeout，我们可以尝试先手动建立 TCP 连接
	// 这样可以更好地控制超时和 Context 取消

	// 1. 建立 TCP 连接 (受 Context 控制)
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, c.handleError(err)
	}
	defer conn.Close()

	// 2. 建立 SSH 连接 (在已有的 TCP 连接上)
	// ssh.NewClientConn 会进行协议握手和认证
	// 注意: NewClientConn 内部有超时机制 (依赖 config.Timeout)，但也受 conn 的读写截止时间影响
	// 我们可以设置一个较短的 Deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second) // 默认总超时
	}
	conn.SetDeadline(deadline)

	cConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return false, c.handleError(err)
	}
	defer cConn.Close()

	// 如果能成功建立连接，说明认证成功
	// 清理资源 (虽然 defer 会处理，但显式关闭是个好习惯)
	// 注意：必须处理 chans 和 reqs，否则 goroutine 可能泄露，但这里我们只是验证认证，直接关闭即可
	go ssh.DiscardRequests(reqs)
	go func() {
		for newChannel := range chans {
			newChannel.Reject(ssh.Prohibited, "plugin does not allow channels")
		}
	}()

	return true, nil
}

// handleError 将底层错误转换为标准错误
func (c *SSHCracker) handleError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()

	// 认证失败通常包含 "unable to authenticate"
	if strings.Contains(msg, "unable to authenticate") {
		return nil // 认证失败不是系统错误，返回 (false, nil)
	}

	// 网络/连接错误
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no route to host") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "handshake failed") ||
		strings.Contains(msg, "target machine actively refused") { // Windows error message
		return brute.ErrConnectionFailed
	}

	// 其他协议错误 (例如版本不匹配)
	return brute.ErrProtocolError
}
