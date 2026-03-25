package interfaces

import (
	"io"
	"net"
	"time"
)

// TransportConnection defines a connection that can handle low-level I/O.
// It is responsible for framing or packetizing the data if necessary (e.g., length-prefixed).
//
// Note regarding Client vs Server:
//   - For Clients: This interface is returned by specific "Dial" or "Connect" functions in the transports package.
//     There represents the active connection after dialing is complete.
//   - For Servers: This interface is returned by the Listener.Accept() method.
type TransportConnection interface {
	io.ReadWriteCloser
	// Extended methods for address information
	LocalAddr() net.Addr
	RemoteAddr() net.Addr

	// Read implemented in io.ReadWriteCloser
	// ReadMessage reads a complete frame/packet and returns it in a newly allocated buffer.
	ReadMessage() ([]byte, error)

	// Deadlines
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// TransportListener defines a listener that waits for incoming connections.
type TransportListener interface {
	Accept() (TransportConnection, error)
	Close() error
	Addr() net.Addr
}
