// Package randomx provides a pure-Go implementation of the RandomX
// proof-of-work algorithm used by Monero and other cryptocurrencies.
//
// RandomX is designed to be ASIC-resistant through heavy use of random
// code execution, large dataset requirements, and CPU-specific operations.
//
// Example usage:
//
//	config := randomx.Config{
//	    Mode:     randomx.FastMode,
//	    CacheKey: []byte("monero seed"),
//	}
//	hasher, err := randomx.New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer hasher.Close()
//
//	hash := hasher.Hash([]byte("block data"))
package randomx

import (
	"errors"
	"fmt"
	"sync"
)

// Mode represents the RandomX operational mode.
type Mode int

const (
	// LightMode uses ~256 MB of memory and computes dataset items on-the-fly.
	// Suitable for memory-constrained environments but significantly slower.
	LightMode Mode = iota

	// FastMode pre-computes a 2+ GB dataset for maximum hashing performance.
	// Recommended for mining and high-throughput applications.
	FastMode
)

// String returns the string representation of the mode.
func (m Mode) String() string {
	switch m {
	case LightMode:
		return "LightMode"
	case FastMode:
		return "FastMode"
	default:
		return fmt.Sprintf("Mode(%d)", m)
	}
}

// Flags represents CPU feature flags for optimization.
type Flags uint32

const (
	// FlagDefault uses standard optimizations available on all platforms.
	FlagDefault Flags = 0

	// FlagAES indicates hardware AES support (AES-NI on x86).
	FlagAES Flags = 1 << 0

	// Future flags can be added here for additional CPU features.
)

// Config specifies the configuration for a RandomX hasher.
type Config struct {
	// Mode determines memory usage and performance characteristics.
	Mode Mode

	// Flags specifies CPU feature optimizations to enable.
	// Use FlagDefault for automatic detection.
	Flags Flags

	// CacheKey is the seed used to generate the cache and dataset.
	// In Monero, this changes every 2048 blocks (~2.8 days).
	// Must not be nil or empty.
	CacheKey []byte
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if len(c.CacheKey) == 0 {
		return errors.New("randomx: cache key must not be empty")
	}

	if c.Mode != LightMode && c.Mode != FastMode {
		return fmt.Errorf("randomx: invalid mode: %v", c.Mode)
	}

	return nil
}

// Hasher computes RandomX hashes. It is safe for concurrent use.
type Hasher struct {
	config Config
	cache  *cache
	ds     *dataset
	closed bool
	mu     sync.RWMutex // Protects closed flag and cache key updates
}

// New creates a new RandomX hasher with the specified configuration.
// The returned hasher must be closed with Close() to free resources.
func New(config Config) (*Hasher, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	h := &Hasher{
		config: config,
	}

	// Initialize cache
	var err error
	h.cache, err = newCache(config.CacheKey)
	if err != nil {
		return nil, fmt.Errorf("randomx: cache initialization: %w", err)
	}

	// Initialize dataset for fast mode
	if config.Mode == FastMode {
		h.ds, err = newDataset(h.cache)
		if err != nil {
			h.cache.release()
			return nil, fmt.Errorf("randomx: dataset initialization: %w", err)
		}
	}

	return h, nil
}

// Hash computes the RandomX hash of the input data.
// This method is safe for concurrent use by multiple goroutines.
func (h *Hasher) Hash(input []byte) [32]byte {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed {
		panic("randomx: Hash called on closed hasher")
	}

	// Get a VM from the pool
	vm := poolGetVM()
	defer poolPutVM(vm)

	// Initialize VM with the hasher's dataset or cache
	vm.init(h.ds, h.cache)

	// Execute the RandomX hash algorithm
	return vm.run(input)
}

// UpdateCacheKey updates the cache key and regenerates the dataset.
// This is an expensive operation (20-30 seconds for fast mode).
// Returns nil if the new key matches the current key.
//
// On error, the hasher remains in its previous state and can continue
// to be used with the old cache key.
func (h *Hasher) UpdateCacheKey(newKey []byte) error {
	if len(newKey) == 0 {
		return errors.New("randomx: cache key must not be empty")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return errors.New("randomx: UpdateCacheKey called on closed hasher")
	}

	// Check if key actually changed
	if bytesEqual(h.config.CacheKey, newKey) {
		return nil
	}

	// Create new cache first (don't release old resources yet)
	var err error
	newCache, err := newCache(newKey)
	if err != nil {
		// Old cache/dataset still intact, hasher remains usable
		return fmt.Errorf("randomx: cache regeneration: %w", err)
	}

	// Create new dataset for fast mode (if needed)
	var newDS *dataset
	if h.config.Mode == FastMode {
		newDS, err = newDataset(newCache)
		if err != nil {
			// Clean up newly created cache, keep old resources intact
			newCache.release()
			return fmt.Errorf("randomx: dataset regeneration: %w", err)
		}
	}

	// Success! Now safely release old resources and swap in new ones
	if h.ds != nil {
		h.ds.release()
	}
	if h.cache != nil {
		h.cache.release()
	}

	h.cache = newCache
	h.ds = newDS

	// Update stored key
	h.config.CacheKey = append([]byte(nil), newKey...)

	return nil
}

// Close releases all resources held by the hasher.
// After Close, the hasher must not be used.
func (h *Hasher) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil
	}

	h.closed = true

	if h.ds != nil {
		h.ds.release()
		h.ds = nil
	}

	if h.cache != nil {
		h.cache.release()
		h.cache = nil
	}

	return nil
}

// IsReady returns true if the hasher is ready to compute hashes.
func (h *Hasher) IsReady() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return !h.closed
}

// bytesEqual compares two byte slices in constant time.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var equal byte = 0
	for i := 0; i < len(a); i++ {
		equal |= a[i] ^ b[i]
	}
	return equal == 0
}
