package transports

import (
	"os"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"

	"github.com/edsrzf/mmap-go"
)

// -----------------------------------------------------------------------------

// ConnectShm opens the file and memory maps it.
// If the file is smaller than TotalSize, it is grown.
func ConnectShm(path string, timeout time.Duration) (interfaces.TransportConnection, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	// Grow file if needed
	if info.Size() < int64(TotalSize) {
		if err := file.Truncate(int64(TotalSize)); err != nil {
			file.Close()
			return nil, err
		}
	}

	// Map the file
	m, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		file.Close()
		return nil, err
	}

	return NewShmTransport(file, m, timeout), nil
}
