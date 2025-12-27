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
	ProtocolNone     ProtocolType = "None"
	ProtocolSayHello ProtocolType = "SayHello"
)

// SocketProfile defines the behavior for a connection strategy.
type SocketProfile interface {
	GetName() string
	GetAddress() string
	GetTransport() TransportType
	GetProtocol() ProtocolType
	GetConnectTimeout() int
}
