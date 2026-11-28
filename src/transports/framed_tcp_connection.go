package transports

// =============================================================================
// ESSENTIAL PROCESS:
// Implements length-prefixed TCP streaming transport (FramedTCPSocket),
// framing application payloads with 4-byte big-endian integers and handling
// framed packet read and write streams with idle deadline refreshes.
//
// DATA FLOW:
// 1. Input: Arbitrary payload byte slices from upper layers.
// 2. Logic: Formats 4-byte big-endian length prefix; writes header followed by payload;
//    reads framed packets with buffer pooling and size verification.
// 3. Output: Raw payload slices stripped of transport framing.
//
// KEY PARAMETERS:
// - FramedTCPSocket: TransportConnection implementation wrapping net.Conn.
// - MaxPayloadSize: 64MB frame boundary limit preventing memory exhaustion.
// =============================================================================

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// MaxPayloadSize defines the upper limit for incoming frames (default 64MB).
const MaxPayloadSize = 64 * 1024 * 1024

// FramedTCPSocket implements interfaces.TransportConnection.
// It uses a 4-byte BigEndian length header for every write.
type FramedTCPSocket struct {
	Conn                net.Conn
	reader              *bufio.Reader
	idleTimeoutNs       atomic.Int64
	customReadDeadline  atomic.Bool
	customWriteDeadline atomic.Bool
	writeMu             sync.Mutex
}

// -----------------------------------------------------------------------------

func NewFramedTCPSocket(conn net.Conn, timeout time.Duration) *FramedTCPSocket {
	s := &FramedTCPSocket{
		Conn: conn,
	}
	s.idleTimeoutNs.Store(int64(timeout))

	if conn != nil {
		s.reader = bufio.NewReader(conn)
		s.refreshReadDeadline()
		s.refreshWriteDeadline()
	}

	return s
}

func (s *FramedTCPSocket) refreshReadDeadline() {
	timeout := time.Duration(s.idleTimeoutNs.Load())
	if timeout > 0 && s.Conn != nil && !s.customReadDeadline.Load() {
		_ = s.Conn.SetReadDeadline(time.Now().Add(timeout))
	}
}

func (s *FramedTCPSocket) refreshWriteDeadline() {
	timeout := time.Duration(s.idleTimeoutNs.Load())
	if timeout > 0 && s.Conn != nil && !s.customWriteDeadline.Load() {
		_ = s.Conn.SetWriteDeadline(time.Now().Add(timeout))
	}
}

// SetIdleTimeout updates the internal idle timeout and refreshes current deadlines.
func (s *FramedTCPSocket) SetIdleTimeout(d time.Duration) error {
	s.idleTimeoutNs.Store(int64(d))
	s.customReadDeadline.Store(false)
	s.customWriteDeadline.Store(false)
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

func (s *FramedTCPSocket) underlyingTCPConn() *net.TCPConn {
	if s.Conn == nil {
		return nil
	}
	if tcpConn, ok := s.Conn.(*net.TCPConn); ok {
		return tcpConn
	}
	if tlsConn, ok := s.Conn.(*tls.Conn); ok {
		if tcpConn, ok := tlsConn.NetConn().(*net.TCPConn); ok {
			return tcpConn
		}
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetKeepAlive enables TCP keepalive with the specified period.
func (s *FramedTCPSocket) SetKeepAlive(period time.Duration) error {
	if tcpConn := s.underlyingTCPConn(); tcpConn != nil {
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
	if tcpConn := s.underlyingTCPConn(); tcpConn != nil {
		return tcpConn.SetNoDelay(enabled)
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetReadBuffer sets the size of the operating system's receive buffer.
func (s *FramedTCPSocket) SetReadBuffer(bytes int) error {
	if tcpConn := s.underlyingTCPConn(); tcpConn != nil {
		return tcpConn.SetReadBuffer(bytes)
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetWriteBuffer sets the size of the operating system's transmit buffer.
func (s *FramedTCPSocket) SetWriteBuffer(bytes int) error {
	if tcpConn := s.underlyingTCPConn(); tcpConn != nil {
		return tcpConn.SetWriteBuffer(bytes)
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetDeadline sets the read and write deadlines associated with the connection.
func (s *FramedTCPSocket) SetDeadline(t time.Time) error {
	isCustom := !t.IsZero()
	s.customReadDeadline.Store(isCustom)
	s.customWriteDeadline.Store(isCustom)
	if s.Conn == nil {
		return nil
	}
	return s.Conn.SetDeadline(t)
}

// -----------------------------------------------------------------------------

// SetReadDeadline sets the deadline for future Read calls.
func (s *FramedTCPSocket) SetReadDeadline(t time.Time) error {
	s.customReadDeadline.Store(!t.IsZero())
	if s.Conn == nil {
		return nil
	}
	return s.Conn.SetReadDeadline(t)
}

// -----------------------------------------------------------------------------

// SetWriteDeadline sets the deadline for future Write calls.
func (s *FramedTCPSocket) SetWriteDeadline(t time.Time) error {
	s.customWriteDeadline.Store(!t.IsZero())
	if s.Conn == nil {
		return nil
	}
	return s.Conn.SetWriteDeadline(t)
}

// -----------------------------------------------------------------------------

// Write prepends the 4-byte length header and transmits the full frame in a single
// synchronized write. Small frames (<= 508 bytes, including 0-byte heartbeats) use a
// stack buffer for zero heap allocations; large cleartext TCP frames use scatter-gather
// writev(2) via net.Buffers; TLS frames coalesce into a single record buffer.
// If writing fails partway, the connection is closed immediately so partial frames
// can never desynchronize the stream.
func (s *FramedTCPSocket) Write(p []byte) (n int, err error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.refreshWriteDeadline()
	if s.Conn == nil {
		return 0, io.ErrClosedPipe
	}

	payloadLen := len(p)

	// Fast Path 1: Small frames (<= 508B, including 0B heartbeats) use stack allocation (0 B/op)
	if payloadLen <= 508 {
		var stackBuf [512]byte
		frame := stackBuf[:4+payloadLen]
		binary.BigEndian.PutUint32(frame[:4], uint32(payloadLen))
		if payloadLen > 0 {
			copy(frame[4:], p)
		}
		totalWritten := 0
		for totalWritten < len(frame) {
			w, err := s.Conn.Write(frame[totalWritten:])
			if err != nil {
				_ = s.Conn.Close()
				return 0, err
			}
			if w == 0 {
				_ = s.Conn.Close()
				return 0, io.ErrShortWrite
			}
			totalWritten += w
		}
		return payloadLen, nil
	}

	// Fast Path 2: Large frames on cleartext *net.TCPConn use writev(2) scatter-gather (zero payload copy)
	if _, isTCP := s.Conn.(*net.TCPConn); isTCP {
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], uint32(payloadLen))
		buffers := net.Buffers{header[:], p}
		written, err := buffers.WriteTo(s.Conn)
		if err != nil {
			_ = s.Conn.Close()
			return 0, err
		}
		if written != int64(4+payloadLen) {
			_ = s.Conn.Close()
			return 0, io.ErrShortWrite
		}
		return payloadLen, nil
	}

	// Path 3: Large frames on *tls.Conn or custom net.Conn coalesce into a single write call
	frame := make([]byte, 4+payloadLen)
	binary.BigEndian.PutUint32(frame[:4], uint32(payloadLen))
	copy(frame[4:], p)

	totalWritten := 0
	for totalWritten < len(frame) {
		w, err := s.Conn.Write(frame[totalWritten:])
		if err != nil {
			_ = s.Conn.Close()
			return 0, err
		}
		if w == 0 {
			_ = s.Conn.Close()
			return 0, io.ErrShortWrite
		}
		totalWritten += w
	}

	return payloadLen, nil
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
	var header [4]byte
	for {
		s.refreshReadDeadline()
		if s.Conn == nil || s.reader == nil {
			return nil, io.EOF
		}
		// 1. Read Length into stack-allocated array
		if _, err := io.ReadFull(s.reader, header[:]); err != nil {
			return nil, err
		}
		length := binary.BigEndian.Uint32(header[:])

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
