package brute

import (
	"context"
	"fmt"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/pkg/utils"

	"github.com/pterm/pterm"
)

// BruteResults 结果集合，用于实现 TabularData 接口以便一次性打印所有结果
type BruteResults []model.BruteResult

// Headers 实现 TabularData 接口
func (rs BruteResults) Headers() []string {
	return []string{"Service", "Host", "Port", "Username", "Password"}
}

// Rows 实现 TabularData 接口
func (rs BruteResults) Rows() [][]string {
	var rows [][]string
	for _, r := range rs {
		rows = append(rows, r.Rows()...)
	}
	return rows
}

// BruteScanner 爆破扫描器
type BruteScanner struct {
	limiters    map[string]*qos.AdaptiveLimiter // 按协议/目标分组的限流器 (可选优化)
	globalLimit *qos.AdaptiveLimiter            // 全局限流器
	dict        *DictManager                    // 字典管理器
	crackers    map[string]Cracker              // 注册的协议爆破器
	mu          sync.RWMutex
}

// NewBruteScanner 创建扫描器
func NewBruteScanner() *BruteScanner {
	// 默认全局并发: 初始 50, 最小 10, 最大 200
	// 爆破虽然是 TCP 连接，但每个 Goroutine 都会长时间占用连接，所以并发不宜过高
	// 相比 PortScan (可以 1000+)，BruteScan 更像 ServiceScan
	return &BruteScanner{
		globalLimit: qos.NewAdaptiveLimiter(50, 10, 200),
		dict:        NewDictManager(),
		crackers:    make(map[string]Cracker),
	}
}

// RegisterCracker 注册协议爆破器
func (s *BruteScanner) RegisterCracker(c Cracker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.crackers[c.Name()] = c
}

// Name 扫描器名称
func (s *BruteScanner) Name() model.TaskType {
	return model.TaskTypeBrute
}

// Run 执行扫描任务
// 注意: Task 应该是针对单个 Service 的爆破任务，或者是包含 service 参数的任务
// 我们假设 Runner 已经将任务分发为 (IP, Port, Service) 的粒度，或者 Task Params 里指定了 service
func (s *BruteScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	startTime := time.Now()

	// 1. 解析参数
	serviceName, ok := task.Params["service"].(string)
	if !ok || serviceName == "" {
		// 尝试从 TaskType 或其他地方推断，或者报错
		// 如果是 TaskTypeBrute，必须有 service 参数
		return nil, fmt.Errorf("missing 'service' parameter for brute scan")
	}

	// 2. 获取 Cracker
	s.mu.RLock()
	cracker, exists := s.crackers[serviceName]
	s.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unsupported service: %s", serviceName)
	}

	// 3. 生成字典
	// 支持从 Task Params 覆盖
	authList := s.dict.Generate(task.Params, cracker.Mode())
	if len(authList) == 0 {
		return []*model.TaskResult{{
			TaskID:      task.ID,
			Status:      model.TaskStatusSuccess,
			ExecutedAt:  startTime,
			CompletedAt: time.Now(),
			Result:      BruteResults{}, // 空结果
		}}, nil
	}

	// 4. 解析目标端口
	// Task.Target 是 IP
	// Task.PortRange 支持逗号分隔的多个端口 (e.g. "22,2222")
	ports := utils.ParseIntList(task.PortRange)
	if len(ports) == 0 {
		return nil, fmt.Errorf("invalid port: %s", task.PortRange)
	}

	// 5. 执行爆破 (单机串行)
	// 获取全局并发令牌 (控制同时爆破的目标数)
	if err := s.globalLimit.Acquire(ctx); err != nil {
		return nil, err // Context Cancelled
	}
	defer s.globalLimit.Release()

	var results []model.BruteResult
	stopOnSuccess := true // 默认找到一个就停止
	if v, ok := task.Params["stop_on_success"].(bool); ok {
		stopOnSuccess = v
	}

	// 双重循环：外层端口，内层字典
	// 理由：TCP连接复用通常针对单端口。切换端口意味着重建连接。
	// 而且逻辑上我们是逐个攻破服务。
	for _, port := range ports {
		// 快速失败检查
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 串行遍历字典
		for _, auth := range authList {
			// 快速失败检查
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			// 单次尝试超时控制 (3s)
			// 这里的超时是针对单次 Login 尝试
			checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)

			success, err := cracker.Check(checkCtx, task.Target, port, auth)
			cancel()

			// 构造日志后缀
			statusSuffix := "Failed"
			if success {
				statusSuffix = "Success"
			} else if err != nil {
				// 如果有错误，可能是网络错误，也可能是协议错误
				statusSuffix = fmt.Sprintf("Error: %v", err)
			}

			// 打印日志
			// 只有在 Debug 模式下才打印每一次尝试，避免刷屏
			if pterm.PrintDebugMessages {
				pterm.Debug.Printf("[Brute] Trying %s:%d | User: %s | Pass: %s | %s\n",
					task.Target, port, auth.Username, auth.Password, statusSuffix)
				pterm.Debug.Printf("[Brute-Detail] Context=%v Result=%v Err=%v\n", ctx, success, err)
			}

			if success {
				// 爆破成功，立即高亮显示
				pterm.Success.Printf("[Brute] FOUND %s:%d | User: %s | Pass: %s\n",
					task.Target, port, auth.Username, auth.Password)

				// 爆破成功
				res := model.BruteResult{
					Service:  serviceName,
					Host:     task.Target,
					Port:     port,
					Username: auth.Username,
					Password: auth.Password,
					Success:  true,
				}
				results = append(results, res)

				if stopOnSuccess {
					goto FINISH
				}
			}
		}
	}

FINISH:
	// 6. 返回结果
	return []*model.TaskResult{{
		TaskID:      task.ID,
		Status:      model.TaskStatusSuccess,
		ExecutedAt:  startTime,
		CompletedAt: time.Now(),
		Result:      BruteResults(results),
	}}, nil
}
