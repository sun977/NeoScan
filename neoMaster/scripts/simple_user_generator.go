package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/auth"
)

// 简单用户数据生成器
// 使用方法:
//
//	源码运行: go run scripts/simple_user_generator.go <用户名> <邮箱> <密码> <密码版本>
//	二进制运行: scripts/simple_user_generator.exe <用户名> <邮箱> <密码> <密码版本>
func main() {
	// 检查命令行参数
	if len(os.Args) != 5 {
		fmt.Println("使用方法:")
		fmt.Println("  源码运行: go run scripts/simple_user_generator.go <用户名> <邮箱> <密码> <密码版本>")
		fmt.Println("  二进制运行: scripts/simple_user_generator.exe <用户名> <邮箱> <密码> <密码版本>")
		fmt.Println("")
		fmt.Println("示例:")
		fmt.Println("  go run scripts/simple_user_generator.go admin admin@example.com AdminPass123! 1")
		fmt.Println("  scripts/simple_user_generator.exe admin admin@example.com AdminPass123! 1")
		fmt.Println("  scripts/simple_user_generator.exe testuser test@example.com TestPass123! 2")
		return
	}

	// 获取命令行参数
	username := os.Args[1]
	email := os.Args[2]
	password := os.Args[3]
	passwordVersionStr := os.Args[4]

	// 解析密码版本号
	passwordVersion, err := strconv.ParseInt(passwordVersionStr, 10, 64)
	if err != nil {
		log.Fatalf("密码版本号解析失败: %v", err)
	}

	// 验证密码版本号范围
	if passwordVersion < 1 {
		log.Fatalf("密码版本号必须大于等于1，当前值: %d", passwordVersion)
	}

	// 验证输入参数
	if err := validateInput(username, email, password); err != nil {
		log.Fatalf("输入验证失败: %v", err)
	}

	// 生成用户数据
	userData, err := generateUserData(username, email, password, passwordVersion)
	if err != nil {
		log.Fatalf("生成用户数据失败: %v", err)
	}

	// 输出JSON结果
	fmt.Println(userData)
}

// validateInput 验证输入参数
func validateInput(username, email, password string) error {
	// 验证用户名
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if len(username) < 3 || len(username) > 50 {
		return fmt.Errorf("用户名长度必须在3-50个字符之间")
	}

	// 验证邮箱格式
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("邮箱格式不正确")
	}

	// 验证密码强度
	if err := auth.ValidatePasswordStrength(password); err != nil {
		return fmt.Errorf("密码强度验证失败: %w", err)
	}

	return nil
}

// generateUserData 生成用户数据
func generateUserData(username, email, password string, passwordVersion int64) (string, error) {
	// 哈希密码
	hashedPassword, err := auth.HashPasswordWithDefaultConfig(password)
	if err != nil {
		return "", fmt.Errorf("密码哈希失败: %w", err)
	}

	// 创建用户数据
	now := time.Now()
	user := &model.User{
		ID:          0,                       // 数据库自增ID，设为0
		Username:    username,                // 用户名
		Email:       email,                   // 邮箱
		Password:    hashedPassword,          // 哈希后的密码
		PasswordV:   passwordVersion,         // 密码版本号（用户指定）
		Nickname:    "",                      // 昵称（空）
		Avatar:      "",                      // 头像（空）
		Phone:       "",                      // 电话（空）
		SocketId:    "",                      // Socket ID（空）
		Remark:      "",                      // 备注（空）
		Status:      model.UserStatusEnabled, // 用户状态：启用
		LastLoginAt: nil,                     // 最后登录时间（新用户为空）
		LastLoginIP: "",                      // 最后登录IP（空）
		CreatedAt:   now,                     // 创建时间
		UpdatedAt:   now,                     // 更新时间
		DeletedAt:   nil,                     // 软删除时间（空）
		Roles:       nil,                     // 用户角色（空）
	}

	// 创建一个包含所有字段的结构体用于JSON输出（包括通常隐藏的字段）
	userOutput := struct {
		ID          uint             `json:"id"`
		Username    string           `json:"username"`
		Email       string           `json:"email"`
		Password    string           `json:"password"`   // 显示哈希后的密码
		PasswordV   int64            `json:"password_v"` // 显示密码版本号
		Nickname    string           `json:"nickname"`
		Avatar      string           `json:"avatar"`
		Phone       string           `json:"phone"`
		SocketId    string           `json:"socket_id"`
		Remark      string           `json:"remark"`
		Status      model.UserStatus `json:"status"`
		LastLoginAt *time.Time       `json:"last_login_at"`
		LastLoginIP string           `json:"last_login_ip"`
		CreatedAt   time.Time        `json:"created_at"`
		UpdatedAt   time.Time        `json:"updated_at"`
		DeletedAt   *time.Time       `json:"deleted_at"`
		Roles       []*model.Role    `json:"roles"`
	}{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Password:    user.Password,
		PasswordV:   user.PasswordV,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Phone:       user.Phone,
		SocketId:    user.SocketId,
		Remark:      user.Remark,
		Status:      user.Status,
		LastLoginAt: user.LastLoginAt,
		LastLoginIP: user.LastLoginIP,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		DeletedAt:   user.DeletedAt,
		Roles:       user.Roles,
	}

	// 转换为JSON格式
	jsonData, err := json.MarshalIndent(userOutput, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON序列化失败: %w", err)
	}

	return string(jsonData), nil
}
