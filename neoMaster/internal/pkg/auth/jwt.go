/**
 * 工具类:JWT工具
 * @author: sun977
 * @date: 2025.08.29
 * @description: JWT工具类
 * @func:
 * 	1.创建JWT
 * 	2.验证JWT
 * 	3.刷新JWT
 */

package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5" // 引入jwt包
)

// JWTClaims JWT声明结构
type JWTClaims struct {
	UserID    uint     `json:"user_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	PasswordV int64    `json:"password_v"` // 密码版本号，用于使旧token失效
	Roles     []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secretKey string, accessTokenTTL, refreshTokenTTL time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// GenerateAccessToken 生成访问令牌
func (j *JWTManager) GenerateAccessToken(userID uint, username, email string, passwordV int64, roles []string) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		UserID:    userID,
		Username:  username,
		Email:     email,
		PasswordV: passwordV,
		Roles:     roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "neoscan",
			Subject:   username,
			Audience:  []string{"neoscan-web"},
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateJTI(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// GenerateRefreshToken 生成刷新令牌
func (j *JWTManager) GenerateRefreshToken(userID uint, username string) (string, error) {
	now := time.Now()
	claims := &jwt.RegisteredClaims{
		Issuer:    "neoscan",
		Subject:   username,
		Audience:  []string{"neoscan-refresh"},
		ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshTokenTTL)),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        generateJTI(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateAccessToken 验证访问令牌
func (j *JWTManager) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// 检查令牌是否过期
		if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
			return nil, errors.New("token has expired")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateRefreshToken 验证刷新令牌
func (j *JWTManager) ValidateRefreshToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		// 检查令牌是否过期
		if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
			return nil, errors.New("refresh token has expired")
		}
		// 检查是否是刷新令牌
		if len(claims.Audience) == 0 || claims.Audience[0] != "neoscan-refresh" {
			return nil, errors.New("invalid refresh token")
		}
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}

// RefreshAccessToken 刷新访问令牌
func (j *JWTManager) RefreshAccessToken(refreshTokenString string, userID uint, username, email string, passwordV int64, roles []string) (string, error) {
	// 验证刷新令牌
	_, err := j.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	// 生成新的访问令牌
	return j.GenerateAccessToken(userID, username, email, passwordV, roles)
}

// ExtractTokenFromHeader 从Authorization头中提取令牌
func ExtractTokenFromHeader(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// generateJTI 生成JWT ID
func generateJTI() string {
	// 使用纳秒级时间戳确保唯一性
	now := time.Now()
	return now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.Nanosecond())
}

// TokenPair 令牌对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// GenerateTokenPair 生成令牌对
func (j *JWTManager) GenerateTokenPair(userID uint, username, email string, passwordV int64, roles []string) (*TokenPair, error) {
	accessToken, err := j.GenerateAccessToken(userID, username, email, passwordV, roles)
	if err != nil {
		return nil, err
	}

	refreshToken, err := j.GenerateRefreshToken(userID, username)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(j.accessTokenTTL.Seconds()),
	}, nil
}

// GetUserIDFromToken 从访问令牌中获取用户ID
func (j *JWTManager) GetUserIDFromToken(tokenString string) (uint, error) {
	claims, err := j.ValidateAccessToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

// GetUsernameFromRefreshToken 从刷新令牌中获取用户名
func (j *JWTManager) GetUsernameFromRefreshToken(tokenString string) (string, error) {
	claims, err := j.ValidateRefreshToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.Subject, nil
}

// GetUsernameFromToken 从令牌中获取用户名
func (j *JWTManager) GetUsernameFromToken(tokenString string) (string, error) {
	claims, err := j.ValidateAccessToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.Username, nil
}

// GetRolesFromToken 从令牌中获取用户角色
func (j *JWTManager) GetRolesFromToken(tokenString string) ([]string, error) {
	claims, err := j.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}
	return claims.Roles, nil
}
