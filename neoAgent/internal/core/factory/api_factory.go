package factory

import (
	"neoagent/internal/core/scanner/api"
)

// NewApiScanner 创建一个标准的 API 扫描器（当前为骨架阶段，不持有任何
// 抓取相关依赖，真正实现时再按需引入）。
func NewApiScanner() *api.ApiScanner {
	return api.NewApiScanner()
}
