package scan

import (
	"context"
	"fmt"

	"neoagent/internal/core/model"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/runner"

	"github.com/spf13/cobra"
)

// NewBruteScanCmd 创建 brute 子命令
func NewBruteScanCmd() *cobra.Command {
	var (
		target        string
		portRange     string
		service       string
		users         string
		passwords     string
		stopOnSuccess bool
	)

	cmd := &cobra.Command{
		Use:   "brute",
		Short: "执行弱口令爆破 (SSH/MySQL/Redis)",
		Long: `针对指定服务执行弱口令爆破。
支持的服务: ssh, mysql, redis.
内置了 Top100 弱口令字典，支持通过参数自定义用户名和密码列表。
`,
		Example: `  # 爆破 SSH (使用内置字典)
  neoagent scan brute -t 192.168.1.1 -p 22 -s ssh

  # 爆破 MySQL (自定义字典)
  neoagent scan brute -t 192.168.1.1 -p 3306 -s mysql --users root --passwords "123456,root,admin"

  # 爆破 Redis (无需用户名)
  neoagent scan brute -t 192.168.1.1 -p 6379 -s redis --passwords "123456,password"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 参数校验
			if target == "" {
				return fmt.Errorf("target is required (-t)")
			}
			if service == "" {
				return fmt.Errorf("service is required (-s)")
			}
			if portRange == "" {
				// 尝试根据服务设置默认端口
				switch service {
				case "ssh":
					portRange = "22"
				case "mysql":
					portRange = "3306"
				case "redis":
					portRange = "6379"
				default:
					return fmt.Errorf("port is required (-p) for service %s", service)
				}
			}

			// 2. 构造 Task
			task := model.NewTask(model.TaskTypeBrute, target)
			task.PortRange = portRange
			task.Params["service"] = service
			task.Params["stop_on_success"] = stopOnSuccess

			if users != "" {
				task.Params["users"] = users
			}
			if passwords != "" {
				task.Params["passwords"] = passwords
			}

			// 3. 执行任务
			ctx := context.Background()

			// 初始化 RunnerManager
			manager := runner.NewRunnerManager()

			fmt.Printf("[*] Starting brute force on %s:%s (%s)...\n", target, portRange, service)

			results, err := manager.Execute(ctx, task)
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
	flags.StringVarP(&target, "target", "t", "", "目标 IP (e.g. 192.168.1.1)")
	flags.StringVarP(&portRange, "port", "p", "", "目标端口 (e.g. 22)")
	flags.StringVarP(&service, "service", "s", "", "服务名称 (ssh/mysql/redis)")
	flags.StringVar(&users, "users", "", "自定义用户名列表 (逗号分隔)")
	flags.StringVar(&passwords, "passwords", "", "自定义密码列表 (逗号分隔)")
	flags.BoolVar(&stopOnSuccess, "stop-on-success", true, "找到一个成功凭据后停止")

	return cmd
}
