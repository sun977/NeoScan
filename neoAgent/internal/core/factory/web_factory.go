package factory

import (
	"neoagent/internal/core/scanner/web"
)

// NewWebScanner 创建一个标准的 Web 扫描器
// 预配置了浏览器管理器、自适应限流器和指纹引擎
func NewWebScanner() *web.WebScanner {
	// 使用 web 包自身的构造函数，它内部已经处理了 BrowserManager 和 Limiter 的初始化
	// 如果未来有全局配置（如最大并发数、代理等），应在这里传入
	scanner := web.NewWebScanner()
	return scanner
}
