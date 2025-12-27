package safesocket

import (
	"github.com/Bastien-Antigravity/safe-socket/src/factory"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
)

// Version defines the current library version.
const Version = "1.1.0"

// Socket is an alias for the Facade to simplify usage.
type Socket = interfaces.Socket

// Create creates a new safe-socket connection using a named profile.
// profileName: "tcp-hello", "tcp"
// address: destination address (e.g., "127.0.0.1:8080")
// publicIP: your public IP (for handshake protocols)
func Create(profileName, address, publicIP string) (Socket, error) {
	// We delegate to the factory, but we need to ensure the factory implementation
	// is accessible and works with the types we expose if needed.
	// Since factory returns *facade.SocketFacade, and Socket is an alias, this works.
	return factory.Create(profileName, address, publicIP)
}

// Expose other useful types if necessary
type SocketConfig = models.SocketConfig
type SocketProfile = interfaces.SocketProfile
type TransportType = interfaces.TransportType
type ProtocolType = interfaces.ProtocolType

// Explicitly export Transport/Protocol constants via variables or just let users import interfaces?
// For a simple lib, letting them import "https://github.com/toto1234567890/safe-socket/src/interfaces" is okay,
// but aliasing commonly used ones is nicer.

const (
	TransportFramedTCP = interfaces.TransportFramedTCP
	TransportUDP       = interfaces.TransportUDP
)
