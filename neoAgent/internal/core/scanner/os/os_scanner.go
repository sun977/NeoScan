package os

import (
	"context"
	"fmt"
	"sync"

	"neoagent/internal/pkg/logger"
)

// OsInfo 操作系统识别结果
type OsInfo struct {
	Name        string // OS名称 (Windows, Linux, etc.)
	Family      string // OS家族 (Windows, Unix, Cisco, etc.)
	Version     string // 版本号
	Accuracy    int    // 置信度 (0-100)
	Fingerprint string // 指纹字符串 (CPE or Raw)
	Source      string // 识别来源 (TTL, Service, Stack)
}

// OsScanEngine 定义 OS 扫描引擎接口
type OsScanEngine interface {
	Name() string
	Scan(ctx context.Context, target string) (*OsInfo, error)
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
	s.Register(NewNmapStackEngine())
	return s
}

// Register 注册引擎
func (s *Scanner) Register(engine OsScanEngine) {
	s.engines[engine.Name()] = engine
}

// Scan 执行扫描
func (s *Scanner) Scan(ctx context.Context, target string, mode string) (*OsInfo, error) {
	var bestResult *OsInfo
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
		// Deep 模式跑所有引擎，取最优
		for _, e := range s.engines {
			enginesToRun = append(enginesToRun, e)
		}
	case "auto":
		// Auto 模式默认跑 TTL，如果有 Service 结果也会利用 (这里简化为跑 TTL)
		// 实际生产中可能先跑 TTL，不准再跑 Stack
		if e, ok := s.engines["ttl"]; ok {
			enginesToRun = append(enginesToRun, e)
		}
		// TODO: 如果有 Nmap Service 引擎，也应该加入
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
		return &OsInfo{Name: "Unknown", Accuracy: 0}, nil
	}
	return bestResult, nil
}
