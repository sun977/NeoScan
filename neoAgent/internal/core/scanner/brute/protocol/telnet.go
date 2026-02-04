package protocol

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"neoagent/internal/core/scanner/brute"

	"github.com/ziutek/telnet"
)

// TelnetCracker Telnet 协议爆破器
//
// 检测原理:
// 1. 状态机交互: 建立 TCP 连接后，通过状态机依次处理 "用户名提示" -> "发送用户名" -> "密码提示" -> "发送密码" -> "结果判定"。
// 2. 正则匹配: 使用正则表达式灵活匹配各种设备千奇百怪的登录提示符 (如 "Login:", "User Name:", "Password:", ">", "#" 等)。
// 3. 结果判定:
//   - 成功: 匹配到 Shell 提示符 (如 "# ", "$ ", "> ")，表明已获得交互权限。
//   - 失败: 匹配到明确的失败关键词 (如 "Login incorrect", "Access denied", "Login failed")，或再次出现 "Login:" 提示。
//
// 4. 超时控制: 每一步读取操作都有严格的超时限制，防止被非标准设备挂起。
type TelnetCracker struct {
	// 预编译正则，提高性能
	reLogin    *regexp.Regexp
	rePassword *regexp.Regexp
	reShell    *regexp.Regexp
	reFail     *regexp.Regexp
}

func NewTelnetCracker() *TelnetCracker {
	// 常见的登录提示符
	// login:, Login:, User Name:, Username:, user:
	reLogin := regexp.MustCompile(`(?i)(login|user\s*name|username|user)[\s:]*$`)

	// 常见的密码提示符
	// Password:, password:, Pass:, pass:
	rePassword := regexp.MustCompile(`(?i)(password|pass)[\s:]*$`)

	// 常见的 Shell 提示符 (登录成功标志)
	// #, $, >, %, 且后面通常跟着空格或行尾
	// 排除一些干扰项，尽量精确
	reShell := regexp.MustCompile(`[#$>%]\s*$`)

	// 常见的失败提示
	// Login incorrect, Login failed, Access denied, Bad password, Authentication failed
	reFail := regexp.MustCompile(`(?i)(incorrect|failed|denied|bad|invalid)`)

	return &TelnetCracker{
		reLogin:    reLogin,
		rePassword: rePassword,
		reShell:    reShell,
		reFail:     reFail,
	}
}

func (c *TelnetCracker) Name() string {
	return "telnet"
}

func (c *TelnetCracker) Mode() brute.AuthMode {
	return brute.AuthModeUserPass
}

func (c *TelnetCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// Telnet 交互通常较慢，且 ziutek/telnet 库主要依赖 SetReadDeadline 控制超时
	// 我们在 Check 内部通过 context 检查来提前退出，但主要还是靠 socket 超时

	addr := fmt.Sprintf("%s:%d", host, port)

	// 建立连接
	// 使用 DialTimeout，且超时时间受 ctx 剩余时间约束
	dialTimeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		dialTimeout = time.Until(deadline)
		if dialTimeout <= 0 {
			return false, brute.ErrConnectionFailed
		}
	}

	conn, err := telnet.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return false, brute.ErrConnectionFailed
	}
	defer conn.Close()

	// 设置读写超时 (每一步交互的超时)
	// 2秒通常足够读取提示符，太短容易漏掉慢设备
	stepTimeout := 3 * time.Second
	conn.SetReadDeadline(time.Now().Add(stepTimeout))
	conn.SetWriteDeadline(time.Now().Add(stepTimeout))

	// Stage 1: 等待登录提示符
	// 跳过所有不可打印字符直到匹配 login 或 password (有些设备只要密码)
	// 或者直接跳过直到数据流静止

	// 读取直到匹配 Login 或 Password
	data, err := c.readUntilMatch(conn, c.reLogin, c.rePassword)
	if err != nil {
		// 读不到提示符，可能是非 Telnet 服务，或者不支持登录
		return false, brute.ErrConnectionFailed
	}

	// 如果直接匹配到 Password (AuthModeOnlyPass 场景，虽然我们申明了 AuthModeUserPass)
	// 但为了健壮性，我们可以处理这种情况
	isPasswordPrompt := c.rePassword.Match(data)

	if !isPasswordPrompt {
		// 匹配到了 Login 提示符，发送用户名
		if err1 := c.sendLine(conn, auth.Username); err1 != nil {
			return false, brute.ErrConnectionFailed
		}

		// Stage 2: 等待密码提示符
		conn.SetReadDeadline(time.Now().Add(stepTimeout))
		_, err = c.readUntilMatch(conn, c.rePassword)
		if err != nil {
			// 发送用户名后没等到密码提示，可能用户名错误直接被踢，或者不需要密码
			// 这里简单处理为失败
			return false, nil
		}
	}

	// Stage 3: 发送密码
	if err1 := c.sendLine(conn, auth.Password); err1 != nil {
		return false, brute.ErrConnectionFailed
	}

	// Stage 4: 判定结果
	// 读取响应，判断是 Shell 还是 Fail
	conn.SetReadDeadline(time.Now().Add(stepTimeout))

	// 读取一段数据进行判定
	// 这里不能简单 readUntil，因为我们不知道是成功还是失败
	// 我们读取直到超时或匹配到特征

	success, err := c.checkResult(conn)
	if err != nil {
		// 判定过程中出错 (如超时且未匹配到任何特征)
		// 这种情况比较暧昧，通常认为是失败
		return false, nil
	}

	return success, nil
}

// readUntilMatch 读取数据直到匹配任意一个正则，或者超时
func (c *TelnetCracker) readUntilMatch(conn *telnet.Conn, regexps ...*regexp.Regexp) ([]byte, error) {
	var buf []byte
	b := make([]byte, 1)

	for {
		n, err := conn.Read(b)
		if n > 0 {
			buf = append(buf, b[0])
			// 检查是否匹配任意正则
			// 为了性能，可以不必每字节都检查，但 Telnet 提示符通常很短
			for _, re := range regexps {
				if re.Match(buf) {
					return buf, nil
				}
			}
		}
		if err != nil {
			return buf, err
		}
	}
}

// sendLine 发送一行数据 (自动追加 \r\n)
func (c *TelnetCracker) sendLine(conn *telnet.Conn, msg string) error {
	buf := []byte(msg + "\r\n")
	_, err := conn.Write(buf)
	return err
}

// checkResult 读取后续数据并判定结果
func (c *TelnetCracker) checkResult(conn *telnet.Conn) (bool, error) {
	var buf []byte
	b := make([]byte, 256) // 缓冲区大一点

	// 循环读取，直到匹配成功/失败，或者超时
	for {
		n, err := conn.Read(b)
		if n > 0 {
			buf = append(buf, b[:n]...)

			// 检查失败特征
			if c.reFail.Match(buf) {
				return false, nil // 明确失败
			}

			// 检查再次出现 Login (也是失败)
			if c.reLogin.Match(buf) {
				return false, nil
			}

			// 检查成功特征 (Shell 提示符)
			if c.reShell.Match(buf) {
				return true, nil
			}
		}

		if err != nil {
			// 读完了或者超时了
			// 如果此时还没有匹配到任何特征，我们只能认为失败
			// 除非我们想做得更激进：如果没报错也没断开，就当成功？不，那样误报太多。
			return false, err
		}
	}
}
