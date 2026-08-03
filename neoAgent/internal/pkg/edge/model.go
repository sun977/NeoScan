// Package edge 提供边缘网络节点（CDN/WAF）识别能力。
//
// 命名与目录结构对齐 rules/edge/ 规则目录：CDN 和 WAF 都是部署在源站
// 前面的边缘网络组件，属于同一个能力域下的两类判断。本次只实现 CDN，
// WAF 的判断方式大概率不只是 IP 段（可能还要看拦截页/响应头特征），
// 届时在本包内新增文件即可，不需要再拆一个新包。
package edge

// ProviderRanges 是单个厂商的 CDN 网段定义，一一对应 rules/edge/cdn.json
// 里的一个数组元素。
type ProviderRanges struct {
	Provider string   `json:"provider"`
	CIDRs    []string `json:"cidrs"`
}
