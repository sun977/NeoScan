//go:build linux

package os

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"time"

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

	// 2. 目标解析
	dstIP := net.ParseIP(target)
	if dstIP == nil {
		return nil, fmt.Errorf("invalid target ip: %s", target)
	}

	// 3. 寻找开放和关闭的端口
	// Nmap OS 探测依赖一个开放端口和一个关闭端口
	// 这里我们尝试快速寻找
	openPort := e.findOpenPort(target)
	if openPort == 0 {
		return nil, fmt.Errorf("no open port found for fingerprinting")
	}
	closedPort := 43210 // 假设这个端口关闭 (通常高端口是关闭的)
	// 严谨的做法是也探测一下 closedPort，确保它回复 RST

	// 4. 执行全量探测
	fp, err := e.executeProbes(ctx, target, openPort, closedPort)
	if err != nil {
		return nil, fmt.Errorf("probe execution failed: %v", err)
	}

	// 5. 匹配指纹
	matchResult := e.db.Match(fp)

	// 6. 构造结果
	info := &OsInfo{
		Source: "NmapStack",
	}

	// 格式化 fingerprint 字符串用于调试或报告
	// 简化显示 T1
	if t1, ok := fp.MatchRule["T1"]; ok {
		info.Fingerprint = "T1(" + t1 + ")"
	} else {
		info.Fingerprint = "Incomplete"
	}

	if matchResult != nil {
		info.Name = matchResult.Fingerprint.Name
		info.Accuracy = int(matchResult.Accuracy)
		info.Family = matchResult.Fingerprint.Class
	} else {
		info.Name = "Unknown"
		info.Accuracy = 0
	}

	return info, nil
}

// findOpenPort 尝试寻找一个开放的 TCP 端口
func (e *NmapStackEngine) findOpenPort(target string) int {
	// 尝试常见端口
	commonPorts := []int{80, 443, 22, 445, 3389, 8080}
	for _, port := range commonPorts {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", target, port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return port
		}
	}
	return 0
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
type NmapServiceEngine struct{}

func NewNmapServiceEngine() *NmapServiceEngine {
	return &NmapServiceEngine{}
}

func (e *NmapServiceEngine) Name() string {
	return "nmap_service"
}

func (e *NmapServiceEngine) Scan(ctx context.Context, target string) (*OsInfo, error) {
	return &OsInfo{
		Name:     "Inferred from Service (Not Implemented Standalone)",
		Accuracy: 0,
	}, nil
}
