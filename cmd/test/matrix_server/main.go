package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Bastien-Antigravity/safe-socket"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

func main() {
	addr := "127.0.0.1:9999"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	fmt.Printf("Matrix Server: Starting on %s\n", addr)
	
	// Create server with tcp-hello profile
	server, err := safesocket.Create("tcp-hello:matrix-server", addr, "", "server", true)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	fmt.Println("Matrix Server: Listening...")

	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Printf("Matrix Server: Accept error: %v\n", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn interfaces.TransportConnection) {
	defer conn.Close()
	
	// Get Peer Identity
	identity := safesocket.GetIdentity(conn)
	peerName := "Unknown"
	peerHost := "Unknown"
	peerPublicIP := "Unknown"

	if identity != nil {
		if name, err := identity.FromName(); err == nil { peerName = name }
		if host, err := identity.FromHost(); err == nil { peerHost = host }
		if ip, err := identity.FromPublicIP(); err == nil { peerPublicIP = ip }
	}
	fmt.Printf("Matrix Server: Accepted connection from %s (Host: %s, PublicIP: %s)\n", peerName, peerHost, peerPublicIP)

	for {
		// Set a long timeout for tests
		conn.SetIdleTimeout(10 * time.Second)

		msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Matrix Server: Connection closed for %s: %v\n", peerName, err)
			return
		}

		if len(msg) == 0 {
			continue // Heartbeat
		}

		payload := string(msg)
		if len(msg) > 100 {
			payload = fmt.Sprintf("<Large Payload: %d bytes>", len(msg))
		}
		fmt.Printf("Matrix Server: Received from %s: %s\n", peerName, payload)
		
		var reply []byte
		if string(msg) == "meta_request" {
			replyStr := fmt.Sprintf("meta:%s,%s,%s", peerName, peerHost, peerPublicIP)
			reply = []byte(replyStr)
		} else {
			// standard echo
			prefix := []byte("echo:")
			reply = make([]byte, len(prefix)+len(msg))
			copy(reply, prefix)
			copy(reply[len(prefix):], msg)
		}

		_, err = conn.Write(reply)
		if err != nil {
			fmt.Printf("Matrix Server: Write error for %s: %v\n", peerName, err)
			return
		}
	}
}
