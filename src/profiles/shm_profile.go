package profiles

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------
// SHM Profiles
// -----------------------------------------------------------------------------

type ShmProfile struct {
	Name           string // Treated as File Path
	ConnectTimeout int
	Protocol       interfaces.ProtocolType
}

func (p *ShmProfile) GetName() string {
	return p.Name
}

// -----------------------------------------------------------------------------

func (p *ShmProfile) GetAddress() string {
	return p.Name
}

// -----------------------------------------------------------------------------

func (p *ShmProfile) GetTransport() interfaces.TransportType {
	return interfaces.TransportShm
}

// -----------------------------------------------------------------------------

func (p *ShmProfile) GetConnectTimeout() int {
	return p.ConnectTimeout
}

// -----------------------------------------------------------------------------

func (p *ShmProfile) GetProtocol() interfaces.ProtocolType {
	return p.Protocol
}

// -----------------------------------------------------------------------------

func NewShmProfile(path string, timeout int) *ShmProfile {
	return &ShmProfile{
		Name:           path,
		ConnectTimeout: timeout,
		Protocol:       interfaces.ProtocolNone,
	}
}

func NewShmHelloProfile(path string, timeout int) *ShmProfile {
	return &ShmProfile{
		Name:           path,
		ConnectTimeout: timeout,
		Protocol:       interfaces.ProtocolHello,
	}
}
