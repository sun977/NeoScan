package model

import "fmt"

// VulnResult 漏洞扫描结果
type VulnResult struct {
	ID          string `json:"id"` // CVE-202X-XXXX
	Name        string `json:"name"`
	Severity    string `json:"severity"` // critical, high, medium, low
	Description string `json:"description"`
	Reference   string `json:"reference"`
}

// Headers 实现 TabularData 接口
func (r VulnResult) Headers() []string {
	return []string{"ID", "Name", "Severity"}
}

// Rows 实现 TabularData 接口
func (r VulnResult) Rows() [][]string {
	return [][]string{{r.ID, r.Name, r.Severity}}
}

// BruteResult 爆破结果
type BruteResult struct {
	Service  string `json:"service"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Success  bool   `json:"success"`
}

// Headers 实现 TabularData 接口
func (r BruteResult) Headers() []string {
	return []string{"Service", "Host", "Port", "Username", "Password"}
}

// Rows 实现 TabularData 接口
func (r BruteResult) Rows() [][]string {
	return [][]string{{
		r.Service,
		r.Host,
		fmt.Sprintf("%d", r.Port),
		r.Username,
		r.Password,
	}}
}

// BruteResults 结果集合，用于实现 TabularData 接口以便一次性打印所有结果
type BruteResults []BruteResult

// Headers 实现 TabularData 接口
func (rs BruteResults) Headers() []string {
	return []string{"Service", "Host", "Port", "Username", "Password"}
}

// Rows 实现 TabularData 接口
func (rs BruteResults) Rows() [][]string {
	var rows [][]string
	for _, r := range rs {
		rows = append(rows, r.Rows()...)
	}
	return rows
}
