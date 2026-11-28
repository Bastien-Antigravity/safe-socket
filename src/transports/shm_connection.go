package transports

// =============================================================================
// ESSENTIAL PROCESS:
// Implements lock-free single-producer single-consumer (SPSC) shared memory (SHM)
// ring buffers via memory-mapped files (mmap) with producer/consumer synchronization
// and adaptive spin-sleep backoff for ultra-low latency IPC.
//
// DATA FLOW:
// 1. Input: Byte slices to write into bidirectional memory-mapped ring buffers.
// 2. Logic: Serializes concurrent writers/readers with local mutexes, coordinates
//    cross-process head/tail offsets atomically, and backs off adaptively when idle.
// 3. Output: Framed payload delivery between local host processes.
//
// KEY PARAMETERS:
// - ShmTransport: TransportConnection implementation backed by memory-mapped file.
// - BufferDataSize: 32MB buffer capacity per direction.
// =============================================================================

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"runtime"
	"sync"
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
	unmapped                 *atomic.Bool
	mmapMu                   *sync.RWMutex
	writeMu                  sync.Mutex
	readMu                   sync.Mutex
	onClose                  func() error
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
		unmapped:                 &atomic.Bool{},
		mmapMu:                   &sync.RWMutex{},
	}

	if timeout > 0 {
		t.readDeadline.Store(time.Now().Add(timeout).UnixNano())
		t.writeDeadline.Store(time.Now().Add(timeout).UnixNano())
	}

	return t
}

// newSessionHandle creates a per-client session view sharing the listener's mmap and synchronization
// without owning the underlying File/MMap unmap lifecycle.
func (t *ShmTransport) newSessionHandle(timeout time.Duration, onClose func() error) *ShmTransport {
	s := &ShmTransport{
		File:                     t.File,
		MMap:                     t.MMap,
		Role:                     t.Role,
		ProduceHead:              t.ProduceHead,
		ProduceTail:              t.ProduceTail,
		ConsumeHead:              t.ConsumeHead,
		ConsumeTail:              t.ConsumeTail,
		ProduceData:              t.ProduceData,
		ConsumeData:              t.ConsumeData,
		ServerStatus:             t.ServerStatus,
		ClientStatus:             t.ClientStatus,
		MyActivity:               t.MyActivity,
		PeerActivity:             t.PeerActivity,
		lastObservedPeerActivity: atomic.LoadUint64(t.PeerActivity),
		idleTimeout:              timeout,
		unmapped:                 t.unmapped,
		mmapMu:                   t.mmapMu,
		onClose:                  onClose,
	}
	if timeout > 0 {
		s.readDeadline.Store(time.Now().Add(timeout).UnixNano())
		s.writeDeadline.Store(time.Now().Add(timeout).UnixNano())
	}
	return s
}

func shmSpinWait(spin *int) {
	*spin++
	switch {
	case *spin <= 16:
		runtime.Gosched()
	case *spin <= 64:
		time.Sleep(1 * time.Microsecond)
	case *spin <= 256:
		time.Sleep(10 * time.Microsecond)
	default:
		time.Sleep(50 * time.Microsecond)
	}
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
	var ns int64
	if !deadline.IsZero() {
		ns = deadline.UnixNano()
	}
	t.readDeadline.Store(ns)
	t.writeDeadline.Store(ns)
	return nil
}

// -----------------------------------------------------------------------------

func (t *ShmTransport) SetReadDeadline(deadline time.Time) error {
	var ns int64
	if !deadline.IsZero() {
		ns = deadline.UnixNano()
	}
	t.readDeadline.Store(ns)
	return nil
}

// -----------------------------------------------------------------------------

func (t *ShmTransport) SetWriteDeadline(deadline time.Time) error {
	var ns int64
	if !deadline.IsZero() {
		ns = deadline.UnixNano()
	}
	t.writeDeadline.Store(ns)
	return nil
}

// -----------------------------------------------------------------------------

// Write (Producer Role, synchronized across goroutines within this process).
func (t *ShmTransport) Write(p []byte) (n int, err error) {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	lenData := uint64(len(p))
	// Framing: 4-byte header
	totalLen := 4 + lenData

	if totalLen > BufferDataSize {
		return 0, io.ErrShortBuffer
	}

	spin := 0
	for {
		if t.closed.Load() || t.unmapped.Load() {
			return 0, io.ErrClosedPipe
		}

		t.mmapMu.RLock()
		if t.closed.Load() || t.unmapped.Load() {
			t.mmapMu.RUnlock()
			return 0, io.ErrClosedPipe
		}

		tail := atomic.LoadUint64(t.ProduceTail)
		head := atomic.LoadUint64(t.ProduceHead)

		if tail-head+totalLen > BufferDataSize {
			t.mmapMu.RUnlock()
			wd := t.writeDeadline.Load()
			if wd > 0 && time.Now().UnixNano() > wd {
				return 0, os.ErrDeadlineExceeded
			}
			shmSpinWait(&spin)
			continue
		}

		// 1. Write Header (4 bytes, BigEndian)
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], uint32(lenData))
		t.writeToRing(tail, header[:])

		// 2. Write Data
		if lenData > 0 {
			t.writeToRing(tail+4, p)
		}

		atomic.AddUint64(t.ProduceTail, totalLen)
		t.refreshWriteDeadline()
		t.mmapMu.RUnlock()

		return int(lenData), nil
	}
}

// writeToRing is a helper to handle wrapped writes.
func (t *ShmTransport) writeToRing(offset uint64, p []byte) {
	if len(p) == 0 {
		return
	}
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

// Read (Consumer Role) reads a framed payload into the caller's slice p.
func (t *ShmTransport) Read(p []byte) (n int, err error) {
	_, n, err = t.readFrame(p, false)
	return n, err
}

// ReadMessage for SHM reads exactly one frame into a newly allocated buffer.
func (t *ShmTransport) ReadMessage() ([]byte, error) {
	buf, _, err := t.readFrame(nil, true)
	return buf, err
}

func (t *ShmTransport) readFrame(dst []byte, allocate bool) ([]byte, int, error) {
	t.readMu.Lock()
	defer t.readMu.Unlock()

	spin := 0
	for {
		if t.closed.Load() || t.unmapped.Load() {
			return nil, 0, io.EOF
		}

		t.mmapMu.RLock()
		if t.closed.Load() || t.unmapped.Load() {
			t.mmapMu.RUnlock()
			return nil, 0, io.EOF
		}

		tail := atomic.LoadUint64(t.ConsumeTail)
		head := atomic.LoadUint64(t.ConsumeHead)

		// Check if we have at least the 4-byte header
		if tail-head < 4 {
			// Check if peer cleanly closed the connection
			peerClosed := (t.Role == "client" && atomic.LoadUint64(t.ServerStatus) == StatusIdle) ||
				(t.Role == "server" && atomic.LoadUint64(t.ClientStatus) == StatusIdle)
			if peerClosed {
				t.mmapMu.RUnlock()
				return nil, 0, io.EOF
			}

			// HEARTBEAT AUDIT FIX: Check if peer is active even without data
			activity := atomic.LoadUint64(t.PeerActivity)
			if activity > t.lastObservedPeerActivity {
				t.refreshReadDeadline()
				t.lastObservedPeerActivity = activity
			}
			t.mmapMu.RUnlock()

			rd := t.readDeadline.Load()
			if rd > 0 && time.Now().UnixNano() > rd {
				return nil, 0, os.ErrDeadlineExceeded
			}
			shmSpinWait(&spin)
			continue
		}

		// 1. Read Header
		var header [4]byte
		t.readFromRing(head, header[:])
		length := uint64(binary.BigEndian.Uint32(header[:]))

		if 4+length > BufferDataSize {
			t.mmapMu.RUnlock()
			return nil, 0, io.ErrShortBuffer
		}

		// 2. Check if entire frame is available
		if tail-head < 4+length {
			t.mmapMu.RUnlock()
			rd := t.readDeadline.Load()
			if rd > 0 && time.Now().UnixNano() > rd {
				return nil, 0, os.ErrDeadlineExceeded
			}
			shmSpinWait(&spin)
			continue
		}

		// 3. Handle Heartbeats (Length 0)
		if length == 0 {
			atomic.AddUint64(t.ConsumeHead, 4)
			t.refreshReadDeadline()
			t.mmapMu.RUnlock()
			spin = 0
			continue
		}

		// 4. Read Body
		if !allocate {
			if uint64(len(dst)) < length {
				t.mmapMu.RUnlock()
				return nil, 0, io.ErrShortBuffer
			}
			t.readFromRing(head+4, dst[:length])
			atomic.AddUint64(t.ConsumeHead, 4+length)
			t.refreshReadDeadline()
			t.mmapMu.RUnlock()
			return nil, int(length), nil
		}

		buf := make([]byte, length)
		t.readFromRing(head+4, buf)
		atomic.AddUint64(t.ConsumeHead, 4+length)
		t.refreshReadDeadline()
		t.mmapMu.RUnlock()
		return buf, int(length), nil
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

func (t *ShmTransport) Close() error {
	// Mark as closed BEFORE unmapping to stop spin-loops safely
	if t.closed.Swap(true) {
		return nil // Already closed
	}

	if t.onClose != nil {
		return t.onClose()
	}

	t.mmapMu.Lock()
	defer t.mmapMu.Unlock()

	t.unmapped.Store(true)

	if t.Role == "client" && t.ClientStatus != nil {
		atomic.StoreUint64(t.ClientStatus, StatusIdle)
	} else if t.Role == "server" && t.ServerStatus != nil {
		atomic.StoreUint64(t.ServerStatus, StatusIdle)
	}

	if err := t.MMap.Unmap(); err != nil {
		_ = t.File.Close()
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
