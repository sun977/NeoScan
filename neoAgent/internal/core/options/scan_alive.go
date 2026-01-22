package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// IpAliveScanOptions 对应 IP存活扫描 的参数
type IpAliveScanOptions struct {
	Target string
	Ping   bool
	Output OutputOptions
}

func NewIpAliveScanOptions() *IpAliveScanOptions {
	return &IpAliveScanOptions{
		Ping: true,
	}
}

func (o *IpAliveScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

func (o *IpAliveScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeIpAliveScan, o.Target)
	task.Timeout = 1 * time.Hour // 默认超时时间

	task.Params["ping"] = o.Ping

	o.Output.ApplyToParams(task.Params)

	return task
}
