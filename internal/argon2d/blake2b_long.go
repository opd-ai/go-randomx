// Package argon2d implements the Argon2d (data-dependent) variant
// required by RandomX. This is a pure-Go port from the RandomX C
// implementation.
//
// Argon2d differs from Argon2i/Argon2id (available in golang.org/x/crypto/argon2)
// in that it uses data-dependent memory access patterns, making it faster
// but vulnerable to side-channel attacks. RandomX specifically requires
// Argon2d for cache generation.
package argon2d

import (
	"encoding/binary"

	"golang.org/x/crypto/blake2b"
)

// Blake2bLong generates variable-length output from Blake2b hash function.
// This implements the Argon2 specification for producing outputs longer
// than Blake2b's native 64-byte maximum.
//
// Algorithm from Argon2 spec section 3.1:
//   - If outlen <= 64 bytes: return Blake2b(input, outlen)
//   - If outlen > 64 bytes:
//     1. V₁ = Blake2b(input || uint32_le(outlen), 64)
//     2. result = V₁[0:32]
//     3. For remaining blocks: Vᵢ = Blake2b(Vᵢ₋₁, 64)
//     4. Append appropriate bytes from each Vᵢ to result
//
// Parameters:
//   - input: Input data to hash
//   - outlen: Desired output length in bytes
//
// Returns:
//   - Output hash of exactly outlen bytes
//
// Reference: https://github.com/P-H-C/phc-winner-argon2/blob/master/argon2-specs.pdf
func Blake2bLong(input []byte, outlen uint32) []byte {
	if outlen == 0 {
		return nil
	}

	// Simple case: output fits in a single Blake2b hash
	if outlen <= 64 {
		h, err := blake2b.New(int(outlen), nil)
		if err != nil {
			// This should never happen for valid output lengths (1-64)
			panic("blake2b.New failed with valid length: " + err.Error())
		}
		h.Write(input)
		return h.Sum(nil)
	}

	// Extended output case: Chain Blake2b hashes together
	//
	// Create initial hash V₁ = Blake2b(input || uint32_le(outlen), 64)
	// The output length is prepended as a 4-byte little-endian value
	// as specified in the Argon2 spec.
	output := make([]byte, outlen)

	// Prepare input with length prefix
	inputWithLen := make([]byte, len(input)+4)
	binary.LittleEndian.PutUint32(inputWithLen, outlen)
	copy(inputWithLen[4:], input)

	// Generate first 64-byte block
	h, _ := blake2b.New512(nil) // 512 bits = 64 bytes
	h.Write(inputWithLen)
	v := h.Sum(nil)

	// Copy first 32 bytes to output (Argon2 spec uses first half)
	copied := copy(output, v[:32])

	// Generate remaining blocks by repeatedly hashing the previous block
	// Each iteration produces 64 bytes, we take what we need
	for copied < int(outlen) {
		h.Reset()
		h.Write(v)
		v = h.Sum(nil)

		// Copy up to 64 bytes or whatever remains
		n := copy(output[copied:], v)
		copied += n
	}

	return output
}
