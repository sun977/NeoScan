package protocol

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"neoagent/internal/core/scanner/brute"

	_ "github.com/sijms/go-ora/v2"
)

// OracleSIDCracker Oracle SID 爆破器
// 专门用于枚举 Oracle 的 SID
type OracleSIDCracker struct {
	// commonSIDs 列表可以复用 OracleCracker 中的，或者在这里定义
	// 暂时在这里不定义，因为 TaskTypeOracleSID 模式下，
	// Runner 会将 SID 字典作为 User/Pass 字典传入，或者作为单独的参数
	// 为了简化，我们假设 Scanner 会把 SID 字典传给 Check 的 auth.Other["sid"] 或者 auth.Username
}

func NewOracleSIDCracker() *OracleSIDCracker {
	return &OracleSIDCracker{}
}

func (c *OracleSIDCracker) Name() string {
	return "oracle-sid"
}

// Mode 返回 AuthModeUserPass，虽然我们只爆破 SID
// 但为了复用 Runner 的逻辑，我们可以把 SID 当作 Username 传入
func (c *OracleSIDCracker) Mode() brute.AuthMode {
	// 这里比较特殊，我们只需要一个参数(SID)
	// 可以使用 AuthModeUserPass，然后忽略 Password，把 SID 放在 Username
	return brute.AuthModeUserPass
}

func (c *OracleSIDCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// 在 oracle-sid 模式下，auth.Username 被视为待测试的 SID
	sid := auth.Username
	if sid == "" {
		return false, nil
	}

	// 尝试连接
	return c.checkSID(ctx, host, port, sid)
}

func (c *OracleSIDCracker) checkSID(ctx context.Context, host string, port int, sid string) (bool, error) {
	// 使用一个极其不可能存在的用户进行连接
	// 如果 SID 存在，Oracle 会校验用户名密码，返回 ORA-01017 (无效用户名/密码)
	// 如果 SID 不存在，Oracle 会返回 ORA-12505 (监听器不知道该 SID)

	probeUser := "neo_sid_probe_user_impossible_exist"
	probePass := "neo_sid_probe_pass_impossible_exist"

	connURL := fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		url.QueryEscape(probeUser),
		url.QueryEscape(probePass),
		host, port, sid)

	if strings.Contains(connURL, "?") {
		connURL += "&CONNECTION TIMEOUT=5000"
	} else {
		connURL += "?CONNECTION TIMEOUT=5000"
	}

	db, err := sql.Open("oracle", connURL)
	if err != nil {
		return false, fmt.Errorf("invalid dsn: %w", err)
	}
	defer db.Close()

	// 强制设置连接参数
	db.SetConnMaxLifetime(5 * time.Second)
	db.SetMaxIdleConns(0)

	// PingContext 检测
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		errMsg := err.Error()
		
		// 关键判定逻辑
		if strings.Contains(errMsg, "ORA-01017") || 
		   strings.Contains(errMsg, "logon denied") || 
		   strings.Contains(errMsg, "ORA-28000") || // 账户锁定
		   strings.Contains(errMsg, "ORA-28001") {  // 密码过期
			// 鉴权相关错误，说明 SID 是有效的！
			// 返回 true 表示"爆破成功"(找到了 SID)
			return true, nil
		}

		// ORA-12505: TNS:listener does not currently know of SID given in connect descriptor
		if strings.Contains(errMsg, "ORA-12505") || strings.Contains(errMsg, "ORA-12514") {
			// SID 无效
			return false, nil
		}
		
		// 其他连接错误 (超时等)
		if strings.Contains(errMsg, "connection refused") ||
		   strings.Contains(errMsg, "i/o timeout") ||
		   strings.Contains(errMsg, "context deadline exceeded") {
			return false, brute.ErrConnectionFailed
		}
		
		// 其他未知错误，暂时认为失败
		return false, nil
	}

	// 如果竟然连接成功了 (极低概率，除非我们的 probeUser 真的存在且密码正确)
	return true, nil
}
