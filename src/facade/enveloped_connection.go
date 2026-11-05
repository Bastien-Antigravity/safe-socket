package facade

// =============================================================================
// ESSENTIAL PROCESS:
// Wraps UDP transport connections to enforce stateless packet encapsulation and
// decapsulation via Cap'n Proto PacketEnvelope frames without persistent sessions.
//
// DATA FLOW:
// 1. Input: Outgoing application payload buffers or incoming UDP datagrams.
// 2. Logic: Encapsulates outgoing data with SenderID; decapsulates incoming
//    packets to extract payload and reconstruct sender HelloMsg identity.
// 3. Output: Raw payload bytes and dynamically updated LastIdentity.
//
// KEY PARAMETERS:
// - LastIdentity: Dynamically updated identity of the sender of the last packet read.
// =============================================================================

import (
	"errors"
	"net"
	"sync"
	"time"

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
	mu           sync.RWMutex
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
	// Max UDP datagram size is 65535 bytes.
	rawBuf := make([]byte, 65535)
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
	e.mu.Lock()
	e.LastIdentity = identity
	e.mu.Unlock()

	// 3. Copy Payload to user buffer
	if len(payload) > len(p) {
		return 0, errors.New("short buffer")
	}
	copy(p, payload)

	return len(payload), nil
}

// -----------------------------------------------------------------------------

// ReadMessage implementation for Enveloped Connection
func (e *EnvelopedConnection) ReadMessage() ([]byte, error) {
	// 1. Read Raw Packet (Envelope)
	rawBuf, err := e.Conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	// 2. Decapsulate
	payload, identity, err := e.Proto.Decapsulate(rawBuf)
	if err != nil {
		return nil, err
	}

	// Store the identity for inspection
	e.mu.Lock()
	e.LastIdentity = identity
	e.mu.Unlock()

	return payload, nil
}

// -----------------------------------------------------------------------------

// GetLastIdentity returns the sender identity of the last decapsulated packet in a thread-safe manner.
func (e *EnvelopedConnection) GetLastIdentity() *schemas.HelloMsg {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.LastIdentity
}

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

// -----------------------------------------------------------------------------

func (e *EnvelopedConnection) SetDeadline(t time.Time) error {
	return e.Conn.SetDeadline(t)
}

// -----------------------------------------------------------------------------

func (e *EnvelopedConnection) SetReadDeadline(t time.Time) error {
	return e.Conn.SetReadDeadline(t)
}

// -----------------------------------------------------------------------------

func (e *EnvelopedConnection) SetWriteDeadline(t time.Time) error {
	return e.Conn.SetWriteDeadline(t)
}

func (e *EnvelopedConnection) SetIdleTimeout(d time.Duration) error {
	return e.Conn.SetIdleTimeout(d)
}

// -----------------------------------------------------------------------------
