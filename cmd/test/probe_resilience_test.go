package test

// =============================================================================
// ESSENTIAL PROCESS:
// Tests server socket resilience against raw TCP health checks and immediate
// disconnects before protocol initiation.
//
// DATA FLOW:
// 1. Input: TCP listener bound with tcp-hello profile.
// 2. Logic: Raw TCP clients connect and close immediately (0-byte probe);
//    subsequently, a valid tcp-hello client connects.
// 3. Output: Server Accept ignores raw probes and successfully receives the valid client.
//
// KEY PARAMETERS:
// - t: Testing handle.
// =============================================================================

import (
	"net"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
)

// -----------------------------------------------------------------------------

func TestTCPProbeResilience(t *testing.T) {
	// 1. Find a free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind free port: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	// 2. Create server with tcp-hello profile
	config := models.SocketConfig{
		Deadline: 1 * time.Second,
	}
	server, err := safesocket.CreateWithConfig("tcp-hello:probe-server", addr, config, "server", true)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() { _ = server.Close() }()

	serverAccepted := make(chan string, 1)
	serverErr := make(chan error, 1)

	// Run server accept loop
	go func() {
		conn, err := server.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer func() { _ = conn.Close() }()

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			serverErr <- err
			return
		}
		serverAccepted <- string(buf[:n])
	}()

	// Wait briefly for server to enter accept loop
	time.Sleep(50 * time.Millisecond)

	// 3. Simulate raw TCP probes (e.g. watchdog-agent IsPortListening or nc -z)
	for i := 0; i < 3; i++ {
		rawConn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err != nil {
			t.Fatalf("Raw probe %d failed to connect: %v", i+1, err)
		}
		// Immediately close without sending any handshake data
		_ = rawConn.Close()
		time.Sleep(20 * time.Millisecond)
	}

	// 4. Now connect with a legitimate tcp-hello client
	client, err := safesocket.CreateWithConfig("tcp-hello:legit-client", addr, config, "client", true)
	if err != nil {
		t.Fatalf("Failed to create legitimate client: %v", err)
	}
	defer func() { _ = client.Close() }()

	payload := "hello from legit client"
	_, err = client.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Client failed to write payload: %v", err)
	}

	// 5. Verify server received payload from the legitimate client
	select {
	case received := <-serverAccepted:
		if received != payload {
			t.Fatalf("Expected '%s', got '%s'", payload, received)
		}
	case err := <-serverErr:
		t.Fatalf("Server Accept returned error unexpectedly: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for server to accept legitimate client after probes")
	}
}

// -----------------------------------------------------------------------------

func TestTCPInvalidProtocolFailsStrictly(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind free port: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	config := models.SocketConfig{
		Deadline: 1 * time.Second,
	}
	server, err := safesocket.CreateWithConfig("tcp-hello:strict-server", addr, config, "server", true)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() { _ = server.Close() }()

	serverErr := make(chan error, 1)
	go func() {
		_, err := server.Accept()
		serverErr <- err
	}()

	time.Sleep(50 * time.Millisecond)

	// Send invalid protocol bytes (framed payload that is not a valid Cap'n Proto HelloMsg)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer func() { _ = conn.Close() }()

	_, _ = conn.Write([]byte{0x00, 0x00, 0x00, 0x04, 'B', 'A', 'D', '!'})

	select {
	case err := <-serverErr:
		if err == nil {
			t.Fatal("Expected server Accept to fail strictly on invalid protocol handshake, but got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for server to reject invalid protocol handshake")
	}
}

