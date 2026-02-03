package protocol

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"neoagent/internal/core/scanner/brute"
)

func TestRedisCracker_HandleError(t *testing.T) {
	c := NewRedisCracker()

	tests := []struct {
		name     string
		errInput error
		want     error
	}{
		{
			name:     "Invalid Password",
			errInput: errors.New("ERR invalid password"),
			want:     nil, // 认证失败
		},
		{
			name:     "WRONGPASS",
			errInput: errors.New("WRONGPASS invalid username-password pair"),
			want:     nil,
		},
		{
			name:     "NOAUTH",
			errInput: errors.New("NOAUTH Authentication required."),
			want:     nil,
		},
		{
			name:     "Timeout",
			errInput: errors.New("dial tcp 1.2.3.4:6379: i/o timeout"),
			want:     brute.ErrConnectionFailed,
		},
		{
			name:     "Connection Refused",
			errInput: errors.New("dial tcp 127.0.0.1:6379: connect: connection refused"),
			want:     brute.ErrConnectionFailed,
		},
		{
			name:     "Protocol Error (HTTP Response)",
			errInput: errors.New("redis: invalid response: HTTP/1.1 400 Bad Request"),
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

func TestRedisCracker_Check_NetworkError(t *testing.T) {
	c := NewRedisCracker()

	// 尝试连接一个本地随机未监听端口
	// l, _ := net.Listen("tcp", "127.0.0.1:0")
	// port := l.Addr().(*net.TCPAddr).Port
	// l.Close() // 关闭监听

	// 为了确保端口真的关闭，我们使用一个极不可能开放的端口
	// 或者，我们只需要知道它会返回错误。
	// go-redis 的 Dial 逻辑比较复杂，包含重试和连接池
	// 我们直接 Mock 一下或者只测试 handleError 逻辑可能更好，但集成测试也很有价值。

	// 使用一个随机端口
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// go-redis 会尝试重试，可能会比较慢，我们在 Check 里面设置了 DialTimeout=3s
	// 这里 context 也是 1s
	success, err := c.Check(ctx, "127.0.0.1", port, brute.Auth{Password: "123"})

	if success {
		t.Error("Expected failure, got success")
	}

	// 此时 err 应该是 "failed to dial... connectex..."
	// handleError 应该返回 ErrConnectionFailed
	if err != brute.ErrConnectionFailed {
		t.Errorf("Expected ErrConnectionFailed, got %v", err)
	}
}

// 模拟协议错误 (非 Redis 协议)
func TestRedisCracker_Check_ProtocolError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("Failed to listen")
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port

	// 启动一个 Dummy Server，发送垃圾数据
	go func() {
		conn, err := l.Accept()
		if err == nil {
			defer conn.Close()
			conn.Write([]byte("NOT REDIS\n"))
			time.Sleep(100 * time.Millisecond)
		}
	}()

	c := NewRedisCracker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	success, err := c.Check(ctx, "127.0.0.1", port, brute.Auth{Password: "123"})

	if success {
		t.Error("Expected failure, got success")
	}

	// go-redis 遇到非法响应会报错
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
