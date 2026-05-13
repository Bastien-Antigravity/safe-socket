package cgo_bridge

import (
	"errors"
	"time"

	"github.com/Bastien-Antigravity/safe-socket"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

func Create(profileName, address, publicIP, socketType string, autoConnect bool) (int32, error) {
	sock, err := safesocket.Create(profileName, address, publicIP, socketType, autoConnect)
	if err != nil {
		return -1, err
	}
	return Register(sock), nil
}

func CreateWithConfig(profileName, address, publicIP string, handshakeTimeoutMs, deadlineMs, heartbeatIntervalMs int, socketType string, autoConnect bool) (int32, error) {
	config := safesocket.SocketConfig{
		PublicIP:          publicIP,
		HandshakeTimeout:  time.Duration(handshakeTimeoutMs) * time.Millisecond,
		Deadline:          time.Duration(deadlineMs) * time.Millisecond,
		HeartbeatInterval: time.Duration(heartbeatIntervalMs) * time.Millisecond,
	}

	sock, err := safesocket.CreateWithConfig(profileName, address, config, socketType, autoConnect)
	if err != nil {
		return -1, err
	}
	return Register(sock), nil
}

func Open(handle int32) error {
	val, ok := Get(handle)
	if !ok {
		return errors.New("invalid handle")
	}

	sock, ok := val.(interfaces.Socket)
	if !ok {
		return errors.New("handle is not a socket")
	}

	return sock.Open()
}

func Close(handle int32) error {
	val, ok := Get(handle)
	if !ok {
		return errors.New("invalid handle")
	}

	var err error
	if sock, ok := val.(interfaces.Socket); ok {
		err = sock.Close()
	} else if conn, ok := val.(interfaces.TransportConnection); ok {
		err = conn.Close()
	} else {
		return errors.New("invalid handle type")
	}

	Unregister(handle)
	return err
}

func Send(handle int32, data []byte) (int32, error) {
	val, ok := Get(handle)
	if !ok {
		return -1, errors.New("invalid handle")
	}

	if sock, ok := val.(interfaces.Socket); ok {
		err := sock.Send(data)
		if err != nil {
			return -1, err
		}
		return int32(len(data)), nil
	}

	if conn, ok := val.(interfaces.TransportConnection); ok {
		n, err := conn.Write(data)
		if err != nil {
			return -1, err
		}
		return int32(n), nil
	}

	return -1, errors.New("invalid handle type for Send")
}

func Receive(handle int32, maxLength int) ([]byte, error) {
	val, ok := Get(handle)
	if !ok {
		return nil, errors.New("invalid handle")
	}

	if sock, ok := val.(interfaces.Socket); ok {
		return sock.Receive()
	}

	if conn, ok := val.(interfaces.TransportConnection); ok {
		tmp := make([]byte, maxLength)
		n, err := conn.Read(tmp)
		if err != nil {
			return nil, err
		}
		return tmp[:n], nil
	}

	return nil, errors.New("invalid handle type for Receive")
}

func Listen(handle int32) error {
	val, ok := Get(handle)
	if !ok {
		return errors.New("invalid handle")
	}

	sock, ok := val.(interfaces.Socket)
	if !ok {
		return errors.New("handle is not a socket")
	}

	return sock.Listen()
}

func Accept(handle int32) (int32, error) {
	val, ok := Get(handle)
	if !ok {
		return -1, errors.New("invalid handle")
	}

	sock, ok := val.(interfaces.Socket)
	if !ok {
		return -1, errors.New("handle is not a socket")
	}

	conn, err := sock.Accept()
	if err != nil {
		return -1, err
	}

	return Register(conn), nil
}

func SetIdleTimeout(handle int32, seconds float64) error {
	val, ok := Get(handle)
	if !ok {
		return errors.New("invalid handle")
	}

	timeout := time.Duration(seconds * float64(time.Second))

	if sock, ok := val.(interfaces.Socket); ok {
		return sock.SetIdleTimeout(timeout)
	}

	if conn, ok := val.(interfaces.TransportConnection); ok {
		return conn.SetIdleTimeout(timeout)
	}

	return errors.New("invalid handle type for SetIdleTimeout")
}

func SetDeadline(handle int32, seconds float64) error {
	val, ok := Get(handle)
	if !ok {
		return errors.New("invalid handle")
	}

	deadline := time.Now().Add(time.Duration(seconds * float64(time.Second)))

	if sock, ok := val.(interfaces.Socket); ok {
		return sock.SetDeadline(deadline)
	}

	if conn, ok := val.(interfaces.TransportConnection); ok {
		return conn.SetDeadline(deadline)
	}

	return errors.New("invalid handle type for SetDeadline")
}
