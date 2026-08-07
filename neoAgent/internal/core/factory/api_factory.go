package factory

import (
	"neoagent/internal/core/scanner/api"
)

// NewApiScanner 创建一个标准的 API 扫描器
// 预配置了浏览器启动器与自适应限流器，模式对齐 web_factory.go
func NewApiScanner() *api.ApiScanner {
	return api.NewApiScanner()
}
