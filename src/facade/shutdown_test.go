package facade

import (
	"net"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
)

type mockProfile struct {
	interfaces.SocketProfile
}

func (m *mockProfile) GetTransport() interfaces.TransportType { return interfaces.TransportFramedTCP }
func (m *mockProfile) GetAddress() string                     { return "127.0.0.1:0" }
func (m *mockProfile) GetConnectTimeout() int                 { return 1000 }
func (m *mockProfile) GetProtocol() interfaces.ProtocolType   { return interfaces.ProtocolNone }

func TestSynchronousShutdown(t *testing.T) {
	profile := &mockProfile{}
	config := models.SocketConfig{Deadline: 1 * time.Second}
	server := NewSocketServer(profile, config)

	// 1. Listen
	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	addr, _ := server.GetAddr()

	// 2. Accept in background
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				return
			}
			// Simulate work: don't close immediately
			go func() {
				time.Sleep(500 * time.Millisecond)
				_ = conn.Close()
			}()
		}
	}()

	// 3. Connect 3 clients
	for i := 0; i < 3; i++ {
		_, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 4. Close server
	start := time.Now()
	err := server.Close()
	if err != nil {
		t.Fatal(err)
	}
	duration := time.Since(start)

	// 5. Verify it waited at least 500ms
	if duration < 500*time.Millisecond {
		t.Errorf("expected server.Close() to wait for connections, but it returned in %v", duration)
	} else {
		t.Logf("Server closed gracefully in %v", duration)
	}
}
