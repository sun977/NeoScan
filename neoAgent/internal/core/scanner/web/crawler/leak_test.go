package crawler

import "testing"

// TestDetectLeaks_AWSAccessKey 验证 AWS AccessKey（AKIA 前缀 + 16 位大写字母数字）
// 能被正确识别，且返回的 Match 已经过脱敏（不能是原始明文）。
func TestDetectLeaks_AWSAccessKey(t *testing.T) {
	raw := "AKIAABCDEFGHIJKLMNOP"
	body := `<script>var ak = "` + raw + `";</script>`

	leaks := DetectLeaks(body)

	if len(leaks) != 1 {
		t.Fatalf("expected 1 leak, got %d: %v", len(leaks), leaks)
	}
	if leaks[0].Type != "aws_ak" {
		t.Fatalf("expected type=aws_ak, got %s", leaks[0].Type)
	}
	if leaks[0].Match == raw {
		t.Fatalf("Match must be masked, but got raw plaintext: %s", leaks[0].Match)
	}
	if leaks[0].Match != "AKIA****MNOP" {
		t.Fatalf("expected masked value AKIA****MNOP, got %s", leaks[0].Match)
	}
}

// TestDetectLeaks_AliyunAccessKey 验证阿里云 AccessKey（LTAI 前缀）能被识别。
func TestDetectLeaks_AliyunAccessKey(t *testing.T) {
	raw := "LTAI4G1234567890abcd"
	body := "config: " + raw

	leaks := DetectLeaks(body)

	if len(leaks) != 1 || leaks[0].Type != "aliyun_ak" {
		t.Fatalf("expected 1 leak of type aliyun_ak, got %v", leaks)
	}
}

// TestDetectLeaks_JWT 验证 JWT（eyJ 开头的三段式 Base64 结构）能被识别。
func TestDetectLeaks_JWT(t *testing.T) {
	raw := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	body := `Authorization: Bearer ` + raw

	leaks := DetectLeaks(body)

	if len(leaks) != 1 || leaks[0].Type != "jwt" {
		t.Fatalf("expected 1 leak of type jwt, got %v", leaks)
	}
}

// TestDetectLeaks_InternalIP 验证三个 RFC 1918 私有网段都能被识别，
// 同时验证一个公网 IP（8.8.8.8）不会被误报。
func TestDetectLeaks_InternalIP(t *testing.T) {
	body := `10.0.0.1 172.16.5.20 192.168.1.100 8.8.8.8`

	leaks := DetectLeaks(body)

	if len(leaks) != 3 {
		t.Fatalf("expected 3 internal_ip leaks (public IP must not match), got %d: %v", len(leaks), leaks)
	}
	for _, l := range leaks {
		if l.Type != "internal_ip" {
			t.Fatalf("expected type=internal_ip, got %s", l.Type)
		}
	}
}

// TestDetectLeaks_NoFalsePositive 验证一段完全不含任何敏感信息的普通页面
// 不会产生任何误报命中。
func TestDetectLeaks_NoFalsePositive(t *testing.T) {
	body := `<html><body><p>Hello World, this is a normal page.</p></body></html>`

	leaks := DetectLeaks(body)

	if len(leaks) != 0 {
		t.Fatalf("expected 0 leaks on clean page, got %d: %v", len(leaks), leaks)
	}
}

// TestMask_ShortStringFullyRedacted 验证 mask 对短字符串（长度 <=8）的处理：
// 必须整体替换为 "****"，不能因为"前4后4"策略导致原文被完整保留或重复暴露。
func TestMask_ShortStringFullyRedacted(t *testing.T) {
	cases := []string{"", "a", "1234", "12345678"} // 最后一个长度正好=8，属于边界值
	for _, s := range cases {
		got := mask(s)
		if got != "****" {
			t.Fatalf("mask(%q) = %q, want \"****\" (short string must be fully redacted)", s, got)
		}
	}
}

// TestMask_LongStringKeepsHeadAndTail 验证长字符串脱敏后保留头尾各 4 位，
// 中间替换为 "****"，既能核实类型又不泄露完整密钥。
func TestMask_LongStringKeepsHeadAndTail(t *testing.T) {
	got := mask("AKIAABCDEFGHIJKLMNOP")
	want := "AKIA****MNOP"
	if got != want {
		t.Fatalf("mask() = %q, want %q", got, want)
	}
}
