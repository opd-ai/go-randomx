package argon2d

// g implements the Blake2b G mixing function used in Argon2 compression.
//
// The G function is the core mixing operation from Blake2b, adapted for Argon2.
// It takes four uint64 values and applies a series of additions, XORs, and
// rotations to thoroughly mix the values.
//
// This function is called during block compression (fill_block) to provide
// cryptographic diffusion across the 1024-byte blocks.
//
// Reference: Blake2b specification Section 3.2
// Reference: Argon2 specification Section 3.3
func g(a, b, c, d uint64) (uint64, uint64, uint64, uint64) {
	// First round: addition and rotation by 32
	a = a + b
	d = rotr64(d^a, 32)
	c = c + d
	b = rotr64(b^c, 24)

	// Second round: addition and rotation by 16
	a = a + b
	d = rotr64(d^a, 16)
	c = c + d
	b = rotr64(b^c, 63)

	return a, b, c, d
}

// rotr64 performs a right rotation of x by n bits.
//
// This is a constant-time operation that doesn't depend on the rotation amount
// being secret, making it safe for cryptographic use.
//
// Example: rotr64(0x123456789ABCDEF0, 8) rotates right by 8 bits
func rotr64(x uint64, n uint) uint64 {
	return (x >> n) | (x << (64 - n))
}

// gRound applies the G function to a 16-element slice in a specific pattern.
//
// This implements one round of Blake2b's compression function, applying G to
// columns and then diagonals. The pattern matches the Blake2b specification
// exactly, ensuring compatibility with the reference Argon2 implementation.
//
// The function operates in-place, modifying the v slice directly.
//
// Reference: Blake2b specification Section 3.2
func gRound(v []uint64) {
	// Column step: apply G function to each column
	v[0], v[4], v[8], v[12] = g(v[0], v[4], v[8], v[12])
	v[1], v[5], v[9], v[13] = g(v[1], v[5], v[9], v[13])
	v[2], v[6], v[10], v[14] = g(v[2], v[6], v[10], v[14])
	v[3], v[7], v[11], v[15] = g(v[3], v[7], v[11], v[15])

	// Diagonal step: apply G function to diagonals
	v[0], v[5], v[10], v[15] = g(v[0], v[5], v[10], v[15])
	v[1], v[6], v[11], v[12] = g(v[1], v[6], v[11], v[12])
	v[2], v[7], v[8], v[13] = g(v[2], v[7], v[8], v[13])
	v[3], v[4], v[9], v[14] = g(v[3], v[4], v[9], v[14])
}
