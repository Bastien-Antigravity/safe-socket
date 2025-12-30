package safesocket

import (
	"github.com/Bastien-Antigravity/safe-socket/src/factory"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
)

// Version defines the current library version.
const Version = "1.1.1"

// Socket is an alias for the Facade to simplify usage.
type Socket = interfaces.Socket

// SocketType defines the role of the socket (Client or Server).
type SocketType = interfaces.SocketType

const (
	SocketTypeClient = interfaces.SocketTypeClient
	SocketTypeServer = interfaces.SocketTypeServer
)

// -----------------------------------------------------------------------------

// Create creates a new safe-socket connection using a named profile.
//
// Parameters:
//   - profileName: "tcp", "tcp-hello", "udp", "udp-hello", "shm", "shm-hello"
//   - address: destination address ("IP:Port" or "FilePath" for SHM)
//   - publicIP: your public IP (Required for "hello" protocols)
//   - socketType: SocketTypeClient or SocketTypeServer
//   - autoConnect: if true, immediately calls Open() / Listen()
func Create(profileName, address, publicIP string, socketType SocketType, autoConnect bool) (Socket, error) {
	// We delegate to the factory
	return factory.Create(profileName, address, publicIP, socketType, autoConnect)
}

// -----------------------------------------------------------------------------

// Expose other useful types if necessary
type SocketConfig = models.SocketConfig
type SocketProfile = interfaces.SocketProfile
type TransportType = interfaces.TransportType
type ProtocolType = interfaces.ProtocolType

// -----------------------------------------------------------------------------

// Explicitly export Transport/Protocol constants via variables or just let users import interfaces?
// For a simple lib, letting them import "https://github.com/toto1234567890/safe-socket/src/interfaces" is okay,
// but aliasing commonly used ones is nicer.

const (
	TransportFramedTCP = interfaces.TransportFramedTCP
	TransportUDP       = interfaces.TransportUDP
	TransportSHM       = interfaces.TransportShm
)
