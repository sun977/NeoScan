package adapter

import (
	"encoding/json"
	"neoagent/internal/core/model"
	adapterModel "neoagent/internal/model/adapter"
	clientModel "neoagent/internal/model/client"
)

// ToTaskStatusReport 将 Core TaskResult 转换为符合 Master 协议的 TaskStatusReport
func (t *TaskTranslator) ToTaskStatusReport(taskID string, internalRes *model.TaskResult) (*clientModel.TaskStatusReport, error) {
	report := &clientModel.TaskStatusReport{
		Status:   string(internalRes.Status), // 假设状态字符串兼容
		ErrorMsg: internalRes.Error,
	}

	if internalRes.Status != model.TaskStatusSuccess {
		return report, nil
	}

	var attributes interface{}

	// 根据 Core 任务类型进行转换
	switch res := internalRes.Result.(type) {
	case []model.IpAliveResult:
		attr := adapterModel.IpAliveAttributes{
			Hosts: make([]adapterModel.HostInfo, 0),
			Summary: &adapterModel.IpAliveSummary{
				TotalScanned: len(res),
			},
		}
		aliveCount := 0
		for _, r := range res {
			if r.Alive {
				aliveCount++
				attr.Hosts = append(attr.Hosts, adapterModel.HostInfo{
					IP:       r.IP,
					RTT:      float64(r.RTT.Microseconds()) / 1000.0,
					TTL:      r.TTL,
					Hostname: r.Hostname,
					OS:       r.OS,
				})
			}
		}
		attr.Summary.AliveCount = aliveCount
		attributes = attr

	case []model.PortServiceResult:
		attr := adapterModel.PortScanAttributes{
			Ports:   make([]adapterModel.PortInfo, 0),
			Summary: &adapterModel.PortScanSummary{},
		}
		openCount := 0
		for _, r := range res {
			if r.Status == "open" || r.Status == "Open" {
				openCount++
			}
			attr.Ports = append(attr.Ports, adapterModel.PortInfo{
				IP:          r.IP,
				Port:        r.Port,
				Proto:       r.Protocol,
				State:       r.Status,
				ServiceHint: r.Service,
				Banner:      r.Banner,
			})
		}
		attr.Summary.OpenCount = openCount
		attributes = attr

	default:
		// 如果无法识别类型，直接使用原始结果
		attributes = internalRes.Result
	}

	// 序列化结果为 JSON 字符串
	if attributes != nil {
		jsonBytes, err := json.Marshal(attributes)
		if err == nil {
			report.Result = string(jsonBytes)
		}
	}

	return report, nil
}
