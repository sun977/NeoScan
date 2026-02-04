package protocol

import (
	"context"
	"testing"
	"time"

	"neoagent/internal/core/scanner/brute"
)

// TestOracleCracker_NetworkError 测试网络不可达的情况
func TestOracleCracker_NetworkError(t *testing.T) {
	cracker := NewOracleCracker()
	
	// 使用一个不可达的 IP
	host := "192.0.2.1" // TEST-NET-1
	port := 1521
	auth := brute.Auth{Username: "system", Password: "wrongpassword"}

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
		if err == nil {
			t.Error("expected error, got nil")
		}
	}

	t.Logf("Duration: %v", duration)
}
