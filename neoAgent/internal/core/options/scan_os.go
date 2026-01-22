package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// OsScanOptions 对应 OsScan 的参数
type OsScanOptions struct {
	Target string
	Output OutputOptions
}

func NewOsScanOptions() *OsScanOptions {
	return &OsScanOptions{}
}

func (o *OsScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

func (o *OsScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeOsScan, o.Target)
	task.Timeout = 30 * time.Minute
	
	o.Output.ApplyToParams(task.Params)
	return task
}
