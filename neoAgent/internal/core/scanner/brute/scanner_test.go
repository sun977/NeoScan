package brute_test

import (
	"context"
	"testing"

	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/brute"
)

// MockCracker 模拟爆破器
type MockCracker struct {
	name        string
	mode        brute.AuthMode
	mockSuccess func(auth brute.Auth) bool
	mockError   func(auth brute.Auth) error
}

func (m *MockCracker) Name() string         { return m.name }
func (m *MockCracker) Mode() brute.AuthMode { return m.mode }
func (m *MockCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	if m.mockError != nil {
		if err := m.mockError(auth); err != nil {
			return false, err
		}
	}
	if m.mockSuccess != nil {
		return m.mockSuccess(auth), nil
	}
	return false, nil
}

func TestBruteScanner_Run(t *testing.T) {
	scanner := brute.NewBruteScanner()

	// 注册 Mock SSH Cracker
	sshCracker := &MockCracker{
		name: "ssh",
		mode: brute.AuthModeUserPass,
		mockSuccess: func(auth brute.Auth) bool {
			return auth.Username == "root" && auth.Password == "123456"
		},
	}
	scanner.RegisterCracker(sshCracker)

	// 构造任务
	task := &model.Task{
		ID:        "test-task-1",
		Type:      model.TaskTypeBrute,
		Target:    "127.0.0.1",
		PortRange: "22",
		Params: map[string]interface{}{
			"service":   "ssh",
			"users":     []string{"admin", "root"},
			"passwords": []string{"123456", "password"},
		},
	}

	// 执行
	ctx := context.Background()
	results, err := scanner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 TaskResult, got %d", len(results))
	}
	result := results[0]

	// 验证结果
	bruteResults, ok := result.Result.(brute.BruteResults)
	if !ok {
		// 尝试断言为 []model.BruteResult (兼容性)
		if list, ok := result.Result.([]model.BruteResult); ok {
			bruteResults = brute.BruteResults(list)
		} else {
			t.Fatalf("Result type assertion failed: expected BruteResults, got %T", result.Result)
		}
	}

	if len(bruteResults) != 1 {
		t.Errorf("Expected 1 brute result, got %d", len(bruteResults))
	} else {
		r := bruteResults[0]
		if !r.Success || r.Username != "root" || r.Password != "123456" {
			t.Errorf("Unexpected result: %+v", r)
		}
	}
}

func TestBruteScanner_Run_StopOnSuccess(t *testing.T) {
	scanner := brute.NewBruteScanner()

	// 模拟只要用户名是 root 就成功
	sshCracker := &MockCracker{
		name: "ssh",
		mode: brute.AuthModeUserPass,
		mockSuccess: func(auth brute.Auth) bool {
			return auth.Username == "root"
		},
	}
	scanner.RegisterCracker(sshCracker)

	task := &model.Task{
		ID:        "test-task-stop",
		Type:      model.TaskTypeBrute,
		Target:    "127.0.0.1",
		PortRange: "22",
		Params: map[string]interface{}{
			"service":         "ssh",
			"users":           []string{"root", "admin"}, // root 先匹配
			"passwords":       []string{"123456", "password"},
			"stop_on_success": true,
		},
	}

	results, _ := scanner.Run(context.Background(), task)
	bruteResults := results[0].Result.(brute.BruteResults)

	// 应该只返回第一个成功的 (root/123456)
	// 因为 stop_on_success=true，找到一个就停止
	if len(bruteResults) != 1 {
		t.Errorf("Expected 1 result (stop on success), got %d", len(bruteResults))
	}
}

func TestBruteScanner_Run_NetworkError(t *testing.T) {
	scanner := brute.NewBruteScanner()

	// Mock 网络错误的 Cracker
	errCracker := &MockCracker{
		name: "mysql",
		mode: brute.AuthModeUserPass,
		mockError: func(auth brute.Auth) error {
			return brute.ErrConnectionFailed
		},
	}
	scanner.RegisterCracker(errCracker)

	task := &model.Task{
		ID:        "test-task-2",
		Type:      model.TaskTypeBrute,
		Target:    "127.0.0.1",
		PortRange: "3306",
		Params: map[string]interface{}{
			"service": "mysql",
		},
	}

	// 即使全是网络错误，Task 本身应该成功完成（返回空结果或错误日志），而不是报错退出
	// 除非是 Context Cancel
	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 TaskResult, got %d", len(results))
	}
	result := results[0]

	bruteResults := result.Result.(brute.BruteResults)
	if len(bruteResults) != 0 {
		t.Errorf("Expected 0 results, got %d", len(bruteResults))
	}

	// 验证限流器是否感知到失败
	// currentLimit 应该下降 (初始 50 -> 35 -> ...)
	if scanner.CurrentLimit() >= 50 {
		t.Errorf("Limiter should decrease on failure, got %d", scanner.CurrentLimit())
	}
}
