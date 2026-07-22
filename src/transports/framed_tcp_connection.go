package transports

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"time"
)

// MaxPayloadSize defines the upper limit for incoming frames (default 64MB).
const MaxPayloadSize = 64 * 1024 * 1024

// FramedTCPSocket implements interfaces.TransportConnection.
// It uses a 4-byte BigEndian length header for every write.
type FramedTCPSocket struct {
	Conn        net.Conn
	reader      *bufio.Reader
	idleTimeout time.Duration
}

// -----------------------------------------------------------------------------

func NewFramedTCPSocket(conn net.Conn, timeout time.Duration) *FramedTCPSocket {
	s := &FramedTCPSocket{
		Conn:        conn,
		idleTimeout: timeout,
	}

	if conn != nil {
		s.reader = bufio.NewReader(conn)
		s.refreshReadDeadline()
		s.refreshWriteDeadline()
	}

	return s
}

func (s *FramedTCPSocket) refreshReadDeadline() {
	if s.idleTimeout > 0 && s.Conn != nil {
		_ = s.Conn.SetReadDeadline(time.Now().Add(s.idleTimeout))
	}
}

func (s *FramedTCPSocket) refreshWriteDeadline() {
	if s.idleTimeout > 0 && s.Conn != nil {
		_ = s.Conn.SetWriteDeadline(time.Now().Add(s.idleTimeout))
	}
}

// SetIdleTimeout updates the internal idle timeout and refreshes current deadlines.
func (s *FramedTCPSocket) SetIdleTimeout(d time.Duration) error {
	s.idleTimeout = d
	if s.Conn == nil {
		return nil
	}
	if d == 0 {
		// Clear deadlines once and for all for 'forever' mode
		_ = s.Conn.SetDeadline(time.Time{})
	} else {
		s.refreshReadDeadline()
		s.refreshWriteDeadline()
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetKeepAlive enables TCP keepalive with the specified period.
func (s *FramedTCPSocket) SetKeepAlive(period time.Duration) error {
	if s.Conn == nil {
		return nil
	}
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
	if s.Conn == nil {
		return nil
	}
	if tcpConn, ok := s.Conn.(*net.TCPConn); ok {
		return tcpConn.SetNoDelay(enabled)
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetReadBuffer sets the size of the operating system's receive buffer.
func (s *FramedTCPSocket) SetReadBuffer(bytes int) error {
	if s.Conn == nil {
		return nil
	}
	if tcpConn, ok := s.Conn.(*net.TCPConn); ok {
		return tcpConn.SetReadBuffer(bytes)
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetWriteBuffer sets the size of the operating system's transmit buffer.
func (s *FramedTCPSocket) SetWriteBuffer(bytes int) error {
	if s.Conn == nil {
		return nil
	}
	if tcpConn, ok := s.Conn.(*net.TCPConn); ok {
		return tcpConn.SetWriteBuffer(bytes)
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetDeadline sets the read and write deadlines associated with the connection.
func (s *FramedTCPSocket) SetDeadline(t time.Time) error {
	if s.Conn == nil {
		return nil
	}
	return s.Conn.SetDeadline(t)
}

// -----------------------------------------------------------------------------

// SetReadDeadline sets the deadline for future Read calls.
func (s *FramedTCPSocket) SetReadDeadline(t time.Time) error {
	if s.Conn == nil {
		return nil
	}
	return s.Conn.SetReadDeadline(t)
}

// -----------------------------------------------------------------------------

// SetWriteDeadline sets the deadline for future Write calls.
func (s *FramedTCPSocket) SetWriteDeadline(t time.Time) error {
	if s.Conn == nil {
		return nil
	}
	return s.Conn.SetWriteDeadline(t)
}

// -----------------------------------------------------------------------------

// Write prepends length and writes data.
func (s *FramedTCPSocket) Write(p []byte) (n int, err error) {
	s.refreshWriteDeadline()
	if s.Conn == nil {
		return 0, io.ErrClosedPipe
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
// HEARTBEAT UPDATE: Automatically skips frames with length 0.
func (s *FramedTCPSocket) Read(p []byte) (n int, err error) {
	for {
		s.refreshReadDeadline()
		if s.Conn == nil || s.reader == nil {
			return 0, io.EOF
		}
		// 1. Peek content check
		// We need 4 bytes for header.
		header, err := s.reader.Peek(4)
		if err != nil {
			return 0, err
		}

		// 2. Decode Length
		length := binary.BigEndian.Uint32(header)

		// OOM PROTECTION: Reject oversized frames before allocation
		if length > MaxPayloadSize {
			return 0, io.ErrUnexpectedEOF // Or custom ErrPayloadTooLarge
		}

		// HEARTBEAT: Length 0 frames are heartbeats. Consume and continue.
		if length == 0 {
			if _, err := s.reader.Discard(4); err != nil {
				return 0, err
			}
			continue
		}

		// 3. Check Buffer Size BEFORE consuming header
		if uint32(len(p)) < length {
			return 0, io.ErrShortBuffer
		}

		// 4. Safe to proceed: Consume Header
		if _, err := s.reader.Discard(4); err != nil {
			return 0, err
		}

		// 5. Read Body
		return io.ReadFull(s.reader, p[:length])
	}
}

// -----------------------------------------------------------------------------

// ReadMessage implements the dynamic read.
// HEARTBEAT UPDATE: Automatically skips frames with length 0.
func (s *FramedTCPSocket) ReadMessage() ([]byte, error) {
	for {
		s.refreshReadDeadline()
		if s.Conn == nil || s.reader == nil {
			return nil, io.EOF
		}
		// 1. Read Length
		header := make([]byte, 4)
		if _, err := io.ReadFull(s.reader, header); err != nil {
			return nil, err
		}
		length := binary.BigEndian.Uint32(header)

		// OOM PROTECTION: Reject oversized frames before allocation
		if length > MaxPayloadSize {
			return nil, io.ErrUnexpectedEOF
		}

		// HEARTBEAT: Length 0 frames are heartbeats. Consume and continue.
		if length == 0 {
			continue
		}

		// 2. Allocate exact size
		buf := make([]byte, length)

		// 3. Read Body
		if _, err := io.ReadFull(s.reader, buf); err != nil {
			return nil, err
		}

		return buf, nil
	}
}

// -----------------------------------------------------------------------------

func (s *FramedTCPSocket) Close() error {
	if s.Conn == nil {
		return nil
	}
	return s.Conn.Close()
}

// -----------------------------------------------------------------------------

// LocalAddr returns the local network address.
func (s *FramedTCPSocket) LocalAddr() net.Addr {
	if s.Conn == nil {
		return nil
	}
	return s.Conn.LocalAddr()
}

// -----------------------------------------------------------------------------

// RemoteAddr returns the remote network address.
func (s *FramedTCPSocket) RemoteAddr() net.Addr {
	if s.Conn == nil {
		return nil
	}
	return s.Conn.RemoteAddr()
}
