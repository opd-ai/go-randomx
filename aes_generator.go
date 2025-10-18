package randomx

import (
	"crypto/aes"
	"crypto/cipher"
)

// AES round keys from RandomX specification

// AesGenerator1R keys - generated from Hash512("RandomX AesGenerator1R keys")
var aesGenerator1RKeys = [4][16]byte{
	{0x53, 0xa5, 0xac, 0x6d, 0x09, 0x66, 0x71, 0x62, 0x2b, 0x55, 0xb5, 0xdb, 0x17, 0x49, 0xf4, 0xb4},
	{0x07, 0xaf, 0x7c, 0x6d, 0x0d, 0x71, 0x6a, 0x84, 0x78, 0xd3, 0x25, 0x17, 0x4e, 0xdc, 0xa1, 0x0d},
	{0xf1, 0x62, 0x12, 0x3f, 0xc6, 0x7e, 0x94, 0x9f, 0x4f, 0x79, 0xc0, 0xf4, 0x45, 0xe3, 0x20, 0x3e},
	{0x35, 0x81, 0xef, 0x6a, 0x7c, 0x31, 0xba, 0xb1, 0x88, 0x4c, 0x31, 0x16, 0x54, 0x91, 0x16, 0x49},
}

// AesGenerator4R keys - generated from Hash512("RandomX AesGenerator4R keys 0-3") and Hash512("RandomX AesGenerator4R keys 4-7")
var aesGenerator4RKeys = [8][16]byte{
	{0xdd, 0xaa, 0x21, 0x64, 0xdb, 0x3d, 0x83, 0xd1, 0x2b, 0x6d, 0x54, 0x2f, 0x3f, 0xd2, 0xe5, 0x99},
	{0x50, 0x34, 0x0e, 0xb2, 0x55, 0x3f, 0x91, 0xb6, 0x53, 0x9d, 0xf7, 0x06, 0xe5, 0xcd, 0xdf, 0xa5},
	{0x04, 0xd9, 0x3e, 0x5c, 0xaf, 0x7b, 0x5e, 0x51, 0x9f, 0x67, 0xa4, 0x0a, 0xbf, 0x02, 0x1c, 0x17},
	{0x63, 0x37, 0x62, 0x85, 0x08, 0x5d, 0x8f, 0xe7, 0x85, 0x37, 0x67, 0xcd, 0x91, 0xd2, 0xde, 0xd8},
	{0x73, 0x6f, 0x82, 0xb5, 0xa6, 0xa7, 0xd6, 0xe3, 0x6d, 0x8b, 0x51, 0x3d, 0xb4, 0xff, 0x9e, 0x22},
	{0xf3, 0x6b, 0x56, 0xc7, 0xd9, 0xb3, 0x10, 0x9c, 0x4e, 0x4d, 0x02, 0xe9, 0xd2, 0xb7, 0x72, 0xb2},
	{0xe7, 0xc9, 0x73, 0xf2, 0x8b, 0xa3, 0x65, 0xf7, 0x0a, 0x66, 0xa9, 0x2b, 0xa7, 0xef, 0x3b, 0xf6},
	{0x09, 0xd6, 0x7c, 0x7a, 0xde, 0x39, 0x58, 0x91, 0xfd, 0xd1, 0x06, 0x0c, 0x2d, 0x76, 0xb0, 0xc0},
}

// aesGenerator1R implements the RandomX AesGenerator1R pseudo-random number generator.
// It produces a sequence of pseudo-random bytes using AES encryption/decryption.
type aesGenerator1R struct {
	state [64]byte // 4 columns of 16 bytes each
	enc   [2]cipher.Block
	dec   [2]cipher.Block
	pos   int // Position in current state (0-63)
}

// newAesGenerator1R creates a new AesGenerator1R initialized with a 64-byte seed.
func newAesGenerator1R(seed []byte) (*aesGenerator1R, error) {
	if len(seed) != 64 {
		panic("aesGenerator1R: seed must be 64 bytes")
	}

	gen := &aesGenerator1R{}
	copy(gen.state[:], seed)

	// Initialize AES cipher blocks for encryption keys (columns 1, 3)
	var err error
	gen.enc[0], err = aes.NewCipher(aesGenerator1RKeys[1][:])
	if err != nil {
		return nil, err
	}
	gen.enc[1], err = aes.NewCipher(aesGenerator1RKeys[3][:])
	if err != nil {
		return nil, err
	}

	// Initialize AES cipher blocks for decryption keys (columns 0, 2)
	gen.dec[0], err = aes.NewCipher(aesGenerator1RKeys[0][:])
	if err != nil {
		return nil, err
	}
	gen.dec[1], err = aes.NewCipher(aesGenerator1RKeys[2][:])
	if err != nil {
		return nil, err
	}

	gen.pos = 64 // Force initial generation
	return gen, nil
}

// generate produces the next 64 bytes of pseudo-random data.
func (g *aesGenerator1R) generate() {
	// Create new state by applying AES to each column
	var newState [64]byte

	// Column 0 (decrypt with key0)
	g.dec[0].Decrypt(newState[0:16], g.state[0:16])

	// Column 1 (encrypt with key1)
	g.enc[0].Encrypt(newState[16:32], g.state[16:32])

	// Column 2 (decrypt with key2)
	g.dec[1].Decrypt(newState[32:48], g.state[32:48])

	// Column 3 (encrypt with key3)
	g.enc[1].Encrypt(newState[48:64], g.state[48:64])

	g.state = newState
	g.pos = 0
}

// getByte returns the next pseudo-random byte.
func (g *aesGenerator1R) getByte() byte {
	if g.pos >= 64 {
		g.generate()
	}
	b := g.state[g.pos]
	g.pos++
	return b
}

// getBytes fills the provided slice with pseudo-random bytes.
func (g *aesGenerator1R) getBytes(dst []byte) {
	for i := range dst {
		dst[i] = g.getByte()
	}
}

// getUint32 returns the next pseudo-random uint32.
func (g *aesGenerator1R) getUint32() uint32 {
	if g.pos+4 > 64 {
		g.generate()
	}
	val := uint32(g.state[g.pos]) |
		uint32(g.state[g.pos+1])<<8 |
		uint32(g.state[g.pos+2])<<16 |
		uint32(g.state[g.pos+3])<<24
	g.pos += 4
	return val
}

// aesGenerator4R implements the RandomX AesGenerator4R pseudo-random number generator.
// Similar to AesGenerator1R but uses 4 AES rounds per column for higher security.
type aesGenerator4R struct {
	state [64]byte // 4 columns of 16 bytes each
	// Cipher blocks for keys 0-7
	enc03 [4]cipher.Block // Encryption with keys 0-3
	dec03 [4]cipher.Block // Decryption with keys 0-3
	enc47 [4]cipher.Block // Encryption with keys 4-7
	dec47 [4]cipher.Block // Decryption with keys 4-7
	pos   int              // Position in current state (0-63)
}

// newAesGenerator4R creates a new AesGenerator4R initialized with a 64-byte seed.
func newAesGenerator4R(seed []byte) (*aesGenerator4R, error) {
	if len(seed) != 64 {
		panic("aesGenerator4R: seed must be 64 bytes")
	}

	gen := &aesGenerator4R{}
	copy(gen.state[:], seed)

	var err error

	// Initialize cipher blocks for keys 0-3 (used for columns 0-1)
	for i := 0; i < 4; i++ {
		gen.enc03[i], err = aes.NewCipher(aesGenerator4RKeys[i][:])
		if err != nil {
			return nil, err
		}
		gen.dec03[i], err = aes.NewCipher(aesGenerator4RKeys[i][:])
		if err != nil {
			return nil, err
		}
	}

	// Initialize cipher blocks for keys 4-7 (used for columns 2-3)
	for i := 0; i < 4; i++ {
		gen.enc47[i], err = aes.NewCipher(aesGenerator4RKeys[4+i][:])
		if err != nil {
			return nil, err
		}
		gen.dec47[i], err = aes.NewCipher(aesGenerator4RKeys[4+i][:])
		if err != nil {
			return nil, err
		}
	}

	gen.pos = 64 // Force initial generation
	return gen, nil
}

// generate produces the next 64 bytes of pseudo-random data.
func (g *aesGenerator4R) generate() {
	var temp [4][16]byte

	// Column 0 (decrypt with keys 0-3, 4 rounds)
	copy(temp[0][:], g.state[0:16])
	for i := 0; i < 4; i++ {
		g.dec03[i].Decrypt(temp[0][:], temp[0][:])
	}

	// Column 1 (encrypt with keys 0-3, 4 rounds)
	copy(temp[1][:], g.state[16:32])
	for i := 0; i < 4; i++ {
		g.enc03[i].Encrypt(temp[1][:], temp[1][:])
	}

	// Column 2 (decrypt with keys 4-7, 4 rounds)
	copy(temp[2][:], g.state[32:48])
	for i := 0; i < 4; i++ {
		g.dec47[i].Decrypt(temp[2][:], temp[2][:])
	}

	// Column 3 (encrypt with keys 4-7, 4 rounds)
	copy(temp[3][:], g.state[48:64])
	for i := 0; i < 4; i++ {
		g.enc47[i].Encrypt(temp[3][:], temp[3][:])
	}

	// Copy results back to state
	copy(g.state[0:16], temp[0][:])
	copy(g.state[16:32], temp[1][:])
	copy(g.state[32:48], temp[2][:])
	copy(g.state[48:64], temp[3][:])
	g.pos = 0
}

// getByte returns the next pseudo-random byte.
func (g *aesGenerator4R) getByte() byte {
	if g.pos >= 64 {
		g.generate()
	}
	b := g.state[g.pos]
	g.pos++
	return b
}

// getBytes fills the provided slice with pseudo-random bytes.
func (g *aesGenerator4R) getBytes(dst []byte) {
	for i := range dst {
		dst[i] = g.getByte()
	}
}

// getUint32 returns the next pseudo-random uint32.
func (g *aesGenerator4R) getUint32() uint32 {
	if g.pos+4 > 64 {
		g.generate()
	}
	val := uint32(g.state[g.pos]) |
		uint32(g.state[g.pos+1])<<8 |
		uint32(g.state[g.pos+2])<<16 |
		uint32(g.state[g.pos+3])<<24
	g.pos += 4
	return val
}

// setState updates the generator's internal state.
// This is used in RandomX to update the generator between program iterations.
func (g *aesGenerator4R) setState(seed []byte) {
	if len(seed) != 64 {
		panic("aesGenerator4R: setState requires 64 bytes")
	}
	copy(g.state[:], seed)
	g.pos = 64 // Force regeneration on next read
}

// aesHash1R implements the RandomX AesHash1R scratchpad hashing algorithm.
// It processes the scratchpad in chunks and produces a 64-byte fingerprint.
type aesHash1R struct {
	state [64]byte // 4 columns of 16 bytes each
	enc   [2]cipher.Block
	dec   [2]cipher.Block
}

// newAesHash1R creates a new AesHash1R instance.
func newAesHash1R() (*aesHash1R, error) {
	h := &aesHash1R{}

	var err error

	// Initialize AES cipher blocks for encryption keys (columns 1, 3)
	h.enc[0], err = aes.NewCipher(aesGenerator1RKeys[1][:])
	if err != nil {
		return nil, err
	}
	h.enc[1], err = aes.NewCipher(aesGenerator1RKeys[3][:])
	if err != nil {
		return nil, err
	}

	// Initialize AES cipher blocks for decryption keys (columns 0, 2)
	h.dec[0], err = aes.NewCipher(aesGenerator1RKeys[0][:])
	if err != nil {
		return nil, err
	}
	h.dec[1], err = aes.NewCipher(aesGenerator1RKeys[2][:])
	if err != nil {
		return nil, err
	}

	return h, nil
}

// hash processes the scratchpad and produces a 64-byte fingerprint.
// The algorithm XORs the scratchpad data into the state using AES rounds.
func (h *aesHash1R) hash(scratchpad []byte) [64]byte {
	// Initialize state to zeros
	for i := range h.state {
		h.state[i] = 0
	}

	// Process scratchpad in 64-byte chunks
	// XOR each chunk into state and apply AES rounds
	for offset := 0; offset < len(scratchpad); offset += 64 {
		// XOR this chunk into the state
		for i := 0; i < 64 && offset+i < len(scratchpad); i++ {
			h.state[i] ^= scratchpad[offset+i]
		}

		// Apply AES rounds to mix the state
		h.mixState()
	}

	return h.state
}

// mixState applies one round of AES encryption/decryption to the state.
func (h *aesHash1R) mixState() {
	var newState [64]byte

	// Column 0 (decrypt with key0)
	h.dec[0].Decrypt(newState[0:16], h.state[0:16])

	// Column 1 (encrypt with key1)
	h.enc[0].Encrypt(newState[16:32], h.state[16:32])

	// Column 2 (decrypt with key2)
	h.dec[1].Decrypt(newState[32:48], h.state[32:48])

	// Column 3 (encrypt with key3)
	h.enc[1].Encrypt(newState[48:64], h.state[48:64])

	h.state = newState
}
