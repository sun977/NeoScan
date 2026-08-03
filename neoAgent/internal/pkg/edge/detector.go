package edge

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"neoagent/internal/pkg/utils"
)

// Detector 是 CDN 网段检测器，持有当前生效的规则集，支持并发读取与
// 运行期重新加载（Reload），为后续 Master 规则同步预留接口——见
// docs/爬虫/Web扫描CDN识别方案.md 第 6.2 节。
type Detector struct {
	mu    sync.RWMutex
	rules []ProviderRanges
}

// NewDetector 创建一个空的检测器，需要显式调用 Load 加载规则后才具备
// 判断能力；加载前 Check 恒定返回 false，不影响调用方的正常流程。
func NewDetector() *Detector {
	return &Detector{}
}

// Load 从指定路径读取 CDN 网段规则文件并替换当前生效的规则集。
//
// 可以被重复调用：每次调用都是一次完整替换（不是增量合并），语义上
// 对应"用一份新规则覆盖旧规则"，与 fpEngine.Reload 的语义保持一致。
func (d *Detector) Load(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read cdn rules file %s: %w", path, err)
	}

	var rules []ProviderRanges
	if err := json.Unmarshal(content, &rules); err != nil {
		return fmt.Errorf("unmarshal cdn rules file %s: %w", path, err)
	}

	d.mu.Lock()
	d.rules = rules
	d.mu.Unlock()
	return nil
}

// Check 判断给定 IP 是否命中某个 CDN 厂商的网段，命中则返回
// (true, 厂商名)，未命中或规则集为空返回 (false, "")。
//
// 传入非法 IP 字符串时同样返回 (false, "")，不返回 error——调用方
// （runOnePort）在无法判断时应该按"不是 CDN"处理，继续走原有流程，
// 而不是因为 CDN 判断本身失败就中断整个扫描任务。
func (d *Detector) Check(ip string) (bool, string) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false, ""
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, pr := range d.rules {
		for _, cidr := range pr.CIDRs {
			if utils.IsIPInCIDR(parsed, cidr) {
				return true, pr.Provider
			}
		}
	}
	return false, ""
}
