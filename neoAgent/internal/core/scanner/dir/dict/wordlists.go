// Package dict 提供 DirScanner 的字典管理能力。
// 内置字典通过 go:embed 编译进二进制，零配置启动；
// 用户自定义字典在运行时从 rules/dir/custom/ 加载，免重新编译。
//
// 注意：此包仅供 scanner/dir 包内部使用，不对其他扫描器暴露（原子扫描器隔离原则）。
package dict

import (
	"bufio"
	"embed"
	"errors"
	"io/fs"
	"strings"
)

// ErrCategoryNotFound 当指定的技术栈分类文件不存在时返回
var ErrCategoryNotFound = errors.New("category wordlist not found")

//go:embed wordlists
var wordlistsFS embed.FS

// LoadBuiltinWordlist 加载内置主字典 dicc.txt。
// 自动过滤空行和 # 开头的注释行。
func LoadBuiltinWordlist() ([]string, error) {
	return loadLines("wordlists/dicc.txt")
}

// LoadCategoryWordlist 按技术栈名称加载对应字典。
// category 支持直接传文件名（如 "wordpress"）或带子目录路径（如 "php/wordpress"）。
// 若两种方式都找不到，返回 ErrCategoryNotFound。
func LoadCategoryWordlist(category string) ([]string, error) {
	// 优先尝试直接路径（如 "php/wordpress"）
	directPath := "wordlists/categories/" + category + ".txt"
	lines, err := loadLines(directPath)
	if err == nil {
		return lines, nil
	}

	// 再尝试在各子目录中递归查找（如 "wordpress" → "php/wordpress.txt"）
	found, err2 := findInSubdirs("wordlists/categories", category+".txt")
	if err2 != nil {
		return nil, ErrCategoryNotFound
	}
	return loadLines(found)
}

// LoadUserAgents 加载 User-Agent 列表。
func LoadUserAgents() ([]string, error) {
	return loadLines("wordlists/user-agents.txt")
}

// ListCategories 列出所有可用的技术栈字典名称（遍历 categories/ 子目录下所有 .txt 文件）。
// 返回格式为相对于 categories/ 的路径，如 "wordpress"、"php/wordpress"。
func ListCategories() []string {
	var categories []string

	_ = fs.WalkDir(wordlistsFS, "wordlists/categories", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".txt") {
			// 截去 "wordlists/categories/" 前缀和 ".txt" 后缀
			rel := strings.TrimPrefix(path, "wordlists/categories/")
			rel = strings.TrimSuffix(rel, ".txt")
			// 排除 generate_wpscan_wordlists.py 等非字典文件（已经过 .txt 过滤，但保险起见）
			categories = append(categories, rel)
		}
		return nil
	})

	return categories
}

// ── 内部辅助函数 ──────────────────────────────────────────────────────────────

// loadLines 从嵌入 FS 读取文件，返回过滤后的路径行列表。
// 自动跳过空行和 # 开头的注释行。
func loadLines(path string) ([]string, error) {
	f, err := wordlistsFS.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r") // 兼容 Windows 换行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

// findInSubdirs 在指定目录的子目录中查找目标文件名，返回第一个匹配的完整路径。
func findInSubdirs(dir, filename string) (string, error) {
	var found string

	err := fs.WalkDir(wordlistsFS, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, "/"+filename) {
			found = path
			return fs.SkipAll // 找到即停
		}
		return nil
	})

	if err != nil || found == "" {
		return "", errors.New("not found")
	}
	return found, nil
}
