package interfaces

import "io"

// TransportConnection defines a connection that can handle low-level I/O.
// It is responsible for framing or packetizing the data if necessary (e.g., length-prefixed).
type TransportConnection interface {
	io.ReadWriteCloser
	// We might add more specific transport controls here if needed.
}
