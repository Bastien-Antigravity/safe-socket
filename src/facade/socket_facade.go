package facade

import (
	"google.golang.org/protobuf/proto"

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

// Send marshals the payload using Protobuf and writes it to the transport.
func (c *SocketFacade) Send(payload proto.Message) error {
	data, err := proto.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = c.transport.Write(data)
	return err
}

// -----------------------------------------------------------------------------

// Receive reads one message from the transport and deserializes it into v (which must be a proto.Message).
func (c *SocketFacade) Receive(v proto.Message) error {
	// We need a buffer to read the data.
	// For simplicity, we allocate a buffer. In a high-perf scenario, we'd reuse or stream.
	// Assuming max message size 1MB for safety.
	buf := make([]byte, 1024*1024)

	n, err := c.transport.Read(buf)
	if err != nil {
		return err
	}

	// Deserialize just the read portion
	return proto.Unmarshal(buf[:n], v)
}

// -----------------------------------------------------------------------------

func (c *SocketFacade) Close() error {
	if c.transport != nil {
		return c.transport.Close()
	}
	return nil
}
