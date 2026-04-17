package safesocket

import (
	"github.com/Bastien-Antigravity/safe-socket/src/facade"
	"github.com/Bastien-Antigravity/safe-socket/src/factory"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/schemas"
)

// Socket is an alias for the Facade to simplify usage.
type Socket = interfaces.Socket

// Identity is an alias for the HelloMsg schema to simplify usage.
type Identity = schemas.HelloMsg

// SocketType aliases removed to simplify API. Use "client" or "server" strings.

// -----------------------------------------------------------------------------

// Create creates a new safe-socket connection using a named profile.
//
// Parameters:
//   - profileName: "tcp", "tcp-hello", "udp", "udp-hello", "shm", "shm-hello"
//   - address: destination address ("IP:Port" or "FilePath" for SHM)
//   - publicIP: your public IP (Required for "hello" protocols)
//   - socketType: "client" or "server"
//   - autoConnect: if true, immediately calls Open() / Listen()
func Create(profileName, address, publicIP string, socketType string, autoConnect bool) (Socket, error) {
	// We delegate to the factory
	return factory.Create(profileName, address, publicIP, socketType, autoConnect)
}

// CreateWithConfig creates a new socket with full configuration control.
// Use this to set Deadlines or other advanced config options.
func CreateWithConfig(profileName, address string, config models.SocketConfig, socketType string, autoConnect bool) (Socket, error) {
	return factory.CreateWithConfig(profileName, address, config, socketType, autoConnect)
}

// -----------------------------------------------------------------------------

// GetIdentity extracts the client identity from a potentially wrapped connection.
// It traverses through Heartbeat, Handshake, or Envelope wrappers to find the HelloMsg.
func GetIdentity(conn interfaces.TransportConnection) *Identity {
	if conn == nil {
		return nil
	}

	// 1. Try HandshakeConnection (TCP/SHM Handshake)
	if hc, ok := conn.(*facade.HandshakeConnection); ok {
		return hc.Identity
	}

	// 2. Try EnvelopedConnection (UDP Stateless Identity)
	if ec, ok := conn.(*facade.EnvelopedConnection); ok {
		return ec.LastIdentity
	}

	// 3. Try peeling Heartbeat wrapper
	if hb, ok := conn.(*facade.HeartbeatConnection); ok {
		return GetIdentity(hb.TransportConnection)
	}

	return nil
}

// -----------------------------------------------------------------------------

// Expose other useful types if necessary
type (
	SocketConfig  = models.SocketConfig
	SocketProfile = interfaces.SocketProfile
	TransportType = interfaces.TransportType
	ProtocolType  = interfaces.ProtocolType
)

// -----------------------------------------------------------------------------

// Explicitly export Transport/Protocol constants via variables or just let users import interfaces?
// For a simple lib, letting them import "https://github.com/toto1234567890/safe-socket/src/interfaces" is okay,
// but aliasing commonly used ones is nicer.

const (
	TransportFramedTCP = interfaces.TransportFramedTCP
	TransportUDP       = interfaces.TransportUDP
	TransportSHM       = interfaces.TransportShm
)
