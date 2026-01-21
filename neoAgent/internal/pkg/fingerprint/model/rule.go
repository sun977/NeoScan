package model

// FingerRule Web指纹规则 (DTO)
// 对应 Master 的 AssetFinger，去除 GORM 依赖
type FingerRule struct {
	Name       string `json:"name"`
	StatusCode string `json:"status_code"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	Footer     string `json:"footer"`
	Header     string `json:"header"`
	Response   string `json:"response"`
	Server     string `json:"server"`
	XPoweredBy string `json:"x_powered_by"`
	Body       string `json:"body"`
	Match      string `json:"match"`
	Enabled    bool   `json:"enabled"`
	Source     string `json:"source"`
}

// CPERule CPE指纹规则 (DTO)
// 对应 Master 的 AssetCPE，去除 GORM 依赖
type CPERule struct {
	Name     string `json:"name"`
	Probe    string `json:"probe"`
	MatchStr string `json:"match_str"` // 正则表达式
	Vendor   string `json:"vendor"`
	Product  string `json:"product"`
	Version  string `json:"version"`
	Update   string `json:"update"`
	Edition  string `json:"edition"`
	Language string `json:"language"`
	Part     string `json:"part"`
	CPE      string `json:"cpe"`
	Enabled  bool   `json:"enabled"`
	Source   string `json:"source"`
}
