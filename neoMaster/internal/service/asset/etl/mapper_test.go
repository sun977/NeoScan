package etl

import (
	"encoding/json"
	"testing"

	orcModel "neomaster/internal/model/orchestrator"

	"github.com/stretchr/testify/assert"
)

func TestMapWebEndpoint(t *testing.T) {
	// 1. Test with full attributes including IP
	jsonAttr := `{
		"url": "http://example.com/login",
		"ip": "1.2.3.4",
		"title": "Login Page",
		"status_code": 200,
		"tech_stack": ["Nginx", "React"],
		"headers": {"Server": "Nginx"},
		"screenshot": "base64-mock"
	}`

	result := &orcModel.StageResult{
		ResultType:  "web_endpoint",
		TargetValue: "example.com", // Target was domain
		Attributes:  jsonAttr,
	}

	bundle, err := MapToAssetBundle(result)
	assert.NoError(t, err)
	assert.NotNil(t, bundle)

	// Verify Host
	assert.Equal(t, "1.2.3.4", bundle.Host.IP)

	// Verify Web
	assert.Len(t, bundle.Webs, 1)
	web := bundle.Webs[0]
	assert.Equal(t, "http://example.com/login", web.URL)
	assert.Equal(t, "example.com", web.Domain)
	assert.Contains(t, web.TechStack, "React")

	// Verify BasicInfo
	var basicInfo map[string]interface{}
	err = json.Unmarshal([]byte(web.BasicInfo), &basicInfo)
	assert.NoError(t, err)
	assert.Equal(t, "Login Page", basicInfo["title"])
	assert.Equal(t, float64(200), basicInfo["status_code"]) // JSON numbers are floats

	// Verify WebDetails
	assert.Len(t, bundle.WebDetails, 1)
	detail := bundle.WebDetails[0]
	assert.Equal(t, "base64-mock", detail.Screenshot)

	// Verify ContentDetails
	var contentDetails map[string]interface{}
	err = json.Unmarshal([]byte(detail.ContentDetails), &contentDetails)
	assert.NoError(t, err)
	assert.Equal(t, "http://example.com/login", contentDetails["url"])

	// Response headers are map in interface{}, assert properly
	respHeaders, ok := contentDetails["response_headers"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Nginx", respHeaders["Server"])
}

func TestMapWebEndpoint_InferIP(t *testing.T) {
	// 2. Test without explicit IP in attributes, infer from TargetValue
	jsonAttr := `{
		"url": "http://192.168.1.100:8080/",
		"title": "Admin"
	}`

	result := &orcModel.StageResult{
		ResultType:  "web_endpoint",
		TargetValue: "192.168.1.100",
		Attributes:  jsonAttr,
	}

	bundle, err := MapToAssetBundle(result)
	assert.NoError(t, err)

	assert.Equal(t, "192.168.1.100", bundle.Host.IP)
	assert.Equal(t, "192.168.1.100", bundle.Webs[0].Domain) // extractDomain might return hostname for IP too
}

func TestMapWebEndpoint_InferIP_FromURLTarget(t *testing.T) {
	// 3. Test when TargetValue is URL
	jsonAttr := `{
		"url": "http://test.com"
	}`

	result := &orcModel.StageResult{
		ResultType:  "web_endpoint",
		TargetValue: "http://test.com",
		Attributes:  jsonAttr,
	}

	bundle, err := MapToAssetBundle(result)
	assert.NoError(t, err)

	assert.Equal(t, "test.com", bundle.Host.IP) // Hostname extraction
}
