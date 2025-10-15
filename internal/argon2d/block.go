package argon2d

import (
	"encoding/binary"
)

// Block size constants from Argon2 specification
const (
	// BlockSize is the size of an Argon2 memory block in bytes (1024 bytes = 1 KB)
	BlockSize = 1024

	// QWordsInBlock is the number of 64-bit words (uint64) in a block (1024 / 8 = 128)
	QWordsInBlock = 128
)

// Block represents a 1024-byte Argon2 memory block as an array of 128 uint64 values.
// This structure is used throughout the Argon2d algorithm for memory operations.
//
// Memory layout: [uint64 x 128] = 1024 bytes
//
// Why uint64 array instead of byte array:
//   - Argon2 performs operations on 64-bit words, not bytes
//   - uint64 operations are more efficient than byte-level operations
//   - Simplifies the mixing and compression functions
//
// Reference: Argon2 spec section 3.3 (Memory Block Structure)
type Block [QWordsInBlock]uint64

// XOR performs in-place XOR of this block with another block.
// This modifies the current block: b[i] = b[i] XOR other[i] for all i.
//
// XOR is a fundamental operation in Argon2d used during:
//   - Block compression (mixing reference blocks)
//   - Final hash computation (XORing all blocks)
//
// Parameters:
//   - other: The block to XOR with
//
// Performance note: This is a hot path function called many times during
// Argon2d execution. The simple loop is efficient and easily optimized
// by the Go compiler.
func (b *Block) XOR(other *Block) {
	for i := range b {
		b[i] ^= other[i]
	}
}

// Copy copies data from another block into this block.
// This is equivalent to: b[i] = other[i] for all i.
//
// Used when creating temporary copies of blocks during compression
// to avoid modifying the original data.
//
// Parameters:
//   - other: The source block to copy from
//
// Implementation note: Uses Go's built-in copy() which is highly optimized
// and may use SIMD instructions on supported platforms.
func (b *Block) Copy(other *Block) {
	copy(b[:], other[:])
}

// Zero clears all data in the block by setting every uint64 to 0.
//
// Security note: This is used to securely erase sensitive data from memory
// after use. While Go doesn't guarantee the compiler won't optimize this away,
// writing zeros is standard practice for cleaning up cryptographic material.
//
// Used when:
//   - Releasing blocks back to a pool
//   - Securely clearing intermediate computation results
//   - Preparing blocks for reuse
func (b *Block) Zero() {
	for i := range b {
		b[i] = 0
	}
}

// FromBytes loads a block from a byte slice.
// The input must be exactly BlockSize (1024) bytes and is interpreted as
// 128 little-endian uint64 values.
//
// Parameters:
//   - data: Input byte slice (must be 1024 bytes)
//
// Returns:
//   - error if data length is not exactly BlockSize
//
// Memory layout: Little-endian encoding is used per Argon2 specification.
// Bytes [0:7] become b[0], bytes [8:15] become b[1], etc.
func (b *Block) FromBytes(data []byte) error {
	if len(data) != BlockSize {
		return &InvalidBlockSizeError{got: len(data), want: BlockSize}
	}

	for i := 0; i < QWordsInBlock; i++ {
		b[i] = binary.LittleEndian.Uint64(data[i*8 : (i+1)*8])
	}

	return nil
}

// ToBytes converts the block to a byte slice.
// Returns a new 1024-byte slice containing the block data encoded as
// little-endian uint64 values.
//
// Returns:
//   - Byte slice of length BlockSize (1024 bytes)
//
// Memory layout: Each uint64 is encoded as 8 bytes in little-endian order.
// b[0] becomes bytes [0:7], b[1] becomes bytes [8:15], etc.
func (b *Block) ToBytes() []byte {
	data := make([]byte, BlockSize)
	for i := 0; i < QWordsInBlock; i++ {
		binary.LittleEndian.PutUint64(data[i*8:(i+1)*8], b[i])
	}
	return data
}

// InvalidBlockSizeError is returned when attempting to load a block from
// a byte slice that is not exactly BlockSize bytes.
type InvalidBlockSizeError struct {
	got  int
	want int
}

func (e *InvalidBlockSizeError) Error() string {
	return "invalid block size: got " + itoa(e.got) + " bytes, want " + itoa(e.want) + " bytes"
}

// itoa is a simple integer to string converter to avoid fmt import
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	buf := make([]byte, 0, 12) // enough for 32-bit int
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}

	if negative {
		buf = append(buf, '-')
	}

	// Reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}

	return string(buf)
}
