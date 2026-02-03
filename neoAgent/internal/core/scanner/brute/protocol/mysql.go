package protocol

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"neoagent/internal/core/scanner/brute"

	"github.com/go-sql-driver/mysql"
)

// MySQLCracker 实现 MySQL 协议爆破
type MySQLCracker struct{}

// NewMySQLCracker 创建 MySQL 爆破器
func NewMySQLCracker() *MySQLCracker {
	return &MySQLCracker{}
}

// Name 返回协议名称
func (c *MySQLCracker) Name() string {
	return "mysql"
}

// Mode 返回爆破模式
func (c *MySQLCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

// Check 验证 MySQL 凭据
func (c *MySQLCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// 构建 DSN (Data Source Name)
	// 格式: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	// 设置连接超时和读取超时，快速失败
	// 不指定 dbname，连接默认库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=3s&readTimeout=3s",
		auth.Username, auth.Password, host, port)

	// 打开数据库连接 (此时不会真正连接，Ping 才会)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		// sql.Open 很少报错，除非驱动名不对
		return false, c.handleError(err)
	}
	defer db.Close()

	// 设置最大连接生命周期，避免连接泄漏
	db.SetConnMaxLifetime(time.Second * 5)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)

	// 使用 PingContext 进行验证
	// 注意: 这里的 timeout=3s 是 DSN 参数控制的 TCP 连接超时
	// ctx 控制的是 Ping 操作的整体超时
	if err := db.PingContext(ctx); err != nil {
		return false, c.handleError(err)
	}

	return true, nil
}

// handleError 将底层错误转换为标准错误
func (c *MySQLCracker) handleError(err error) error {
	if err == nil {
		return nil
	}

	// 尝试转换为 MySQL 驱动错误
	if driverErr, ok := err.(*mysql.MySQLError); ok {
		switch driverErr.Number {
		case 1045, 1044: // Access denied
			return nil // 认证失败，不是连接错误
		}
	}

	msg := strings.ToLower(err.Error())

	// 文本匹配兜底
	if strings.Contains(msg, "access denied") {
		return nil
	}

	// 连接/网络错误
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no route to host") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "bad connection") ||
		strings.Contains(msg, "driver: bad connection") ||
		strings.Contains(msg, "target machine actively refused") || // Windows
		err == mysql.ErrInvalidConn {
		return brute.ErrConnectionFailed
	}

	// 其他协议错误
	return brute.ErrProtocolError
}
