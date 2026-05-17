package facade

import (
	"net"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/transports"
)

func TestDynamicHeartbeatRestart(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer func() { _ = ln.Close() }()

	clientConn, _ := net.Dial("tcp", ln.Addr().String())
	defer func() { _ = clientConn.Close() }()

	// Start with heartbeats enabled
	sock := transports.NewFramedTCPSocket(clientConn, 1*time.Second)
	h := NewHeartbeatConnection(sock, 100*time.Millisecond)

	// Disable heartbeats via Infinite Wait
	_ = h.SetIdleTimeout(0)
	// Wait to ensure goroutine would have stopped (internal check)
	time.Sleep(200 * time.Millisecond)

	// Restart heartbeats
	_ = h.SetIdleTimeout(500 * time.Millisecond)

	// If we reach here without panic or deadlock, the restart logic is healthy
	_ = h.Close()
}
