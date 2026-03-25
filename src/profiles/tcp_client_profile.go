package profiles

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------
// TCP Client Profile (Raw)
// -----------------------------------------------------------------------------

// TcpClientProfile implements the SocketProfile interface for a raw TCP connection
// without any handshake protocol.
type TcpClientProfile struct {
	Name    string
	Address string
	Timeout int
}

// NewTcpClientProfile creates a new instance of a TCP profile without a protocol.
func NewTcpClientProfile(name, address string, timeout int) *TcpClientProfile {
	return &TcpClientProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

func (p *TcpClientProfile) GetName() string {
	return p.Name
}

func (p *TcpClientProfile) GetAddress() string {
	return p.Address
}

func (p *TcpClientProfile) GetTransport() interfaces.TransportType {
	return interfaces.TransportFramedTCP
}

func (p *TcpClientProfile) GetProtocol() interfaces.ProtocolType {
	return interfaces.ProtocolNone
}

func (p *TcpClientProfile) GetConnectTimeout() int {
	return p.Timeout
}

// -----------------------------------------------------------------------------
// TCP Hello Client Profile
// -----------------------------------------------------------------------------

// TcpHelloClientProfile implements the SocketProfile interface for a TCP connection
// that performs a handshake protocol upon connection.
type TcpHelloClientProfile struct {
	Name           string
	Address        string
	ConnectTimeout int
	Protocol       interfaces.ProtocolType
}

// NewTcpHelloClientProfile creates a new instance of a TCP profile with the Hello protocol.
func NewTcpHelloClientProfile(name, address string, timeout int) *TcpHelloClientProfile {
	return &TcpHelloClientProfile{
		Name:           name,
		Address:        address,
		ConnectTimeout: timeout,
		Protocol:       interfaces.ProtocolHello,
	}
}

func (p *TcpHelloClientProfile) GetName() string {
	return p.Name
}

func (p *TcpHelloClientProfile) GetAddress() string {
	return p.Address
}

func (p *TcpHelloClientProfile) GetTransport() interfaces.TransportType {
	return interfaces.TransportFramedTCP
}

func (p *TcpHelloClientProfile) GetProtocol() interfaces.ProtocolType {
	return p.Protocol
}

func (p *TcpHelloClientProfile) GetConnectTimeout() int {
	return p.ConnectTimeout
}
