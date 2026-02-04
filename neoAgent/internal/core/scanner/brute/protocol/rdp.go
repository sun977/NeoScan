package protocol

import (
	"context"
	"fmt"

	"neoagent/internal/core/scanner/brute"
	grdp "neoagent/internal/core/scanner/brute/protocol/rdp"
)

// RDPCracker RDP 协议爆破器
//
// 实现原理:
// 基于移植自 sbscan/grdp 的 RDP 协议栈。
// 支持 Standard RDP Security 和 NLA (CredSSP/NTLMv2) 认证。
// 移除了不必要的图形化处理逻辑，仅保留认证核心。
type RDPCracker struct{}

func NewRDPCracker() *RDPCracker {
	return &RDPCracker{}
}

func (c *RDPCracker) Name() string {
	return "rdp"
}

func (c *RDPCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

func (c *RDPCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// RDP 协议比较复杂，连接建立可能较慢
	// grdp 库内部使用 net.DialTimeout，这里我们要做一层 context 适配
	// 但 grdp.Login 接口是阻塞的，且内部硬编码了 3s 超时
	// 我们可以通过 goroutine + select 来实现 context 控制

	target := fmt.Sprintf("%s:%d", host, port)
	domain := "" // 默认域为空，或者可以从 auth.Other["domain"] 获取

	type result struct {
		success bool
		err     error
	}
	ch := make(chan result, 1)

	go func() {
		// 调用 grdp 的 Login 接口
		// grdp.Login 会自动尝试 SSL (NLA) 和 RDP (Standard) 两种模式
		err := grdp.Login(target, domain, auth.Username, auth.Password)
		if err == nil {
			ch <- result{success: true, err: nil}
		} else {
			// 区分认证失败和连接错误
			// grdp 的错误处理比较粗糙，通常返回自定义 error
			// 如果是 "login failed"，说明连接成功但认证失败
			// 如果是 "[dial err]"，说明网络问题
			errMsg := err.Error()
			if errMsg == "login failed" || errMsg == "PROTOCOL_RDP" {
				ch <- result{success: false, err: nil}
			} else {
				// 视为连接错误
				ch <- result{success: false, err: brute.ErrConnectionFailed}
			}
		}
	}()

	select {
	case res := <-ch:
		return res.success, res.err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
