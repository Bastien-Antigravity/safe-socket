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

func (p *HelloProtocol) Execute(conn interfaces.TransportConnection, profile interfaces.SocketProfile, config models.SocketConfig) error {
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
