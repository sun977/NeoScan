package os

import (
	"context"
	"fmt"
	"sync"

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
}

func NewScanner() *Scanner {
	s := &Scanner{
		engines: make(map[string]OsScanEngine),
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
		// 用户反馈: "deep的时候source是TTL，应该不对，deep是强制指定nmap逻辑探测"
		if e, ok := s.engines["nmap_stack"]; ok {
			enginesToRun = append(enginesToRun, e)
		}
	case "auto":
		// Auto 模式：并发运行所有可用引擎，通过竞速机制选择置信度最高的结果
		// 包含: TTL Engine (Fast, ~80%), Nmap Stack Engine (Deep, Precision), Nmap Service Engine (If available)
		for _, e := range s.engines {
			enginesToRun = append(enginesToRun, e)
		}
	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}

	for _, engine := range enginesToRun {
		wg.Add(1)
		go func(e OsScanEngine) {
			defer wg.Done()
			info, err := e.Scan(ctx, target)
			if err != nil {
				logger.Error(err)
				return
			}
			if info == nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			// 简单的择优逻辑：置信度高的优先
			if bestResult == nil || info.Accuracy > bestResult.Accuracy {
				bestResult = info
			}
		}(engine)
	}

	wg.Wait()

	if bestResult == nil {
		return &model.OsInfo{Name: "Unknown", Accuracy: 0}, nil
	}
	return bestResult, nil
}
