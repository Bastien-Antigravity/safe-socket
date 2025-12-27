package profiles

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// TcpProfile is a pre-configured profile for TCP transport with NO protocol.
type TcpProfile struct {
	Name    string
	Address string
	Timeout int
}

func NewTcpProfile(name, address string, timeout int) *TcpProfile {
	return &TcpProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

func (p *TcpProfile) GetName() string {
	return p.Name
}

func (p *TcpProfile) GetAddress() string {
	return p.Address
}

func (p *TcpProfile) GetTransport() interfaces.TransportType {
	return interfaces.TransportFramedTCP
}

func (p *TcpProfile) GetProtocol() interfaces.ProtocolType {
	return interfaces.ProtocolNone
}

func (p *TcpProfile) GetConnectTimeout() int {
	return p.Timeout
}
