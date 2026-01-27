package nmap_service

import (
	"time"

	"github.com/dlclark/regexp2"
)

// Status 扫描状态
type Status int

const (
	// Closed 端口关闭
	Closed Status = iota
	// Open 端口开放
	Open
	// Matched 匹配成功
	Matched
	// NotMatched 匹配失败
	NotMatched
)

// FingerPrint 服务指纹信息
type FingerPrint struct {
	ProbeName        string `json:"probe_name"`
	MatchRegexString string `json:"match_regex_string"`
	Service          string `json:"service"`
	ProductName      string `json:"product_name"`
	Version          string `json:"version"`
	Info             string `json:"info"`
	Hostname         string `json:"hostname"`
	OperatingSystem  string `json:"operating_system"`
	DeviceType       string `json:"device_type"`
	CPE              string `json:"cpe"`
}

// Response 扫描响应
type Response struct {
	Raw         string       `json:"raw"`
	FingerPrint *FingerPrint `json:"finger_print"`
}

// Probe Nmap 探针定义
type Probe struct {
	Name        string
	Protocol    string
	ProbeString string
	Wait        time.Duration
	Ports       []int
	SslPorts    []int
	Rarity      int
	Fallback    []string

	// 匹配规则组
	MatchGroup     []*Match
	SoftMatchGroup []*Match
}

// Match 匹配规则
type Match struct {
	IsSoft              bool
	Service             string
	Pattern             string
	PatternRegexp       *regexp2.Regexp
	VersionInfoTemplate string // 版本提取模板 (e.g. "p/$1/ v/$2/")
}
