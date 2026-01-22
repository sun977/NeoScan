package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model" // 引入核心模型
)

type AssetScanOptions struct {
	Target   string
	Port     string
	Rate     int
	OSDetect bool
	Ping     bool
	Output   OutputOptions
}

func NewAssetScanOptions() *AssetScanOptions {
	return &AssetScanOptions{
		Port: "top1000",
		Rate: 1000,
		Ping: true,
	}
}

// Validate 验证资产扫描选项是否有效
func (o *AssetScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

// ToTask 将资产扫描选项转换为任务模型
func (o *AssetScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeAssetScan, o.Target)
	task.PortRange = o.Port
	task.Timeout = 2 * time.Hour // 默认超时时间

	task.Params["rate"] = o.Rate
	task.Params["os_detect"] = o.OSDetect
	task.Params["ping"] = o.Ping

	o.Output.ApplyToParams(task.Params)

	return task
}
