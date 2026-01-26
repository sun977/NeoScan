package os

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"syscall"
	"time"

	"neoagent/internal/core/lib/network/netraw"
)

// NmapStackEngine 基于 Nmap OS DB 的 TCP/IP 协议栈指纹识别
type NmapStackEngine struct {
	// db *nmap.OsDB
}

func NewNmapStackEngine() *NmapStackEngine {
	return &NmapStackEngine{}
}

func (e *NmapStackEngine) Name() string {
	return "nmap_stack"
}

func (e *NmapStackEngine) Scan(ctx context.Context, target string) (*OsInfo, error) {
	// 1. 平台检查 (仅 Linux 支持)
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("nmap stack fingerprinting is only supported on linux")
	}

	// 2. Demo: 发送 T1 Probe (SYN with TCP Options)
	// 假设目标开放了 80 端口 (实际应从 PortServiceScanner 获取)
	// 这里硬编码用于验证 netraw 能力
	dstIP := net.ParseIP(target)
	if dstIP == nil {
		return nil, fmt.Errorf("invalid target ip: %s", target)
	}

	// 创建 Raw Socket
	socket, err := netraw.NewRawSocket(syscall.IPPROTO_TCP)
	if err != nil {
		return nil, fmt.Errorf("create raw socket failed: %v (root required?)", err)
	}
	defer socket.Close()

	// 启动接收器
	resultChan := make(chan string, 1)
	go func() {
		buf := make([]byte, 1500)
		// 接收超时 2秒
		n, src, err1 := socket.Receive(buf, 2*time.Second)
		if err1 != nil {
			// logger.LogBusinessError("NmapEngine", "receive failed", err1)
			return
		}
		// 简单判断是否来自目标且是 SYN/ACK (Flags=0x12)
		// 这里不做严谨解析，仅验证链路
		if src.Equal(dstIP) && n > 20 {
			// TCP Header starts at buf[20] (assuming 20 bytes IP header)
			// Flags at offset 13
			if n > 33 {
				flags := buf[20+13]
				if flags == 0x12 { // SYN+ACK
					resultChan <- "Received SYN/ACK from " + src.String()
				}
			}
		}
	}()

	// 构建 T1 Probe 包
	// Options: WScale(10), NOP, MSS(1460), Timestamp, SACK_OK
	// 简化版：仅发送 SYN
	srcIP := net.ParseIP("192.168.1.100") // TODO: 自动获取本机 IP

	// 构建 TCP 头
	tcpHeader := netraw.BuildTCPHeader(54321, 80, 100, 0, 0x02) // SYN

	// 构建 IP 包
	packet, err := netraw.BuildIPv4Packet(srcIP, dstIP, syscall.IPPROTO_TCP, tcpHeader)
	if err != nil {
		return nil, err
	}

	// 发送
	err = socket.Send(dstIP, packet)
	if err != nil {
		return nil, fmt.Errorf("send failed: %v", err)
	}

	// 等待结果
	select {
	case res := <-resultChan:
		return &OsInfo{
			Name:        "Linux/Network Device (Inferred by T1 Response)",
			Accuracy:    60,
			Fingerprint: res,
			Source:      "NmapStack/T1",
		}, nil
	case <-time.After(3 * time.Second):
		return nil, fmt.Errorf("timeout waiting for T1 response")
	}
}

// NmapServiceEngine 基于 PortServiceScanner 结果的 OS 识别
// 这是一个更实用的替代方案
type NmapServiceEngine struct {
	// 需要注入 PortServiceScanner 或者接收外部的扫描结果
}

func (e *NmapServiceEngine) Name() string {
	return "nmap_service"
}

// Scan 在这里可能会触发端口扫描，或者复用已有结果
func (e *NmapServiceEngine) Scan(ctx context.Context, target string) (*OsInfo, error) {
	// 实际逻辑应该集成在 PortServiceScanner 中，
	// 当发现 Service Banner 包含 "Windows" 或 "Linux" 时直接提取。
	// 这里作为独立 Engine 比较困难，因为需要先扫端口。

	return &OsInfo{
		Name:     "Inferred from Service (Not Implemented Standalone)",
		Accuracy: 0,
	}, nil
}
