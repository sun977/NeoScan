/**
 * @author: Sun977
 * @date: 2025.12.03
 * @description: 临时放在这里，后续漏洞管理专门开辟位置存放
 * @func:
 * 包含：
 */
package asset

import (
	"fmt"
	"neomaster/internal/model/asset"
	"strings"
)

// SetNucleiPoc 辅助方法：设置 Nuclei 类型的 PoC
// 自动处理内容格式，如果是完整 YAML 则存入 Content，如果是路径则存入 Content 并标记
func SetNucleiPoc(p *asset.AssetVulnPoc, templateContent string, templateID string) {
	p.PocType = "yaml"

	// 简单的启发式判断：如果包含 "id: " 和 "info:"，则认为是 YAML 内容
	if strings.Contains(templateContent, "id:") && strings.Contains(templateContent, "info:") {
		p.Content = templateContent
		// 尝试从 YAML 中提取描述 (这里只是简单的字符串处理，实际项目可以使用 yaml 库)
		p.Description = fmt.Sprintf("Nuclei Template: %s", templateID)
	} else {
		// 假设是路径或 ID
		p.Content = templateContent // e.g., "cves/2021/CVE-2021-1234.yaml"
		p.Description = fmt.Sprintf("Reference to Nuclei Template: %s", templateID)
	}

	if p.Source == "" {
		p.Source = "nuclei-templates"
	}
}
