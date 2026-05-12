package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/facade"
	"github.com/Bastien-Antigravity/safe-socket/src/factory"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
)

// -----------------------------------------------------------------------------
// TCP Tests
// -----------------------------------------------------------------------------

// TestTCP_Raw Verifies basic TCP send/receive using the Factory
func TestTCP_Raw(t *testing.T) {
	addr := "127.0.0.1:9003" // Port 9003

	// 1. Start Server
	server, err := factory.Create("tcp", addr, "", "server", true)
	if err != nil {
		t.Fatalf("Failed to create TCP server: %v", err)
	}
	defer func() { _ = server.Close() }()

	errChan := make(chan error, 1)
	go func() {
		// Accept block
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

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
	client, err := factory.Create("tcp", addr, "", "client", true)
	if err != nil {
		t.Fatalf("Failed to create TCP client: %v", err)
	}
	defer func() { _ = client.Close() }()

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
	buf, err := client.Receive()
	if err != nil {
		t.Fatalf("Client receive failed: %v", err)
	}
	if string(buf) != "TCP_PONG" {
		t.Fatalf("Client received unexpected: %s", string(buf))
	}
}

// TestTCP_Raw_Write_Method Verifies the new Write() method (Send alias)
func TestTCP_Raw_Write_Method(t *testing.T) {
	addr := "127.0.0.1:9010" // Distinct port

	// 1. Start Server
	server, err := factory.Create("tcp", addr, "", "server", true)
	if err != nil {
		t.Fatalf("Failed to create TCP server: %v", err)
	}
	defer func() { _ = server.Close() }()

	errChan := make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			errChan <- fmt.Errorf("Read failed: %v", err)
			return
		}

		if string(buf[:n]) != "WRITE_TEST" {
			errChan <- fmt.Errorf("Unexpected message: %s", string(buf[:n]))
			return
		}

		_, _ = conn.Write([]byte("ACK"))
		errChan <- nil
	}()

	// 2. Start Client
	client, err := factory.Create("tcp", addr, "", "client", true)
	if err != nil {
		t.Fatalf("Failed to create TCP client: %v", err)
	}
	defer func() { _ = client.Close() }()

	// 3. Use the new Write method (which returns count and error)
	if _, err := client.Write([]byte("WRITE_TEST")); err != nil {
		t.Fatalf("Client Write failed: %v", err)
	}

	// 4. Verify
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for write exchange")
	}
}

// TestTCP_Hello Verifies TCP with Hello Protocol Handshake
func TestTCP_Hello(t *testing.T) {
	addr := "127.0.0.1:9004" // Port 9004

	// 1. Start Server
	server, err := factory.Create("tcp-hello:test-server", addr, "127.0.0.1", "server", true)
	if err != nil {
		t.Fatalf("Failed to create TCP Hello server: %v", err)
	}
	defer func() { _ = server.Close() }()

	errChan := make(chan error, 1)
	go func() {
		// Accept - Should block until Handshake is complete
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed (Handshake error?): %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

		// Verify Identity Access (Unwrap Heartbeat if present)
		inner := conn
		if hb, ok := conn.(*facade.HeartbeatConnection); ok {
			inner = hb.TransportConnection
		}

		if hc, ok := inner.(*facade.HandshakeConnection); ok {
			name, _ := hc.Identity.FromName()
			fmt.Printf("Handshake Identity Name: %s\n", name)
		} else {
			errChan <- fmt.Errorf("Connection is not a HandshakeConnection")
			return
		}

		// If accepted, handshake succeeded. Just echo.
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		_, _ = conn.Write(buf[:n])

		errChan <- nil
	}()

	// 2. Start Client
	client, err := factory.Create("tcp-hello:test-client", addr, "127.0.0.1", "client", false)
	if err != nil {
		t.Fatalf("Failed to create TCP Hello client: %v", err)
	}
	// Open() performs the handshake
	if err := client.Open(); err != nil {
		t.Fatalf("Client Open (Handshake) failed: %v", err)
	}
	defer func() { _ = client.Close() }()

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
	server, err := factory.Create("udp", addr, "", "server", true)
	if err != nil {
		t.Fatalf("Failed to create UDP server: %v", err)
	}
	defer func() { _ = server.Close() }()

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
	client, err := factory.Create("udp", addr, "", "client", true)
	if err != nil {
		t.Fatalf("Failed to create UDP client: %v", err)
	}
	defer func() { _ = client.Close() }()

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
	server, err := factory.Create("udp-hello:test-udp-server", addr, "", "server", true)
	if err != nil {
		t.Fatalf("Failed to create UDP Hello server: %v", err)
	}
	defer func() { _ = server.Close() }()

	errChan := make(chan error, 1)
	go func() {
		// Accept will block until the first packet arrives.
		// That packet will be Decapsulated by the server facade.
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

		// Allow time for testing (Unwrap Heartbeat if present)
		var env *facade.EnvelopedConnection
		if hb, ok := conn.(*facade.HeartbeatConnection); ok {
			env = hb.TransportConnection.(*facade.EnvelopedConnection)
		} else {
			env = conn.(*facade.EnvelopedConnection)
		}
		env.Proto = protocols.NewHelloProtocol().(*protocols.HelloProtocol)
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
	client, err := factory.Create("udp-hello:test-udp-client", addr, "127.0.0.1", "client", false)
	if err != nil {
		t.Fatalf("Failed to create UDP Hello client: %v", err)
	}

	if err := client.Open(); err != nil {
		t.Fatalf("Client Open failed: %v", err)
	}
	defer func() { _ = client.Close() }()

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

// TestUDP_Reliable Verifies UDP with Reliability Layer (ACKs/Retries)
func TestUDP_Reliable(t *testing.T) {
	addr := "127.0.0.1:9003"

	// 1. Start Server with Reliable: true
	config := models.SocketConfig{
		Reliable: true,
	}
	server, err := factory.CreateWithConfig("udp", addr, config, "server", true)
	if err != nil {
		t.Fatalf("Failed to create Reliable UDP server: %v", err)
	}
	defer func() { _ = server.Close() }()

	errChan := make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			errChan <- fmt.Errorf("Read failed: %v", err)
			return
		}

		msg := string(buf[:n])
		if msg != "RELIABLE_PING" {
			errChan <- fmt.Errorf("Unexpected message: %s", msg)
			return
		}

		_, err = conn.Write([]byte("RELIABLE_PONG"))
		errChan <- err
	}()

	// 2. Start Client with Reliable: true
	client, err := factory.CreateWithConfig("udp", addr, config, "client", true)
	if err != nil {
		t.Fatalf("Failed to create Reliable UDP client: %v", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Send([]byte("RELIABLE_PING")); err != nil {
		t.Fatalf("Client send failed: %v", err)
	}

	resp, err := client.Receive()
	if err != nil {
		t.Fatalf("Client receive failed: %v", err)
	}

	if string(resp) != "RELIABLE_PONG" {
		t.Errorf("Unexpected response: %s", string(resp))
	}

	// 3. Wait for server result
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for Reliable UDP exchange")
	}
}


// TestSHM_FullExchange Verifies server/client exchange via SHM
func TestSHM_FullExchange(t *testing.T) {
	path := "test_shm_exchange"
	defer func() { _ = os.Remove(path) }()

	// 1. Create Server
	server, err := factory.Create("shm", path, "", "server", true)
	if err != nil {
		t.Fatalf("Failed to create SHM server: %v", err)
	}
	defer func() { _ = server.Close() }()

	// 2. Create Client
	client, err := factory.Create("shm", path, "", "client", false)
	if err != nil {
		t.Fatalf("Failed to create SHM client: %v", err)
	}

	// 3. Accept and Open
	errChan := make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			errChan <- err
			return
		}
		defer func() { _ = conn.Close() }()

		msg, err := conn.ReadMessage()
		if err != nil {
			errChan <- err
			return
		}
		if string(msg) != "Hello SHM" {
			errChan <- fmt.Errorf("unexpected message: %s", string(msg))
			return
		}

		_, err = conn.Write([]byte("Hello Back"))
		errChan <- err
	}()

	if err := client.Open(); err != nil {
		t.Fatalf("Failed to open SHM client: %v", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Send([]byte("Hello SHM")); err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	resp, err := client.Receive()
	if err != nil {
		t.Fatalf("Failed to receive: %v", err)
	}

	if string(resp) != "Hello Back" {
		t.Errorf("Unexpected response: %s", string(resp))
	}

	if err := <-errChan; err != nil {
		t.Errorf("Server error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// TLS Tests
// -----------------------------------------------------------------------------

// TestTLS_Hello Verifies TCP with TLS and Hello Protocol
func TestTLS_Hello(t *testing.T) {
	addr := "127.0.0.1:9004"

	// 1. Generate Certificates
	certFile := "test_server.crt"
	keyFile := "test_server.key"
	err := generateSelfSignedCert(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to generate certs: %v", err)
	}
	defer func() {
		_ = os.Remove(certFile)
		_ = os.Remove(keyFile)
	}()

	// 2. Start Server with TLS config
	config := models.SocketConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		Deadline: 1 * time.Second,
	}
	server, err := factory.CreateWithConfig("tls-hello:test-tls-server", addr, config, "server", true)
	if err != nil {
		t.Fatalf("Failed to create TLS server: %v", err)
	}
	defer func() { _ = server.Close() }()

	errChan := make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			errChan <- fmt.Errorf("Accept failed: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			errChan <- fmt.Errorf("Read failed: %v", err)
			return
		}

		msg := string(buf[:n])
		if msg != "TLS_HELLO" {
			errChan <- fmt.Errorf("Unexpected message: %s", msg)
			return
		}

		_, err = conn.Write([]byte("TLS_BACK"))
		errChan <- err
	}()

	// 3. Start Client with TLS config
	clientConfig := models.SocketConfig{
		InsecureSkipVerify: true, // Self-signed
		Deadline:           1 * time.Second,
	}
	client, err := factory.CreateWithConfig("tls-hello:test-tls-client", addr, clientConfig, "client", true)
	if err != nil {
		t.Fatalf("Failed to create TLS client: %v", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Send([]byte("TLS_HELLO")); err != nil {
		t.Fatalf("Client send failed: %v", err)
	}

	resp, err := client.Receive()
	if err != nil {
		t.Fatalf("Client receive failed: %v", err)
	}

	if string(resp) != "TLS_BACK" {
		t.Errorf("Unexpected response: %s", string(resp))
	}

	if err := <-errChan; err != nil {
		t.Errorf("Server error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func generateSelfSignedCert(certPath, keyPath string) error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"SafeSocket Test"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return err
	}

	return nil
}


