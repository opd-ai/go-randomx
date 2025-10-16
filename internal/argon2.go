package internal

import (
	"encoding/binary"

	"github.com/opd-ai/go-randomx/internal/argon2d"
	"golang.org/x/crypto/blake2b"
)

// Argon2Config specifies Argon2 parameters for RandomX.
type Argon2Config struct {
	Time      uint32 // Number of iterations
	Memory    uint32 // Memory in KB
	Threads   uint8  // Parallelism factor
	OutputLen uint32 // Output length in bytes
	Salt      []byte // Salt value
}

// DefaultRandomXArgon2Config returns the Argon2 configuration used by RandomX.
func DefaultRandomXArgon2Config(salt []byte) Argon2Config {
	return Argon2Config{
		Time:      3,      // 3 iterations
		Memory:    262144, // 256 MB
		Threads:   1,      // Single-threaded
		OutputLen: 262144, // 256 KB output
		Salt:      salt,
	}
}

// Argon2d computes Argon2d hash (data-dependent, used by RandomX).
//
// WARNING: This is currently using a PLACEHOLDER implementation that does NOT
// produce correct Argon2d output. See file header for details.
func Argon2d(password []byte, config Argon2Config) []byte {
	// TODO: Implement proper Argon2d
	// Current placeholder uses simple Blake2b-based derivation
	// This is WRONG but allows development to continue on other components

	return argon2dPlaceholder(password, config.Salt, config.Time, config.Memory, config.OutputLen)
}

// Argon2dCache generates the RandomX cache using proper Argon2d.
// This uses the custom Argon2d implementation in internal/argon2d which
// provides full data-dependent addressing as required by RandomX.
//
// RandomX parameters:
//   - Memory: 256 MB (262144 KB)
//   - Time: 3 passes
//   - Lanes: 1 (single-threaded)
//   - Output: 256 KB cache
//
// The key is used as both password and salt, following RandomX specification.
func Argon2dCache(key []byte) []byte {
	return argon2d.Argon2dCache(key)
}

// argon2dPlaceholder is a TEMPORARY placeholder for Argon2d.
// This provides deterministic output for testing but does NOT match Argon2d spec.
//
// This function implements a simplified memory-hard function using Blake2b
// to allow development of other components while proper Argon2d is being implemented.
/*func argon2dPlaceholder(password, salt []byte, time, memory, keyLen uint32) []byte {
	// Create initial Blake2b hash of password + salt
	h, _ := blake2b.New512(nil)
	h.Write(password)
	h.Write(salt)
	h.Write([]byte{byte(time), byte(memory >> 24), byte(memory >> 16), byte(memory >> 8), byte(memory)})
	initialHash := h.Sum(nil)

	// Simulate memory-hard function with iterated hashing
	// This is NOT cryptographically equivalent to Argon2d
	current := make([]byte, 64)
	copy(current, initialHash)

	// Simplified "memory-hard" iterations
	iterations := int(time) * 1000 // Scale iterations
	for i := 0; i < iterations; i++ {
		h.Reset()
		h.Write(current)
		binary.LittleEndian.PutUint64(current[:8], uint64(i))
		current = h.Sum(current[:0])
	}

	// Expand to desired output length using Blake2b in counter mode
	output := make([]byte, keyLen)
	h.Reset()
	h.Write(current)
	block := h.Sum(nil)

	for i := uint32(0); i < keyLen; i += 64 {
		copy(output[i:], block)
		if i+64 < keyLen {
			h.Reset()
			h.Write(block)
			binary.LittleEndian.PutUint64(block[:8], uint64(i/64)+1)
			block = h.Sum(block[:0])
		}
	}

	return output[:keyLen]
}*/
