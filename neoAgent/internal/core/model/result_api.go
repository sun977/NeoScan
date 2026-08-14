package model

import "fmt"

// APIInfo 从 JS/HTML 中提取的接口调用地址。语义上不同于页面导航链接
// （crawler.Page 里的 links）——它描述的是"前端代码运行时会向这个地址
// 发起数据请求"，而不是"这是一个可以被继续爬取的页面"，因此不会、
// 也不应该被塞进 BFS 队列继续抓取。见 API扫描功能设计.md 第六节。
type APIInfo struct {
	URL        string `json:"url"`              // 接口地址：可能是完整 URL，也可能是相对路径，不在提取阶段强行拼接成绝对地址
	Method     string `json:"method,omitempty"` // 有明确证据才填（如识别到 axios.post(...) 才填 "POST"），识别不到就留空
	Source     string `json:"source"`           // 来源：inline（内联 <script>）或具体的 js 文件 URL
	Confidence string `json:"confidence"`       // high/medium/low，见 API扫描功能设计.md 7.2 节
}

// ApiResult 是 ApiScanner 的任务产出。与 WebResult 完全独立，不复用它的
// 任何字段——ApiScanner 不做指纹识别、不截图、不判断 CDN，这些 WebResult
// 字段对 ApiResult 没有意义，见 docs/爬虫/web扫描模块重构文档.md 7.3 节。
type ApiResult struct {
	URL           string    `json:"url"`                      // 本页面 URL
	Depth         int       `json:"depth"`                    // BFS 深度，0 表示首页
	APIs          []APIInfo `json:"apis,omitempty"`           // 从 JS/HTML 中静态提取到的接口调用地址清单
	APIsTruncated bool      `json:"apis_truncated,omitempty"` // 本页引用的外链 JS 文件数超过 MaxJSFiles 被截断
}

// Headers 实现 TabularData 接口。
// 每行展示一个 APIInfo，列：页面 URL、深度、接口 URL、Method、置信度、来源。
func (r ApiResult) Headers() []string {
	return []string{"Page", "Depth", "API URL", "Method", "Confidence", "Source"}
}

// Rows 实现 TabularData 接口。
// 每个 APIInfo 独立成一行，Page/Depth 每行都完整填写（不省略），
// 这样每行是完全自描述的，支持流式逐行追加输出，也方便过滤和排序。
// 页面没有接口时输出一行占位（便于定位爬取覆盖范围）。
func (r ApiResult) Rows() [][]string {
	depth := fmt.Sprintf("%d", r.Depth)

	// truncNote 标注在 Source 列末尾，提示本页 JS 文件被截断、提取可能不完整。
	// 每行都附带，方便用户在任意行看到截断提示，而不必找"最后一行"。
	truncNote := ""
	if r.APIsTruncated {
		truncNote = " [truncated]"
	}

	if len(r.APIs) == 0 {
		// 无接口：占位一行，方便看出该页面被爬到但没有提取到接口
		return [][]string{{r.URL, depth, "(none)", "", "", truncNote}}
	}

	var rows [][]string
	for _, api := range r.APIs {
		// URL 和 Source 均完整输出，不截断，让终端/pterm 自行处理列宽，
		// 这样每行是完全自描述的，方便流式输出、管道过滤和 CSV/JSON 导出。
		rows = append(rows, []string{r.URL, depth, api.URL, api.Method, api.Confidence, api.Source + truncNote})
	}
	return rows
}
