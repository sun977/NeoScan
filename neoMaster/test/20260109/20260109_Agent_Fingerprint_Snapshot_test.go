package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"neomaster/internal/config"
	agentService "neomaster/internal/service/agent"

	"github.com/stretchr/testify/assert"
)

func TestAgentFingerprintSnapshot_BuildIsDeterministic(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	assert.NoError(t, os.MkdirAll(filepath.Join(tmp, "nested"), 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(tmp, "a.json"), []byte("{\"a\":1}"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(tmp, "nested", "b.json"), []byte("{\"b\":2}"), 0o644))

	// 使用 Mock 配置，但实际上这里不需要配置，因为我们通过 ruleType 无法直接控制路径，
	// 但是 BuildSnapshot 内部如果没配置，会用 rulePathMap 或者默认 rules/type。
	// **问题**：原来的测试直接传入了 `tmp` 目录作为路径。
	// 现在的 agentService.BuildSnapshot 逻辑是：如果配置没给，就去 map 找，如果 map 没给，就去 `rules/{type}` 找。
	// 它不再接受任意路径参数！这是个 Breaking Change。
	// **解决方案**：为了测试，我们需要让 Service 知道指纹规则的路径是 `tmp`。
	// 我们可以 Mock Config，或者给 AgentUpdateService 加一个内部方法用于测试（不推荐破坏封装）。
	// 最好的方法是 Mock Config，让 GetFingerprintRulePath 返回 tmp。

	cfg := &config.Config{
		App: config.AppConfig{
			Rules: config.RulesConfig{
				RootPath: tmp,
				Fingerprint: config.RuleDirConfig{
					Dir: ".",
				},
			},
		},
	}
	svc := agentService.NewAgentUpdateService(cfg)

	s1, err := svc.BuildSnapshot(ctx, agentService.RuleTypeFingerprint)
	assert.NoError(t, err)
	assert.NotNil(t, s1)
	assert.NotEmpty(t, s1.VersionHash)
	assert.Equal(t, 2, s1.FileCount)
	assert.Equal(t, "application/zip", s1.ContentType)
	assert.Contains(t, s1.FileName, s1.VersionHash)
	assert.Greater(t, len(s1.Bytes), 0)

	s2, err := svc.BuildSnapshot(ctx, agentService.RuleTypeFingerprint)
	assert.NoError(t, err)
	assert.NotNil(t, s2)
	assert.Equal(t, s1.VersionHash, s2.VersionHash)
	assert.Equal(t, s1.FileCount, s2.FileCount)
	assert.Equal(t, s1.FileName, s2.FileName)
	assert.Equal(t, s1.Bytes, s2.Bytes)
}

func TestAgentFingerprintSnapshot_VersionChangesWhenRuleChanges(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	filePath := filepath.Join(tmp, "rule.json")
	assert.NoError(t, os.WriteFile(filePath, []byte("v1"), 0o644))
	assert.NoError(t, os.Chtimes(filePath, time.Unix(1, 0), time.Unix(1, 0)))

	cfg := &config.Config{
		App: config.AppConfig{
			Rules: config.RulesConfig{
				RootPath: tmp,
				Fingerprint: config.RuleDirConfig{
					Dir: ".",
				},
			},
		},
	}
	svc := agentService.NewAgentUpdateService(cfg)

	s1, err := svc.BuildSnapshot(ctx, agentService.RuleTypeFingerprint)
	assert.NoError(t, err)
	assert.NotNil(t, s1)
	assert.NotEmpty(t, s1.VersionHash)

	assert.NoError(t, os.WriteFile(filePath, []byte("v2"), 0o644))
	assert.NoError(t, os.Chtimes(filePath, time.Unix(2, 0), time.Unix(2, 0)))

	s2, err := svc.BuildSnapshot(ctx, agentService.RuleTypeFingerprint)
	assert.NoError(t, err)
	assert.NotNil(t, s2)
	assert.NotEmpty(t, s2.VersionHash)

	assert.NotEqual(t, s1.VersionHash, s2.VersionHash)
}
