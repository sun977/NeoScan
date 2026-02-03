package os

import (
	"context"
	"fmt"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/os/nmap_stack"
	"neoagent/internal/pkg/logger"
)

// OsScanEngine 定义 OS 扫描引擎接口
type OsScanEngine interface {
	Name() string
	Scan(ctx context.Context, target string) (*model.OsInfo, error)
}

// Scanner OS 扫描器主控
type Scanner struct {
	engines map[string]OsScanEngine
	// OS Scanner 是针对单个 Target 运行的 (One Task One Target)
	// 但如果是批量扫描，我们需要 QoS。
	// 然而，目前的 OS Scanner 架构是：Run(Task) -> Scan(Target)
	// 它的 Run 方法是在 scan_os.go (Options层) 还是 adapter_os.go (Runner层)？
	// 让我们看一眼 adapter_os.go，这里没有显示。
	// 但根据接口，OSScanner 应该实现 Scanner 接口的 Run 方法。
	// 这里的 Scanner struct 并没有 Run 方法，只有 Scan 方法。
	// 看来这个 Scanner 是一个 "Service"，被 adapter_os.go 包装了？
	// 或者它只是一个 helper？
	// 
	// 无论如何，OS 识别通常针对单机，并发发生在 Host 层面 (由 Pipeline 控制)。
	// 如果 OSScanner.Scan 是阻塞的，并且内部没有并发（除了多引擎并行），
	// 那么 QoS 应该在 Pipeline 层或者调用 OSScanner 的地方控制。
	// 
	// 不过，如果 OSScanner 内部使用了 Nmap Stack Engine，那个引擎会发包。
	// 我们可以在那里引入 QoS。
	// 
	// 鉴于 OSScanner 结构简单，我们在这里引入 QoS 主要是为了 RTO (超时控制)。
	// 至于并发，因为 Scan(ctx, target) 是针对单个目标的，并发由外部 Runner 控制。
	// 如果外部 Runner 没有限流，这里也限制不住（除非用全局单例 Limiter）。
	
	rttEstimator *qos.RttEstimator
}

func NewScanner() *Scanner {
	s := &Scanner{
		engines: make(map[string]OsScanEngine),
		rttEstimator: qos.NewRttEstimator(),
	}
	// 注册默认引擎
	s.Register(NewTTLEngine())
	s.Register(nmap_stack.NewNmapStackEngine())
	s.Register(NewNmapServiceEngine())
	return s
}

// Register 注册引擎
func (s *Scanner) Register(engine OsScanEngine) {
	s.engines[engine.Name()] = engine
}

// Scan 执行扫描
func (s *Scanner) Scan(ctx context.Context, target string, mode string) (*model.OsInfo, error) {
	var bestResult *model.OsInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 定义要运行的引擎
	var enginesToRun []OsScanEngine

	switch mode {
	case "fast":
		if e, ok := s.engines["ttl"]; ok {
			enginesToRun = append(enginesToRun, e)
		}
	case "deep":
		// Deep 模式强制使用 Nmap 逻辑探测
		if e, ok := s.engines["nmap_stack"]; ok {
			enginesToRun = append(enginesToRun, e)
		}
	case "auto":
		// Auto 模式：并发运行所有可用引擎
		for _, e := range s.engines {
			enginesToRun = append(enginesToRun, e)
		}
	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}

	// 使用动态超时
	timeout := s.rttEstimator.Timeout()
	// OS 探测需要较长时间
	scanTimeout := timeout * 4
	if scanTimeout < 2*time.Second {
		scanTimeout = 2 * time.Second
	}
	
	// 创建带超时的 Context
	scanCtx, cancel := context.WithTimeout(ctx, scanTimeout)
	defer cancel()

	start := time.Now()
	
	for _, engine := range enginesToRun {
		wg.Add(1)
		go func(e OsScanEngine) {
			defer wg.Done()
			
			// 这里我们无法精确测量 RTT，因为 Scan 接口返回的是 result。
			// 只有 NmapStackEngine 这种发包的才有 RTT 概念。
			// 简单的：如果成功了，就认为网络是通的。
			
			info, err := e.Scan(scanCtx, target)
			if err != nil {
				logger.Debugf("OS scan engine '%s' failed on %s: %v", e.Name(), target, err)
				return
			}
			if info == nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			if bestResult == nil || info.Accuracy > bestResult.Accuracy {
				bestResult = info
			}
		}(engine)
	}

	wg.Wait()

	// 简单的 QoS 反馈：如果有结果，说明网络通畅，更新 RTT (虽然不太准，但比没有强)
	// 注意：这里的 duration 包含了引擎处理时间，可能偏大。
	// 所以我们只在非常快返回的时候更新（例如 TTL 引擎）
	// 或者只更新 Limiter (如果有的话)
	// 这里的 RTT 更新需要谨慎，避免污染。暂不更新 RTT，只利用 RTT 来设置 Timeout。
	_ = time.Since(start)

	if bestResult == nil {
		return &model.OsInfo{Name: "Unknown", Accuracy: 0}, nil
	}
	return bestResult, nil
}
