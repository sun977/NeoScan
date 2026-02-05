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
}

func NewScanRunOptions() *ScanRunOptions {
	return &ScanRunOptions{
		Concurrency: 10,
		PortRange:   "top1000",
		EnableBrute: false, // 默认不开启爆破
	}
}

func (o *ScanRunOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}
