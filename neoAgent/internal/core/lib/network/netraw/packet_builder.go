package netraw

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"

	"golang.org/x/net/ipv4"
)

// TCP Options
const (
	TCPOptionEOL        = 0
	TCPOptionNOP        = 1
	TCPOptionMSS        = 2
	TCPOptionWScale     = 3
	TCPOptionSACKPermit = 4
	TCPOptionSACK       = 5
	TCPOptionTimestamp  = 8
)

// TCPOption represents a TCP option
type TCPOption struct {
	Kind   uint8
	Length uint8
	Data   []byte
}

// BuildIPv4Packet 构建 IPv4 头部和负载
func BuildIPv4Packet(src, dst net.IP, protocol int, payload []byte) ([]byte, error) {
	header := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TotalLen: ipv4.HeaderLen + len(payload),
		ID:       rand.Intn(65535),
		TTL:      64,
		Protocol: protocol,
		Src:      src,
		Dst:      dst,
	}

	h, err := header.Marshal()
	if err != nil {
		return nil, err
	}

	return append(h, payload...), nil
}

// Checksum 计算 16-bit One's Complement Checksum
func Checksum(data []byte) uint16 {
	var (
		sum    uint32
		length = len(data)
		index  int
	)

	for length > 1 {
		sum += uint32(binary.BigEndian.Uint16(data[index:]))
		index += 2
		length -= 2
	}

	if length > 0 {
		sum += uint32(uint8(data[index])) << 8
	}

	for (sum >> 16) > 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return uint16(^sum)
}

// BuildTCPHeaderWithChecksum 构建完整的 TCP 头部 (含 Options 和 Checksum)
func BuildTCPHeaderWithChecksum(srcIP, dstIP net.IP, srcPort, dstPort int, seq, ack uint32, flags int, window uint16, urgentPtr uint16, options []TCPOption) ([]byte, error) {
	// 1. 构建 Options 数据
	var optBuf bytes.Buffer
	for _, opt := range options {
		optBuf.WriteByte(opt.Kind)
		if opt.Kind == TCPOptionNOP || opt.Kind == TCPOptionEOL {
			continue
		}
		optBuf.WriteByte(opt.Length)
		optBuf.Write(opt.Data)
	}

	// Padding to 4-byte boundary
	padLen := (4 - (optBuf.Len() % 4)) % 4
	for i := 0; i < padLen; i++ {
		optBuf.WriteByte(TCPOptionNOP) // 通常用 NOP 填充，或者 EOL
	}
	optData := optBuf.Bytes()

	// 2. 计算 Data Offset
	// Base header 20 bytes + options length
	headerLen := 20 + len(optData)
	if headerLen > 60 {
		return nil, fmt.Errorf("tcp header too large: %d", headerLen)
	}
	dataOffset := headerLen / 4

	// 3. 构建 TCP Header 基础部分
	h := make([]byte, headerLen)

	binary.BigEndian.PutUint16(h[0:], uint16(srcPort))
	binary.BigEndian.PutUint16(h[2:], uint16(dstPort))
	binary.BigEndian.PutUint32(h[4:], seq)
	binary.BigEndian.PutUint32(h[8:], ack)

	// Data Offset (4 bits) + Reserved (3 bits) + Flags (9 bits)
	// h[12]: Data Offset (4) + Res (4, high part)
	// h[13]: Res (2, low part) + Flags (6) -> wait, standard is:
	// Offset(4) | Reserved(3) | NS(1) | CWR(1) | ECE(1) | URG(1) | ACK(1) | PSH(1) | RST(1) | SYN(1) | FIN(1)
	// Flags int usually covers the lower 9 bits (NS to FIN)
	// Flags: 0000 0000 0000 0000 0000 0001 1111 1111
	// Let's assume input 'flags' maps to standard TCP flags bitmask

	// h[12] = (DataOffset << 4) | ((Reserved & 0x07) << 1) | (NS)
	// For simplicity, assume Reserved is 0.
	// But ECN uses Reserved bits? No, ECN uses ECE and CWR which are part of the flags/reserved byte.
	// Standard Layout:
	// Byte 12: Data Offset(4) | Reserved(3) | NS(1)
	// Byte 13: CWR(1) | ECE(1) | URG(1) | ACK(1) | PSH(1) | RST(1) | SYN(1) | FIN(1)

	// Let's support full 9-bit flags + Reserved if needed.
	// But usually 'flags' argument is just the lower 8 bits or 9 bits.
	// We will assume 'flags' contains CWR/ECE/URG/ACK/PSH/RST/SYN/FIN.
	// NS is bit 8 (0x100).

	// Flags definition commonly:
	// FIN 0x01
	// SYN 0x02
	// RST 0x04
	// PSH 0x08
	// ACK 0x10
	// URG 0x20
	// ECE 0x40
	// CWR 0x80
	// NS  0x100

	h[12] = byte((dataOffset << 4) | ((flags >> 8) & 0x01))
	h[13] = byte(flags & 0xFF)

	binary.BigEndian.PutUint16(h[14:], window)
	// Checksum h[16] initially 0
	binary.BigEndian.PutUint16(h[18:], urgentPtr)

	// Copy Options
	copy(h[20:], optData)

	// 4. 计算 Checksum (Pseudo Header + TCP Header + Data)
	// Data is empty here for pure probe
	ph := make([]byte, 12)
	copy(ph[0:4], srcIP.To4())
	copy(ph[4:8], dstIP.To4())
	ph[8] = 0 // Reserved
	ph[9] = 6 // Protocol TCP
	binary.BigEndian.PutUint16(ph[10:], uint16(headerLen))

	var buf bytes.Buffer
	buf.Write(ph)
	buf.Write(h)

	checksum := Checksum(buf.Bytes())
	binary.BigEndian.PutUint16(h[16:], checksum)

	return h, nil
}

// BuildICMPEchoRequest 构建 ICMP Echo Request
func BuildICMPEchoRequest(id, seq int, payload []byte) ([]byte, error) {
	// Type(8), Code(0), Checksum(2), ID(2), Seq(2)
	h := make([]byte, 8)
	h[0] = 8 // Echo Request
	h[1] = 0 // Code 0

	binary.BigEndian.PutUint16(h[4:], uint16(id))
	binary.BigEndian.PutUint16(h[6:], uint16(seq))

	// Checksum (Header + Payload)
	var buf bytes.Buffer
	buf.Write(h)
	buf.Write(payload)

	checksum := Checksum(buf.Bytes())
	binary.BigEndian.PutUint16(h[2:], checksum)

	return append(h, payload...), nil
}

// BuildUDPHeader 构建 UDP 头部
func BuildUDPHeader(srcIP, dstIP net.IP, srcPort, dstPort int, payload []byte) ([]byte, error) {
	length := 8 + len(payload)
	h := make([]byte, 8)

	binary.BigEndian.PutUint16(h[0:], uint16(srcPort))
	binary.BigEndian.PutUint16(h[2:], uint16(dstPort))
	binary.BigEndian.PutUint16(h[4:], uint16(length))
	// Checksum at h[6]

	// Pseudo Header for Checksum
	ph := make([]byte, 12)
	copy(ph[0:4], srcIP.To4())
	copy(ph[4:8], dstIP.To4())
	ph[8] = 0
	ph[9] = 17 // Protocol UDP
	binary.BigEndian.PutUint16(ph[10:], uint16(length))

	var buf bytes.Buffer
	buf.Write(ph)
	buf.Write(h)
	buf.Write(payload)

	checksum := Checksum(buf.Bytes())
	// UDP Checksum 0 means no checksum, but if calculated 0, should be 0xFFFF
	if checksum == 0 {
		checksum = 0xFFFF
	}
	binary.BigEndian.PutUint16(h[6:], checksum)

	return append(h, payload...), nil
}

// BuildTCPHeader (Legacy compatibility)
func BuildTCPHeader(srcPort, dstPort int, seq, ack uint32, flags int) []byte {
	// This legacy function does not support checksum calculation with IP context
	// It returns a raw header without checksum
	h, _ := BuildTCPHeaderWithChecksum(nil, nil, srcPort, dstPort, seq, ack, flags, 65535, 0, nil)
	return h
}
