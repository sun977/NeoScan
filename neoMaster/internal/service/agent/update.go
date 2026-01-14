package agent

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"neomaster/internal/config"
	"neomaster/internal/pkg/logger"
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
	Signature   string `json:"signature,omitempty"` // 规则签名 (HMAC-SHA256)
}

// SnapshotCacheItem 缓存项
// - 缓存结构 ： SnapshotCacheItem 存储了 {快照数据, 目录最后修改时间} 。
// - 命中逻辑 ：每次请求时，先快速遍历一遍目录树获取最新的 mtime （这个操作比读取内容快几个数量级）。如果 mtime 没变，直接返回内存中的快照。
// - 自动更新 ：一旦检测到 mtime 变大，立即触发重建流程，并更新缓存。
type SnapshotCacheItem struct {
	Snapshot    *RuleSnapshot // 缓存的快照
	LastModTime time.Time     // 缓存对应的目录最后修改时间
}

// AgentUpdateService Agent更新服务接口
// 负责构建和获取各种类型的规则库快照
type AgentUpdateService interface {
	GetSnapshotInfo(ctx context.Context, ruleType RuleType) (*RuleSnapshotInfo, error)  // 获取指定类型规则的快照信息
	BuildSnapshot(ctx context.Context, ruleType RuleType) (*RuleSnapshot, error)        // 构建规则快照
	GetEncryptedSnapshot(ctx context.Context, ruleType RuleType) (*RuleSnapshot, error) // 获取加密/签名的快照
}

type agentUpdateService struct {
	cfg   *config.Config
	cache map[RuleType]*SnapshotCacheItem // 内存缓存 缓存了 规则快照
	mu    sync.RWMutex                    // 读写锁保护缓存
}

func NewAgentUpdateService(cfg *config.Config) AgentUpdateService {
	return &agentUpdateService{
		cfg:   cfg,
		cache: make(map[RuleType]*SnapshotCacheItem),
	}
}

// GetSnapshotInfo 获取指定类型规则的快照信息
// 优化：优先读取缓存，如果文件系统有变动才重新构建
func (s *agentUpdateService) GetSnapshotInfo(ctx context.Context, ruleType RuleType) (*RuleSnapshotInfo, error) {
	snap, err := s.BuildSnapshot(ctx, ruleType)
	if err != nil {
		return nil, err
	}
	return &snap.RuleSnapshotInfo, nil
}

// BuildSnapshot 构建指定类型规则的快照
// 优化：实现基于目录 mtime 的惰性缓存
// 有缓存规则 -- 校验缓存规则有效性 --- 有效 --- 直接返回缓存规则快照
//
//	--- 无效 --- 删除旧缓存规则并重新构建规则快照返回(同时更新缓存规则)
//
// 无缓存规则 -- 构建规则快照返回(同时更新缓存规则)
func (s *agentUpdateService) BuildSnapshot(ctx context.Context, ruleType RuleType) (*RuleSnapshot, error) {
	// 1. 解析规则路径
	resolvedPath := s.resolveRulePath(ruleType)

	// 2. 获取目录的最新修改时间 (Latest ModTime)
	// 注意：仅检查根目录的 ModTime 是不够的，因为修改子文件不一定会更新根目录的 mtime (取决于 OS 和文件系统)
	// 严谨的做法是遍历所有文件取最大的 mtime。
	// 为了性能平衡，我们采用 "快速检查" 策略：
	// 如果目录下的文件变动频繁，建议用 fsnotify 或后台定时轮询。
	// 这里为了简单且高效，我们遍历目录树获取最新的 ModTime，因为 List 操作比 Read+Zip+Hash 快得多。
	latestModTime, err := getLatestModTime(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check rule directory time: %w", err)
	}

	// 3. 检查缓存
	s.mu.RLock()
	cachedItem, exists := s.cache[ruleType] // 如果缓存存在，则尝试获取快照
	s.mu.RUnlock()

	if exists && cachedItem.Snapshot != nil {
		// 如果缓存存在，且目录最后修改时间没有晚于缓存记录的时间，直接返回缓存
		// Equal 判断可能因为精度问题不准，使用 !After 更稳妥
		if !latestModTime.After(cachedItem.LastModTime) {
			// logger.LogInfo("Hit rule snapshot cache", "type", ruleType)
			// 缓存规则有效,就直接返回
			return cachedItem.Snapshot, nil
		}
	}

	// 4. 缓存未命中或已过期，重新构建 (加锁防止并发重复构建)
	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查 (Double Check)
	cachedItem, exists = s.cache[ruleType]
	if exists && cachedItem.Snapshot != nil {
		if !latestModTime.After(cachedItem.LastModTime) {
			return cachedItem.Snapshot, nil
		}
	}

	logger.LogBusinessOperation("rebuild_rule_snapshot", 0, "system", "localhost", "", "info", "rule files changed, rebuilding snapshot", map[string]interface{}{
		"rule_type":       ruleType,
		"latest_mod_time": latestModTime,
	})

	// --- 执行构建逻辑 (原逻辑) ---

	// 检查目录是否存在
	filePaths, err := listRuleFiles(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list rule files for type %s at %s: %w", ruleType, resolvedPath, err)
	}

	// 构建确定性 ZIP
	zipBytes, err := buildDeterministicZip(ctx, resolvedPath, filePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to build zip for type %s: %w", ruleType, err)
	}

	// 计算 Hash
	h := md5.Sum(zipBytes)
	version := hex.EncodeToString(h[:])

	// 构建 RuleSnapshot
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

	// 5. 更新缓存
	s.cache[ruleType] = &SnapshotCacheItem{
		Snapshot:    snapshot,
		LastModTime: latestModTime,
	}

	return snapshot, nil
}

// resolveRulePath 解析规则路径
func (s *agentUpdateService) resolveRulePath(ruleType RuleType) string {
	rulePath := ""
	if s.cfg != nil {
		if ruleType == RuleTypeFingerprint {
			rulePath = s.cfg.GetFingerprintRulePath()
		}
	}
	resolvedPath := strings.TrimSpace(rulePath)
	if resolvedPath == "" {
		defaultPath, ok := rulePathMap[ruleType]
		if !ok {
			defaultPath = filepath.Join("rules", string(ruleType))
		}
		resolvedPath = defaultPath
	}
	return resolvedPath
}

// getLatestModTime 递归获取目录中所有文件最新的修改时间
func getLatestModTime(root string) (time.Time, error) {
	var latest time.Time

	// 如果根目录不存在，直接返回错误，由上层处理
	info, err := os.Stat(root)
	if err != nil {
		return latest, err
	}
	latest = info.ModTime()

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return nil // 忽略无法获取信息的文件
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})

	return latest, err
}

// listRuleFiles 递归列出目录下所有文件的相对路径（不包含目录）
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

// buildDeterministicZip 构建确定性 ZIP 文件
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

// GetEncryptedSnapshot 获取加密/签名的快照
// 流程: 原始数据 -> AES-GCM 加密 -> HMAC-SHA256 签名
// Agent 流程: 验证签名 -> AES-GCM 解密 -> 解压 ZIP
func (s *agentUpdateService) GetEncryptedSnapshot(ctx context.Context, ruleType RuleType) (*RuleSnapshot, error) {
	// 1. 复用 BuildSnapshot 获取原始快照
	// 注意：BuildSnapshot 返回的是缓存对象的指针，不能直接修改其 Bytes 字段，否则会污染缓存！
	// 必须拷贝一份 RuleSnapshot 对象
	cachedSnapshot, err := s.BuildSnapshot(ctx, ruleType)
	if err != nil {
		return nil, err
	}

	// 浅拷贝结构体
	snapshot := *cachedSnapshot
	// 此时 snapshot.Bytes 仍然指向缓存的切片，后续加密会生成新的切片，所以这里是安全的

	// 2. 获取密钥
	secret := ""
	if s.cfg != nil {
		secret = s.cfg.Security.Agent.RuleEncryptionKey
	}

	// 3. 加密与签名
	if secret != "" {
		// A. 加密 (AES-GCM)
		encryptedBytes, err := encryptData(secret, snapshot.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt snapshot: %w", err)
		}
		snapshot.Bytes = encryptedBytes
		snapshot.ContentType = "application/octet-stream" // 更改内容类型为二进制流
		// 文件名增加后缀以示区分，或者保持不变由 Agent 处理，这里建议保持不变或加 .enc

		// B. 签名 (HMAC-SHA256) - 对加密后的数据签名
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(snapshot.Bytes)
		signature := hex.EncodeToString(h.Sum(nil))
		snapshot.Signature = signature
	}

	// 返回加密/签名的快照
	return &snapshot, nil
}
