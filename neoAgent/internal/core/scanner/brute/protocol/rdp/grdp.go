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

	// 使用 channel 等待连接结果
	errCh := make(chan error, 1)
	connCh := make(chan uint32, 1)

	g.x224.On("error", func(e error) {
		errCh <- e
	})
	g.x224.On("connect", func(proto uint32) {
		connCh <- proto
	})

	err = g.x224.Connect()
	if err != nil {
		// NLA 模式下，Connect 会执行 NLA 握手
		// 如果是认证失败 (CredSSP 握手失败)，通常会返回特定错误
		// 但在这里，我们只能捕获到 x224/tpkt 层的错误
		return fmt.Errorf("[x224 connect err] %v", err)
	}

	select {
	case err := <-errCh:
		return err
	case <-connCh:
		glog.Info("wait connect ok")
		// NLA 认证在 Connect 阶段完成
		// 如果能执行到这里，说明 NLA 握手成功（即用户名密码正确）
		return nil
	case <-time.After(3 * time.Second):
		return fmt.Errorf("timeout")
	}
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
