package adapter

import (
	"neoagent/internal/core/model"
	adapterModel "neoagent/internal/model/adapter"
)

// ToMasterResult 将 Core TaskResult 转换为符合契约的 DTO
// taskID: 任务 ID
// internalRes: Core 返回的原始结果
func (t *TaskTranslator) ToMasterResult(taskID string, internalRes *model.TaskResult) (*model.TaskResult, error) {
	// 1. 准备顶层结构
	// 注意：这里的 Result 字段类型需要是 interface{} 或 map[string]interface{}
	// 因为我们要放入 Attributes 和 Evidence
	masterRes := &model.TaskResult{
		TaskID:      taskID,
		Status:      internalRes.Status,
		ExecutedAt:  internalRes.ExecutedAt,
		CompletedAt: internalRes.CompletedAt,
		Error:       internalRes.Error,
	}

	if internalRes.Status != model.TaskStatusSuccess {
		// 如果失败，只返回错误信息，不需要 Result
		return masterRes, nil
	}

	var attributes interface{}
	evidence := make(map[string]string)

	// 2. 根据 Core 任务类型进行转换
	// internalRes.Result 实际上是具体的 Scanner 返回的结构体切片，例如 []model.IpAliveResult
	// 我们需要断言并转换为 adapterModel 中的 DTO

	// 由于 model.TaskResult.Type 字段并未在 TaskResult 定义中直接暴露（它只在 Task 中），
	// 但通常我们可以通过上下文或 internalRes.Result 的类型来推断。
	// 或者，我们在调用 ToMasterResult 时应该传入 TaskType。
	// 为了简化，这里假设 internalRes.Result 是可以直接类型断言的。

	// 更好的做法是让 Scanner 返回的结果类型更加明确。
	// 目前 internal/core/model/result_types.go 定义了 Result 类型。

	switch res := internalRes.Result.(type) {
	case []model.IpAliveResult:
		// 转换 IpAlive
		attr := adapterModel.IpAliveAttributes{
			Hosts: make([]adapterModel.HostInfo, 0),
			Summary: &adapterModel.IpAliveSummary{
				TotalScanned: len(res),
				// ElapsedMs 需要从外部传入或计算
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
		// 转换 PortScan
		attr := adapterModel.PortScanAttributes{
			Ports:   make([]adapterModel.PortInfo, 0),
			Summary: &adapterModel.PortScanSummary{
				// Total/Open 需要统计
			},
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
				State:       r.Status, // 需确保全是小写 open
				ServiceHint: r.Service,
				Banner:      r.Banner,
			})
		}
		attr.Summary.OpenCount = openCount
		attributes = attr

	// TODO: 补充 Service, OS, Web 等其他类型的转换
	// 目前 Core 中尚未统一定义这些 Result 类型，或者还没完全实现。
	// 我们先实现已有的 Alive 和 Port。

	default:
		// 未知类型，尝试直接序列化或作为通用 Map
		// 但为了符合契约，最好记录日志
		attributes = map[string]interface{}{
			"raw_data": res,
			"note":     "unsupported_result_type_in_adapter",
		}
	}

	// 3. 构造最终的 TopLevelResult
	// 由于 model.TaskResult.Result 是 interface{}，我们可以放入任何东西。
	// 但 Master 期望的是 { "attributes": {...}, "evidence": {...} }

	// 使用 adapterModel.TopLevelResult 包装
	topLevel := adapterModel.TopLevelResult{
		Attributes: attributes,
		Evidence:   evidence,
	}

	// 将 TopLevelResult 转为 map[string]interface{}，或者直接赋值给 Result
	// model.TaskResult 定义 Result 为 interface{}，所以直接赋值即可。
	// 但为了确保 JSON 序列化后的字段名正确 (attributes, evidence)，
	// 我们最好确保 JSON 库能正确处理 TopLevelResult。
	// 经检查 adapterModel.TopLevelResult 有 json tags。
	masterRes.Result = topLevel

	return masterRes, nil
}
