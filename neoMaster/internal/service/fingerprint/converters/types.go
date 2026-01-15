// 定义了规则转换器的通用接口
// 用于第三方规则格式转换进入 NeoScan 内部标准模型
package converters

import "neomaster/internal/model/asset"

// RuleConverter 定义了规则转换器的通用接口
// 所有的第三方规则格式转换器都必须实现此接口
type RuleConverter interface {
	// Decode 将特定格式的字节流转换为 NeoScan 内部标准模型
	// 返回: 指纹列表, CPE列表, 错误
	Decode(data []byte) ([]*asset.AssetFinger, []*asset.AssetCPE, error)

	// Encode 将 NeoScan 内部标准模型转换为特定格式 (可选，主要用于 StandardJSON)
	// 对于只读的第三方格式 (如 Goby)，可以返回错误或空
	Encode(fingers []*asset.AssetFinger, cpes []*asset.AssetCPE) ([]byte, error)
}

// ConverterType 定义转换器类型
type ConverterType string

const (
	TypeStandard ConverterType = "standard"
	TypeGoby     ConverterType = "goby"
	TypeEHole    ConverterType = "ehole"
	// 后续可扩展 TypeWappalyzer, TypeFingers 等
)
