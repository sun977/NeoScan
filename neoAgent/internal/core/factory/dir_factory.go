package factory

import (
	"neoagent/internal/core/scanner/dir"
)

// NewDirScanner 创建一个标准的目录扫描器。签名无参数，与 RunnerManager
// 的注册调用一致，见 docs/原子模块设计/目录扫描设计/DirScanner开发实施文档.md Task 4.1。
func NewDirScanner() *dir.DirScanner {
	return dir.NewDirScanner()
}
