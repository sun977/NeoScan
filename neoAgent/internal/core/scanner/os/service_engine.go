package os

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

// NmapServiceEngine 基于服务 Banner 的 OS 识别引擎
//
// 设计哲学: "Sniper vs Carpet Bombing"
// 该引擎用于 OS 扫描的原子能力建设 (Phase 3)，目标是独立、快速地推断 OS，
// 而不是依赖庞大的 PortServiceScanner 进行全量端口扫描。
//
// 核心职责:
// 1. 作为 "狙击手" (Sniper)，仅探测最关键的几个端口 (22, 80, 443 等)。
// 2. 通过 Banner 关键字 (如 "Microsoft-IIS", "Ubuntu") 快速识别 OS。
// 3. 为 OsScanner 提供独立的兜底能力，即使没有进行全量端口扫描也能获取 OS 信息。
//
// 未来演进 (Phase 4):
// 在全流程编排中，如果已经运行了 PortServiceScanner，本引擎应支持复用其结果，
// 避免重复探测。
type NmapServiceEngine struct {
	targetPorts []int
}

func NewNmapServiceEngine() *NmapServiceEngine {
	return &NmapServiceEngine{
		// 优先探测最可能暴露 OS 信息的端口
		targetPorts: []int{22, 80, 443, 21, 8080},
	}
}

func (e *NmapServiceEngine) Name() string {
	return "nmap_service"
}

func (e *NmapServiceEngine) Scan(ctx context.Context, target string) (*OsInfo, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var bestInfo *OsInfo

	// 并发探测端口
	for _, port := range e.targetPorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			info := e.probePort(ctx, target, p)
			if info != nil {
				mu.Lock()
				defer mu.Unlock()
				// 简单的择优逻辑：取准确度最高的
				if bestInfo == nil || info.Accuracy > bestInfo.Accuracy {
					bestInfo = info
				}
			}
		}(port)
	}

	wg.Wait()

	if bestInfo == nil {
		return nil, fmt.Errorf("no os info inferred from services")
	}

	return bestInfo, nil
}

func (e *NmapServiceEngine) probePort(ctx context.Context, target string, port int) *OsInfo {
	address := fmt.Sprintf("%s:%d", target, port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return nil
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	var banner string

	// 针对不同端口发送不同的探测包
	switch port {
	case 80, 8080, 443:
		// 发送 HTTP 请求
		req := "HEAD / HTTP/1.0\r\n\r\n"
		_, err = conn.Write([]byte(req))
		if err == nil {
			reader := bufio.NewReader(conn)
			// 读取前几行寻找 Server 头
			for i := 0; i < 10; i++ {
				line, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				if strings.HasPrefix(line, "Server:") {
					banner = strings.TrimSpace(strings.TrimPrefix(line, "Server:"))
					break
				}
			}
		}
	default:
		// 默认读取 Banner (SSH, FTP 等通常连接后直接发送 Banner)
		// 稍微等待一下服务器发送数据
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err == nil {
			banner = strings.TrimSpace(line)
		}
	}

	if banner == "" {
		return nil
	}

	return e.matchBanner(banner, port)
}

func (e *NmapServiceEngine) matchBanner(banner string, port int) *OsInfo {
	// 简单的正则匹配规则库
	// 实际生产中应该加载外部规则文件

	// 1. Windows 特征
	if strings.Contains(banner, "Microsoft-IIS") {
		return &OsInfo{
			Name:           "Windows",
			Family:         "Windows",
			Accuracy:       90,
			Fingerprint:    fmt.Sprintf("Service: Port %d", port),
			RawFingerprint: fmt.Sprintf("Service: Port %d, Banner: %s", port, banner),
			Source:         "Service",
		}
	}
	if strings.Contains(banner, "Microsoft FTP") {
		return &OsInfo{
			Name:           "Windows",
			Family:         "Windows",
			Accuracy:       90,
			Fingerprint:    fmt.Sprintf("Service: Port %d", port),
			RawFingerprint: fmt.Sprintf("Service: Port %d, Banner: %s", port, banner),
			Source:         "Service",
		}
	}

	// 2. Linux/Unix 特征
	if strings.Contains(banner, "Ubuntu") {
		return &OsInfo{
			Name:           "Linux (Ubuntu)",
			Family:         "Linux",
			Accuracy:       95,
			Fingerprint:    fmt.Sprintf("Service: Port %d", port),
			RawFingerprint: fmt.Sprintf("Service: Port %d, Banner: %s", port, banner),
			Source:         "Service",
		}
	}
	if strings.Contains(banner, "Debian") {
		return &OsInfo{
			Name:           "Linux (Debian)",
			Family:         "Linux",
			Accuracy:       95,
			Fingerprint:    fmt.Sprintf("Service: Port %d", port),
			RawFingerprint: fmt.Sprintf("Service: Port %d, Banner: %s", port, banner),
			Source:         "Service",
		}
	}
	if strings.Contains(banner, "CentOS") {
		return &OsInfo{
			Name:           "Linux (CentOS)",
			Family:         "Linux",
			Accuracy:       95,
			Fingerprint:    fmt.Sprintf("Service: Port %d", port),
			RawFingerprint: fmt.Sprintf("Service: Port %d, Banner: %s", port, banner),
			Source:         "Service",
		}
	}
	if strings.Contains(banner, "FreeBSD") {
		return &OsInfo{
			Name:           "FreeBSD",
			Family:         "FreeBSD",
			Accuracy:       95,
			Fingerprint:    fmt.Sprintf("Service: Port %d", port),
			RawFingerprint: fmt.Sprintf("Service: Port %d, Banner: %s", port, banner),
			Source:         "Service",
		}
	}

	// Red Hat / RHEL / EL 特征
	if strings.Contains(banner, "Red Hat") || strings.Contains(banner, "RHEL") || strings.Contains(banner, ".el") {
		return &OsInfo{
			Name:           "Linux (Red Hat/CentOS)",
			Family:         "Linux",
			Accuracy:       95,
			Fingerprint:    fmt.Sprintf("Service: Port %d", port),
			RawFingerprint: fmt.Sprintf("Service: Port %d, Banner: %s", port, banner),
			Source:         "Service",
		}
	}

	// 3. 通用 SSH 特征
	if strings.Contains(banner, "OpenSSH") {
		// Windows 上的 OpenSSH 通常包含 "Windows" 字样
		if strings.Contains(banner, "Windows") {
			return &OsInfo{
				Name:           "Windows (OpenSSH)",
				Family:         "Windows",
				Accuracy:       90,
				Fingerprint:    fmt.Sprintf("Service: Port %d", port),
				RawFingerprint: fmt.Sprintf("Service: Port %d, Banner: %s", port, banner),
				Source:         "Service",
			}
		}

		// 否则大概率是 Linux/Unix
		// 提高置信度到 85，以覆盖 TTL (80) 的结果
		return &OsInfo{
			Name:           "Linux/Unix (OpenSSH)",
			Family:         "Unix",
			Accuracy:       85,
			Fingerprint:    fmt.Sprintf("Service: Port %d", port),
			RawFingerprint: fmt.Sprintf("Service: Port %d, Banner: %s", port, banner),
			Source:         "Service",
		}
	}

	return nil
}

// 辅助函数: 正则匹配
func matchRegex(pattern, text string) bool {
	matched, _ := regexp.MatchString(pattern, text)
	return matched
}
