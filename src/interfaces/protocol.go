package interfaces

import (
	"github.com/Bastien-Antigravity/safe-socket/src/models"
)

// Protocol defines the application-level handshake or initial interaction logic.
// It decouples "What we say when we connect" from "How we connect".
type Protocol interface {
	// Execute executes the handshake sequence or protocol logic.
	Execute(conn TransportConnection, profile SocketProfile, config models.SocketConfig) error
}
