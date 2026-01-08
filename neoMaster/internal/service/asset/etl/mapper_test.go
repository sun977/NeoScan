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
		"endpoints": [
			{
				"url": "http://example.com/login",
				"ip": "1.2.3.4",
				"title": "Login Page",
				"status_code": 200,
				"tech_stack": ["Nginx", "React"],
				"headers": {"Server": "Nginx"},
				"screenshot": "base64-mock"
			}
		]
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
		"endpoints": [
			{
				"url": "http://192.168.1.100:8080/",
				"title": "Admin"
			}
		]
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
		"endpoints": [
			{
				"url": "http://test.com"
			}
		]
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

func TestMapVulnFinding(t *testing.T) {
	jsonAttr := `{
		"findings": [
			{
				"id": "SCAN-123",
				"cve": "CVE-2021-44228",
				"name": "Log4j RCE",
				"type": "RCE",
				"severity": "critical",
				"description": "Log4j RCE vulnerability",
				"confidence": 0.95,
				"target_type": "service",
				"port": 8080,
				"evidence": "Payload: ${jndi:ldap://...}"
			}
		]
	}`

	result := &orcModel.StageResult{
		ResultType:  "vuln_finding",
		TargetValue: "10.0.0.1",
		Attributes:  jsonAttr,
	}

	bundle, err := MapToAssetBundle(result)
	assert.NoError(t, err)
	assert.NotNil(t, bundle)

	// Verify Host
	assert.Equal(t, "10.0.0.1", bundle.Host.IP)

	// Verify Vuln
	assert.Len(t, bundle.Vulns, 1)
	vuln := bundle.Vulns[0]
	assert.Equal(t, "CVE-2021-44228", vuln.CVE)
	assert.Equal(t, "SCAN-123", vuln.IDAlias)
	assert.Equal(t, "critical", vuln.Severity)
	assert.Equal(t, "service", vuln.TargetType)
	assert.Equal(t, 0.95, vuln.Confidence)
	assert.Contains(t, vuln.Evidence, "${jndi:ldap://...}")

	// Verify Attributes
	var attr map[string]interface{}
	err = json.Unmarshal([]byte(vuln.Attributes), &attr)
	assert.NoError(t, err)
	assert.Equal(t, "Log4j RCE", attr["name"])
	assert.Equal(t, float64(8080), attr["port"])
	assert.Equal(t, "10.0.0.1", attr["ip"])
}

func TestMapPocScan(t *testing.T) {
	jsonAttr := `{
		"poc_results": [
			{
				"poc_id": "CVE-2022-1234",
				"target": "http://10.0.0.1:8080/vulnerable",
				"status": "confirmed",
				"severity": "high",
				"evidence_ref": "evidence/123.txt"
			},
			{
				"poc_id": "CVE-2022-5678",
				"target": "10.0.0.1:22",
				"status": "not_vulnerable",
				"severity": "medium",
				"evidence_ref": "evidence/456.txt"
			}
		]
	}`

	result := &orcModel.StageResult{
		ResultType:  "poc_scan",
		TargetValue: "10.0.0.1",
		Attributes:  jsonAttr,
	}

	bundle, err := MapToAssetBundle(result)
	assert.NoError(t, err)
	assert.NotNil(t, bundle)

	// Verify Host
	assert.Equal(t, "10.0.0.1", bundle.Host.IP)

	// Verify Vulns
	assert.Len(t, bundle.Vulns, 1) // Only confirmed one
	vuln := bundle.Vulns[0]

	assert.Equal(t, "CVE-2022-1234", vuln.CVE)
	assert.Equal(t, "CVE-2022-1234", vuln.IDAlias)
	assert.Equal(t, "high", vuln.Severity)
	assert.Equal(t, 100.0, vuln.Confidence)
	assert.Equal(t, "web", vuln.TargetType) // deduced from http prefix
	assert.Equal(t, "verified", vuln.VerifyStatus)
	assert.Equal(t, "poc_scanner", vuln.VerifiedBy)
	assert.NotNil(t, vuln.VerifiedAt)

	// Verify Attributes
	var attr map[string]interface{}
	err = json.Unmarshal([]byte(vuln.Attributes), &attr)
	assert.NoError(t, err)
	assert.Equal(t, "CVE-2022-1234", attr["poc_id"])
	assert.Equal(t, "http://10.0.0.1:8080/vulnerable", attr["target"])
}
