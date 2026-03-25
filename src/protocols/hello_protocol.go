package protocols

import (
	"errors"
	"os"

	"capnproto.org/go/capnp/v3"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/schemas"
)

// HelloProtocol implements the "Initiate" logic.
type HelloProtocol struct{}

// -----------------------------------------------------------------------------

func NewHelloProtocol() interfaces.Protocol {
	return &HelloProtocol{}
}

// -----------------------------------------------------------------------------

// Initiate executes the handshake sequence or protocol logic (Client).
func (p *HelloProtocol) Initiate(conn interfaces.TransportConnection, profile interfaces.SocketProfile, config models.SocketConfig) error {
	hostname, _ := os.Hostname()

	// PublicIP is required for the Hello Protocol
	publicIP := config.PublicIP
	if publicIP == "" {
		return errors.New("PublicIP is required in SocketConfig for HelloProtocol")
	}

	// Cap'n Proto Message Construction
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return err
	}

	helloMsg, err := schemas.NewRootHelloMsg(seg)
	if err != nil {
		return err
	}

	// Dynamic Address Resolution from Transport
	localAddr := conn.LocalAddr().String()
	remoteAddr := conn.RemoteAddr().String()

	_ = helloMsg.SetFromName(profile.GetName())
	_ = helloMsg.SetFromHost(hostname)
	_ = helloMsg.SetFromAddress(localAddr) // Actual bound address
	_ = helloMsg.SetToAddress(remoteAddr)  // Target address
	_ = helloMsg.SetFromPublicIP(publicIP)

	// Marshal to bytes
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	// 1. Write Data
	_, err = conn.Write(data)
	return err
}

// -----------------------------------------------------------------------------

// WaitInitiation waits for a HelloMsg from the client and unmarshals it.
func (p *HelloProtocol) WaitInitiation(conn interfaces.TransportConnection) (*schemas.HelloMsg, error) {
	// 1. Prepare Buffer
	// Use a larger buffer (4KB) to avoid io.ErrShortBuffer from framed transport
	// if the message contains long strings (host, address, etc).
	buf := make([]byte, 4096)

	// 2. Read from connection
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	// 3. Unmarshal
	// Cap'n Proto requires the full buffer or a stream.
	// With FramedTCP, we have the full message in buf[:n].
	// However, we need to handle proper unmarshalling from bytes.

	msg, err := capnp.Unmarshal(buf[:n])
	if err != nil {
		return nil, err
	}

	// Extract the root struct
	helloMsg, err := schemas.ReadRootHelloMsg(msg)
	if err != nil {
		return nil, err
	}

	return &helloMsg, nil
}

// -----------------------------------------------------------------------------

// Encapsulate wraps the user payload into a PacketEnvelope.
func (p *HelloProtocol) Encapsulate(data []byte, profile interfaces.SocketProfile, config models.SocketConfig) ([]byte, error) {
	// We use the lightweight PacketEnvelope for connectionless transport
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	envelope, err := schemas.NewRootPacketEnvelope(seg)
	if err != nil {
		return nil, err
	}

	// We only send the Name (ID) to save bandwidth
	_ = envelope.SetSenderID(profile.GetName())
	_ = envelope.SetPayload(data)

	return msg.Marshal()
}

// -----------------------------------------------------------------------------

// Decapsulate unwraps a PacketEnvelope and returns the payload.
// It also reconstructs a partial HelloMsg for identity verification.
func (p *HelloProtocol) Decapsulate(packet []byte) ([]byte, *schemas.HelloMsg, error) {
	msg, err := capnp.Unmarshal(packet)
	if err != nil {
		return nil, nil, err
	}

	envelope, err := schemas.ReadRootPacketEnvelope(msg)
	if err != nil {
		return nil, nil, err
	}

	payload, err := envelope.Payload()
	if err != nil {
		return nil, nil, err
	}

	senderID, err := envelope.SenderID()
	if err != nil {
		return nil, nil, err
	}

	// Reconstruct a temporary HelloMsg for the interface contract
	// We only have the Name.
	// Note: We need a new message segment to create this struct if we want to return it as a Verify-able object
	// or we just return a nil HelloMsg with the name populated?
	// The interface expects *schemas.HelloMsg.

	// Let's create a minimal HelloMsg to satisfy the return type
	_, metaSeg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	helloMsg, _ := schemas.NewRootHelloMsg(metaSeg)
	_ = helloMsg.SetFromName(senderID)
	// Other fields are empty/default

	// Ensure the underlying message stays alive if needed, but here we return the struct.
	// Note: The struct is bound to 'metaMsg'. As long as helloMsg is used, it should be fine.
	// However, we are returning a pointer to a struct that lives in a local segment...
	// Wait, Cap'n Proto Go structs are value types that reference the segment.
	// NewRootHelloMsg returns `HelloMsg` (struct), not pointer.
	// But our signature returns `*schemas.HelloMsg`.
	// We will take the address of the struct.

	return payload, &helloMsg, nil
}
