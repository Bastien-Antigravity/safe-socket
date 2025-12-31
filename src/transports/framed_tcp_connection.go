package transports

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"time"
)

// FramedTCPSocket implements interfaces.TransportConnection.
// It uses a 4-byte BigEndian length header for every write.
type FramedTCPSocket struct {
	Conn    net.Conn
	reader  *bufio.Reader
	Timeout time.Duration
}

// -----------------------------------------------------------------------------

func NewFramedTCPSocket(conn net.Conn, timeout time.Duration) *FramedTCPSocket {
	return &FramedTCPSocket{
		Conn:    conn,
		reader:  bufio.NewReader(conn),
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
// SAFE UPDATE: Uses Peek/Discard to ensure header is not lost if buffer is too short.
func (s *FramedTCPSocket) Read(p []byte) (n int, err error) {
	if s.Timeout > 0 {
		_ = s.Conn.SetReadDeadline(time.Now().Add(s.Timeout))
	}

	// 1. Peek content check
	// We need 4 bytes for header.
	header, err := s.reader.Peek(4)
	if err != nil {
		return 0, err
	}

	// 2. Decode Length
	length := binary.BigEndian.Uint32(header)

	// 3. Check Buffer Size BEFORE consuming header
	if uint32(len(p)) < length {
		return 0, io.ErrShortBuffer
	}

	// 4. Safe to proceed: Consume Header
	if _, err := s.reader.Discard(4); err != nil {
		// Should not happen if Peek succeeded, unless connection closed in between micro-ops
		return 0, err
	}

	// 5. Read Body
	// We use ReadFull directly on the bufio reader.
	return io.ReadFull(s.reader, p[:length])
}

// -----------------------------------------------------------------------------

// ReadMessage implements the dynamic read.
func (s *FramedTCPSocket) ReadMessage() ([]byte, error) {
	if s.Timeout > 0 {
		_ = s.Conn.SetReadDeadline(time.Now().Add(s.Timeout))
	}

	// 1. Read Length (Must use reader to avoid skipping buffered data)
	header := make([]byte, 4)
	if _, err := io.ReadFull(s.reader, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)

	// 2. Allocate exact size
	buf := make([]byte, length)

	// 3. Read Body
	if _, err := io.ReadFull(s.reader, buf); err != nil {
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
