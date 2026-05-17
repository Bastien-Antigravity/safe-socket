package facade

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// RUDP Header Constants
const (
	RudpHeaderSize = 17
	RudpTypeData   = 0
	RudpTypeAck    = 1
)

// ReliableConnection wraps an unreliable transport (UDP) to provide:
// - Sequence numbers
// - Acknowledgments (ACKs)
// - Retransmissions
// - Deduplication
type ReliableConnection struct {
	interfaces.TransportConnection

	nextSeq       uint64
	lastRemoteSeq uint64

	unacked      map[uint64]*pendingPacket
	receivedSeqs map[uint64]time.Time // For deduplication

	mu        sync.Mutex
	stopRetry chan struct{}
	closeOnce sync.Once

	retryInterval time.Duration
	maxRetries    int
}

type pendingPacket struct {
	data    []byte
	sentAt  time.Time
	retries int
}

// -----------------------------------------------------------------------------

func NewReliableConnection(conn interfaces.TransportConnection) *ReliableConnection {
	rc := &ReliableConnection{
		TransportConnection: conn,
		unacked:             make(map[uint64]*pendingPacket),
		receivedSeqs:        make(map[uint64]time.Time),
		stopRetry:           make(chan struct{}),
		retryInterval:       150 * time.Millisecond,
		maxRetries:          15, // High resilience
		nextSeq:             1,  // 0 is "No ACK"
	}
	go rc.retryLoop()
	return rc
}

// -----------------------------------------------------------------------------

func (c *ReliableConnection) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	seq := c.nextSeq
	c.nextSeq++
	ack := c.lastRemoteSeq
	c.mu.Unlock()

	// 1. Construct RUDP Packet
	// [Type(1)] [Seq(8)] [Ack(8)] [Payload]
	packet := make([]byte, RudpHeaderSize+len(p))
	packet[0] = RudpTypeData
	binary.BigEndian.PutUint64(packet[1:9], seq)
	binary.BigEndian.PutUint64(packet[9:17], ack)
	copy(packet[17:], p)

	// 2. Store for retransmission
	c.mu.Lock()
	c.unacked[seq] = &pendingPacket{
		data:   packet,
		sentAt: time.Now(),
	}
	c.mu.Unlock()

	// 3. Send
	_, err = c.TransportConnection.Write(packet)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// -----------------------------------------------------------------------------

func (c *ReliableConnection) Read(p []byte) (n int, err error) {
	for {
		buf := make([]byte, 65535) // Max UDP packet size
		nRaw, err := c.TransportConnection.Read(buf)
		if err != nil {
			return 0, err
		}

		if nRaw < RudpHeaderSize {
			continue // Junk packet
		}

		pType := buf[0]
		remoteSeq := binary.BigEndian.Uint64(buf[1:9])
		remoteAck := binary.BigEndian.Uint64(buf[9:17])

		c.handleRemoteAck(remoteAck)

		if pType == RudpTypeAck {
			continue // Just an ACK, loop for more data
		}

		if pType == RudpTypeData {
			c.mu.Lock()
			// Update our last remote seq seen
			if remoteSeq > c.lastRemoteSeq {
				c.lastRemoteSeq = remoteSeq
			}

			// Deduplication
			if _, exists := c.receivedSeqs[remoteSeq]; exists {
				c.mu.Unlock()
				c.sendAck(remoteSeq)
				continue
			}
			c.receivedSeqs[remoteSeq] = time.Now()
			c.mu.Unlock()

			// Send ACK
			c.sendAck(remoteSeq)

			// Return payload
			payload := buf[RudpHeaderSize:nRaw]
			if len(payload) > len(p) {
				return 0, errors.New("short buffer")
			}
			copy(p, payload)
			return len(payload), nil
		}
	}
}

// -----------------------------------------------------------------------------

func (c *ReliableConnection) ReadMessage() ([]byte, error) {
	buf := make([]byte, 65535)
	n, err := c.Read(buf)
	if err != nil {
		return nil, err
	}
	res := make([]byte, n)
	copy(res, buf[:n])
	return res, nil
}

// -----------------------------------------------------------------------------

func (c *ReliableConnection) handleRemoteAck(ack uint64) {
	if ack == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.unacked, ack)
}

func (c *ReliableConnection) sendAck(seq uint64) {
	// [Type(1)] [Seq(8)] [Ack(8)]
	ackPacket := make([]byte, RudpHeaderSize)
	ackPacket[0] = RudpTypeAck
	binary.BigEndian.PutUint64(ackPacket[1:9], 0)
	binary.BigEndian.PutUint64(ackPacket[9:17], seq)

	_, _ = c.TransportConnection.Write(ackPacket)
}

// -----------------------------------------------------------------------------

func (c *ReliableConnection) retryLoop() {
	ticker := time.NewTicker(c.retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.performRetries()
		case <-c.stopRetry:
			return
		}
	}
}

func (c *ReliableConnection) performRetries() {
	c.mu.Lock()
	now := time.Now()
	var toRetry [][]byte
	var toDrop []uint64

	for seq, pkt := range c.unacked {
		if now.Sub(pkt.sentAt) > c.retryInterval {
			pkt.retries++
			if pkt.retries > c.maxRetries {
				toDrop = append(toDrop, seq)
			} else {
				pkt.sentAt = now
				toRetry = append(toRetry, pkt.data)
			}
		}
	}
	c.mu.Unlock()

	for _, seq := range toDrop {
		c.mu.Lock()
		delete(c.unacked, seq)
		c.mu.Unlock()
	}

	for _, data := range toRetry {
		_, _ = c.TransportConnection.Write(data)
	}

	// Periodic cleanup of receivedSeqs
	c.mu.Lock()
	for seq, t := range c.receivedSeqs {
		if now.Sub(t) > 30*time.Second {
			delete(c.receivedSeqs, seq)
		}
	}
	c.mu.Unlock()
}

// -----------------------------------------------------------------------------

func (c *ReliableConnection) Close() error {
	c.closeOnce.Do(func() {
		close(c.stopRetry)
	})
	return c.TransportConnection.Close()
}
