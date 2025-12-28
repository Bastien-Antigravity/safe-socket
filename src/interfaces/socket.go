package interfaces

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

	// Client Methods
	Open() error
	Send(data []byte) error
	Receive(buf []byte) (int, error)

	// Server Methods
	Listen() error
	Accept() (TransportConnection, error)
	// Addr() net.Addr // Optional: might be useful to expose listener address
}
