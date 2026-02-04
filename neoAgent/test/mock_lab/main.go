package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Mock Server 用于在无 Docker 环境下测试 Agent 的基础连接和认证逻辑
// 注意：这不能替代真实环境测试，只能验证 Agent 的网络层和基础协议解析逻辑

func main() {
	var wg sync.WaitGroup

	// 1. Mock SSH (Port 2222)
	wg.Add(1)
	go func() {
		defer wg.Done()
		startTCPServer("SSH", 2222, handleSSH)
	}()

	// 2. Mock Redis (Port 63790)
	wg.Add(1)
	go func() {
		defer wg.Done()
		startTCPServer("Redis", 63790, handleRedis)
	}()

	// 3. Mock HTTP/Elasticsearch (Port 9200)
	wg.Add(1)
	go func() {
		defer wg.Done()
		startHTTPServer("HTTP", 9200)
	}()

	// 4. Mock Generic TCP Sink (Port 33061 - MySQL)
	// 仅接受连接，测试 Agent 的连接超时或握手超时
	wg.Add(1)
	go func() {
		defer wg.Done()
		startTCPServer("MySQL-Sink", 33061, handleSink)
	}()

	// 5. Mock RDP (Port 33890)
	// 发送垃圾数据，测试 Agent 是否会 Panic
	wg.Add(1)
	go func() {
		defer wg.Done()
		startTCPServer("RDP", 33890, handleRDP)
	}()

	fmt.Println("Mock Lab started. Press Ctrl+C to exit.")
	wg.Wait()
}

func handleRDP(conn net.Conn) {
	defer conn.Close()
	// 发送 5 字节数据，测试 tpkt.go 是否会因为长度检查不足而 panic (slice bounds out of range)
	conn.Write([]byte("Hello"))
}

func startTCPServer(name string, port int, handler func(net.Conn)) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Printf("[%s] Failed to start: %v\n", name, err)
		return
	}
	fmt.Printf("[%s] Listening on :%d\n", name, port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handler(conn)
	}
}

func handleSink(conn net.Conn) {
	defer conn.Close()
	// 什么都不做，或者发送一些垃圾数据
	// 让 Agent 认为端口是通的，但是协议握手会失败
	time.Sleep(100 * time.Millisecond)
	conn.Write([]byte("Not MySQL Protocol\n"))
}

func handleSSH(conn net.Conn) {
	defer conn.Close()
	// 1. 发送 Banner
	conn.Write([]byte("SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5\r\n"))

	// 2. 读取客户端 Banner
	buf := make([]byte, 1024)
	conn.Read(buf)

	// 3. 模拟后续交互（极其简化，仅为了让 Agent 不立即报错）
	// 真实的 SSH 握手非常复杂，这里只需要让 Agent 的 SSH Client 认为连上了即可
	// 但实际上 Agent 的 SSH Client 会进行密钥交换，这里 Mock Server 无法完成
	// 所以 Agent 最终会报 Handshake Failed，但这足以证明 TCP 连通性和 Banner 识别
}

func handleRedis(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		fmt.Printf("[Redis] Received: %s\n", line)

		if strings.HasPrefix(line, "*") {
			// Handle RESP Array
			var count int
			fmt.Sscanf(line, "*%d", &count)
			var args []string
			for i := 0; i < count; i++ {
				// Read length line ($N)
				_, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				// Read value line
				val, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				args = append(args, strings.TrimSpace(val))
			}

			if len(args) == 0 {
				continue
			}

			cmd := strings.ToUpper(args[0])
			if cmd == "HELLO" {
				// HELLO 3 AUTH default password123
				if len(args) >= 5 && strings.ToUpper(args[2]) == "AUTH" && args[4] == "password123" {
					// Return Map: {server: redis}
					conn.Write([]byte("%1\r\n$6\r\nserver\r\n$5\r\nredis\r\n"))
				} else {
					conn.Write([]byte("-ERR invalid auth\r\n"))
				}
			} else if cmd == "AUTH" {
				pass := ""
				if len(args) == 2 {
					pass = args[1]
				} else if len(args) == 3 {
					pass = args[2] // user pass
				}
				if pass == "password123" {
					conn.Write([]byte("+OK\r\n"))
				} else {
					conn.Write([]byte("-ERR invalid password\r\n"))
				}
			} else if cmd == "PING" {
				conn.Write([]byte("+PONG\r\n"))
			} else {
				conn.Write([]byte("-ERR unknown command\r\n"))
			}
			continue
		}

		// 简单模拟 RESP 协议 (Inline Commands)
		if strings.HasPrefix(strings.ToUpper(line), "PING") {
			conn.Write([]byte("+PONG\r\n"))
		} else if strings.HasPrefix(strings.ToUpper(line), "AUTH") {
			parts := strings.Fields(line)
			if len(parts) > 1 && parts[1] == "password123" {
				conn.Write([]byte("+OK\r\n"))
			} else {
				conn.Write([]byte("-ERR invalid password\r\n"))
			}
		} else if strings.HasPrefix(strings.ToUpper(line), "QUIT") {
			conn.Write([]byte("+OK\r\n"))
			return
		} else {
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}
}

func startHTTPServer(name string, port int) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "elastic" || pass != "password123" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "green"}`))
	})

	fmt.Printf("[%s] Listening on :%d\n", name, port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
