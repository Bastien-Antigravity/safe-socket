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
// Create is the simplified library entry point for creating any type of Socket.
//
// Parameters:
//   - profileName: e.g. "tcp-hello" (currently supported)
//   - address: destination address to connect to (Client) or bind to (Server) (e.g. "127.0.0.1:8081")
//   - publicIP: this node's public IP (used for protocol handshake data)
//   - socketType: "client" or "server" (case-insensitive)
//   - autoConnect: if true, automatically calls Open() (Client) or Listen() (Server)
//
// Returns:
//
//	An interfaces.Socket which can be used to Send/Receive (Client) or Accept (Server).
func Create(profileName, address, publicIP string, socketType string, autoConnect bool) (interfaces.Socket, error) {
	st, err := parseSocketType(socketType)
	if err != nil {
		return nil, err
	}

	var p interfaces.SocketProfile

	switch profileName {
	case "tcp-hello":
		// Default timeout 5 seconds for library usage
		if st == interfaces.SocketTypeClient {
			p = profiles.NewTcpHelloClientProfile("TcpClient", address, 5000)
		} else {
			p = profiles.NewTcpHelloServerProfile("TcpServer", address, 5000)
		}
	case "tcp":
		if st == interfaces.SocketTypeClient {
			p = profiles.NewTcpClientProfile("TcpRaw", address, 5000)
		} else {
			p = profiles.NewTcpServerProfile("TcpRaw", address, 5000)
		}

	// UDP Support
	case "udp":
		p = profiles.NewUdpProfile("UdpRaw", address, 5000)
	case "udp-hello":
		p = profiles.NewUdpHelloProfile("UdpHello", address, 5000)

	// SHM Support
	case "shm":
		p = profiles.NewShmProfile(address, 5000) // address is path
	case "shm-hello":
		p = profiles.NewShmHelloProfile(address, 5000)
	default:
		return nil, fmt.Errorf("unknown profile: %s", profileName)
	}

	config := models.SocketConfig{
		PublicIP: publicIP,
	}

	if autoConnect {
		return CreateOpenSocket(p, config, socketType)
	}

	return CreateSocket(p, config, socketType)
}

// -----------------------------------------------------------------------------

// CreateOpenSocket constructs a Socket based on the provided parameters and opens/listens it.
func CreateOpenSocket(p interfaces.SocketProfile, config models.SocketConfig, socketType string) (interfaces.Socket, error) {
	// 1. Create the Socket facade
	socket, err := CreateSocket(p, config, socketType)
	if err != nil {
		return nil, err
	}

	st, _ := parseSocketType(socketType) // Already validated if coming from internal flow, but public API needs check

	// 2. Open (Client) or Listen (Server)
	if st == interfaces.SocketTypeClient {
		if err := socket.Open(); err != nil {
			return nil, err
		}
	} else {
		if err := socket.Listen(); err != nil {
			return nil, err
		}
	}

	return socket, nil
}

// -----------------------------------------------------------------------------

// CreateSocket constructs a Socket facade based on the provided SocketProfile.
// It returns the socket in a closed/unlistening state.
// Useful for connection pools or when deferred connection is required.
func CreateSocket(p interfaces.SocketProfile, config models.SocketConfig, socketType string) (interfaces.Socket, error) {
	st, err := parseSocketType(socketType)
	if err != nil {
		return nil, err
	}

	if st == interfaces.SocketTypeClient {
		return facade.NewSocketClient(p, config), nil
	}
	return facade.NewSocketServer(p), nil
}

func parseSocketType(t string) (interfaces.SocketType, error) {
	switch t {
	case "client", "CLIENT":
		return interfaces.SocketTypeClient, nil
	case "server", "SERVER":
		return interfaces.SocketTypeServer, nil
	default:
		return 0, fmt.Errorf("invalid socket type: %s (expected 'client' or 'server')", t)
	}
}
