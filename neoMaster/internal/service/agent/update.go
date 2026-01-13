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

type FingerprintSnapshotInfo struct {
	VersionHash string `json:"version_hash"`
	RulePath    string `json:"rule_path"`
	FileCount   int    `json:"file_count"`
}

type FingerprintSnapshot struct {
	FingerprintSnapshotInfo
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Bytes       []byte `json:"-"`
}

type AgentUpdateService interface {
	GetFingerprintSnapshotInfo(ctx context.Context) (*FingerprintSnapshotInfo, error)
	BuildFingerprintSnapshot(ctx context.Context) (*FingerprintSnapshot, error)
}

type agentUpdateService struct {
	cfg *config.Config
}

func NewAgentUpdateService(cfg *config.Config) AgentUpdateService {
	return &agentUpdateService{cfg: cfg}
}

func (s *agentUpdateService) GetFingerprintSnapshotInfo(ctx context.Context) (*FingerprintSnapshotInfo, error) {
	snap, err := s.BuildFingerprintSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return &snap.FingerprintSnapshotInfo, nil
}

func (s *agentUpdateService) BuildFingerprintSnapshot(ctx context.Context) (*FingerprintSnapshot, error) {
	rulePath := ""
	if s.cfg != nil {
		rulePath = s.cfg.GetFingerprintRulePath()
	}

	resolvedPath := strings.TrimSpace(rulePath)
	if resolvedPath == "" {
		resolvedPath = "rules/fingerprint"
	}

	filePaths, err := listRuleFiles(resolvedPath)
	if err != nil {
		return nil, err
	}

	zipBytes, err := buildDeterministicZip(ctx, resolvedPath, filePaths)
	if err != nil {
		return nil, err
	}

	h := md5.Sum(zipBytes)
	version := hex.EncodeToString(h[:])

	snapshot := &FingerprintSnapshot{
		FingerprintSnapshotInfo: FingerprintSnapshotInfo{
			VersionHash: version,
			RulePath:    resolvedPath,
			FileCount:   len(filePaths),
		},
		FileName:    fmt.Sprintf("fingerprint_snapshot_%s.zip", version),
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
