package profiles

// =============================================================================
// ESSENTIAL PROCESS:
// Defines TCP server socket profiles for inbound listener sockets with or without
// application-level handshake and peer identity verification.
//
// DATA FLOW:
// 1. Input: Local bind address host:port, idle timeouts, and protocol configuration.
// 2. Logic: Implements SocketProfile to declare TCP listening transport attributes.
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
	Name    string
	Address string
	Timeout int
}

// NewTcpServerProfile creates a new instance of a TCP server profile without a protocol.
func NewTcpServerProfile(name, address string, timeout int) *TcpServerProfile {
	return &TcpServerProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

func (p *TcpServerProfile) GetName() string {
	return p.Name
}

func (p *TcpServerProfile) GetAddress() string {
	return p.Address
}

func (p *TcpServerProfile) GetTransport() interfaces.TransportType {
	return interfaces.TransportFramedTCP
}

func (p *TcpServerProfile) GetProtocol() interfaces.ProtocolType {
	return interfaces.ProtocolNone
}

func (p *TcpServerProfile) GetConnectTimeout() int {
	return p.Timeout
}

// -----------------------------------------------------------------------------
// TCP Hello Server Profile
// -----------------------------------------------------------------------------

// TcpHelloServerProfile implements the SocketProfile interface for a TCP server
// that performs a handshake protocol upon connection.
type TcpHelloServerProfile struct {
	Name           string
	Address        string
	ConnectTimeout int
	Protocol       interfaces.ProtocolType
}

// NewTcpHelloServerProfile creates a new instance of a TCP server profile with the Hello protocol.
func NewTcpHelloServerProfile(name, address string, timeout int) *TcpHelloServerProfile {
	return &TcpHelloServerProfile{
		Name:           name,
		Address:        address,
		ConnectTimeout: timeout,
		Protocol:       interfaces.ProtocolHello,
	}
}

func (p *TcpHelloServerProfile) GetName() string {
	return p.Name
}

func (p *TcpHelloServerProfile) GetAddress() string {
	return p.Address
}

func (p *TcpHelloServerProfile) GetTransport() interfaces.TransportType {
	return interfaces.TransportFramedTCP
}

func (p *TcpHelloServerProfile) GetProtocol() interfaces.ProtocolType {
	return p.Protocol
}

func (p *TcpHelloServerProfile) GetConnectTimeout() int {
	return p.ConnectTimeout
}
