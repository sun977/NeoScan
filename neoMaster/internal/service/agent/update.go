package agent

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"neomaster/internal/config"
)

// RuleType 定义规则库类型
type RuleType string

const (
	RuleTypeFingerprint RuleType = "fingerprint"
	RuleTypePOC         RuleType = "poc"
	RuleTypeVirus       RuleType = "virus"
	RuleTypeWebShell    RuleType = "webshell"
)

// rulePathMap 定义规则类型到文件系统路径的映射
// 将来如果有新规则类型，只需在此处添加
var rulePathMap = map[RuleType]string{
	RuleTypeFingerprint: "rules/fingerprint",
	RuleTypePOC:         "rules/poc",
	RuleTypeVirus:       "rules/virus",
	RuleTypeWebShell:    "rules/webshell",
}

// RuleSnapshotInfo 通用的规则快照信息
type RuleSnapshotInfo struct {
	Type        RuleType `json:"type"`         // 规则类型
	VersionHash string   `json:"version_hash"` // 版本哈希
	RulePath    string   `json:"rule_path"`    // 规则路径
	FileCount   int      `json:"file_count"`   // 文件数量
}

// RuleSnapshot 规则快照，包含二进制数据
type RuleSnapshot struct {
	RuleSnapshotInfo
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Bytes       []byte `json:"-"`
}

// AgentUpdateService Agent更新服务接口
// 负责构建和获取各种类型的规则库快照
type AgentUpdateService interface {
	GetSnapshotInfo(ctx context.Context, ruleType RuleType) (*RuleSnapshotInfo, error)
	BuildSnapshot(ctx context.Context, ruleType RuleType) (*RuleSnapshot, error)
}

type agentUpdateService struct {
	cfg *config.Config
}

func NewAgentUpdateService(cfg *config.Config) AgentUpdateService {
	return &agentUpdateService{cfg: cfg}
}

// GetSnapshotInfo 获取指定类型规则的快照信息
func (s *agentUpdateService) GetSnapshotInfo(ctx context.Context, ruleType RuleType) (*RuleSnapshotInfo, error) {
	snap, err := s.BuildSnapshot(ctx, ruleType)
	if err != nil {
		return nil, err
	}
	return &snap.RuleSnapshotInfo, nil
}

// BuildSnapshot 构建指定类型规则的快照
func (s *agentUpdateService) BuildSnapshot(ctx context.Context, ruleType RuleType) (*RuleSnapshot, error) {
	// 1. 获取基础路径
	// 优先从配置获取，如果配置没有特定的，则使用默认映射
	// 目前配置似乎只针对 fingerprint 做了特殊处理，未来可以扩展配置结构
	rulePath := ""
	if s.cfg != nil {
		if ruleType == RuleTypeFingerprint {
			rulePath = s.cfg.GetFingerprintRulePath()
		}
		// TODO: 支持其他类型的自定义配置路径
	}

	// 2. 如果配置为空，使用默认映射
	resolvedPath := strings.TrimSpace(rulePath)
	if resolvedPath == "" {
		defaultPath, ok := rulePathMap[ruleType]
		if !ok {
			// 如果是未知的规则类型，我们可以报错，或者默认尝试 rules/{type}
			defaultPath = filepath.Join("rules", string(ruleType))
		}
		resolvedPath = defaultPath
	}

	// 3. 检查目录是否存在，不存在则尝试创建（或者报错，取决于策略）
	// 这里我们选择只读，如果目录不存在 listRuleFiles 会报错
	filePaths, err := listRuleFiles(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list rule files for type %s at %s: %w", ruleType, resolvedPath, err)
	}

	// 4. 构建确定性 ZIP
	zipBytes, err := buildDeterministicZip(ctx, resolvedPath, filePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to build zip for type %s: %w", ruleType, err)
	}

	// 5. 计算 Hash
	h := md5.Sum(zipBytes)
	version := hex.EncodeToString(h[:])

	snapshot := &RuleSnapshot{
		RuleSnapshotInfo: RuleSnapshotInfo{
			Type:        ruleType,
			VersionHash: version,
			RulePath:    resolvedPath,
			FileCount:   len(filePaths),
		},
		FileName:    fmt.Sprintf("%s_snapshot_%s.zip", ruleType, version),
		ContentType: "application/zip",
		Bytes:       zipBytes,
	}
	return snapshot, nil
}

func listRuleFiles(root string) ([]string, error) {
	stat, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("rule path is not a directory: %s", root)
	}

	var relPaths []string

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rell, err1 := filepath.Rel(root, path)
		if err1 != nil {
			return err1
		}
		rell = filepath.ToSlash(rell)
		relPaths = append(relPaths, rell)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(relPaths)
	return relPaths, nil
}

func buildDeterministicZip(ctx context.Context, root string, relPaths []string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	zw := zip.NewWriter(buf)

	for _, rel := range relPaths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		abs := filepath.Join(root, filepath.FromSlash(rel))
		content, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}

		h := &zip.FileHeader{
			Name:   rel,
			Method: zip.Deflate,
		}
		h.SetModTime(time.Unix(0, 0))

		w, err := zw.CreateHeader(h)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(content); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
