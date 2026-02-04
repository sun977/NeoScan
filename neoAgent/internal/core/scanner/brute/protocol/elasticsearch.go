package protocol

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"neoagent/internal/core/scanner/brute"
)

// ElasticsearchCracker Elasticsearch 协议爆破器
//
// 检测原理:
// 1. 利用 Elasticsearch 的 REST API 进行认证探测。
// 2. 目标端点: /_security/_authenticate (专门用于验证用户身份的 API)。
// 3. 结果判定:
//   - 200 OK: 认证成功。
//   - 401 Unauthorized: 认证失败。
//   - 其他状态码: 视为失败或服务异常。
//
// 4. 细节处理:
//   - 跳过 TLS 证书验证 (InsecureSkipVerify: true)，因为绝大多数内部 ES 使用自签名证书。
//   - 自动处理 HTTP/HTTPS 协议升级 (如果服务器要求 HTTPS)。
type ElasticsearchCracker struct{}

func NewElasticsearchCracker() *ElasticsearchCracker {
	return &ElasticsearchCracker{}
}

func (c *ElasticsearchCracker) Name() string {
	return "elasticsearch"
}

func (c *ElasticsearchCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

func (c *ElasticsearchCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// 默认尝试 HTTP
	// 实用主义: 不去探测端口是否是 SSL，直接发请求。
	// 如果服务器是 HTTPS，Go 的 http client 会报错，我们捕获错误重试 HTTPS。
	return c.checkProtocol(ctx, host, port, auth, "http")
}

func (c *ElasticsearchCracker) checkProtocol(ctx context.Context, host string, port int, auth brute.Auth, scheme string) (bool, error) {
	url := fmt.Sprintf("%s://%s:%d/_security/_authenticate", scheme, host, port)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth(auth.Username, auth.Password)

	// 配置 HTTP Client
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 忽略证书错误
			// 禁用 Keep-Alive，因为我们是爆破，不需要长连接，减少资源占用
			DisableKeepAlives: true,
		},
		// 设置超时，虽然 context 有超时，但 client 最好也有
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		// 如果是 HTTP 请求发到了 HTTPS 端口，client 会报错
		// 错误信息通常包含 "Client sent an HTTP request to an HTTPS server"
		// 此时我们应该尝试 HTTPS
		if scheme == "http" && isHttpsError(err) {
			return c.checkProtocol(ctx, host, port, auth, "https")
		}
		return false, brute.ErrConnectionFailed
	}
	defer resp.Body.Close()

	// 200 OK 表示认证成功
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	// 401 Unauthorized 表示认证失败
	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}

	// 其他状态码，也视为失败 (可能是服务没开启 Security 插件，或者 API 不存在)
	// 如果是 404，说明可能没装 x-pack security，这种情况下爆破无意义，或者需要换端点
	// 这里简单处理，非 200 即失败
	return false, nil
}

// isHttpsError 判断是否是因为 HTTP 请求发到了 HTTPS 端口导致的错误
func isHttpsError(err error) bool {
	if err == nil {
		return false
	}
	// errMsg := err.Error()
	// Go http client 的典型错误信息
	// "http: server gave HTTP response to HTTPS client" (反过来)
	// 或者服务器返回的 body 包含 "This combination of host and port requires TLS"
	// 但 client.Do 遇到协议不匹配通常会直接报错
	// 常见的错误特征: "malformed HTTP response", "oversized record received with length" (TLS record)
	// 为了简化，如果是连接错误，且原协议是 HTTP，我们都值得试一下 HTTPS (在 Check 中控制逻辑)

	// 这里我们放宽策略：只要是连接层面的错误，且当前是 HTTP，Check 函数逻辑允许重试 HTTPS
	// 但为了严谨，我们尽量匹配特征
	return true // 简单粗暴：HTTP 失败了就试 HTTPS，反正成本很低
}
