package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"neomaster/internal/pkg/tool_adapter/models"
)

// MasscanJSONParser 解析 Masscan JSON 输出
// Masscan 的 JSON 输出通常是流式的（每行一个 JSON 对象），或者是一个包含多个对象的数组（如果在最后才输出）
// 这里的实现假设输入是一个合法的 JSON 数组（如果 Masscan 使用 -oJ 并在结束后生成了完整的 JSON 文件）
// 或者我们会尝试按行解析
type MasscanJSONParser struct{}

// MasscanRecord 单条记录结构
type MasscanRecord struct {
	IP        string `json:"ip"`
	Timestamp string `json:"timestamp"`
	Ports     []struct {
		Port   int    `json:"port"`
		Proto  string `json:"proto"`
		Status string `json:"status"`
		Reason string `json:"reason"`
		TTL    int    `json:"ttl"`
	} `json:"ports"`
}

// Parse 解析 Masscan JSON 输出
func (p *MasscanJSONParser) Parse(output string) (*models.ToolScanResult, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return &models.ToolScanResult{
			ToolName:  "masscan",
			Status:    "success",
			RawOutput: output,
		}, nil
	}

	var records []MasscanRecord

	// 尝试解析为 JSON 数组
	if strings.HasPrefix(output, "[") {
		if err := json.Unmarshal([]byte(output), &records); err != nil {
			// 如果整体解析失败，尝试去掉最后可能存在的逗号（Masscan bug?）
			// 或者尝试按行解析
			return nil, fmt.Errorf("failed to unmarshal masscan json array: %w", err)
		}
	} else {
		// 尝试按行解析 (JSON Lines)
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// 有些 Masscan 输出可能以 comma 结尾，如果不是数组格式
			line = strings.TrimSuffix(line, ",")

			var record MasscanRecord
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				// 忽略解析错误的行，或者报错？
				// 为了健壮性，忽略
				continue
			}
			records = append(records, record)
		}
	}

	result := &models.ToolScanResult{
		ToolName:  "masscan",
		StartTime: time.Now().Unix(), // Masscan 结果中通常只有 record timestamp，没有 scan start/end
		EndTime:   time.Now().Unix(),
		Status:    "success",
		RawOutput: output,
	}

	hostMap := make(map[string]bool)

	for _, r := range records {
		// HostInfo
		if !hostMap[r.IP] {
			result.Hosts = append(result.Hosts, models.HostInfo{
				IP:     r.IP,
				Status: "up",
			})
			hostMap[r.IP] = true
		}

		// PortInfo
		for _, p := range r.Ports {
			if p.Status != "open" {
				continue
			}
			result.Ports = append(result.Ports, models.PortInfo{
				IP:    r.IP,
				Port:  p.Port,
				Proto: p.Proto,
				State: p.Status,
				// Masscan 基础扫描不包含 Service/Product/Version
			})
		}
	}

	return result, nil
}
