package model

import "fmt"

// WebResult Web扫描结果
type WebResult struct {
	URL             string            `json:"url"`
	IP              string            `json:"ip"`
	Port            int               `json:"port"`
	Title           string            `json:"title"`
	StatusCode      int               `json:"status_code"`
	ContentLength   int64             `json:"content_length"`
	ResponseHeaders map[string]string `json:"headers,omitempty"`
	TechStack       []string          `json:"tech_stack,omitempty"` // 识别到的技术栈
	Screenshot      string            `json:"screenshot,omitempty"` // Base64
	Favicon         string            `json:"favicon,omitempty"`    // Base64

	// --- 以下为爬虫功能新增字段，均为 omitempty，不影响现有序列化 ---
	Depth  int        `json:"depth,omitempty"`  // 爬取深度，0 = 首页
	Forms  []FormInfo `json:"forms,omitempty"`  // 表单/输入点
	Params []string   `json:"params,omitempty"` // URL Query 参数名集合
	Leaks  []LeakInfo `json:"leaks,omitempty"`  // 被动泄露检测结果

	// --- 以下为边缘网络组件识别功能新增字段，均为 omitempty，不影响现有序列化 ---
	EdgeComponents []EdgeComponent `json:"edge_components,omitempty"` // 命中的边缘网络组件列表（CDN/WAF/...），一个目标可能同时命中多个
}

// IsEdgeNode 判断是否命中任意边缘网络组件。调用方（如 Web 扫描器判断要不要
// 跳过截图/深度爬取）通常不关心具体命中的是 CDN 还是 WAF，只关心"这是不是
// 一个边缘节点"，用这个方法即可，不需要遍历 EdgeComponents。
func (r WebResult) IsEdgeNode() bool {
	return len(r.EdgeComponents) > 0
}

// FormInfo 表单信息（攻击面输入点）
type FormInfo struct {
	Action string   `json:"action"`
	Method string   `json:"method"`
	Fields []string `json:"fields"`
}

// LeakInfo 被动泄露检测命中信息
type LeakInfo struct {
	Type    string `json:"type"`              // aws_ak / aliyun_ak / jwt / internal_ip / ...
	Match   string `json:"match"`             // 脱敏后的命中内容，禁止存储明文密钥
	Context string `json:"context,omitempty"` // 命中上下文片段（可选）
}

// EdgeComponent 描述命中的一个边缘网络组件（CDN/WAF/反 DDoS 等）。
// 一个 WebResult 可以同时命中多个（如 Cloudflare 常见 CDN+WAF 二合一），
// 因此 WebResult 里用切片承载，不用一个组件类型对应一对标量字段。
type EdgeComponent struct {
	Type     string `json:"type"`     // "cdn" / "waf"，字符串而非枚举类型，避免跨包引入类型依赖
	Provider string `json:"provider"` // 厂商名，如 "Cloudflare"
}

// Headers 实现 TabularData 接口
func (r WebResult) Headers() []string {
	return []string{"URL", "IP", "Port", "Status", "Len", "Title", "Server", "TechStack"}
}

// Rows 实现 TabularData 接口
func (r WebResult) Rows() [][]string {
	stack := ""
	if len(r.TechStack) > 0 {
		stack = fmt.Sprintf("%v", r.TechStack)
		// 简单的截断显示
		if len(stack) > 50 {
			stack = stack[:47] + "..."
		}
	}

	server := ""
	if r.ResponseHeaders != nil {
		if s, ok := r.ResponseHeaders["Server"]; ok {
			server = s
		} else if s, ok := r.ResponseHeaders["server"]; ok {
			server = s
		}
	}
	if len(server) > 30 {
		server = server[:27] + "..."
	}

	return [][]string{{
		r.URL,
		r.IP,
		fmt.Sprintf("%d", r.Port),
		fmt.Sprintf("%d", r.StatusCode),
		fmt.Sprintf("%d", r.ContentLength),
		r.Title,
		server,
		stack,
	}}
}
