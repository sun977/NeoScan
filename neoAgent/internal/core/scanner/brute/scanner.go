package brute

import (
	"context"
	"fmt"
	"time"

	"neoagent/internal/core/model"
	"neoagent/internal/pkg/logger"
)

const ScannerName = "brute_scanner"

// Cracker 定义特定协议的爆破接口
type Cracker interface {
	Name() string
	// Crack 尝试爆破
	// user, pass: 用户名和密码
	// 返回: success, error
	Crack(ctx context.Context, host string, port int, user, pass string) (bool, error)
}

// BruteScanner 弱口令爆破扫描器
type BruteScanner struct {
	crackers map[string]Cracker
}

func NewBruteScanner() *BruteScanner {
	s := &BruteScanner{
		crackers: make(map[string]Cracker),
	}
	// 注册 Crackers
	// s.Register(NewSSHCracker())
	return s
}

func (s *BruteScanner) Name() model.TaskType {
	return model.TaskTypeBruteScan
}

func (s *BruteScanner) Register(c Cracker) {
	s.crackers[c.Name()] = c
}

func (s *BruteScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	// 1. 解析参数
	// 需要: Target IP, Port, Protocol (Service Name), UserDict, PassDict
	// 注意: BruteScan 通常是 PortScan 的后续步骤。
	// 输入可能是一个已知的开放端口和服务。

	target := task.Target
	port := task.Port // 假设 Task 中有 Port 字段? model.Task 定义中没有 Port 字段，只有 PortRange。
	// 但如果是 Brute 任务，通常是针对单个 Service 的。
	// 我们暂且从 Params 中获取 port 和 service

	var service string
	var portInt int

	if p, ok := task.Params["port"]; ok {
		if v, ok := p.(float64); ok {
			portInt = int(v)
		} else if v, ok := p.(int); ok {
			portInt = v
		}
	}

	if s, ok := task.Params["service"]; ok {
		service = s.(string)
	}

	if portInt == 0 || service == "" {
		return nil, fmt.Errorf("port and service are required for brute scan")
	}

	// 获取对应的 Cracker
	cracker, ok := s.crackers[service]
	if !ok {
		// 没有对应的 Cracker，忽略或返回错误
		return nil, fmt.Errorf("no cracker found for service: %s", service)
	}

	// 准备字典
	// 这里简化：使用内置默认字典或 Params 传入的字典
	users := []string{"root", "admin"}
	passwords := []string{"123456", "password", "root", "admin"}

	if u, ok := task.Params["users"]; ok {
		if us, ok := u.([]string); ok {
			users = us
		}
	}
	if p, ok := task.Params["passwords"]; ok {
		if ps, ok := p.([]string); ok {
			passwords = ps
		}
	}

	var results []*model.TaskResult

	// 简单的双重循环爆破 (实际应使用 Goroutine Pool)
	// 但考虑到对单服务的连接限制，通常单服务的爆破是串行的或低并发的

	for _, user := range users {
		for _, pass := range passwords {
			// 检查 Context
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			// 执行爆破
			// 设置单次尝试超时
			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			success, err := cracker.Crack(timeoutCtx, target, portInt, user, pass)
			cancel()

			if err != nil {
				// 网络错误等，记录 debug
				logger.Debugf("Crack error %s:%d %s/%s: %v", target, portInt, user, pass, err)
				continue
			}

			if success {
				// 成功！
				res := &model.TaskResult{
					TaskID: task.ID,
					Status: model.TaskStatusSuccess,
					Result: &model.BruteResult{
						IP:       target,
						Port:     portInt,
						Service:  service,
						Username: user,
						Password: pass,
					},
					ExecutedAt:  time.Now(),
					CompletedAt: time.Now(),
				}
				results = append(results, res)

				// 爆破成功一个通常就够了？或者继续？
				// 默认停止
				return results, nil
			}
		}
	}

	return results, nil
}
