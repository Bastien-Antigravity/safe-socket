package transports

import (
	"encoding/binary"
	"io"
	"net"
	"time"
)

// FramedTCPSocket implements interfaces.TransportConnection.
// It uses a 4-byte BigEndian length header for every write.
type FramedTCPSocket struct {
	Conn    net.Conn
	Timeout time.Duration
}

// -----------------------------------------------------------------------------

func NewFramedTCPSocket(conn net.Conn, timeout time.Duration) *FramedTCPSocket {
	return &FramedTCPSocket{
		Conn:    conn,
		Timeout: timeout,
	}
}

// -----------------------------------------------------------------------------

// SetKeepAlive enables TCP keepalive with the specified period.
func (s *FramedTCPSocket) SetKeepAlive(period time.Duration) error {
	if tcpConn, ok := s.Conn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			return err
		}
		return tcpConn.SetKeepAlivePeriod(period)
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetNoDelay controls Nagle's algorithm (true = disable Nagle, lower latency).
func (s *FramedTCPSocket) SetNoDelay(enabled bool) error {
	if tcpConn, ok := s.Conn.(*net.TCPConn); ok {
		return tcpConn.SetNoDelay(enabled)
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetReadBuffer sets the size of the operating system's receive buffer.
func (s *FramedTCPSocket) SetReadBuffer(bytes int) error {
	if tcpConn, ok := s.Conn.(*net.TCPConn); ok {
		return tcpConn.SetReadBuffer(bytes)
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetWriteBuffer sets the size of the operating system's transmit buffer.
func (s *FramedTCPSocket) SetWriteBuffer(bytes int) error {
	if tcpConn, ok := s.Conn.(*net.TCPConn); ok {
		return tcpConn.SetWriteBuffer(bytes)
	}
	return nil
}

// -----------------------------------------------------------------------------

// Write prepends length and writes data.
func (s *FramedTCPSocket) Write(p []byte) (n int, err error) {
	if s.Timeout > 0 {
		_ = s.Conn.SetWriteDeadline(time.Now().Add(s.Timeout))
	}

	// 1. Prepare Header (4 bytes length)Endian)
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(p)))

	// 2. Write Header
	_, err = s.Conn.Write(header)
	if err != nil {
		return 0, err
	}

	// 3. Write Data
	return s.Conn.Write(p)
}

// -----------------------------------------------------------------------------

// Read expects a 4-byte BigEndian length header, then reads that many bytes.
func (s *FramedTCPSocket) Read(p []byte) (n int, err error) {
	if s.Timeout > 0 {
		_ = s.Conn.SetReadDeadline(time.Now().Add(s.Timeout))
	}

	// 1. Read Length (4 bytes)
	header := make([]byte, 4)
	if _, err := io.ReadFull(s.Conn, header); err != nil {
		return 0, err
	}

	length := binary.BigEndian.Uint32(header)

	// 2. Check provided buffer size
	if uint32(len(p)) < length {
		return 0, io.ErrShortBuffer
	}

	// 3. Read Body
	return io.ReadFull(s.Conn, p[:length])
}

// -----------------------------------------------------------------------------

// ReadMessage implements the dynamic read.
func (s *FramedTCPSocket) ReadMessage() ([]byte, error) {
	if s.Timeout > 0 {
		_ = s.Conn.SetReadDeadline(time.Now().Add(s.Timeout))
	}

	// 1. Read Length
	header := make([]byte, 4)
	if _, err := io.ReadFull(s.Conn, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)

	// 2. Allocate exact size
	buf := make([]byte, length)

	// 3. Read Body
	if _, err := io.ReadFull(s.Conn, buf); err != nil {
		return nil, err
	}

	return buf, nil
}

// -----------------------------------------------------------------------------

func (s *FramedTCPSocket) Close() error {
	return s.Conn.Close()
}

// -----------------------------------------------------------------------------

// LocalAddr returns the local network address.
func (s *FramedTCPSocket) LocalAddr() net.Addr {
	return s.Conn.LocalAddr()
}

// -----------------------------------------------------------------------------

// RemoteAddr returns the remote network address.
func (s *FramedTCPSocket) RemoteAddr() net.Addr {
	return s.Conn.RemoteAddr()
}
