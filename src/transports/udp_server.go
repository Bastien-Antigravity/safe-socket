package transports

// =============================================================================
// ESSENTIAL PROCESS:
// Implements UDP listener handling with full 64KB datagram reads and per-peer
// session demultiplexing so multi-packet and RUDP conversations stay routed to
// their active session connection.
//
// DATA FLOW:
// 1. Input: Local UDP bind address host:port and timeout duration.
// 2. Logic: Binds net.ListenUDP, runs demuxLoop reading up to 65,535-byte datagrams,
//    routes packets by remoteAddr to existing sessions or emits new sessions via Accept().
// 3. Output: TransportConnection representing the peer datagram session.
//
// KEY PARAMETERS:
// - ListenUDP: Binds UDP listening socket and starts demuxLoop.
// - UdpListener: Demultiplexing listener implementation for UDP datagrams.
// =============================================================================

import (
	"net"
	"sync"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// UdpListener implements interfaces.TransportListener for UDP with per-peer session demultiplexing.
type UdpListener struct {
	Conn       *net.UDPConn
	Timeout    time.Duration
	sessionsMu sync.Mutex
	sessions   map[string]*UdpSocket
	acceptChan chan *UdpSocket
	errChan    chan error
	closed     chan struct{}
	closeOnce  sync.Once
}

// -----------------------------------------------------------------------------

func ListenUDP(address string, timeout time.Duration) (interfaces.TransportListener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	// Optimizations (match client)
	_ = conn.SetReadBuffer(4 * 1024 * 1024)
	_ = conn.SetWriteBuffer(4 * 1024 * 1024)

	l := &UdpListener{
		Conn:       conn,
		Timeout:    timeout,
		sessions:   make(map[string]*UdpSocket),
		acceptChan: make(chan *UdpSocket, 128),
		errChan:    make(chan error, 1),
		closed:     make(chan struct{}),
	}
	go l.demuxLoop()
	return l, nil
}

func (l *UdpListener) demuxLoop() {
	buf := make([]byte, udpMaxDatagramSize)
	for {
		n, remoteAddr, err := l.Conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-l.closed:
				return
			default:
				select {
				case l.errChan <- err:
				default:
				}
				return
			}
		}

		pkt := make([]byte, n)
		copy(pkt, buf[:n])

		key := remoteAddr.String()
		l.sessionsMu.Lock()
		existing, ok := l.sessions[key]
		if ok {
			l.sessionsMu.Unlock()
			existing.enqueuePacket(pkt)
			continue
		}

		var session *UdpSocket
		session = NewTransientUdpSocket(l.Conn, remoteAddr, pkt, l.Timeout, func() {
			l.sessionsMu.Lock()
			if cur, exists := l.sessions[key]; exists && cur == session {
				delete(l.sessions, key)
			}
			l.sessionsMu.Unlock()
		})
		l.sessions[key] = session
		l.sessionsMu.Unlock()

		select {
		case l.acceptChan <- session:
		case <-l.closed:
			_ = session.Close()
			return
		}
	}
}

// -----------------------------------------------------------------------------

// Accept waits for the next new remote UDP peer session and returns its demultiplexed socket.
func (l *UdpListener) Accept() (interfaces.TransportConnection, error) {
	select {
	case session := <-l.acceptChan:
		return session, nil
	case err := <-l.errChan:
		return nil, err
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

// -----------------------------------------------------------------------------

func (l *UdpListener) Close() error {
	var err error
	l.closeOnce.Do(func() {
		close(l.closed)
		if l.Conn != nil {
			err = l.Conn.Close()
		}
	})
	return err
}

// -----------------------------------------------------------------------------

func (l *UdpListener) Addr() net.Addr {
	if l.Conn == nil {
		return nil
	}
	return l.Conn.LocalAddr()
}
