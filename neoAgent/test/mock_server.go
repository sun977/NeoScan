package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	// 1. Ubuntu Apache (Port 18080)
	go startServer(18080, "HTTP/1.1 200 OK\r\nServer: Ubuntu/Apache\r\nContent-Length: 0\r\n\r\n")

	// 2. Debian OpenSSH (Port 18022)
	go startServer(18022, "SSH-2.0-OpenSSH_7.4p1 Debian-10+deb9u7\r\n")

	// 3. Windows FTP (Port 18021)
	go startServer(18021, "220 Microsoft FTP Service\r\n")

	// Keep alive
	fmt.Println("Mock Server running on 18080, 18022, 18021...")
	select {}
}

func startServer(port int, banner string) {
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Printf("Failed to listen on %d: %v\n", port, err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, banner)
	}
}

func handleConnection(conn net.Conn, banner string) {
	defer conn.Close()
	// Write banner
	conn.Write([]byte(banner))
	// Wait a bit
	time.Sleep(1 * time.Second)
}
