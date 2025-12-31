package facade

import (
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/schemas"
)

// HandshakeConnection wraps a transport and exposes the initial handshake identity.
// It implements interfaces.TransportConnection by embedding the underlying connection.
type HandshakeConnection struct {
	interfaces.TransportConnection
	Identity *schemas.HelloMsg
}

// -----------------------------------------------------------------------------

// LocalAddr and RemoteAddr are promoted automatically by embedding,
// but we can override them if we wanted to (we don't need to).

// We explicitly implement the interface methods to be safe, though embedding handles it.
// The only addition is the Identity field.

// Ensure HandshakeConnection implements TransportConnection
var _ interfaces.TransportConnection = (*HandshakeConnection)(nil)

// -----------------------------------------------------------------------------

func NewHandshakeConnection(conn interfaces.TransportConnection, identity *schemas.HelloMsg) *HandshakeConnection {
	return &HandshakeConnection{
		TransportConnection: conn,
		Identity:            identity,
	}
}

// LocalAddr and RemoteAddr are promoted automatically by embedding.
