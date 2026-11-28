package factory

// =============================================================================
// ESSENTIAL PROCESS:
// Instantiates and composes Socket instances from profile definitions and runtime configs,
// automatically selecting optimal transports (SHM vs FramedTCP) and protocol decorators.
//
// DATA FLOW:
// 1. Input: Profile names (tcp-hello, shm, udp-reliable), target addresses, and configs.
// 2. Logic: Resolves profile specifications, checks machine locality, configures timeouts,
//    and wraps transport layers with Handshake, Heartbeat, or Reliability facades.
// 3. Output: Initialized Socket interface ready for listening or client connection.
//
// KEY PARAMETERS:
// - Create: Factory constructor method accepting profile name, address, and config.
// =============================================================================

import (
	"fmt"
	"strings"

	"github.com/Bastien-Antigravity/safe-socket/src/facade"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/profiles"
	"github.com/Bastien-Antigravity/safe-socket/src/utils"
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

	p, resolvedCfg, err := prepareProfileAndConfig(profileName, address, config, st, socketType)
	if err != nil {
		return nil, err
	}

	if autoConnect {
		return CreateOpenSocket(p, resolvedCfg, socketType)
	}

	return CreateSocket(p, resolvedCfg, socketType)
}

// CreateClient creates a role-segregated ClientSocket using unified hybrid defaults.
func CreateClient(profileName, address, publicIP string, autoConnect bool) (interfaces.ClientSocket, error) {
	return CreateClientWithConfig(profileName, address, models.SocketConfig{PublicIP: publicIP}, autoConnect)
}

// CreateClientWithConfig creates a role-segregated ClientSocket with explicit configuration.
func CreateClientWithConfig(profileName, address string, config models.SocketConfig, autoConnect bool) (interfaces.ClientSocket, error) {
	p, resolvedCfg, err := prepareProfileAndConfig(profileName, address, config, interfaces.SocketTypeClient, "client")
	if err != nil {
		return nil, err
	}

	client := facade.NewSocketClient(p, resolvedCfg)
	if autoConnect {
		if err := client.Open(); err != nil {
			return nil, err
		}
	}
	return client, nil
}

// CreateServer creates a role-segregated ServerSocket using unified hybrid defaults.
func CreateServer(profileName, address, publicIP string, autoConnect bool) (interfaces.ServerSocket, error) {
	return CreateServerWithConfig(profileName, address, models.SocketConfig{PublicIP: publicIP}, autoConnect)
}

// CreateServerWithConfig creates a role-segregated ServerSocket with explicit configuration.
func CreateServerWithConfig(profileName, address string, config models.SocketConfig, autoConnect bool) (interfaces.ServerSocket, error) {
	p, resolvedCfg, err := prepareProfileAndConfig(profileName, address, config, interfaces.SocketTypeServer, "server")
	if err != nil {
		return nil, err
	}

	server := facade.NewSocketServer(p, resolvedCfg)
	if autoConnect {
		if err := server.Listen(); err != nil {
			return nil, err
		}
	}
	return server, nil
}

func prepareProfileAndConfig(profileName, address string, config models.SocketConfig, st interfaces.SocketType, socketType string) (interfaces.SocketProfile, models.SocketConfig, error) {
	// 1. Determine Identity and Profile Key
	profileKey := profileName
	if parts := strings.Split(profileName, ":"); len(parts) > 1 {
		profileKey = parts[0]
	}

	// 2. Determine Handshake / Connect Timeout
	timeout := int(config.HandshakeTimeout.Milliseconds())
	if timeout <= 0 {
		isLocal := utils.GetMachineDetector().IsLocalAddress(address)

		if strings.HasPrefix(profileKey, "shm") {
			timeout = DefaultShmHandshakeTimeout
		} else if isLocal {
			timeout = DefaultLocalHandshakeTimeout
		} else {
			timeout = DefaultHandshakeTimeout
		}
	}

	// 3. Apply Unified Hybrid Timeout Defaults (or honor explicit infinite timeout)
	if config.DisableTimeout || config.Deadline < 0 {
		config.DisableTimeout = true
		config.Deadline = 0
		if config.HeartbeatInterval == 0 {
			config.HeartbeatInterval = -1
		}
	} else if config.Deadline == 0 {
		defaultDeadline, defaultHB := models.ResolveHybridTimeouts(profileKey, socketType, models.IsDockerRuntime())
		config.Deadline = defaultDeadline
		if defaultDeadline == 0 {
			config.DisableTimeout = true
		}
		if config.HeartbeatInterval == 0 {
			config.HeartbeatInterval = defaultHB
		}
	}

	p, err := createProfile(profileName, address, st, timeout)
	if err != nil {
		return nil, config, err
	}
	return p, config, nil
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
	// Intelligent auto-encryption profile: selects plain TCP for local machine, TLS for remote machine
	case "auto-hello", "auto":
		detector := utils.GetMachineDetector()
		isLocal := detector.IsLocalAddress(address)

		if isLocal {
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
		} else {
			if identity == "" {
				if st == interfaces.SocketTypeClient {
					identity = "TlsClient-Generic"
				} else {
					identity = "TlsServer-Generic"
				}
			}
			if st == interfaces.SocketTypeClient {
				return profiles.NewTlsHelloClientProfile(identity, address, timeout), nil
			}
			return profiles.NewTlsHelloServerProfile(identity, address, timeout), nil
		}

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

	// TLS Support (Uses TCP transport with TLS config)
	case "tls-hello":
		if identity == "" {
			if st == interfaces.SocketTypeClient {
				identity = "TlsClient-Generic"
			} else {
				identity = "TlsServer-Generic"
			}
		}
		if st == interfaces.SocketTypeClient {
			return profiles.NewTlsHelloClientProfile(identity, address, timeout), nil
		}
		return profiles.NewTlsHelloServerProfile(identity, address, timeout), nil

	case "tls":
		if identity == "" {
			identity = "TlsRaw-Generic"
		}
		if st == interfaces.SocketTypeClient {
			return profiles.NewTlsClientProfile(identity, address, timeout), nil
		}
		return profiles.NewTlsServerProfile(identity, address, timeout), nil

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
		if identity == "" {
			identity = "ShmRaw-Generic"
		}
		return profiles.NewShmProfileWithIdentity(identity, address, timeout), nil
	case "shm-hello":
		if identity == "" {
			identity = "ShmHello-Generic"
		}
		return profiles.NewShmHelloProfileWithIdentity(identity, address, timeout), nil
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
