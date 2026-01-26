package netraw

import (
	"encoding/binary"
	"net"

	"golang.org/x/net/ipv4"
)

// BuildIPv4Packet 构建 IPv4 头部和负载
// 仅用于 Raw Socket 场景
func BuildIPv4Packet(src, dst net.IP, protocol int, payload []byte) ([]byte, error) {
	header := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TotalLen: ipv4.HeaderLen + len(payload),
		ID:       12345, // TODO: 随机化
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

// Checksum 计算 TCP/IP 校验和
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

	sum += (sum >> 16)
	return uint16(^sum)
}

// BuildTCPHeader 构建 TCP 头部 (简化版)
// 注意：实际使用时需要填充更多字段和伪首部校验和
func BuildTCPHeader(srcPort, dstPort int, seq, ack uint32, flags int) []byte {
	// 20 bytes TCP Header
	h := make([]byte, 20)

	binary.BigEndian.PutUint16(h[0:], uint16(srcPort))
	binary.BigEndian.PutUint16(h[2:], uint16(dstPort))
	binary.BigEndian.PutUint32(h[4:], seq)
	binary.BigEndian.PutUint32(h[8:], ack)

	// Data Offset (4 bits) + Reserved (3 bits) + Flags (9 bits)
	// Offset = 5 (20 bytes)
	// h[12] = (Data Offset << 4) | (Reserved << 1) | (NS)
	h[12] = 0x50
	// h[13] = Flags
	h[13] = byte(flags)

	binary.BigEndian.PutUint16(h[14:], 65535) // Window Size
	// Checksum at h[16], Urgent Pointer at h[18]

	return h
}
