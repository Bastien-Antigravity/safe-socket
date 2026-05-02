package transports

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestOOMProtection(t *testing.T) {
	// 1. Start a real listener
	addr := "127.0.0.1:0"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	actualAddr := ln.Addr().String()

	// 2. Accept loop in background
	errChan := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		socket := NewFramedTCPSocket(conn, 1*time.Second)
		_, err = socket.ReadMessage()
		errChan <- err
	}()

	// 3. Client: Send an oversized header
	client, err := net.Dial("tcp", actualAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Send 20MB length (Max is 16MB)
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, 20*1024*1024)
	_, _ = client.Write(header)

	// 4. Verify the server dropped the connection
	select {
	case err := <-errChan:
		if err == nil {
			t.Error("expected error for oversized payload, got nil")
		} else {
			t.Logf("Successfully rejected oversized payload: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for server to reject connection")
	}
}
