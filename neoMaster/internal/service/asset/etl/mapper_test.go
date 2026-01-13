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

	bundles, err := MapToAssetBundles(result)
	assert.NoError(t, err)
	assert.NotEmpty(t, bundles)
	bundle := bundles[0]

	// Verify Host
	assert.Equal(t, "1.2.3.4", bundle.Host.IP)

	// Verify Web
	assert.Len(t, bundle.WebAssets, 1)
	wa := bundle.WebAssets[0]
	assert.NotNil(t, wa)
	assert.NotNil(t, wa.Web)
	web := wa.Web
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
	assert.NotNil(t, wa.Detail)
	detail := wa.Detail
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

	bundles, err := MapToAssetBundles(result)
	assert.NoError(t, err)
	assert.NotEmpty(t, bundles)
	bundle := bundles[0]

	assert.Equal(t, "192.168.1.100", bundle.Host.IP)
	assert.Len(t, bundle.WebAssets, 1)
	assert.NotNil(t, bundle.WebAssets[0])
	assert.NotNil(t, bundle.WebAssets[0].Web)
	assert.Equal(t, "192.168.1.100", bundle.WebAssets[0].Web.Domain)
}

func TestMapIPAlive(t *testing.T) {
	jsonAttr := `{
		"hosts": [
			{"ip": "192.168.1.10", "rtt": 0.45, "ttl": 64, "hostname": "test-host", "os": "Linux"},
			{"ip": "192.168.1.11", "rtt": 1.20, "ttl": 128}
		],
		"summary": {
			"alive_count": 2,
			"total_scanned": 256,
			"elapsed_ms": 1500
		}
	}`

	result := &orcModel.StageResult{
		ResultType:  "ip_alive",
		TargetValue: "192.168.1.0/24",
		Attributes:  jsonAttr,
	}

	bundles, err := MapToAssetBundles(result)
	assert.NoError(t, err)
	assert.Len(t, bundles, 2)

	// Verify First Bundle
	assert.Equal(t, "192.168.1.10", bundles[0].Host.IP)
	assert.Equal(t, "test-host", bundles[0].Host.Hostname)
	assert.Equal(t, "Linux", bundles[0].Host.OS)

	// Verify Second Bundle
	assert.Equal(t, "192.168.1.11", bundles[1].Host.IP)
	assert.Empty(t, bundles[1].Host.Hostname)
	assert.Empty(t, bundles[1].Host.OS)
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

	bundles, err := MapToAssetBundles(result)
	assert.NoError(t, err)
	assert.NotEmpty(t, bundles)
	bundle := bundles[0]

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

	bundles, err := MapToAssetBundles(result)
	assert.NoError(t, err)
	assert.NotEmpty(t, bundles)
	bundle := bundles[0]

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

	bundles, err := MapToAssetBundles(result)
	assert.NoError(t, err)
	assert.NotEmpty(t, bundles)
	bundle := bundles[0]

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
}

func TestMapPortScan_Batch(t *testing.T) {
	jsonAttr := `{
		"ports": [
			{"ip": "192.168.1.10", "port": 80, "proto": "tcp", "state": "open"},
			{"ip": "192.168.1.11", "port": 22, "proto": "tcp", "state": "open"}
		]
	}`

	result := &orcModel.StageResult{
		ResultType:  "fast_port_scan",
		TargetValue: "192.168.1.0/24",
		Attributes:  jsonAttr,
	}

	bundles, err := MapToAssetBundles(result)
	assert.NoError(t, err)
	assert.Len(t, bundles, 2)

	// Check bundling
	ips := make(map[string]int)
	for _, b := range bundles {
		ips[b.Host.IP] = len(b.Services)
	}

	assert.Equal(t, 1, ips["192.168.1.10"])
	assert.Equal(t, 1, ips["192.168.1.11"])
}

func TestMapWebEndpoint_Batch(t *testing.T) {
	jsonAttr := `{
		"endpoints": [
			{
				"url": "http://example.com/login",
				"ip": "1.2.3.4",
				"title": "Login Page"
			},
			{
				"url": "http://admin.internal/login",
				"ip": "10.0.0.1",
				"title": "Admin Login"
			}
		]
	}`

	result := &orcModel.StageResult{
		ResultType:  "web_endpoint",
		TargetValue: "batch-scan",
		Attributes:  jsonAttr,
	}

	bundles, err := MapToAssetBundles(result)
	assert.NoError(t, err)
	assert.Len(t, bundles, 2)

	ips := make(map[string]int)
	for _, b := range bundles {
		ips[b.Host.IP] = len(b.WebAssets)
	}

	assert.Equal(t, 1, ips["1.2.3.4"])
	assert.Equal(t, 1, ips["10.0.0.1"])
}

func TestMapPasswordAudit(t *testing.T) {
	jsonAttr := `{
		"accounts": [
			{"username": "admin", "service": "ssh", "host": "192.168.1.10", "port": 22, "weak_password": true, "credential": "admin:123456", "success": true},
			{"username": "root", "service": "ssh", "host": "192.168.1.10", "port": 22, "weak_password": false, "success": false},
			{"username": "dbuser", "service": "mysql", "host": "192.168.1.11", "port": 3306, "weak_password": true, "credential": "dbuser:password", "success": true}
		]
	}`

	result := &orcModel.StageResult{
		ResultType:  "password_audit",
		TargetValue: "192.168.1.0/24",
		Attributes:  jsonAttr,
	}

	bundles, err := MapToAssetBundles(result)
	assert.NoError(t, err)
	assert.Len(t, bundles, 2)

	// Verify First Bundle (192.168.1.10)
	var b1 *AssetBundle
	for _, b := range bundles {
		if b.Host.IP == "192.168.1.10" {
			b1 = b
			break
		}
	}
	assert.NotNil(t, b1)
	assert.Len(t, b1.Vulns, 1) // Only 1 vuln record (SSH weak password) containing 2 accounts

	sshVuln := b1.Vulns[0]
	assert.Equal(t, "high", sshVuln.Severity)
	assert.Equal(t, "open", sshVuln.Status)
	assert.Equal(t, "neosc:neosc-rules:weak-password:ssh", sshVuln.IDAlias)

	var sshAttr map[string]interface{}
	err = json.Unmarshal([]byte(sshVuln.Attributes), &sshAttr)
	assert.NoError(t, err)

	accounts, ok := sshAttr["accounts"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, accounts, 2) // admin and root

	// Verify Second Bundle (192.168.1.11)
	var b2 *AssetBundle
	for _, b := range bundles {
		if b.Host.IP == "192.168.1.11" {
			b2 = b
			break
		}
	}
	assert.NotNil(t, b2)
	assert.Len(t, b2.Vulns, 1)
	mysqlVuln := b2.Vulns[0]
	assert.Equal(t, "high", mysqlVuln.Severity)
	assert.Equal(t, "neosc:neosc-rules:weak-password:mysql", mysqlVuln.IDAlias)
}
