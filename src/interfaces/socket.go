package interfaces

// =============================================================================
// ESSENTIAL PROCESS:
// Defines role-segregated ClientSocket and ServerSocket interfaces alongside the
// legacy unified Socket contract for high-level safe-socket operations.
//
// DATA FLOW:
// 1. Input: Higher-level consumer requests (Open, Listen, Accept, Send, Receive).
// 2. Logic: Enforces clean client vs server contracts implemented by SocketClient and SocketServer.
// 3. Output: Operates socket state and delivers received payloads.
//
// KEY PARAMETERS:
// - ClientSocket: Client-side dial, I/O, and deadline contract.
// - ServerSocket: Server-side listener, accept, and shutdown contract.
// - Socket: Unified backward-compatible interface combining both roles.
// =============================================================================

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

// ClientSocket defines the role-segregated contract for client-side safe-socket connections.
type ClientSocket interface {
	Close() error
	SetLogger(logger Logger)

	Open() error
	Send(data []byte) error
	Write(data []byte) (int, error)
	Read(p []byte) (int, error)
	Receive() ([]byte, error)
	ReadMessage() ([]byte, error)

	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetIdleTimeout(d time.Duration) error
}

// -----------------------------------------------------------------------------

// ServerSocket defines the role-segregated contract for server-side safe-socket listeners.
type ServerSocket interface {
	Close() error
	SetLogger(logger Logger)

	Listen() error
	Accept() (TransportConnection, error)
	GetAddr() (string, error)
	SetIdleTimeout(d time.Duration) error
}

// -----------------------------------------------------------------------------

// Socket defines the unified high-level interface for a safe-socket connection.
// Retained for backward compatibility; prefer ClientSocket or ServerSocket for new code.
type Socket interface {
	// Common
	Close() error
	SetLogger(logger Logger)

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

	// IdleTimeout controls the 'activity-refresh' duration.
	// Every successful Read/Write will refresh the absolute deadline to Now() + d.
	// Set to 0 to disable automated refreshing.
	SetIdleTimeout(d time.Duration) error

	// Server Methods
	Listen() error
	Accept() (TransportConnection, error)
}
