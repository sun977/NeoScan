package nmap

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestUserCustomRule 测试用户自定义的 Cool-admin 规则
func TestUserCustomRule(t *testing.T) {
	// 1. 读取用户修改的规则文件
	// 路径：neoAgent/rules/fingerprint/nmap-service-probes
	cwd, _ := os.Getwd()
	var rulesPath string
	// 简单适配路径
	if filepath.Base(cwd) == "nmap" {
		rulesPath = "../../../../../rules/fingerprint/nmap-service-probes"
	} else {
		rulesPath = "rules/fingerprint/nmap-service-probes"
	}

	content, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Logf("Skipping user rule test: file not found at %s", rulesPath)
		return
	}

	// 2. 初始化引擎
	engine := NewNmapEngine()
	engine.SetRules(string(content))

	// 触发解析，确保规则格式正确
	if err := engine.ensureInit(); err != nil {
		t.Fatalf("Failed to parse user rules: %v", err)
	}

	// 3. 验证是否包含 Cool-admin 探针
	if _, ok := engine.probeMap["Cool-admin"]; !ok {
		t.Fatal("Custom probe 'Cool-admin' not found in the rules file")
	}
	t.Log("Found custom probe 'Cool-admin'")

	// 4. 模拟 Cool-admin 服务
	// 规则: match http m|^HTTP/1\.1 200 OK.*<title>COOL-AMIND|s p/cool-amind/
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// 读取请求 (Nmap 会发送 "GET / HTTP/1.0\r\n\r\n")
			buf := make([]byte, 1024)
			conn.Read(buf)

			// 发送匹配的响应
			resp := "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n<html><head><title>COOL-AMIND</title></head><body>Login</body></html>"
			conn.Write([]byte(resp))
			conn.Close()
		}
	}()

	// 5. 扫描测试
	ctx := context.Background()
	// 注意：Cool-admin 探针关联了端口 8001。
	// 如果我们的 Mock Server 端口不是 8001，引擎可能不会默认使用 Cool-admin 探针（除非 rarity 足够低）。
	// 检查用户定义的 Cool-admin 没有定义 rarity，默认是多少？
	// 如果没定义 rarity，parser 会默认吗？
	// 让我们看看 parser.go

	// 为了确保 Cool-admin 被使用，我们可以强制 engine.allProbes 包含它，
	// 并且如果它不是通用探针，我们可能需要 mock 端口为 8001。
	// 但我们的 mock server 端口是随机的。

	// 检查规则内容：
	// Probe TCP Cool-admin q|GET / HTTP/1.0\r\n\r\n|
	// ports 8001

	// 所以如果扫描端口不是 8001，这个探针可能不会被触发（除非它被认为是通用的）。
	// 默认 Rarity 是多少？
	// 在 parser.go 中，我们没设置默认值，int 默认为 0。
	// 如果 Rarity 是 0，它就是非常通用的，会被所有端口扫描。
	// 让我们确认一下。

	fp, err := engine.Scan(ctx, "127.0.0.1", port, 2*time.Second)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if fp == nil {
		// 如果没扫出来，可能是因为 Rarity 问题，或者端口不匹配。
		// 尝试手动指定端口为 8001 (虽然实际连接的是 port)
		// 不行，Scan 函数用 port 去连接。

		// 我们可以检查一下 Cool-admin 的 Rarity。
		p := engine.probeMap["Cool-admin"]
		t.Logf("Cool-admin Rarity: %d", p.Rarity)

		t.Fatal("Fingerprint not found")
	}

	t.Logf("Fingerprint: %+v", fp)

	if fp.ProductName != "cool-amind" { // 注意用户规则写的是 cool-amind
		t.Errorf("Expected product cool-amind, got %s", fp.ProductName)
	}
}

// TestRealNmapRules 测试加载真实的 nmap-service-probes 文件
func TestRealNmapRules(t *testing.T) {
	// ... (保持原样)
	cwd, _ := os.Getwd()
	var rulesPath string
	if filepath.Base(cwd) == "nmap" {
		rulesPath = "../../../../../rules/fingerprint/nmap-service-probes"
	} else {
		rulesPath = "rules/fingerprint/nmap-service-probes"
	}

	content, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Logf("Skipping real rules test: rules file not found at %s", rulesPath)
		return
	}

	engine := NewNmapEngine()
	engine.SetRules(string(content))

	err = engine.ensureInit()
	if err != nil {
		t.Fatalf("Failed to parse real rules: %v", err)
	}

	// 验证 SSH 端口 (22) 是否有特定探针
	sshProbes := engine.getProbesForPort(22)
	if len(sshProbes) == 0 {
		t.Error("No probes found for port 22")
	}
}

// TestScanLocalMock 模拟一个简单的服务进行扫描测试
func TestScanLocalMock(t *testing.T) {
	// ... (保持原样)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Write([]byte("SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5\r\n"))
			buf := make([]byte, 1024)
			conn.Read(buf)
			conn.Close()
		}
	}()

	mockRules := `
Probe TCP NULL q||
match ssh m/^SSH-([\d.]+)-OpenSSH_([\w._-]+)[ \r\n]/ p/OpenSSH/ v/$2/ i/protocol $1/
`
	engine := NewNmapEngine()
	engine.SetRules(mockRules)

	ctx := context.Background()
	fp, err := engine.Scan(ctx, "127.0.0.1", port, 2*time.Second)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if fp == nil {
		t.Fatal("Fingerprint not found")
	}

	if fp.Service != "ssh" {
		t.Errorf("Expected service ssh, got %s", fp.Service)
	}
}
