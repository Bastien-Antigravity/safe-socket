package factory

import (
	"errors"
	"time"

	"fmt"

	"github.com/Bastien-Antigravity/safe-socket/src/facade"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/profiles"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
	"github.com/Bastien-Antigravity/safe-socket/src/transports"
)

// -----------------------------------------------------------------------------

// Create is the simplified library entry point.
// profileName: "tcp-hello" (currently supported)
// address: destination address to connect to (e.g. "127.0.0.1:8081")
// publicIP: this node's public IP (for protocol handshake)
func Create(profileName, address, publicIP string) (*facade.SocketFacade, error) {
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

// CreateSocket constructs a SocketFacade based on the provided SocketProfile.
func CreateSocket(p interfaces.SocketProfile, config models.SocketConfig) (*facade.SocketFacade, error) {
	// 1. Create Transport & Connect
	var conn interfaces.TransportConnection
	var err error

	timeout := time.Duration(p.GetConnectTimeout()) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	switch p.GetTransport() {
	case interfaces.TransportFramedTCP:
		conn, err = transports.Connect(p.GetAddress(), timeout)
	case interfaces.TransportShm:
		// For SHM, we use Name as the identifier/path
		conn, err = transports.ConnectShm(p.GetName(), timeout)
	case interfaces.TransportUDP:
		conn, err = transports.ConnectUDP(p.GetAddress(), timeout)
	default:
		return nil, errors.New("unsupported transport type")
	}

	if err != nil {
		return nil, err
	}

	// 2. Perform Protocol (if specified)
	if p.GetProtocol() != "" && p.GetProtocol() != interfaces.ProtocolNone {
		var proto interfaces.Protocol
		// Currently only one protocol supported
		proto = protocols.NewHelloProtocol()

		if err := proto.Execute(conn, p, config); err != nil {
			conn.Close()
			return nil, err
		}
	}

	// 4. Return Facade
	return facade.NewSocketFacade(p, conn, config), nil
}
