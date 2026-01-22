// TaskTranslator 负责将 Master 下发的任务 转换为 Core 可以处理的任务模型
// 从精简角度考虑，agent 仅支持资产扫描、端口扫描、服务扫描、漏洞扫描、POC 扫描、密码扫描、Web 扫描这 7 种任务类型
// Master 支持的扫描更细致（有14种扫描类型），所以需要根据 Master 任务类型进行转换，多余的类型通过参数的形式实现
package adapter

import (
	"time"

	"neoagent/internal/core/model"
	"neoagent/internal/pkg/protocol"
)

// TaskTranslator 负责将 Master 协议转换为 Core 模型
type TaskTranslator struct{}

func NewTaskTranslator() *TaskTranslator {
	return &TaskTranslator{}
}

// ToCoreTask 将 MasterTask 转换为 Core Task
func (t *TaskTranslator) ToCoreTask(mt protocol.MasterTask) *model.Task {
	var task *model.Task

	switch mt.Type {
	case protocol.MasterTypeIpAliveScan:
		task = model.NewTask(model.TaskTypeAssetScan, mt.Target)
		task.Params["ping"] = true
		task.Params["port"] = "" // 不扫端口

	case protocol.MasterTypeFastPortScan:
		task = model.NewTask(model.TaskTypeAssetScan, mt.Target)
		task.Params["ping"] = false
		task.Params["port"] = "top100"
		task.Params["os_detect"] = false

	case protocol.MasterTypeFullPortScan:
		task = model.NewTask(model.TaskTypePortScan, mt.Target)
		task.PortRange = "1-65535"
		task.Params["service_detect"] = false // 全端口扫描不默认开启深度识别，太慢

	case protocol.MasterTypeServiceScan:
		task = model.NewTask(model.TaskTypeServiceScan, mt.Target)
		// 如果 Master 指定了端口，使用指定端口，否则默认全扫
		if p, ok := mt.Params["port"]; ok {
			task.PortRange = p
		} else {
			task.PortRange = "1-65535"
		}
		// service_scan 默认就是为了识别服务，不需要额外参数开关

	case protocol.MasterTypeVulnScan:
		task = model.NewTask(model.TaskTypeVulnScan, mt.Target)
		task.Params["templates"] = "cves"
		task.Params["severity"] = "critical,high,medium"

	case protocol.MasterTypePocScan:
		task = model.NewTask(model.TaskTypeVulnScan, mt.Target)
		if poc, ok := mt.Params["poc"]; ok {
			task.Params["templates"] = poc
		} else {
			task.Params["templates"] = "custom_pocs"
		}

	case protocol.MasterTypePassScan:
		task = model.NewTask(model.TaskTypeVulnScan, mt.Target)
		task.Params["templates"] = "weak_passwords"

	case protocol.MasterTypeWebScan:
		task = model.NewTask(model.TaskTypeWebScan, mt.Target)
		task.Params["crawl"] = true
		task.Params["method"] = "GET"

	case protocol.MasterTypeApiScan:
		task = model.NewTask(model.TaskTypeWebScan, mt.Target)
		task.Params["mode"] = "api"
		if path, ok := mt.Params["path"]; ok {
			task.Params["path"] = path
		}

	case protocol.MasterTypeDirScan:
		task = model.NewTask(model.TaskTypeDirScan, mt.Target)
		if dict, ok := mt.Params["dict"]; ok {
			task.Params["dict"] = dict
		}

	case protocol.MasterTypeSubDomainScan:
		task = model.NewTask(model.TaskTypeSubdomain, mt.Target)
		task.Params["mode"] = "brute"

	case protocol.MasterTypeProxyScan:
		task = model.NewTask(model.TaskTypeServiceScan, mt.Target)
		// 常见代理端口
		task.PortRange = "1080,8080,8888,3128,7890"
		// 代理探测需要深度协议识别，使用 ServiceScan 更合适

	case protocol.MasterTypeFileScan:
		task = model.NewTask(model.TaskTypeRawCmd, mt.Target)
		// 文件扫描通常需要在 Agent 本地执行命令，需谨慎处理
		task.Params["cmd"] = "yara"

	case protocol.MasterTypeOtherScan:
		task = model.NewTask(model.TaskTypeRawCmd, mt.Target)
		if cmd, ok := mt.Params["cmd"]; ok {
			task.Params["cmd"] = cmd
		}

	default:
		// 未知类型，记录日志或忽略
		return nil
	}

	// 注入通用属性
	task.ID = mt.ID
	if mt.Timeout > 0 {
		task.Timeout = time.Duration(mt.Timeout) * time.Second
	}

	// 合并其他 Master Params
	for k, v := range mt.Params {
		// 避免覆盖已设置的关键参数
		if _, exists := task.Params[k]; !exists {
			task.Params[k] = v
		}
	}

	return task
}
