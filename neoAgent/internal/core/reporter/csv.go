package reporter

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sync"

	"neoagent/internal/core/model"
)

// CsvReporter 负责将结果导出为 CSV 文件
type CsvReporter struct {
	FilePath string
	mu       sync.Mutex
	headers  []string
	file     *os.File
	writer   *csv.Writer
}

func NewCsvReporter(filePath string) *CsvReporter {
	return &CsvReporter{
		FilePath: filePath,
	}
}

// Report 并不适合流式写入 CSV，因为我们可能需要先确定表头。
// 对于 CLI 工具，通常是在所有扫描完成后一次性导出。
// 所以我们提供一个专门的 Export 方法，或者在 Report 中追加。
// 为了简化 CLI 调用，这里提供一个静态方法 SaveCsvResult，与 json 的逻辑类似。
func (r *CsvReporter) Report(ctx context.Context, result *model.TaskResult) error {
	// 暂不实现流式 Report，因为表头可能不确定
	return nil
}

// SaveCsvResult 这是一个辅助函数，用于一次性将结果保存为 CSV
func SaveCsvResult(path string, results []*model.TaskResult) error {
	if len(results) == 0 {
		return nil
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create csv file: %v", err)
	}
	defer f.Close()

	// 写入 UTF-8 BOM，防止 Excel 打开乱码
	f.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(f)
	defer w.Flush()

	var headers []string
	var allRows [][]string

	// 1. 收集数据
	for _, res := range results {
		if res.Result == nil {
			continue
		}

		// 尝试转换为 TabularData
		if tabular, ok := res.Result.(TabularData); ok {
			if len(headers) == 0 {
				headers = tabular.Headers()
			}
			allRows = append(allRows, tabular.Rows()...)
		} else if list, ok := res.Result.([]interface{}); ok {
			// 处理 Result 是 Slice 的情况
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

	if len(headers) == 0 {
		return fmt.Errorf("no tabular data found to export")
	}

	// 2. 写入表头
	if err := w.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// 3. 写入行数据
	if err := w.WriteAll(allRows); err != nil {
		return fmt.Errorf("failed to write rows: %v", err)
	}

	fmt.Printf("[+] Results saved to %s\n", path)
	return nil
}
