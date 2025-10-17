/**
 * 用户仓库层:会话数据访问
 * @author: sun977
 * @date: 2025.09.25
 * @description: 会话数据交互层(内存存储,适合单实例部署)
 * @func:单纯数据访问,不应该包含业务逻辑
 * @note: 尚未启用，和 neoMaster\internal\repo\redis\session.go 保持一致(可在配置文件中配置,二选一)
 */
// internal/repo/memory/session.go
package memory

import (
	"context"
	"fmt"
	"neomaster/internal/model/system"
	"sync"
	"time"
)

// SessionRepository 内存会话存储库
type SessionRepository struct {
	sessions         map[uint64]*sessionEntry
	revokedTokens    map[string]*tokenEntry
	refreshTokens    map[string]*refreshTokenEntry
	passwordVersions map[uint64]int64
	mutex            sync.RWMutex
}

// sessionEntry 会话条目
type sessionEntry struct {
	data       *system.SessionData
	expiration time.Time
}

// tokenEntry 令牌条目
type tokenEntry struct {
	data       *system.TokenData
	expiration time.Time
}

// refreshTokenEntry 刷新令牌条目
type refreshTokenEntry struct {
	userID     uint64
	tokenID    string
	expiration time.Time
}

// NewSessionRepository 创建内存会话存储库实例
func NewSessionRepository() *SessionRepository {
	repo := &SessionRepository{
		sessions:         make(map[uint64]*sessionEntry),
		revokedTokens:    make(map[string]*tokenEntry),
		refreshTokens:    make(map[string]*refreshTokenEntry),
		passwordVersions: make(map[uint64]int64),
	}

	// 启动过期清理goroutine
	go repo.cleanupExpired()

	return repo
}

// cleanupExpired 定期清理过期条目
func (r *SessionRepository) cleanupExpired() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		r.mutex.Lock()

		// 清理过期会话
		for userID, entry := range r.sessions {
			if now.After(entry.expiration) {
				delete(r.sessions, userID)
			}
		}

		// 清理过期撤销令牌
		for tokenID, entry := range r.revokedTokens {
			if now.After(entry.expiration) {
				delete(r.revokedTokens, tokenID)
			}
		}

		// 清理过期刷新令牌
		for tokenID, entry := range r.refreshTokens {
			if now.After(entry.expiration) {
				delete(r.refreshTokens, tokenID)
			}
		}

		r.mutex.Unlock()
	}
}

// StoreSession 存储用户会话信息
func (r *SessionRepository) StoreSession(ctx context.Context, userID uint64, sessionData *system.SessionData, expiration time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.sessions[userID] = &sessionEntry{
		data:       sessionData,
		expiration: time.Now().Add(expiration),
	}

	return nil
}

// GetSession 获取用户会话信息
func (r *SessionRepository) GetSession(ctx context.Context, userID uint64) (*system.SessionData, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entry, exists := r.sessions[userID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(entry.expiration) {
		delete(r.sessions, userID)
		return nil, fmt.Errorf("session not found")
	}

	return entry.data, nil
}

// DeleteSession 删除用户会话
func (r *SessionRepository) DeleteSession(ctx context.Context, userID uint64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.sessions, userID)
	return nil
}

// UpdateSessionExpiry 更新会话过期时间
func (r *SessionRepository) UpdateSessionExpiry(ctx context.Context, userID uint64, expiration time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	entry, exists := r.sessions[userID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	entry.expiration = time.Now().Add(expiration)
	return nil
}

// StoreToken 存储令牌信息
func (r *SessionRepository) StoreToken(ctx context.Context, tokenID string, tokenData *system.TokenData, expiration time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.revokedTokens[tokenID] = &tokenEntry{
		data:       tokenData,
		expiration: time.Now().Add(expiration),
	}

	return nil
}

// GetToken 获取令牌信息
func (r *SessionRepository) GetToken(ctx context.Context, tokenID string) (*system.TokenData, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entry, exists := r.revokedTokens[tokenID]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	if time.Now().After(entry.expiration) {
		delete(r.revokedTokens, tokenID)
		return nil, fmt.Errorf("token not found")
	}

	return entry.data, nil
}

// RevokeToken 撤销令牌
func (r *SessionRepository) RevokeToken(ctx context.Context, tokenID string, expiration time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.revokedTokens[tokenID] = &tokenEntry{
		data:       &system.TokenData{}, // 只需要记录撤销状态
		expiration: time.Now().Add(expiration),
	}

	return nil
}

// IsTokenRevoked 检查令牌是否已被撤销
func (r *SessionRepository) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entry, exists := r.revokedTokens[tokenID]
	if !exists {
		return false, nil
	}

	if time.Now().After(entry.expiration) {
		delete(r.revokedTokens, tokenID)
		return false, nil
	}

	return true, nil
}

// DeleteToken 删除令牌信息
func (r *SessionRepository) DeleteToken(ctx context.Context, tokenID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.revokedTokens, tokenID)
	return nil
}

// GetUserSessions 获取用户的所有会话
func (r *SessionRepository) GetUserSessions(ctx context.Context, userID uint64) ([]*system.SessionData, error) {
	// 对于内存存储，我们假设每个用户只有一个会话
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entry, exists := r.sessions[userID]
	if !exists {
		return []*system.SessionData{}, nil
	}

	if time.Now().After(entry.expiration) {
		delete(r.sessions, userID)
		return []*system.SessionData{}, nil
	}

	return []*system.SessionData{entry.data}, nil
}

// DeleteAllUserSessions 删除用户的所有会话
func (r *SessionRepository) DeleteAllUserSessions(ctx context.Context, userID uint64) error {
	// 对于内存存储，我们假设每个用户只有一个会话
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.sessions, userID)
	return nil
}

// StoreRefreshToken 存储刷新令牌
func (r *SessionRepository) StoreRefreshToken(ctx context.Context, userID uint64, tokenID string, expiration time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.refreshTokens[tokenID] = &refreshTokenEntry{
		userID:     userID,
		tokenID:    tokenID,
		expiration: time.Now().Add(expiration),
	}

	return nil
}

// ValidateRefreshToken 验证刷新令牌
func (r *SessionRepository) ValidateRefreshToken(ctx context.Context, userID uint64, tokenID string) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entry, exists := r.refreshTokens[tokenID]
	if !exists {
		return false, nil
	}

	if time.Now().After(entry.expiration) {
		delete(r.refreshTokens, tokenID)
		return false, nil
	}

	return entry.userID == userID, nil
}

// DeleteRefreshToken 删除刷新令牌
func (r *SessionRepository) DeleteRefreshToken(ctx context.Context, userID uint64, tokenID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.refreshTokens, tokenID)
	return nil
}

// StorePasswordVersion 存储用户密码版本号
func (r *SessionRepository) StorePasswordVersion(ctx context.Context, userID uint64, passwordV int64, expiration time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.passwordVersions[userID] = passwordV
	// 注意：内存存储中无法直接设置过期时间，需要依赖定期清理
	return nil
}

// GetPasswordVersion 获取用户密码版本号
func (r *SessionRepository) GetPasswordVersion(ctx context.Context, userID uint64) (int64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	version, exists := r.passwordVersions[userID]
	if !exists {
		return 0, fmt.Errorf("password version not found")
	}

	return version, nil
}

// DeletePasswordVersion 删除用户密码版本
func (r *SessionRepository) DeletePasswordVersion(ctx context.Context, userID uint64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.passwordVersions, userID)
	return nil
}

// Ping 检查存储连接（内存存储始终返回nil）
func (r *SessionRepository) Ping(ctx context.Context) error {
	return nil
}

// Close 关闭存储连接（内存存储不需要实际关闭）
func (r *SessionRepository) Close() error {
	return nil
}
