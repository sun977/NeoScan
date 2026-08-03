package edge

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTestRules 在临时目录下写入一份测试专属的 CDN 规则固件，返回文件路径。
// 不依赖 rules/edge/cdn.json 的生产数据，避免生产规则变更影响测试稳定性。
func writeTestRules(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cdn.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test rules file: %v", err)
	}
	return path
}

const testRulesFixture = `[
	{
		"provider": "TestCDN-A",
		"cidrs": ["1.1.1.0/24"]
	},
	{
		"provider": "TestCDN-B",
		"cidrs": ["2.2.2.0/24", "3.3.3.0/24"]
	}
]`

// 场景 1：Load 成功后，Check 命中网段内 IP，返回 (true, 对应厂商名)。
func TestDetector_Check_Hit(t *testing.T) {
	d := NewDetector()
	path := writeTestRules(t, testRulesFixture)

	if err := d.Load(path); err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	hit, provider := d.Check("1.1.1.100")
	if !hit || provider != "TestCDN-A" {
		t.Fatalf("Check(1.1.1.100) = (%v, %q), want (true, \"TestCDN-A\")", hit, provider)
	}

	// 第二个厂商、第二条 CIDR 也要能命中，确认遍历逻辑没有漏掉。
	hit, provider = d.Check("3.3.3.1")
	if !hit || provider != "TestCDN-B" {
		t.Fatalf("Check(3.3.3.1) = (%v, %q), want (true, \"TestCDN-B\")", hit, provider)
	}
}

// 场景 2：Check 传入不在任何网段的 IP，返回 (false, "")。
func TestDetector_Check_Miss(t *testing.T) {
	d := NewDetector()
	path := writeTestRules(t, testRulesFixture)

	if err := d.Load(path); err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	hit, provider := d.Check("8.8.8.8")
	if hit || provider != "" {
		t.Fatalf("Check(8.8.8.8) = (%v, %q), want (false, \"\")", hit, provider)
	}
}

// 场景 3：Check 传入非法 IP 字符串，返回 (false, "")，不 panic。
func TestDetector_Check_InvalidIP(t *testing.T) {
	d := NewDetector()
	path := writeTestRules(t, testRulesFixture)

	if err := d.Load(path); err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	hit, provider := d.Check("not-an-ip")
	if hit || provider != "" {
		t.Fatalf("Check(\"not-an-ip\") = (%v, %q), want (false, \"\")", hit, provider)
	}
}

// 场景 4：Load 传入不存在的文件路径，返回 error；此时 Check 仍然安全
// 返回 (false, "")，不影响调用方现有流程。
func TestDetector_Load_FileNotExist(t *testing.T) {
	d := NewDetector()

	err := d.Load(filepath.Join(t.TempDir(), "not-exist.json"))
	if err == nil {
		t.Fatal("Load() with non-existent path should return error, got nil")
	}

	hit, provider := d.Check("1.1.1.100")
	if hit || provider != "" {
		t.Fatalf("Check() after failed Load() = (%v, %q), want (false, \"\")", hit, provider)
	}
}

// 场景 4 补充：规则文件存在但 JSON 格式非法，同样返回 error 而不是 panic。
func TestDetector_Load_InvalidJSON(t *testing.T) {
	d := NewDetector()
	path := writeTestRules(t, `{not valid json`)

	if err := d.Load(path); err == nil {
		t.Fatal("Load() with invalid JSON should return error, got nil")
	}

	hit, provider := d.Check("1.1.1.100")
	if hit || provider != "" {
		t.Fatalf("Check() after failed Load() = (%v, %q), want (false, \"\")", hit, provider)
	}
}

// 场景 5：Load 被调用两次（模拟未来的 Reload 场景），第二次传入的规则
// 完全替换第一次的结果——验证 Load 不是只能调用一次，且是完整替换
// 而不是增量合并。
func TestDetector_Load_Reload_FullyReplaces(t *testing.T) {
	d := NewDetector()

	firstPath := writeTestRules(t, testRulesFixture)
	if err := d.Load(firstPath); err != nil {
		t.Fatalf("first Load() unexpected error: %v", err)
	}

	// 第一次加载后，1.1.1.0/24 应该能命中。
	if hit, _ := d.Check("1.1.1.100"); !hit {
		t.Fatal("expected hit after first Load(), got miss")
	}

	secondPath := writeTestRules(t, `[
		{
			"provider": "TestCDN-C",
			"cidrs": ["9.9.9.0/24"]
		}
	]`)
	if err := d.Load(secondPath); err != nil {
		t.Fatalf("second Load() unexpected error: %v", err)
	}

	// 第二次加载后，旧规则（1.1.1.0/24）必须失效，不是追加合并。
	if hit, _ := d.Check("1.1.1.100"); hit {
		t.Fatal("expected old rule to be replaced after second Load(), but still hit")
	}

	// 新规则（9.9.9.0/24）必须生效。
	hit, provider := d.Check("9.9.9.1")
	if !hit || provider != "TestCDN-C" {
		t.Fatalf("Check(9.9.9.1) after second Load() = (%v, %q), want (true, \"TestCDN-C\")", hit, provider)
	}
}

// 额外场景：从未调用过 Load 的全新 Detector，Check 必须安全返回
// (false, "")，不能因为规则集为 nil 而 panic——对应 NewDetector 的注释承诺。
func TestDetector_Check_BeforeLoad(t *testing.T) {
	d := NewDetector()

	hit, provider := d.Check("1.1.1.100")
	if hit || provider != "" {
		t.Fatalf("Check() before any Load() = (%v, %q), want (false, \"\")", hit, provider)
	}
}
