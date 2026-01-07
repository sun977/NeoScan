package etl

import (
	"context"
	"encoding/json"
	"fmt"

	assetModel "neomaster/internal/model/asset"
	"neomaster/internal/service/fingerprint"
)

// enrichWithFingerprint 使用指纹服务对 AssetBundle 进行增强 (Enrichment)
// 这是一个可选的 ETL 步骤，用于：
// 1. 对没有指纹信息但有 Banner 的服务进行补充识别
// 2. 对 Web 站点进行深度指纹分析
func enrichWithFingerprint(ctx context.Context, bundle *AssetBundle, fpService fingerprint.Service) error {
	if bundle == nil {
		return nil
	}

	// 1. Enrich Services (PortScan Result -> Service Fingerprint)
	// 场景：Nmap 快速扫描只返回了 Banner，但没有进行 CPE 识别
	for _, svc := range bundle.Services {
		// 如果已有明确的产品标识，且没有强制覆盖标志，则跳过
		if svc.Product != "" {
			continue
		}
		// 如果没有 Banner，无法识别，跳过
		if svc.Banner == "" {
			continue
		}

		input := &fingerprint.Input{
			Target:   bundle.Host.IP,
			Port:     svc.Port,
			Protocol: svc.Proto, // 注意：AssetService 字段是 Proto，但 Input 是 Protocol (需确认)
			Banner:   svc.Banner,
		}
		// AssetService.Proto 通常是 "tcp", Input.Protocol 也是
		input.Protocol = svc.Proto

		res, err := fpService.Identify(ctx, input)
		if err != nil {
			// Log error but continue
			continue
		}
		if res.Best != nil {
			svc.Product = res.Best.Product
			svc.Version = res.Best.Version
			svc.CPE = res.Best.CPE
			// Vendor 字段 AssetService 暂时没有，忽略
		}
	}

	// 2. Enrich Webs (Web Detail -> Web Fingerprint)
	// 场景：Web 爬虫抓取了 Headers/Body，但未进行 Wappalyzer 识别
	// 需要建立 URL -> AssetWeb 的映射，因为 WebDetails 和 Webs 是分离的
	webMap := make(map[string]*assetModel.AssetWeb)
	for _, w := range bundle.Webs {
		webMap[w.URL] = w
	}

	for _, d := range bundle.WebDetails {
		// 解析 ContentDetails 提取 URL, Headers, Body
		var cd map[string]interface{}
		if err := json.Unmarshal([]byte(d.ContentDetails), &cd); err != nil {
			continue
		}
		urlStr, ok := cd["url"].(string)
		if !ok {
			continue
		}

		web, exists := webMap[urlStr]
		if !exists {
			continue
		}

		// 构建指纹输入
		headersMap := make(map[string]string)
		if h, ok := cd["response_headers"].(map[string]interface{}); ok {
			for k, v := range h {
				headersMap[k] = fmt.Sprint(v)
			}
		}
		
		// 尝试获取 Body (如果 ContentDetails 包含)
		bodyStr := ""
		if b, ok := cd["body"].(string); ok {
			bodyStr = b
		}

		input := &fingerprint.Input{
			Target:  web.Domain, 
			Headers: headersMap,
			Body:    bodyStr,
		}
		// 端口解析
		// 如果 URL 包含端口，可以在这里解析，但在 Web 指纹识别中通常 Headers/Body 更重要

		res, err := fpService.Identify(ctx, input)
		if err != nil {
			continue
		}

		if len(res.Matches) > 0 {
			updateWebTechStack(web, res.Matches)
		}
	}

	return nil
}

// updateWebTechStack 更新 AssetWeb 的 TechStack 字段
func updateWebTechStack(web *assetModel.AssetWeb, matches []fingerprint.Match) {
	var currentStack []string
	
	// 尝试解析现有的 TechStack
	if web.TechStack != "" && web.TechStack != "{}" {
		// 可能是 JSON 数组字符串，也可能是其他格式，需容错
		if err := json.Unmarshal([]byte(web.TechStack), &currentStack); err != nil {
			// 如果解析失败，假设是空
			currentStack = []string{}
		}
	}

	// 合并新发现的组件
	seen := make(map[string]bool)
	for _, s := range currentStack {
		seen[s] = true
	}

	changed := false
	for _, m := range matches {
		if !seen[m.Product] {
			currentStack = append(currentStack, m.Product)
			seen[m.Product] = true
			changed = true
		}
	}

	if changed {
		newStackJSON, _ := json.Marshal(currentStack)
		web.TechStack = string(newStackJSON)
	}
}
