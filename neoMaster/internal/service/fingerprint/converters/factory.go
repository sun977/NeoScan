package converters

import "fmt"

// Factory 转换器工厂
type Factory struct {
	converters map[ConverterType]RuleConverter
}

// NewFactory 创建工厂实例并注册所有支持的转换器
func NewFactory() *Factory {
	f := &Factory{
		converters: make(map[ConverterType]RuleConverter),
	}
	f.registerDefaults()
	return f
}

// registerDefaults 注册默认转换器
// 1. StandardJSON 格式 (默认 NeoScan 内部格式)
// 2. Goby 格式
// 3. EHole 格式 (待实现)
// 4. 其他格式 (如 Nmap, Nessus) (待实现)
func (f *Factory) registerDefaults() {
	f.Register(TypeStandard, NewStandardJSONConverter())
	f.Register(TypeGoby, NewGobyConverter())
	// f.Register(TypeEHole, NewEHoleConverter()) // 待实现
}

// Register 注册新的转换器
func (f *Factory) Register(t ConverterType, c RuleConverter) {
	f.converters[t] = c
}

// Get 获取指定类型的转换器
func (f *Factory) Get(t ConverterType) (RuleConverter, error) {
	c, ok := f.converters[t]
	if !ok {
		return nil, fmt.Errorf("unsupported rule format: %s", t)
	}
	return c, nil
}
