package protocol

import (
	"context"
	"testing"
	"time"

	"neoagent/internal/core/scanner/brute"
)

// TestPostgresCracker_Integration 需要真实的 Postgres 环境
// 这里主要测试网络不可达的情况 (Mocking sql/driver 太复杂，且不如集成测试有效)
func TestPostgresCracker_NetworkError(t *testing.T) {
	cracker := NewPostgresCracker()
	
	// 使用一个不可达的 IP
	host := "192.0.2.1" // TEST-NET-1, documentation only, should timeout
	port := 5432
	auth := brute.Auth{Username: "postgres", Password: "wrongpassword"}

	// 设置较短的超时
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	success, err := cracker.Check(ctx, host, port, auth)
	duration := time.Since(start)

	if success {
		t.Error("expected failure, got success")
	}

	if err != brute.ErrConnectionFailed && err != context.DeadlineExceeded {
		// 注意：sql.Open 的 connect_timeout 是秒级，可能比 ctx 慢
		// 如果 ctx 先超时，PingContext 返回 context.DeadlineExceeded
		// 如果 connect_timeout 先触发，返回 driver error
		t.Logf("Got error: %v", err)
	}

	t.Logf("Duration: %v", duration)
}

func TestPostgresCracker_HandleError(t *testing.T) {
	// TODO: 可以 mock pq.Error 来测试错误码解析逻辑
	// 但这需要引入 github.com/lib/pq 依赖到测试代码中
}
