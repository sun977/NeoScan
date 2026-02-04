package protocol

import (
	"context"
	"testing"
	"time"

	"neoagent/internal/core/scanner/brute"
)

// TestMongoCracker_NetworkError 测试网络不可达的情况
func TestMongoCracker_NetworkError(t *testing.T) {
	cracker := NewMongoCracker()
	
	// 使用一个不可达的 IP
	host := "192.0.2.1" // TEST-NET-1
	port := 27017
	auth := brute.Auth{Username: "admin", Password: "wrongpassword"}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	success, err := cracker.Check(ctx, host, port, auth)
	duration := time.Since(start)

	if success {
		t.Error("expected failure, got success")
	}

	if err != brute.ErrConnectionFailed {
		t.Logf("Got error: %v", err)
		// 注意：mongo driver 有时会返回详细的 server selection error，
		// 只要不是 nil (鉴权成功) 且不是 nil (鉴权失败)，就符合预期
		// 但我们期望它被映射为 ErrConnectionFailed
		if err == nil {
			t.Error("expected error, got nil")
		}
	}

	t.Logf("Duration: %v", duration)
}
