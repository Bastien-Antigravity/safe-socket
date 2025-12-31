package facade

import (
	"errors"
	"net"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
	"github.com/Bastien-Antigravity/safe-socket/src/schemas"
)

// EnvelopedConnection wraps a real TransportConnection.
// It uses HelloProtocol to:
// - Encapsulate outgoing data (Write -> Wrapped HelloMsg)
// - Decapsulate incoming data (Read Wrapped HelloMsg -> Payload)
type EnvelopedConnection struct {
	Conn    interfaces.TransportConnection
	Profile interfaces.SocketProfile
	Config  models.SocketConfig
	Proto   *protocols.HelloProtocol // Casted for access to Encapsulate/Decapsulate

	// LastIdentity holds the identity of the sender of the last packet read.
	// This allows UDP users to inspect who sent the data.
	LastIdentity *schemas.HelloMsg
}

// -----------------------------------------------------------------------------

func NewEnvelopedConnection(conn interfaces.TransportConnection, p interfaces.SocketProfile, c models.SocketConfig) *EnvelopedConnection {
	return &EnvelopedConnection{
		Conn:    conn,
		Profile: p,
		Config:  c,
		Proto:   protocols.NewHelloProtocol().(*protocols.HelloProtocol),
	}
}

// -----------------------------------------------------------------------------

func (e *EnvelopedConnection) Write(p []byte) (n int, err error) {
	// 1. Encapsulate Data into HelloMsg
	wrappedData, err := e.Proto.Encapsulate(p, e.Profile, e.Config)
	if err != nil {
		return 0, err
	}

	// 2. Write wrapped data to underlying transport
	// Note: We return len(p) to pretend we wrote the user's data, not the overhead size
	_, err = e.Conn.Write(wrappedData)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// -----------------------------------------------------------------------------

func (e *EnvelopedConnection) Read(p []byte) (n int, err error) {
	// 1. Read wrapped packet from underlying transport
	// We need a buffer big enough for the Envelope + Payload.
	// 4KB default should cover most UDP MTUs + Overhead.
	rawBuf := make([]byte, 8192)
	nRaw, err := e.Conn.Read(rawBuf)
	if err != nil {
		return 0, err
	}

	// 2. Decapsulate
	payload, identity, err := e.Proto.Decapsulate(rawBuf[:nRaw])
	if err != nil {
		return 0, err
	}

	// Store the identity for inspection
	e.LastIdentity = identity

	// 3. Copy Payload to user buffer
	if len(payload) > len(p) {
		return 0, errors.New("short buffer")
	}
	copy(p, payload)

	return len(payload), nil
}

// -----------------------------------------------------------------------------

func (e *EnvelopedConnection) Close() error {
	return e.Conn.Close()
}

// -----------------------------------------------------------------------------

// LocalAddr returns the local network address.
func (e *EnvelopedConnection) LocalAddr() net.Addr {
	return e.Conn.LocalAddr()
}

// -----------------------------------------------------------------------------

// RemoteAddr returns the remote network address.
func (e *EnvelopedConnection) RemoteAddr() net.Addr {
	return e.Conn.RemoteAddr()
}
