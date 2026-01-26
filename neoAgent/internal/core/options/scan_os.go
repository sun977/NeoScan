package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// OsScanOptions 对应 OsScan 的参数
type OsScanOptions struct {
	Target string
	Mode   string // fast (TTL only), deep (Nmap OS DB + Service), auto (Hybrid)
	Output OutputOptions
}

func NewOsScanOptions() *OsScanOptions {
	return &OsScanOptions{
		Mode: "auto",
	}
}

func (o *OsScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	if o.Mode != "fast" && o.Mode != "deep" && o.Mode != "auto" {
		return fmt.Errorf("invalid mode: %s", o.Mode)
	}
	return nil
}

func (o *OsScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeOsScan, o.Target)
	task.Timeout = 30 * time.Minute
	task.Params["mode"] = o.Mode

	o.Output.ApplyToParams(task.Params)
	return task
}
