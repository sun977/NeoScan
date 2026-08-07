// Package api 提供 API 扫描能力的骨架。当前阶段仅搭建 ApiScanner 结构体与
// RunnerManager 所需的接口实现，真正的抓取与 JS 接口提取逻辑由后续实施文档
// 在本包内独立补充实现，不依赖 web 包或任何跨扫描器共享的爬虫模块（各原子
// 扫描器功能自成一体、互不依赖，见 web扫描模块重构文档.md）。
package api

import (
	"context"

	"neoagent/internal/core/model"
)

// ApiScanner 是独立于 WebScanner 的原子扫描器，与 WebScanner 平级。骨架阶段
// 不持有任何抓取相关的依赖（browser/limiter 等），待真正实现抓取能力时再由
// 本包自行决定需要哪些内部依赖。
type ApiScanner struct{}

// NewApiScanner 创建一个 ApiScanner 实例。
func NewApiScanner() *ApiScanner {
	return &ApiScanner{}
}

// Name 扫描器名称。
func (s *ApiScanner) Name() model.TaskType {
	return model.TaskTypeApiScan
}

// Run 执行 API 扫描任务。骨架阶段未实现任何抓取/分析逻辑，直接返回空结果，
// 待后续实施文档补充完整实现。
func (s *ApiScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	return nil, nil
}
