package test

import (
	"context"
	"os"
	"strings"
	"testing"

	"neomaster/internal/service/fingerprint"
	"neomaster/internal/service/fingerprint/engines/http"
	"neomaster/internal/service/fingerprint/engines/service"

	"github.com/stretchr/testify/assert"
)

func TestFingerprintService(t *testing.T) {
	// 1. 初始化引擎
	httpEngine := http.NewHTTPEngine(nil)          // Mock DB repo as nil
	serviceEngine := service.NewServiceEngine(nil) // Mock DB repo as nil

	// 2. 初始化服务
	fpService := fingerprint.NewFingerprintService(httpEngine, serviceEngine)

	// 3. 加载测试规则
	// 3.1 Goby
	gobyRuleContent := `{
		"rule": [
			{
				"name": "ThinkPHP",
				"rule": "header=\"thinkphp\" || body=\"thinkphp\"",
				"product": "ThinkPHP",
				"company": "ThinkPHP",
				"category": "framework"
			}
		]
	}`
	tmpGobyFile := "temp_goby_rules.json"
	os.WriteFile(tmpGobyFile, []byte(gobyRuleContent), 0644)
	defer os.Remove(tmpGobyFile)
	_ = httpEngine.LoadRules(tmpGobyFile)

	// 3.2 CMS
	cmsRuleContent := `{
		"name": "CMS Sample",
		"version": "2.0.0",
		"samples": [
			{
				"id": 1,
				"name": "WordPress",
				"rule": {
					"header": "Just another WordPress site"
				}
			},
			{
				"id": 2,
				"name": "VenusTech Gateway",
				"rule": {
					"title": "天玥运维安全网关"
				}
			}
		]
	}`
	tmpCMSFile := "temp_cms_rules.json"
	os.WriteFile(tmpCMSFile, []byte(cmsRuleContent), 0644)
	defer os.Remove(tmpCMSFile)
	_ = httpEngine.LoadRules(tmpCMSFile)

	ctx := context.Background()

	// 4. 测试用例
	cases := []struct {
		Name           string
		Input          *fingerprint.Input
		ExpectedProd   string
		ExpectedVendor string
	}{
		{
			Name: "HTTP Match - ThinkPHP",
			Input: &fingerprint.Input{
				Headers: map[string]string{"X-Powered-By": "thinkphp"},
			},
			ExpectedProd:   "ThinkPHP",
			ExpectedVendor: "ThinkPHP",
		},
		{
			Name: "Service Match - Nginx",
			Input: &fingerprint.Input{
				Banner: "nginx/1.18.0",
			},
			ExpectedProd:   "nginx",
			ExpectedVendor: "f5",
		},
		{
			Name: "CMS Match - WordPress",
			Input: &fingerprint.Input{
				Headers: map[string]string{"X-Description": "Just another WordPress site"},
			},
			ExpectedProd:   "WordPress",
			ExpectedVendor: "wordpress",
		},
		{
			Name: "CMS Match - VenusTech",
			Input: &fingerprint.Input{
				Body: "<html><head><title>天玥运维安全网关</title></head></html>",
			},
			ExpectedProd:   "VenusTech Gateway",
			ExpectedVendor: "venustech gateway",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result, err := fpService.Identify(ctx, c.Input)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotNil(t, result.Best)

			assert.Equal(t, c.ExpectedProd, result.Best.Product)
			if c.ExpectedVendor != "" {
				// 模糊匹配 vendor (大小写/空格)
				assert.Contains(t, strings.ToLower(result.Best.Vendor), strings.ToLower(c.ExpectedVendor))
			}
			assert.Contains(t, result.Best.CPE, "cpe:2.3:")
		})
	}
}
