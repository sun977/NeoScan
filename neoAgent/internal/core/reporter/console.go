package reporter

import (
	"context"
	"fmt"

	"neoagent/internal/core/model"

	"github.com/pterm/pterm" // 引入 pterm 库用于控制台输出
)

// ConsoleReporter 控制台输出
type ConsoleReporter struct{}

func NewConsoleReporter() *ConsoleReporter {
	return &ConsoleReporter{}
}

func (r *ConsoleReporter) Report(ctx context.Context, result *model.TaskResult) error {
	// 如果结果为空，不输出
	if result == nil || result.Result == nil {
		return nil
	}

	// 尝试转换为 TabularData
	if tabular, ok := result.Result.(TabularData); ok {
		return r.printTable(tabular)
	}

	// 如果没有实现 TabularData，回退到默认的 Key-Value 打印
	// 或者如果是一个 Slice，尝试遍历打印
	if results, ok := result.Result.([]interface{}); ok {
		// 这种情况比较复杂，暂时简化处理：
		// 如果 Slice 中的元素实现了 TabularData，则聚合打印
		if len(results) > 0 {
			if first, ok := results[0].(TabularData); ok {
				headers := first.Headers()
				var rows [][]string
				for _, item := range results {
					if t, ok := item.(TabularData); ok {
						rows = append(rows, t.Rows()...)
					}
				}
				return r.printTableFromData(headers, rows)
			}
		}
	}

	// 最后的回退：打印原始数据
	pterm.Info.Println("Raw Result:", result.Result)
	return nil
}

// PrintResults 这是一个 helper 方法，用于直接打印结果列表 (适配 CLI 的逻辑)
func (r *ConsoleReporter) PrintResults(results []*model.TaskResult) {
	if len(results) == 0 {
		pterm.Warning.Println("No results found.")
		return
	}

	// 聚合所有结果
	var headers []string
	var allRows [][]string

	for i, res := range results {
		if res.Result == nil {
			continue
		}

		if tabular, ok := res.Result.(TabularData); ok {
			if len(headers) == 0 {
				headers = tabular.Headers()
			}
			allRows = append(allRows, tabular.Rows()...)
		} else if list, ok := res.Result.([]interface{}); ok {
			// 处理 Result 是 Slice 的情况 (例如 PortServiceScanner 返回多个 PortServiceResult)
			for _, item := range list {
				if tabular, ok := item.(TabularData); ok {
					if len(headers) == 0 {
						headers = tabular.Headers()
					}
					allRows = append(allRows, tabular.Rows()...)
				}
			}
		}
	}

	if len(headers) > 0 && len(allRows) > 0 {
		r.printTableFromData(headers, allRows)
	} else {
		// 无法表格化，回退
		for _, res := range results {
			pterm.Println(res.Result)
		}
	}
}

func (r *ConsoleReporter) printTable(data TabularData) error {
	return r.printTableFromData(data.Headers(), data.Rows())
}

func (r *ConsoleReporter) printTableFromData(headers []string, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	// 使用 pterm 渲染表格
	tableData := pterm.TableData{headers}
	tableData = append(tableData, rows...)

	err := pterm.DefaultTable.
		WithHasHeader(true).
		WithBoxed(false). // 简洁风格
		WithData(tableData).
		Render()

	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}
	return nil
}
