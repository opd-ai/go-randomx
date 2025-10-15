package internal

import (
	"golang.org/x/crypto/argon2"
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
func Argon2d(password []byte, config Argon2Config) []byte {
	// RandomX uses Argon2d (data-dependent mode)
	return argon2.Key(
		password,
		config.Salt,
		config.Time,
		config.Memory,
		config.Threads,
		config.OutputLen,
	)
}

// Argon2dCache generates RandomX cache using Argon2d.
func Argon2dCache(seed []byte) []byte {
	// RandomX cache generation parameters
	const (
		argonTime    = 3
		argonMemory  = 262144 // 256 MB
		argonThreads = 1
		cacheSize    = 262144 // 256 KB
	)

	// Use "RandomX\x03" as salt (RandomX v1.1.x)
	salt := []byte("RandomX\x03")

	return argon2.Key(seed, salt, argonTime, argonMemory, argonThreads, cacheSize)
}
