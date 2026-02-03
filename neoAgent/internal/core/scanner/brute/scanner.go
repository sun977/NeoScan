package brute

import (
	"context"
	"fmt"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/pkg/utils"
)

// BruteResult 爆破结果
type BruteResult struct {
	Service  string `json:"service"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Success  bool   `json:"success"`
}

// Headers 实现 TabularData 接口
func (r BruteResult) Headers() []string {
	return []string{"Service", "Host", "Port", "Username", "Password"}
}

// Rows 实现 TabularData 接口
func (r BruteResult) Rows() [][]string {
	return [][]string{{
		r.Service,
		r.Host,
		utils.IntToString(r.Port),
		r.Username,
		r.Password,
	}}
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
			Result:      []BruteResult{}, // 空结果
		}}, nil
	}

	// 4. 解析目标端口
	// Task.Target 是 IP
	// Task.PortRange 可能是单个端口
	port := utils.StringToInt(task.PortRange, 0)
	if port == 0 {
		return nil, fmt.Errorf("invalid port: %s", task.PortRange)
	}

	// 5. 执行爆破 (单机串行)
	// 获取全局并发令牌 (控制同时爆破的目标数)
	if err := s.globalLimit.Acquire(ctx); err != nil {
		return nil, err // Context Cancelled
	}
	defer s.globalLimit.Release()

	var results []BruteResult
	stopOnSuccess := true // 默认找到一个就停止
	if v, ok := task.Params["stop_on_success"].(bool); ok {
		stopOnSuccess = v
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

		if success {
			// 爆破成功
			res := BruteResult{
				Service:  serviceName,
				Host:     task.Target,
				Port:     port,
				Username: auth.Username,
				Password: auth.Password,
				Success:  true,
			}
			results = append(results, res)

			// QoS 反馈: 成功也算是一种"网络通畅"的信号，可以适当增加并发
			// 但爆破成功主要取决于弱口令是否存在，而不是网络质量
			// 这里为了保持 Limiter 活跃，可以调用 OnSuccess
			s.globalLimit.OnSuccess()

			if stopOnSuccess {
				break
			}
		}

		if err != nil {
			// 错误处理
			if err == ErrConnectionFailed {
				// 网络错误 (超时/拒绝) -> 降低并发
				s.globalLimit.OnFailure()

				// 可选: 连续网络错误直接跳过该主机
				// 这里简单处理：记录日志或继续
			} else {
				// 协议错误或认证失败 (ErrAuthFailed 通常为 nil error + false result，但也可能显式返回)
				// 视为正常交互 -> 保持并发
				s.globalLimit.OnSuccess()
			}
		} else {
			// err == nil && !success (认证失败)
			s.globalLimit.OnSuccess()
		}

		// 微小延迟，避免 CPU 100% 或触发极其敏感的防火墙
		time.Sleep(10 * time.Millisecond)
	}

	return []*model.TaskResult{{
		TaskID:      task.ID,
		Status:      model.TaskStatusSuccess,
		ExecutedAt:  startTime,
		CompletedAt: time.Now(),
		Result:      results,
	}}, nil
}
