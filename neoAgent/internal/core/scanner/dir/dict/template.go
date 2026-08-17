package dict

import "strings"

// ExtensionMode 控制扩展名展开行为。
type ExtensionMode int

const (
	// ExtensionModeClassic 默认模式：仅替换路径中的 %EXT% 占位符。
	// 不含 %EXT% 的行原样返回（不追加扩展名变体）。
	ExtensionModeClassic ExtensionMode = iota

	// ExtensionModeForce 强制模式：对不含扩展名的路径，额外追加 {path}.{ext} 变体。
	// 含 %EXT% 的路径照常替换。
	ExtensionModeForce
)

const extToken = "%EXT%"

// ExpandLine 将单条字典路径展开为一组路径。
//
//   - Classic 模式：若行含 %EXT%，用 extensions 替换后返回多行；若不含 %EXT% 原样返回。
//   - Force 模式：在 Classic 行为基础上，对不含 %EXT% 的行额外追加 "{line}.{ext}" 变体。
//   - 若 extensions 为空，%EXT% 行原样返回（不展开）。
func ExpandLine(line string, extensions []string, mode ExtensionMode) []string {
	hasToken := strings.Contains(line, extToken)

	// extensions 为空时，无论什么模式都原样返回
	if len(extensions) == 0 {
		return []string{line}
	}

	if hasToken {
		// 替换 %EXT% 占位符
		result := make([]string, 0, len(extensions))
		for _, ext := range extensions {
			expanded := strings.ReplaceAll(line, extToken, ext)
			result = append(result, expanded)
		}
		return result
	}

	// 不含 %EXT%
	if mode == ExtensionModeForce {
		// 原路径保留，再追加 .ext 变体
		result := make([]string, 0, 1+len(extensions))
		result = append(result, line)
		for _, ext := range extensions {
			result = append(result, line+"."+ext)
		}
		return result
	}

	// Classic 模式：不含 %EXT% 原样返回
	return []string{line}
}
