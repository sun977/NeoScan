package converters

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// LegacyCMSRule 定义旧版 CMS 规则结构
type LegacyCMSRule struct {
	Name string `json:"name"`
	Rule struct {
		StatusCode string `json:"status_code"`
		URL        string `json:"url"`
		Title      string `json:"title"`
		Subtitle   string `json:"subtitle"`
		Footer     string `json:"footer"`
		Header     string `json:"header"`
		Response   string `json:"response"`
		Server     string `json:"server"`
		XPoweredBy string `json:"x_powered_by"`
	} `json:"rule"`
}

// LegacyCMSFile 定义旧版 CMS 文件结构
type LegacyCMSFile struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Type    string          `json:"type"`
	Samples []LegacyCMSRule `json:"samples"`
}

// LegacyServiceRule 定义旧版服务规则结构
type LegacyServiceRule struct {
	MatchStr string `json:"match_str"`
	Vendor   string `json:"vendor"`
	Product  string `json:"product"`
	Part     string `json:"part"`
	CPE      string `json:"cpe"`
}

// LegacyServiceFile 定义旧版服务文件结构
type LegacyServiceFile struct {
	Name    string              `json:"name"`
	Version string              `json:"version"`
	Type    string              `json:"type"`
	Samples []LegacyServiceRule `json:"samples"`
}

func TestStandardJSONConverter_LegacyCompatibility(t *testing.T) {
	// 1. 读取 Legacy CMS 文件
	cmsData, err := os.ReadFile("../../../../../rules/fingerprint/system/cms/default_cms.json")
	if err != nil {
		t.Skip("Skipping legacy test: default_cms.json not found")
	}

	// 2. 读取 Legacy Service 文件
	serviceData, err := os.ReadFile("../../../../../rules/fingerprint/system/service/default_service.json")
	if err != nil {
		t.Skip("Skipping legacy test: default_service.json not found")
	}

	converter := NewStandardJSONConverter()

	// 3. 测试 StandardJSONConverter 是否能解析 Legacy CMS 格式
	// 注意：StandardJSONConverter 目前设计为解析 StandardJSON 格式
	// Legacy 格式与 StandardJSON 结构完全不同，StandardJSONConverter.Decode 预期会失败
	// 但我们需要确认它如何失败，或者是否需要我们添加适配层

	// 尝试直接 Decode CMS 数据
	_, _, err = converter.Decode(cmsData)
	// 这里我们预期它可能会失败，或者解析出空数据，因为结构不匹配
	// 如果我们希望支持 Legacy 格式，我们需要修改 StandardJSONConverter 或提供专门的转换逻辑

	// 让我们先手动解析 Legacy 格式，验证我们对 Legacy 结构的理解是正确的
	var legacyCMS LegacyCMSFile
	err = json.Unmarshal(cmsData, &legacyCMS)
	assert.NoError(t, err, "Should be able to unmarshal legacy CMS file")
	assert.NotEmpty(t, legacyCMS.Samples)

	// 验证第一条规则 (WordPress)
	wp := legacyCMS.Samples[0]
	assert.Equal(t, "WordPress", wp.Name)
	assert.Equal(t, "200", wp.Rule.StatusCode)

	// 让我们手动解析 Legacy Service 格式
	var legacyService LegacyServiceFile
	err = json.Unmarshal(serviceData, &legacyService)
	assert.NoError(t, err, "Should be able to unmarshal legacy Service file")
	assert.NotEmpty(t, legacyService.Samples)

	// 验证第一条规则 (OpenSSH)
	ssh := legacyService.Samples[0]
	assert.Contains(t, ssh.MatchStr, "SSH")
	assert.Equal(t, "openssh", ssh.Product)

	// 结论：Legacy 格式确实与 StandardJSON 格式不同。
	// StandardJSON 格式期望顶层有 "fingers" 和 "cpes" 字段。
	// Legacy CMS 格式顶层有 "samples" 且内部结构为 name + rule 对象。
	// Legacy Service 格式顶层有 "samples" 且内部结构直接为规则字段。

	// 如果用户希望导入这些 Legacy 文件，我们需要在 StandardJSONConverter 中添加逻辑
	// 或者在 RuleManager 中处理。
	// 鉴于 StandardJSONConverter 的职责是 DB <-> StandardJSON，
	// 我们可能需要一个 DecodeLegacy 方法。
}
