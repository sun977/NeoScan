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

	assetModel "neomaster/internal/model/asset"
	orcModel "neomaster/internal/model/orchestrator"
)

// AssetBundle 资产数据包
// 聚合了从 StageResult 中提取出的各类资产实体
// Processor 拿到此包后，根据字段是否非空调用对应的 Merger 逻辑
type AssetBundle struct {
	ProjectID  uint64                       // 所属项目ID (来自 StageResult)
	Host       *assetModel.AssetHost        // 主机资产 (必选)
	Services   []*assetModel.AssetService   // 关联的服务列表
	Webs       []*assetModel.AssetWeb       // 关联的 Web 站点
	WebDetails []*assetModel.AssetWebDetail // 关联的 Web 站点详情 (爬虫结果) Web 详情 (通常与 Webs 一一对应)
	Vulns      []*assetModel.AssetVuln      // 关联的漏洞列表
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
			Version:     s.Version,
			CPE:         s.CPE,
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
	// TODO: 实现逻辑
	return nil, nil
}

// mapWebEndpoint 映射 Web 扫描结果
func mapWebEndpoint(result *orcModel.StageResult) (*AssetBundle, error) {
	var attr struct {
		URL        string            `json:"url"`
		IP         string            `json:"ip"`
		Title      string            `json:"title"`
		Headers    map[string]string `json:"headers"`
		Screenshot string            `json:"screenshot"`
		TechStack  []string          `json:"tech_stack"`
		StatusCode int               `json:"status_code"`
		Favicon    string            `json:"favicon"`
	}
	if err := json.Unmarshal([]byte(result.Attributes), &attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	u, err := url.Parse(attr.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %s", attr.URL)
	}

	// 1. 确定 Host IP
	hostIP := result.TargetValue
	if attr.IP != "" {
		hostIP = attr.IP
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

	port := 80
	if strings.EqualFold(u.Scheme, "https") {
		port = 443
	}
	if u.Port() != "" {
		fmt.Sscanf(u.Port(), "%d", &port)
	}

	// 2. 构建 AssetWeb
	techStackJSON, _ := json.Marshal(attr.TechStack)
	basicInfo := map[string]interface{}{
		"title":       attr.Title,
		"status_code": attr.StatusCode,
		"favicon":     attr.Favicon,
		"headers":     attr.Headers,
		"port":        port,
		"service":     u.Scheme,
	}
	basicInfoJSON, _ := json.Marshal(basicInfo)

	web := &assetModel.AssetWeb{
		URL:       attr.URL,
		Domain:    u.Hostname(),
		TechStack: string(techStackJSON),
		BasicInfo: string(basicInfoJSON),
		Tags:      "{}",
	}

	// 3. 构建 AssetWebDetail (如果包含详细信息)
	// 注意: 此时没有 AssetWebID，Merger 会处理关联
	// 如果有截图或完整 Headers，建议创建 Detail
	var details []*assetModel.AssetWebDetail
	if attr.Screenshot != "" || len(attr.Headers) > 0 {
		contentDetails := map[string]interface{}{
			"url":              attr.URL,
			"response_headers": attr.Headers,
		}
		contentDetailsJSON, _ := json.Marshal(contentDetails)

		detail := &assetModel.AssetWebDetail{
			ContentDetails: string(contentDetailsJSON),
			Screenshot:     attr.Screenshot, // 可能是 ID 或 Base64，由 Processor/Handler 进一步处理
		}
		details = append(details, detail)
	}

	return &AssetBundle{
		Host:       host,
		Webs:       []*assetModel.AssetWeb{web},
		WebDetails: details,
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
