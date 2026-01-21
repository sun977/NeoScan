package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

type SubdomainScanOptions struct {
	Domain  string
	Dict    string
	Threads int
	Output  OutputOptions
}

func NewSubdomainScanOptions() *SubdomainScanOptions {
	return &SubdomainScanOptions{
		Threads: 10,
	}
}

func (o *SubdomainScanOptions) Validate() error {
	if o.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	return nil
}

func (o *SubdomainScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeSubdomain, o.Domain)
	task.Timeout = 1 * time.Hour

	task.Params["dict"] = o.Dict
	task.Params["threads"] = o.Threads

	o.Output.ApplyToParams(task.Params)

	return task
}
