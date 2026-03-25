package interfaces

// TransportType defines the underlying transport mechanism.
type TransportType string

const (
	TransportFramedTCP TransportType = "FramedTCP"
	TransportShm       TransportType = "SharedMemory"
	TransportUDP       TransportType = "UDP"
)

// ProtocolType defines the application-level handshake or startup protocol.
type ProtocolType string

const (
	ProtocolNone  ProtocolType = "none"
	ProtocolHello ProtocolType = "hello"
)

// SocketProfile defines the behavior for a connection strategy.
type SocketProfile interface {
	// -------------------------------------------------------------------------
	// GetName returns the unique identifier for this profile.
	GetName() string

	// -------------------------------------------------------------------------
	// GetAddress returns the network address (IP:Port or Path).
	GetAddress() string

	// -------------------------------------------------------------------------
	// GetTransport returns the type of transport to use.
	GetTransport() TransportType

	// -------------------------------------------------------------------------
	// GetProtocol returns the startup protocol to execute.
	GetProtocol() ProtocolType

	// -------------------------------------------------------------------------
	// GetConnectTimeout returns the timeout for establishing the connection in milliseconds.
	GetConnectTimeout() int
}
