package facade

import (
	"errors"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
	"github.com/Bastien-Antigravity/safe-socket/src/transports"
)

// SocketClient implements the interfaces.Socket interface for Client-side operations.
// It handles establishing connections (Open), sending/receiving data, and protocol handshakes.
// Server-side methods (Listen, Accept) will return errors.
type SocketClient struct {
	Profile   interfaces.SocketProfile
	Config    models.SocketConfig
	transport interfaces.TransportConnection
	Logger    *interfaces.Logger
}

// -----------------------------------------------------------------------------

func NewSocketClient(p interfaces.SocketProfile, c models.SocketConfig) *SocketClient {
	return &SocketClient{
		Profile: p,
		Config:  c,
	}
}

// -----------------------------------------------------------------------------

// Open establishes the connection using the configured transport and protocol.
func (c *SocketClient) Open() error {
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

	// 2. Encapsulation / Handshake Logic
	// Case A: UDP + Hello (Stateless Envelope)
	if c.Profile.GetTransport() == interfaces.TransportUDP &&
		c.Profile.GetProtocol() == interfaces.ProtocolHello {

		// Wrap connection to handle Per-Packet Encapsulation
		conn = NewEnvelopedConnection(conn, c.Profile, c.Config)

		// No initial handshake packet sent here.
		// The first Send() will carry the identity.

	} else if c.Profile.GetProtocol() != "" && c.Profile.GetProtocol() != interfaces.ProtocolNone {
		// Case B: Connection-Oriented (TCP/SHM) + Hello
		// Perform Standard Handshake
		var proto interfaces.Protocol
		// Currently only one protocol supported
		proto = protocols.NewHelloProtocol()

		if err := proto.Initiate(conn, c.Profile, c.Config); err != nil {
			conn.Close()
			return err
		}
	}

	c.transport = conn
	return nil
}

// -----------------------------------------------------------------------------

// Send writes the raw data to the transport.
func (c *SocketClient) Send(data []byte) error {
	if c.transport == nil {
		return errors.New("socket not open")
	}
	_, err := c.transport.Write(data)
	return err
}

// Write implements the io.Writer interface in logger.
func (c *SocketClient) Write(data []byte) (int, error) {
	if c.transport == nil {
		return 0, errors.New("socket not open")
	}
	n, err := c.transport.Write(data)
	return n, err
}

// -----------------------------------------------------------------------------

// Receive reads from the transport into a newly allocated buffer.
// It returns the data read and any error encountered.
func (c *SocketClient) Receive() ([]byte, error) {
	if c.transport == nil {
		return nil, errors.New("socket not open")
	}
	return c.transport.ReadMessage()
}

// Read reads from the transport into the provided buffer (io.Reader compliance).
func (c *SocketClient) Read(p []byte) (int, error) {
	if c.transport == nil {
		return 0, errors.New("socket not open")
	}
	return c.transport.Read(p)
}

// -----------------------------------------------------------------------------

func (c *SocketClient) Close() error {
	if c.transport != nil {
		err := c.transport.Close()
		c.transport = nil
		return err
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetDeadline sets the read and write deadlines associated with the connection.
func (c *SocketClient) SetDeadline(t time.Time) error {
	if c.transport == nil {
		return errors.New("socket not open")
	}
	return c.transport.SetDeadline(t)
}

// -----------------------------------------------------------------------------

// SetReadDeadline sets the deadline for future Read calls.
func (c *SocketClient) SetReadDeadline(t time.Time) error {
	if c.transport == nil {
		return errors.New("socket not open")
	}
	return c.transport.SetReadDeadline(t)
}

// -----------------------------------------------------------------------------

// SetWriteDeadline sets the deadline for future Write calls.
func (c *SocketClient) SetWriteDeadline(t time.Time) error {
	if c.transport == nil {
		return errors.New("socket not open")
	}
	return c.transport.SetWriteDeadline(t)
}

// -----------------------------------------------------------------------------

// Bind logger to safe-socket
func (c *SocketClient) SetLogger(logger *interfaces.Logger) {
	c.Logger = logger
}

// -----------------------------------------------------------------------------
// Server Methods (Not Supported for Client)
// -----------------------------------------------------------------------------

func (c *SocketClient) Listen() error {
	return errors.New("method Listen not supported for Client socket")
}

func (c *SocketClient) Accept() (interfaces.TransportConnection, error) {
	return nil, errors.New("method Accept not supported for Client socket")
}
