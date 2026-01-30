// TaskTranslator 负责将 Master 下发的任务 转换为 Core 可以处理的任务模型
// 从精简角度考虑，agent 仅支持资产扫描、端口扫描、服务扫描、漏洞扫描、POC 扫描、密码扫描、Web 扫描这 7 种任务类型
package adapter

import (
	"encoding/json"
	"neoagent/internal/core/model"
	clientModel "neoagent/internal/model/client"
)

// TaskTranslator 负责将 Master 协议转换为 Core 模型
type TaskTranslator struct{}

func NewTaskTranslator() *TaskTranslator {
	return &TaskTranslator{}
}

// ToCoreTask 将 Client Task 转换为 Core Task
func (t *TaskTranslator) ToCoreTask(ct *clientModel.Task) (*model.Task, error) {
	// 1. 解析 InputTarget
	var targets []clientModel.Target
	// 如果 InputTarget 为空或者解析失败，视为无目标任务或从 Params 获取
	if ct.InputTarget != "" {
		_ = json.Unmarshal([]byte(ct.InputTarget), &targets)
	}

	targetValue := ""
	meta := make(map[string]interface{})
	if len(targets) > 0 {
		targetValue = targets[0].Value
		for k, v := range targets[0].Meta {
			meta[k] = v
		}
	}

	// 2. 创建基础 Core Task
	// 默认使用 Master 传递的类型，如果 switch 中有特殊处理则覆盖
	coreTask := model.NewTask(model.TaskType(ct.TaskType), targetValue)
	coreTask.ID = ct.TaskID
	
	// 3. 合并 Meta 到 Params
	for k, v := range meta {
		coreTask.Params[k] = v
	}

	// 4. 根据任务类型精细化配置
	// 假设 Master 传递的 TaskType 字符串与这里的 case 匹配
	switch ct.TaskType {
	case "ip_alive_scan":
		coreTask.Type = model.TaskTypeIpAliveScan
		coreTask.Params["ping"] = true
		coreTask.Params["port"] = ""

	case "fast_port_scan":
		coreTask.Type = model.TaskTypePortScan
		coreTask.Params["port"] = "top100"
		coreTask.Params["service_detect"] = false

	case "full_port_scan":
		coreTask.Type = model.TaskTypePortScan
		coreTask.PortRange = "1-65535"
		coreTask.Params["service_detect"] = false

	case "service_scan":
		coreTask.Type = model.TaskTypeServiceScan
		if p, ok := meta["port"]; ok {
			coreTask.PortRange = p.(string)
		} else {
			coreTask.PortRange = "1-65535"
		}

	case "vuln_scan":
		coreTask.Type = model.TaskTypeVulnScan
		coreTask.Params["templates"] = "cves"
		coreTask.Params["severity"] = "critical,high,medium"

	case "poc_scan":
		coreTask.Type = model.TaskTypeVulnScan
		// POC 名称可能在 tool_params 或 meta 中
		coreTask.Params["templates"] = "custom_pocs"

	case "weak_pass_scan":
		coreTask.Type = model.TaskTypeVulnScan
		coreTask.Params["templates"] = "weak_passwords"

	case "web_scan":
		coreTask.Type = model.TaskTypeWebScan
		coreTask.Params["crawl"] = true
		coreTask.Params["method"] = "GET"

	case "api_scan":
		coreTask.Type = model.TaskTypeWebScan
		coreTask.Params["mode"] = "api"

	case "dir_scan":
		coreTask.Type = model.TaskTypeDirScan

	case "subdomain_scan":
		coreTask.Type = model.TaskTypeSubdomain
		coreTask.Params["mode"] = "brute"
	}
	
	// 5. 处理 ToolParams (如果需要覆盖)
	if ct.ToolParams != "" {
		coreTask.Params["tool_params"] = ct.ToolParams
	}

	return coreTask, nil
}
