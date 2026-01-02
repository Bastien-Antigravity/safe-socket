package facade

import (
	"errors"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
	"github.com/Bastien-Antigravity/safe-socket/src/transports"
)

// SocketServer implements the interfaces.Socket interface for Server-side operations.
// It handles listening for connections (Listen) and accepting them (Accept).
// It also executes the handshake protocol (if configured) during Accept.
// Client-side methods (Open, Send, Receive) will return errors, as the Server itself
// does not send/receive data directly; the *accepted connection* does.
type SocketServer struct {
	Profile  interfaces.SocketProfile
	Config   models.SocketConfig
	listener interfaces.TransportListener
	Logger   *interfaces.Logger
}

// -----------------------------------------------------------------------------

// NewSocketServer creates a new instance of SocketServer.
func NewSocketServer(p interfaces.SocketProfile, config models.SocketConfig) *SocketServer {
	return &SocketServer{
		Profile: p,
		Config:  config,
	}
}

// -----------------------------------------------------------------------------

// Listen starts listening on the address specified by the profile.
func (s *SocketServer) Listen() error {
	if s.listener != nil {
		return errors.New("server already listening")
	}

	var ln interfaces.TransportListener
	var err error

	timeout := time.Duration(s.Profile.GetConnectTimeout()) * time.Millisecond

	switch s.Profile.GetTransport() {
	case interfaces.TransportFramedTCP:
		ln, err = transports.Listen(s.Profile.GetAddress(), timeout)
	case interfaces.TransportUDP:
		ln, err = transports.ListenUDP(s.Profile.GetAddress(), timeout)
	case interfaces.TransportShm:
		return errors.New("SHM server listener not yet implemented")
	default:
		return errors.New("unsupported transport type for listening")
	}

	if err != nil {
		return err
	}

	s.listener = ln
	return nil
}

// -----------------------------------------------------------------------------

// Accept accepts a new connection and performs the handshake if defined.
func (s *SocketServer) Accept() (interfaces.TransportConnection, error) {
	if s.listener == nil {
		return nil, errors.New("server not listening")
	}

	// 1. Accept raw transport connection
	conn, err := s.listener.Accept()
	if err != nil {
		return nil, err
	}

	// 1b. Apply Server Config Deadline (Factory Default)
	if s.Config.Deadline > 0 {
		deadline := time.Now().Add(s.Config.Deadline)
		if err := conn.SetDeadline(deadline); err != nil {
			conn.Close()
			return nil, err
		}
	}

	// 2. Encapsulation / Handshake Logic
	// Case A: UDP + Hello (Stateless Envelope)
	if s.Profile.GetTransport() == interfaces.TransportUDP &&
		s.Profile.GetProtocol() == interfaces.ProtocolHello {

		// Wrap connection to handle Per-Packet Decapsulation
		config := models.SocketConfig{} // Server doesn't usually use config for receiving, but wrapper needs it struct
		conn = NewEnvelopedConnection(conn, s.Profile, config)

	} else if s.Profile.GetProtocol() != "" && s.Profile.GetProtocol() != interfaces.ProtocolNone {
		// Case B: Connection-Oriented (TCP) + Hello
		// Perform Standard Handshake (Wait for Client to send Hello)
		var proto interfaces.Protocol
		// Currently only one protocol supported
		proto = protocols.NewHelloProtocol()

		// Note: The handshake itself will respect the Deadline set in 1b because it uses Read/Write on the conn.
		helloMsg, err := proto.WaitInitiation(conn)
		if err != nil {
			conn.Close()
			return nil, err
		}

		// Wrap with identity
		conn = NewHandshakeConnection(conn, helloMsg)
	}

	return conn, nil
}

// -----------------------------------------------------------------------------

// Close stops the server.
func (s *SocketServer) Close() error {
	if s.listener != nil {
		err := s.listener.Close()
		s.listener = nil
		return err
	}
	return nil
}

// -----------------------------------------------------------------------------

// Bind logger to safe-socket
func (s *SocketServer) SetLogger(logger *interfaces.Logger) {
	s.Logger = logger
}

// -----------------------------------------------------------------------------
// Client Methods (Not Supported for Server)
// -----------------------------------------------------------------------------

func (s *SocketServer) Open() error {
	return errors.New("method Open not supported for Server socket")
}

func (s *SocketServer) Send(data []byte) error {
	return errors.New("method Send not supported for Server socket")
}

func (s *SocketServer) Write(data []byte) (int, error) {
	return 0, errors.New("method Write not supported for Server socket")
}

func (s *SocketServer) Receive() ([]byte, error) {
	return nil, errors.New("method Receive not supported for Server socket")
}

func (s *SocketServer) Read(p []byte) (int, error) {
	return 0, errors.New("method Read not supported for Server socket")
}

func (s *SocketServer) SetDeadline(t time.Time) error {
	return errors.New("method SetDeadline not supported for Server listener (use Config.Deadline for accepted conns)")
}

func (s *SocketServer) SetReadDeadline(t time.Time) error {
	return errors.New("method SetReadDeadline not supported for Server listener")
}

func (s *SocketServer) SetWriteDeadline(t time.Time) error {
	return errors.New("method SetWriteDeadline not supported for Server listener")
}
