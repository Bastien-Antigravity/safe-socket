package factory

import (
	"fmt"

	"github.com/Bastien-Antigravity/safe-socket/src/facade"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/profiles"
)

// -----------------------------------------------------------------------------

// Create is the simplified library entry point.
// profileName: "tcp-hello" (currently supported)
// address: destination address to connect to (e.g. "127.0.0.1:8081")
// publicIP: this node's public IP (for protocol handshake)
// Returns an open Socket interface or an error.
func Create(profileName, address, publicIP string) (interfaces.Socket, error) {
	var p interfaces.SocketProfile

	switch profileName {
	case "tcp-hello":
		// Default timeout 5 seconds for library usage
		p = profiles.NewTcpHelloProfile("TcpClient", address, 5000)
	case "tcp":
		p = profiles.NewTcpProfile("TcpRaw", address, 5000)
	default:
		return nil, fmt.Errorf("unknown profile: %s", profileName)
	}

	config := models.SocketConfig{
		PublicIP: publicIP,
	}

	return CreateSocket(p, config)
}

// CreateSocket constructs a SocketFacade based on the provided SocketProfile and opens it.
func CreateSocket(p interfaces.SocketProfile, config models.SocketConfig) (interfaces.Socket, error) {
	// 1. Create the Facade with configuration
	socket := facade.NewSocketFacade(p, config)

	// 2. Open the connection (Lazy Init / Auto Connect)
	if err := socket.Open(); err != nil {
		return nil, err
	}

	return socket, nil
}
