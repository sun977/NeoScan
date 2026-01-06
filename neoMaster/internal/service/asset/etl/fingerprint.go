// FingerprintMatcher 指纹识别与标准化接口
// 职责: 将杂乱的工具指纹 (Nmap/Masscan/Nuclei) 统一映射为 CPE 标准格式
package etl

import (
	"context"
)

// FingerprintMatcher 指纹识别接口
type FingerprintMatcher interface {
	// Match 将原始 Banner/Service 字符串映射为标准指纹
	// input: 如 "nginx/1.18.0 (Ubuntu)"
	// output: Product="nginx", Version="1.18.0", Vendor="F5", CPE="..."
	Match(ctx context.Context, input string) (*Fingerprint, error)
}

// Fingerprint 标准指纹结构
type Fingerprint struct {
	Product string
	Version string
	Vendor  string
	CPE     string
	Type    string // os, app, hardware
}
