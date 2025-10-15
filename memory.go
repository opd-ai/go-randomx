package randomx

import (
	"sync"
	"unsafe"
)

const (
	// Memory alignment for optimal CPU cache performance
	cacheLineSize = 64

	// Scratchpad sizes (in bytes)
	scratchpadL1Size = 16384   // 16 KB
	scratchpadL2Size = 262144  // 256 KB
	scratchpadL3Size = 2097152 // 2 MB
)

// Global pools for memory reuse to minimize allocations

var (
	// VM instance pool
	vmPool = sync.Pool{
		New: func() interface{} {
			return &virtualMachine{
				reg: [8]uint64{},
				mem: allocateScratchpad(),
			}
		},
	}

	// Scratchpad pool for VM memory
	scratchpadPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, scratchpadL3Size)
		},
	}
)

// poolGetVM retrieves a VM instance from the pool.
func poolGetVM() *virtualMachine {
	vm := vmPool.Get().(*virtualMachine)
	vm.reset()
	return vm
}

// poolPutVM returns a VM instance to the pool for reuse.
func poolPutVM(vm *virtualMachine) {
	if vm != nil {
		vmPool.Put(vm)
	}
}

// allocateScratchpad acquires a scratchpad buffer from the pool.
func allocateScratchpad() []byte {
	return scratchpadPool.Get().([]byte)
}

// releaseScratchpad returns a scratchpad buffer to the pool.
func releaseScratchpad(pad []byte) {
	if pad != nil && len(pad) == scratchpadL3Size {
		// Clear sensitive data before returning to pool
		for i := range pad {
			pad[i] = 0
		}
		scratchpadPool.Put(pad)
	}
}

// allocateAlignedDataset allocates a large aligned buffer for dataset storage.
// The dataset is read-only after initialization, so GC scanning is minimal.
func allocateAlignedDataset(size int) []byte {
	// Allocate slightly larger to allow alignment
	buf := make([]byte, size+cacheLineSize)

	// Calculate aligned offset
	offset := cacheLineSize - (int(uintptr(unsafe.Pointer(&buf[0]))) % cacheLineSize)
	if offset == cacheLineSize {
		offset = 0
	}

	// Return aligned slice
	return buf[offset : offset+size]
}

// releaseDataset releases a dataset buffer.
// In Go, we rely on GC, but we can hint that the data is no longer needed.
func releaseDataset(data []byte) {
	// Clear reference to help GC
	// The actual memory will be freed by the garbage collector
	data = nil
}

// copyBytes copies src to dst efficiently.
// This is a helper to avoid import of bytes package.
func copyBytes(dst, src []byte) int {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	for i := 0; i < n; i++ {
		dst[i] = src[i]
	}
	return n
}

// zeroBytes clears a byte slice securely.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
