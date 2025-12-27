package facade

import (
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
)

type SocketFacade struct {
	Profile   interfaces.SocketProfile
	Config    models.SocketConfig
	transport interfaces.TransportConnection
}

// -----------------------------------------------------------------------------

func NewSocketFacade(p interfaces.SocketProfile, t interfaces.TransportConnection, c models.SocketConfig) *SocketFacade {
	return &SocketFacade{
		Profile:   p,
		Config:    c,
		transport: t,
	}
}

// -----------------------------------------------------------------------------

// Send writes the raw data to the transport.
func (c *SocketFacade) Send(data []byte) error {
	_, err := c.transport.Write(data)
	return err
}

// -----------------------------------------------------------------------------

// Receive reads from the transport into the provided buffer.
// It returns the number of bytes read and any error encountered.
func (c *SocketFacade) Receive(buf []byte) (int, error) {
	return c.transport.Read(buf)
}

// -----------------------------------------------------------------------------

func (c *SocketFacade) Close() error {
	if c.transport != nil {
		return c.transport.Close()
	}
	return nil
}
