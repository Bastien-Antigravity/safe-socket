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
//
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
	config := models.SocketConfig{
		PublicIP: publicIP,
	}
	return CreateWithConfig(profileName, address, config, socketType, autoConnect)
}

// CreateWithConfig is the extended library entry point allowing full configuration.
//
// Parameters:
//   - profileName: e.g. "tcp-hello"
//   - address: destination address
//   - config: models.SocketConfig (allows setting Deadlines, PublicIP, etc.)
//   - socketType: "client" or "server"
//   - autoConnect: if true, automatically calls Open() / Listen()
func CreateWithConfig(profileName, address string, config models.SocketConfig, socketType string, autoConnect bool) (interfaces.Socket, error) {
	st, err := parseSocketType(socketType)
	if err != nil {
		return nil, err
	}

	p, err := createProfile(profileName, address, st)
	if err != nil {
		return nil, err
	}

	if autoConnect {
		return CreateOpenSocket(p, config, socketType)
	}

	return CreateSocket(p, config, socketType)
}

func createProfile(profileName, address string, st interfaces.SocketType) (interfaces.SocketProfile, error) {
	switch profileName {
	case "tcp-hello":
		// Default timeout 5 seconds for library usage
		if st == interfaces.SocketTypeClient {
			return profiles.NewTcpHelloClientProfile("TcpClient", address, 5000), nil
		}
		return profiles.NewTcpHelloServerProfile("TcpServer", address, 5000), nil
	case "tcp":
		if st == interfaces.SocketTypeClient {
			return profiles.NewTcpClientProfile("TcpRaw", address, 5000), nil
		}
		return profiles.NewTcpServerProfile("TcpRaw", address, 5000), nil

	// UDP Support
	case "udp":
		return profiles.NewUdpProfile("UdpRaw", address, 5000), nil
	case "udp-hello":
		return profiles.NewUdpHelloProfile("UdpHello", address, 5000), nil

	// SHM Support
	case "shm":
		return profiles.NewShmProfile(address, 5000), nil // address is path
	case "shm-hello":
		return profiles.NewShmHelloProfile(address, 5000), nil
	default:
		return nil, fmt.Errorf("unknown profile: %s", profileName)
	}
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
	return facade.NewSocketServer(p, config), nil
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
