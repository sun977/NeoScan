package brute

import (
	"strings"
)

// DefaultTopUsers 内置 Top 用户名 (Keep it small for binary size)
var DefaultTopUsers = []string{
	"root", "admin", "user", "test", "guest",
	"postgres", "mysql", "oracle", "weblogic",
	"administrator", "service", "system",
}

// DefaultTopPasswords 内置 Top 弱口令
var DefaultTopPasswords = []string{
	"123456", "password", "12345678", "123456789", "12345", "123",
	"root", "admin", "test", "111111", "1234567",
	"%user%", "%user%123", "%user%@123", "123%user%",
}

// DictManager 字典管理器
type DictManager struct {
	// 可以在这里扩展文件加载逻辑，目前保持无状态
}

// NewDictManager 创建字典管理器
func NewDictManager() *DictManager {
	return &DictManager{}
}

// Generate 生成爆破凭据列表
// params: 任务参数
//   - "users": []string 或 string (逗号分隔), 覆盖内置用户名
//   - "passwords": []string 或 string (逗号分隔), 覆盖内置密码
// mode: 爆破模式
func (d *DictManager) Generate(params map[string]interface{}, mode AuthMode) []Auth {
	var list []Auth

	// 1. 提取 Users 和 Passwords
	users := extractStringSlice(params, "users", DefaultTopUsers)
	passs := extractStringSlice(params, "passwords", DefaultTopPasswords)

	// 2. 根据模式生成
	switch mode {
	case AuthModeUserPass:
		// 笛卡尔积: User * Pass
		for _, u := range users {
			for _, p := range passs {
				// 动态替换 %user%
				realPass := strings.ReplaceAll(p, "%user%", u)
				list = append(list, Auth{Username: u, Password: realPass})
			}
		}

	case AuthModeOnlyPass:
		// 仅遍历密码
		for _, p := range passs {
			// 在 OnlyPass 模式下，%user% 通常没有意义，除非有上下文
			// 这里简单处理：如果有 %user%，替换为空字符串或者保持原样？
			// Qscan 的逻辑是 OnlyPass 模式下不需要用户名。
			// 但如果密码本身包含 %user%，我们应该替换成什么？
			// 对于 Redis/VNC，通常没有用户名的概念。
			// 策略：OnlyPass 模式下，忽略包含 %user% 的密码，或者替换为 "admin"/"root" (猜测)？
			// 简单起见，替换为 "root" (假设默认用户) 或者直接保留。
			// 更好的策略：如果密码包含 %user%，则跳过，因为没有 user。
			// 或者，暂时不处理 %user% 在 OnlyPass 下的情况，直接保留。
			
			// Update: 实际上有些 OnlyPass 场景可能也隐含用户名 (如 VNC 有时默认 user)，
			// 但标准 OnlyPass 确实只发密码。
			// 考虑到 %user% 是为了 User:Pass 组合设计的，这里我们做个防御性处理：
			// 如果包含 %user%，则替换为 "admin" (最常见默认用户)
			realPass := strings.ReplaceAll(p, "%user%", "admin")
			list = append(list, Auth{Password: realPass})
		}

	case AuthModeNone:
		// 空凭据 (用于探测未授权)
		list = append(list, Auth{})
	}

	return list
}

// extractStringSlice 从 map 中提取字符串切片
func extractStringSlice(m map[string]interface{}, key string, defaultVal []string) []string {
	if m == nil {
		return defaultVal
	}

	val, ok := m[key]
	if !ok {
		return defaultVal
	}

	switch v := val.(type) {
	case []string:
		if len(v) == 0 {
			return defaultVal
		}
		return v
	case string:
		if v == "" {
			return defaultVal
		}
		// 支持逗号分隔
		parts := strings.Split(v, ",")
		var res []string
		for _, p := range parts {
			trim := strings.TrimSpace(p)
			if trim != "" {
				res = append(res, trim)
			}
		}
		if len(res) == 0 {
			return defaultVal
		}
		return res
	case []interface{}:
		var res []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				res = append(res, s)
			}
		}
		if len(res) == 0 {
			return defaultVal
		}
		return res
	}

	return defaultVal
}
