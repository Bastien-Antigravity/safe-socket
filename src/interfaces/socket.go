package interfaces

import (
	"time"
)

// -----------------------------------------------------------------------------

// SocketType defines the role of the socket (Client or Server).
type SocketType int

const (
	SocketTypeClient SocketType = iota
	SocketTypeServer
)

// -----------------------------------------------------------------------------

// Socket defines the unified high-level interface for a safe-socket connection.
// It encompasses both Client and Server operations. Implementations should return
// errors for unsupported operations based on their role.
type Socket interface {
	// Common
	Close() error
	SetLogger(logger *Logger)

	// Client Methods
	Open() error
	Send(data []byte) error
	// used to complies with io.Writer interface, in logger
	Write(data []byte) (int, error)
	// Read into a fixed buffer (complies with io.Reader)
	Read(p []byte) (int, error)
	// custom Read method, simpler to use than Read(p []byte)
	Receive() ([]byte, error)

	// Deadlines (Simulating net.Conn behavior)
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error

	// Server Methods
	Listen() error
	Accept() (TransportConnection, error)
	// Addr() net.Addr // Optional: might be useful to expose listener address
}
