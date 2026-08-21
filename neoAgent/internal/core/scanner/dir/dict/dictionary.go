package dict

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"neoagent/internal/pkg/logger"
)

const (
	// defaultMaxSize 字典条目上限，防止内存爆炸
	defaultMaxSize = 500_000
)

// DirOptions 字典构建所需的选项子集。
// 完整的 DirScanOptions 定义在 options 包中，这里只声明 Dictionary 需要的字段，
// 避免循环依赖（options 包会 import scanner/dir，不能反向 import）。
type DirOptions struct {
	Wordlists  []string      // 用户额外指定的字典文件路径
	Categories []string      // 技术栈分类字典名称（如 "wordpress"、"php/laravel"），见 LoadCategoryWordlist
	Extensions []string      // 文件扩展名列表（用于 %EXT% 展开）
	ExtMode    ExtensionMode // 扩展模式（Classic / Force）
	Uppercase  bool          // 全部转大写
	Lowercase  bool          // 全部转小写
	Capital    bool          // 首字母大写（路径首字符）
	Prefixes   []string      // 追加前缀
	Suffixes   []string      // 追加后缀

	// SkipBuiltin 跳过内置字典（dicc.txt），仅使用 Wordlists/自定义规则。
	// 默认 false（生产环境行为不变）。仅供单元测试构造小规模、确定性的
	// 字典使用，避免真实内置字典中的噪声路径干扰断言。
	SkipBuiltin bool
}

// Dictionary 是线程安全的双队列字典迭代器。
//
// 设计：
//   - main  队列：预加载的字典条目，初始化后只读
//   - extra 队列：递归扫描发现的动态子路径，优先消费
//
// Next() 先消费 extra，再消费 main；两者都耗尽返回 ("", false)。
type Dictionary struct {
	mu       sync.Mutex
	main     []string // 主字典（预加载，初始化后不再追加）
	extra    []string // 动态扩展队列（递归子路径）
	mainIdx  int
	extraIdx int
	maxSize  int // 字典条目上限
	total    int // 已生成条目计数（含展开后的数量）
}

// New 创建并初始化字典。加载顺序：
//  1. 内置字典（go:embed dicc.txt）
//  2. opts.Wordlists 中用户指定的额外文件
//  3. opts.Categories 中指定的技术栈分类字典（内置，go:embed categories/）
//  4. rules/dir/custom/*.txt（运行时加载，多路径 fallback）
//
// 加载完成后按 opts 中的扩展/大小写/前后缀配置展开。
func New(opts *DirOptions) (*Dictionary, error) {
	if opts == nil {
		opts = &DirOptions{}
	}

	d := &Dictionary{
		maxSize: defaultMaxSize,
	}

	// 1. 内置字典（测试场景可通过 SkipBuiltin 跳过，见 DirOptions 注释）
	if !opts.SkipBuiltin {
		builtin, err := LoadBuiltinWordlist()
		if err != nil {
			logger.Warnf("[DirDict] Failed to load builtin wordlist: %v", err)
			builtin = nil
		}
		d.addToMain(builtin)
	}

	// 2. 用户额外指定的字典文件
	for _, wlPath := range opts.Wordlists {
		lines, err := loadExternalFile(wlPath)
		if err != nil {
			logger.Warnf("[DirDict] Failed to load wordlist %s: %v", wlPath, err)
			continue
		}
		d.addToMain(lines)
	}

	// 3. 用户指定的技术栈分类字典（如 --category wordpress,spring）
	for _, category := range opts.Categories {
		lines, err := LoadCategoryWordlist(category)
		if err != nil {
			logger.Warnf("[DirDict] Failed to load category wordlist %q: %v", category, err)
			continue
		}
		d.addToMain(lines)
	}

	// 4. rules/dir/custom/*.txt 运行时加载（多路径 fallback，独立实现，不依赖其他扫描器）
	customLines := loadCustomRules()
	d.addToMain(customLines)

	// 5. 展开（%EXT% 替换、大小写、前后缀）
	d.expand(opts)

	logger.Infof("[DirDict] Dictionary initialized: %d entries (max=%d)", len(d.main), d.maxSize)
	return d, nil
}

// Next 从字典中取下一条路径。
// 优先返回 extra（递归新增的路径），extra 耗尽后返回 main。
// 两者均耗尽时返回 ("", false)。线程安全。
func (d *Dictionary) Next() (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 优先消费 extra
	if d.extraIdx < len(d.extra) {
		path := d.extra[d.extraIdx]
		d.extraIdx++
		return path, true
	}
	// 再消费 main
	if d.mainIdx < len(d.main) {
		path := d.main[d.mainIdx]
		d.mainIdx++
		return path, true
	}
	return "", false
}

// AddExtra 将一批路径追加到 extra 队列（递归扫描时调用）。
// 超过 maxSize 的条目直接丢弃。线程安全。
func (d *Dictionary) AddExtra(paths ...string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, p := range paths {
		if d.total >= d.maxSize {
			logger.Warnf("[DirDict] Max entries (%d) reached, dropping extra path: %s", d.maxSize, p)
			break
		}
		d.extra = append(d.extra, p)
		d.total++
	}
}

// Len 返回剩余未消费的条目数（估算，非精确同步值）。
func (d *Dictionary) Len() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return (len(d.extra) - d.extraIdx) + (len(d.main) - d.mainIdx)
}

// Total 返回字典当前总条目数（main + extra 的大小之和）。
func (d *Dictionary) Total() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.main) + len(d.extra)
}

// ── 内部方法 ──────────────────────────────────────────────────────────────────

// addToMain 批量追加到主字典，超限时截断。
func (d *Dictionary) addToMain(lines []string) {
	for _, line := range lines {
		if d.total >= d.maxSize {
			logger.Warnf("[DirDict] Max entries (%d) reached, truncating wordlist", d.maxSize)
			return
		}
		d.main = append(d.main, line)
		d.total++
	}
}

// expand 按 opts 展开主字典（原地替换）。
func (d *Dictionary) expand(opts *DirOptions) {
	if len(opts.Extensions) == 0 &&
		!opts.Uppercase && !opts.Lowercase && !opts.Capital &&
		len(opts.Prefixes) == 0 && len(opts.Suffixes) == 0 {
		return // 无需展开
	}

	var expanded []string
	for _, line := range d.main {
		// 1. 扩展名展开
		lines := ExpandLine(line, opts.Extensions, opts.ExtMode)

		// 2. 大小写变换 + 前后缀
		for _, l := range lines {
			variants := applyTransforms(l, opts)
			expanded = append(expanded, variants...)
		}

		// 超限截断
		if len(expanded) >= d.maxSize {
			expanded = expanded[:d.maxSize]
			logger.Warnf("[DirDict] Max entries (%d) reached during expansion, truncating", d.maxSize)
			break
		}
	}
	d.main = expanded
	d.total = len(d.main)
}

// applyTransforms 对单条路径应用大小写变换和前后缀组合。
// 若同时设置 Uppercase/Lowercase/Capital，优先级：Uppercase > Lowercase > Capital。
func applyTransforms(path string, opts *DirOptions) []string {
	// 大小写变换
	transformed := path
	if opts.Uppercase {
		transformed = strings.ToUpper(path)
	} else if opts.Lowercase {
		transformed = strings.ToLower(path)
	} else if opts.Capital {
		transformed = capitalizeFirst(path)
	}

	// 无前后缀时直接返回
	if len(opts.Prefixes) == 0 && len(opts.Suffixes) == 0 {
		return []string{transformed}
	}

	// 生成前后缀组合
	prefixes := opts.Prefixes
	if len(prefixes) == 0 {
		prefixes = []string{""} // 空前缀：不加前缀的原始路径
	}
	suffixes := opts.Suffixes
	if len(suffixes) == 0 {
		suffixes = []string{""} // 空后缀
	}

	var result []string
	for _, pfx := range prefixes {
		for _, sfx := range suffixes {
			result = append(result, pfx+transformed+sfx)
		}
	}
	// 如果有前缀或后缀，也保留原始路径
	if len(opts.Prefixes) > 0 || len(opts.Suffixes) > 0 {
		result = append([]string{transformed}, result...)
	}
	return result
}

// capitalizeFirst 将路径中第一个字母字符大写。
// 例："/admin" → "/Admin"，"admin" → "Admin"。
func capitalizeFirst(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsLetter(r) {
			runes[i] = unicode.ToUpper(r)
			return string(runes)
		}
	}
	return s
}

// loadExternalFile 从文件系统加载外部字典文件，过滤空行和注释。
func loadExternalFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

// loadCustomRules 从 rules/dir/custom/*.txt 加载用户自定义字典。
// 多路径 fallback 依次尝试，命中第一个存在的目录即加载。
// 不引用任何其他扫描器的代码（原子扫描器隔离原则）。
func loadCustomRules() []string {
	// 多路径 fallback，覆盖不同运行目录（项目根/测试包目录等）
	candidateDirs := []string{
		"rules/dir/custom",
		"../rules/dir/custom",
		"../../rules/dir/custom",
		"../../../rules/dir/custom",
		"../../../../rules/dir/custom",
		"../../../../../rules/dir/custom",
		"neoAgent/rules/dir/custom",
	}

	for _, dir := range candidateDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // 目录不存在或无法读取，尝试下一个
		}

		hasReadDir := true // ReadDir 成功执行，确认目录存在
		var all []string
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
				continue
			}
			fullPath := filepath.Join(dir, entry.Name())
			lines, loadErr := loadExternalFile(fullPath)
			if loadErr != nil {
				logger.Warnf("[DirDict] Failed to load custom rule %s: %v", fullPath, loadErr)
				continue
			}
			all = append(all, lines...)
			logger.Infof("[DirDict] Loaded custom wordlist: %s (%d entries)", fullPath, len(lines))
		}

		if len(all) > 0 {
			return all
		}
		// 目录已确认存在但无可用文件，不继续 fallback
		_ = hasReadDir
		return nil
	}

	return nil
}
