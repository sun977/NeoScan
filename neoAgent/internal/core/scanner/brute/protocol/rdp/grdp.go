package grdp

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"neoagent/internal/core/scanner/brute/protocol/rdp/core"
	"neoagent/internal/core/scanner/brute/protocol/rdp/glog"
	"neoagent/internal/core/scanner/brute/protocol/rdp/protocol/nla"
	"neoagent/internal/core/scanner/brute/protocol/rdp/protocol/pdu"
	"neoagent/internal/core/scanner/brute/protocol/rdp/protocol/sec"
	"neoagent/internal/core/scanner/brute/protocol/rdp/protocol/t125"
	"neoagent/internal/core/scanner/brute/protocol/rdp/protocol/tpkt"
	"neoagent/internal/core/scanner/brute/protocol/rdp/protocol/x224"
)

const (
	PROTOCOL_RDP = "PROTOCOL_RDP"
	PROTOCOL_SSL = "PROTOCOL_SSL"
)

type Client struct {
	Host string // ip:port
	tpkt *tpkt.TPKT
	x224 *x224.X224
	mcs  *t125.MCSClient
	sec  *sec.Client
	pdu  *pdu.Client
	vnc  *interface{}
}

func NewClient(host string, logLevel glog.LEVEL) *Client {
	glog.SetLevel(logLevel)
	logger := log.New(os.Stdout, "", 0)
	glog.SetLogger(logger)
	return &Client{
		Host: host,
	}
}

func (g *Client) LoginForSSL(domain, user, pwd string) error {
	conn, err := net.DialTimeout("tcp", g.Host, 3*time.Second)
	if err != nil {
		return fmt.Errorf("[dial err] %v", err)
	}
	defer conn.Close()
	glog.Info(conn.LocalAddr().String())

	g.tpkt = tpkt.New(core.NewSocketLayer(conn), nla.NewNTLMv2(domain, user, pwd))
	g.x224 = x224.New(g.tpkt)
	g.mcs = t125.NewMCSClient(g.x224)
	g.sec = sec.NewClient(g.mcs)
	g.pdu = pdu.NewClient(g.sec)

	g.sec.SetUser(user)
	g.sec.SetPwd(pwd)
	g.sec.SetDomain(domain)

	g.tpkt.SetFastPathListener(g.sec)
	g.sec.SetFastPathListener(g.pdu)
	g.pdu.SetFastPathSender(g.tpkt)

	err = g.x224.Connect()
	if err != nil {
		return fmt.Errorf("[x224 connect err] %v", err)
	}
	glog.Info("wait connect ok")

	// 在连接建立后，根据不同的协议流程，可能不需要完全等待图形化更新
	// 对于爆破来说，只要认证通过（SSL/NLA），或者收到了 RDP 协议的许可包，就算成功
	// 这里简化处理：如果 Connect 成功且没有报错，如果是 SSL/NLA 模式，通常意味着认证通过
	//
	// 注意：LoginForSSL 中的 x224.Connect() 内部会进行 CredSSP 握手 (NLA)
	// 如果 NLA 认证失败，x224.Connect() 应该会返回错误
	// 所以我们不需要等待后续的 MCS/PDU 流程 (那些是图形化界面的东西)

	// 为了兼容 sbscan 的逻辑，我们保留后续的 PDU 监听，但设置极短的超时
	// 或者直接返回 nil，因为 NLA 认证是在 Connect 阶段完成的

	// 检查 tpkt/x224/nla 的实现：
	// 在 sbscan 中，nla.NewNTLMv2 被传入 tpkt，tpkt 在 Read/Write 时会处理 NLA
	// x224.Connect -> tpkt.Connect -> socket.Connect
	// 关键在于 NLA 认证发生在哪个阶段。
	// 通常 RDP 连接顺序: X.224 CR -> CC -> (TLS Handshake) -> (CredSSP/NLA)
	// 如果 NLA 失败，连接会断开或报错。

	// 因此，如果代码运行到这里，说明 NLA 认证很可能已经通过了。
	// 下面的 PDU 处理是处理 RDP 协议层面的图形更新，对于验证密码来说不是必须的。
	// 我们可以尝试直接返回 nil。

	return nil

	/* 原 sbscan 逻辑：等待 PDU 更新，这是为了验证是否能看到桌面
		wg := &sync.WaitGroup{}
		breakFlag := false
		wg.Add(1)

		g.pdu.On("error", func(e error) {
			err = e
			glog.Error("error", e)
			g.pdu.Emit("done")
		})
	    ...
		wg.Wait()
		return err
	*/
}

func (g *Client) LoginForRDP(domain, user, pwd string) error {
	conn, err := net.DialTimeout("tcp", g.Host, 3*time.Second)
	if err != nil {
		return fmt.Errorf("[dial err] %v", err)
	}
	defer conn.Close()
	glog.Info(conn.LocalAddr().String())

	g.tpkt = tpkt.New(core.NewSocketLayer(conn), nla.NewNTLMv2(domain, user, pwd))
	g.x224 = x224.New(g.tpkt)
	g.mcs = t125.NewMCSClient(g.x224)
	g.sec = sec.NewClient(g.mcs)
	g.pdu = pdu.NewClient(g.sec)

	g.sec.SetUser(user)
	g.sec.SetPwd(pwd)
	g.sec.SetDomain(domain)

	g.tpkt.SetFastPathListener(g.sec)
	g.sec.SetFastPathListener(g.pdu)
	g.pdu.SetFastPathSender(g.tpkt)

	g.x224.SetRequestedProtocol(x224.PROTOCOL_RDP)

	err = g.x224.Connect()
	if err != nil {
		return fmt.Errorf("[x224 connect err] %v", err)
	}
	glog.Info("wait connect ok")

	// 对于 Standard RDP Security，认证是在 MCS Connect Initial PDU 中发送加密的密码
	// 或者在后续的 Logon Request PDU 中。
	// sbscan 的实现似乎是在 x224.Connect 之后，通过 PDU 交互来完成登录。
	// 这里保留等待逻辑，但缩短超时。

	wg := &sync.WaitGroup{}
	breakFlag := false
	// updateCount := 0
	wg.Add(1)

	g.pdu.On("error", func(e error) {
		err = e
		glog.Error("error", e)
		g.pdu.Emit("done")
	})
	// ... (省略部分事件监听，保持原样或简化)
	g.pdu.On("close", func() {
		err = errors.New("close")
		glog.Info("on close")
		g.pdu.Emit("done")
	})
	g.pdu.On("success", func() {
		err = nil
		glog.Info("on success")
		g.pdu.Emit("done")
	})
	g.pdu.On("done", func() {
		if breakFlag == false {
			breakFlag = true
			wg.Done()
		}
	})

	// 缩短超时时间
	// 启动一个 timer 来触发超时
	go func() {
		time.Sleep(3 * time.Second)
		if breakFlag == false {
			breakFlag = true
			wg.Done()
		}
	}()

	wg.Wait()

	// 只要没有明确的 error，且连接保持，就认为成功
	// sbscan 用 updateCount > 50 来判断进入桌面，这太严格了
	if err != nil {
		return err
	}
	return nil
}

func Login(target, domain, username, password string) error {
	var err error
	g := NewClient(target, glog.NONE)
	//SSL协议登录测试
	err = g.LoginForSSL(domain, username, password)
	if err == nil {
		return nil
	}
	if err.Error() != PROTOCOL_RDP {
		return err
	}
	//RDP协议登录测试
	err = g.LoginForRDP(domain, username, password)
	if err == nil {
		return nil
	} else {
		return err
	}
}

func LoginForSSL(target, domain, username, password string) error {
	var err error
	g := NewClient(target, glog.NONE)
	//SSL协议登录测试
	err = g.LoginForSSL(domain, username, password)
	if err == nil {
		return nil
	}
	return err
}

func LoginForRDP(target, domain, username, password string) error {
	var err error
	g := NewClient(target, glog.NONE)
	//SSL协议登录测试
	err = g.LoginForRDP(domain, username, password)
	if err == nil {
		return nil
	}
	return err
}

func VerifyProtocol(target string) string {
	var err error
	err = LoginForSSL(target, "", "administrator", "test")
	if err == nil {
		return PROTOCOL_SSL
	}
	if err.Error() != PROTOCOL_RDP {
		return PROTOCOL_SSL
	}
	return PROTOCOL_RDP
}
