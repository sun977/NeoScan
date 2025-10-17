// Package auth 提供JWT认证相关的服务层实现
// 这个包封装了JWT令牌的生成、验证、刷新等核心业务逻辑
package auth

import (
	"context" // 用于传递请求上下文，支持超时控制和取消操作
	"errors"  // 用于创建和处理错误信息
	"fmt"     // 用于格式化字符串和错误信息
	"neomaster/internal/model/system"
	"time" // 用于处理时间相关操作，如令牌过期时间计算

	"neomaster/internal/pkg/auth"   // 导入JWT工具包，提供底层JWT操作
	"neomaster/internal/pkg/logger" // 导入日志管理器
	"neomaster/internal/repo/redis" // 导入Redis会话仓库，用于缓存用户密码版本

	"github.com/golang-jwt/jwt/v5" // 导入JWT库，用于令牌解析和验证
)

// TokenBlacklistService 令牌黑名单服务接口
// 这个接口定义了令牌黑名单操作的标准方法
// 通过接口解耦JWTService和SessionService之间的循环依赖
// type TokenBlacklistService interface {
// RevokeToken 撤销令牌（将令牌添加到黑名单）
// tokenJTI: JWT的唯一标识符
// expiration: 黑名单过期时间，通常设置为令牌的剩余有效时间
// RevokeToken(ctx context.Context, tokenJTI string, expiration time.Duration) error

// IsTokenRevoked 检查令牌是否已被撤销（是否在黑名单中）
// tokenJTI: JWT的唯一标识符
// IsTokenRevoked(ctx context.Context, tokenJTI string) (bool, error)
// }

// JWTService JWT认证服务结构体
// 这是服务层的核心结构，封装了JWT相关的所有业务逻辑
// 采用依赖注入的方式，将JWT管理器、用户服务和令牌黑名单服务作为依赖项
type JWTService struct {
	jwtManager  *auth.JWTManager         // JWT管理器，负责令牌的底层操作（生成、验证、解析）
	userService *UserService             // 用户服务，负责用户相关的业务逻辑
	redisRepo   *redis.SessionRepository // 会话仓库，负责与Redis交互，缓存用户密码版本
}

// NewJWTService 创建JWT服务实例
// 这是一个构造函数，使用依赖注入模式创建JWTService实例
// 参数:
//   - jwtManager: JWT管理器实例，提供令牌操作的底层功能
//   - userService: 用户服务实例，提供用户业务逻辑功能
//   - redisRepo: Redis会话仓库实例，提供与Redis交互的功能
//   - blacklistService: 令牌黑名单服务实例，提供令牌撤销和黑名单检查功能
//
// 返回: JWTService指针，包含所有JWT相关的业务方法
func NewJWTService(
	jwtManager *auth.JWTManager,
	userService *UserService,
	redisRepo *redis.SessionRepository) *JWTService {
	return &JWTService{
		jwtManager:  jwtManager,  // 注入JWT管理器依赖
		userService: userService, // 注入用户服务依赖
		redisRepo:   redisRepo,   // 注入Redis会话仓库依赖
	}
}

// GenerateTokens 生成访问令牌和刷新令牌
// 这是JWT服务的核心方法之一，为已认证用户生成令牌对
// 参数:
//   - ctx: 请求上下文，用于超时控制和请求追踪
//   - user: 用户模型实例，包含用户基本信息
//
// 返回: TokenPair指针（包含access_token和refresh_token）和错误信息
func (s *JWTService) GenerateTokens(ctx context.Context, user *system.User) (*auth.TokenPair, error) {
	// 参数验证：确保用户对象不为空
	// 这是防御性编程的体现，避免空指针异常
	if user == nil {
		logger.LogError(errors.New("user cannot be nil"), "", 0, "", "token_generate", "POST", map[string]interface{}{
			"operation": "generate_tokens",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("user cannot be nil")
	}

	// 获取用户完整信息，包括角色和权限
	// 这里需要查询数据库获取用户的角色权限信息，用于构建JWT声明
	userWithPerms, err := s.userService.GetUserWithRolesAndPermissions(ctx, user.ID)
	if err != nil {
		// 使用fmt.Errorf包装错误，保留原始错误信息，便于调试
		logger.LogError(err, "", uint(user.ID), "", "token_generate", "POST", map[string]interface{}{
			"operation": "generate_tokens",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// 构建权限列表（通过角色获取）
	// 权限格式为 "resource:action"，如 "user:create", "post:read"
	// 使用make([]string, 0)创建空切片，容量会自动扩展
	permissions := make([]string, 0)
	for _, role := range userWithPerms.Roles { // 遍历用户的所有角色
		for _, perm := range role.Permissions { // 遍历每个角色的所有权限
			// 将权限格式化为 "资源:操作" 的字符串格式
			// 这种格式便于后续权限验证时的字符串匹配
			permissions = append(permissions, fmt.Sprintf("%s:%s", perm.Resource, perm.Action))
		}
	}

	// 构建角色列表
	// 预分配切片容量，避免多次内存重新分配，提高性能
	roles := make([]string, 0, len(userWithPerms.Roles))
	for _, role := range userWithPerms.Roles { // 遍历用户角色
		roles = append(roles, role.Name) // 提取角色名称
	}

	// 生成令牌对
	// 调用JWT管理器生成访问令牌和刷新令牌
	// 传入用户的关键信息用于构建JWT声明
	tokenPair, err := s.jwtManager.GenerateTokenPair(
		userWithPerms.ID,        // 用户ID，作为JWT的Subject
		userWithPerms.Username,  // 用户名，用于标识用户
		userWithPerms.Email,     // 用户邮箱，额外的用户标识
		userWithPerms.PasswordV, // 密码版本号，用于密码变更后使旧令牌失效
		roles,                   // 用户角色列表，用于权限控制
	)
	if err != nil {
		// 令牌生成失败，包装错误信息返回
		logger.LogError(err, "", uint(userWithPerms.ID), "", "token_generate", "POST", map[string]interface{}{
			"operation": "generate_tokens",
			"username":  userWithPerms.Username,
			"roles":     roles,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	// 记录成功生成令牌的业务日志
	logger.LogBusinessOperation("generate_tokens", uint(userWithPerms.ID), userWithPerms.Username, "", "", "success", "令牌生成成功", map[string]interface{}{
		"roles":             roles,
		"permissions_count": len(permissions),
		"token_prefix":      tokenPair.AccessToken[:10] + "...", // 只记录token前缀
		"expires_in":        tokenPair.ExpiresIn,
		"timestamp":         logger.NowFormatted(),
	})

	// 成功生成令牌对，返回给调用者
	return tokenPair, nil
}

// ValidateAccessToken 验证访问令牌
// 这个方法用于验证客户端提供的访问令牌是否有效
// 包括签名验证、过期时间检查和黑名单检查
// 参数:
//   - tokenString: 待验证的JWT令牌字符串
//
// 返回: JWT声明信息和错误信息
func (s *JWTService) ValidateAccessToken(tokenString string) (*auth.JWTClaims, error) {
	// 第一步：调用JWT管理器验证令牌的签名、过期时间等
	// 这里会检查令牌格式、签名有效性、是否过期等
	claims, err := s.jwtManager.ValidateAccessToken(tokenString)
	if err != nil {
		// 验证失败，包装错误信息，提供更明确的错误描述
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	// // 检查密码版本（关键步骤）[启用的话会出现相互调用故障]
	// validVersion, err := s.ValidatePasswordVersion(context.Background(), tokenString)
	// if err != nil || !validVersion {
	// 	return nil, errors.New("token version mismatch")
	// }

	// 第二步：检查令牌是否在黑名单中（已被撤销）【不再检查redis黑名单】
	// 使用令牌的JTI（JWT ID）进行黑名单检查
	// if claims.ID != "" {
	// 	isRevoked, err := s.blacklistService.IsTokenRevoked(context.Background(), claims.ID)
	// 	if err != nil {
	// 		// 黑名单检查失败，记录错误日志但不阻止验证（降级处理）
	// 		// 这样可以避免Redis故障导致所有令牌验证失败
	// 		logger.LogError(err, "", uint(claims.UserID), "", "token_blacklist_check", "GET", map[string]interface{}{
	// 			"operation":    "validate_access_token",
	// 			"token_prefix": tokenString[:10] + "...",
	// 			"jti":          claims.ID,
	// 			"timestamp":    logger.NowFormatted(),
	// 		})
	// 		// 继续验证流程，不因黑名单检查失败而拒绝令牌
	// 	} else if isRevoked {
	// 		// 令牌已被撤销，拒绝访问
	// 		logger.LogError(errors.New("token has been revoked"), "", uint(claims.UserID), "", "token_revoked", "GET", map[string]interface{}{
	// 			"operation":    "validate_access_token",
	// 			"token_prefix": tokenString[:10] + "...",
	// 			"jti":          claims.ID,
	// 			"timestamp":    logger.NowFormatted(),
	// 		})
	// 		return nil, errors.New("token has been revoked")
	// 	}
	// }

	// 验证成功，返回解析出的JWT声明信息
	return claims, nil
}

// ValidateRefreshToken 验证刷新令牌
// 刷新令牌用于获取新的访问令牌，通常有更长的有效期
// 参数:
//   - tokenString: 待验证的刷新令牌字符串
//
// 返回: JWT标准声明信息和错误信息
func (s *JWTService) ValidateRefreshToken(tokenString string) (*jwt.RegisteredClaims, error) {
	// 验证刷新令牌的有效性
	// 刷新令牌通常只包含标准声明，不包含用户权限等敏感信息
	claims, err := s.jwtManager.ValidateRefreshToken(tokenString)
	if err != nil {
		// 刷新令牌验证失败，返回错误
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// // 检查密码版本（关键步骤）[启用的话会出现相互调用故障]
	// validVersion, err := s.ValidatePasswordVersion(context.Background(), tokenString)
	// if err != nil || !validVersion {
	// 	return nil, errors.New("token version mismatch")
	// }

	// 添加黑名单检查（与AccessToken相同）
	// if claims.ID != "" {
	// 	isRevoked, err := s.blacklistService.IsTokenRevoked(context.Background(), claims.ID)
	// 	if err != nil {
	// 		logger.LogError(err, "", 0, "", "refresh_token_blacklist_check", "GET", map[string]interface{}{
	// 			"operation":    "validate_refresh_token",
	// 			"token_prefix": tokenString[:10] + "...",
	// 			"jti":          claims.ID,
	// 			"timestamp":    logger.NowFormatted(),
	// 		})
	// 		// 根据安全策略决定是否继续
	// 	} else if isRevoked {
	// 		logger.LogError(errors.New("refresh token has been revoked"), "", 0, "", "refresh_token_revoked", "GET", map[string]interface{}{
	// 			"operation":    "validate_refresh_token",
	// 			"token_prefix": tokenString[:10] + "...",
	// 			"jti":          claims.ID,
	// 			"timestamp":    logger.NowFormatted(),
	// 		})
	// 		return nil, errors.New("refresh token has been revoked")
	// 	}
	// }

	// 验证成功，返回标准JWT声明
	return claims, nil
}

// RefreshTokens 刷新令牌
// 使用有效的刷新令牌获取新的访问令牌和刷新令牌对
// 这是JWT无状态认证中延长会话的核心机制
// 参数:
//   - ctx: 请求上下文
//   - refreshToken: 有效的刷新令牌字符串
//
// 返回: 新的令牌对和错误信息
func (s *JWTService) RefreshTokens(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	// 第一步：验证刷新令牌的有效性
	// 检查令牌格式、签名、过期时间等
	_, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		// 刷新令牌无效，直接返回错误
		return nil, err
	}

	// 第二步：从刷新令牌中提取用户名
	// 刷新令牌的Subject字段包含用户名信息
	username, err := s.jwtManager.GetUsernameFromRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get username from token: %w", err)
	}

	// 第三步：根据用户名获取最新的用户信息
	// 这里需要查询数据库确保用户仍然存在且状态正常
	user, err := s.userService.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 检查用户是否存在
	if user == nil {
		return nil, errors.New("user not found")
	}

	// 检查用户账户状态
	// 如果用户被禁用，则不允许刷新令牌
	if !user.IsActive() {
		return nil, errors.New("user is inactive")
	}

	// 第四步：为用户生成新的令牌对
	// 这会重新获取用户的最新角色权限信息
	return s.GenerateTokens(ctx, user)
}

// GetUserFromToken 从令牌中获取用户信息
// 这个方法用于根据访问令牌获取完整的用户信息
// 常用于需要用户详细信息的业务场景
// 参数:
//   - ctx: 请求上下文
//   - tokenString: 有效的访问令牌字符串
//
// // 返回: 用户模型实例和错误信息
// func (s *JWTService) GetUserFromToken(ctx context.Context, tokenString string) (*model.User, error) {
// 	// 第一步：验证访问令牌并获取声明信息
// 	claims, err := s.ValidateAccessToken(tokenString)
// 	if err != nil {
// 		// 令牌验证失败，直接返回错误
// 		return nil, err
// 	}

// 	// 第二步：根据令牌中的用户ID查询数据库获取用户信息
// 	// 这确保获取的是最新的用户数据，而不是令牌中可能过时的信息
// 	user, err := s.userService.GetUserByID(ctx, claims.UserID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get user: %w", err)
// 	}

// 	// 检查用户是否存在
// 	// 用户可能在令牌有效期内被删除
// 	if user == nil {
// 		return nil, errors.New("user not found")
// 	}

// 	// 返回完整的用户信息
// 	return user, nil
// }

// CheckTokenExpiry 检查令牌是否即将过期
// 这个方法用于提前检测令牌是否接近过期，便于客户端主动刷新令牌
// 参数:
//   - tokenString: 待检查的访问令牌字符串
//   - threshold: 过期阈值时间，如果剩余时间小于此值则认为即将过期
//
// 返回: 是否即将过期的布尔值和错误信息
func (s *JWTService) CheckTokenExpiry(tokenString string, threshold time.Duration) (bool, error) {
	// 验证令牌并获取声明信息
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		// 令牌无效，返回错误
		return false, err
	}

	// 检查令牌是否包含过期时间
	// 标准JWT应该包含exp声明
	if claims.ExpiresAt == nil {
		return false, errors.New("token has no expiry time")
	}

	// 获取令牌过期时间
	expiryTime := claims.ExpiresAt.Time
	// 计算剩余时间，如果小于等于阈值则认为即将过期
	// time.Until返回到指定时间的持续时间
	return time.Until(expiryTime) <= threshold, nil
}

// // RevokeToken 撤销令牌（将令牌添加到黑名单）[未使用]
// 参数:
//   - ctx: 请求上下文
//   - tokenJTI: JWT的唯一标识符
//   - expiration: 黑名单过期时间，通常设置为令牌的剩余有效时间
//
// 返回: 错误信息
func (s *JWTService) RevokeToken(ctx context.Context, tokenJTI string, expiration time.Duration) error {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	type clientIPKeyType struct{}
	clientIP, _ := ctx.Value(clientIPKeyType{}).(string)

	if tokenJTI == "" {
		return errors.New("token JTI cannot be empty")
	}

	// 调用Redis仓库的RevokeToken方法
	// 这里遵循了层级调用关系：Service → Repository
	err := s.redisRepo.RevokeToken(ctx, tokenJTI, expiration)
	if err != nil {
		logger.LogError(err, "", 0, clientIP, "revoke_token", "POST", map[string]interface{}{
			"operation": "revoke_token",
			"jti":       tokenJTI,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// 记录令牌撤销的业务日志
	logger.LogBusinessOperation("revoke_token", 0, "", clientIP, "", "success", "令牌撤销成功", map[string]interface{}{
		"jti":       tokenJTI,
		"timestamp": logger.NowFormatted(),
	})

	return nil
}

// GetTokenClaims 获取令牌声明信息
// 这个方法用于解析令牌并获取其中包含的声明信息
// 与ValidateAccessToken功能相同，提供更语义化的方法名
// 参数:
//   - tokenString: JWT令牌字符串
//
// 返回: JWT声明信息和错误信息
func (s *JWTService) GetTokenClaims(tokenString string) (*auth.JWTClaims, error) {
	// 直接调用ValidateAccessToken方法
	// 这里复用验证逻辑，确保只有有效令牌才能获取声明信息
	return s.ValidateAccessToken(tokenString)
}

// IsTokenValid 检查令牌是否有效
// 这是一个便捷方法，只返回令牌是否有效的布尔值
// 适用于只需要知道令牌有效性而不需要具体错误信息的场景
// 参数:
//   - tokenString: JWT令牌字符串
//
// 返回: 令牌是否有效的布尔值
func (s *JWTService) IsTokenValid(tokenString string) bool {
	// 调用ValidateAccessToken进行验证
	// 忽略具体的错误信息，只关心是否验证成功
	_, err := s.ValidateAccessToken(tokenString)
	// 如果没有错误，则令牌有效
	return err == nil
}

// GetTokenRemainingTime 获取令牌剩余有效时间
// 这个方法用于计算令牌还有多长时间过期
// 可用于客户端显示会话剩余时间或决定是否需要刷新令牌
// 参数:
//   - tokenString: JWT令牌字符串
//
// 返回: 剩余有效时间和错误信息
func (s *JWTService) GetTokenRemainingTime(tokenString string) (time.Duration, error) {
	// 验证令牌并获取声明信息
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		// 令牌无效，返回0时间和错误
		return 0, err
	}

	// 检查令牌是否包含过期时间
	if claims.ExpiresAt == nil {
		return 0, errors.New("token has no expiry time")
	}

	// 获取过期时间并计算剩余时间
	expiryTime := claims.ExpiresAt.Time
	remaining := time.Until(expiryTime)

	// 如果剩余时间为负数，说明令牌已过期
	if remaining < 0 {
		return 0, errors.New("token has expired")
	}

	// 返回剩余有效时间
	return remaining, nil
}

// ValidateUserPermission 验证用户是否具有特定权限
// 这个方法用于基于角色的访问控制（RBAC），检查用户是否有执行特定操作的权限
// 参数:
//   - ctx: 请求上下文
//   - userID: 用户ID
//   - resource: 资源名称，如 "user", "post", "admin"
//   - action: 操作名称，如 "create", "read", "update", "delete"
//
// 返回: 是否具有权限的布尔值和错误信息
func (s *JWTService) ValidateUserPermission(ctx context.Context, userID uint, resource, action string) (bool, error) {
	// 从数据库获取用户的所有权限
	// 这里会通过用户角色关联查询获取权限列表
	permissions, err := s.userService.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// 遍历用户权限列表，查找匹配的权限
	for _, perm := range permissions {
		// 检查资源和操作是否完全匹配
		// 权限验证采用精确匹配策略
		if perm.Resource == resource && perm.Action == action {
			// 找到匹配的权限，返回true
			return true, nil
		}
	}

	// 没有找到匹配的权限，返回false
	return false, nil
}

// ValidateUserRole 验证用户是否具有特定角色
// 这个方法用于角色验证，检查用户是否被分配了特定的角色
// 角色验证通常用于粗粒度的权限控制
// 参数:
//   - ctx: 请求上下文
//   - userID: 用户ID
//   - roleName: 角色名称，如 "admin", "user", "moderator"
//
// 返回: 是否具有该角色的布尔值和错误信息
func (s *JWTService) ValidateUserRole(ctx context.Context, userID uint, roleName string) (bool, error) {
	// 从数据库获取用户的所有角色
	roles, err := s.userService.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	// 遍历用户角色列表，查找匹配的角色
	for _, role := range roles {
		// 检查角色名称是否匹配
		if role.Name == roleName {
			// 找到匹配的角色，返回true
			return true, nil
		}
	}

	// 没有找到匹配的角色，返回false
	return false, nil
}

// ValidateUserRoleFromToken 从令牌中验证用户是否具有特定角色
// 这是一个便捷方法，直接从JWT令牌中获取角色信息进行验证
// 参数:
//   - tokenString: JWT令牌字符串
//   - roleName: 角色名称
//
// 返回: 是否具有该角色的布尔值和错误信息
func (s *JWTService) ValidateUserRoleFromToken(tokenString, roleName string) (bool, error) {
	// 验证令牌并获取声明信息
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		return false, err
	}

	// 检查令牌中的角色列表
	for _, role := range claims.Roles {
		if role == roleName {
			return true, nil
		}
	}

	return false, nil
}

// ValidateUserPermissionFromToken 从JWT令牌验证用户权限的便捷方法
// 这个方法提供了基于令牌的权限验证，无需预先获取用户ID
// 参数:
//   - tokenString: JWT访问令牌
//   - resource: 资源名称
//   - action: 操作名称
//
// 返回: 是否具有权限的布尔值和错误信息
func (s *JWTService) ValidateUserPermissionFromToken(tokenString, resource, action string) (bool, error) {
	// 从令牌中获取用户ID
	userID, err := s.jwtManager.GetUserIDFromToken(tokenString)
	if err != nil {
		return false, fmt.Errorf("failed to get user ID from token: %w", err)
	}

	// 调用基于用户ID的权限验证方法
	return s.ValidateUserPermission(context.Background(), userID, resource, action)
}

// ValidatePasswordVersion 验证令牌中的密码版本是否与用户当前密码版本匹配
// 这是一个重要的安全机制，用于在用户修改密码后使所有旧令牌失效
// 当用户修改密码时，密码版本号会递增，使得基于旧版本号的令牌失效
// 参数:
//   - ctx: 请求上下文
//   - tokenString: 待验证的访问令牌字符串
//
// 返回: 密码版本是否匹配的布尔值和错误信息
func (s *JWTService) ValidatePasswordVersion(ctx context.Context, tokenString string) (bool, error) {
	// 首先验证令牌并获取声明信息
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		// 令牌验证失败，直接返回错误
		return false, err
	}

	// 从数据库获取用户当前的密码版本号
	// 优先从缓存获取以提高性能，缓存未命中时查询数据库
	// 从缓存获取用户密码版本
	currentPasswordV, err := s.redisRepo.GetPasswordVersion(ctx, uint64(claims.UserID))
	if err != nil {
		return false, fmt.Errorf("failed to get user password version from cache: %w", err)
	}
	// 没有返回 0
	if currentPasswordV == 0 {
		// 缓存未命中，从数据库获取密码版本
		currentPasswordV, err = s.userService.GetUserPasswordVersion(ctx, uint(claims.UserID))
		if err != nil {
			return false, fmt.Errorf("failed to get user password version from database: %w", err)
		}
	}

	// 比较令牌中的密码版本与数据库中的当前版本
	// 如果版本号不匹配，说明用户在令牌签发后修改了密码
	// 此时应该拒绝该令牌，要求用户重新登录
	return claims.PasswordV == currentPasswordV, nil
}
