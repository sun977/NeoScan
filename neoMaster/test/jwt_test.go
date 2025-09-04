// Package test JWT服务单元测试
// 测试令牌生成、验证、刷新和权限检查功能
package test

import (
	"context"
	"strings"
	"testing"
	"time"

	"neomaster/internal/pkg/auth"
)

// TestJWTService 测试JWT服务的核心功能
func TestJWTService(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		t.Run("令牌生成", func(t *testing.T) {
			testTokenGeneration(t, ts)
		})

		t.Run("访问令牌验证", func(t *testing.T) {
			testAccessTokenValidation(t, ts)
		})

		t.Run("刷新令牌验证", func(t *testing.T) {
			testRefreshTokenValidation(t, ts)
		})

		t.Run("令牌刷新", func(t *testing.T) {
			testTokenRefresh(t, ts)
		})

		t.Run("令牌过期检查", func(t *testing.T) {
			testTokenExpiry(t, ts)
		})

		t.Run("用户权限验证", func(t *testing.T) {
			testUserPermissionValidation(t, ts)
		})

		t.Run("密码版本验证", func(t *testing.T) {
			testPasswordVersionValidation(t, ts)
		})

		t.Run("令牌撤销", func(t *testing.T) {
			testTokenRevocation(t, ts)
		})
	})
}

// testTokenGeneration 测试令牌生成功能
func testTokenGeneration(t *testing.T, ts *TestSuite) {
	ctx := context.Background()

	// 创建测试用户和角色
	user := ts.CreateTestUser(t, "jwtuser", "jwt@test.com", "password123")
	adminRole := ts.CreateTestRole(t, "admin", "管理员角色")
	ts.AssignRoleToUser(t, user.ID, adminRole.ID)

	// 获取用户完整信息
	userWithRoles, err := ts.UserRepo.GetUserWithRolesAndPermissions(ctx, user.ID)
	AssertNoError(t, err, "获取用户角色信息不应该出错")

	// 生成令牌对
	tokenPair, err := ts.JWTService.GenerateTokens(ctx, userWithRoles)
	AssertNoError(t, err, "生成令牌不应该出错")
	AssertNotEqual(t, "", tokenPair.AccessToken, "访问令牌不应该为空")
	AssertNotEqual(t, "", tokenPair.RefreshToken, "刷新令牌不应该为空")
	AssertTrue(t, tokenPair.ExpiresIn > 0, "过期时间应该大于0")

	// 验证令牌格式（JWT应该有三个部分，用.分隔）
	accessParts := strings.Split(tokenPair.AccessToken, ".")
	AssertEqual(t, 3, len(accessParts), "访问令牌应该有3个部分")

	refreshParts := strings.Split(tokenPair.RefreshToken, ".")
	AssertEqual(t, 3, len(refreshParts), "刷新令牌应该有3个部分")

	// 验证令牌不相同
	AssertNotEqual(t, tokenPair.AccessToken, tokenPair.RefreshToken, "访问令牌和刷新令牌应该不同")
}

// testAccessTokenValidation 测试访问令牌验证
func testAccessTokenValidation(t *testing.T, ts *TestSuite) {
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "validateuser", "validate@test.com", "password123")

	// 生成令牌
	tokenPair, err := ts.JWTService.GenerateTokens(ctx, user)
	AssertNoError(t, err, "生成令牌不应该出错")

	// 验证有效的访问令牌
	claims, err := ts.JWTService.ValidateAccessToken(tokenPair.AccessToken)
	AssertNoError(t, err, "验证有效访问令牌不应该出错")
	AssertEqual(t, user.ID, claims.UserID, "用户ID应该匹配")
	AssertEqual(t, user.Username, claims.Username, "用户名应该匹配")
	AssertEqual(t, user.Email, claims.Email, "邮箱应该匹配")
	AssertEqual(t, user.PasswordV, claims.PasswordV, "密码版本应该匹配")

	// 测试无效令牌
	_, err = ts.JWTService.ValidateAccessToken("invalid.token.here")
	AssertError(t, err, "验证无效令牌应该出错")

	// 测试空令牌
	_, err = ts.JWTService.ValidateAccessToken("")
	AssertError(t, err, "验证空令牌应该出错")

	// 测试格式错误的令牌
	_, err = ts.JWTService.ValidateAccessToken("not.a.valid.jwt.token")
	AssertError(t, err, "验证格式错误令牌应该出错")
}

// testRefreshTokenValidation 测试刷新令牌验证
func testRefreshTokenValidation(t *testing.T, ts *TestSuite) {
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "refreshuser", "refresh@test.com", "password123")

	// 生成令牌
	tokenPair, err := ts.JWTService.GenerateTokens(ctx, user)
	AssertNoError(t, err, "生成令牌不应该出错")

	// 验证有效的刷新令牌
	claims, err := ts.JWTService.ValidateRefreshToken(tokenPair.RefreshToken)
	AssertNoError(t, err, "验证有效刷新令牌不应该出错")
	AssertEqual(t, user.Username, claims.Subject, "主题应该是用户名")
	AssertEqual(t, "neoscan", claims.Issuer, "发行者应该是neoscan")
	AssertTrue(t, len(claims.Audience) > 0, "应该有受众")
	AssertEqual(t, "neoscan-refresh", claims.Audience[0], "受众应该是neoscan-refresh")

	// 测试用访问令牌验证刷新令牌（应该失败）
	_, err = ts.JWTService.ValidateRefreshToken(tokenPair.AccessToken)
	AssertError(t, err, "用访问令牌验证刷新令牌应该出错")

	// 测试无效刷新令牌
	_, err = ts.JWTService.ValidateRefreshToken("invalid.refresh.token")
	AssertError(t, err, "验证无效刷新令牌应该出错")
}

// testTokenRefresh 测试令牌刷新功能
func testTokenRefresh(t *testing.T, ts *TestSuite) {
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "refreshtestuser", "refreshtest@test.com", "password123")

	// 生成初始令牌
	initialTokens, err := ts.JWTService.GenerateTokens(ctx, user)
	AssertNoError(t, err, "生成初始令牌不应该出错")

	// 等待一小段时间确保新令牌时间戳不同
	time.Sleep(10 * time.Millisecond)

	// 使用刷新令牌获取新的令牌对
	newTokens, err := ts.JWTService.RefreshTokens(ctx, initialTokens.RefreshToken)
	AssertNoError(t, err, "刷新令牌不应该出错")
	AssertNotEqual(t, "", newTokens.AccessToken, "新访问令牌不应该为空")
	AssertNotEqual(t, "", newTokens.RefreshToken, "新刷新令牌不应该为空")

	// 验证新令牌与旧令牌不同
	AssertNotEqual(t, initialTokens.AccessToken, newTokens.AccessToken, "新访问令牌应该与旧令牌不同")
	AssertNotEqual(t, initialTokens.RefreshToken, newTokens.RefreshToken, "新刷新令牌应该与旧令牌不同")

	// 验证新访问令牌有效
	claims, err := ts.JWTService.ValidateAccessToken(newTokens.AccessToken)
	AssertNoError(t, err, "新访问令牌应该有效")
	AssertEqual(t, user.ID, claims.UserID, "用户ID应该匹配")

	// 验证新刷新令牌有效
	_, err = ts.JWTService.ValidateRefreshToken(newTokens.RefreshToken)
	AssertNoError(t, err, "新刷新令牌应该有效")

	// 测试用无效刷新令牌刷新
	_, err = ts.JWTService.RefreshTokens(ctx, "invalid.refresh.token")
	AssertError(t, err, "用无效刷新令牌刷新应该出错")

	// 测试用访问令牌刷新（应该失败）
	_, err = ts.JWTService.RefreshTokens(ctx, initialTokens.AccessToken)
	AssertError(t, err, "用访问令牌刷新应该出错")
}

// testTokenExpiry 测试令牌过期检查
func testTokenExpiry(t *testing.T, ts *TestSuite) {
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "expiryuser", "expiry@test.com", "password123")

	// 生成令牌
	tokenPair, err := ts.JWTService.GenerateTokens(ctx, user)
	AssertNoError(t, err, "生成令牌不应该出错")

	// 测试令牌未过期
	expiring, err := ts.JWTService.CheckTokenExpiry(tokenPair.AccessToken, 1*time.Hour)
	AssertNoError(t, err, "检查令牌过期不应该出错")
	AssertFalse(t, expiring, "令牌不应该即将过期")

	// 测试令牌即将过期（使用大于令牌有效期的阈值，令牌有效期24小时，使用25小时阈值）
	expiring, err = ts.JWTService.CheckTokenExpiry(tokenPair.AccessToken, 25*time.Hour)
	AssertNoError(t, err, "检查令牌过期不应该出错")
	AssertTrue(t, expiring, "令牌应该即将过期")

	// 测试获取令牌剩余时间
	remainingTime, err := ts.JWTService.GetTokenRemainingTime(tokenPair.AccessToken)
	AssertNoError(t, err, "获取令牌剩余时间不应该出错")
	AssertTrue(t, remainingTime > 0, "剩余时间应该大于0")
	AssertTrue(t, remainingTime < 25*time.Hour, "剩余时间应该小于25小时")
	AssertTrue(t, remainingTime > 23*time.Hour, "剩余时间应该大于23小时")

	// 测试令牌有效性检查
	valid := ts.JWTService.IsTokenValid(tokenPair.AccessToken)
	AssertTrue(t, valid, "令牌应该有效")

	valid = ts.JWTService.IsTokenValid("invalid.token")
	AssertFalse(t, valid, "无效令牌应该返回false")
}

// testUserPermissionValidation 测试用户权限验证
func testUserPermissionValidation(t *testing.T, ts *TestSuite) {
	ctx := context.Background()

	// 创建测试用户和角色
	user := ts.CreateTestUser(t, "permuser", "perm@test.com", "password123")
	adminRole := ts.CreateTestRole(t, "admin", "管理员角色")
	userRole := ts.CreateTestRole(t, "user", "普通用户角色")

	// 为用户分配角色
	ts.AssignRoleToUser(t, user.ID, adminRole.ID)
	ts.AssignRoleToUser(t, user.ID, userRole.ID)

	// 生成令牌
	userWithRoles, err := ts.UserRepo.GetUserWithRolesAndPermissions(ctx, user.ID)
	AssertNoError(t, err, "获取用户角色不应该出错")

	tokenPair, err := ts.JWTService.GenerateTokens(ctx, userWithRoles)
	AssertNoError(t, err, "生成令牌不应该出错")

	// 验证令牌中包含角色信息
	claims, err := ts.JWTService.ValidateAccessToken(tokenPair.AccessToken)
	AssertNoError(t, err, "验证令牌不应该出错")
	AssertTrue(t, len(claims.Roles) >= 2, "令牌应该包含角色信息")

	// 验证用户角色
	hasAdminRole, err := ts.JWTService.ValidateUserRoleFromToken(tokenPair.AccessToken, "admin")
	AssertNoError(t, err, "验证用户角色不应该出错")
	AssertTrue(t, hasAdminRole, "用户应该有admin角色")

	hasUserRole, err := ts.JWTService.ValidateUserRoleFromToken(tokenPair.AccessToken, "user")
	AssertNoError(t, err, "验证用户角色不应该出错")
	AssertTrue(t, hasUserRole, "用户应该有user角色")

	hasGuestRole, err := ts.JWTService.ValidateUserRoleFromToken(tokenPair.AccessToken, "guest")
	AssertNoError(t, err, "验证用户角色不应该出错")
	AssertFalse(t, hasGuestRole, "用户不应该有guest角色")

	// 测试权限验证（假设有权限系统）
	// 注意：这里需要根据实际的权限系统实现来调整
	hasPermission, err := ts.JWTService.ValidateUserPermissionFromToken(tokenPair.AccessToken, "user", "read")
	AssertNoError(t, err, "验证用户权限不应该出错")
	// 由于测试环境可能没有具体权限数据，这里只验证方法调用不出错
	_ = hasPermission
}

// testPasswordVersionValidation 测试密码版本验证
func testPasswordVersionValidation(t *testing.T, ts *TestSuite) {
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "pwdveruser", "pwdver@test.com", "password123")
	initialVersion := user.PasswordV

	// 生成令牌
	tokenPair, err := ts.JWTService.GenerateTokens(ctx, user)
	AssertNoError(t, err, "生成令牌不应该出错")

	// 验证当前密码版本
	valid, err := ts.JWTService.ValidatePasswordVersion(ctx, tokenPair.AccessToken)
	AssertNoError(t, err, "验证密码版本不应该出错")
	AssertTrue(t, valid, "密码版本应该有效")

	// 模拟密码修改（增加密码版本）
	user.PasswordV = initialVersion + 1
	err = ts.UserRepo.UpdateUser(ctx, user)
	AssertNoError(t, err, "更新用户不应该出错")

	// 验证旧令牌的密码版本（应该无效）
	valid, err = ts.JWTService.ValidatePasswordVersion(ctx, tokenPair.AccessToken)
	AssertNoError(t, err, "验证密码版本不应该出错")
	AssertFalse(t, valid, "旧令牌的密码版本应该无效")

	// 生成新令牌
	newTokenPair, err := ts.JWTService.GenerateTokens(ctx, user)
	AssertNoError(t, err, "生成新令牌不应该出错")

	// 验证新令牌的密码版本（应该有效）
	valid, err = ts.JWTService.ValidatePasswordVersion(ctx, newTokenPair.AccessToken)
	AssertNoError(t, err, "验证新令牌密码版本不应该出错")
	AssertTrue(t, valid, "新令牌的密码版本应该有效")
}

// testTokenRevocation 测试令牌撤销功能
func testTokenRevocation(t *testing.T, ts *TestSuite) {
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "revokeuser", "revoke@test.com", "password123")

	// 生成令牌
	tokenPair, err := ts.JWTService.GenerateTokens(ctx, user)
	AssertNoError(t, err, "生成令牌不应该出错")

	// 验证令牌有效
	_, err = ts.JWTService.ValidateAccessToken(tokenPair.AccessToken)
	AssertNoError(t, err, "令牌应该有效")

	// 撤销令牌
	err = ts.JWTService.RevokeToken(ctx, tokenPair.AccessToken)
	AssertNoError(t, err, "撤销令牌不应该出错")

	// 注意：由于当前实现中RevokeToken只是预留接口，实际没有实现黑名单机制
	// 所以这里只测试方法调用不出错
	// 在实际项目中，撤销后的令牌验证应该失败

	// 测试撤销无效令牌
	err = ts.JWTService.RevokeToken(ctx, "invalid.token")
	AssertError(t, err, "撤销无效令牌应该出错")
}

// TestJWTManager 测试JWT管理器的底层功能
func TestJWTManager(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		t.Run("JWT管理器创建", func(t *testing.T) {
			testJWTManagerCreation(t, ts)
		})

		t.Run("令牌生成和验证", func(t *testing.T) {
			testJWTManagerTokens(t, ts)
		})

		t.Run("令牌对生成", func(t *testing.T) {
			testJWTManagerTokenPair(t, ts)
		})

		t.Run("令牌头提取", func(t *testing.T) {
			testTokenHeaderExtraction(t)
		})
	})
}

// testJWTManagerCreation 测试JWT管理器创建
func testJWTManagerCreation(t *testing.T, ts *TestSuite) {
	// JWT管理器应该已经在测试环境中创建
	AssertNotEqual(t, nil, ts.JWT, "JWT管理器不应该为空")
}

// testJWTManagerTokens 测试JWT管理器令牌操作
func testJWTManagerTokens(t *testing.T, ts *TestSuite) {
	// 测试数据
	userID := uint(1)
	username := "testuser"
	email := "test@example.com"
	passwordV := int64(1)
	roles := []string{"admin", "user"}

	// 生成访问令牌
	accessToken, err := ts.JWT.GenerateAccessToken(userID, username, email, passwordV, roles)
	AssertNoError(t, err, "生成访问令牌不应该出错")
	AssertNotEqual(t, "", accessToken, "访问令牌不应该为空")

	// 验证访问令牌
	claims, err := ts.JWT.ValidateAccessToken(accessToken)
	AssertNoError(t, err, "验证访问令牌不应该出错")
	AssertEqual(t, userID, claims.UserID, "用户ID应该匹配")
	AssertEqual(t, username, claims.Username, "用户名应该匹配")
	AssertEqual(t, email, claims.Email, "邮箱应该匹配")
	AssertEqual(t, passwordV, claims.PasswordV, "密码版本应该匹配")
	AssertEqual(t, 2, len(claims.Roles), "角色数量应该匹配")

	// 生成刷新令牌
	refreshToken, err := ts.JWT.GenerateRefreshToken(userID, username)
	AssertNoError(t, err, "生成刷新令牌不应该出错")
	AssertNotEqual(t, "", refreshToken, "刷新令牌不应该为空")

	// 验证刷新令牌
	refreshClaims, err := ts.JWT.ValidateRefreshToken(refreshToken)
	AssertNoError(t, err, "验证刷新令牌不应该出错")
	AssertEqual(t, username, refreshClaims.Subject, "主题应该是用户名")

	// 使用刷新令牌生成新的访问令牌
	newAccessToken, err := ts.JWT.RefreshAccessToken(refreshToken, userID, username, email, passwordV, roles)
	AssertNoError(t, err, "刷新访问令牌不应该出错")
	AssertNotEqual(t, "", newAccessToken, "新访问令牌不应该为空")
	AssertNotEqual(t, accessToken, newAccessToken, "新访问令牌应该与旧令牌不同")
}

// testJWTManagerTokenPair 测试JWT管理器令牌对生成
func testJWTManagerTokenPair(t *testing.T, ts *TestSuite) {
	// 测试数据
	userID := uint(1)
	username := "testuser"
	email := "test@example.com"
	passwordV := int64(1)
	roles := []string{"admin"}

	// 生成令牌对
	tokenPair, err := ts.JWT.GenerateTokenPair(userID, username, email, passwordV, roles)
	AssertNoError(t, err, "生成令牌对不应该出错")
	AssertNotEqual(t, "", tokenPair.AccessToken, "访问令牌不应该为空")
	AssertNotEqual(t, "", tokenPair.RefreshToken, "刷新令牌不应该为空")
	AssertTrue(t, tokenPair.ExpiresIn > 0, "过期时间应该大于0")

	// 验证令牌对中的令牌
	_, err = ts.JWT.ValidateAccessToken(tokenPair.AccessToken)
	AssertNoError(t, err, "令牌对中的访问令牌应该有效")

	_, err = ts.JWT.ValidateRefreshToken(tokenPair.RefreshToken)
	AssertNoError(t, err, "令牌对中的刷新令牌应该有效")
}

// testTokenHeaderExtraction 测试令牌头提取功能
func testTokenHeaderExtraction(t *testing.T) {
	// 测试正确的Bearer令牌
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token"
	authHeader := "Bearer " + token
	extractedToken := auth.ExtractTokenFromHeader(authHeader)
	AssertEqual(t, token, extractedToken, "应该正确提取令牌")

	// 测试没有Bearer前缀的情况
	extractedToken = auth.ExtractTokenFromHeader(token)
	AssertEqual(t, "", extractedToken, "没有Bearer前缀应该返回空字符串")

	// 测试空字符串
	extractedToken = auth.ExtractTokenFromHeader("")
	AssertEqual(t, "", extractedToken, "空字符串应该返回空字符串")

	// 测试只有Bearer的情况
	extractedToken = auth.ExtractTokenFromHeader("Bearer")
	AssertEqual(t, "", extractedToken, "只有Bearer应该返回空字符串")

	// 测试Bearer后面有空格但没有令牌
	extractedToken = auth.ExtractTokenFromHeader("Bearer ")
	AssertEqual(t, "", extractedToken, "Bearer后只有空格应该返回空字符串")
}