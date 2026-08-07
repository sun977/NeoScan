package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// ApiScanOptions 骨架版本：只包含抓取相关的参数（Target/Ports/Crawl/CrawlDepth），
// 与 WebScanOptions 的对应字段语义完全一致，直接照搬，不重新设计。
// MaxFiles（单页最多下载的外链 JS 文件数）由 Web-JS接口提取实施文档.md
// 第二节追加，本步骤不包含。
type ApiScanOptions struct {
	Target     string
	Ports      string
	Crawl      string // "auto"(默认) / "true" / "false"，语义与 WebScanOptions.Crawl 一致
	CrawlDepth int
	Output     OutputOptions
}

func NewApiScanOptions() *ApiScanOptions {
	return &ApiScanOptions{
		Ports:      "80,443",
		Crawl:      "auto",
		CrawlDepth: 2,
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

	switch o.Crawl {
	case "true":
		task.Params["crawl"] = true
	case "false":
		task.Params["crawl"] = false
	}
	if o.CrawlDepth > 0 {
		task.Params["crawl_depth"] = o.CrawlDepth
	}

	o.Output.ApplyToParams(task.Params)
	return task
}
