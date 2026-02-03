package protocol

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"neoagent/internal/core/scanner/brute"

	"github.com/go-sql-driver/mysql"
)

func TestMySQLCracker_HandleError(t *testing.T) {
	c := NewMySQLCracker()

	tests := []struct {
		name     string
		errInput error
		want     error
	}{
		{
			name: "Access Denied (Code 1045)",
			errInput: &mysql.MySQLError{
				Number:  1045,
				Message: "Access denied for user 'root'@'localhost'",
			},
			want: nil, // 认证失败
		},
		{
			name:     "Timeout",
			errInput: errors.New("dial tcp 1.2.3.4:3306: i/o timeout"),
			want:     brute.ErrConnectionFailed,
		},
		{
			name:     "Connection Refused",
			errInput: errors.New("dial tcp 127.0.0.1:3306: connect: connection refused"),
			want:     brute.ErrConnectionFailed,
		},
		{
			name:     "Bad Connection",
			errInput: mysql.ErrInvalidConn, // driver: bad connection
			want:     brute.ErrConnectionFailed,
		},
		{
			name:     "Unknown Error",
			errInput: errors.New("some weird error"),
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

func TestMySQLCracker_Check_NetworkError(t *testing.T) {
	c := NewMySQLCracker()

	// 尝试连接一个本地随机未监听端口
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close() // 关闭监听

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

// 模拟协议错误 (非 MySQL 协议)
func TestMySQLCracker_Check_ProtocolError(t *testing.T) {
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
			conn.Write([]byte("NOT MYSQL\n"))
			time.Sleep(100 * time.Millisecond)
		}
	}()

	c := NewMySQLCracker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	success, err := c.Check(ctx, "127.0.0.1", port, brute.Auth{Username: "root", Password: "123"})

	if success {
		t.Error("Expected failure, got success")
	}

	// MySQL 驱动通常会报 "packets.go:36: unexpected EOF" 或 "malformed packet"
	// handleError 应该将其归类为 ProtocolError (非 nil)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
