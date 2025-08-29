/**
 * 工具类:密码工具
 * @author: sun977
 * @date: 2025.08.29
 * @description: J密码工具类
 * @func:
 * 	1.哈希密码
 * 	2.验证密码
 */
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2" // 引入Argon2id算法
)

// PasswordConfig 密码配置
type PasswordConfig struct {
	Memory      uint32 // 内存使用量 (KB)
	Iterations  uint32 // 迭代次数
	Parallelism uint8  // 并行度
	SaltLength  uint32 // 盐长度
	KeyLength   uint32 // 密钥长度
}

// DefaultPasswordConfig 默认密码配置
var DefaultPasswordConfig = &PasswordConfig{
	Memory:      64 * 1024, // 64MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// PasswordManager 密码管理器
type PasswordManager struct {
	config *PasswordConfig
}

// NewPasswordManager 创建密码管理器
func NewPasswordManager(config *PasswordConfig) *PasswordManager {
	if config == nil {
		config = DefaultPasswordConfig
	}
	return &PasswordManager{
		config: config,
	}
}

// HashPassword 哈希密码
func (pm *PasswordManager) HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// 生成随机盐
	salt, err := generateRandomBytes(pm.config.SaltLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// 使用Argon2id算法哈希密码
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		pm.config.Iterations,
		pm.config.Memory,
		pm.config.Parallelism,
		pm.config.KeyLength,
	)

	// 编码为base64字符串
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// 格式: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		pm.config.Memory,
		pm.config.Iterations,
		pm.config.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

// VerifyPassword 验证密码
func (pm *PasswordManager) VerifyPassword(password, encodedHash string) (bool, error) {
	if password == "" || encodedHash == "" {
		return false, errors.New("password and hash cannot be empty")
	}

	// 解析哈希字符串
	config, salt, hash, err := pm.decodeHash(encodedHash)
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// 使用相同参数哈希输入密码
	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		config.Iterations,
		config.Memory,
		config.Parallelism,
		config.KeyLength,
	)

	// 使用常量时间比较防止时序攻击
	return subtle.ConstantTimeCompare(hash, otherHash) == 1, nil
}

// decodeHash 解码哈希字符串
func (pm *PasswordManager) decodeHash(encodedHash string) (*PasswordConfig, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, errors.New("invalid hash format")
	}

	// 检查算法
	if parts[1] != "argon2id" {
		return nil, nil, nil, errors.New("unsupported algorithm")
	}

	// 解析版本
	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, nil, errors.New("incompatible version")
	}

	// 解析参数
	config := &PasswordConfig{}
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &config.Memory, &config.Iterations, &config.Parallelism)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// 解码盐
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid salt: %w", err)
	}
	config.SaltLength = uint32(len(salt))

	// 解码哈希
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid hash: %w", err)
	}
	config.KeyLength = uint32(len(hash))

	return config, salt, hash, nil
}

// generateRandomBytes 生成随机字节
func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// ValidatePasswordStrength 验证密码强度
func ValidatePasswordStrength(password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters long")
	}

	if len(password) > 128 {
		return errors.New("password must be no more than 128 characters long")
	}

	// 检查是否包含至少一个字母和一个数字
	hasLetter := false
	hasDigit := false

	for _, char := range password {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			hasLetter = true
		}
		if char >= '0' && char <= '9' {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			break
		}
	}

	if !hasLetter {
		return errors.New("password must contain at least one letter")
	}

	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}

	return nil
}

// GenerateRandomPassword 生成随机密码
func GenerateRandomPassword(length int) (string, error) {
	if length < 6 {
		length = 6
	}
	if length > 128 {
		length = 128
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	for i := range b {
		randomBytes, err := generateRandomBytes(1)
		if err != nil {
			return "", err
		}
		b[i] = charset[randomBytes[0]%byte(len(charset))]
	}
	return string(b), nil
}

// HashPasswordWithDefaultConfig 使用默认配置哈希密码
func HashPasswordWithDefaultConfig(password string) (string, error) {
	pm := NewPasswordManager(nil)
	return pm.HashPassword(password)
}

// VerifyPasswordWithDefaultConfig 使用默认配置验证密码
func VerifyPasswordWithDefaultConfig(password, encodedHash string) (bool, error) {
	pm := NewPasswordManager(nil)
	return pm.VerifyPassword(password, encodedHash)
}
