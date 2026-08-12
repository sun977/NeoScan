package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// ApiScanOptions 是 ApiScan 任务的 CLI 参数集合。不含 Crawl 三态开关——
// ApiScan 的深度爬取不是可选项，是这个扫描器存在的核心意义，见
// docs/API扫描-js提取/API扫描功能设计.md 第五节。
type ApiScanOptions struct {
	Target     string
	Ports      string
	CrawlDepth int
	MaxJSFiles int
	Output     OutputOptions
}

func NewApiScanOptions() *ApiScanOptions {
	return &ApiScanOptions{
		Ports:      "80,443",
		CrawlDepth: 2,
		MaxJSFiles: 20,
	}
}

func (o *ApiScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

func (o *ApiScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeApiScan, o.Target)
	task.PortRange = o.Ports
	task.Timeout = 30 * time.Minute

	if o.CrawlDepth > 0 {
		task.Params["crawl_depth"] = o.CrawlDepth
	}
	if o.MaxJSFiles > 0 {
		task.Params["max_js_files"] = o.MaxJSFiles
	}

	o.Output.ApplyToParams(task.Params)
	return task
}
