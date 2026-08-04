package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// defaultBrutePorts 按服务名给出默认探测端口，与 CLI 历史行为保持一致
var defaultBrutePorts = map[string]string{
	"ssh":           "22",
	"mysql":         "3306",
	"redis":         "6379",
	"postgres":      "5432",
	"postgresql":    "5432",
	"ftp":           "21",
	"mongo":         "27017",
	"mongodb":       "27017",
	"clickhouse":    "9000",
	"smb":           "445",
	"mssql":         "1433",
	"oracle":        "1521",
	"oracle-sid":    "1521",
	"telnet":        "23",
	"elasticsearch": "9200",
	"snmp":          "161",
}

// BruteScanOptions 对应 弱口令爆破 的参数
type BruteScanOptions struct {
	Target    string
	Port      string
	Service   string
	Users     []string
	Passwords []string

	// ScanAll 为 true 表示尝试所有凭据；默认 false，即找到一个成功即停止
	ScanAll bool
}

func NewBruteScanOptions() *BruteScanOptions {
	return &BruteScanOptions{}
}

// Validate 校验参数，并在 Port 缺省时按 Service 推断默认端口
// 注意: 端口推断必须在 Validate 中完成（而不是放在 ToTask 或 CLI 里），
// 因为它属于"参数合法性/完整性"的一部分，Validate 之后 Port 必须已确定。
func (o *BruteScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	if o.Service == "" {
		return fmt.Errorf("service is required (ssh, mysql, redis, postgres, ftp, mongo, clickhouse, smb, mssql, oracle, oracle-sid, telnet, elasticsearch, snmp)")
	}

	if o.Port == "" {
		defaultPort, ok := defaultBrutePorts[o.Service]
		if !ok {
			return fmt.Errorf("port is required (-p): no default port known for service %q", o.Service)
		}
		o.Port = defaultPort
	}

	return nil
}

func (o *BruteScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeBrute, o.Target)
	task.PortRange = o.Port
	task.Timeout = 1 * time.Hour

	task.Params["service"] = o.Service
	// 用户指定 ScanAll 时，stop_on_success 为 false
	task.Params["stop_on_success"] = !o.ScanAll

	if len(o.Users) > 0 {
		task.Params["users"] = o.Users
	}
	if len(o.Passwords) > 0 {
		task.Params["passwords"] = o.Passwords
	}

	return task
}
