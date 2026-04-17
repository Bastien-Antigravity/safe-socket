package test

import (
	"net"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket"
	"github.com/Bastien-Antigravity/safe-socket/src/facade"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/schemas"
)

// Minimal mock for testing wrappers
type MockTransport struct {
	interfaces.TransportConnection
}

func (m *MockTransport) Write(p []byte) (n int, err error) { return len(p), nil }
func (m *MockTransport) Close() error                     { return nil }
func (m *MockTransport) LocalAddr() net.Addr            { return nil }
func (m *MockTransport) RemoteAddr() net.Addr           { return nil }
func (m *MockTransport) ReadMessage() ([]byte, error)   { return nil, nil }
func (m *MockTransport) SetDeadline(t time.Time) error  { return nil }
func (m *MockTransport) SetReadDeadline(t time.Time) error { return nil }
func (m *MockTransport) SetWriteDeadline(t time.Time) error { return nil }
func (m *MockTransport) SetIdleTimeout(d time.Duration) error { return nil }

func TestGetIdentity(t *testing.T) {
	// Mock transport
	mock := &MockTransport{}

	identity := &schemas.HelloMsg{}

	// 1. Handshake Wrapper
	hc := facade.NewHandshakeConnection(mock, identity)
	
	result := safesocket.GetIdentity(hc)
	if result != identity {
		t.Errorf("GetIdentity failed for HandshakeConnection: expected %p, got %p", identity, result)
	}

	// 2. Heartbeat -> Handshake Wrapper
	hb := facade.NewHeartbeatConnection(hc, 0)
	defer hb.Close()

	result = safesocket.GetIdentity(hb)
	if result != identity {
		t.Errorf("GetIdentity failed for Heartbeat wrapper: expected %p, got %p", identity, result)
	}

	// 3. Envelope Wrapper
	ec := &facade.EnvelopedConnection{
		Conn:         mock,
		LastIdentity: identity,
	}

	result = safesocket.GetIdentity(ec)
	if result != identity {
		t.Errorf("GetIdentity failed for EnvelopedConnection: expected %p, got %p", identity, result)
	}

	// 4. Raw connection (nil result)
	result = safesocket.GetIdentity(mock)
	if result != nil {
		t.Errorf("GetIdentity should return nil for raw connection, got %p", result)
	}
}
