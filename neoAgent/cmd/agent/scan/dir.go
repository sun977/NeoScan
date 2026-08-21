package scan

import (
	"context"
	"fmt"

	"neoagent/internal/core/model"
	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/runner"

	"github.com/spf13/cobra"
)

// NewDirScanCmd 创建 dir 子命令（目录/文件挖掘）。
// 参数设计见 docs/原子模块设计/目录扫描设计/目录扫描参数设计文档.md。
func NewDirScanCmd() *cobra.Command {
	opts := options.NewDirScanOptions()
	var (
		headers     []string
		headersFile string
		excludeText []string
	)

	cmd := &cobra.Command{
		Use:   "dir [target]",
		Short: "目录/文件挖掘",
		Long:  `使用字典对目标 Web 服务进行目录与敏感文件扫描，内置通配符（动态 404）检测，避免误报。`,
		Example: `  # 基本扫描（使用内置字典）
  neoagent scan dir https://example.com

  # 指定自定义字典与扩展名
  neoagent scan dir -t https://example.com -w wordlist.txt -e php,bak,env

  # 开启深度递归 + 限制递归深度
  neoagent scan dir https://example.com -r --deep-recursive --max-recursion-depth 2

  # 批量扫描多个目标
  neoagent scan dir --targets-file targets.txt`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 位置参数优先于 -t/--target（两者互斥场景下，显式位置参数更符合直觉）。
			if len(args) > 0 {
				opts.Target = args[0]
			}

			// 合并 -H 与 --headers-file：文件内容按行追加到 -H 列表，
			// ToTask() 之后不再关心两者来源。
			if headersFile != "" {
				fileHeaders, err := loadList(headersFile)
				if err != nil {
					return fmt.Errorf("failed to load headers file: %w", err)
				}
				headers = append(headers, fileHeaders...)
			}
			opts.Headers = headers
			opts.ExcludeText = excludeText

			if err := opts.Validate(); err != nil {
				return err
			}
			opts.Output = globalOutputOptions

			manager := runner.NewRunnerManager()
			targets, err := resolveDirTargets(opts)
			if err != nil {
				return err
			}

			var allResults []*model.TaskResult
			for _, target := range targets {
				opts.Target = target
				task := opts.ToTask()

				if !opts.Quiet {
					fmt.Printf("[*] Starting Dir Scan against %s...\n", target)
				}

				ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
				results, err := manager.Execute(ctx, task)
				cancel()
				if err != nil {
					fmt.Printf("[-] Dir scan failed for %s: %v\n", target, err)
					continue
				}
				allResults = append(allResults, results...)
			}

			if !opts.Quiet {
				fmt.Printf("[+] Dir Scan done: %d target(s) scanned\n", len(targets))
			}

			console := reporter.NewConsoleReporter()
			console.PrintResults(allResults)

			if globalOutputOptions.OutputJson != "" {
				saveJsonResult(globalOutputOptions.OutputJson, allResults)
			}
			if globalOutputOptions.OutputCsv != "" {
				if err := reporter.SaveCsvResult(globalOutputOptions.OutputCsv, allResults); err != nil {
					fmt.Printf("[-] Failed to save csv: %v\n", err)
				}
			}

			return nil
		},
	}

	flags := cmd.Flags()

	// P0 核心参数
	flags.StringVarP(&opts.Target, "target", "t", opts.Target, "目标 URL（也可用位置参数指定，如 scan dir <target>）")
	flags.StringVarP(&opts.Extensions, "extensions", "e", opts.Extensions, "文件扩展名列表（逗号分隔，用于 %EXT% 展开）")
	flags.StringVarP(&opts.Wordlists, "wordlists", "w", opts.Wordlists, "自定义字典文件路径（默认使用内置字典）")
	flags.StringVar(&opts.Category, "category", opts.Category, "追加内置技术栈分类字典（逗号分隔，如 wordpress,php/laravel）")
	flags.IntVar(&opts.Threads, "threads", opts.Threads, "并发线程数")
	flags.IntVar(&opts.Timeout, "timeout", opts.Timeout, "单个请求超时时间（秒）")
	flags.BoolVarP(&opts.Recursive, "recursive", "r", opts.Recursive, "对命中的目录进行递归扫描")

	// P1 重要参数
	flags.BoolVarP(&opts.ForceExtensions, "force-extensions", "f", opts.ForceExtensions, "对不含 %EXT% 的字典条目也追加扩展名变体")
	flags.StringVarP(&opts.ExcludeStatus, "exclude-status", "x", opts.ExcludeStatus, "排除的状态码（逗号分隔）")
	flags.BoolVar(&opts.DeepRecursive, "deep-recursive", opts.DeepRecursive, "深度递归：命中目录时逐级展开父目录一并扫描")
	flags.BoolVar(&opts.ForceRecursive, "force-recursive", opts.ForceRecursive, "强制递归：对每个命中路径都继续扫描，不限于目录")
	flags.IntVarP(&opts.MaxRecursionDepth, "max-recursion-depth", "R", opts.MaxRecursionDepth, "最大递归深度（0~10）")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", opts.Verbose, "输出详细扫描信息")
	flags.BoolVarP(&opts.Quiet, "quiet", "q", opts.Quiet, "静默模式，仅输出最终结果")
	flags.StringArrayVarP(&headers, "header", "H", nil, "自定义请求头 \"Key: Value\"，可多次指定")
	flags.StringVarP(&opts.Proxy, "proxy", "p", opts.Proxy, "代理地址（如 http://127.0.0.1:8080）")
	flags.BoolVarP(&opts.FollowRedirects, "follow-redirects", "F", opts.FollowRedirects, "跟随重定向")
	flags.IntVar(&opts.MaxRetries, "max-retries", opts.MaxRetries, "请求失败最大重试次数")
	flags.StringVarP(&opts.Method, "method", "m", opts.Method, "HTTP 请求方法")

	// P2 可选参数
	flags.StringVar(&opts.Prefixes, "prefixes", opts.Prefixes, "追加前缀（逗号分隔）")
	flags.StringVar(&opts.Suffixes, "suffixes", opts.Suffixes, "追加后缀（逗号分隔）")
	flags.BoolVarP(&opts.Uppercase, "uppercase", "U", opts.Uppercase, "字典条目转大写")
	flags.BoolVarP(&opts.Lowercase, "lowercase", "L", opts.Lowercase, "字典条目转小写")
	flags.BoolVarP(&opts.Capital, "capital", "C", opts.Capital, "字典条目首字母大写")
	flags.StringVarP(&opts.IncludeStatus, "include-status", "i", opts.IncludeStatus, "仅包含的状态码（逗号分隔，优先于 --exclude-status）")
	flags.StringArrayVar(&excludeText, "exclude-text", nil, "响应体中排除的关键字，可多次指定")
	flags.StringVar(&opts.ExcludeRegex, "exclude-regex", opts.ExcludeRegex, "响应体排除正则")
	flags.StringVar(&opts.ExcludeSize, "exclude-size", opts.ExcludeSize, "排除的响应体大小（字节，逗号分隔）")
	flags.IntVar(&opts.RateLimit, "rate-limit", opts.RateLimit, "每秒最大请求数（0=不限）")
	flags.IntVar(&opts.Delay, "delay", opts.Delay, "每次请求间隔（毫秒）")
	flags.IntVar(&opts.MaxTime, "max-time", opts.MaxTime, "整个扫描的最长运行时间（秒，0=不限）")
	flags.IntVar(&opts.TargetMaxTime, "target-max-time", opts.TargetMaxTime, "单个目标的最长扫描时间（秒，0=不限）")
	flags.StringVar(&headersFile, "headers-file", "", "从文件加载请求头，每行一条 \"Key: Value\"")
	flags.BoolVar(&opts.RandomAgent, "random-agent", opts.RandomAgent, "使用内置 User-Agent 池随机轮换")
	flags.StringVar(&opts.Auth, "auth", opts.Auth, "Basic Auth 凭据，格式 user:pass")
	flags.StringVar(&opts.UserAgent, "user-agent", opts.UserAgent, "自定义 User-Agent（优先级低于 --random-agent）")
	flags.StringVar(&opts.IP, "ip", opts.IP, "绑定本地出口 IP")
	flags.StringVar(&opts.NetworkInterface, "network-interface", opts.NetworkInterface, "绑定网络接口")
	flags.StringVarP(&opts.TargetsFile, "targets-file", "l", "", "批量目标文件，每行一个 URL（与位置参数/--target 二选一）")

	return cmd
}

// resolveDirTargets 根据 opts.Target / opts.TargetsFile 解析出待扫描的目标列表。
// Validate() 已保证两者至少一个非空，此处不再重复校验。
func resolveDirTargets(opts *options.DirScanOptions) ([]string, error) {
	if opts.TargetsFile == "" {
		return []string{opts.Target}, nil
	}
	targets, err := loadList(opts.TargetsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load targets file: %w", err)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("targets file %q contains no valid target", opts.TargetsFile)
	}
	return targets, nil
}
