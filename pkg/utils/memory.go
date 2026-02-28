package utils

import (
	"runtime"
	"syscall"
)

// ZeroMemory clears a byte slice by overwriting its contents with zeros.
func ZeroMemory(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

// LockMemory prevents a byte slice from being swapped to disk on Linux systems.
// This is critical for storing decrypted private keys in memory.
func LockMemory(data []byte) error {
	// mlock prevents memory from being paged to the swap area.
	return syscall.Mlock(data)
}

// UnlockMemory allows a byte slice to be swapped to disk again.
func UnlockMemory(data []byte) error {
	return syscall.Munlock(data)
}

// ForceGC triggers an immediate garbage collection to clear transient memory artifacts.
func ForceGC() {
	runtime.GC()
}
