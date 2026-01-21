package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

type DirScanOptions struct {
	Target     string
	Dict       string
	Extensions string
	Threads    int
	Output     OutputOptions
}

func NewDirScanOptions() *DirScanOptions {
	return &DirScanOptions{
		Threads: 10,
	}
}

func (o *DirScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

func (o *DirScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeDirScan, o.Target)
	task.Timeout = 2 * time.Hour

	task.Params["dict"] = o.Dict
	task.Params["extensions"] = o.Extensions
	task.Params["threads"] = o.Threads

	o.Output.ApplyToParams(task.Params)

	return task
}
