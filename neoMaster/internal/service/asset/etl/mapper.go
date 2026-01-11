/**
 * Mapper 结构映射器
 * @author: sun977
 * @date: 2025.01.06
 * @description: 负责将 StageResult (标准 JSON 结构) 映射为内部 Asset 实体模型。
 * 它不解析原始工具输出(XML/Text)，只处理归一化后的数据。
 */
package etl

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	assetModel "neomaster/internal/model/asset"
	orcModel "neomaster/internal/model/orchestrator"
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

// MapToAssetBundle 将通用的 StageResult 映射为标准资产包
// 根据 ResultType 分发到具体的映射逻辑
func MapToAssetBundle(result *orcModel.StageResult) (*AssetBundle, error) {
	var bundle *AssetBundle
	var err error

	switch result.ResultType {
	case "ip_alive":
		bundle, err = mapIPAlive(result)
	case "fast_port_scan", "full_port_scan":
		bundle, err = mapPortScan(result)
	case "service_fingerprint":
		bundle, err = mapServiceFingerprint(result)
	case "vuln_finding":
		bundle, err = mapVulnFinding(result)
	case "poc_scan":
		bundle, err = mapPocScan(result)
	case "web_endpoint":
		bundle, err = mapWebEndpoint(result)
	case "password_audit":
		bundle, err = mapPasswordAudit(result)
	case "proxy_detection":
		bundle, err = mapProxyDetection(result)
	case "directory_scan":
		bundle, err = mapDirectoryScan(result)
	case "subdomain_discovery":
		bundle, err = mapSubdomainDiscovery(result)
	case "api_discovery":
		bundle, err = mapApiDiscovery(result)
	case "file_discovery":
		bundle, err = mapFileDiscovery(result)
	case "other_scan":
		bundle, err = mapOtherScan(result)
	default:
		return nil, fmt.Errorf("unsupported result type for mapping: %s", result.ResultType)
	}

	if err != nil {
		return nil, err
	}
	if bundle != nil {
		bundle.ProjectID = result.ProjectID
	}
	return bundle, nil
}

// mapIPAlive 映射探活结果
func mapIPAlive(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}

// mapPortScan 映射端口扫描结果
func mapPortScan(result *orcModel.StageResult) (*AssetBundle, error) {
	var attr PortScanAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	host := &assetModel.AssetHost{
		IP:             result.TargetValue,
		Tags:           "{}",
		SourceStageIDs: "[]",
	}

	var services []*assetModel.AssetService
	for _, p := range attr.Ports {
		if p.State == "open" {
			services = append(services, &assetModel.AssetService{
				Port:        p.Port,
				Proto:       p.Proto,
				Name:        p.ServiceHint,
				Fingerprint: "{}",
				Tags:        "{}",
			})
		}
	}

	return &AssetBundle{
		Host:     host,
		Services: services,
	}, nil
}

// mapServiceFingerprint 映射服务指纹识别结果
func mapServiceFingerprint(result *orcModel.StageResult) (*AssetBundle, error) {
	var attr ServiceFingerprintAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	host := &assetModel.AssetHost{
		IP:             result.TargetValue,
		Tags:           "{}",
		SourceStageIDs: "[]",
	}

	var services []*assetModel.AssetService
	for _, s := range attr.Services {
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

		// 如果没有 CPE 但有详细版本信息，尝试补充 (可选，依赖 Matcher)
		// 这里简单处理，直接使用 Agent 提供的数据
		services = append(services, svc)
	}

	return &AssetBundle{
		Host:     host,
		Services: services,
	}, nil
}

// mapVulnFinding 映射漏洞发现结果
func mapVulnFinding(result *orcModel.StageResult) (*AssetBundle, error) {
	var attr VulnFindingAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	var vulns []*assetModel.AssetVuln

	for _, finding := range attr.Findings {
		// 1. 构造标准化 Attributes 用于存储
		stdAttr := map[string]interface{}{
			"name":        finding.Name,
			"type":        finding.Type,
			"description": finding.Description,
			"solution":    finding.Solution,
			"reference":   finding.Reference,
			"port":        finding.Port,
			"url":         finding.URL,
			"ip":          result.TargetValue, // 假设 TargetValue 是 IP
		}
		stdAttrJSON, _ := json.Marshal(stdAttr)

		// 2. 构造 AssetVuln
		cve := finding.CVE
		if cve == "" {
			cve = extractCVE(finding.ID)
		}

		vuln := &assetModel.AssetVuln{
			TargetType: finding.TargetType,
			// TargetRefID: 0, // 留给 Merger 填充
			CVE:        cve,
			IDAlias:    finding.ID,
			Severity:   finding.Severity,
			Confidence: finding.Confidence,
			Evidence:   fmt.Sprintf(`{"raw": "%s"}`, finding.Evidence),
			Attributes: string(stdAttrJSON),
			Status:     "open",
		}
		vulns = append(vulns, vuln)
	}

	// 3. 构造 Host (总是需要的，用于定位)
	host := &assetModel.AssetHost{
		IP:             result.TargetValue,
		Tags:           "{}",
		SourceStageIDs: "[]",
	}

	return &AssetBundle{
		Host:  host,
		Vulns: vulns,
	}, nil
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
func mapPocScan(result *orcModel.StageResult) (*AssetBundle, error) {
	var attr PocScanAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	var vulns []*assetModel.AssetVuln
	now := time.Now()

	for _, res := range attr.PocResults {
		if res.Status != "confirmed" {
			continue
		}

		stdAttr := map[string]interface{}{
			"poc_id":       res.PocID,
			"target":       res.Target,
			"evidence_ref": res.EvidenceRef,
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

		vulns = append(vulns, vuln)
	}

	hostIP := result.TargetValue
	if strings.Contains(hostIP, "://") {
		if uTarget, err := url.Parse(hostIP); err == nil && uTarget.Hostname() != "" {
			hostIP = uTarget.Hostname()
		}
	}

	host := &assetModel.AssetHost{
		IP:             hostIP,
		Tags:           "{}",
		SourceStageIDs: "[]",
	}

	return &AssetBundle{
		Host:  host,
		Vulns: vulns,
	}, nil
}

// mapWebEndpoint 映射 Web 扫描结果
func mapWebEndpoint(result *orcModel.StageResult) (*AssetBundle, error) {
	var attr WebEndpointAttributes
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	// 1. 确定 Host IP (基于 TargetValue 或第一个 Endpoint)
	// 注意：如果 Endpoints 中包含不同 IP，这里只能取一个作为主 Host
	// 更复杂的场景可能需要拆分 Bundle，但目前假设 StageResult 针对单一目标
	hostIP := result.TargetValue
	if len(attr.Endpoints) > 0 && attr.Endpoints[0].IP != "" {
		hostIP = attr.Endpoints[0].IP
	} else {
		// 尝试从 TargetValue 解析 IP (如果 TargetValue 是 URL)
		if uTarget, err := url.Parse(result.TargetValue); err == nil && uTarget.Hostname() != "" {
			hostIP = uTarget.Hostname()
		}
	}

	host := &assetModel.AssetHost{
		IP:             hostIP,
		Tags:           "{}",
		SourceStageIDs: "[]",
	}

	var webAssets []*WebAsset // 包含 web 资产信息和详情信息

	for _, ep := range attr.Endpoints {
		u, err := url.Parse(ep.URL)
		if err != nil {
			continue // Skip invalid URL
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

		webAssets = append(webAssets, &WebAsset{Web: web, Detail: detail})
	}

	return &AssetBundle{
		Host:      host,
		WebAssets: webAssets,
	}, nil
}

// mapPasswordAudit 映射密码审计结果
func mapPasswordAudit(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}

// mapProxyDetection 映射代理检测结果
func mapProxyDetection(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}

// mapDirectoryScan 映射目录扫描结果
func mapDirectoryScan(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}

// mapSubdomainDiscovery 映射子域发现结果
func mapSubdomainDiscovery(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}

// mapApiDiscovery 映射API发现结果
func mapApiDiscovery(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}

// mapFileDiscovery 映射文件发现结果
func mapFileDiscovery(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}

// mapOtherScan 映射其他扫描结果
func mapOtherScan(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}
