package protocol

import (
	"context"
	"time"

	"neoagent/internal/core/scanner/brute"

	"github.com/gosnmp/gosnmp"
)

// SNMPCracker SNMP 协议爆破器
//
// 检测原理:
// 1. SNMP 是基于 UDP 的协议，因此没有建立连接的概念。
// 2. 验证机制是 "Community String" (团体名)，类似于密码。
// 3. 探测方式: 发送一个简单的 GetRequest 查询系统描述 (sysDescr, OID: 1.3.6.1.2.1.1.1.0)。
// 4. 结果判定:
//    - 收到响应且无错误 -> 团体名正确。
//    - 超时 -> 团体名错误 (或网络不通)。
// 5. 版本策略: 优先尝试 v2c，这是目前最通用的版本。
type SNMPCracker struct{}

func NewSNMPCracker() *SNMPCracker {
	return &SNMPCracker{}
}

func (c *SNMPCracker) Name() string {
	return "snmp"
}

func (c *SNMPCracker) Mode() brute.AuthMode {
	// SNMP 只需要 Community String，我们将其映射为 Password
	// Username 字段会被忽略
	return brute.AuthModeOnlyPass
}

func (c *SNMPCracker) Check(ctx context.Context, host string, port int, auth brute.Auth) (bool, error) {
	// 1. 初始化 GoSNMP 客户端
	// 注意: GoSNMP 不是并发安全的，每次都需要新建实例
	params := &gosnmp.GoSNMP{
		Target:    host,
		Port:      uint16(port),
		Community: auth.Password,      // 团体名作为密码传入
		Version:   gosnmp.Version2c,   // 默认使用 v2c，覆盖面最广
		Timeout:   2 * time.Second,    // 严格超时，UDP 丢包或认证失败表现一致
		Retries:   0,                  // 不重试，或者仅重试一次。为了速度，这里设为0，依靠上层调度或接受少量漏报
		Transport: "udp",
	}

	// 2. 建立连接 (UDP 只是初始化 socket，不会产生网络流量)
	if err := params.Connect(); err != nil {
		return false, brute.ErrConnectionFailed
	}
	defer params.Conn.Close()

	// 3. 发送 GetRequest 查询 sysDescr
	// OID: 1.3.6.1.2.1.1.1.0 (System Description)
	// 这是一个标准的 MIB-II 对象，几乎所有 SNMP 代理都支持
	oids := []string{"1.3.6.1.2.1.1.1.0"}
	
	// 使用带 Context 的调用 (如果库支持的话，GoSNMP 的 Get 也是同步阻塞的)
	// 这里我们需要用一个 goroutine + select 来实现 context 控制，或者直接依赖库的 Timeout
	// 简单起见，直接调用 Get，依赖 params.Timeout
	
	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	result, err := params.Get(oids)
	if err != nil {
		// 区分超时和其他错误
		// 但对于 UDP 爆破，任何错误通常都意味着认证失败（或网络不可达）
		// 我们无法区分 "Community 错误导致的丢包" 和 "网络导致的丢包"
		// 所以统一视为验证失败
		return false, nil
	}

	// 4. 判定结果
	// 如果返回结果非空，且 Error 为 NoError，则认为认证成功
	if result != nil && result.Error == gosnmp.NoError {
		// 进一步防御性检查: 确保返回了变量绑定
		if len(result.Variables) > 0 {
			return true, nil
		}
	}

	return false, nil
}
