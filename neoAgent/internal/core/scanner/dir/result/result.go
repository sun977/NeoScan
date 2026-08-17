// Package result 定义 DirScanner 的输出数据结构。
// 此包仅供 scanner/dir 包内部使用（原子扫描器隔离原则）。
package result

import (
	"fmt"
	"strconv"
	"time"
)

// DirHit 表示一次成功命中的目录/文件扫描结果。
type DirHit struct {
	Path        string        `json:"path"`                  // 请求路径（如 /admin）
	Status      int           `json:"status"`                // HTTP 状态码
	Size        int64         `json:"size"`                  // 响应体大小（字节）
	Words       int           `json:"words,omitempty"`       // 响应体词数
	Lines       int           `json:"lines,omitempty"`       // 响应体行数
	Title       string        `json:"title,omitempty"`       // HTML <title> 内容
	Location    string        `json:"location,omitempty"`    // 重定向目标（301/302 时）
	ContentType string        `json:"content_type,omitempty"` // Content-Type
	RTT         time.Duration `json:"rtt,omitempty"`         // 请求往返时间
}

// String 实现 fmt.Stringer。
// 格式：[STATUS] /path (SIZE bytes)
func (h *DirHit) String() string {
	return fmt.Sprintf("[%d] %s (%d bytes)", h.Status, h.Path, h.Size)
}

// ScanStats 扫描统计信息。
type ScanStats struct {
	TotalRequests  int           `json:"total_requests"`
	SuccessfulReqs int           `json:"successful_reqs"` // 通过过滤器的请求数
	FilteredReqs   int           `json:"filtered_reqs"`   // 被过滤掉的请求数
	WildcardReqs   int           `json:"wildcard_reqs"`   // 被通配符检测拦截的请求数
	ErrorReqs      int           `json:"error_reqs"`      // 请求出错数
	AvgRTT         time.Duration `json:"avg_rtt,omitempty"`
	MaxRTT         time.Duration `json:"max_rtt,omitempty"`
	MinRTT         time.Duration `json:"min_rtt,omitempty"`
}

// DirResult 是 DirScanner 单次扫描的完整结果，实现 reporter.TabularData 接口。
type DirResult struct {
	Target    string     `json:"target"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time,omitempty"`
	DictSize  int        `json:"dict_size"`  // 字典总条目数
	Found     int        `json:"found"`      // 命中数量
	Hits      []*DirHit  `json:"hits"`
	Stats     ScanStats  `json:"stats"`
}

// NewDirResult 创建空的 DirResult。
func NewDirResult(target string, dictSize int) *DirResult {
	return &DirResult{
		Target:    target,
		StartTime: time.Now(),
		DictSize:  dictSize,
		Hits:      make([]*DirHit, 0),
	}
}

// Add 追加一个命中结果并更新计数。线程安全由调用方保证（DirScanner 通过 channel 汇总）。
func (r *DirResult) Add(hit *DirHit) {
	r.Hits = append(r.Hits, hit)
	r.Found++
}

// Finish 标记扫描结束时间。
func (r *DirResult) Finish() {
	r.EndTime = time.Now()
}

// Headers 实现 reporter.TabularData 接口，返回表头。
func (r *DirResult) Headers() []string {
	return []string{"Status", "Path", "Size", "Title", "Location"}
}

// Rows 实现 reporter.TabularData 接口，返回数据行。
func (r *DirResult) Rows() [][]string {
	rows := make([][]string, 0, len(r.Hits))
	for _, h := range r.Hits {
		location := h.Location
		if location == "" {
			location = "-"
		}
		title := h.Title
		if title == "" {
			title = "-"
		}
		rows = append(rows, []string{
			strconv.Itoa(h.Status),
			h.Path,
			formatSize(h.Size),
			title,
			location,
		})
	}
	return rows
}

// formatSize 将字节数格式化为可读字符串。
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
}
