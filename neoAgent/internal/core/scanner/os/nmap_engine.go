package os

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"syscall"
	"time"

	"neoagent/internal/core/lib/network/netraw"
	"neoagent/internal/pkg/fingerprint/engines/nmap"
)

// NmapStackEngine 基于 Nmap OS DB 的 TCP/IP 协议栈指纹识别
type NmapStackEngine struct {
	db *nmap.OSDB
}

func NewNmapStackEngine() *NmapStackEngine {
	// 解析 OS DB
	// 注意: 这里应该只解析一次，最好是单例模式或在 Agent 启动时加载
	// 为了演示，这里简化处理
	db, err := nmap.ParseOSDB(nmap.NmapOSDB)
	if err != nil {
		fmt.Printf("Warning: Failed to parse Nmap OS DB: %v\n", err)
	}
	return &NmapStackEngine{
		db: db,
	}
}

func (e *NmapStackEngine) Name() string {
	return "nmap_stack"
}

func (e *NmapStackEngine) Scan(ctx context.Context, target string) (*OsInfo, error) {
	// 1. 平台检查 (仅 Linux 支持)
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("nmap stack fingerprinting is only supported on linux")
	}

	if e.db == nil {
		return nil, fmt.Errorf("nmap os db not initialized")
	}

	// 2. Demo: 发送 T1 Probe (SYN with TCP Options)
	// 假设目标开放了 80 端口 (实际应从 PortServiceScanner 获取)
	dstIP := net.ParseIP(target)
	if dstIP == nil {
		return nil, fmt.Errorf("invalid target ip: %s", target)
	}

	// 获取本机 IP
	srcIP, err := getLocalIP(dstIP)
	if err != nil {
		return nil, fmt.Errorf("failed to get local ip: %v", err)
	}

	// 创建 Raw Socket
	socket, err := netraw.NewRawSocket(syscall.IPPROTO_TCP)
	if err != nil {
		return nil, fmt.Errorf("create raw socket failed: %v (root required?)", err)
	}
	defer socket.Close()

	// 结果通道
	type scanResult struct {
		packet []byte
		err    error
	}
	resultChan := make(chan scanResult, 1)

	// 启动接收器
	go func() {
		buf := make([]byte, 1500)
		// 接收超时 3秒
		n, src, err1 := socket.Receive(buf, 3*time.Second)
		if err1 != nil {
			resultChan <- scanResult{err: err1}
			return
		}
		// 简单判断是否来自目标且是 SYN/ACK (Flags=0x12)
		if src.Equal(dstIP) && n > 20 {
			// TCP Header starts at buf[20] (assuming 20 bytes IP header)
			if n > 33 {
				flags := buf[20+13]
				if flags == 0x12 { // SYN+ACK
					// 复制包数据，避免 race condition
					pkt := make([]byte, n)
					copy(pkt, buf[:n])
					resultChan <- scanResult{packet: pkt}
					return
				}
			}
		}
		resultChan <- scanResult{err: fmt.Errorf("no matching packet received")}
	}()

	// 构建 T1 Probe 包
	// 简化版：仅发送 SYN，不带 Nmap 特定的复杂 Options
	// TODO: 完善 T1 Probe 的 Options 构造 (WScale, NOP, MSS, Timestamp, SACK_OK)

	// 构建 TCP 头
	// SrcPort: 54321, DstPort: 80, Seq: 100, Ack: 0, Flags: SYN
	tcpHeader := netraw.BuildTCPHeader(54321, 80, 100, 0, 0x02)

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
		if res.err != nil {
			return nil, fmt.Errorf("probe failed: %v", res.err)
		}

		// 3. 生成指纹
		fp := &nmap.OSFingerprint{
			MatchRule: make(map[string]string),
		}

		// 解析响应包并生成 T1 行
		// 这里简化实现，仅提取少量字段演示
		t1Line := generateT1Line(res.packet)
		fp.MatchRule["T1"] = t1Line

		// 4. 匹配
		matchResult := e.db.Match(fp)

		info := &OsInfo{
			Fingerprint: t1Line,
			Source:      "NmapStack/T1",
		}

		if matchResult != nil {
			info.Name = matchResult.Fingerprint.Name
			info.Accuracy = int(matchResult.Accuracy)
			info.Family = matchResult.Fingerprint.Class // 需要在 OsInfo 中添加 Family 字段? 暂无，放入 Name 或 Info
		} else {
			info.Name = "Unknown (Fingerprint: " + t1Line + ")"
			info.Accuracy = 0
		}

		return info, nil

	case <-time.After(4 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// generateT1Line 根据响应包生成 T1 指纹行
// 示例格式: R=Y%DF=Y%W=...
func generateT1Line(packet []byte) string {
	// 假设 packet 包含 IP Header (20 bytes) + TCP Header
	if len(packet) < 40 {
		return "R=N"
	}

	// IP Header: 0-19
	// TCP Header: 20-end

	// DF bit (IP Offset 6, bit 1)
	// Flags at offset 6: 3 bits (Reserved, DF, MF) + 13 bits Fragment Offset
	// 0x40 = 0100 0000 (DF set)
	ipFlags := packet[6]
	df := "N"
	if ipFlags&0x40 != 0 {
		df = "Y"
	}

	// TCP Window Size (Offset 20+14, 2 bytes)
	win := 0
	if len(packet) >= 36 {
		win = int(packet[34])<<8 | int(packet[35])
	}

	// 构造指纹行
	// 注意: 真实的 Nmap 指纹包含更多字段 (S, A, F, RD, Q, etc.)
	// 这里仅用于演示匹配逻辑
	// Nmap OS DB 中的数值通常使用十六进制
	return fmt.Sprintf("R=Y%%DF=%s%%W=%X", df, win)
}

// getLocalIP 获取与目标通信的本地 IP
func getLocalIP(dst net.IP) (net.IP, error) {
	conn, err := net.Dial("udp", dst.String()+":80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
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
