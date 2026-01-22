/**
 * 结果上报接口定义
 * @author: Sun977
 * @date: 2026.01.21
 * @description: 定义结果输出的通用接口，解耦 Cluster/Console/File 输出。
 */

package reporter

import (
	"context"
	"neoagent/internal/core/model"
)

// TabularData 是一个可以被渲染为表格的数据接口
// 任何想要在控制台漂亮打印的 Result 都应该实现此接口
type TabularData interface {
	Headers() []string
	Rows() [][]string
}

// Reporter 定义结果上报的行为
type Reporter interface {
	// Report 上报/输出任务结果
	Report(ctx context.Context, result *model.TaskResult) error
}

// MultiReporter 支持同时向多个目标上报 (e.g., Console + File)
type MultiReporter struct {
	reporters []Reporter
}

func NewMultiReporter(reporters ...Reporter) *MultiReporter {
	return &MultiReporter{
		reporters: reporters,
	}
}

func (m *MultiReporter) Report(ctx context.Context, result *model.TaskResult) error {
	var errs []error
	for _, r := range m.reporters {
		if err := r.Report(ctx, result); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0] // 简单返回第一个错误，实际可能需要聚合错误
	}
	return nil
}
