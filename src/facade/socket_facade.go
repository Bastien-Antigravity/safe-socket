package facade

import (
	"errors"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
	"github.com/Bastien-Antigravity/safe-socket/src/transports"
)

type SocketFacade struct {
	Profile   interfaces.SocketProfile
	Config    models.SocketConfig
	transport interfaces.TransportConnection
}

// -----------------------------------------------------------------------------

func NewSocketFacade(p interfaces.SocketProfile, c models.SocketConfig) *SocketFacade {
	return &SocketFacade{
		Profile: p,
		Config:  c,
	}
}

// -----------------------------------------------------------------------------

// Open establishes the connection using the configured transport and protocol.
func (c *SocketFacade) Open() error {
	if c.transport != nil {
		return errors.New("socket already open")
	}

	// 1. Create Transport & Connect
	var conn interfaces.TransportConnection
	var err error

	timeout := time.Duration(c.Profile.GetConnectTimeout()) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	switch c.Profile.GetTransport() {
	case interfaces.TransportFramedTCP:
		conn, err = transports.Connect(c.Profile.GetAddress(), timeout)
	case interfaces.TransportShm:
		// For SHM, we use Name as the identifier/path
		conn, err = transports.ConnectShm(c.Profile.GetName(), timeout)
	case interfaces.TransportUDP:
		conn, err = transports.ConnectUDP(c.Profile.GetAddress(), timeout)
	default:
		return errors.New("unsupported transport type")
	}

	if err != nil {
		return err
	}

	// 2. Perform Protocol (if specified)
	if c.Profile.GetProtocol() != "" && c.Profile.GetProtocol() != interfaces.ProtocolNone {
		var proto interfaces.Protocol
		// Currently only one protocol supported
		proto = protocols.NewHelloProtocol()

		if err := proto.Execute(conn, c.Profile, c.Config); err != nil {
			conn.Close()
			return err
		}
	}

	c.transport = conn
	return nil
}

// -----------------------------------------------------------------------------

// Send writes the raw data to the transport.
func (c *SocketFacade) Send(data []byte) error {
	if c.transport == nil {
		return errors.New("socket not open")
	}
	_, err := c.transport.Write(data)
	return err
}

// -----------------------------------------------------------------------------

// Receive reads from the transport into the provided buffer.
// It returns the number of bytes read and any error encountered.
func (c *SocketFacade) Receive(buf []byte) (int, error) {
	if c.transport == nil {
		return 0, errors.New("socket not open")
	}
	return c.transport.Read(buf)
}

// -----------------------------------------------------------------------------

func (c *SocketFacade) Close() error {
	if c.transport != nil {
		err := c.transport.Close()
		c.transport = nil
		return err
	}
	return nil
}
