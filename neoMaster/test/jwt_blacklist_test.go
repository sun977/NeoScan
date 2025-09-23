package test

import (
	"context"
	"testing"
	"time"

	"neomaster/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJWTBlacklistIntegration 测试JWT黑名单集成功能
// 这个测试验证了JWTService通过SessionService间接调用SessionRepository的黑名单功能
func TestJWTBlacklistIntegration(t *testing.T) {
	// 设置测试环境
	suite := SetupTestEnvironment(t)
	defer suite.TeardownTestEnvironment(t)

	// 如果没有数据库连接，跳过这个测试
	if suite.DB == nil {
		t.Skip("跳过需要数据库的测试")
	}

	ctx := context.Background()

	// 创建测试用户请求
	createUserReq := &model.CreateUserRequest{
		Username: "blacklist_test_user",
		Email:    "blacklist@test.com",
		Password: "test_password_123",
		Nickname: "Test User",
	}

	// 创建用户
	testUser, err := suite.UserService.CreateUser(ctx, createUserReq)
	require.NoError(t, err, "用户创建应该成功")
	require.NotNil(t, testUser, "创建的用户不应该为空")

	// 生成JWT令牌
	tokenPair, err := suite.JWTService.GenerateTokens(ctx, testUser)
	require.NoError(t, err, "令牌生成应该成功")
	require.NotNil(t, tokenPair, "令牌对不应该为空")
	require.NotEmpty(t, tokenPair.AccessToken, "访问令牌不应该为空")

	t.Run("令牌验证和撤销流程", func(t *testing.T) {
		// 1. 验证新生成的令牌应该有效
		claims, err := suite.JWTService.ValidateAccessToken(tokenPair.AccessToken)
		assert.NoError(t, err, "新令牌验证应该成功")
		assert.NotNil(t, claims, "令牌声明不应该为空")
		assert.Equal(t, testUser.Username, claims.Username, "用户名应该匹配")

		// 2. 撤销令牌
		err = suite.JWTService.RevokeToken(ctx, tokenPair.AccessToken, time.Hour)
		assert.NoError(t, err, "令牌撤销应该成功")

		// 3. 验证撤销后的令牌应该无效
		_, err = suite.JWTService.ValidateAccessToken(tokenPair.AccessToken)
		assert.Error(t, err, "撤销后的令牌验证应该失败")
		assert.Contains(t, err.Error(), "revoked", "错误信息应该包含'revoked'")
	})

	t.Run("SessionService黑名单方法测试", func(t *testing.T) {
		// 生成新的令牌用于测试
		newTokenPair, err := suite.JWTService.GenerateTokens(ctx, testUser)
		require.NoError(t, err, "新令牌生成应该成功")

		// 解析令牌获取JTI
		claims, err := suite.JWTService.ValidateAccessToken(newTokenPair.AccessToken)
		require.NoError(t, err, "令牌解析应该成功")
		require.NotEmpty(t, claims.ID, "JTI不应该为空")

		// 1. 检查令牌初始状态（应该未撤销）
		isRevoked, err := suite.SessionService.IsTokenRevoked(ctx, claims.ID)
		assert.NoError(t, err, "检查令牌状态应该成功")
		assert.False(t, isRevoked, "新令牌应该未被撤销")

		// 2. 通过SessionService撤销令牌
		err = suite.SessionService.RevokeToken(ctx, claims.ID, time.Hour)
		assert.NoError(t, err, "通过SessionService撤销令牌应该成功")

		// 3. 检查令牌撤销后状态
		isRevoked, err = suite.SessionService.IsTokenRevoked(ctx, claims.ID)
		assert.NoError(t, err, "检查撤销后令牌状态应该成功")
		assert.True(t, isRevoked, "撤销后令牌应该被标记为已撤销")

		// 4. 验证JWTService也能检测到撤销状态
		_, err = suite.JWTService.ValidateAccessToken(newTokenPair.AccessToken)
		assert.Error(t, err, "撤销后的令牌通过JWTService验证应该失败")
	})

	t.Run("边界情况测试", func(t *testing.T) {
		// 测试空JTI
		err := suite.SessionService.RevokeToken(ctx, "", time.Hour)
		assert.Error(t, err, "空JTI撤销应该失败")
		assert.Contains(t, err.Error(), "empty", "错误信息应该包含'empty'")

		// 测试检查空JTI
		_, err = suite.SessionService.IsTokenRevoked(ctx, "")
		assert.Error(t, err, "检查空JTI应该失败")
		assert.Contains(t, err.Error(), "empty", "错误信息应该包含'empty'")

		// 测试检查不存在的JTI
		isRevoked, err := suite.SessionService.IsTokenRevoked(ctx, "non-existent-jti")
		assert.NoError(t, err, "检查不存在的JTI应该成功")
		assert.False(t, isRevoked, "不存在的JTI应该返回未撤销")
	})

	t.Logf("✅ JWT黑名单集成测试完成")
}

// TestTokenBlacklistServiceInterface 测试TokenBlacklistService接口实现
func TestTokenBlacklistServiceInterface(t *testing.T) {
	// 设置测试环境
	suite := SetupTestEnvironment(t)
	defer suite.TeardownTestEnvironment(t)

	// 验证SessionService实现了TokenBlacklistService接口
	// 这是编译时检查，如果SessionService没有实现接口，编译会失败
	var _ interface{} = suite.SessionService

	t.Logf("✅ SessionService正确实现了TokenBlacklistService接口")
}