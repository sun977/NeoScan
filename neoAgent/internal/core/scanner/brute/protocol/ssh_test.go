package protocol

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"neoagent/internal/core/scanner/brute"
)

// mockSSHServer 启动一个简单的 SSH Server 用于测试
// 返回监听地址和关闭函数
func mockSSHServer(t *testing.T, user, pass string) (string, func()) {
	// config := &ssh.ServerConfig{
	// 	PasswordCallback: func(c ssh.ConnMetadata, passInput []byte) (*ssh.Permissions, error) {
	// 		if c.User() == user && string(passInput) == pass {
	// 			return nil, nil
	// 		}
	// 		return nil, fmt.Errorf("password rejected for %q", c.User())
	// 	},
	// }

	// 生成私钥 (为了测试速度，生成一个简单的 key)
	// 注意: 实际生成 RSA key 比较慢，这里为了测试方便，使用一个预生成的 key 或者快速生成
	// 为了避免依赖 key 生成库，我们尝试加载一个硬编码的测试 Key (如果不方便，就跳过 Server 测试)
	// 这里使用 ED25519 比较快
	// privateKey, err := rsa.GenerateKey(rand.Reader, 2048) ...
	// 为了简化，我们暂时只测试网络连接部分，或者跳过全流程测试如果环境不允许。
	// 但既然是单元测试，最好能跑通。
	// 鉴于生成 Key 代码较多，这里仅测试 "handleError" 逻辑和基本的 TCP 连通性。
	// 如果需要完整测试，需要引入 key 生成逻辑。

	// 方案 B: 只测试 handleError 和结构体逻辑，不启动真实 Server。
	// 方案 C: 尝试连接一个不存在的端口测试超时。
	return "", func() {}
}

func TestSSHCracker_HandleError(t *testing.T) {
	c := NewSSHCracker()

	tests := []struct {
		name     string
		errInput error
		want     error
	}{
		{
			name:     "Auth Failed",
			errInput: fmt.Errorf("ssh: handshake failed: ssh: unable to authenticate, attempted methods [none password], no supported methods remain"),
			want:     nil, // 认证失败返回 nil error, false
		},
		{
			name:     "Timeout",
			errInput: fmt.Errorf("dial tcp 1.2.3.4:22: i/o timeout"),
			want:     brute.ErrConnectionFailed,
		},
		{
			name:     "Connection Refused",
			errInput: fmt.Errorf("dial tcp 127.0.0.1:22: connect: connection refused"),
			want:     brute.ErrConnectionFailed,
		},
		{
			name:     "Unknown Error",
			errInput: fmt.Errorf("some weird error"),
			want:     brute.ErrProtocolError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.handleError(tt.errInput)
			if got != tt.want {
				t.Errorf("handleError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSHCracker_Check_NetworkError(t *testing.T) {
	c := NewSSHCracker()

	// 尝试连接一个本地随机未监听端口，应该返回 Connection Refused
	// 获取一个可用端口然后不监听它
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close() // 关闭监听，确保连接被拒绝

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	success, err := c.Check(ctx, "127.0.0.1", port, brute.Auth{Username: "root", Password: "123"})

	if success {
		t.Error("Expected failure, got success")
	}

	if err != brute.ErrConnectionFailed {
		t.Errorf("Expected ErrConnectionFailed, got %v", err)
	}
}

// 模拟一个简单的 TCP Server，只接受连接但不进行 SSH 握手，测试超时或协议错误
func TestSSHCracker_Check_ProtocolError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("Failed to listen")
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port

	// 启动一个 Dummy Server，发送垃圾数据
	go func() {
		conn, err1 := l.Accept()
		if err1 == nil {
			defer conn.Close()
			conn.Write([]byte("NOT SSH\n"))
			io.Copy(io.Discard, conn) // 读取并丢弃
		}
	}()

	c := NewSSHCracker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	success, err := c.Check(ctx, "127.0.0.1", port, brute.Auth{Username: "root", Password: "123"})

	if success {
		t.Error("Expected failure, got success")
	}

	// 这里可能会报 protocol error 或者 handshake failed
	// ssh.NewClientConn 遇到非 SSH 协议头会报错
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
