// FingerprintMatcher 指纹识别与标准化接口
// 职责: 将杂乱的工具指纹 (Nmap/Masscan/Nuclei) 统一映射为 CPE 标准格式
package etl

import (
	"context"

	"neomaster/internal/service/fingerprint"
)

// MatchContext 指纹匹配上下文
// Deprecated: Use fingerprint.Input instead
type MatchContext = fingerprint.Input

// FingerprintMatcher 指纹识别接口
// Deprecated: Use fingerprint.Service instead
type FingerprintMatcher interface {
	Match(ctx context.Context, input MatchContext) (*Fingerprint, error)
	LoadRules(path string, ruleType string) error
}

// Fingerprint 标准指纹结构
// Deprecated: Use fingerprint.Match instead
type Fingerprint struct {
	Product string
	Version string
	Vendor  string
	CPE     string
	Type    string
}
