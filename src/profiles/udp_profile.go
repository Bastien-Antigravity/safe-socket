package profiles

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------
// UDP Profiles
// -----------------------------------------------------------------------------

type UdpProfile struct {
	Name           string
	Address        string
	ConnectTimeout int
	Protocol       interfaces.ProtocolType
}

func (p *UdpProfile) GetName() string                        { return p.Name }
func (p *UdpProfile) GetAddress() string                     { return p.Address }
func (p *UdpProfile) GetTransport() interfaces.TransportType { return interfaces.TransportUDP }
func (p *UdpProfile) GetConnectTimeout() int                 { return p.ConnectTimeout }
func (p *UdpProfile) GetProtocol() interfaces.ProtocolType   { return p.Protocol }

// -----------------------------------------------------------------------------

func NewUdpProfile(name, address string, timeout int) *UdpProfile {
	return &UdpProfile{
		Name:           name,
		Address:        address,
		ConnectTimeout: timeout,
		Protocol:       interfaces.ProtocolNone,
	}
}

func NewUdpHelloProfile(name, address string, timeout int) *UdpProfile {
	return &UdpProfile{
		Name:           name,
		Address:        address,
		ConnectTimeout: timeout,
		Protocol:       interfaces.ProtocolHello,
	}
}
