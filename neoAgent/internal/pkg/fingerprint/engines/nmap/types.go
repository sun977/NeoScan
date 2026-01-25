package nmap

import (
	"regexp"
)

// Status 表示端口或服务探测的状态
type Status int

const (
	StatusClosed     Status = iota // 端口关闭
	StatusOpen                     // 端口开放 (但未匹配到服务)
	StatusMatched                  // 成功匹配到服务指纹
	StatusNotMatched               // 端口开放但未匹配到服务 (Unknown)
	StatusUnknown                  // 未知状态
)

func (s Status) String() string {
	switch s {
	case StatusClosed:
		return "Closed"
	case StatusOpen:
		return "Open"
	case StatusMatched:
		return "Matched"
	case StatusNotMatched:
		return "NotMatched"
	default:
		return "Unknown"
	}
}

// PortList 端口列表
type PortList []int

// Probe 代表 Nmap nmap-service-probes 中的一个 Probe 指令
// 例如: Probe TCP NULL q||
type Probe struct {
	Name        string   // 探针名称 (e.g. NULL, GetRequest)
	Protocol    string   // 协议 (TCP/UDP)
	ProbeString string   // 原始探测包字符串 (q|...|)
	Rarity      int      // 稀有度 (1-9)
	Ports       PortList // 适用端口列表 (ports 指令)
	SslPorts    PortList // SSL适用端口列表 (sslports 指令)
	Fallback    string   // Fallback 探针名称

	MatchGroup     []*Match // 关联的 Match 指令集合
	SoftMatchGroup []*Match // 关联的 SoftMatch 指令集合
}

// Match 代表 Nmap nmap-service-probes 中的一个 match/softmatch 指令
// 例如: match ftp m/^220.*FTP/ p/vsftpd/
type Match struct {
	IsSoft        bool           // 是否为 softmatch
	Service       string         // 服务名称 (e.g. ftp, http)
	Pattern       string         // 正则表达式字符串
	PatternRegexp *regexp.Regexp // 编译后的正则表达式
	VersionInfo   string         // 原始版本信息字符串 (e.g. p/Bind/ v/9.x/)

	// 预编译的版本提取规则 (可选优化)
}

// FingerPrint 代表服务识别结果
type FingerPrint struct {
	ProbeName        string // 触发匹配的探针名称
	MatchRegexString string // 匹配的正则字符串

	Service         string // 服务名称 (e.g. ssh)
	ProductName     string // 产品名称 (p/)
	Version         string // 版本号 (v/)
	Info            string // 附加信息 (i/)
	Hostname        string // 主机名 (h/)
	OperatingSystem string // 操作系统 (o/)
	DeviceType      string // 设备类型 (d/)
	CPE             string // CPE 标识 (cpe:/)
}
