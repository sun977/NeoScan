package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

type AssetScanOptions struct {
	Target   string
	Port     string
	Rate     int
	OSDetect bool
	Ping     bool
}

func NewAssetScanOptions() *AssetScanOptions {
	return &AssetScanOptions{
		Port: "top1000",
		Rate: 1000,
		Ping: true,
	}
}

func (o *AssetScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

func (o *AssetScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeAssetScan, o.Target)
	task.PortRange = o.Port
	task.Timeout = 2 * time.Hour // 默认超时时间

	task.Params["rate"] = o.Rate
	task.Params["os_detect"] = o.OSDetect
	task.Params["ping"] = o.Ping

	return task
}
