package converters

import (
	"encoding/json"
	"testing"

	"neomaster/internal/model/asset"

	"github.com/stretchr/testify/assert"
)

func TestStandardJSONConverter_Encode_Decode(t *testing.T) {
	converter := NewStandardJSONConverter()

	// 1. 准备测试数据
	fingers := []*asset.AssetFinger{
		{
			Name:       "WordPress",
			StatusCode: "200",
			URL:        "/wp-login.php",
			Title:      "Log In",
			Match:      "regex:wp-.*",
			Enabled:    true,
			Source:     "system",
		},
		{
			Name:    "CustomApp",
			Header:  "X-Custom-App",
			Enabled: false,
			Source:  "custom",
		},
	}

	cpes := []*asset.AssetCPE{
		{
			Name:     "Nginx",
			Probe:    "HTTP",
			MatchStr: "^nginx/(.*)$",
			Vendor:   "nginx",
			Product:  "nginx",
			Version:  "$1",
			Part:     "a",
			CPE:      "cpe:/a:nginx:nginx:$1",
			Enabled:  true,
			Source:   "system",
		},
		{
			Name:     "CustomDevice",
			Probe:    "TCP",
			MatchStr: "device_v1",
			Part:     "h",
			Enabled:  true,
			Source:   "custom",
		},
	}

	// 2. 测试 Encode
	data, err := converter.Encode(fingers, cpes)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// 验证 JSON 结构
	var stdData StandardJSON
	err = json.Unmarshal(data, &stdData)
	assert.NoError(t, err)
	assert.Equal(t, "1.0", stdData.Version)
	assert.Equal(t, "NeoScan Export", stdData.Source)
	assert.Len(t, stdData.Fingers, 2)
	assert.Len(t, stdData.CPEs, 2)

	// 验证 Encode 细节 (特别是 Source 字段)
	assert.Equal(t, "WordPress", stdData.Fingers[0].Name)
	assert.Equal(t, "system", stdData.Fingers[0].Source)
	assert.Equal(t, "CustomApp", stdData.Fingers[1].Name)
	assert.Equal(t, "custom", stdData.Fingers[1].Source)

	assert.Equal(t, "Nginx", stdData.CPEs[0].Name)
	assert.Equal(t, "system", stdData.CPEs[0].Source)
	assert.Equal(t, "CustomDevice", stdData.CPEs[1].Name)
	assert.Equal(t, "custom", stdData.CPEs[1].Source)

	// 3. 测试 Decode
	decodedFingers, decodedCPEs, err := converter.Decode(data)
	assert.NoError(t, err)

	// 验证 Decode 结果与原始数据一致
	assert.Len(t, decodedFingers, 2)
	assert.Len(t, decodedCPEs, 2)

	// Finger 1
	assert.Equal(t, fingers[0].Name, decodedFingers[0].Name)
	assert.Equal(t, fingers[0].StatusCode, decodedFingers[0].StatusCode)
	assert.Equal(t, fingers[0].URL, decodedFingers[0].URL)
	assert.Equal(t, fingers[0].Title, decodedFingers[0].Title)
	assert.Equal(t, fingers[0].Match, decodedFingers[0].Match)
	assert.Equal(t, fingers[0].Enabled, decodedFingers[0].Enabled)
	assert.Equal(t, fingers[0].Source, decodedFingers[0].Source)

	// Finger 2
	assert.Equal(t, fingers[1].Name, decodedFingers[1].Name)
	assert.Equal(t, fingers[1].Header, decodedFingers[1].Header)
	assert.Equal(t, fingers[1].Enabled, decodedFingers[1].Enabled)
	assert.Equal(t, fingers[1].Source, decodedFingers[1].Source)

	// CPE 1
	assert.Equal(t, cpes[0].Name, decodedCPEs[0].Name)
	assert.Equal(t, cpes[0].Probe, decodedCPEs[0].Probe)
	assert.Equal(t, cpes[0].MatchStr, decodedCPEs[0].MatchStr)
	assert.Equal(t, cpes[0].Vendor, decodedCPEs[0].Vendor)
	assert.Equal(t, cpes[0].Product, decodedCPEs[0].Product)
	assert.Equal(t, cpes[0].Version, decodedCPEs[0].Version)
	assert.Equal(t, cpes[0].Part, decodedCPEs[0].Part)
	assert.Equal(t, cpes[0].CPE, decodedCPEs[0].CPE)
	assert.Equal(t, cpes[0].Enabled, decodedCPEs[0].Enabled)
	assert.Equal(t, cpes[0].Source, decodedCPEs[0].Source)

	// CPE 2
	assert.Equal(t, cpes[1].Name, decodedCPEs[1].Name)
	assert.Equal(t, cpes[1].Probe, decodedCPEs[1].Probe)
	assert.Equal(t, cpes[1].MatchStr, decodedCPEs[1].MatchStr)
	assert.Equal(t, cpes[1].Part, decodedCPEs[1].Part)
	assert.Equal(t, cpes[1].Enabled, decodedCPEs[1].Enabled)
	assert.Equal(t, cpes[1].Source, decodedCPEs[1].Source)
}

func TestStandardJSONConverter_Decode_EmptySource(t *testing.T) {
	// 测试向后兼容性：如果 JSON 中没有 source 字段
	jsonStr := `{
  "version": "1.0",
  "source": "NeoScan Export",
  "fingers": [
    {
      "name": "OldRule",
      "enabled": true
    }
  ],
  "cpes": []
}`
	converter := NewStandardJSONConverter()
	fingers, _, err := converter.Decode([]byte(jsonStr))
	assert.NoError(t, err)
	assert.Len(t, fingers, 1)
	assert.Equal(t, "OldRule", fingers[0].Name)
	assert.Equal(t, "", fingers[0].Source) // 应该是空字符串
}
