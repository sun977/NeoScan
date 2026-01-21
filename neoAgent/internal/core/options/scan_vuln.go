package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

type VulnScanOptions struct {
	Target    string
	Templates string
	Severity  string
}

func NewVulnScanOptions() *VulnScanOptions {
	return &VulnScanOptions{
		Severity: "medium,high,critical",
	}
}

func (o *VulnScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

func (o *VulnScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeVulnScan, o.Target)
	task.Timeout = 1 * time.Hour

	task.Params["templates"] = o.Templates
	task.Params["severity"] = o.Severity

	return task
}
