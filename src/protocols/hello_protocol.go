package protocols

import (
	"os"

	"google.golang.org/protobuf/proto"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/schemas"
)

// HelloProtocol implements the "SayHello" logic.
type HelloProtocol struct{}

// -----------------------------------------------------------------------------

func NewHelloProtocol() interfaces.Protocol {
	return &HelloProtocol{}
}

// -----------------------------------------------------------------------------

func (p *HelloProtocol) SayHello(conn interfaces.TransportConnection, profile interfaces.SocketProfile, config models.SocketConfig) error {
	hostname, _ := os.Hostname()

	// Default to config or localhost
	publicIP := config.PublicIP
	if publicIP == "" {
		publicIP = "127.0.0.1"
	}

	msg := &schemas.HelloMsg{
		Name:     profile.GetName(),
		Host:     hostname,
		Address:  profile.GetAddress(),
		PublicIP: publicIP,
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	// 1. Write Data
	_, err = conn.Write(data)
	return err
}

// -----------------------------------------------------------------------------

// WaitHello waits for a HelloMsg from the client and unmarshals it.
func (p *HelloProtocol) WaitHello(conn interfaces.TransportConnection) (*schemas.HelloMsg, error) {
	// 1. Prepare Buffer
	// Hello message is small, 1024 bytes should be plenty.
	// Note: Framed transport handles the actual size.
	buf := make([]byte, 1024)

	// 2. Read from connection
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	// 3. Unmarshal
	msg := &schemas.HelloMsg{}
	if err := proto.Unmarshal(buf[:n], msg); err != nil {
		return nil, err
	}

	return msg, nil
}
