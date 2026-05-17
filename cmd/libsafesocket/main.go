package main

/*
#include <stdlib.h>
#include <string.h>
#include "../../src/cgo_bridge/helpers.h"
*/
import "C"

import (
	"github.com/Bastien-Antigravity/safe-socket/src/cgo_bridge"
	"unsafe"
)

func main() {}

func setError(err error) {
	if err != nil {
		cStr := C.CString(err.Error())
		C.set_socket_error(cStr)
		C.free(unsafe.Pointer(cStr))
	} else {
		C.set_socket_error(nil)
	}
}

//export SafeSocket_FreeString
func SafeSocket_FreeString(ptr *C.char) {
	if ptr != nil {
		C.free(unsafe.Pointer(ptr))
	}
}

//export SafeSocket_GetSocketError
func SafeSocket_GetSocketError() *C.char {
	return C.last_socket_error
}

//export SafeSocket_Create
func SafeSocket_Create(profileName, address, publicIP, socketType *C.char, autoConnect C.int) int32 {
	pName := cgo_bridge.SanitizeString(C.GoString(profileName))
	addr := cgo_bridge.SanitizeString(C.GoString(address))
	pIP := cgo_bridge.SanitizeString(C.GoString(publicIP))
	sType := cgo_bridge.SanitizeString(C.GoString(socketType))
	auto := autoConnect != 0

	handle, err := cgo_bridge.Create(pName, addr, pIP, sType, auto)
	setError(err)
	return handle
}

//export SafeSocket_CreateExtended
func SafeSocket_CreateExtended(profileName, address, publicIP, socketType *C.char, handshakeTimeoutMs, deadlineMs, heartbeatIntervalMs C.int, autoConnect C.int) int32 {
	pName := cgo_bridge.SanitizeString(C.GoString(profileName))
	addr := cgo_bridge.SanitizeString(C.GoString(address))
	pIP := cgo_bridge.SanitizeString(C.GoString(publicIP))
	sType := cgo_bridge.SanitizeString(C.GoString(socketType))
	auto := autoConnect != 0

	handle, err := cgo_bridge.CreateWithConfig(
		pName, addr, pIP,
		int(handshakeTimeoutMs), int(deadlineMs), int(heartbeatIntervalMs),
		sType, auto,
	)
	setError(err)
	return handle
}

//export SafeSocket_Open
func SafeSocket_Open(handle int32) int32 {
	err := cgo_bridge.Open(handle)
	setError(err)
	if err != nil {
		return -1
	}
	return 0
}

//export SafeSocket_Close
func SafeSocket_Close(handle int32) int32 {
	err := cgo_bridge.Close(handle)
	setError(err)
	if err != nil {
		return -1
	}
	return 0
}

//export SafeSocket_Send
func SafeSocket_Send(handle int32, data *C.uchar, length C.int) int32 {
	buf := C.GoBytes(unsafe.Pointer(data), length)
	n, err := cgo_bridge.Send(handle, buf)
	setError(err)
	return n
}

//export SafeSocket_Receive
func SafeSocket_Receive(handle int32, buffer *C.uchar, maxLength C.int) int32 {
	data, err := cgo_bridge.Receive(handle, int(maxLength))
	setError(err)
	if err != nil {
		return -1
	}

	if len(data) > int(maxLength) {
		return -1
	}

	if len(data) > 0 {
		C.memmove(unsafe.Pointer(buffer), unsafe.Pointer(&data[0]), C.size_t(len(data)))
	}
	return int32(len(data))
}

//export SafeSocket_Listen
func SafeSocket_Listen(handle int32) int32 {
	err := cgo_bridge.Listen(handle)
	setError(err)
	if err != nil {
		return -1
	}
	return 0
}

//export SafeSocket_Accept
func SafeSocket_Accept(handle int32) int32 {
	connHandle, err := cgo_bridge.Accept(handle)
	setError(err)
	return connHandle
}

//export SafeSocket_SetIdleTimeout
func SafeSocket_SetIdleTimeout(handle int32, seconds C.double) int32 {
	err := cgo_bridge.SetIdleTimeout(handle, float64(seconds))
	setError(err)
	if err != nil {
		return -1
	}
	return 0
}

//export SafeSocket_SetDeadline
func SafeSocket_SetDeadline(handle int32, seconds C.double) int32 {
	err := cgo_bridge.SetDeadline(handle, float64(seconds))
	setError(err)
	if err != nil {
		return -1
	}
	return 0
}
