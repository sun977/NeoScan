package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

type WebScanOptions struct {
	Target     string
	Ports      string
	Path       string
	Method     string
	Crawl      string // "auto"(默认) / "true" / "false"
	CrawlDepth int
	Output     OutputOptions
}

func NewWebScanOptions() *WebScanOptions {
	return &WebScanOptions{
		Ports:      "80,443",
		Path:       "/",
		Method:     "GET",
		Crawl:      "auto",
		CrawlDepth: 2,
	}
}

func (o *WebScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

func (o *WebScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeWebScan, o.Target)
	task.PortRange = o.Ports
	task.Timeout = 30 * time.Minute

	task.Params["path"] = o.Path
	task.Params["method"] = o.Method

	switch o.Crawl {
	case "true":
		task.Params["crawl"] = true
	case "false":
		task.Params["crawl"] = false
	// "auto" 或任何其他值：不写 key，交给 WebScanner 自动判断
	}
	if o.CrawlDepth > 0 {
		task.Params["crawl_depth"] = o.CrawlDepth
	}

	o.Output.ApplyToParams(task.Params)

	return task
}
