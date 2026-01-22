package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// ServiceScanOptions 对应 ServiceScan 的参数
type ServiceScanOptions struct {
	Target string
	Port   string // 目标端口 (e.g. "80,443")
	Output OutputOptions
}

func NewServiceScanOptions() *ServiceScanOptions {
	return &ServiceScanOptions{}
}

func (o *ServiceScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

func (o *ServiceScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeServiceScan, o.Target)
	task.PortRange = o.Port
	task.Timeout = 1 * time.Hour
	
	o.Output.ApplyToParams(task.Params)
	return task
}
