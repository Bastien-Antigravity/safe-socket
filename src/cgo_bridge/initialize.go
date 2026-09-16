package cgo_bridge

// =============================================================================
// ESSENTIAL PROCESS:
// Manages opaque handle registration and object lifecycle for the CGO bridge,
// mapping integer handles to Go Socket and Connection instances across language barriers.
//
// DATA FLOW:
// 1. Input: Arbitrary Go socket/connection objects.
// 2. Logic: Assigns thread-safe atomic handle IDs in Registry sync.Map.
// 3. Output: 32-bit integer handles passed across C ABI to Python/Rust callers.
//
// KEY PARAMETERS:
// - Register: Allocates and registers an opaque handle.
// - Unregister: Frees handle from memory registry.
// =============================================================================

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
