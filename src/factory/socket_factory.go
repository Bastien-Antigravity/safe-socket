package factory

import (
	"fmt"

	"github.com/Bastien-Antigravity/safe-socket/src/facade"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/profiles"
	"strings"
	"time"
)

const (
	// DefaultHandshakeTimeout is used for network transports (TCP/UDP)
	DefaultHandshakeTimeout = 500 // 0.5s
	// DefaultLocalHandshakeTimeout is used for loopback (127.0.0.1/localhost)
	DefaultLocalHandshakeTimeout = 200 // 0.2s
	// DefaultShmHandshakeTimeout is used for Shared Memory (SHM)
	DefaultShmHandshakeTimeout = 100 // 100ms
)

// -----------------------------------------------------------------------------

// Create is the simplified library entry point for creating any type of Socket.
//
// Parameters:
//   - profileName: e.g. "tcp-hello" (currently supported)
//   - address: destination address to connect to (Client) or bind to (Server) (e.g. "127.0.0.1:8081")
//   - publicIP: this node's public IP (used for protocol handshake data, now optional)
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

	// 1. Determine Identity and Profile Key
	profileKey := profileName
	if parts := strings.Split(profileName, ":"); len(parts) > 1 {
		profileKey = parts[0]
	}

	// 2. Determine Timeout
	timeout := int(config.HandshakeTimeout.Milliseconds())
	if timeout <= 0 {
		isLocal := strings.Contains(address, "localhost") ||
			strings.Contains(address, "127.0.0.1") ||
			strings.Contains(address, "::1")

		if strings.HasPrefix(profileKey, "shm") {
			timeout = DefaultShmHandshakeTimeout
		} else if isLocal {
			timeout = DefaultLocalHandshakeTimeout
		} else {
			timeout = DefaultHandshakeTimeout
		}
	}

	// 3. Apply Defaults to Config for runtime responsiveness
	if config.Deadline == 0 {
		config.Deadline = time.Duration(timeout) * time.Millisecond
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 2 * time.Second // 2s default for high responsiveness
	}

	p, err := createProfile(profileName, address, st, timeout)
	if err != nil {
		return nil, err
	}

	if autoConnect {
		return CreateOpenSocket(p, config, socketType)
	}

	return CreateSocket(p, config, socketType)
}

func createProfile(profileName, address string, st interfaces.SocketType, timeout int) (interfaces.SocketProfile, error) {
	// 1. Support Compound Names (syntax: "profile:identity")
	// If no colon is present, identity defaults to an internal fallback in the switch.
	var identity string
	profileKey := profileName
	if parts := strings.Split(profileName, ":"); len(parts) > 1 {
		profileKey = parts[0]
		identity = parts[1]
	}

	switch profileKey {
	case "tcp-hello":
		// Default identities must be non-empty strings for identity-aware transports
		if identity == "" {
			if st == interfaces.SocketTypeClient {
				identity = "TcpClient-Generic"
			} else {
				identity = "TcpServer-Generic"
			}
		}

		if st == interfaces.SocketTypeClient {
			return profiles.NewTcpHelloClientProfile(identity, address, timeout), nil
		}
		return profiles.NewTcpHelloServerProfile(identity, address, timeout), nil

	case "tcp":
		if identity == "" {
			identity = "TcpRaw-Generic"
		}
		if st == interfaces.SocketTypeClient {
			return profiles.NewTcpClientProfile(identity, address, timeout), nil
		}
		return profiles.NewTcpServerProfile(identity, address, timeout), nil

	// UDP Support
	case "udp":
		if identity == "" {
			identity = "UdpRaw-Generic"
		}
		return profiles.NewUdpProfile(identity, address, timeout), nil
	case "udp-hello":
		if identity == "" {
			identity = "UdpHello-Generic"
		}
		return profiles.NewUdpHelloProfile(identity, address, timeout), nil

	// SHM Support
	case "shm":
		return profiles.NewShmProfile(address, timeout), nil // address is path
	case "shm-hello":
		return profiles.NewShmHelloProfile(address, timeout), nil
	default:
		return nil, fmt.Errorf("unknown profile: %s", profileKey)
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
