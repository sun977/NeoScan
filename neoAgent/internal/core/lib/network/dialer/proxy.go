package dialer

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// ProxyDialer 代理拨号器 (支持 SOCKS5)
type ProxyDialer struct {
	ProxyURL *url.URL
	Timeout  time.Duration
	forward  proxy.Dialer
}

func NewProxyDialer(proxyAddr string, timeout time.Duration) (*ProxyDialer, error) {
	u, err := url.Parse(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy address: %v", err)
	}

	var forward proxy.Dialer = proxy.Direct
	
	// 如果是 SOCKS5 代理
	if u.Scheme == "socks5" {
		var auth *proxy.Auth
		if u.User != nil {
			auth = &proxy.Auth{
				User: u.User.Username(),
			}
			if p, ok := u.User.Password(); ok {
				auth.Password = p
			}
		}
		
		forward, err = proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create socks5 dialer: %v", err)
		}
	} else {
		// 暂时只支持 SOCKS5，HTTP 代理需要 CONNECT 方法支持 (net/http 有，但 raw tcp 需要自己实现)
		// 这里先占位，或者直接报错
		return nil, fmt.Errorf("unsupported proxy scheme: %s (only socks5 is supported for raw tcp)", u.Scheme)
	}

	return &ProxyDialer{
		ProxyURL: u,
		Timeout:  timeout,
		forward:  forward,
	}, nil
}

func (d *ProxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	// golang.org/x/net/proxy 只有 Dial 接口，没有 DialContext
	// 我们可以在外层包装一个带超时的 Dial
	// 或者使用 context aware 的 proxy 库 (比较少见)
	
	// 这里使用简单的 goroutine + select 来模拟 context timeout
	type dialResult struct {
		Conn net.Conn
		Err  error
	}
	
	ch := make(chan dialResult, 1)
	
	go func() {
		// 注意: proxy.Dialer 接口通常没有 Timeout 参数，它依赖底层的 net.Dialer
		// 但这里的 d.forward 是通过 proxy.SOCKS5 创建的，它默认使用直连作为 forwarder
		// 我们很难直接控制连接超时，除非 proxy 库暴露了
		
		// 暂时直接调用 Dial
		conn, err := d.forward.Dial(network, address)
		ch <- dialResult{Conn: conn, Err: err}
	}()
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.Conn, res.Err
	}
}
