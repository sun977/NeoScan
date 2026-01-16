package registry

import (
	"fmt"
	"sync"

	"neomaster/internal/pkg/tool_adapter/command"
	"neomaster/internal/pkg/tool_adapter/parser"
)

// Adapter 封装了一个工具的所有适配逻辑
type Adapter struct {
	Name    string
	Builder command.CommandBuilder
	Parser  parser.Parser
}

// ToolRegistry 工具注册中心 (单例模式)
type ToolRegistry struct {
	adapters map[string]*Adapter
	mu       sync.RWMutex
}

var (
	instance *ToolRegistry
	once     sync.Once
)

// GetRegistry 获取注册中心单例
func GetRegistry() *ToolRegistry {
	once.Do(func() {
		instance = &ToolRegistry{
			adapters: make(map[string]*Adapter),
		}
	})
	return instance
}

// Register 注册一个工具适配器
// 如果 builder 或 parser 为 nil，则对应功能不可用，但在运行时检查
func (r *ToolRegistry) Register(name string, builder command.CommandBuilder, parser parser.Parser) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.adapters[name] = &Adapter{
		Name:    name,
		Builder: builder,
		Parser:  parser,
	}
}

// GetBuilder 获取指定工具的命令构建器
func (r *ToolRegistry) GetBuilder(name string) (command.CommandBuilder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[name]
	if !exists {
		return nil, fmt.Errorf("tool adapter not found: %s", name)
	}
	if adapter.Builder == nil {
		return nil, fmt.Errorf("command builder not implemented for tool: %s", name)
	}
	return adapter.Builder, nil
}

// GetParser 获取指定工具的结果解析器
func (r *ToolRegistry) GetParser(name string) (parser.Parser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[name]
	if !exists {
		return nil, fmt.Errorf("tool adapter not found: %s", name)
	}
	if adapter.Parser == nil {
		return nil, fmt.Errorf("result parser not implemented for tool: %s", name)
	}
	return adapter.Parser, nil
}

// RegisterAdapter 快捷注册函数 (Global)
func RegisterAdapter(name string, builder command.CommandBuilder, parser parser.Parser) {
	GetRegistry().Register(name, builder, parser)
}
