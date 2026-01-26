//go:build !linux

package os

import (
	"context"
	"fmt"
)

// NmapStackEngine 基于 Nmap OS DB 的 TCP/IP 协议栈指纹识别 (Stub for non-Linux)
type NmapStackEngine struct{}

func NewNmapStackEngine() *NmapStackEngine {
	return &NmapStackEngine{}
}

func (e *NmapStackEngine) Name() string {
	return "nmap_stack"
}

func (e *NmapStackEngine) Scan(ctx context.Context, target string) (*OsInfo, error) {
	return nil, fmt.Errorf("nmap stack fingerprinting is only supported on linux")
}

// NmapServiceEngine 基于 PortServiceScanner 结果的 OS 识别 (Stub for non-Linux)
type NmapServiceEngine struct{}

func NewNmapServiceEngine() *NmapServiceEngine {
	return &NmapServiceEngine{}
}

func (e *NmapServiceEngine) Name() string {
	return "nmap_service"
}

func (e *NmapServiceEngine) Scan(ctx context.Context, target string) (*OsInfo, error) {
	return &OsInfo{
		Name:     "Inferred from Service (Not Implemented on this platform)",
		Accuracy: 0,
	}, nil
}
