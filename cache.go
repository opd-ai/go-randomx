package randomx

import (
	"fmt"

	"github.com/opd-ai/go-randomx/internal"
)

const (
	// Cache size in bytes (256 MB = 262144 blocks * 1024 bytes)
	// RandomX uses the entire Argon2d memory as the cache
	cacheSize = 262144 * 1024

	// Number of cache items (each item is 64 bytes)
	cacheItems = cacheSize / 64
)

// cache holds the RandomX cache initialized from a seed using Argon2.
// The cache is used to generate dataset items in light mode or to
// initialize the full dataset in fast mode.
type cache struct {
	data        []byte                 // Raw cache data (256 MB)
	key         []byte                 // Cache key (seed) used to generate this cache
	programs    []*superscalarProgram  // Superscalar programs for dataset generation (8 programs)
	reciprocals []uint64               // Pre-computed reciprocals for IMUL_RCP instructions
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

	// Generate superscalar programs for dataset item generation
	gen := newBlake2Generator(seed)
	c.programs = make([]*superscalarProgram, cacheAccesses)
	
	for i := 0; i < cacheAccesses; i++ {
		c.programs[i] = generateSuperscalarProgram(gen)
		
		// Pre-compute reciprocals for IMUL_RCP instructions in this program
		for j := range c.programs[i].instructions {
			instr := &c.programs[i].instructions[j]
			if instr.opcode == ssIMUL_RCP {
				// Store the reciprocal value and update imm32 to point to it
				rcp := reciprocal(instr.imm32)
				instr.imm32 = uint32(len(c.reciprocals))
				c.reciprocals = append(c.reciprocals, rcp)
			}
		}
	}

	return c, nil
}

// release frees the cache resources.
func (c *cache) release() {
	if c.data != nil {
		zeroBytes(c.data)
		c.data = nil
	}
	c.key = nil
	c.programs = nil
	c.reciprocals = nil
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
