package facade

import (
	"sync"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// HeartbeatConnection wraps a transport and periodically sends 0-length heartbeats.
type HeartbeatConnection struct {
	interfaces.TransportConnection
	stopHeartbeat chan struct{}
	closeOnce     sync.Once
}

func NewHeartbeatConnection(conn interfaces.TransportConnection) *HeartbeatConnection {
	h := &HeartbeatConnection{
		TransportConnection: conn,
		stopHeartbeat:       make(chan struct{}),
	}
	go h.start()
	return h
}

func (h *HeartbeatConnection) start() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_, _ = h.TransportConnection.Write([]byte{})
		case <-h.stopHeartbeat:
			return
		}
	}
}

func (h *HeartbeatConnection) Close() error {
	h.closeOnce.Do(func() {
		close(h.stopHeartbeat)
	})
	return h.TransportConnection.Close()
}
