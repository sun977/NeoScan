package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

type PortScanOptions struct {
	Target        string
	Port          string
	Rate          int
	ServiceDetect bool
}

func NewPortScanOptions() *PortScanOptions {
	return &PortScanOptions{
		Rate:          1000,
		ServiceDetect: true,
	}
}

func (o *PortScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	if o.Port == "" {
		return fmt.Errorf("port range is required")
	}
	return nil
}

func (o *PortScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypePortScan, o.Target)
	task.PortRange = o.Port
	task.Timeout = 1 * time.Hour
	
	task.Params["rate"] = o.Rate
	task.Params["service_detect"] = o.ServiceDetect
	
	return task
}
