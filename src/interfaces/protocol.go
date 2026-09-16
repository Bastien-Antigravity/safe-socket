package interfaces

// =============================================================================
// ESSENTIAL PROCESS:
// Defines the Protocol interface contract, decoupling protocol handshake negotiation
// and packet encapsulation from underlying physical network transports.
//
// DATA FLOW:
// 1. Input: Active TransportConnection, SocketProfile, and SocketConfig.
// 2. Logic: Initiates handshakes, waits for authentication, and encapsulates/decapsulates packets.
// 3. Output: Returns verified HelloMsg schemas and unwrapped payload data.
//
// KEY PARAMETERS:
// - Protocol: Unified interface for connection-oriented and stateless handshakes.
// =============================================================================

import (
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/schemas"
)

// Protocol defines the application-level handshake or initial interaction logic.
// It decouples "What we say when we connect" from "How we connect".
type Protocol interface {
	// -------------------------------------------------------------------------
	// Initiate executes the handshake sequence or protocol logic (Client).
	Initiate(conn TransportConnection, profile SocketProfile, config models.SocketConfig) error

	// -------------------------------------------------------------------------
	// WaitInitiation waits for and processes the handshake sequence (Server).
	WaitInitiation(conn TransportConnection) (*schemas.HelloMsg, error)
}
