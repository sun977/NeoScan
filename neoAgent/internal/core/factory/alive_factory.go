package factory

import (
	"neoagent/internal/core/scanner/alive"
)

// NewAliveScanner 创建存活扫描器
// 返回的 IpAliveScanner 实现了 Runner 接口
func NewAliveScanner() *alive.IpAliveScanner {
	return alive.NewIpAliveScanner()
}
