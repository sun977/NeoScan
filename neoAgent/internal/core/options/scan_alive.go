package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// IpAliveScanOptions 对应 IP存活扫描 的参数
type IpAliveScanOptions struct {
	Target string

	// 协议开关 (默认全关，代表自动模式；一旦开启任意一个，进入 Manual 模式)
	EnableArp  bool // --arp
	EnableIcmp bool // --icmp
	EnableTcp  bool // --tcp (TCP Full Connect)

	// TCP探测端口
	TcpPorts []int // --tcp-ports

	// 并发控制
	Concurrency int // --concurrency (默认 1000)

	// 其他选项
	ResolveHostname bool // --resolve-hostname (默认 false)

	Output OutputOptions
}

// 默认探测端口
var DefaultAliveTcpPorts = []int{22, 23, 80, 139, 512, 443, 445, 3389}

func NewIpAliveScanOptions() *IpAliveScanOptions {
	return &IpAliveScanOptions{
		TcpPorts:    DefaultAliveTcpPorts,
		Concurrency: 1000,
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

	// 序列化参数
	task.Params["enable_arp"] = o.EnableArp
	task.Params["enable_icmp"] = o.EnableIcmp
	task.Params["enable_tcp"] = o.EnableTcp
	task.Params["tcp_ports"] = o.TcpPorts
	task.Params["concurrency"] = o.Concurrency
	task.Params["resolve_hostname"] = o.ResolveHostname

	o.Output.ApplyToParams(task.Params)

	return task
}
