package protocol

// MasterScanType 定义 Master 下发的业务扫描类型
type MasterScanType string

const (
	MasterTypeIdle          MasterScanType = "idle"          // 空闲
	MasterTypeIpAliveScan   MasterScanType = "ipAliveScan"   // IP探活
	MasterTypeFastPortScan  MasterScanType = "fastPortScan"  // 快速端口扫描
	MasterTypeFullPortScan  MasterScanType = "fullPortScan"  // 全量端口扫描
	MasterTypeServiceScan   MasterScanType = "serviceScan"   // 服务扫描
	MasterTypeVulnScan      MasterScanType = "vulnScan"      // 漏洞扫描
	MasterTypePocScan       MasterScanType = "pocScan"       // POC扫描
	MasterTypeWebScan       MasterScanType = "webScan"       // Web扫描
	MasterTypePassScan      MasterScanType = "passScan"      // 弱密码扫描
	MasterTypeProxyScan     MasterScanType = "proxyScan"     // 代理服务探测
	MasterTypeDirScan       MasterScanType = "dirScan"       // 目录扫描
	MasterTypeSubDomainScan MasterScanType = "subDomainScan" // 子域名扫描
	MasterTypeApiScan       MasterScanType = "apiScan"       // API资产扫描
	MasterTypeFileScan      MasterScanType = "fileScan"      // 文件扫描
	MasterTypeOtherScan     MasterScanType = "otherScan"     // 其他扫描
)

// MasterTask 定义 Master 下发的原始任务结构
type MasterTask struct {
	ID      string            `json:"id"`
	Type    MasterScanType    `json:"type"`
	Target  string            `json:"target"`
	Params  map[string]string `json:"params"` // Master下发的参数通常是简单的kv
	Timeout int64             `json:"timeout"`
}
