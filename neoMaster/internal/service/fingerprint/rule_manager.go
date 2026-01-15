// RuleManager 负责指纹规则的导入导出和生命周期管理
// 它不参与运行时的匹配逻辑，只负责数据的 I/O
package fingerprint

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"neomaster/internal/config"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	assetrepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/fingerprint/converters"
)

// RuleManager 负责指纹规则的导入导出和生命周期管理
// 它不参与运行时的匹配逻辑，只负责数据的 I/O
type RuleManager struct {
	fingerRepo    assetrepo.AssetFingerRepository
	cpeRepo       assetrepo.AssetCPERepository
	converter     converters.StandardJSONConverter
	mu            sync.RWMutex   // 读写锁，保护并发操作
	backupDir     string         // 备份目录
	encryptionKey string         // 规则加密密钥
	config        *config.Config // 全局配置
}

// NewRuleManager 创建管理器
func NewRuleManager(fingerRepo assetrepo.AssetFingerRepository, cpeRepo assetrepo.AssetCPERepository, encryptionKey string, cfg *config.Config) *RuleManager {
	// 获取备份路径，优先使用配置，否则使用默认值
	// 默认结构: rules/backups/fingerprint
	backupDir := "rules/backups/fingerprint"

	if cfg != nil {
		rulesRoot := "rules"
		if cfg.App.Rules.RootPath != "" {
			rulesRoot = cfg.App.Rules.RootPath
		}

		backupRoot := "backups"
		if cfg.App.Rules.Backup.Dir != "" {
			backupRoot = cfg.App.Rules.Backup.Dir
		}
		// 规则备份目录: ROOT_DIR/BACKUP_DIR/ + fingerprint 【以后可以扩充 poc 配置】
		backupDir = filepath.Join(rulesRoot, backupRoot, "fingerprint")
	}

	// 确保目录存在
	if err := utils.MkdirAll(backupDir, 0755); err != nil {
		logger.LogBusinessError(err, "system", 0, "localhost", "", "mkdir", map[string]interface{}{
			"path": backupDir,
		})
	}

	return &RuleManager{
		fingerRepo:    fingerRepo,
		cpeRepo:       cpeRepo,
		converter:     *converters.NewStandardJSONConverter(),
		backupDir:     backupDir,
		encryptionKey: encryptionKey,
		config:        cfg,
	}
}

// ExportRules 导出所有规则 (Admin Backup)
func (m *RuleManager) ExportRules(ctx context.Context) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.exportRulesInternal(ctx, false) // 导出所有，包括禁用的
}

// ExportRulesWithSignature 导出规则并附带签名 (HMAC 或 SHA256)
// 返回: data, signature, error
// 为了简单，目前直接计算 SHA256，不使用密钥。
// 如果需要防篡改（防止伪造），应使用 HMAC 和密钥。
// 但这里主要是为了完整性校验（防止传输错误），SHA256 足够。
// 为了方便前端或Agent验证，我们可以返回一个 JSON 包装结构？
// 或者只返回 Header？
// 这里为了保持 ExportRules 接口的纯粹性，我们返回 data，让 Handler 决定如何包装。
// Handler 可以将 SHA256 放入 Header。
func (m *RuleManager) CalculateSignature(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// PublishRulesToDisk 发布规则到磁盘 (Agent Download) 将当前数据库中的规则发布到磁盘文件 (原子操作)
// 只发布 Enabled=true 的规则
func (m *RuleManager) PublishRulesToDisk(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. Export Data (Only Enabled)
	data, err := m.exportRulesInternal(ctx, true)
	if err != nil {
		return err
	}

	// 2. 写入临时文件
	// 默认路径：rules/fingerprint/rules.json (这里简化处理，直接覆盖单个大文件)
	// TODO: 后续应支持拆分文件，这里保持与 StandardJSONConverter 一致，输出单个 JSON
	// 为了兼容 Agent 的目录扫描逻辑，我们把这个文件放在 config 中定义的目录，或者默认目录
	// 这里假设目录结构是: rules/fingerprint/
	// AgentUpdateService 会扫描该目录下所有文件并打包
	targetDir := "rules/fingerprint"
	if err := utils.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create rule dir: %w", err)
	}

	targetFile := filepath.Join(targetDir, "neoscan_fingerprint_rules.json")
	tmpFile := targetFile + ".tmp"

	if err := utils.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write tmp file: %w", err)
	}

	// 3. 原子重命名 (覆盖)
	if err := utils.Rename(tmpFile, targetFile); err != nil {
		return fmt.Errorf("failed to rename rule file: %w", err)
	}

	// 4. 更新 mtime (确保 AgentUpdateService 能感知到变化)
	// Rename 会保留原文件的 mtime (取决于 FS 实现)，为了保险，显式 Touch 一下
	now := time.Now()
	if err := utils.Chtimes(targetFile, now, now); err != nil {
		logger.LogBusinessError(err, "system", 0, "localhost", "", "touch", map[string]interface{}{
			"file": targetFile,
		})
		// Touch 失败不影响功能，只是可能延迟生效
	}

	return nil
}

// exportRulesInternal 内部导出逻辑，不加锁，供内部调用
func (m *RuleManager) exportRulesInternal(ctx context.Context, onlyEnabled bool) ([]byte, error) {
	var err error
	var fingers []*asset.AssetFinger
	var cpes []*asset.AssetCPE

	// 1. Fetch Fingers
	if onlyEnabled {
		fingers, err = m.fingerRepo.FindEnabled(ctx)
	} else {
		fingers, err = m.fingerRepo.ListAll(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch fingers failed: %w", err)
	}

	// 2. Fetch CPEs
	if onlyEnabled {
		cpes, err = m.cpeRepo.FindEnabled(ctx)
	} else {
		cpes, err = m.cpeRepo.ListAll(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch cpes failed: %w", err)
	}

	// 3. Convert
	return m.converter.Encode(fingers, cpes)
}

// GetRuleStats 获取规则库统计信息
func (m *RuleManager) GetRuleStats(ctx context.Context) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取指纹总数
	fingers, err := m.fingerRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("count fingers failed: %w", err)
	}
	fingerCount := len(fingers)

	// 获取 CPE 总数
	cpes, err := m.cpeRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("count cpes failed: %w", err)
	}
	cpeCount := len(cpes)

	// 获取最后更新时间
	var lastUpdate time.Time
	if fingerCount > 0 {
		lastUpdate = time.Now()
	}

	return map[string]interface{}{
		"finger_count": fingerCount,
		"cpe_count":    cpeCount,
		"last_update":  lastUpdate,
		"version":      fmt.Sprintf("%d.%d", cpeCount, fingerCount),
	}, nil
}

// ImportRules 导入规则 (覆盖或增量)
// 包含自动备份和原子锁
// expectedSignature: 如果不为空，则进行完整性校验
func (m *RuleManager) ImportRules(ctx context.Context, data []byte, overwrite bool, expectedSignature string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 0. Integrity Check
	if expectedSignature != "" {
		currentSignature := m.CalculateSignature(data)
		if currentSignature != expectedSignature {
			return fmt.Errorf("integrity check failed: expected %s, got %s", expectedSignature, currentSignature)
		}
	}

	// 1. Auto Backup (Snapshot)
	if err := m.createBackup(ctx); err != nil {
		// 备份失败是否阻止导入？
		// 严格模式下应该阻止，保证数据安全
		return fmt.Errorf("auto backup failed: %w", err)
	}

	// 2. Decode & Validate
	fingers, cpes, err := m.converter.Decode(data)
	if err != nil {
		return fmt.Errorf("invalid rule format: %w", err)
	}

	// 3. Save to DB
	// TODO: 建议后续在 Repo 层实现 Transaction 接口
	for _, f := range fingers {
		if err := m.fingerRepo.Upsert(ctx, f); err != nil {
			logger.LogError(err, "", 0, "", "import_rules", "", map[string]interface{}{"name": f.Name})
		}
	}

	for _, c := range cpes {
		if err := m.cpeRepo.Upsert(ctx, c); err != nil {
			logger.LogError(err, "", 0, "", "import_rules", "", map[string]interface{}{"name": c.Name})
		}
	}

	return nil
}

// createBackup 创建当前规则库的快照
func (m *RuleManager) createBackup(ctx context.Context) error {
	data, err := m.exportRulesInternal(ctx, false) // 备份全量数据 true 只导出 Enabled false 导出所有
	if err != nil {
		return err
	}

	// 如果没有数据，是否跳过备份？建议还是备份一个空的，保持逻辑一致
	if len(data) == 0 {
		// return nil
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("rules_backup_%s.json", timestamp)
	filepath := filepath.Join(m.backupDir, filename)

	if err := utils.WriteFile(filepath, data, 0644); err != nil {
		return err
	}

	logger.LogBusinessOperation("auto_backup", 0, "", "system", "", "success", "auto backup created", map[string]interface{}{
		"file": filepath,
		"size": len(data),
	})

	return nil
}

// ListBackups 列出所有可用备份
func (m *RuleManager) ListBackups() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var backups []string
	files, err := utils.ReadDir(m.backupDir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "rules_backup_") && strings.HasSuffix(f.Name(), ".json") {
			backups = append(backups, f.Name())
		}
	}

	// 按时间倒序排列 (最新的在前)
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))

	return backups, nil
}

// Rollback 回滚到指定备份
func (m *RuleManager) Rollback(ctx context.Context, filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Validate Filename
	// 防止路径穿越攻击
	baseName := filepath.Base(filename)
	if baseName != filename {
		return fmt.Errorf("invalid filename")
	}
	targetPath := filepath.Join(m.backupDir, filename)

	// 2. Read Backup File
	data, err := utils.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("read backup file failed: %w", err)
	}

	// 3. Import (Use internal logic to avoid deadlock or recursive backup)
	// 回滚操作本身是否需要再次备份？
	// 策略：回滚前也备份当前状态（作为 "rollback_from_xxx"），防止回滚错误
	// 为了简化，这里暂不自动备份回滚前的状态，用户应确保选择正确的备份

	fingers, cpes, err := m.converter.Decode(data)
	if err != nil {
		return fmt.Errorf("decode backup file failed: %w", err)
	}

	// 4. Restore to DB
	// 回滚通常意味着"恢复到确切状态"，所以这里应该考虑清空现有数据再插入？
	// 或者使用 Upsert 覆盖？Upsert 无法删除已存在但备份中没有的数据。
	// 对于指纹库，通常是追加型，但如果之前的错误导入引入了脏数据，Upsert 无法清除。
	// TODO: 理想情况是 truncate table then insert，或者软删除所有 then insert。
	// 鉴于目前 Repo 接口限制，我们先使用 Upsert，这至少能恢复已知指纹的正确状态。
	// 如果需要完全一致，需要 Repo 提供 Reset/Clear 接口。

	for _, f := range fingers {
		if err := m.fingerRepo.Upsert(ctx, f); err != nil {
			logger.LogError(err, "", 0, "", "rollback_rules", "", map[string]interface{}{"name": f.Name})
		}
	}

	for _, c := range cpes {
		if err := m.cpeRepo.Upsert(ctx, c); err != nil {
			logger.LogError(err, "", 0, "", "rollback_rules", "", map[string]interface{}{"name": c.Name})
		}
	}

	return nil
}
