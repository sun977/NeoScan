package factory

import (
	"neoagent/internal/core/scanner/os"
)

// NewOsScanner 创建操作系统扫描器
// 注意: 返回的 *os.Scanner 并不直接实现 Runner 接口，需要 OsRunner 适配
func NewOsScanner() *os.Scanner {
	return os.NewScanner()
}
