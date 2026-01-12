/**
 * Mapper 结构映射器
 * @author: sun977
 * @date: 2025.01.06
 * @description: 负责将 StageResult (标准 JSON 结构) 映射为内部 Asset 实体模型。
 * 它不解析原始工具输出(XML/Text)，只处理归一化后的数据。
 */
package etl

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	assetModel "neomaster/internal/model/asset"
	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
)

// AssetBundle 资产数据包
// 聚合了从 StageResult 中提取出的各类资产实体
// Processor 拿到此包后，根据字段是否非空调用对应的 Merger 逻辑
type AssetBundle struct {
	ProjectID uint64                     // 所属项目ID (来自 StageResult)
	Host      *assetModel.AssetHost      // 主机资产 (必选)
	Services  []*assetModel.AssetService // 关联的服务列表
	WebAssets []*WebAsset                // 关联的 Web 站点及其详情
	Vulns     []*assetModel.AssetVuln    // 关联的漏洞列表
}

type WebAsset struct {
	Web    *assetModel.AssetWeb       // Web 站点资产
	Detail *assetModel.AssetWebDetail // Web 站点详情
}

// MapToAssetBundles 将通用的 StageResult 映射为标准资产包列表
// 根据 ResultType 分发到具体的映射逻辑
func MapToAssetBundles(result *orcModel.StageResult) ([]*AssetBundle, error) {
	var bundles []*AssetBundle
	var singleBundle *AssetBundle
	var err error

	switch result.ResultType {
	case "ip_alive":
		return mapIPAlive(result)
	case "fast_port_scan", "full_port_scan":
		return mapPortScan(result)
	case "service_fingerprint":
		return mapServiceFingerprint(result)
	case "vuln_finding":
		return mapVulnFinding(result)
	case "poc_scan":
		return mapPocScan(result)
	case "web_endpoint":
		return mapWebEndpoint(result)
	case "password_audit":
		singleBundle, err = mapPasswordAudit(result)
	case "proxy_detection":
		singleBundle, err = mapProxyDetection(result)
	case "directory_scan":
		singleBundle, err = mapDirectoryScan(result)
	case "subdomain_discovery":
		singleBundle, err = mapSubdomainDiscovery(result)
	case "api_discovery":
		singleBundle, err = mapApiDiscovery(result)
	case "file_discovery":
		singleBundle, err = mapFileDiscovery(result)
	case "other_scan":
		singleBundle, err = mapOtherScan(result)
	default:
		return nil, fmt.Errorf("unsupported result type for mapping: %s", result.ResultType)
	}

	if err != nil {
		return nil, err
	}
	if singleBundle != nil {
		singleBundle.ProjectID = result.ProjectID
		bundles = append(bundles, singleBundle)
	}
	return bundles, nil
}

// mapIPAlive 映射探活结果
// 仅仅发现存货主机，不做关联service/web/vuln等的动作
func mapIPAlive(result *orcModel.StageResult) ([]*AssetBundle, error) {
	var attr IPAliveAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	if len(attr.Hosts) == 0 {
		return nil, nil
	}

	var bundles []*AssetBundle

	for _, h := range attr.Hosts {
		// 构造 AssetHost
		host := &assetModel.AssetHost{
			IP:             h.IP,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}

		// 构造 Bundle
		bundle := &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
		}
		bundles = append(bundles, bundle)
	}

	return bundles, nil
}

// mapPortScan 映射端口扫描结果
func mapPortScan(result *orcModel.StageResult) ([]*AssetBundle, error) {
	var attr PortScanAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	servicesByIP := make(map[string][]*assetModel.AssetService)
	defaultIP := result.TargetValue
	// 如果 TargetValue 是网段，且没有具体 IP，这可能是一个问题
	// 但通常 Attributes 里会有 IP

	for _, p := range attr.Ports {
		if p.State != "open" {
			continue
		}

		targetIP := p.IP
		if targetIP == "" {
			targetIP = defaultIP
		}

		svc := &assetModel.AssetService{
			Port:        p.Port,
			Proto:       p.Proto,
			Name:        p.ServiceHint,
			Fingerprint: "{}",
			Tags:        "{}",
		}
		servicesByIP[targetIP] = append(servicesByIP[targetIP], svc)
	}

	var bundles []*AssetBundle

	for ip, services := range servicesByIP {
		host := &assetModel.AssetHost{
			IP:             ip,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
			Services:  services,
		})
	}

	// 如果没有发现服务，但 TargetValue 是单个 IP，则返回空服务的 Bundle
	if len(bundles) == 0 && !strings.Contains(defaultIP, "/") {
		host := &assetModel.AssetHost{
			IP:             defaultIP,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
		})
	}

	return bundles, nil
}

// mapServiceFingerprint 映射服务指纹识别结果
func mapServiceFingerprint(result *orcModel.StageResult) ([]*AssetBundle, error) {
	var attr ServiceFingerprintAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	servicesByIP := make(map[string][]*assetModel.AssetService)
	defaultIP := result.TargetValue

	for _, s := range attr.Services {
		targetIP := s.IP
		if targetIP == "" {
			targetIP = defaultIP
		}

		svc := &assetModel.AssetService{
			Port:        s.Port,
			Proto:       s.Proto,
			Name:        s.Name,
			Product:     s.Name, // Fingerprint Name is Product
			Version:     s.Version,
			CPE:         s.CPE,
			Banner:      s.Banner,
			Fingerprint: "{}",
			Tags:        "{}",
		}
		servicesByIP[targetIP] = append(servicesByIP[targetIP], svc)
	}

	var bundles []*AssetBundle

	for ip, services := range servicesByIP {
		host := &assetModel.AssetHost{
			IP:             ip,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
			Services:  services,
		})
	}

	if len(bundles) == 0 && !strings.Contains(defaultIP, "/") {
		host := &assetModel.AssetHost{
			IP:             defaultIP,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
		})
	}

	return bundles, nil
}

// mapVulnFinding 映射漏洞发现结果
func mapVulnFinding(result *orcModel.StageResult) ([]*AssetBundle, error) {
	var attr VulnFindingAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	vulnsByIP := make(map[string][]*assetModel.AssetVuln)
	defaultIP := result.TargetValue

	for _, finding := range attr.Findings {
		targetIP := finding.IP
		if targetIP == "" {
			targetIP = defaultIP
		}

		// 1. 构造标准化 Attributes 用于存储
		stdAttr := map[string]interface{}{
			"name":        finding.Name,
			"type":        finding.Type,
			"description": finding.Description,
			"solution":    finding.Solution,
			"reference":   finding.Reference,
			"port":        finding.Port,
			"url":         finding.URL,
			"ip":          targetIP,
		}
		stdAttrJSON, _ := json.Marshal(stdAttr)

		// 2. 构造 AssetVuln
		cve := finding.CVE
		if cve == "" {
			cve = extractCVE(finding.ID)
		}
		idAlias := strings.TrimSpace(finding.ID)
		if idAlias == "" {
			idAlias = strings.TrimSpace(cve)
		}
		if idAlias == "" {
			t := strings.TrimSpace(finding.Type)
			n := strings.TrimSpace(finding.Name)
			u := strings.TrimSpace(finding.URL)
			p := finding.Port
			if t != "" || n != "" || u != "" || p != 0 {
				idAlias = fmt.Sprintf("%s|%s|%d|%s", t, n, p, u)
			}
		}
		if idAlias == "" {
			s := sha1.Sum(stdAttrJSON)
			idAlias = "hash:" + hex.EncodeToString(s[:])
		}

		vuln := &assetModel.AssetVuln{
			TargetType: finding.TargetType,
			// TargetRefID: 0, // 留给 Merger 填充
			CVE:        cve,
			IDAlias:    idAlias,
			Severity:   finding.Severity,
			Confidence: finding.Confidence,
			Evidence:   fmt.Sprintf(`{"raw": "%s"}`, finding.Evidence),
			Attributes: string(stdAttrJSON),
			Status:     "open",
		}
		vulnsByIP[targetIP] = append(vulnsByIP[targetIP], vuln)
	}

	var bundles []*AssetBundle

	for ip, vulns := range vulnsByIP {
		host := &assetModel.AssetHost{
			IP:             ip,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
			Vulns:     vulns,
		})
	}

	if len(bundles) == 0 && !strings.Contains(defaultIP, "/") {
		host := &assetModel.AssetHost{
			IP:             defaultIP,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
		})
	}

	return bundles, nil
}

// extractCVE 从 ID 字符串中提取 CVE 编号
func extractCVE(id string) string {
	id = strings.ToUpper(id)
	if strings.HasPrefix(id, "CVE-") {
		return id
	}
	// 简单的正则提取可以后续补充
	return ""
}

// mapPocScan 映射PoC验证结果
func mapPocScan(result *orcModel.StageResult) ([]*AssetBundle, error) {
	var attr PocScanAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	vulnsByIP := make(map[string][]*assetModel.AssetVuln)
	defaultIP := result.TargetValue
	if strings.Contains(defaultIP, "://") {
		if uTarget, err := url.Parse(defaultIP); err == nil && uTarget.Hostname() != "" {
			defaultIP = uTarget.Hostname()
		}
	}

	now := time.Now()

	for _, res := range attr.PocResults {
		if res.Status != "confirmed" {
			continue
		}

		targetIP := res.IP
		if targetIP == "" {
			// 尝试从 Target 解析 IP
			if u, err := url.Parse(res.Target); err == nil && u.Hostname() != "" {
				targetIP = u.Hostname()
			} else {
				// 简单的 IP:Port 解析
				if strings.Contains(res.Target, ":") && !strings.Contains(res.Target, "://") {
					parts := strings.Split(res.Target, ":")
					targetIP = parts[0]
				} else {
					targetIP = defaultIP
				}
			}
		}

		stdAttr := map[string]interface{}{
			"poc_id":       res.PocID,
			"target":       res.Target,
			"evidence_ref": res.EvidenceRef,
			"ip":           targetIP,
		}
		stdAttrJSON, _ := json.Marshal(stdAttr)

		cve := extractCVE(res.PocID)
		vuln := &assetModel.AssetVuln{
			TargetType:   "unknown",
			TargetRefID:  0,
			CVE:          cve,
			IDAlias:      res.PocID,
			Severity:     res.Severity,
			Confidence:   100.0,
			Evidence:     fmt.Sprintf(`{"ref": "%s"}`, res.EvidenceRef),
			Attributes:   string(stdAttrJSON),
			Status:       "open",
			VerifyStatus: "verified",
			VerifiedBy:   "poc_scanner",
			VerifiedAt:   &now,
		}

		if strings.HasPrefix(res.Target, "http") {
			vuln.TargetType = "web"
		} else if strings.Contains(res.Target, ":") {
			vuln.TargetType = "service"
		} else {
			vuln.TargetType = "host"
		}

		vulnsByIP[targetIP] = append(vulnsByIP[targetIP], vuln)
	}

	var bundles []*AssetBundle

	for ip, vulns := range vulnsByIP {
		host := &assetModel.AssetHost{
			IP:             ip,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
			Vulns:     vulns,
		})
	}

	if len(bundles) == 0 && !strings.Contains(defaultIP, "/") {
		host := &assetModel.AssetHost{
			IP:             defaultIP,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
		})
	}

	return bundles, nil
}

// mapWebEndpoint 映射 Web 扫描结果
func mapWebEndpoint(result *orcModel.StageResult) ([]*AssetBundle, error) {
	var attr WebEndpointAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	webAssetsByIP := make(map[string][]*WebAsset)
	defaultIP := result.TargetValue
	if strings.Contains(defaultIP, "://") {
		if uTarget, err := url.Parse(defaultIP); err == nil && uTarget.Hostname() != "" {
			defaultIP = uTarget.Hostname()
		}
	}

	for _, ep := range attr.Endpoints {
		u, err := url.Parse(ep.URL)
		if err != nil {
			continue // Skip invalid URL
		}

		targetIP := ep.IP
		if targetIP == "" {
			// 优先使用 endpoint 的 IP，如果没有则尝试从 URL 解析 Hostname
			if u.Hostname() != "" {
				targetIP = u.Hostname()
			} else {
				targetIP = defaultIP
			}
		}

		port := 80
		if strings.EqualFold(u.Scheme, "https") {
			port = 443
		}
		if u.Port() != "" {
			fmt.Sscanf(u.Port(), "%d", &port)
		}

		// 2. 构建 AssetWeb
		techStackJSON, _ := json.Marshal(ep.TechStack)
		basicInfo := map[string]interface{}{
			"title":       ep.Title,
			"status_code": ep.StatusCode,
			"favicon":     ep.Favicon,
			"headers":     ep.Headers,
			"port":        port,
			"service":     u.Scheme,
		}
		basicInfoJSON, _ := json.Marshal(basicInfo)

		web := &assetModel.AssetWeb{
			URL:       ep.URL,
			Domain:    u.Hostname(),
			TechStack: string(techStackJSON),
			BasicInfo: string(basicInfoJSON),
			Tags:      "{}", // Tags 字段留空，后续删除,已经使用TagSystem支持打标
		}

		// 3. 构建 AssetWebDetail
		var detail *assetModel.AssetWebDetail
		if ep.Screenshot != "" || len(ep.Headers) > 0 {
			contentDetails := map[string]interface{}{
				"url":              ep.URL,
				"response_headers": ep.Headers,
			}
			contentDetailsJSON, _ := json.Marshal(contentDetails)
			detail = &assetModel.AssetWebDetail{
				ContentDetails: string(contentDetailsJSON),
				Screenshot:     ep.Screenshot,
				// TODO: 添加更多字段
			}
		}

		webAssetsByIP[targetIP] = append(webAssetsByIP[targetIP], &WebAsset{Web: web, Detail: detail})
	}

	var bundles []*AssetBundle

	for ip, webAssets := range webAssetsByIP {
		host := &assetModel.AssetHost{
			IP:             ip,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
			WebAssets: webAssets,
		})
	}

	if len(bundles) == 0 && !strings.Contains(defaultIP, "/") {
		host := &assetModel.AssetHost{
			IP:             defaultIP,
			Tags:           "{}",
			SourceStageIDs: "[]",
		}
		bundles = append(bundles, &AssetBundle{
			ProjectID: result.ProjectID,
			Host:      host,
		})
	}

	return bundles, nil
}

// mapPasswordAudit 映射密码审计结果
func mapPasswordAudit(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	err := fmt.Errorf("mapper not implemented: %s", result.ResultType)
	logger.LogError(err, "", 0, "", "etl.mapper.mapPasswordAudit", "", map[string]interface{}{
		"task_id":      result.TaskID,
		"project_id":   result.ProjectID,
		"stage_id":     result.StageID,
		"result_type":  result.ResultType,
		"target_type":  result.TargetType,
		"target_value": result.TargetValue,
	})
	return nil, err
}

// mapProxyDetection 映射代理检测结果
func mapProxyDetection(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	err := fmt.Errorf("mapper not implemented: %s", result.ResultType)
	logger.LogError(err, "", 0, "", "etl.mapper.mapProxyDetection", "", map[string]interface{}{
		"task_id":      result.TaskID,
		"project_id":   result.ProjectID,
		"stage_id":     result.StageID,
		"result_type":  result.ResultType,
		"target_type":  result.TargetType,
		"target_value": result.TargetValue,
	})
	return nil, err
}

// mapDirectoryScan 映射目录扫描结果
func mapDirectoryScan(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	err := fmt.Errorf("mapper not implemented: %s", result.ResultType)
	logger.LogError(err, "", 0, "", "etl.mapper.mapDirectoryScan", "", map[string]interface{}{
		"task_id":      result.TaskID,
		"project_id":   result.ProjectID,
		"stage_id":     result.StageID,
		"result_type":  result.ResultType,
		"target_type":  result.TargetType,
		"target_value": result.TargetValue,
	})
	return nil, err
}

// mapSubdomainDiscovery 映射子域发现结果
func mapSubdomainDiscovery(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	err := fmt.Errorf("mapper not implemented: %s", result.ResultType)
	logger.LogError(err, "", 0, "", "etl.mapper.mapSubdomainDiscovery", "", map[string]interface{}{
		"task_id":      result.TaskID,
		"project_id":   result.ProjectID,
		"stage_id":     result.StageID,
		"result_type":  result.ResultType,
		"target_type":  result.TargetType,
		"target_value": result.TargetValue,
	})
	return nil, err
}

// mapApiDiscovery 映射API发现结果
func mapApiDiscovery(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	err := fmt.Errorf("mapper not implemented: %s", result.ResultType)
	logger.LogError(err, "", 0, "", "etl.mapper.mapApiDiscovery", "", map[string]interface{}{
		"task_id":      result.TaskID,
		"project_id":   result.ProjectID,
		"stage_id":     result.StageID,
		"result_type":  result.ResultType,
		"target_type":  result.TargetType,
		"target_value": result.TargetValue,
	})
	return nil, err
}

// mapFileDiscovery 映射文件发现结果
func mapFileDiscovery(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	err := fmt.Errorf("mapper not implemented: %s", result.ResultType)
	logger.LogError(err, "", 0, "", "etl.mapper.mapFileDiscovery", "", map[string]interface{}{
		"task_id":      result.TaskID,
		"project_id":   result.ProjectID,
		"stage_id":     result.StageID,
		"result_type":  result.ResultType,
		"target_type":  result.TargetType,
		"target_value": result.TargetValue,
	})
	return nil, err
}

// mapOtherScan 映射其他扫描结果
func mapOtherScan(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	err := fmt.Errorf("mapper not implemented: %s", result.ResultType)
	logger.LogError(err, "", 0, "", "etl.mapper.mapOtherScan", "", map[string]interface{}{
		"task_id":      result.TaskID,
		"project_id":   result.ProjectID,
		"stage_id":     result.StageID,
		"result_type":  result.ResultType,
		"target_type":  result.TargetType,
		"target_value": result.TargetValue,
	})
	return nil, err
}
