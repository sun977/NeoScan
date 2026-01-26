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
