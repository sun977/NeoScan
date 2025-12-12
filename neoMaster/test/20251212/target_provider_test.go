package test

import (
	"context"
	"encoding/json"
	"testing"

	"neomaster/internal/pkg/matcher"
	"neomaster/internal/service/orchestrator/policy"
)

// MockSourceProvider 模拟源提供者
type MockSourceProvider struct {
	Targets []policy.Target
}

func (m *MockSourceProvider) Name() string { return "mock" }
func (m *MockSourceProvider) Provide(ctx context.Context, config policy.TargetSourceConfig, seedTargets []string) ([]policy.Target, error) {
	return m.Targets, nil
}
func (m *MockSourceProvider) HealthCheck(ctx context.Context) error { return nil }

func TestTargetProvider_Rules(t *testing.T) {
	// 1. 初始化 Provider
	targetProvider := policy.NewTargetProvider(nil)

	// 注册 Mock Provider
	mockTargets := []policy.Target{
		{Type: "ip", Value: "192.168.1.1", Source: "mock", Meta: map[string]string{"os": "linux", "device": "server"}},
		{Type: "ip", Value: "10.0.0.1", Source: "mock", Meta: map[string]string{"os": "windows", "device": "workstation"}},
		{Type: "domain", Value: "example.com", Source: "mock", Meta: map[string]string{"os": "linux", "device": "web"}},
		{Type: "url", Value: "http://test.com", Source: "mock", Meta: nil},
	}
	targetProvider.RegisterProvider("mock", &MockSourceProvider{Targets: mockTargets})

	ctx := context.Background()

	// 2. 测试 SkipRule (复杂规则)
	// 规则: Skip if (os == windows) OR (type == url)
	skipRule := matcher.MatchRule{
		Or: []matcher.MatchRule{
			{Field: "os", Operator: "equals", Value: "windows"}, // 访问提升后的 meta
			{Field: "type", Operator: "equals", Value: "url"},
		},
	}

	skipRuleJSON, _ := json.Marshal(skipRule)

	// 手动构造 JSON 字符串以确保准确性
	policyJSON := `{
		"target_sources": [{"source_type": "mock", "target_type": "ip"}],
		"skip_enabled": true,
		"skip_rule": ` + string(skipRuleJSON) + `
	}`

	targets, err := targetProvider.ResolveTargets(ctx, policyJSON, nil)
	if err != nil {
		t.Fatalf("ResolveTargets failed: %v", err)
	}

	// 预期结果:
	// 192.168.1.1 (linux, server) -> Keep
	// 10.0.0.1 (windows) -> Skip
	// example.com (linux, web) -> Keep
	// http://test.com (url) -> Skip

	if len(targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(targets))
		for _, tgt := range targets {
			t.Logf("Got target: %s (%s)", tgt.Value, tgt.Type)
		}
	}

	// 3. 测试 WhitelistRule
	// 规则: Whitelist if value starts_with "192.168"
	whitelistRule := matcher.MatchRule{
		Field:    "value",
		Operator: "starts_with",
		Value:    "192.168",
	}
	whitelistRuleJSON, _ := json.Marshal(whitelistRule)

	policyJSON2 := `{
		"target_sources": [{"source_type": "mock", "target_type": "ip"}],
		"whitelist_enabled": true,
		"whitelist_rule": ` + string(whitelistRuleJSON) + `
	}`

	targets2, err := targetProvider.ResolveTargets(ctx, policyJSON2, nil)
	if err != nil {
		t.Fatalf("ResolveTargets failed: %v", err)
	}

	// 预期结果: Only 192.168.1.1
	if len(targets2) != 1 {
		t.Errorf("Expected 1 target, got %d", len(targets2))
	}
	if len(targets2) > 0 && targets2[0].Value != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1, got %s", targets2[0].Value)
	}
}
