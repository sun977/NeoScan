package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

type WebScanOptions struct {
	Target string
	Ports  string
	Path   string
	Method string
	Output OutputOptions
}

func NewWebScanOptions() *WebScanOptions {
	return &WebScanOptions{
		Ports:  "80,443",
		Path:   "/",
		Method: "GET",
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

	o.Output.ApplyToParams(task.Params)

	return task
}
