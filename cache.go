package randomx

import (
	"fmt"

	"github.com/opd-ai/go-randomx/internal"
)

const (
	// Cache size in bytes (256 KB, contains 32768 64-byte items)
	cacheSize = 262144

	// Number of cache items
	cacheItems = cacheSize / 64
)

// cache holds the RandomX cache initialized from a seed using Argon2.
// The cache is used to generate dataset items in light mode or to
// initialize the full dataset in fast mode.
type cache struct {
	data []byte // Raw cache data (256 KB)
	key  []byte // Cache key (seed) used to generate this cache
}

// newCache creates a new RandomX cache from the given seed.
func newCache(seed []byte) (*cache, error) {
	if len(seed) == 0 {
		return nil, fmt.Errorf("cache seed must not be empty")
	}

	c := &cache{
		key:  append([]byte(nil), seed...), // Copy seed
		data: make([]byte, cacheSize),
	}

	// Generate cache using Argon2d
	cacheData := internal.Argon2dCache(seed)
	if len(cacheData) != cacheSize {
		return nil, fmt.Errorf("argon2 output size mismatch: got %d, want %d",
			len(cacheData), cacheSize)
	}

	copy(c.data, cacheData)

	return c, nil
}

// release frees the cache resources.
func (c *cache) release() {
	if c.data != nil {
		zeroBytes(c.data)
		c.data = nil
	}
	c.key = nil
}

// getItem returns the cache item at the specified index.
// Each item is 64 bytes.
func (c *cache) getItem(index uint32) []byte {
	if index >= cacheItems {
		index = index % cacheItems
	}
	offset := index * 64
	return c.data[offset : offset+64]
}
