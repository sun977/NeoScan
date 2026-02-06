package fingerprint

// Input 识别输入
type Input struct {
	Target     string            // IP or Domain
	Port       int               // Port number
	Protocol   string            // tcp, udp, http
	StatusCode int               // HTTP Status Code
	Banner     string            // Raw service banner
	Headers    map[string]string      // HTTP Headers
	Body       string                 // HTTP Body
	// RichContext 存储由 WebScanner (go-rod) 提取的高级特征
	// e.g. "dom": map, "js": map, "meta": map, "cookies": map
	RichContext map[string]interface{}
	// 可扩展字段: Cert info, Icon hash...
}

// Result 识别结果
type Result struct {
	Matches []Match // 可能命中多个指纹
	Best    *Match  // 优先级最高的指纹
}

type Match struct {
	Product    string
	Version    string
	Vendor     string
	CPE        string // 标准化 CPE 2.3
	Type       string // app, os, hardware
	Confidence int    // 置信度
	Source     string // 来源 (goby, nmap, custom)
}

// MatchEngine 匹配引擎接口
type MatchEngine interface {
	Match(input *Input) ([]Match, error)
	Type() string
}
