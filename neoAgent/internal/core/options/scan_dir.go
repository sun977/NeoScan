package options

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"neoagent/internal/core/model"
)

// DirScanOptions 是 dir_scan 任务的 CLI 参数集合，字段对应
// scanner/dir.DirOptions（设计文档 8.2 节）。key 命名统一使用下划线，
// 与 dir.parseDirOptions() 的解析约定保持一致。
type DirScanOptions struct {
	Target string

	// 字典配置
	Wordlists       string
	Category        string
	Extensions      string
	ForceExtensions bool
	Prefixes        string
	Suffixes        string
	Uppercase       bool
	Lowercase       bool
	Capital         bool

	// 扫描控制
	Threads           int
	Timeout           int
	MaxRetries        int
	Method            string
	Recursive         bool
	DeepRecursive     bool
	ForceRecursive    bool
	MaxRecursionDepth int
	RateLimit         int
	Delay             int
	MaxTime           int
	TargetMaxTime     int

	// 过滤配置
	IncludeStatus string
	ExcludeStatus string
	ExcludeSize   string
	ExcludeText   []string
	ExcludeRegex  string

	// 请求配置：Headers 已在 CLI 层合并了 -H 与 --headers-file 的内容，
	// ToTask() 不再关心两者的来源区别。
	Headers          []string
	FollowRedirects  bool
	Proxy            string
	RandomAgent      bool
	UserAgent        string
	Auth             string
	IP               string
	NetworkInterface string

	// 多目标
	TargetsFile string

	// 输出/日志：仅控制 CLI 本地打印详细程度，不下发给 Runner，
	// 因此不写入 task.Params（Runner 不关心 CLI 端的展示偏好）。
	Verbose bool
	Quiet   bool

	Output OutputOptions
}

// NewDirScanOptions 返回带默认值的 DirScanOptions（实施文档 Task 3.1 要求的默认值）。
func NewDirScanOptions() *DirScanOptions {
	return &DirScanOptions{
		Threads:           25,
		Timeout:           10,
		MaxRetries:        2,
		ExcludeStatus:     "404,500,502,503",
		Recursive:         true,
		MaxRecursionDepth: 3,
		Extensions:        "php,asp,html,js,json,bak,git,env",
		Method:            "GET",
	}
}

// Validate 校验参数合法性，并编译/解析出结构化的过滤条件。
// Target 与 TargetsFile 至少一个非空；ExcludeStatus/ExcludeRegex 等在此处
// 完成一次性解析（失败即报错），避免非法配置进入扫描器后才炸。
func (o *DirScanOptions) Validate() error {
	if o.Target == "" && o.TargetsFile == "" {
		return fmt.Errorf("target is required (use <target> or --targets-file)")
	}

	if _, err := parseIntListCSV(o.ExcludeStatus); err != nil {
		return fmt.Errorf("invalid --exclude-status: %w", err)
	}
	if o.IncludeStatus != "" {
		if _, err := parseIntListCSV(o.IncludeStatus); err != nil {
			return fmt.Errorf("invalid --include-status: %w", err)
		}
	}
	if o.ExcludeSize != "" {
		if _, err := parseInt64ListCSV(o.ExcludeSize); err != nil {
			return fmt.Errorf("invalid --exclude-size: %w", err)
		}
	}
	if o.ExcludeRegex != "" {
		if _, err := regexp.Compile(o.ExcludeRegex); err != nil {
			return fmt.Errorf("invalid --exclude-regex: %w", err)
		}
	}

	if o.MaxRecursionDepth < 0 || o.MaxRecursionDepth > 10 {
		return fmt.Errorf("--max-recursion-depth must be in range 0~10, got %d", o.MaxRecursionDepth)
	}
	if o.Threads < 1 || o.Threads > 500 {
		return fmt.Errorf("--threads must be in range 1~500, got %d", o.Threads)
	}
	if o.Timeout < 1 || o.Timeout > 300 {
		return fmt.Errorf("--timeout must be in range 1~300, got %d", o.Timeout)
	}

	return nil
}

// ToTask 将 CLI 参数写入 task.Params，key 全部使用下划线命名，
// 与 scanner/dir.parseDirOptions() 的解析 key 一一对应。
func (o *DirScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeDirScan, o.Target)
	task.Timeout = 2 * time.Hour

	p := task.Params

	if o.Wordlists != "" {
		p["wordlists"] = o.Wordlists
	}
	if o.Category != "" {
		p["category"] = o.Category
	}
	if o.Extensions != "" {
		p["extensions"] = o.Extensions
	}
	p["force_extensions"] = o.ForceExtensions
	if o.Prefixes != "" {
		p["prefixes"] = o.Prefixes
	}
	if o.Suffixes != "" {
		p["suffixes"] = o.Suffixes
	}
	p["uppercase"] = o.Uppercase
	p["lowercase"] = o.Lowercase
	p["capital"] = o.Capital

	p["threads"] = o.Threads
	p["timeout"] = o.Timeout
	p["max_retries"] = o.MaxRetries
	if o.Method != "" {
		p["method"] = o.Method
	}
	p["recursive"] = o.Recursive
	p["deep_recursive"] = o.DeepRecursive
	p["force_recursive"] = o.ForceRecursive
	p["max_recursion_depth"] = o.MaxRecursionDepth
	if o.RateLimit > 0 {
		p["rate_limit"] = o.RateLimit
	}
	if o.Delay > 0 {
		p["delay"] = o.Delay
	}
	if o.MaxTime > 0 {
		p["max_time"] = o.MaxTime
	}
	if o.TargetMaxTime > 0 {
		p["target_max_time"] = o.TargetMaxTime
	}

	if o.IncludeStatus != "" {
		p["include_status"] = o.IncludeStatus
	}
	if o.ExcludeStatus != "" {
		p["exclude_status"] = o.ExcludeStatus
	}
	if o.ExcludeSize != "" {
		p["exclude_size"] = o.ExcludeSize
	}
	if len(o.ExcludeText) > 0 {
		p["exclude_text"] = strings.Join(o.ExcludeText, ",")
	}
	if o.ExcludeRegex != "" {
		p["exclude_regex"] = o.ExcludeRegex
	}

	p["follow_redirects"] = o.FollowRedirects
	if o.Proxy != "" {
		p["proxy"] = o.Proxy
	}
	if o.IP != "" {
		p["ip"] = o.IP
	}
	if o.NetworkInterface != "" {
		p["network_interface"] = o.NetworkInterface
	}
	if o.UserAgent != "" {
		p["user_agent"] = o.UserAgent
	}
	p["random_agent"] = o.RandomAgent
	if o.Auth != "" {
		p["auth"] = o.Auth
	}
	if len(o.Headers) > 0 {
		p["headers"] = strings.Join(o.Headers, "\n")
	}

	o.Output.ApplyToParams(p)

	return task
}

// ── 参数解析辅助函数（与 scanner/dir 包内的同名逻辑独立实现，仅用于 Validate 阶段预校验）──

func parseIntListCSV(s string) ([]int, error) {
	var result []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid integer", part)
		}
		result = append(result, n)
	}
	return result, nil
}

func parseInt64ListCSV(s string) ([]int64, error) {
	var result []int64
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid size", part)
		}
		result = append(result, n)
	}
	return result, nil
}
