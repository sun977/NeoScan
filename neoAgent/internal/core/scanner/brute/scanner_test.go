package brute

import (
	"context"
	"testing"

	"neoagent/internal/core/model"
)

// MockCracker 模拟爆破器
type MockCracker struct {
	name        string
	mode        AuthMode
	mockSuccess func(auth Auth) bool
	mockError   func(auth Auth) error
}

func (m *MockCracker) Name() string   { return m.name }
func (m *MockCracker) Mode() AuthMode { return m.mode }
func (m *MockCracker) Check(ctx context.Context, host string, port int, auth Auth) (bool, error) {
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
	scanner := NewBruteScanner()

	// 注册 Mock SSH Cracker
	sshCracker := &MockCracker{
		name: "ssh",
		mode: AuthModeUserPass,
		mockSuccess: func(auth Auth) bool {
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
	bruteResults, ok := result.Result.(BruteResults)
	if !ok {
		// 尝试断言为 []BruteResult (兼容性)
		if list, ok := result.Result.([]BruteResult); ok {
			bruteResults = BruteResults(list)
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
	scanner := NewBruteScanner()

	// 模拟只要用户名是 root 就成功
	sshCracker := &MockCracker{
		name: "ssh",
		mode: AuthModeUserPass,
		mockSuccess: func(auth Auth) bool {
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
	bruteResults := results[0].Result.(BruteResults)

	// 应该只返回第一个成功的 (root/123456)
	// 因为 stop_on_success=true，找到一个就停止
	if len(bruteResults) != 1 {
		t.Errorf("Expected 1 result (stop on success), got %d", len(bruteResults))
	}
}

func TestBruteScanner_Run_NetworkError(t *testing.T) {
	scanner := NewBruteScanner()

	// Mock 网络错误的 Cracker
	errCracker := &MockCracker{
		name: "mysql",
		mode: AuthModeUserPass,
		mockError: func(auth Auth) error {
			return ErrConnectionFailed
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

	bruteResults := result.Result.(BruteResults)
	if len(bruteResults) != 0 {
		t.Errorf("Expected 0 results, got %d", len(bruteResults))
	}

	// 验证限流器是否感知到失败
	// currentLimit 应该下降 (初始 50 -> 35 -> ...)
	if scanner.globalLimit.CurrentLimit() >= 50 {
		t.Errorf("Limiter should decrease on failure, got %d", scanner.globalLimit.CurrentLimit())
	}
}
