package interfaces

import (
	"io"
	"net"
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
}

// TransportListener defines a listener that waits for incoming connections.
type TransportListener interface {
	Accept() (TransportConnection, error)
	Close() error
	Addr() net.Addr
}
