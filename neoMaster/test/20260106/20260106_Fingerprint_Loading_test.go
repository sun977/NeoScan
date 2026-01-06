package test

import (
	"context"
	"path/filepath"
	"testing"

	"neomaster/internal/service/fingerprint"
	"neomaster/internal/service/fingerprint/engines/http"
	"neomaster/internal/service/fingerprint/engines/service"

	"github.com/stretchr/testify/assert"
)

func TestFingerprintLoading(t *testing.T) {
	// 1. 初始化引擎
	httpEngine := http.NewHTTPEngine(nil)
	serviceEngine := service.NewServiceEngine(nil)

	// 2. 初始化服务
	fpService := fingerprint.NewFingerprintService(httpEngine, serviceEngine)

	// 3. 加载规则目录 (使用项目中的实际规则文件)
	// 假设测试运行在 neoMaster 目录下
	ruleDir := filepath.Join("..", "..", "rules", "fingerprint")
	// 注意: 如果运行目录不是根目录，需要调整相对路径
	// 更好的方式是获取绝对路径
	absRuleDir, _ := filepath.Abs(ruleDir)
	t.Logf("Loading rules from: %s", absRuleDir)

	err := fpService.LoadRules(absRuleDir)
	assert.NoError(t, err)

	// 4. 验证统计信息
	stats := fpService.GetStats()
	t.Logf("Stats: %v", stats)
	assert.Equal(t, 2, stats["engines"])

	// 5. 验证是否能匹配加载的规则
	ctx := context.Background()

	// 5.1 验证 Custom HTTP Rule (WordPress)
	// 根据 custom.json 中的规则: header="Just another WordPress site"
	input1 := &fingerprint.Input{
		Headers: map[string]string{"X-Desc": "Just another WordPress site"},
	}
	res1, err := fpService.Identify(ctx, input1)
	assert.NoError(t, err)
	assert.NotNil(t, res1)
	if res1 != nil && res1.Best != nil {
		t.Logf("Matched: %s (%s)", res1.Best.Product, res1.Best.Source)
		assert.Equal(t, "WordPress", res1.Best.Product)
	} else {
		t.Error("Failed to match WordPress from custom.json")
	}

	// 5.2 验证 Service Rule (SSH)
	// 根据 services.json 中的规则: (?i)^SSH-[\d\.]+-OpenSSH_([\w\.]+)
	input2 := &fingerprint.Input{
		Banner: "SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1",
	}
	res2, err := fpService.Identify(ctx, input2)
	assert.NoError(t, err)
	assert.NotNil(t, res2)
	if res2 != nil && res2.Best != nil {
		t.Logf("Matched: %s (%s)", res2.Best.Product, res2.Best.CPE)
		assert.Equal(t, "openssh", res2.Best.Product)
		assert.Contains(t, res2.Best.CPE, "8.2p1")
	} else {
		t.Error("Failed to match SSH from services.json")
	}
}
