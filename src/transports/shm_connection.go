package transports

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/edsrzf/mmap-go"
)

// SPSC Ring Buffer Layout:
// [0-7]   : Head (Consumer Index) - uint64
// [8-15]  : Tail (Producer Index) - uint64
// [16-...] : Data Buffer

const (
	// Bidirectional: Two buffers of 32MB each
	BufferDataSize = 32 * 1024 * 1024 // 32 MB per direction
	MetaSize       = 128              // Header size
	TotalSize      = MetaSize + (BufferDataSize * 2)
)

// Metadata Offsets
const (
	// Buffer A (Client -> Server)
	OffsetHeadA = 0
	OffsetTailA = 8
	// Buffer B (Server -> Client)
	OffsetHeadB = 16
	OffsetTailB = 24

	OffsetServerStatus   = 32
	OffsetClientStatus   = 40
	OffsetServerActivity = 48
	OffsetClientActivity = 56
)

// Status Values
const (
	StatusIdle      = 0
	StatusListening = 1
	StatusConnected = 2
)

// ShmTransport implements a Shared Memory Ring Buffer transport.
type ShmTransport struct {
	File                     *os.File
	MMap                     mmap.MMap
	Role                     string  // "client" or "server"
	ProduceHead              *uint64 // Remote head (for capacity check)
	ProduceTail              *uint64 // Local tail
	ConsumeHead              *uint64 // Local head
	ConsumeTail              *uint64 // Remote tail
	ProduceData              []byte  // My write region
	ConsumeData              []byte  // My read region
	ServerStatus             *uint64
	ClientStatus             *uint64
	MyActivity               *uint64
	PeerActivity             *uint64
	lastObservedPeerActivity uint64
	readDeadline             atomic.Int64
	writeDeadline            atomic.Int64
	idleTimeout              time.Duration
	closed                   atomic.Bool
}

// -----------------------------------------------------------------------------

func NewShmTransport(f *os.File, m mmap.MMap, role string, timeout time.Duration) *ShmTransport {
	srvStatus := (*uint64)(unsafe.Pointer(&m[OffsetServerStatus]))
	cliStatus := (*uint64)(unsafe.Pointer(&m[OffsetClientStatus]))
	srvActivity := (*uint64)(unsafe.Pointer(&m[OffsetServerActivity]))
	cliActivity := (*uint64)(unsafe.Pointer(&m[OffsetClientActivity]))

	var pHead, pTail, cHead, cTail *uint64
	var pData, cData []byte
	var myActivity, peerActivity *uint64

	// Buffer A is [MetaSize : MetaSize + BufferDataSize]
	// Buffer B is [MetaSize + BufferDataSize : TotalSize]
	bufA := m[MetaSize : MetaSize+BufferDataSize]
	bufB := m[MetaSize+BufferDataSize : TotalSize]

	if role == "client" {
		// Client writes to A, reads from B
		pHead = (*uint64)(unsafe.Pointer(&m[OffsetHeadA]))
		pTail = (*uint64)(unsafe.Pointer(&m[OffsetTailA]))
		cHead = (*uint64)(unsafe.Pointer(&m[OffsetHeadB]))
		cTail = (*uint64)(unsafe.Pointer(&m[OffsetTailB]))
		pData = bufA
		cData = bufB
		myActivity = cliActivity
		peerActivity = srvActivity
	} else {
		// Server writes to B, reads from A
		pHead = (*uint64)(unsafe.Pointer(&m[OffsetHeadB]))
		pTail = (*uint64)(unsafe.Pointer(&m[OffsetTailB]))
		cHead = (*uint64)(unsafe.Pointer(&m[OffsetHeadA]))
		cTail = (*uint64)(unsafe.Pointer(&m[OffsetTailA]))
		pData = bufB
		cData = bufA
		myActivity = srvActivity
		peerActivity = cliActivity
	}

	t := &ShmTransport{
		File:                     f,
		MMap:                     m,
		Role:                     role,
		ProduceHead:              pHead,
		ProduceTail:              pTail,
		ConsumeHead:              cHead,
		ConsumeTail:              cTail,
		ProduceData:              pData,
		ConsumeData:              cData,
		ServerStatus:             srvStatus,
		ClientStatus:             cliStatus,
		MyActivity:               myActivity,
		PeerActivity:             peerActivity,
		lastObservedPeerActivity: atomic.LoadUint64(peerActivity),
		idleTimeout:              timeout,
	}

	if timeout > 0 {
		t.readDeadline.Store(time.Now().Add(timeout).UnixNano())
		t.writeDeadline.Store(time.Now().Add(timeout).UnixNano())
	}

	return t
}

func (t *ShmTransport) refreshReadDeadline() {
	if t.idleTimeout > 0 {
		t.readDeadline.Store(time.Now().Add(t.idleTimeout).UnixNano())
	}
	if t.MyActivity != nil {
		atomic.StoreUint64(t.MyActivity, uint64(time.Now().UnixNano()))
	}
}

func (t *ShmTransport) refreshWriteDeadline() {
	if t.idleTimeout > 0 {
		t.writeDeadline.Store(time.Now().Add(t.idleTimeout).UnixNano())
	}
	if t.MyActivity != nil {
		atomic.StoreUint64(t.MyActivity, uint64(time.Now().UnixNano()))
	}
}

// SetIdleTimeout updates the internal idle timeout and refreshes current deadlines.
func (t *ShmTransport) SetIdleTimeout(d time.Duration) error {
	t.idleTimeout = d
	if d == 0 {
		t.readDeadline.Store(0)
		t.writeDeadline.Store(0)
	} else {
		t.refreshReadDeadline()
		t.refreshWriteDeadline()
	}
	return nil
}

// -----------------------------------------------------------------------------

func (t *ShmTransport) SetDeadline(deadline time.Time) error {
	t.readDeadline.Store(deadline.UnixNano())
	t.writeDeadline.Store(deadline.UnixNano())
	return nil
}

// -----------------------------------------------------------------------------

func (t *ShmTransport) SetReadDeadline(deadline time.Time) error {
	t.readDeadline.Store(deadline.UnixNano())
	return nil
}

// -----------------------------------------------------------------------------

func (t *ShmTransport) SetWriteDeadline(deadline time.Time) error {
	t.writeDeadline.Store(deadline.UnixNano())
	return nil
}

// -----------------------------------------------------------------------------

// Write (Producer Role)
func (t *ShmTransport) Write(p []byte) (n int, err error) {
	lenData := uint64(len(p))
	// Framing: 4-byte header
	totalLen := 4 + lenData

	if totalLen > BufferDataSize {
		return 0, io.ErrShortBuffer
	}

	for {
		if t.closed.Load() {
			return 0, io.ErrClosedPipe
		}

		tail := atomic.LoadUint64(t.ProduceTail)
		head := atomic.LoadUint64(t.ProduceHead)

		if tail-head+totalLen > BufferDataSize {
			wd := t.writeDeadline.Load()
			if wd > 0 && time.Now().UnixNano() > wd {
				return 0, os.ErrDeadlineExceeded
			}
			time.Sleep(1 * time.Microsecond)
			continue
		}

		// 1. Write Header (4 bytes, BigEndian)
		header := make([]byte, 4)
		binary.BigEndian.PutUint32(header, uint32(lenData))
		t.writeToRing(tail, header)

		// 2. Write Data
		if lenData > 0 {
			t.writeToRing(tail+4, p)
		}

		atomic.AddUint64(t.ProduceTail, totalLen)
		t.refreshWriteDeadline()

		return int(lenData), nil
	}
}

// writeToRing is a helper to handle wrapped writes.
func (t *ShmTransport) writeToRing(offset uint64, p []byte) {
	lenData := uint64(len(p))
	writeIdx := offset % BufferDataSize

	if writeIdx+lenData <= BufferDataSize {
		copy(t.ProduceData[writeIdx:], p)
	} else {
		firstPart := BufferDataSize - writeIdx
		copy(t.ProduceData[writeIdx:], p[:firstPart])
		copy(t.ProduceData[0:], p[firstPart:])
	}
}

// -----------------------------------------------------------------------------

// Read (Consumer Role)
func (t *ShmTransport) Read(p []byte) (n int, err error) {
	for {
		if t.closed.Load() {
			return 0, io.EOF
		}

		tail := atomic.LoadUint64(t.ConsumeTail)
		head := atomic.LoadUint64(t.ConsumeHead)

		// Check if we have at least the 4-byte header
		if tail-head < 4 {
			// HEARTBEAT AUDIT FIX: Check if peer is active even without data
			activity := atomic.LoadUint64(t.PeerActivity)
			if activity > t.lastObservedPeerActivity {
				t.refreshReadDeadline()
				t.lastObservedPeerActivity = activity
			}

			rd := t.readDeadline.Load()
			if rd > 0 && time.Now().UnixNano() > rd {
				return 0, os.ErrDeadlineExceeded
			}
			time.Sleep(1 * time.Microsecond)
			continue
		}

		// 1. Read Header
		header := make([]byte, 4)
		t.readFromRing(head, header)
		length := uint64(binary.BigEndian.Uint32(header))

		// 2. Check if entire frame is available
		if tail-head < 4+length {
			// Frame incomplete, wait
			time.Sleep(1 * time.Microsecond)
			continue
		}

		// 3. Handle Heartbeats (Length 0)
		if length == 0 {
			atomic.AddUint64(t.ConsumeHead, 4)
			continue
		}

		// 4. Read Body
		if uint64(len(p)) < length {
			return 0, io.ErrShortBuffer
		}

		t.readFromRing(head+4, p[:length])

		atomic.AddUint64(t.ConsumeHead, 4+length)
		t.refreshReadDeadline()

		return int(length), nil
	}
}

// readFromRing is a helper to handle wrapped reads.
func (t *ShmTransport) readFromRing(offset uint64, p []byte) {
	lenData := uint64(len(p))
	readIdx := offset % BufferDataSize

	if readIdx+lenData <= BufferDataSize {
		copy(p, t.ConsumeData[readIdx:readIdx+lenData])
	} else {
		firstPart := BufferDataSize - readIdx
		copy(p[:firstPart], t.ConsumeData[readIdx:])
		copy(p[firstPart:], t.ConsumeData[:lenData-firstPart])
	}
}

// -----------------------------------------------------------------------------

// ReadMessage for SHM reads exactly one frame.
func (t *ShmTransport) ReadMessage() ([]byte, error) {
	for {
		if t.closed.Load() {
			return nil, io.EOF
		}

		tail := atomic.LoadUint64(t.ConsumeTail)
		head := atomic.LoadUint64(t.ConsumeHead)

		if tail-head < 4 {
			// HEARTBEAT AUDIT FIX: Check if peer is active even without data
			activity := atomic.LoadUint64(t.PeerActivity)
			if activity > t.lastObservedPeerActivity {
				t.refreshReadDeadline()
				t.lastObservedPeerActivity = activity
			}

			rd := t.readDeadline.Load()
			if rd > 0 && time.Now().UnixNano() > rd {
				return nil, os.ErrDeadlineExceeded
			}
			time.Sleep(1 * time.Microsecond)
			continue
		}

		// 1. Read Header
		header := make([]byte, 4)
		t.readFromRing(head, header)
		length := uint64(binary.BigEndian.Uint32(header))

		// 2. Check if entire frame is available
		if tail-head < 4+length {
			time.Sleep(1 * time.Microsecond)
			continue
		}

		// 3. Handle Heartbeats (Length 0)
		if length == 0 {
			atomic.AddUint64(t.ConsumeHead, 4)
			continue
		}

		// 4. Allocate and Read Body
		buf := make([]byte, length)
		t.readFromRing(head+4, buf)

		atomic.AddUint64(t.ConsumeHead, 4+length)
		t.refreshReadDeadline()
		return buf, nil
	}
}

// -----------------------------------------------------------------------------

func (t *ShmTransport) Close() error {
	// Mark as closed BEFORE unmapping to stop spin-loops safely
	if t.closed.Swap(true) {
		return nil // Already closed
	}

	// Flush? MMap usually syncs periodically.
	if err := t.MMap.Unmap(); err != nil {
		_ = t.File.Close() // Best effort close file
		return err
	}
	return t.File.Close()
}

// -----------------------------------------------------------------------------

// LocalAddr returns the local network address (SHM pseudo-address).
func (t *ShmTransport) LocalAddr() net.Addr {
	return ShmAddr{}
}

// -----------------------------------------------------------------------------

// RemoteAddr returns the remote network address (SHM pseudo-address).
func (t *ShmTransport) RemoteAddr() net.Addr {
	return ShmAddr{}
}

// -----------------------------------------------------------------------------

// ShmAddr implements net.Addr for Shared Memory.
type ShmAddr struct{}

func (a ShmAddr) Network() string { return "shm" }
func (a ShmAddr) String() string  { return "memory" }
