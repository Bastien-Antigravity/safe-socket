package transports

// =============================================================================
// ESSENTIAL PROCESS:
// Implements UDP datagram transport connection handling for safe-socket,
// supporting both connected client sockets and demultiplexed server session sockets.
//
// DATA FLOW:
// 1. Input: Datagram bytes or demultiplexed packets from UdpListener.
// 2. Logic: Performs ReadFromUDP / WriteToUDP or channel-backed session reads
//    with independent per-session idle deadlines.
// 3. Output: TransportConnection interface for UDP datagram streams.
//
// KEY PARAMETERS:
// - UdpSocket: Datagram connection struct wrapping *net.UDPConn or a server session channel.
// =============================================================================

import (
	"errors"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const udpMaxDatagramSize = 65535

// UdpSocket implements interfaces.TransportConnection over UDP.
// Note: UDP is unreliable and unordered unless wrapped with ReliableConnection.
type UdpSocket struct {
	Conn        *net.UDPConn
	idleTimeout time.Duration

	// Server-Side: Demultiplexed session fields
	TransientRemoteAddr *net.UDPAddr // If set, Write() uses WriteToUDP
	RecvBuf             []byte       // Legacy one-shot pre-read buffer
	recvChan            chan []byte  // Demultiplexed packet queue for server sessions
	sessionClosed       chan struct{}
	closeOnce           sync.Once
	onClose             func()
	readDeadlineNs      atomic.Int64
	writeDeadlineNs     atomic.Int64
	mu                  sync.Mutex
}

// -----------------------------------------------------------------------------

func NewUdpSocket(conn *net.UDPConn, timeout time.Duration) *UdpSocket {
	s := &UdpSocket{
		Conn:        conn,
		idleTimeout: timeout,
	}
	if conn != nil {
		s.refreshReadDeadline()
		s.refreshWriteDeadline()
	}
	return s
}

// NewTransientUdpSocket creates a session socket representing a remote UDP peer,
// fed by the listener's demultiplexer without mutating the listener's own read deadline.
func NewTransientUdpSocket(conn *net.UDPConn, addr *net.UDPAddr, data []byte, timeout time.Duration, onClose ...func()) *UdpSocket {
	s := &UdpSocket{
		Conn:                conn,
		TransientRemoteAddr: addr,
		idleTimeout:         timeout,
		recvChan:            make(chan []byte, 64),
		sessionClosed:       make(chan struct{}),
	}
	if len(onClose) > 0 {
		s.onClose = onClose[0]
	}
	if len(data) > 0 {
		s.recvChan <- data
	} else if data != nil {
		s.recvChan <- data
	}
	if timeout > 0 {
		s.readDeadlineNs.Store(time.Now().Add(timeout).UnixNano())
		s.writeDeadlineNs.Store(time.Now().Add(timeout).UnixNano())
	}
	return s
}

func (s *UdpSocket) enqueuePacket(pkt []byte) {
	if s.recvChan == nil {
		return
	}
	select {
	case <-s.sessionClosed:
		return
	default:
	}
	s.refreshReadDeadline()
	select {
	case s.recvChan <- pkt:
	default:
		// Drop oldest buffered packet on burst saturation to keep newest telemetry fresh
		select {
		case <-s.recvChan:
		default:
		}
		select {
		case s.recvChan <- pkt:
		default:
		}
	}
}

func (s *UdpSocket) refreshReadDeadline() {
	s.mu.Lock()
	idle := s.idleTimeout
	s.mu.Unlock()

	if s.recvChan != nil {
		if idle > 0 {
			s.readDeadlineNs.Store(time.Now().Add(idle).UnixNano())
		}
		return
	}
	if idle > 0 && s.Conn != nil {
		_ = s.Conn.SetReadDeadline(time.Now().Add(idle))
	}
}

func (s *UdpSocket) refreshWriteDeadline() {
	s.mu.Lock()
	idle := s.idleTimeout
	s.mu.Unlock()

	if s.recvChan != nil {
		if idle > 0 {
			s.writeDeadlineNs.Store(time.Now().Add(idle).UnixNano())
		}
		return
	}
	if idle > 0 && s.Conn != nil {
		_ = s.Conn.SetWriteDeadline(time.Now().Add(idle))
	}
}

// SetIdleTimeout updates the internal idle timeout and refreshes current deadlines.
func (s *UdpSocket) SetIdleTimeout(d time.Duration) error {
	s.mu.Lock()
	s.idleTimeout = d
	s.mu.Unlock()

	if s.recvChan != nil {
		if d == 0 {
			s.readDeadlineNs.Store(0)
			s.writeDeadlineNs.Store(0)
		} else {
			s.refreshReadDeadline()
			s.refreshWriteDeadline()
		}
		return nil
	}

	if s.Conn == nil {
		return nil
	}
	if d == 0 {
		// Clear deadlines once and for all for 'forever' mode
		_ = s.Conn.SetDeadline(time.Time{})
	} else {
		s.refreshReadDeadline()
		s.refreshWriteDeadline()
	}
	return nil
}

// -----------------------------------------------------------------------------

// Write sends a datagram.
// Warning: Messages larger than MTU (usually 1500 bytes) will be fragmented.
// Messages larger than 64KB will fail.
func (s *UdpSocket) Write(p []byte) (n int, err error) {
	s.refreshWriteDeadline()
	if s.Conn == nil {
		return 0, net.ErrClosed
	}

	// If this is a transient server socket, reply to the specific remote address
	if s.TransientRemoteAddr != nil {
		select {
		case <-s.sessionClosed:
			return 0, net.ErrClosed
		default:
		}
		return s.Conn.WriteToUDP(p, s.TransientRemoteAddr)
	}

	// Otherwise, use standard Write (client-side connected socket)
	return s.Conn.Write(p)
}

// -----------------------------------------------------------------------------

func (s *UdpSocket) readSessionPacket() ([]byte, error) {
	s.mu.Lock()
	if s.RecvBuf != nil {
		buf := s.RecvBuf
		s.RecvBuf = nil
		s.mu.Unlock()
		return buf, nil
	}
	s.mu.Unlock()

	// Drain any already-buffered packet before evaluating read deadline expiration.
	select {
	case pkt := <-s.recvChan:
		s.refreshReadDeadline()
		return pkt, nil
	default:
	}

	rdNs := s.readDeadlineNs.Load()
	if rdNs == 0 {
		select {
		case pkt := <-s.recvChan:
			s.refreshReadDeadline()
			return pkt, nil
		case <-s.sessionClosed:
			return nil, net.ErrClosed
		}
	}

	remaining := time.Until(time.Unix(0, rdNs))
	if remaining <= 0 {
		return nil, os.ErrDeadlineExceeded
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()

	select {
	case pkt := <-s.recvChan:
		s.refreshReadDeadline()
		return pkt, nil
	case <-s.sessionClosed:
		return nil, net.ErrClosed
	case <-timer.C:
		return nil, os.ErrDeadlineExceeded
	}
}

// Read reads a datagram.
// Note: If 'p' is smaller than the incoming datagram, the excess data is discarded.
func (s *UdpSocket) Read(p []byte) (n int, err error) {
	if s.recvChan != nil {
		pkt, err := s.readSessionPacket()
		if err != nil {
			return 0, err
		}
		n = copy(p, pkt)
		return n, nil
	}

	s.refreshReadDeadline()
	if s.RecvBuf != nil {
		n := copy(p, s.RecvBuf)
		s.RecvBuf = nil // consumed
		return n, nil
	}

	if s.Conn == nil {
		return 0, net.ErrClosed
	}
	return s.Conn.Read(p)
}

// -----------------------------------------------------------------------------

// ReadMessage for UDP reads a full datagram (up to 64KB) and returns it.
func (s *UdpSocket) ReadMessage() ([]byte, error) {
	if s.recvChan != nil {
		return s.readSessionPacket()
	}

	s.refreshReadDeadline()
	if s.RecvBuf != nil {
		result := make([]byte, len(s.RecvBuf))
		copy(result, s.RecvBuf)
		s.RecvBuf = nil // consumed
		return result, nil
	}

	if s.Conn == nil {
		return nil, net.ErrClosed
	}

	tmp := make([]byte, udpMaxDatagramSize)
	n, remoteAddr, err := s.Conn.ReadFromUDP(tmp)
	if err != nil {
		return nil, err
	}
	if remoteAddr != nil {
		s.TransientRemoteAddr = remoteAddr
	}

	result := make([]byte, n)
	copy(result, tmp[:n])
	return result, nil
}

// -----------------------------------------------------------------------------

func (s *UdpSocket) Close() error {
	// Server-side transient sockets share the listener's *net.UDPConn;
	// closing a transient socket must not close the underlying listener.
	if s.TransientRemoteAddr != nil {
		s.closeOnce.Do(func() {
			if s.sessionClosed != nil {
				close(s.sessionClosed)
			}
			if s.onClose != nil {
				s.onClose()
			}
			s.mu.Lock()
			s.RecvBuf = nil
			s.Conn = nil
			s.mu.Unlock()
		})
		return nil
	}
	if s.Conn == nil {
		return nil
	}
	return s.Conn.Close()
}

// -----------------------------------------------------------------------------

// SetDeadline sets the read and write deadlines.
func (s *UdpSocket) SetDeadline(t time.Time) error {
	if s.recvChan != nil {
		var ns int64
		if !t.IsZero() {
			ns = t.UnixNano()
		}
		s.readDeadlineNs.Store(ns)
		s.writeDeadlineNs.Store(ns)
		return nil
	}
	if s.Conn == nil {
		return nil
	}
	return s.Conn.SetDeadline(t)
}

// -----------------------------------------------------------------------------

// SetReadDeadline sets the read deadline.
func (s *UdpSocket) SetReadDeadline(t time.Time) error {
	if s.recvChan != nil {
		var ns int64
		if !t.IsZero() {
			ns = t.UnixNano()
		}
		s.readDeadlineNs.Store(ns)
		return nil
	}
	if s.Conn == nil {
		return nil
	}
	return s.Conn.SetReadDeadline(t)
}

// -----------------------------------------------------------------------------

// SetWriteDeadline sets the write deadline.
func (s *UdpSocket) SetWriteDeadline(t time.Time) error {
	if s.recvChan != nil {
		var ns int64
		if !t.IsZero() {
			ns = t.UnixNano()
		}
		s.writeDeadlineNs.Store(ns)
		return nil
	}
	if s.Conn == nil {
		return nil
	}
	return s.Conn.SetWriteDeadline(t)
}

// -----------------------------------------------------------------------------

// LocalAddr returns the local network address.
func (s *UdpSocket) LocalAddr() net.Addr {
	if s.Conn == nil {
		return nil
	}
	return s.Conn.LocalAddr()
}

// -----------------------------------------------------------------------------

// RemoteAddr returns the remote network address.
func (s *UdpSocket) RemoteAddr() net.Addr {
	if s.TransientRemoteAddr != nil {
		return s.TransientRemoteAddr
	}
	if s.Conn == nil {
		return nil
	}
	return s.Conn.RemoteAddr()
}

var _ error = errors.New("")
