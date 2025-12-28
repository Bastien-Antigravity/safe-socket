package test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/facade"
	"github.com/Bastien-Antigravity/safe-socket/src/factory"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
)

// -----------------------------------------------------------------------------
// TCP Tests
// -----------------------------------------------------------------------------

// TestTCP_Raw Verifies basic TCP send/receive using the Factory
func TestTCP_Raw(t *testing.T) {
	addr := "127.0.0.1:9003" // Port 9003

	// 1. Start Server
	server, err := factory.Create("tcp", addr, "", interfaces.SocketTypeServer, true)
	if err != nil {
		t.Fatalf("Failed to create TCP server: %v", err)
	}
	defer server.Close()

	errChan := make(chan error, 1)
	go func() {
		// Accept block
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed: %v", err)
			return
		}
		defer conn.Close()

		// Read
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			errChan <- fmt.Errorf("Read failed: %v", err)
			return
		}

		msg := string(buf[:n])
		if msg != "TCP_PING" {
			errChan <- fmt.Errorf("Unexpected message: %s", msg)
			return
		}

		// Reply
		if _, err := conn.Write([]byte("TCP_PONG")); err != nil {
			errChan <- fmt.Errorf("Write failed: %v", err)
			return
		}
		errChan <- nil
	}()

	// 2. Start Client
	// Give server a moment to bind? Usually Listen() is synchronous so fine.
	client, err := factory.Create("tcp", addr, "", interfaces.SocketTypeClient, true)
	if err != nil {
		t.Fatalf("Failed to create TCP client: %v", err)
	}
	defer client.Close()

	// 3. Exchange
	if err := client.Send([]byte("TCP_PING")); err != nil {
		t.Fatalf("Client send failed: %v", err)
	}

	// 4. Wait for server done
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for TCP Raw exchange")
	}

	// 5. Client Read verification
	buf := make([]byte, 1024)
	n, err := client.Receive(buf)
	if err != nil {
		t.Fatalf("Client receive failed: %v", err)
	}
	if string(buf[:n]) != "TCP_PONG" {
		t.Fatalf("Client received unexpected: %s", string(buf[:n]))
	}
}

// TestTCP_Hello Verifies TCP with Hello Protocol Handshake
func TestTCP_Hello(t *testing.T) {
	addr := "127.0.0.1:9004" // Port 9004

	// 1. Start Server
	server, err := factory.Create("tcp-hello", addr, "127.0.0.1", interfaces.SocketTypeServer, true)
	if err != nil {
		t.Fatalf("Failed to create TCP Hello server: %v", err)
	}
	defer server.Close()

	errChan := make(chan error, 1)
	go func() {
		// Accept - Should block until Handshake is complete
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed (Handshake error?): %v", err)
			return
		}
		defer conn.Close()

		// If accepted, handshake succeeded. Just echo.
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		conn.Write(buf[:n])

		errChan <- nil
	}()

	// 2. Start Client
	client, err := factory.Create("tcp-hello", addr, "127.0.0.1", interfaces.SocketTypeClient, false)
	if err != nil {
		t.Fatalf("Failed to create TCP Hello client: %v", err)
	}
	// Open() performs the handshake
	if err := client.Open(); err != nil {
		t.Fatalf("Client Open (Handshake) failed: %v", err)
	}
	defer client.Close()

	// 3. Send Data
	if err := client.Send([]byte("HELLO_TCP")); err != nil {
		t.Fatalf("Client Send failed: %v", err)
	}

	// 4. Wait
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for TCP Hello exchange")
	}
}

// -----------------------------------------------------------------------------
// UDP Tests
// -----------------------------------------------------------------------------

// TestUDP_Raw Verifies basic UDP send/receive using the Factory
func TestUDP_Raw(t *testing.T) {
	addr := "127.0.0.1:9001"

	// 1. Start Server
	server, err := factory.Create("udp", addr, "", interfaces.SocketTypeServer, true)
	if err != nil {
		t.Fatalf("Failed to create UDP server: %v", err)
	}
	defer server.Close()

	errChan := make(chan error, 1)
	go func() {
		// Accept on UDP returns a transient socket with the packet
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed: %v", err)
			return
		}

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			errChan <- fmt.Errorf("Read failed: %v", err)
			return
		}

		msg := string(buf[:n])
		if msg != "UDP_PING" {
			errChan <- fmt.Errorf("Unexpected message: %s", msg)
			return
		}

		// Reply
		if _, err := conn.Write([]byte("UDP_PONG")); err != nil {
			errChan <- fmt.Errorf("Write failed: %v", err)
			return
		}
		errChan <- nil
	}()

	// 2. Start Client
	client, err := factory.Create("udp", addr, "", interfaces.SocketTypeClient, true)
	if err != nil {
		t.Fatalf("Failed to create UDP client: %v", err)
	}
	defer client.Close()

	if err := client.Send([]byte("UDP_PING")); err != nil {
		t.Fatalf("Client send failed: %v", err)
	}

	// 3. Wait for result
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for UDP exchange")
	}
}

// TestUDP_Hello Verifies UDP with Hello Protocol + Stateless Envelope
func TestUDP_Hello(t *testing.T) {
	addr := "127.0.0.1:9002"

	// 1. Start Server
	server, err := factory.Create("udp-hello", addr, "", interfaces.SocketTypeServer, true)
	if err != nil {
		t.Fatalf("Failed to create UDP Hello server: %v", err)
	}
	defer server.Close()

	errChan := make(chan error, 1)
	go func() {
		// Accept will block until the first packet arrives.
		// That packet will be Decapsulated by the server facade.
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed: %v", err)
			return
		}
		defer conn.Close()

		// Allow time for testing
		conn.(*facade.EnvelopedConnection).Proto = protocols.NewHelloProtocol().(*protocols.HelloProtocol)
		// Note: The facade sets this up, but we need to ensure type assertions work if checking internals.
		// Actually, we just read.

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			errChan <- fmt.Errorf("Read failed: %v", err)
			return
		}

		msg := string(buf[:n])
		if msg != "HELLO_DATA" {
			errChan <- fmt.Errorf("Unexpected payload: %s", msg)
			return
		}
		errChan <- nil
	}()

	// 2. Start Client
	client, err := factory.Create("udp-hello", addr, "127.0.0.1", interfaces.SocketTypeClient, false)
	if err != nil {
		t.Fatalf("Failed to create UDP Hello client: %v", err)
	}

	if err := client.Open(); err != nil {
		t.Fatalf("Client Open failed: %v", err)
	}
	defer client.Close()

	// 3. Send Data (triggers the Envelope + Send)
	// Open() alone sends nothing for UDP-Hello now!
	if err := client.Send([]byte("HELLO_DATA")); err != nil {
		t.Fatalf("Client Send failed: %v", err)
	}

	// 4. Wait
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for UDP Hello exchange")
	}
}

// -----------------------------------------------------------------------------
// SHM Tests
// -----------------------------------------------------------------------------

// TestSHM_Creation Verifies we can create the SHM objects
func TestSHM_Creation(t *testing.T) {
	path := "test_shm_file"
	defer os.Remove(path)

	// Only testing Client creation as Server side isn't implemented effectively for SHM yet
	client, err := factory.Create("shm", path, "", interfaces.SocketTypeClient, false)
	if err != nil {
		t.Fatalf("Failed to create SHM client: %v", err)
	}

	if err := client.Open(); err != nil {
		t.Fatalf("Failed to open SHM: %v", err)
	}
	client.Close()
}
