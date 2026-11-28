package profiles

// =============================================================================
// ESSENTIAL PROCESS:
// Defines TCP server socket profiles for inbound listener sockets with or without
// application-level handshake and peer identity verification.
//
// DATA FLOW:
// 1. Input: Local bind address host:port, idle timeouts, and protocol configuration.
// 2. Logic: Embeds BaseProfile to declare TCP listening transport attributes.
// 3. Output: TcpServerProfile and TcpHelloServerProfile configurations.
//
// KEY PARAMETERS:
// - TcpServerProfile: Profile for raw framed TCP listener.
// - TcpHelloServerProfile: Profile enforcing Cap'n Proto HelloMsg verification on accept.
// =============================================================================

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------
// TCP Server Profile (Raw)
// -----------------------------------------------------------------------------

// TcpServerProfile implements the SocketProfile interface for a raw TCP server
// without any handshake protocol.
type TcpServerProfile struct {
	BaseProfile
}

// NewTcpServerProfile creates a new instance of a TCP server profile without a protocol.
func NewTcpServerProfile(name, address string, timeout int) *TcpServerProfile {
	return &TcpServerProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportFramedTCP,
			Protocol:       interfaces.ProtocolNone,
			ConnectTimeout: timeout,
		},
	}
}

// -----------------------------------------------------------------------------
// TCP Hello Server Profile
// -----------------------------------------------------------------------------

// TcpHelloServerProfile implements the SocketProfile interface for a TCP server
// that performs a handshake protocol upon connection.
type TcpHelloServerProfile struct {
	BaseProfile
}

// NewTcpHelloServerProfile creates a new instance of a TCP server profile with the Hello protocol.
func NewTcpHelloServerProfile(name, address string, timeout int) *TcpHelloServerProfile {
	return &TcpHelloServerProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportFramedTCP,
			Protocol:       interfaces.ProtocolHello,
			ConnectTimeout: timeout,
		},
	}
}
