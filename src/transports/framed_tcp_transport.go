package transports

import (
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
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

// Connect dialer helper for FramedTCPSocket.
func Connect(address string, timeout time.Duration) (interfaces.TransportConnection, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, err
	}
	// Note: We deliberately use the 'timeout' for both connection AND subsequent read/writes
	return NewFramedTCPSocket(conn, timeout), nil
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

func (s *FramedTCPSocket) Close() error {
	return s.Conn.Close()
}
