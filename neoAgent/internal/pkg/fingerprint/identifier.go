package fingerprint

import (
	"context"
	"fmt"
	"sync"

	"neoagent/internal/pkg/fingerprint/model"
)

// Service 指纹识别服务接口
type Service interface {
	Identify(ctx context.Context, input *Input) (*Result, error)
	LoadRules(fingerRules []model.FingerRule, cpeRules []model.CPERule)
	GetStats() map[string]int
}

// HTTPEngineReloader 定义 HTTP 引擎重载接口
type HTTPEngineReloader interface {
	Reload(rules []model.FingerRule)
}

// ServiceEngineReloader 定义 Service 引擎重载接口
type ServiceEngineReloader interface {
	Reload(rules []model.CPERule)
}

// serviceImpl 指纹识别服务实现
type serviceImpl struct {
	engines []MatchEngine
	mu      sync.RWMutex
}

// NewFingerprintService 创建指纹识别服务实例
func NewFingerprintService(engines ...MatchEngine) Service {
	return &serviceImpl{
		engines: engines,
	}
}

// Identify 识别资产指纹
func (s *serviceImpl) Identify(ctx context.Context, input *Input) (*Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allMatches []Match

	for _, engine := range s.engines {
		matches, err := engine.Match(input)
		if err != nil {
			fmt.Printf("Engine %s error: %v\n", engine.Type(), err)
			continue
		}
		if len(matches) > 0 {
			allMatches = append(allMatches, matches...)
		}
	}

	if len(allMatches) == 0 {
		return nil, nil
	}

	best := selectBestMatch(allMatches)

	return &Result{
		Matches: allMatches,
		Best:    best,
	}, nil
}

// LoadRules 加载规则
func (s *serviceImpl) LoadRules(fingerRules []model.FingerRule, cpeRules []model.CPERule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, engine := range s.engines {
		switch e := engine.(type) {
		case HTTPEngineReloader:
			if len(fingerRules) > 0 {
				e.Reload(fingerRules)
			}
		case ServiceEngineReloader:
			if len(cpeRules) > 0 {
				e.Reload(cpeRules)
			}
		}
	}
}

// GetStats 获取统计信息
func (s *serviceImpl) GetStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]int{
		"engines": len(s.engines),
	}
}

// selectBestMatch 选择最佳匹配
func selectBestMatch(matches []Match) *Match {
	if len(matches) == 0 {
		return nil
	}
	best := &matches[0]
	for i := 1; i < len(matches); i++ {
		if matches[i].Confidence > best.Confidence {
			best = &matches[i]
		} else if matches[i].Confidence == best.Confidence {
			if matches[i].Version != "" && best.Version == "" {
				best = &matches[i]
			}
		}
	}
	return best
}
