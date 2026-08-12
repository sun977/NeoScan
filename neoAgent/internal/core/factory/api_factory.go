package factory

import (
	"neoagent/internal/core/scanner/api"
)

// NewApiScanner 创建一个标准的 API 扫描器。签名无参数，与 RunnerManager
// 的注册调用一致，见 docs/API扫描-js提取/API扫描实施文档.md 第十二节。
func NewApiScanner() *api.ApiScanner {
	return api.NewApiScanner()
}
