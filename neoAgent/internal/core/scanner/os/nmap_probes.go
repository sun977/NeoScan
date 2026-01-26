//go:build linux

package os

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"syscall"
	"time"

	"neoagent/internal/core/lib/network/netraw"
	"neoagent/internal/pkg/fingerprint/engines/nmap"
)

// Probe Types
const (
	ProbeTypeSEQ = iota
	ProbeTypeECN
	ProbeTypeT2
	ProbeTypeT3
	ProbeTypeT4
	ProbeTypeT5
	ProbeTypeT6
	ProbeTypeT7
	ProbeTypeIE
	ProbeTypeU1
)

type ProbeRequest struct {
	Type     int
	Packet   []byte
	Protocol int
	SrcPort  int
	DstPort  int
	SendTime time.Time
}

type ProbeResponse struct {
	Type     int
	Packet   []byte
	RecvTime time.Time
	SrcIP    net.IP
}

// executeProbes 执行全量探测
func (e *NmapStackEngine) executeProbes(ctx context.Context, target string, openPort, closedPort int) (*nmap.OSFingerprint, error) {
	dstIP := net.ParseIP(target)
	srcIP, err := getLocalIP(dstIP)
	if err != nil {
		return nil, err
	}

	// 初始化 Raw Sockets
	tcpConn, err := netraw.NewRawSocket(syscall.IPPROTO_TCP)
	if err != nil {
		return nil, fmt.Errorf("tcp socket error: %v", err)
	}
	defer tcpConn.Close()

	udpConn, err := netraw.NewRawSocket(syscall.IPPROTO_UDP)
	if err != nil {
		return nil, fmt.Errorf("udp socket error: %v", err)
	}
	defer udpConn.Close()

	icmpConn, err := netraw.NewRawSocket(syscall.IPPROTO_ICMP)
	if err != nil {
		return nil, fmt.Errorf("icmp socket error: %v", err)
	}
	defer icmpConn.Close()

	// 准备探测包
	probes := buildAllProbes(srcIP, dstIP, openPort, closedPort)

	// 启动接收器
	responses := make(map[int]*ProbeResponse)
	recvCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 统一接收处理
	go e.receiverLoop(recvCtx, tcpConn, udpConn, icmpConn, dstIP, probes, responses)

	// 发送探测包
	// Nmap 顺序: SEQ(1-6) -> IE(1-2) -> ECN -> T2-T7 -> U1
	// 这里为了简单，按顺序发送，每个间隔 100ms
	for _, p := range probes {
		var conn *netraw.RawSocket
		switch p.Protocol {
		case syscall.IPPROTO_TCP:
			conn = tcpConn
		case syscall.IPPROTO_UDP:
			conn = udpConn
		case syscall.IPPROTO_ICMP:
			conn = icmpConn
		}

		p.SendTime = time.Now()
		if err := conn.Send(dstIP, p.Packet); err != nil {
			// Log error but continue
			fmt.Printf("Probe send failed: %v\n", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 等待接收完成 (超时 3s)
	select {
	case <-time.After(3 * time.Second):
		cancel()
	case <-ctx.Done():
		cancel()
	}

	// 生成指纹
	fp := generateFingerprint(responses)
	return fp, nil
}

func (e *NmapStackEngine) receiverLoop(ctx context.Context, tcp, udp, icmp *netraw.RawSocket, targetIP net.IP, probes []*ProbeRequest, responses map[int]*ProbeResponse) {
	// 简单的轮询读取 (实际应该用 Select 或多协程，这里简化为三个协程写入同一个 map，需要锁)
	// 由于 map 非并发安全，改用 channel 传递 response
	respChan := make(chan *ProbeResponse, 20)

	// 启动 3 个 listener
	go listenSocket(ctx, tcp, targetIP, respChan)
	go listenSocket(ctx, udp, targetIP, respChan)
	go listenSocket(ctx, icmp, targetIP, respChan)

	for {
		select {
		case <-ctx.Done():
			return
		case resp := <-respChan:
			// 匹配 Response 到 Probe
			// 根据端口、协议、Seq 等匹配
			matchedProbe := matchProbe(resp, probes)
			if matchedProbe != nil {
				responses[matchedProbe.Type] = resp
			}
		}
	}
}

func listenSocket(ctx context.Context, sock *netraw.RawSocket, targetIP net.IP, out chan<- *ProbeResponse) {
	buf := make([]byte, 1500)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Non-blocking read or short timeout
			n, src, err := sock.Receive(buf, 100*time.Millisecond)
			if err != nil {
				continue
			}
			if !src.Equal(targetIP) {
				continue
			}

			// Copy data
			pkt := make([]byte, n)
			copy(pkt, buf[:n])

			// 简单协议判断
			// IP Header 20 bytes. Protocol at 9.
			if n < 20 {
				continue
			}
			protocol := int(pkt[9])

			out <- &ProbeResponse{
				Packet:   pkt,
				RecvTime: time.Now(),
				SrcIP:    src,
				Type:     protocol, // 临时用 protocol 标记，后面再细分
			}
		}
	}
}

// matchProbe 匹配响应到具体的探测请求
func matchProbe(resp *ProbeResponse, probes []*ProbeRequest) *ProbeRequest {
	// 解析 IP Header
	ipHeaderLen := (int(resp.Packet[0]) & 0x0F) * 4
	if len(resp.Packet) < ipHeaderLen+8 {
		return nil
	}

	protocol := resp.Packet[9]
	payload := resp.Packet[ipHeaderLen:]

	for _, p := range probes {
		// 1. 协议匹配
		if int(protocol) != p.Protocol {
			continue
		}

		// 2. 端口/ID 匹配
		switch p.Protocol {
		case syscall.IPPROTO_TCP:
			// TCP: SrcPort matches Probe DstPort, DstPort matches Probe SrcPort
			// TCP Header: Src(0-1), Dst(2-3)
			srcPort := int(binary.BigEndian.Uint16(payload[0:2]))
			dstPort := int(binary.BigEndian.Uint16(payload[2:4]))
			if srcPort == p.DstPort && dstPort == p.SrcPort {
				// 还可以检查 ACK == SEQ + 1
				return p
			}
		case syscall.IPPROTO_UDP:
			// UDP: ICMP Port Unreachable?
			// 如果是 UDP 响应 (极少见)，匹配端口
			// 如果是 ICMP Error (常见)，需要解析 Inner IP Header
		case syscall.IPPROTO_ICMP:
			// Echo Reply: ID matches
			// Type(0), Code(0), Checksum(2), ID(4), Seq(6)
			msgType := payload[0]
			if msgType == 0 { // Echo Reply
				id := int(binary.BigEndian.Uint16(payload[4:6]))
				seq := int(binary.BigEndian.Uint16(payload[6:8]))
				// 这里的 ID/Seq 是我们在构建 Probe 时设置的
				// 假设我们在 Probe Packet 里也是标准 ICMP
				// 需要从 p.Packet 解析出 ID/Seq
				// 简化：p.Packet[20:] is ICMP Header (IP header is 20)
				// Wait, p.Packet includes IP Header
				reqPayload := p.Packet[20:]
				reqID := int(binary.BigEndian.Uint16(reqPayload[4:6]))
				reqSeq := int(binary.BigEndian.Uint16(reqPayload[6:8]))

				if id == reqID && seq == reqSeq {
					return p
				}
			}
		}
	}
	return nil
}

// buildAllProbes 构建所有 Nmap 探测包
func buildAllProbes(src, dst net.IP, open, closed int) []*ProbeRequest {
	var probes []*ProbeRequest
	baseSrcPort := 40000 + rand.Intn(10000)

	// --- SEQ Probes (6 packets) ---
	// SEQ1: SYN, Opts: WScale(10), NOP, MSS(1460), TS, SACKPerm
	// SEQ2: SYN, Opts: MSS(1400), WScale(0), SACKPerm, TS, EOL
	// ... 省略具体细节，实现几个典型的

	// SEQ1
	opts1 := []netraw.TCPOption{
		{Kind: netraw.TCPOptionWScale, Length: 3, Data: []byte{10}},
		{Kind: netraw.TCPOptionNOP},
		{Kind: netraw.TCPOptionMSS, Length: 4, Data: []byte{0x05, 0xB4}}, // 1460
		{Kind: netraw.TCPOptionTimestamp, Length: 10, Data: make([]byte, 8)},
		{Kind: netraw.TCPOptionSACKPermit, Length: 2},
	}
	probes = append(probes, makeTCPProbe(src, dst, baseSrcPort+1, open, 1, netraw.TCPOptionMSS, opts1, ProbeTypeSEQ)) // Use MSS flag? No, SYN

	// --- ECN Probe ---
	// SYN | ECE | CWR
	// Opts: WScale(10), NOP, MSS(1460), SACKPerm, NOP, NOP
	optsECN := []netraw.TCPOption{
		{Kind: netraw.TCPOptionWScale, Length: 3, Data: []byte{10}},
		{Kind: netraw.TCPOptionNOP},
		{Kind: netraw.TCPOptionMSS, Length: 4, Data: []byte{0x05, 0xB4}},
		{Kind: netraw.TCPOptionSACKPermit, Length: 2},
		{Kind: netraw.TCPOptionNOP},
		{Kind: netraw.TCPOptionNOP},
	}
	// Flags: SYN(2) | ECE(64) | CWR(128) = 194
	probes = append(probes, makeTCPProbe(src, dst, baseSrcPort+10, open, 0, 0xC2, optsECN, ProbeTypeECN))

	// --- T2-T7 Probes ---
	// T2: Open Port, NULL, Opts: WScale(10), NOP, MSS(265), TS, SACK
	optsT2 := []netraw.TCPOption{
		{Kind: netraw.TCPOptionWScale, Length: 3, Data: []byte{10}},
		{Kind: netraw.TCPOptionNOP},
		{Kind: netraw.TCPOptionMSS, Length: 4, Data: []byte{0x01, 0x09}}, // 265
		{Kind: netraw.TCPOptionTimestamp, Length: 10, Data: make([]byte, 8)},
		{Kind: netraw.TCPOptionSACKPermit, Length: 2},
	}
	probes = append(probes, makeTCPProbe(src, dst, baseSrcPort+2, open, 0, 0, optsT2, ProbeTypeT2))

	// T3: Open Port, SYN|FIN|URG|PSH, Opts: ...
	// Flags: SYN(2)|FIN(1)|URG(32)|PSH(8) = 43
	probes = append(probes, makeTCPProbe(src, dst, baseSrcPort+3, open, 0, 0x2B, optsT2, ProbeTypeT3))

	// T4: Open Port, ACK, Opts: ...
	// Flags: ACK(16)
	probes = append(probes, makeTCPProbe(src, dst, baseSrcPort+4, open, 0, 0x10, optsT2, ProbeTypeT4))

	// T5: Closed Port, SYN
	probes = append(probes, makeTCPProbe(src, dst, baseSrcPort+5, closed, 0, 0x02, optsT2, ProbeTypeT5))

	// T6: Closed Port, ACK
	probes = append(probes, makeTCPProbe(src, dst, baseSrcPort+6, closed, 0, 0x10, optsT2, ProbeTypeT6))

	// T7: Closed Port, FIN|PSH|URG
	// Flags: FIN(1)|PSH(8)|URG(32) = 41
	probes = append(probes, makeTCPProbe(src, dst, baseSrcPort+7, closed, 0, 0x29, optsT2, ProbeTypeT7))

	// --- IE Probe ---
	iePkt, _ := netraw.BuildIPv4Packet(src, dst, syscall.IPPROTO_ICMP, makeICMPPayload(1, 1))
	probes = append(probes, &ProbeRequest{
		Type:     ProbeTypeIE,
		Packet:   iePkt,
		Protocol: syscall.IPPROTO_ICMP,
	})

	// --- U1 Probe ---
	// UDP to closed port, payload 'C' * 300
	udpPayload := bytes.Repeat([]byte{'C'}, 300)
	udpHeader, _ := netraw.BuildUDPHeader(src, dst, baseSrcPort+8, closed, udpPayload)
	u1Pkt, _ := netraw.BuildIPv4Packet(src, dst, syscall.IPPROTO_UDP, udpHeader)
	probes = append(probes, &ProbeRequest{
		Type:     ProbeTypeU1,
		Packet:   u1Pkt,
		Protocol: syscall.IPPROTO_UDP,
		SrcPort:  baseSrcPort + 8,
		DstPort:  closed,
	})

	return probes
}

func makeTCPProbe(src, dst net.IP, srcPort, dstPort int, seq uint32, flags int, opts []netraw.TCPOption, pType int) *ProbeRequest {
	// TCP Header
	tcpHeader, _ := netraw.BuildTCPHeaderWithChecksum(src, dst, srcPort, dstPort, seq, 0, flags, 1024, 0, opts)
	// IPv4 Packet
	pkt, _ := netraw.BuildIPv4Packet(src, dst, syscall.IPPROTO_TCP, tcpHeader)

	return &ProbeRequest{
		Type:     pType,
		Packet:   pkt,
		Protocol: syscall.IPPROTO_TCP,
		SrcPort:  srcPort,
		DstPort:  dstPort,
	}
}

func makeICMPPayload(id, seq int) []byte {
	// ICMP Echo Request
	data := make([]byte, 120) // Payload 120 bytes of 0x00
	pkt, _ := netraw.BuildICMPEchoRequest(id, seq, data)
	return pkt
}

// generateFingerprint 生成指纹
func generateFingerprint(responses map[int]*ProbeResponse) *nmap.OSFingerprint {
	fp := &nmap.OSFingerprint{
		MatchRule: make(map[string]string),
	}

	// 1. SEQ/OPS/WIN/T1
	// 这里需要综合 6 个 SEQ 包的结果。简化：只用 T1 (SEQ1) 的响应
	if resp, ok := responses[ProbeTypeSEQ]; ok {
		fp.MatchRule["T1"] = parseTCPResponse(resp)
	} else {
		fp.MatchRule["T1"] = "R=N"
	}

	// 2. T2-T7
	types := []int{ProbeTypeT2, ProbeTypeT3, ProbeTypeT4, ProbeTypeT5, ProbeTypeT6, ProbeTypeT7}
	names := []string{"T2", "T3", "T4", "T5", "T6", "T7"}
	for i, t := range types {
		if resp, ok := responses[t]; ok {
			fp.MatchRule[names[i]] = parseTCPResponse(resp)
		} else {
			fp.MatchRule[names[i]] = "R=N"
		}
	}

	// 3. ECN
	if resp, ok := responses[ProbeTypeECN]; ok {
		fp.MatchRule["ECN"] = parseTCPResponse(resp) // ECN parsing is slightly different (CC, etc.)
	} else {
		fp.MatchRule["ECN"] = "R=N"
	}

	// 4. IE
	if resp, ok := responses[ProbeTypeIE]; ok {
		fp.MatchRule["IE"] = parseICMPResponse(resp)
	} else {
		fp.MatchRule["IE"] = "R=N"
	}

	// 5. U1 (UDP)
	if resp, ok := responses[ProbeTypeU1]; ok {
		fp.MatchRule["U1"] = parseU1Response(resp)
	} else {
		fp.MatchRule["U1"] = "R=N"
	}

	return fp
}

func parseTCPResponse(resp *ProbeResponse) string {
	// 解析 TCP 响应生成 Nmap 格式字符串
	// R=Y%DF=Y%W=...

	packet := resp.Packet
	if len(packet) < 40 {
		return "R=Y" // 收到包但太短
	}

	// IP Header
	ipFlags := packet[6]
	df := "N"
	if ipFlags&0x40 != 0 {
		df = "Y"
	}

	// TCP Header
	tcpOffset := (int(packet[0]) & 0x0F) * 4
	if len(packet) < tcpOffset+20 {
		return fmt.Sprintf("R=Y%%DF=%s", df)
	}
	tcpBase := packet[tcpOffset:]

	// Window
	win := int(binary.BigEndian.Uint16(tcpBase[14:16]))

	// Flags
	flags := tcpBase[13]
	s_flag := "Z"
	if flags&0x02 != 0 {
		s_flag = "S"
	} // SYN
	a_flag := "Z"
	if flags&0x10 != 0 {
		a_flag = "A"
	} // ACK
	f_flag := "Z"
	if flags&0x01 != 0 {
		f_flag = "F"
	} // FIN

	// TTL Guess (TG)
	// 通常 Nmap 会根据 TTL 推断初始 TTL
	ttl := int(packet[8])

	// Options
	// TODO: Parse options (O=...)

	return fmt.Sprintf("R=Y%%DF=%s%%TG=%X%%W=%X%%S=%s%%A=%s%%F=%s", df, ttl, win, s_flag, a_flag, f_flag)
}

func parseICMPResponse(resp *ProbeResponse) string {
	// IE(R=Y%DFI=N%T=...)
	// 简单实现
	return "R=Y%DFI=N"
}

func parseU1Response(resp *ProbeResponse) string {
	// U1(R=Y%DF=N%T=...)
	// 如果收到的是 ICMP Port Unreachable，说明 R=Y
	return "R=Y%DF=N"
}
