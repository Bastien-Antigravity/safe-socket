package main

/*
#include <string.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/Bastien-Antigravity/safe-socket"
	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

var (
	registry    sync.Map
	nextHandle  int32
	lastError   string
	registryMut sync.Mutex
)

// Register stores a socket/connection and returns a handle.
func Register(val interface{}) int32 {
	registryMut.Lock()
	defer registryMut.Unlock()
	nextHandle++
	registry.Store(nextHandle, val)
	return nextHandle
}

// Get retrieves a value from the registry.
func Get(handle int32) (interface{}, bool) {
	return registry.Load(handle)
}

// Unregister removes a handle from the registry.
func Unregister(handle int32) {
	registry.Delete(handle)
}

//export GetLastError
func GetLastError() *C.char {
	return C.CString(lastError)
}

func setError(err error) {
	if err != nil {
		lastError = err.Error()
	} else {
		lastError = ""
	}
}

//export CreateSocket
func CreateSocket(profileName, address, publicIP, socketType *C.char, autoConnect C.int) int32 {
	pName := C.GoString(profileName)
	addr := C.GoString(address)
	pIP := C.GoString(publicIP)
	sType := C.GoString(socketType)
	auto := autoConnect != 0

	sock, err := safesocket.Create(pName, addr, pIP, sType, auto)
	if err != nil {
		setError(err)
		return -1
	}

	return Register(sock)
}

//export SocketOpen
func SocketOpen(handle int32) int32 {
	val, ok := Get(handle)
	if !ok {
		setError(errors.New("invalid handle"))
		return -1
	}

	sock, ok := val.(interfaces.Socket)
	if !ok {
		setError(errors.New("handle is not a socket"))
		return -1
	}

	err := sock.Open()
	setError(err)
	if err != nil {
		return -1
	}
	return 0
}

//export SocketClose
func SocketClose(handle int32) int32 {
	val, ok := Get(handle)
	if !ok {
		setError(errors.New("invalid handle"))
		return -1
	}

	if sock, ok := val.(interfaces.Socket); ok {
		err := sock.Close()
		setError(err)
		Unregister(handle)
		if err != nil {
			return -1
		}
		return 0
	}

	if conn, ok := val.(interfaces.TransportConnection); ok {
		err := conn.Close()
		setError(err)
		Unregister(handle)
		if err != nil {
			return -1
		}
		return 0
	}

	setError(errors.New("invalid handle type"))
	return -1
}

//export SocketSend
func SocketSend(handle int32, data *C.uchar, length C.int) int32 {
	val, ok := Get(handle)
	if !ok {
		setError(errors.New("invalid handle"))
		return -1
	}

	buf := C.GoBytes(unsafe.Pointer(data), length)

	if sock, ok := val.(interfaces.Socket); ok {
		err := sock.Send(buf)
		setError(err)
		if err != nil {
			return -1
		}
		return 0
	}

	if conn, ok := val.(interfaces.TransportConnection); ok {
		n, err := conn.Write(buf)
		setError(err)
		if err != nil {
			return -1
		}
		return int32(n)
	}

	setError(errors.New("invalid handle type for Send"))
	return -1
}

//export SocketReceive
func SocketReceive(handle int32, buffer *C.uchar, maxLength C.int) int32 {
	val, ok := Get(handle)
	if !ok {
		setError(errors.New("invalid handle"))
		return -1
	}

	var data []byte
	var err error

	if sock, ok := val.(interfaces.Socket); ok {
		data, err = sock.Receive()
	} else if conn, ok := val.(interfaces.TransportConnection); ok {
		tmp := make([]byte, maxLength)
		n, e := conn.Read(tmp)
		if e == nil {
			data = tmp[:n]
		}
		err = e
	} else {
		setError(errors.New("invalid handle type for Receive"))
		return -1
	}

	setError(err)
	if err != nil {
		return -1
	}

	if len(data) > int(maxLength) {
		setError(fmt.Errorf("received data exceeds buffer length (%d > %d)", len(data), maxLength))
		return -1
	}

	// Copy data to C buffer
	C.memmove(unsafe.Pointer(buffer), unsafe.Pointer(&data[0]), C.size_t(len(data)))
	return int32(len(data))
}

//export SocketListen
func SocketListen(handle int32) int32 {
	val, ok := Get(handle)
	if !ok {
		setError(errors.New("invalid handle"))
		return -1
	}

	sock, ok := val.(interfaces.Socket)
	if !ok {
		setError(errors.New("handle is not a socket"))
		return -1
	}

	err := sock.Listen()
	setError(err)
	if err != nil {
		return -1
	}
	return 0
}

//export SocketAccept
func SocketAccept(handle int32) int32 {
	val, ok := Get(handle)
	if !ok {
		setError(errors.New("invalid handle"))
		return -1
	}

	sock, ok := val.(interfaces.Socket)
	if !ok {
		setError(errors.New("handle is not a socket"))
		return -1
	}

	conn, err := sock.Accept()
	setError(err)
	if err != nil {
		return -1
	}

	return Register(conn)
}

//export SocketSetDeadline
func SocketSetDeadline(handle int32, seconds C.double) int32 {
	val, ok := Get(handle)
	if !ok {
		setError(errors.New("invalid handle"))
		return -1
	}

	deadline := time.Now().Add(time.Duration(float64(seconds) * float64(time.Second)))

	if sock, ok := val.(interfaces.Socket); ok {
		err := sock.SetDeadline(deadline)
		setError(err)
		if err != nil {
			return -1
		}
		return 0
	}

	if conn, ok := val.(interfaces.TransportConnection); ok {
		err := conn.SetDeadline(deadline)
		setError(err)
		if err != nil {
			return -1
		}
		return 0
	}

	setError(errors.New("invalid handle type for SetDeadline"))
	return -1
}

func main() {}
