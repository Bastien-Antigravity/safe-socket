package cgo_bridge

/*
#include "helpers.h"
*/
import "C"

import (
	"sync"
)

var (
	Registry    sync.Map
	NextHandle  int32
	RegistryMut sync.Mutex
)

// Register stores a socket/connection and returns a handle.
func Register(val interface{}) int32 {
	RegistryMut.Lock()
	defer RegistryMut.Unlock()
	NextHandle++
	Registry.Store(NextHandle, val)
	return NextHandle
}

// Get retrieves a value from the registry.
func Get(handle int32) (interface{}, bool) {
	return Registry.Load(handle)
}

// Unregister removes a handle from the registry.
func Unregister(handle int32) {
	Registry.Delete(handle)
}
