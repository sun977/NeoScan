/**
 * 用户仓库层:会话数据访问
 * @author: sun977
 * @date: 2025.09.05
 * @description: 会话数据交互层(Redis存储,适合多实例部署)
 * @func:单纯数据访问,不应该包含业务逻辑
 * @note: 留下了兼容性代码，后续有时间扩展单用户多会话支持，目前仅支持单用户单会话
 */
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"neomaster/internal/model/system"
	"time"

	// "neomaster/internal/pkg/utils"

	"github.com/go-redis/redis/v8"
)

// SessionRepository Redis会话存储库
type SessionRepository struct {
	client *redis.Client
}

// NewSessionRepository 创建会话存储库实例
func NewSessionRepository(client *redis.Client) *SessionRepository {
	return &SessionRepository{
		client: client,
	}
}

// generateUniqueSessionID 生成唯一的会话ID
// func (r *SessionRepository) generateUniqueSessionID() (string, error) {
// 	sessionID, err := utils.GenerateUUID()
// 	if err != nil {
// 		return "", fmt.Errorf("utils.GenerateUUID failed to generate UUID: %w", err)
// 	}
// 	return sessionID, nil
// }

// StoreSession 存储用户会话信息
func (r *SessionRepository) StoreSession(ctx context.Context, userID uint64, sessionData *system.SessionData, expiration time.Duration) error {
	// 序列化会话数据
	data, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// // 支持单用户多会话
	// // 生成唯一的会话ID
	// sessionID, err := r.generateUniqueSessionID()
	// if err != nil {
	// 	return fmt.Errorf("failed to generate unique session id: %w", err)
	// }
	// // 生成会话键[KEY:session:user:{userID}:{sessionID}]
	// sessionKey := r.createSessionKey(userID, sessionID)

	// // 生成会话键
	sessionKey := r.getSessionKey(userID)

	// 存储到Redis
	err = r.client.Set(ctx, sessionKey, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	return nil
}

// GetSession 获取用户会话信息
func (r *SessionRepository) GetSession(ctx context.Context, userID uint64) (*system.SessionData, error) {
	// 生成会话键
	sessionKey := r.getSessionKey(userID)

	// 从Redis获取数据
	data, err := r.client.Get(ctx, sessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// 反序列化会话数据
	var sessionData system.SessionData
	err = json.Unmarshal([]byte(data), &sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return &sessionData, nil
}

// DeleteSession 删除用户会话
func (r *SessionRepository) DeleteSession(ctx context.Context, userID uint64) error {
	// 生成会话键
	sessionKey := r.getSessionKey(userID)

	// 从Redis删除
	err := r.client.Del(ctx, sessionKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// UpdateSessionExpiry 更新会话过期时间
func (r *SessionRepository) UpdateSessionExpiry(ctx context.Context, userID uint64, expiration time.Duration) error {
	// 生成会话键
	sessionKey := r.getSessionKey(userID)

	// 更新过期时间
	err := r.client.Expire(ctx, sessionKey, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to update session expiry: %w", err)
	}

	return nil
}

// StoreToken 存储令牌信息（用于令牌黑名单或白名单）
func (r *SessionRepository) StoreToken(ctx context.Context, tokenID string, tokenData *system.TokenData, expiration time.Duration) error {
	// 序列化令牌数据
	data, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	// 生成令牌键
	tokenKey := r.getTokenKey(tokenID)

	// 存储到Redis
	err = r.client.Set(ctx, tokenKey, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// GetToken 获取令牌信息
func (r *SessionRepository) GetToken(ctx context.Context, tokenID string) (*system.TokenData, error) {
	// 生成令牌键
	tokenKey := r.getTokenKey(tokenID)

	// 从Redis获取数据
	data, err := r.client.Get(ctx, tokenKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// 反序列化令牌数据
	var tokenData system.TokenData
	err = json.Unmarshal([]byte(data), &tokenData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	return &tokenData, nil
}

// RevokeToken 撤销令牌（添加到黑名单）[实际上是存入了redis缓存中("revoked:token:20250919173619-856583300")]
func (r *SessionRepository) RevokeToken(ctx context.Context, tokenID string, expiration time.Duration) error {
	// 生成撤销令牌键
	revokedKey := r.getRevokedTokenKey(tokenID)

	// 添加到黑名单，值为撤销时间戳
	revokedAt := time.Now().Unix()
	// 使用redis的set命令将撤销令牌键添加到缓存中，值为撤销时间戳
	err := r.client.Set(ctx, revokedKey, revokedAt, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// IsTokenRevoked 检查令牌是否已被撤销
func (r *SessionRepository) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	// 生成撤销令牌键
	revokedKey := r.getRevokedTokenKey(tokenID)

	// 检查是否存在
	exists, err := r.client.Exists(ctx, revokedKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token revocation: %w", err)
	}

	return exists > 0, nil
}

// DeleteToken 删除令牌信息
func (r *SessionRepository) DeleteToken(ctx context.Context, tokenID string) error {
	// 生成令牌键
	tokenKey := r.getTokenKey(tokenID)

	// 从Redis删除
	err := r.client.Del(ctx, tokenKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}

// GetUserSessions 获取用户的所有会话（用于多设备登录管理）
func (r *SessionRepository) GetUserSessions(ctx context.Context, userID uint64) ([]*system.SessionData, error) {
	// 生成用户会话模式键 (使用通配符匹配所有会话) ["session:user:1:*"]
	pattern := r.getUserSessionPattern(userID)
	// pattern := r.getSessionKey(userID)

	// 获取匹配的键
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user session keys: %w", err)
	}

	if len(keys) == 0 {
		return []*system.SessionData{}, nil
	}

	// 批量获取会话数据
	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	var sessions []*system.SessionData
	for _, value := range values {
		if value == nil {
			continue
		}

		var sessionData system.SessionData
		err = json.Unmarshal([]byte(value.(string)), &sessionData)
		if err != nil {
			continue // 跳过无效数据
		}

		sessions = append(sessions, &sessionData)
	}

	return sessions, nil
}

// DeleteAllUserSessions 删除用户的所有会话
func (r *SessionRepository) DeleteAllUserSessions(ctx context.Context, userID uint64) error {
	// 生成用户会话模式键
	pattern := r.getUserSessionPattern(userID)

	// 获取匹配的键
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get user session keys: %w", err)
	}

	if len(keys) == 0 {
		return nil // 没有会话需要删除
	}

	// 批量删除
	err = r.client.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// StoreRefreshToken 存储刷新令牌
func (r *SessionRepository) StoreRefreshToken(ctx context.Context, userID uint64, tokenID string, expiration time.Duration) error {
	// 生成刷新令牌键
	refreshKey := r.getRefreshTokenKey(userID, tokenID)

	// 直接存储令牌ID作为值（简化存储结构）
	err := r.client.Set(ctx, refreshKey, tokenID, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// ValidateRefreshToken 验证刷新令牌
func (r *SessionRepository) ValidateRefreshToken(ctx context.Context, userID uint64, tokenID string) (bool, error) {
	// 生成刷新令牌键
	refreshKey := r.getRefreshTokenKey(userID, tokenID)

	// 检查是否存在
	exists, err := r.client.Exists(ctx, refreshKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	return exists > 0, nil
}

// DeleteRefreshToken 删除刷新令牌
func (r *SessionRepository) DeleteRefreshToken(ctx context.Context, userID uint64, tokenID string) error {
	// 生成刷新令牌键
	refreshKey := r.getRefreshTokenKey(userID, tokenID)

	// 删除刷新令牌
	err := r.client.Del(ctx, refreshKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	return nil
}

// 私有方法：生成各种键名

// getSessionKey 生成会话键[用于精确查询]
// 会话键用于存储用户的会话数据，键的格式为 [session:user:{userID}:] (保留一个:分割空位用于后续扩展sessionID)
func (r *SessionRepository) getSessionKey(userID uint64) string {
	return fmt.Sprintf("session:user:%d:", userID)
}

// // createSessionKey 创建会话键(包含用户ID和会话ID)[KEY:session:user:{userID}:{sessionID}]
// func (r *SessionRepository) createSessionKey(userID uint64, sessionID string) string {
// 	return fmt.Sprintf("session:user:%d:%s", userID, sessionID)
// }

// getUserSessionPattern 生成用户会话模式键[用于批量操作]
func (r *SessionRepository) getUserSessionPattern(userID uint64) string {
	return fmt.Sprintf("session:user:%d:*", userID)
}

// getTokenKey 生成令牌键
func (r *SessionRepository) getTokenKey(tokenID string) string {
	return fmt.Sprintf("token:%s", tokenID)
}

// getRevokedTokenKey 生成撤销令牌键
func (r *SessionRepository) getRevokedTokenKey(tokenID string) string {
	return fmt.Sprintf("revoked:token:%s", tokenID)
}

// getRefreshTokenKey 生成刷新令牌键[KEY:refresh:user:{userID}:token:{tokenID}]
func (r *SessionRepository) getRefreshTokenKey(userID uint64, tokenID string) string {
	return fmt.Sprintf("refresh:user:%d:token:%s", userID, tokenID)
}

// Ping 检查Redis连接
func (r *SessionRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close 关闭Redis连接
func (r *SessionRepository) Close() error {
	return r.client.Close()
}

// StorePasswordVersion 存储用户密码版本号到缓存
func (r *SessionRepository) StorePasswordVersion(ctx context.Context, userID uint64, passwordV int64, expiration time.Duration) error {
	key := r.getPasswordVersionKey(userID)
	err := r.client.Set(ctx, key, passwordV, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to store password version: %w", err)
	}
	return nil
}

// GetPasswordVersion 从缓存获取用户密码版本号
func (r *SessionRepository) GetPasswordVersion(ctx context.Context, userID uint64) (int64, error) {
	key := r.getPasswordVersionKey(userID)
	result, err := r.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("password version not found in cache")
		}
		return 0, fmt.Errorf("failed to get password version: %w", err)
	}
	return result, nil
}

// DeletePasswordVersion 删除用户密码版本缓存
func (r *SessionRepository) DeletePasswordVersion(ctx context.Context, userID uint64) error {
	key := r.getPasswordVersionKey(userID)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete password version: %w", err)
	}
	return nil
}

// getPasswordVersionKey 生成密码版本缓存键[KEY:password_version:<userID>]
func (r *SessionRepository) getPasswordVersionKey(userID uint64) string {
	return fmt.Sprintf("password_version:%d", userID)
}
