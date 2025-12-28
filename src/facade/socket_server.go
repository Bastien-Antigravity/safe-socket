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
	listener interfaces.TransportListener
}

// NewSocketServer creates a new instance of SocketServer.
func NewSocketServer(p interfaces.SocketProfile) *SocketServer {
	return &SocketServer{
		Profile: p,
	}
}

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

	// 2. Encapsulation / Handshake Logic
	// Case A: UDP + Hello (Stateless Envelope)
	if s.Profile.GetTransport() == interfaces.TransportUDP &&
		s.Profile.GetProtocol() == interfaces.ProtocolHello {

		// Wrap connection to handle Per-Packet Decapsulation
		// Note: 'conn' here is a TransientUdpSocket containing the first packet.
		// The first Read() on EnvelopedConnection will read that packet from TransientSocket,
		// Decapsulate it, and return the Payload.
		config := models.SocketConfig{} // Server doesn't usually use config for receiving, but wrapper needs it struct
		conn = NewEnvelopedConnection(conn, s.Profile, config)

	} else if s.Profile.GetProtocol() != "" && s.Profile.GetProtocol() != interfaces.ProtocolNone {
		// Case B: Connection-Oriented (TCP) + Hello
		// Perform Standard Handshake (Wait for Client to send Hello)
		var proto interfaces.Protocol
		// Currently only one protocol supported
		proto = protocols.NewHelloProtocol()

		if _, err := proto.WaitInitiation(conn); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return conn, nil
}

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
// Client Methods (Not Supported for Server)
// -----------------------------------------------------------------------------

func (s *SocketServer) Open() error {
	return errors.New("method Open not supported for Server socket")
}

func (s *SocketServer) Send(data []byte) error {
	return errors.New("method Send not supported for Server socket")
}

func (s *SocketServer) Receive(buf []byte) (int, error) {
	return 0, errors.New("method Receive not supported for Server socket")
}
