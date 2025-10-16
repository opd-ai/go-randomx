package internal

import (
	"github.com/opd-ai/go-randomx/internal/argon2d"
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
// This delegates to the proper Argon2d implementation in internal/argon2d
// which provides full data-dependent addressing as required by RandomX.
func Argon2d(password []byte, config Argon2Config) []byte {
	// Convert salt to match argon2d.Argon2d signature
	salt := config.Salt
	if salt == nil {
		salt = []byte{} // Empty salt if none provided
	}

	return argon2d.Argon2d(password, salt, config.Time, config.Memory, uint32(config.Threads), config.OutputLen)
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
