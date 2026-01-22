package alive

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"time"
	"unsafe"
)

// Windows 下使用 iphlpapi.dll 的 SendARP 函数
// DWORD SendARP(
//   IPAddr DestIP,
//   IPAddr SrcIP,
//   PULONG pMacAddr,
//   PULONG PhyAddrLen
// );

var (
	modIphlpapi = syscall.NewLazyDLL("iphlpapi.dll")
	procSendARP = modIphlpapi.NewProc("SendARP")
)

type ArpProber struct{}

func NewArpProber() *ArpProber {
	return &ArpProber{}
}

func (p *ArpProber) Probe(ctx context.Context, ip string, timeout time.Duration) (*ProbeResult, error) {
	// 转换 IP 字符串为 uint32
	destIP := net.ParseIP(ip)
	if destIP == nil {
		return nil, fmt.Errorf("invalid ip: %s", ip)
	}
	// 处理 IPv4
	destIPv4 := destIP.To4()
	if destIPv4 == nil {
		return nil, fmt.Errorf("not an ipv4 address: %s", ip)
	}

	// Windows SendARP 需要 IP 是网络字节序的 uint32 (实际上 SendARP 文档说是 IPAddr，即 unsigned long)
	// 但在 Go 的 syscall 中，我们需要正确传递。
	// net.IP.To4() 返回的是 [4]byte，需要转成 uint32
	destIpInt := binary.LittleEndian.Uint32(destIPv4)

	macAddr := make([]byte, 6)
	macAddrLen := uint32(len(macAddr))

	// 调用 SendARP
	// r1, _, _ := procSendARP.Call(
	// 	uintptr(destIpInt),
	// 	0, // SrcIP, 0 means default
	// 	uintptr(unsafe.Pointer(&macAddr[0])),
	// 	uintptr(unsafe.Pointer(&macAddrLen)),
	// )

	// 在 goroutine 中执行以支持 timeout
	// 注意：SendARP 是阻塞的，Windows API 层面没有简单的 timeout 参数 (除了系统默认)。
	// 我们可以用 goroutine 包装一下，虽然不能真正打断 SendARP，但可以控制上层返回。

	done := make(chan bool, 1)
	start := time.Now()

	go func() {
		// SendARP returns NO_ERROR (0) on success
		r1, _, _ := procSendARP.Call(
			uintptr(destIpInt),
			0,
			uintptr(unsafe.Pointer(&macAddr[0])),
			uintptr(unsafe.Pointer(&macAddrLen)),
		)
		if r1 == 0 { // NO_ERROR
			done <- true
		} else {
			done <- false
		}
	}()

	select {
	case success := <-done:
		if success {
			latency := time.Since(start)
			return NewProbeResult(true, latency, 0), nil
		}
		return &ProbeResult{Alive: false}, nil
	case <-ctx.Done():
		return &ProbeResult{Alive: false}, ctx.Err()
	}
}
