package scan

import (
	"context"
	"fmt"

	"os"
	"strings"

	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/runner"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// loadList 解析用户输入，支持文件路径或逗号分隔的字符串
func loadList(input string) ([]string, error) {
	if input == "" {
		return nil, nil
	}

	// 1. 尝试作为文件读取
	// 判断文件是否存在且不是目录
	info, err := os.Stat(input)
	if err == nil && !info.IsDir() {
		content, err := os.ReadFile(input)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", input, err)
		}
		// 按行分割，支持 \r\n 和 \n
		lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
		var result []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				result = append(result, line)
			}
		}
		return result, nil
	}

	// 2. 作为逗号分隔字符串处理
	parts := strings.Split(input, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result, nil
}

// NewBruteScanCmd 创建 brute 子命令
func NewBruteScanCmd() *cobra.Command {
	opts := options.NewBruteScanOptions()
	var (
		users     string
		passwords string
	)

	cmd := &cobra.Command{
		Use:   "brute",
		Short: "执行弱口令爆破 (SSH/MySQL/Redis/Postgres/FTP/MongoDB/ClickHouse/SMB/MSSQL/Oracle/OracleSID/Telnet)",
		Long: `针对指定服务执行弱口令爆破。
支持的服务: ssh, mysql, redis, postgres, ftp, mongo, clickhouse, smb, mssql, oracle, oracle-sid, telnet, elasticsearch, snmp.
内置了 Top100 弱口令字典，支持通过参数自定义用户名和密码列表。
`,
		Example: `  # 爆破 SSH (使用内置字典)
  neoagent scan brute -t 192.168.1.1 -p 22 -s ssh

  # 爆破 MySQL (自定义字典)
  neoagent scan brute -t 192.168.1.1 -p 3306 -s mysql --users root --pass "123456,root,admin"

  # 爆破 Redis (无需用户名)
  neoagent scan brute -t 192.168.1.1 -p 6379 -s redis --pass "123456,password"

  # 爆破 PostgreSQL
  neoagent scan brute -t 192.168.1.1 -p 5432 -s postgres
  
  # 爆破 FTP
  neoagent scan brute -t 192.168.1.1 -p 21 -s ftp

  # 爆破 MongoDB
  neoagent scan brute -t 192.168.1.1 -p 27017 -s mongo

  # 爆破 ClickHouse
  neoagent scan brute -t 192.168.1.1 -p 9000 -s clickhouse

  # 爆破 SMB
  neoagent scan brute -t 192.168.1.1 -p 445 -s smb

  # 爆破 MSSQL
  neoagent scan brute -t 192.168.1.1 -p 1433 -s mssql

  # 爆破 Oracle
  neoagent scan brute -t 192.168.1.1 -p 1521 -s oracle

  # 爆破 Oracle SID (将 SID 字典作为用户字典传入)
  neoagent scan brute -t 192.168.1.1 -p 1521 -s oracle-sid --users "orcl,XE,ORACLE"

  # 爆破 Telnet
  neoagent scan brute -t 192.168.1.1 -p 23 -s telnet

  # 爆破 Elasticsearch
  neoagent scan brute -t 192.168.1.1 -p 9200 -s elasticsearch

  # 爆破 SNMP
  neoagent scan brute -t 192.168.1.1 -p 161 -s snmp --pass public`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 解析 users/passwords（支持文件路径或逗号分隔字符串）
			if users != "" {
				userList, err := loadList(users)
				if err != nil {
					return fmt.Errorf("failed to load users: %w", err)
				}
				opts.Users = userList
			}
			if passwords != "" {
				passList, err := loadList(passwords)
				if err != nil {
					return fmt.Errorf("failed to load passwords: %w", err)
				}
				opts.Passwords = passList
			}

			// 2. 参数校验（含端口默认值推断）
			if err := opts.Validate(); err != nil {
				return err
			}
			if opts.Port != "" {
				fmt.Printf("[*] Using port %s for service %s\n", opts.Port, opts.Service)
			}

			// 3. 构造 Task 并执行
			task := opts.ToTask()

			// 初始化 RunnerManager（工厂内已完成全部原子扫描器的统一注册，
			// CLI 与 Master 调度共用同一份注册表，避免能力不一致）
			manager := runner.NewRunnerManager()

			pterm.Info.Printf("Starting brute force task: %s:%s (%s)...\n", opts.Target, opts.Port, opts.Service)

			results, err := manager.Execute(context.Background(), task)
			if err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}

			// 4. 输出结果
			// 使用 ConsoleReporter 统一输出
			rep := reporter.NewConsoleReporter()
			rep.PrintResults(results)

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", "", "目标 IP (e.g. 192.168.1.1)")
	flags.StringVarP(&opts.Port, "port", "p", "", "目标端口 (e.g. 22)")
	flags.StringVarP(&opts.Service, "service", "s", "", "服务名称 (ssh/mysql/redis)")
	flags.StringVarP(&users, "users", "u", "", "自定义用户名列表 (逗号分隔)")
	flags.StringVar(&passwords, "pass", "", "自定义密码列表 (逗号分隔)")
	flags.BoolVarP(&opts.ScanAll, "all", "a", false, "尝试所有凭据 (默认: 找到一个成功后即停止)")

	return cmd
}
