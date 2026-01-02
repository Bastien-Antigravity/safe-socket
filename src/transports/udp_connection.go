package transports

import (
	"net"
	"time"
)

// UdpSocket implements interfaces.TransportConnection over UDP.
// Note: UDP is unreliable and unordered.
// UdpSocket implements interfaces.TransportConnection over UDP.
// Note: UDP is unreliable and unordered.
type UdpSocket struct {
	Conn *net.UDPConn
	// Timeout time.Duration // Removed in favor of explicit SetDeadline

	// Server-Side: "Transient" socket fields
	TransientRemoteAddr *net.UDPAddr // If set, Write() uses WriteToUDP
	RecvBuf             []byte       // If set, Read() returns this buffer first (one-shot)
}

// -----------------------------------------------------------------------------

func NewUdpSocket(conn *net.UDPConn, timeout time.Duration) *UdpSocket {
	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}
	return &UdpSocket{
		Conn: conn,
	}
}

// NewTransientUdpSocket creates a socket representing a single packet from a sender.
func NewTransientUdpSocket(conn *net.UDPConn, addr *net.UDPAddr, data []byte, timeout time.Duration) *UdpSocket {
	// Transient socket inherits the underlying conn's state usually, but
	// if we want to enforce the specific timeout on this "session", we might set it.
	// However, sharing the conn means sharing the deadline.
	// For now, we apply it if provided.
	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}

	return &UdpSocket{
		Conn:                conn,
		TransientRemoteAddr: addr,
		RecvBuf:             data,
	}
}

// -----------------------------------------------------------------------------

// Write sends a datagram.
// Warning: Messages larger than MTU (usually 1500 bytes) will be fragmented.
// Messages larger than 64KB will fail.
func (s *UdpSocket) Write(p []byte) (n int, err error) {
	// OPTIMIZATION: Removed SetWriteDeadline logic from hot path.

	// If this is a transient server socket, reply to the specific remote address
	if s.TransientRemoteAddr != nil {
		return s.Conn.WriteToUDP(p, s.TransientRemoteAddr)
	}

	// Otherwise, use standard Write (client-side connected socket)
	return s.Conn.Write(p)
}

// -----------------------------------------------------------------------------

// Read reads a datagram.
// Note: If 'p' is smaller than the incoming datagram, the excess data is discarded by the OS.
func (s *UdpSocket) Read(p []byte) (n int, err error) {
	// If we have a pre-read buffer (Transient Server Socket), return it immediately
	if s.RecvBuf != nil {
		n := copy(p, s.RecvBuf)
		s.RecvBuf = nil // consumed
		return n, nil
	}

	// OPTIMIZATION: Removed SetReadDeadline logic from hot path.
	return s.Conn.Read(p)
}

// -----------------------------------------------------------------------------

// ReadMessage for UDP allocates a buffer large enough for a standard UDP packet (64KB max),
// reads, and returns the sliced data.
func (s *UdpSocket) ReadMessage() ([]byte, error) {
	// If we have a pre-read buffer (Transient Server Socket), return it immediately
	if s.RecvBuf != nil {
		result := make([]byte, len(s.RecvBuf))
		copy(result, s.RecvBuf)
		s.RecvBuf = nil // consumed
		return result, nil
	}

	// OPTIMIZATION: Removed SetReadDeadline logic from hot path.

	// Max UDP packet size is technically ~65535.
	// We allocate a temp buffer.
	tmp := make([]byte, 65535)

	n, remoteAddr, err := s.Conn.ReadFromUDP(tmp)
	if err != nil {
		return nil, err
	}

	// For Transient logic (server side reply), we store the remote addr
	// Note: Thread safety issue here if sharing socket, but UDP socket per-packet model usually implies single thread or copy.
	// But actually, UdpSocket struct is updated with TransientRemoteAddr.
	s.TransientRemoteAddr = remoteAddr

	// Return a copy of exactly n bytes
	// Or return slice? Slice keeps underlying array alive. Copy is safer for memory if array is huge.
	// Given 64KB is small, we can copy to fit.
	result := make([]byte, n)
	copy(result, tmp[:n])
	return result, nil
}

// -----------------------------------------------------------------------------

func (s *UdpSocket) Close() error {
	return s.Conn.Close()
}

// -----------------------------------------------------------------------------

// SetDeadline sets the read and write deadlines.
func (s *UdpSocket) SetDeadline(t time.Time) error {
	return s.Conn.SetDeadline(t)
}

// -----------------------------------------------------------------------------

// SetReadDeadline sets the read deadline.
func (s *UdpSocket) SetReadDeadline(t time.Time) error {
	return s.Conn.SetReadDeadline(t)
}

// -----------------------------------------------------------------------------

// SetWriteDeadline sets the write deadline.
func (s *UdpSocket) SetWriteDeadline(t time.Time) error {
	return s.Conn.SetWriteDeadline(t)
}

// -----------------------------------------------------------------------------

// LocalAddr returns the local network address.
func (s *UdpSocket) LocalAddr() net.Addr {
	return s.Conn.LocalAddr()
}

// -----------------------------------------------------------------------------

// RemoteAddr returns the remote network address.
func (s *UdpSocket) RemoteAddr() net.Addr {
	if s.TransientRemoteAddr != nil {
		return s.TransientRemoteAddr
	}
	return s.Conn.RemoteAddr()
}
