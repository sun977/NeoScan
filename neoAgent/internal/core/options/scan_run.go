package options

import (
	"fmt"
)

// ScanRunOptions 定义 run 命令的参数 (Core Level)
// 这个结构体用于将 CLI 参数传递给 Pipeline
type ScanRunOptions struct {
	Target      string
	Concurrency int
	PortRange   string
	ShowSummary bool

	// 爆破相关选项
	EnableBrute bool   // --brute
	BruteUsers  string // --users
	BrutePass   string // --pass

	// Web 扫描相关选项
	NoWeb         bool // --no-web (默认自动扫描 Web，此标志用于禁用)
	WebScreenshot bool // --screenshot (默认关闭，需显式开启)
}

func NewScanRunOptions() *ScanRunOptions {
	return &ScanRunOptions{
		Concurrency:   10,        // 默认 10 个 IP 并发
		PortRange:     "top1000", // 默认 top1000
		EnableBrute:   false,     // 默认关闭爆破
		NoWeb:         false,     // 默认开启 Web 扫描
		WebScreenshot: false,     // 默认关闭截图
	}
}

func (o *ScanRunOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}
