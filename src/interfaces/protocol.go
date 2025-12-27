package interfaces

import (
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/schemas"
)

// Protocol defines the application-level handshake or initial interaction logic.
// It decouples "What we say when we connect" from "How we connect".
type Protocol interface {
	// SayHello executes the handshake sequence or protocol logic (Client).
	SayHello(conn TransportConnection, profile SocketProfile, config models.SocketConfig) error

	// WaitHello waits for and processes the handshake sequence (Server).
	WaitHello(conn TransportConnection) (*schemas.HelloMsg, error)
}
