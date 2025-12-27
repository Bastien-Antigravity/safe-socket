package interfaces

// -----------------------------------------------------------------------------

// Socket defines the high-level interface for a safe-socket connection.
type Socket interface {
	// Open establishes the connection using the configured transport and protocol.
	Open() error

	// Send transmits raw data over the socket.
	Send(data []byte) error

	// Receive reads raw data from the socket.
	Receive(buf []byte) (int, error)

	// Close terminates the connection.
	Close() error
}
