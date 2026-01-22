//go:build !windows

package alive

import (
	"context"
	"time"
)

// Linux/Unix 下的 ARP 实现
// 这里暂时做一个简单的 stub 或者使用 udp 探测作为降级，
// 真正的 Linux ARP 需要 raw socket (AF_PACKET)，比较复杂且需要 root。
// 考虑到 "Pragmatic" 原则，如果不是 root 运行，raw socket 会失败。
// 我们可以尝试读取 /proc/net/arp 但那不是主动探测。
// 鉴于时间限制，这里暂时返回 false (不支持)，或者如果用户提供了 arping 工具路径可以调用。
// 为了保持代码整洁，我们暂时让非 Windows 环境下的 ArpProber 总是返回 false 
// (除非我们引入 gopacket 或者类似的库，但不想引入重依赖)。
// 
// 修正：我们可以尝试发送一个 UDP 包到高端口，然后监听 ICMP Port Unreachable，
// 但这其实不是 ARP。
// 
// 既然是 Linux，我们假设用户可能有 root 权限。
// 但为了快速实现且无依赖，我们这里先留空，或者使用一个简单的 ping fallback。
// 
// 更好的方案：在 Linux 下，如果用户指定了 ARP，我们提示需要 Root，
// 并且可以使用 "os/exec" 调用系统的 arping 命令如果存在。

type ArpProber struct{}

func NewArpProber() *ArpProber {
	return &ArpProber{}
}

func (p *ArpProber) Probe(ctx context.Context, ip string, timeout time.Duration) (bool, error) {
	// TODO: Implement Linux ARP probing using Raw Socket or 'arping' command
	// For now, return false to indicate not implemented/supported without extra deps
	return false, nil
}
