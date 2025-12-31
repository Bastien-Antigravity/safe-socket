package transports

import (
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
	// Fixed Buffer Size: 64MB + 16 bytes header
	// Using a power of 2 for easy wrapping (though modulo works too).
	// Let's settle on a strict 64MB data payload.
	BufferDataSize = 64 * 1024 * 1024 // 64 MB
	MetaSize       = 16
	TotalSize      = MetaSize + BufferDataSize
)

// ShmTransport implements a Shared Memory Ring Buffer transport.
type ShmTransport struct {
	File    *os.File
	MMap    mmap.MMap
	Head    *uint64 // Pointer to shared memory Head
	Tail    *uint64 // Pointer to shared memory Tail
	Data    []byte  // Slice pointing to shared data region
	Timeout time.Duration
}

// -----------------------------------------------------------------------------

func NewShmTransport(f *os.File, m mmap.MMap, timeout time.Duration) *ShmTransport {
	// Map the pointers directly to the byte slice
	// Go slices are safe, but we need atomic access to the headers.
	// Using unsafe to cast 8 bytes to *uint64.

	// Pointers to the header in the MMap region
	headerPtr := unsafe.Pointer(&m[0])
	head := (*uint64)(headerPtr)

	tailPtr := unsafe.Pointer(&m[8])
	tail := (*uint64)(tailPtr)

	return &ShmTransport{
		File:    f,
		MMap:    m,
		Head:    head,
		Tail:    tail,
		Data:    m[MetaSize:],
		Timeout: timeout,
	}
}

// -----------------------------------------------------------------------------

// Write (Producer Role)
// Writes data to the Ring Buffer.
func (t *ShmTransport) Write(p []byte) (n int, err error) {
	lenData := uint64(len(p))
	if lenData > BufferDataSize {
		return 0, io.ErrShortBuffer // Too big for entire buffer
	}

	// 1. Check available space
	// We use standard ring buffer arithmetic.
	// We only wrap Tail virtually (Tail keeps increasing).
	// Actual index = Tail % BufferDataSize.

	deadline := time.Now().Add(t.Timeout)
	for {
		tail := atomic.LoadUint64(t.Tail) // Where we want to write
		head := atomic.LoadUint64(t.Head) // Where consumer is

		// Capacity check: Tail - Head < BufferDataSize
		if tail-head+lenData > BufferDataSize {
			// Full. Spin-wait.
			if t.Timeout > 0 && time.Now().After(deadline) {
				return 0, os.ErrDeadlineExceeded
			}
			time.Sleep(1 * time.Microsecond) // Polite spin
			continue
		}

		// 2. Write Data
		// Handle wrapping if the write crosses the boundary
		writeIdx := tail % BufferDataSize

		if writeIdx+lenData <= BufferDataSize {
			// Continuous write
			copy(t.Data[writeIdx:], p)
		} else {
			// Split write
			firstPart := BufferDataSize - writeIdx
			copy(t.Data[writeIdx:], p[:firstPart])
			copy(t.Data[0:], p[firstPart:])
		}

		// 3. Commit: Update Tail
		// Commit-Store ensure data is visible before index update (on x86 this is free, on ARM needs barrier)
		// Go atomic.Store acts as a release barrier.
		atomic.AddUint64(t.Tail, lenData)

		return int(lenData), nil
	}
}

// -----------------------------------------------------------------------------

// Read (Consumer Role)
func (t *ShmTransport) Read(p []byte) (n int, err error) {
	// Blocking Read
	deadline := time.Now().Add(t.Timeout)
	for {
		tail := atomic.LoadUint64(t.Tail)
		head := atomic.LoadUint64(t.Head)

		if head == tail {
			// Empty. Spin-wait.
			if t.Timeout > 0 && time.Now().After(deadline) {
				return 0, os.ErrDeadlineExceeded
			}
			time.Sleep(1 * time.Microsecond)
			continue
		}

		// Calculate how much we can read
		available := tail - head
		lenBuf := uint64(len(p))

		// If needed, we frame-read.
		// Actually, the interface says "Read(p)". Typical Socket behavior is to read what's available up to len(p).
		toRead := available
		if toRead > lenBuf {
			toRead = lenBuf
		}

		readIdx := head % BufferDataSize

		if readIdx+toRead <= BufferDataSize {
			// Continuous read
			copy(p, t.Data[readIdx:readIdx+toRead])
		} else {
			// Split read
			firstPart := BufferDataSize - readIdx
			copy(p, t.Data[readIdx:BufferDataSize])
			copy(p[firstPart:], t.Data[0:toRead-firstPart])
		}

		// Commit: Update Head
		atomic.AddUint64(t.Head, toRead)

		return int(toRead), nil
	}
}

// -----------------------------------------------------------------------------

// ReadMessage for SHM reads all available data or blocks until data arrives.
// It acts as a "Frame Read" if the producer writes in frames, but technically it reads "everything available".
func (t *ShmTransport) ReadMessage() ([]byte, error) {
	// Blocking Read logic duplicated from Read() but with allocation
	deadline := time.Now().Add(t.Timeout)
	for {
		tail := atomic.LoadUint64(t.Tail)
		head := atomic.LoadUint64(t.Head)

		if head == tail {
			// Empty. Spin-wait.
			if t.Timeout > 0 && time.Now().After(deadline) {
				return nil, os.ErrDeadlineExceeded
			}
			time.Sleep(1 * time.Microsecond)
			continue
		}

		// Calculate how much we can read
		available := tail - head
		// We limit to a reasonable max if needed, but here we read all available
		if available > BufferDataSize {
			// Should strictly not happen unless overrun, but handle
			available = BufferDataSize
		}

		toRead := available
		buf := make([]byte, toRead)

		readIdx := head % BufferDataSize
		if readIdx+toRead <= BufferDataSize {
			copy(buf, t.Data[readIdx:readIdx+toRead])
		} else {
			firstPart := BufferDataSize - readIdx
			copy(buf[:firstPart], t.Data[readIdx:])
			copy(buf[firstPart:], t.Data[:toRead-firstPart])
		}

		atomic.AddUint64(t.Head, toRead)
		return buf, nil
	}
}

// -----------------------------------------------------------------------------

func (t *ShmTransport) Close() error {
	// Flush? MMap usually syncs periodically.
	t.MMap.Unmap()
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

// ShmAddr implements net.Addr for Shared Memory.
type ShmAddr struct{}

func (a ShmAddr) Network() string { return "shm" }
func (a ShmAddr) String() string  { return "memory" }
