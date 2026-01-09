package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"neomaster/internal/service/agent_update"

	"github.com/stretchr/testify/assert"
)

func TestAgentFingerprintSnapshot_BuildIsDeterministic(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	assert.NoError(t, os.MkdirAll(filepath.Join(tmp, "nested"), 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(tmp, "a.json"), []byte("{\"a\":1}"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(tmp, "nested", "b.json"), []byte("{\"b\":2}"), 0o644))

	s1, err := agent_update.BuildFingerprintSnapshot(ctx, tmp)
	assert.NoError(t, err)
	assert.NotNil(t, s1)
	assert.NotEmpty(t, s1.VersionHash)
	assert.Equal(t, 2, s1.FileCount)
	assert.Equal(t, "application/zip", s1.ContentType)
	assert.Contains(t, s1.FileName, s1.VersionHash)
	assert.Greater(t, len(s1.Bytes), 0)

	s2, err := agent_update.BuildFingerprintSnapshot(ctx, tmp)
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

	s1, err := agent_update.BuildFingerprintSnapshot(ctx, tmp)
	assert.NoError(t, err)
	assert.NotNil(t, s1)
	assert.NotEmpty(t, s1.VersionHash)

	assert.NoError(t, os.WriteFile(filePath, []byte("v2"), 0o644))
	assert.NoError(t, os.Chtimes(filePath, time.Unix(2, 0), time.Unix(2, 0)))

	s2, err := agent_update.BuildFingerprintSnapshot(ctx, tmp)
	assert.NoError(t, err)
	assert.NotNil(t, s2)
	assert.NotEmpty(t, s2.VersionHash)

	assert.NotEqual(t, s1.VersionHash, s2.VersionHash)
}
