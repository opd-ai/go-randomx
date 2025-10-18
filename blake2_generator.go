package randomx

import (
	"github.com/opd-ai/go-randomx/internal"
)

// blake2Generator is a deterministic pseudo-random number generator
// based on Blake2b. It's used to generate superscalar programs.
//
// The generator maintains a 64-byte state that is repeatedly hashed
// with Blake2b to produce a stream of pseudo-random bytes.
type blake2Generator struct {
	data [64]byte // Current Blake2b-512 output
	pos  int      // Position in current output (0-63)
}

// newBlake2Generator creates a new Blake2Generator initialized with a seed.
// The seed is hashed with Blake2b-512 to create the initial state.
func newBlake2Generator(seed []byte) *blake2Generator {
	g := &blake2Generator{
		pos: 64, // Force initial generation
	}
	
	// Hash the seed to get initial state
	hash := internal.Blake2b512(seed)
	copy(g.data[:], hash[:])
	
	return g
}

// generate produces the next 64 bytes of pseudo-random data.
// This is called automatically when the current buffer is exhausted.
func (g *blake2Generator) generate() {
	// Hash the current state to get the next state
	hash := internal.Blake2b512(g.data[:])
	copy(g.data[:], hash[:])
	g.pos = 0
}

// getByte returns the next pseudo-random byte.
func (g *blake2Generator) getByte() byte {
	if g.pos >= 64 {
		g.generate()
	}
	b := g.data[g.pos]
	g.pos++
	return b
}

// getUint32 returns the next pseudo-random uint32 in little-endian format.
func (g *blake2Generator) getUint32() uint32 {
	// Get 4 bytes and combine into uint32
	b0 := uint32(g.getByte())
	b1 := uint32(g.getByte())
	b2 := uint32(g.getByte())
	b3 := uint32(g.getByte())
	
	return b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)
}
