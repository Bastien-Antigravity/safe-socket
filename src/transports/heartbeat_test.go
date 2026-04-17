package transports

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestFramedTCPHeartbeatRead(t *testing.T) {
	// 1. Start a local TCP listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	// 2. Server-side goroutine to send heartbeats
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Send three heartbeats (0-length frames)
		// Frame: [4-byte BigEndian length]
		heartbeat := make([]byte, 4)
		binary.BigEndian.PutUint32(heartbeat, 0)

		for i := 0; i < 3; i++ {
			time.Sleep(100 * time.Millisecond)
			_, _ = conn.Write(heartbeat)
		}

		// Send one actual payload
		payload := []byte("hello")
		header := make([]byte, 4)
		binary.BigEndian.PutUint32(header, uint32(len(payload)))
		_, _ = conn.Write(header)
		_, _ = conn.Write(payload)
	}()

	// 3. Client-side: Connect and wrap in FramedTCPSocket
	rawConn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer rawConn.Close()

	sock := NewFramedTCPSocket(rawConn, 1*time.Second)

	// 4. Verify heartbeats return n=0
	for i := 0; i < 3; i++ {
		buf := make([]byte, 1024)
		n, err := sock.Read(buf)
		if err != nil {
			t.Errorf("Iteration %d: Expected no error on heartbeat, got %v", i, err)
		}
		if n != 0 {
			t.Errorf("Iteration %d: Expected n=0 for heartbeat, got %d", i, n)
		}
	}

	// 5. Verify subsequent payload is read correctly
	buf := make([]byte, 1024)
	n, err := sock.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read payload: %v", err)
	}
	if string(buf[:n]) != "hello" {
		t.Errorf("Expected 'hello', got %s", string(buf[:n]))
	}
}

func TestFramedTCPHeartbeatReadMessage(t *testing.T) {
	// Similar test for ReadMessage
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	addr := ln.Addr().String()

	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()

		hb := make([]byte, 4)
		binary.BigEndian.PutUint32(hb, 0)
		_, _ = conn.Write(hb)

		payload := []byte("world")
		header := make([]byte, 4)
		binary.BigEndian.PutUint32(header, uint32(len(payload)))
		_, _ = conn.Write(header)
		_, _ = conn.Write(payload)
	}()

	rawConn, _ := net.Dial("tcp", addr)
	defer rawConn.Close()
	sock := NewFramedTCPSocket(rawConn, 1*time.Second)

	// Heartbeat
	msg, err := sock.ReadMessage()
	if err != nil {
		t.Errorf("Expected no error on heartbeat, got %v", err)
	}
	if len(msg) != 0 {
		t.Errorf("Expected empty buffer for heartbeat, got %d bytes", len(msg))
	}

	// Payload
	msg, err = sock.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}
	if string(msg) != "world" {
		t.Errorf("Expected 'world', got %s", string(msg))
	}
}
