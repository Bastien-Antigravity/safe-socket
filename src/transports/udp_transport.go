package transports

import (
	"net"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// UdpSocket implements interfaces.TransportConnection over UDP.
// Note: UDP is unreliable and unordered.
// -----------------------------------------------------------------------------
type UdpSocket struct {
	Conn    *net.UDPConn
	Timeout time.Duration
}

// -----------------------------------------------------------------------------

func NewUdpSocket(conn *net.UDPConn, timeout time.Duration) *UdpSocket {
	return &UdpSocket{
		Conn:    conn,
		Timeout: timeout,
	}
}

// -----------------------------------------------------------------------------

// ConnectUDP creates a UDP connection.
// Note: UDP is connectionless. "Dial" just sets the default destination address.
func ConnectUDP(address string, timeout time.Duration) (interfaces.TransportConnection, error) {
	// Resolve address
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	// Dial (connect) to setting the default remote address
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	// Reliability: Increase OS buffers to reduce packet drops
	// 4MB buffers (adjust based on needs/OS limits)
	_ = conn.SetReadBuffer(4 * 1024 * 1024)
	_ = conn.SetWriteBuffer(4 * 1024 * 1024)

	return NewUdpSocket(conn, timeout), nil
}

// -----------------------------------------------------------------------------

// Write sends a datagram.
// Warning: Messages larger than MTU (usually 1500 bytes) will be fragmented.
// Messages larger than 64KB will fail.
// Write sends a datagram.
// Warning: Messages larger than MTU (usually 1500 bytes) will be fragmented.
// Messages larger than 64KB will fail.
func (s *UdpSocket) Write(p []byte) (n int, err error) {
	if s.Timeout > 0 {
		_ = s.Conn.SetWriteDeadline(time.Now().Add(s.Timeout))
	}
	return s.Conn.Write(p)
}

// -----------------------------------------------------------------------------

// Read reads a datagram.
// Note: If 'p' is smaller than the incoming datagram, the excess data is discarded by the OS.
func (s *UdpSocket) Read(p []byte) (n int, err error) {
	if s.Timeout > 0 {
		_ = s.Conn.SetReadDeadline(time.Now().Add(s.Timeout))
	}
	return s.Conn.Read(p)
}

// -----------------------------------------------------------------------------

func (s *UdpSocket) Close() error {
	return s.Conn.Close()
}
