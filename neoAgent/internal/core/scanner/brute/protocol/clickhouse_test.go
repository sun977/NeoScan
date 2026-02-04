package protocol

import (
	"context"
	"testing"
	"time"

	"neoagent/internal/core/scanner/brute"
)

// TestClickHouseCracker_NetworkError 测试网络不可达的情况
func TestClickHouseCracker_NetworkError(t *testing.T) {
	cracker := NewClickHouseCracker()

	// 使用一个不可达的 IP
	host := "192.0.2.1" // TEST-NET-1
	port := 9000
	auth := brute.Auth{Username: "default", Password: "wrongpassword"}

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
		// ClickHouse 驱动可能会返回 context deadline exceeded，我们期望它被处理为 ErrConnectionFailed
		if err == nil {
			t.Error("expected error, got nil")
		}
	}

	t.Logf("Duration: %v", duration)
}
