// Package argon2d implements Argon2d (data-dependent mode) for RandomX.
// This file contains the public API and initialization functions.
package argon2d

import (
	"encoding/binary"

	"golang.org/x/crypto/blake2b"
)

const (
	// Argon2Version is the version number (0x13 = 19 decimal)
	Argon2Version = 0x13

	// Argon2d type identifier (0 = data-dependent)
	Argon2TypeD = 0

	// DefaultTagLength is the output hash length in bytes (32 for RandomX)
	DefaultTagLength = 32
)

// initialHash computes H0, the initial hash for Argon2d.
// This hash serves as the seed for initializing the first two blocks
// and establishes the initial state for memory filling.
//
// H0 = Blake2b(lanes, tagLength, memory, timeCost, version, type,
//
//	len(password), password, len(salt), salt,
//	len(secret), secret, len(data), data)
//
// All multi-byte integers are encoded as little-endian uint32.
//
// Parameters:
//   - lanes: Number of parallel lanes (1 for RandomX)
//   - tagLength: Output hash length in bytes (32 for RandomX)
//   - memory: Memory size in KB (256*1024 KB = 256 MB for RandomX)
//   - timeCost: Number of passes (3 for RandomX)
//   - password: Input key/password
//   - salt: Salt value
//   - secret: Optional secret key (nil for RandomX)
//   - data: Optional associated data (nil for RandomX)
//
// Returns: H0 as 64-byte Blake2b hash
func initialHash(lanes, tagLength, memory, timeCost uint32,
	password, salt, secret, data []byte) [64]byte {

	// Compute total input size for Blake2b
	// Format: 10 uint32 values + variable-length fields
	inputSize := 10*4 + len(password) + len(salt) + len(secret) + len(data)
	input := make([]byte, inputSize)

	offset := 0

	// Write fixed parameters as little-endian uint32
	binary.LittleEndian.PutUint32(input[offset:], lanes)
	offset += 4

	binary.LittleEndian.PutUint32(input[offset:], tagLength)
	offset += 4

	binary.LittleEndian.PutUint32(input[offset:], memory)
	offset += 4

	binary.LittleEndian.PutUint32(input[offset:], timeCost)
	offset += 4

	binary.LittleEndian.PutUint32(input[offset:], Argon2Version)
	offset += 4

	binary.LittleEndian.PutUint32(input[offset:], Argon2TypeD)
	offset += 4

	// Write password with length prefix
	binary.LittleEndian.PutUint32(input[offset:], uint32(len(password)))
	offset += 4
	copy(input[offset:], password)
	offset += len(password)

	// Write salt with length prefix
	binary.LittleEndian.PutUint32(input[offset:], uint32(len(salt)))
	offset += 4
	copy(input[offset:], salt)
	offset += len(salt)

	// Write secret with length prefix (may be empty)
	binary.LittleEndian.PutUint32(input[offset:], uint32(len(secret)))
	offset += 4
	if len(secret) > 0 {
		copy(input[offset:], secret)
		offset += len(secret)
	}

	// Write associated data with length prefix (may be empty)
	binary.LittleEndian.PutUint32(input[offset:], uint32(len(data)))
	offset += 4
	if len(data) > 0 {
		copy(input[offset:], data)
		// offset += len(data) // Not needed, this is the last field
	}

	// Compute Blake2b-512 hash (64 bytes)
	return blake2b.Sum512(input)
}

// initializeMemory fills the first two blocks of each lane from H0.
// Each block is generated using Blake2bLong with H0 as input plus
// block index and lane index.
//
// For each lane i:
//
//	Block[i][0] = Blake2bLong(H0 || 0 || i, 1024)
//	Block[i][1] = Blake2bLong(H0 || 1 || i, 1024)
//
// Parameters:
//   - memory: Pre-allocated memory blocks to initialize
//   - lanes: Number of parallel lanes
//   - h0: Initial hash (64 bytes) from initialHash()
func initializeMemory(memory []Block, lanes uint32, h0 [64]byte) {
	laneLength := uint32(len(memory)) / lanes

	for lane := uint32(0); lane < lanes; lane++ {
		// Prepare input for Blake2bLong: H0 || blockIndex || laneIndex
		// blockIndex and laneIndex are uint32 little-endian
		input := make([]byte, 72) // 64 + 4 + 4
		copy(input[0:64], h0[:])

		// Initialize block 0 of this lane
		binary.LittleEndian.PutUint32(input[64:68], 0) // block index 0
		binary.LittleEndian.PutUint32(input[68:72], lane)
		block0Bytes := Blake2bLong(input, 1024)
		memory[lane*laneLength].FromBytes(block0Bytes)

		// Initialize block 1 of this lane
		binary.LittleEndian.PutUint32(input[64:68], 1) // block index 1
		// lane index stays the same
		block1Bytes := Blake2bLong(input, 1024)
		memory[lane*laneLength+1].FromBytes(block1Bytes)
	}
}

// finalizeHash computes the final hash by XORing all blocks in each lane.
// For single-lane Argon2 (used by RandomX), this XORs all blocks together
// and then applies Blake2b to produce the final output.
//
// Algorithm per Argon2 specification:
//  1. For each lane, XOR all blocks: C = B[0] XOR B[1] XOR ... XOR B[n-1]
//  2. Final = Blake2b(C, tagLength)
//
// Parameters:
//   - memory: Memory blocks after fillMemory
//   - lanes: Number of parallel lanes
//   - tagLength: Desired output length in bytes
//
// Returns: Final hash output
func finalizeHash(memory []Block, lanes, tagLength uint32) []byte {
	laneLength := uint32(len(memory)) / lanes

	// XOR all blocks in the first lane (RandomX uses lanes=1)
	// For multi-lane, would need to XOR across lanes
	var finalBlock Block
	finalBlock = memory[0] // Start with block 0

	// XOR with all remaining blocks in lane 0
	for i := uint32(1); i < laneLength; i++ {
		finalBlock.XOR(&memory[i])
	}

	// Convert final block to bytes
	finalBlockBytes := finalBlock.ToBytes()

	// Apply Blake2b to produce final hash of desired length
	result := Blake2bLong(finalBlockBytes, tagLength)

	return result
}

// Argon2d computes the Argon2d hash (data-dependent mode).
// This is the main entry point that orchestrates the entire algorithm.
//
// Algorithm:
//  1. Compute H0 = initialHash(parameters, password, salt)
//  2. Initialize first two blocks from H0
//  3. Fill remaining blocks using data-dependent addressing
//  4. Finalize by XORing all blocks and applying Blake2b
//
// Parameters:
//   - password: Input key/password
//   - salt: Salt value (should be random, at least 8 bytes)
//   - timeCost: Number of passes over memory (3 for RandomX)
//   - memorySizeKB: Memory size in kilobytes (262144 = 256 MB for RandomX)
//   - lanes: Number of parallel lanes (1 for RandomX)
//   - tagLength: Output hash length in bytes (32 for RandomX)
//
// Returns: Argon2d hash output
//
// For RandomX:
//
//	Argon2d(key, salt, 3, 262144, 1, 32)
func Argon2d(password, salt []byte, timeCost, memorySizeKB, lanes, tagLength uint32) []byte {
	// Step 1: Compute H0
	h0 := initialHash(lanes, tagLength, memorySizeKB, timeCost, password, salt, nil, nil)

	// Step 2: Allocate memory
	// Each block is 1024 bytes, so number of blocks = memorySizeKB
	numBlocks := memorySizeKB
	memory := make([]Block, numBlocks)

	// Step 3: Initialize first two blocks of each lane from H0
	initializeMemory(memory, lanes, h0)

	// Step 4: Fill memory using data-dependent addressing
	// segmentLength is calculated internally as laneLength / SyncPoints
	fillMemory(memory, timeCost, lanes)

	// Step 5: Finalize hash
	result := finalizeHash(memory, lanes, tagLength)

	return result
}

// Argon2dCache generates a RandomX cache using Argon2d.
// This is a convenience wrapper for RandomX-specific parameters.
//
// RandomX uses:
//   - Memory: 256 MB (262144 KB)
//   - Time cost: 3 passes
//   - Lanes: 1 (single-threaded)
//   - Tag length: 256 KB output (to be interpreted as blocks)
//
// The output is 256 KB of data representing the RandomX cache.
//
// RandomX uses the key as both password AND salt (not a separate fixed salt).
// This is documented in the RandomX specification and confirmed by the reference
// C++ implementation.
func Argon2dCache(key []byte) []byte {
	const (
		memorySizeKB = 262144 // 256 MB
		timeCost     = 3      // 3 passes
		lanes        = 1      // Single-threaded
		cacheSize    = 262144 // 256 KB cache output
	)

	// RandomX uses the key as both password and salt
	// This matches the RandomX C++ reference implementation
	return Argon2d(key, key, timeCost, memorySizeKB, lanes, cacheSize)
}
