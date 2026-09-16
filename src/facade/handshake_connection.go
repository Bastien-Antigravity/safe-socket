package facade

// =============================================================================
// ESSENTIAL PROCESS:
// Decorates a TransportConnection to cache the authenticated peer HelloMsg identity
// established during initial TCP/TLS/SHM handshake negotiation.
//
// DATA FLOW:
// 1. Input: Underlying TransportConnection and unmarshaled Cap'n Proto HelloMsg.
// 2. Logic: Embeds the transport connection while preserving verified peer metadata.
// 3. Output: Readily accessible peer identity via safesocket.GetIdentity.
//
// KEY PARAMETERS:
// - Identity: Verified Cap'n Proto HelloMsg identity struct.
// =============================================================================

import (
	"time"

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

func (h *HandshakeConnection) SetIdleTimeout(d time.Duration) error {
	return h.TransportConnection.SetIdleTimeout(d)
}
