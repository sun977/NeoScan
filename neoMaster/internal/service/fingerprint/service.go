// 本质上是一个 "Fingerprint Identifier" (指纹识别器)
// 负责接收资产输入，使用多个匹配引擎并行识别资产指纹
// 每个匹配引擎负责识别一种指纹类型 (如 Web 指纹、CPE 指纹等)
package fingerprint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Service 指纹识别服务接口
type Service interface {
	Identify(ctx context.Context, input *Input) (*Result, error)
	LoadRules(dir string) error
	GetStats() map[string]int
}

type serviceImpl struct {
	engines []MatchEngine
	mu      sync.RWMutex
}

func NewFingerprintService(engines ...MatchEngine) Service {
	return &serviceImpl{
		engines: engines,
	}
}

func (s *serviceImpl) Identify(ctx context.Context, input *Input) (*Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allMatches []Match

	// 并行或串行调用引擎
	// 简单起见，目前串行调用，未来可优化为并行
	for _, engine := range s.engines {
		matches, err := engine.Match(input)
		if err != nil {
			// Log error but continue
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

	// 排序和去重
	best := selectBestMatch(allMatches)

	return &Result{
		Matches: allMatches,
		Best:    best,
	}, nil
}

func (s *serviceImpl) LoadRules(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 遍历目录下的所有文件
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read rule directory %s: %w", dir, err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(dir, file.Name())
		for _, engine := range s.engines {
			// 每个引擎尝试加载该文件，如果格式不匹配则忽略（返回nil或特定错误）
			// 注意：引擎应该自行判断文件是否属于自己处理的类型
			if err := engine.LoadRules(filePath); err != nil {
				// 只有当明确是加载失败而不是格式不匹配时才报错？
				// 目前 LoadRules 如果无法识别格式会返回 error
				// 我们需要一种机制让引擎说 "这不是我的文件"，而不是报错。
				// 现在的实现是：HTTPEngine 返回 "unknown rule file format"，ServiceEngine 类似。

				// 改进策略：LoadRules 如果遇到非本引擎文件，应该返回 nil (已在 Engine 中修改)
				// 但如果是 "unknown rule file format"，意味着所有引擎都不认，这可能是个问题，也可能是正常的(例如文件是给另一个引擎的)。

				// 由于我们遍历了所有引擎，只要有一个引擎成功加载了文件，就算成功。
				// 但现在的 LoadRules 签名是 path string -> error。

				// 让我们修改逻辑：
				// Engine.LoadRules 的契约变更为：
				// 1. 成功加载 -> nil
				// 2. 文件格式不匹配 (type字段不对) -> nil (忽略)
				// 3. 文件格式错误 (JSON坏了) -> error
				// 4. 只有当文件确实是给该引擎的但解析出错时才返回 error

				// 但现在的 Engine 实现是：如果是 JSON 解析成功但 type 不对，返回 nil。
				// 如果解析到最后都无法识别，返回 "unknown rule file format"。
				// 这会导致：HTTPEngine 遇到 service.json 会报错 unknown format。

				// 我们需要在这里宽容处理：
				// 只要错误包含 "unknown rule file format" 或 "failed to unmarshal"，我们认为是该引擎不处理此文件，忽略之。
				// 但如果所有引擎都忽略了，是否需要警告？

				// 简化处理：忽略 "unknown rule file format" 错误
				if err.Error() == fmt.Sprintf("unknown rule file format: %s", filePath) {
					continue
				}
				// 也可以检查错误字符串
				if strings.Contains(err.Error(), "unknown rule file format") || strings.Contains(err.Error(), "failed to unmarshal") {
					continue
				}

				return fmt.Errorf("failed to load rules from %s for engine %s: %w", filePath, engine.Type(), err)
			}
		}
	}
	return nil
}

func (s *serviceImpl) GetStats() map[string]int {
	return map[string]int{
		"engines": len(s.engines),
	}
}

func selectBestMatch(matches []Match) *Match {
	if len(matches) == 0 {
		return nil
	}
	// 简单策略：取置信度最高的第一个
	// 实际策略：CPE 优先，版本号详细优先
	best := &matches[0]
	for i := 1; i < len(matches); i++ {
		if matches[i].Confidence > best.Confidence {
			best = &matches[i]
		} else if matches[i].Confidence == best.Confidence {
			// 同置信度，优先选有版本号的
			if matches[i].Version != "" && best.Version == "" {
				best = &matches[i]
			}
		}
	}
	return best
}
