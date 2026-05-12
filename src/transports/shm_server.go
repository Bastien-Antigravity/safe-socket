package transports

import (
	"errors"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/edsrzf/mmap-go"
)

// ShmListener implements interfaces.TransportListener for Shared Memory.
type ShmListener struct {
	path        string
	timeout     time.Duration
	transport   *ShmTransport
	acceptCount int32
}

// -----------------------------------------------------------------------------

// ListenShm creates (or opens) the SHM file and prepares it for a client connection.
func ListenShm(path string, timeout time.Duration) (interfaces.TransportListener, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	// Ensure file is the correct size
	if err := file.Truncate(int64(TotalSize)); err != nil {
		_ = file.Close()
		return nil, err
	}

	m, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	t := NewShmTransport(file, m, "server", timeout)

	// Reset metadata for a clean start
	atomic.StoreUint64(t.ProduceHead, 0)
	atomic.StoreUint64(t.ProduceTail, 0)
	atomic.StoreUint64(t.ConsumeHead, 0)
	atomic.StoreUint64(t.ConsumeTail, 0)
	atomic.StoreUint64(t.ServerStatus, StatusListening)
	atomic.StoreUint64(t.ClientStatus, StatusIdle)
	now := uint64(time.Now().UnixNano())
	atomic.StoreUint64(t.MyActivity, now)
	atomic.StoreUint64(t.PeerActivity, now)

	return &ShmListener{
		path:      path,
		timeout:   timeout,
		transport: t,
	}, nil
}

// -----------------------------------------------------------------------------

// Accept waits for a client to attach to the SHM file.
// Note: Current implementation is 1-to-1 (Point-to-Point).
func (l *ShmListener) Accept() (interfaces.TransportConnection, error) {
	if !atomic.CompareAndSwapInt32(&l.acceptCount, 0, 1) {
		// In a real multi-tenant scenario, we would wait for a new file request.
		// For now, we only support one client per SHM path.
		return nil, errors.New("SHM listener only supports one concurrent connection")
	}

	// Wait for client to set ClientStatus to Connected
	// Using a relatively slow poll here as Accept is not in the hot path.
	for {
		if atomic.LoadUint64(l.transport.ClientStatus) == StatusConnected {
			break
		}
		if l.transport.closed.Load() {
			return nil, errors.New("listener closed")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Acknowledge connection
	atomic.StoreUint64(l.transport.ServerStatus, StatusConnected)

	return l.transport, nil
}

// -----------------------------------------------------------------------------

func (l *ShmListener) Close() error {
	return l.transport.Close()
}

// -----------------------------------------------------------------------------

func (l *ShmListener) Addr() net.Addr {
	return l.transport.LocalAddr()
}
