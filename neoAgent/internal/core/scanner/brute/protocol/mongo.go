package protocol

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"neoagent/internal/core/scanner/brute"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoCracker MongoDB 协议爆破器
type MongoCracker struct{}

func NewMongoCracker() *MongoCracker {
	return &MongoCracker{}
}

func (c *MongoCracker) Name() string {
	return "mongo"
}

func (c *MongoCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

func (c *MongoCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// 构建连接 URI
	// 注意：必须对用户名和密码进行 URL 编码，防止特殊字符导致 URI 解析失败
	u := &url.URL{
		Scheme: "mongodb",
		User:   url.UserPassword(auth.Username, auth.Password),
		Host:   fmt.Sprintf("%s:%d", host, port),
		// 可以在这里添加 query params，如 authSource=admin
		// 默认通常是对 admin 库进行认证
	}
	// 添加连接超时参数
	q := u.Query()
	q.Set("connectTimeoutMS", "3000") // 3秒连接超时
	u.RawQuery = q.Encode()

	uri := u.String()

	// 创建 client options
	clientOptions := options.Client().ApplyURI(uri)

	// 创建连接
	// 注意：mongo.Connect 只是创建客户端并开始后台监控，不一定立即发起 IO
	// 真正的连接检查通常在 Ping 时发生
	// 但为了确保鉴权，我们需要发起一个操作

	// 使用带有超时的 ctx，虽然 URI 里设置了 connectTimeoutMS，但为了保险起见，双重保障
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, clientOptions)
	if err != nil {
		// Connect 错误通常是配置错误，但也可能是早期的连接失败
		return false, brute.ErrConnectionFailed
	}
	defer func() {
		// 务必断开连接，释放资源
		_ = client.Disconnect(context.Background())
	}()

	// 执行 Ping 来触发连接和鉴权
	// Ping 会等待服务器选择，如果鉴权失败，这里应该会报错
	if err := client.Ping(connectCtx, readpref.Primary()); err != nil {
		return c.handleError(err)
	}

	return true, nil
}

// handleError 解析 MongoDB 错误
func (c *MongoCracker) handleError(err error) (bool, error) {
	if err == nil {
		return true, nil
	}

	errMsg := err.Error()

	// 鉴权失败
	// 错误信息通常包含 "Authentication failed" (code 18)
	// 或者 "command auth failed"
	if strings.Contains(errMsg, "Authentication failed") ||
		strings.Contains(errMsg, "auth failed") ||
		strings.Contains(errMsg, "sasl conversation error") {
		return false, nil // 密码错误
	}

	// 网络错误/连接超时
	// "server selection error": 找不到服务器
	// "context deadline exceeded"
	// "connection refused"
	// "i/o timeout"
	if strings.Contains(errMsg, "server selection error") ||
		strings.Contains(errMsg, "context deadline exceeded") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "no reachable servers") {
		return false, brute.ErrConnectionFailed
	}

	// 其他错误，作为网络错误处理，或者记录日志
	// 无法确定是否是鉴权失败时，宁可认为是网络错误，避免误报
	return false, brute.ErrConnectionFailed
}
