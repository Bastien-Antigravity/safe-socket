package transports

import (
	"net"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// -----------------------------------------------------------------------------

// Connect dialer helper for FramedTCPSocket.
func Connect(address string, timeout time.Duration) (interfaces.TransportConnection, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, err
	}
	// Note: We deliberately use the 'timeout' for both connection AND subsequent read/writes
	socket := NewFramedTCPSocket(conn, timeout)

	// Apply TCP Optimizations
	// 1. KeepAlive (detect dead peers)
	_ = socket.SetKeepAlive(30 * time.Second)

	// 2. NoDelay (Disable Nagle's algorithm for lower latency)
	_ = socket.SetNoDelay(true)

	// 3. Buffer Sizes (High throughput support)
	// 4MB buffers matching UDP reliability config
	_ = socket.SetReadBuffer(4 * 1024 * 1024)
	_ = socket.SetWriteBuffer(4 * 1024 * 1024)

	return socket, nil
}
