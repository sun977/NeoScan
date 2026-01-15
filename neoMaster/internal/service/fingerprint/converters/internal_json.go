// StandardJSONConverter 实现 RuleConverter 接口
// 负责 DB 模型与 StandardJSON 格式之间的转换
// 双向支持 :
// - Encode : DB Models -> Standard JSON (用于导出/备份/Agent分发)
// - Decode : Standard JSON -> DB Models (用于导入/恢复)

package converters

import (
	"encoding/json"
	"fmt"
	"time"

	"neomaster/internal/model/asset"
)

// StandardJSON 结构定义了 NeoScan 内部标准指纹文件的格式
// 用于管理员导入/导出，包含完整元数据
type StandardJSON struct {
	Version   string          `json:"version"`           // 格式版本 (e.g. "1.0")
	Timestamp time.Time       `json:"timestamp"`         // 导出时间
	Source    string          `json:"source"`            // 来源说明 (e.g. "NeoScan Export")
	Fingers   []FingerRuleDTO `json:"fingers,omitempty"` // Web 指纹列表
	CPEs      []CPERuleDTO    `json:"cpes,omitempty"`    // CPE 指纹列表
}

// FingerRuleDTO 是 AssetFinger 的数据传输对象 (DTO)
// 移除了 ID, CreatedAt 等 DB 专属字段，保留业务字段
type FingerRuleDTO struct {
	Name       string `json:"name"`
	StatusCode string `json:"status_code,omitempty"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	Subtitle   string `json:"subtitle,omitempty"`
	Footer     string `json:"footer,omitempty"`
	Header     string `json:"header,omitempty"`
	Response   string `json:"response,omitempty"`
	Server     string `json:"server,omitempty"`
	XPoweredBy string `json:"x_powered_by,omitempty"`
	Body       string `json:"body,omitempty"`
	Match      string `json:"match,omitempty"`
	Enabled    bool   `json:"enabled"` // 默认为 true，omitempty 可能导致 false 丢失，所以不加 omitempty
	Source     string `json:"source,omitempty"`
}

// CPERuleDTO 是 AssetCPE 的数据传输对象 (DTO)
type CPERuleDTO struct {
	Name     string `json:"name"`
	Probe    string `json:"probe,omitempty"`
	MatchStr string `json:"match_str"`
	Vendor   string `json:"vendor,omitempty"`
	Product  string `json:"product,omitempty"`
	Version  string `json:"version,omitempty"`
	Update   string `json:"update,omitempty"`
	Edition  string `json:"edition,omitempty"`
	Language string `json:"language,omitempty"`
	Part     string `json:"part,omitempty"`
	CPE      string `json:"cpe,omitempty"`
	Enabled  bool   `json:"enabled"`
	Source   string `json:"source,omitempty"`
}

// StandardJSONConverter 实现 RuleConverter 接口
// 负责 DB 模型与 StandardJSON 格式之间的转换
type StandardJSONConverter struct{}

// NewStandardJSONConverter 创建转换器实例
func NewStandardJSONConverter() *StandardJSONConverter {
	return &StandardJSONConverter{}
}

// Encode 将 DB 模型列表序列化为标准 JSON 字节流
// 如果 fingers 或 cpes 为 nil 或空，则生成的 JSON 对应字段也将被省略
func (c *StandardJSONConverter) Encode(fingers []*asset.AssetFinger, cpes []*asset.AssetCPE) ([]byte, error) {
	var fingerDTOs []FingerRuleDTO
	if len(fingers) > 0 {
		fingerDTOs = make([]FingerRuleDTO, 0, len(fingers))
		for _, f := range fingers {
			fingerDTOs = append(fingerDTOs, FingerRuleDTO{
				Name:       f.Name,
				StatusCode: f.StatusCode,
				URL:        f.URL,
				Title:      f.Title,
				Subtitle:   f.Subtitle,
				Footer:     f.Footer,
				Header:     f.Header,
				Response:   f.Response,
				Server:     f.Server,
				XPoweredBy: f.XPoweredBy,
				Body:       f.Body,
				Match:      f.Match,
				Enabled:    f.Enabled,
				Source:     f.Source,
			})
		}
	}

	var cpeDTOs []CPERuleDTO
	if len(cpes) > 0 {
		cpeDTOs = make([]CPERuleDTO, 0, len(cpes))
		for _, c := range cpes {
			cpeDTOs = append(cpeDTOs, CPERuleDTO{
				Name:     c.Name,
				Probe:    c.Probe,
				MatchStr: c.MatchStr,
				Vendor:   c.Vendor,
				Product:  c.Product,
				Version:  c.Version,
				Update:   c.Update,
				Edition:  c.Edition,
				Language: c.Language,
				Part:     c.Part,
				CPE:      c.CPE,
				Enabled:  c.Enabled,
				Source:   c.Source,
			})
		}
	}

	// 3. 构建标准结构
	stdData := StandardJSON{
		Version:   "1.0",
		Timestamp: time.Now(),
		Source:    "NeoScan Export",
		Fingers:   fingerDTOs,
		CPEs:      cpeDTOs,
	}

	// 4. 序列化 (使用 Indent 方便阅读)
	return json.MarshalIndent(stdData, "", "  ")
}

// Decode 将标准 JSON 字节流反序列化为 DB 模型列表
func (c *StandardJSONConverter) Decode(data []byte) ([]*asset.AssetFinger, []*asset.AssetCPE, error) {
	var stdData StandardJSON
	if err := json.Unmarshal(data, &stdData); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal standard json: %w", err)
	}

	// 1. 转换 DTOs -> Fingers
	fingers := make([]*asset.AssetFinger, 0, len(stdData.Fingers))
	for _, dto := range stdData.Fingers {
		fingers = append(fingers, &asset.AssetFinger{
			Name:       dto.Name,
			StatusCode: dto.StatusCode,
			URL:        dto.URL,
			Title:      dto.Title,
			Subtitle:   dto.Subtitle,
			Footer:     dto.Footer,
			Header:     dto.Header,
			Response:   dto.Response,
			Server:     dto.Server,
			XPoweredBy: dto.XPoweredBy,
			Body:       dto.Body,
			Match:      dto.Match,
			Enabled:    dto.Enabled,
			Source:     dto.Source,
		})
	}

	// 2. 转换 DTOs -> CPEs
	cpes := make([]*asset.AssetCPE, 0, len(stdData.CPEs))
	for _, dto := range stdData.CPEs {
		cpes = append(cpes, &asset.AssetCPE{
			Name:     dto.Name,
			Probe:    dto.Probe,
			MatchStr: dto.MatchStr,
			Vendor:   dto.Vendor,
			Product:  dto.Product,
			Version:  dto.Version,
			Update:   dto.Update,
			Edition:  dto.Edition,
			Language: dto.Language,
			Part:     dto.Part,
			CPE:      dto.CPE,
			Enabled:  dto.Enabled,
			Source:   dto.Source,
		})
	}

	return fingers, cpes, nil
}
