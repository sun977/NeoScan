package parser

import (
	"encoding/xml"
	"fmt"

	"neomaster/internal/pkg/tool_adapter/models"
)

// NmapXMLParser 解析 Nmap XML 输出
type NmapXMLParser struct{}

// NmapRun XML 根节点结构
type NmapRun struct {
	XMLName  xml.Name `xml:"nmaprun"`
	Scanner  string   `xml:"scanner,attr"`
	Version  string   `xml:"version,attr"`
	Start    int64    `xml:"start,attr"`
	StartStr string   `xml:"startstr,attr"`
	Hosts    []Host   `xml:"host"`
	RunStats RunStats `xml:"runstats"`
}

type Host struct {
	StartTime int64     `xml:"starttime,attr"`
	EndTime   int64     `xml:"endtime,attr"`
	Status    Status    `xml:"status"`
	Addresses []Address `xml:"address"`
	Hostnames Hostnames `xml:"hostnames"`
	Ports     Ports     `xml:"ports"`
	Os        Os        `xml:"os"`
}

type Status struct {
	State     string  `xml:"state,attr"`
	Reason    string  `xml:"reason,attr"`
	ReasonTTL float64 `xml:"reason_ttl,attr"`
}

type Address struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"` // ipv4, ipv6, mac
}

type Hostnames struct {
	Hostnames []Hostname `xml:"hostname"`
}

type Hostname struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type Ports struct {
	Ports []Port `xml:"port"`
}

type Port struct {
	Protocol string  `xml:"protocol,attr"`
	PortID   int     `xml:"portid,attr"`
	State    State   `xml:"state"`
	Service  Service `xml:"service"`
}

type State struct {
	State     string `xml:"state,attr"` // open, closed
	Reason    string `xml:"reason,attr"`
	ReasonTTL int    `xml:"reason_ttl,attr"`
}

type Service struct {
	Name      string   `xml:"name,attr"`
	Product   string   `xml:"product,attr"`
	Version   string   `xml:"version,attr"`
	ExtraInfo string   `xml:"extrainfo,attr"`
	CPEs      []string `xml:"cpe"`
}

type Os struct {
	OsMatches []OsMatch `xml:"osmatch"`
}

type OsMatch struct {
	Name     string `xml:"name,attr"`
	Accuracy int    `xml:"accuracy,attr"`
}

type RunStats struct {
	Finished Finished `xml:"finished"`
}

type Finished struct {
	Time    int64   `xml:"time,attr"`
	Elapsed float64 `xml:"elapsed,attr"`
	Summary string  `xml:"summary,attr"`
	Exit    string  `xml:"exit,attr"`
}

// Parse 解析 Nmap XML 输出
func (p *NmapXMLParser) Parse(output string) (*models.ToolScanResult, error) {
	var run NmapRun
	if err := xml.Unmarshal([]byte(output), &run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nmap xml: %w", err)
	}

	result := &models.ToolScanResult{
		ToolName:  "nmap",
		StartTime: run.Start,
		EndTime:   run.RunStats.Finished.Time,
		Status:    "success",
		RawOutput: output,
	}

	if run.RunStats.Finished.Exit != "success" {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Nmap exited with status: %s", run.RunStats.Finished.Exit)
	}

	for _, h := range run.Hosts {
		// 1. 提取 IP
		var ip string
		for _, addr := range h.Addresses {
			if addr.AddrType == "ipv4" || addr.AddrType == "ipv6" {
				ip = addr.Addr
				break
			}
		}
		if ip == "" {
			continue // 跳过没有 IP 的主机
		}

		// 2. 提取 Hostname
		var hostname string
		if len(h.Hostnames.Hostnames) > 0 {
			hostname = h.Hostnames.Hostnames[0].Name
		}

		// 3. 提取 OS
		var osName string
		if len(h.Os.OsMatches) > 0 {
			osName = h.Os.OsMatches[0].Name // 取第一个匹配度最高的
		}

		// 添加 HostInfo
		result.Hosts = append(result.Hosts, models.HostInfo{
			IP:       ip,
			Hostname: hostname,
			OS:       osName,
			Status:   h.Status.State,
			TTL:      int(h.Status.ReasonTTL),
		})

		// 4. 提取 Ports
		for _, p := range h.Ports.Ports {
			// 只记录 open 的端口
			if p.State.State != "open" {
				continue
			}

			cpe := ""
			if len(p.Service.CPEs) > 0 {
				cpe = p.Service.CPEs[0]
			}

			result.Ports = append(result.Ports, models.PortInfo{
				IP:      ip,
				Port:    p.PortID,
				Proto:   p.Protocol,
				State:   p.State.State,
				Service: p.Service.Name,
				Product: p.Service.Product,
				Version: p.Service.Version,
				Banner:  fmt.Sprintf("%s %s %s", p.Service.Product, p.Service.Version, p.Service.ExtraInfo),
				CPE:     cpe,
			})
		}
	}

	// 简单的去重逻辑（如果需要）
	// 目前 append 即可

	return result, nil
}
