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

	assetModel "neomaster/internal/model/asset"
	orcModel "neomaster/internal/model/orchestrator"
)

// AssetBundle 资产数据包
// 聚合了从 StageResult 中提取出的各类资产实体
// Processor 拿到此包后，根据字段是否非空调用对应的 Merger 逻辑
type AssetBundle struct {
	Host       *assetModel.AssetHost        // 主机资产 (必选)
	Services   []*assetModel.AssetService   // 关联的服务列表
	Webs       []*assetModel.AssetWeb       // 关联的 Web 站点
	WebDetails []*assetModel.AssetWebDetail // 关联的 Web 站点详情 (爬虫结果) Web 详情 (通常与 Webs 一一对应)
	Vulns      []*assetModel.AssetVuln      // 关联的漏洞列表
}

// MapToAssetBundle 将通用的 StageResult 映射为标准资产包
// 根据 ResultType 分发到具体的映射逻辑
func MapToAssetBundle(result *orcModel.StageResult) (*AssetBundle, error) {
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
		return mapPasswordAudit(result)
	case "proxy_detection":
		return mapProxyDetection(result)
	case "directory_scan":
		return mapDirectoryScan(result)
	case "subdomain_discovery":
		return mapSubdomainDiscovery(result)
	case "api_discovery":
		return mapApiDiscovery(result)
	case "file_discovery":
		return mapFileDiscovery(result)
	case "other_scan":
		return mapOtherScan(result)
	default:
		return nil, fmt.Errorf("unsupported result type for mapping: %s", result.ResultType)
	}
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
	// TODO: 实现逻辑
	return nil, nil
}

// mapVulnFinding 映射漏洞发现结果
func mapVulnFinding(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}

// mapPocScan 映射PoC验证结果
func mapPocScan(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
}

// mapWebEndpoint 映射 Web 扫描结果
func mapWebEndpoint(result *orcModel.StageResult) (*AssetBundle, error) {
	// TODO: 实现逻辑
	return nil, nil
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
