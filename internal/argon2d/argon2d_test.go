package argon2d

import (
	"bytes"
	"testing"
)

// TestInitialHash_Basic verifies initialHash produces consistent output.
func TestInitialHash_Basic(t *testing.T) {
	password := []byte("password")
	salt := []byte("somesalt")

	h0 := initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)

	// Should produce 64-byte output
	if len(h0) != 64 {
		t.Errorf("initialHash produced %d bytes, expected 64", len(h0))
	}

	// Should not be all zeros
	allZero := true
	for _, b := range h0 {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("initialHash produced all zeros")
	}
}

// TestInitialHash_Deterministic verifies same inputs produce same output.
func TestInitialHash_Deterministic(t *testing.T) {
	password := []byte("test-password")
	salt := []byte("test-salt")

	h1 := initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)
	h2 := initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)

	if h1 != h2 {
		t.Error("initialHash is not deterministic")
	}
}

// TestInitialHash_DifferentInputs verifies different inputs produce different outputs.
func TestInitialHash_DifferentInputs(t *testing.T) {
	password1 := []byte("password1")
	password2 := []byte("password2")
	salt := []byte("somesalt")

	h1 := initialHash(1, 32, 256*1024, 3, password1, salt, nil, nil)
	h2 := initialHash(1, 32, 256*1024, 3, password2, salt, nil, nil)

	if h1 == h2 {
		t.Error("Different passwords produced identical hashes")
	}

	// Try different salt
	h3 := initialHash(1, 32, 256*1024, 3, password1, []byte("othersalt"), nil, nil)
	if h1 == h3 {
		t.Error("Different salts produced identical hashes")
	}

	// Try different parameters
	h4 := initialHash(1, 32, 512*1024, 3, password1, salt, nil, nil)
	if h1 == h4 {
		t.Error("Different memory sizes produced identical hashes")
	}
}

// TestInitialHash_WithSecret verifies secret key integration.
func TestInitialHash_WithSecret(t *testing.T) {
	password := []byte("password")
	salt := []byte("salt")
	secret := []byte("secret-key")

	h1 := initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)
	h2 := initialHash(1, 32, 256*1024, 3, password, salt, secret, nil)

	if h1 == h2 {
		t.Error("Secret key did not affect hash")
	}
}

// TestInitialHash_WithData verifies associated data integration.
func TestInitialHash_WithData(t *testing.T) {
	password := []byte("password")
	salt := []byte("salt")
	data := []byte("associated-data")

	h1 := initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)
	h2 := initialHash(1, 32, 256*1024, 3, password, salt, nil, data)

	if h1 == h2 {
		t.Error("Associated data did not affect hash")
	}
}

// TestInitialHash_EmptyInputs verifies handling of empty inputs.
func TestInitialHash_EmptyInputs(t *testing.T) {
	// Empty password and salt should still work
	h := initialHash(1, 32, 256*1024, 3, []byte{}, []byte{}, nil, nil)

	// Should not be all zeros (parameters still contribute)
	allZero := true
	for _, b := range h {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("initialHash with empty inputs produced all zeros")
	}
}

// TestInitialHash_LargeInputs verifies handling of large inputs.
func TestInitialHash_LargeInputs(t *testing.T) {
	// Create large password and salt (1 KB each)
	password := bytes.Repeat([]byte("a"), 1024)
	salt := bytes.Repeat([]byte("b"), 1024)

	h := initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)

	// Should handle large inputs without panic
	if len(h) != 64 {
		t.Errorf("initialHash with large inputs produced %d bytes, expected 64", len(h))
	}
}

// TestInitialHash_ParameterSensitivity tests sensitivity to all parameters.
func TestInitialHash_ParameterSensitivity(t *testing.T) {
	password := []byte("password")
	salt := []byte("salt")

	base := initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)

	tests := []struct {
		name   string
		lanes  uint32
		tag    uint32
		memory uint32
		time   uint32
	}{
		{"different lanes", 2, 32, 256 * 1024, 3},
		{"different tag", 1, 64, 256 * 1024, 3},
		{"different memory", 1, 32, 512 * 1024, 3},
		{"different time", 1, 32, 256 * 1024, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := initialHash(tt.lanes, tt.tag, tt.memory, tt.time, password, salt, nil, nil)
			if h == base {
				t.Errorf("%s did not affect hash", tt.name)
			}
		})
	}
}

// Benchmark initialHash performance.
func BenchmarkInitialHash(b *testing.B) {
	password := []byte("benchmark-password")
	salt := []byte("benchmark-salt")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)
	}
}

// Benchmark initialHash with large inputs.
func BenchmarkInitialHash_LargeInputs(b *testing.B) {
	password := bytes.Repeat([]byte("a"), 1024)
	salt := bytes.Repeat([]byte("b"), 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)
	}
}

// TestInitializeMemory_Basic verifies memory initialization from H0.
func TestInitializeMemory_Basic(t *testing.T) {
	// Generate H0
	password := []byte("password")
	salt := []byte("salt")
	h0 := initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)

	// Create memory (32 blocks for testing)
	const numBlocks = 32
	memory := make([]Block, numBlocks)

	// Initialize memory
	initializeMemory(memory, 1, h0)

	// Verify blocks 0 and 1 are non-zero
	allZero0 := true
	for _, v := range memory[0] {
		if v != 0 {
			allZero0 = false
			break
		}
	}
	if allZero0 {
		t.Error("Block 0 is all zeros after initialization")
	}

	allZero1 := true
	for _, v := range memory[1] {
		if v != 0 {
			allZero1 = false
			break
		}
	}
	if allZero1 {
		t.Error("Block 1 is all zeros after initialization")
	}

	// Verify blocks 0 and 1 are different
	if memory[0] == memory[1] {
		t.Error("Blocks 0 and 1 are identical")
	}

	// Verify remaining blocks are still zero
	for i := 2; i < numBlocks; i++ {
		allZero := true
		for _, v := range memory[i] {
			if v != 0 {
				allZero = false
				break
			}
		}
		if !allZero {
			t.Errorf("Block %d was modified (should still be zero)", i)
		}
	}
}

// TestInitializeMemory_Deterministic verifies consistent initialization.
func TestInitializeMemory_Deterministic(t *testing.T) {
	password := []byte("test")
	salt := []byte("salt")
	h0 := initialHash(1, 32, 256*1024, 3, password, salt, nil, nil)

	// Initialize two separate memory arrays
	memory1 := make([]Block, 32)
	memory2 := make([]Block, 32)

	initializeMemory(memory1, 1, h0)
	initializeMemory(memory2, 1, h0)

	// Should be identical
	if memory1[0] != memory2[0] {
		t.Error("Block 0 initialization is not deterministic")
	}
	if memory1[1] != memory2[1] {
		t.Error("Block 1 initialization is not deterministic")
	}
}

// TestInitializeMemory_DifferentH0 verifies different H0 produces different blocks.
func TestInitializeMemory_DifferentH0(t *testing.T) {
	h0_1 := initialHash(1, 32, 256*1024, 3, []byte("password1"), []byte("salt"), nil, nil)
	h0_2 := initialHash(1, 32, 256*1024, 3, []byte("password2"), []byte("salt"), nil, nil)

	memory1 := make([]Block, 32)
	memory2 := make([]Block, 32)

	initializeMemory(memory1, 1, h0_1)
	initializeMemory(memory2, 1, h0_2)

	// Should be different
	if memory1[0] == memory2[0] {
		t.Error("Different H0 values produced identical block 0")
	}
	if memory1[1] == memory2[1] {
		t.Error("Different H0 values produced identical block 1")
	}
}

// TestInitializeMemory_MultiLane verifies multi-lane initialization.
func TestInitializeMemory_MultiLane(t *testing.T) {
	h0 := initialHash(2, 32, 256*1024, 3, []byte("password"), []byte("salt"), nil, nil)

	// Create memory for 2 lanes (64 blocks total, 32 per lane)
	const numBlocks = 64
	memory := make([]Block, numBlocks)

	initializeMemory(memory, 2, h0)

	// Verify lane 0, blocks 0 and 1
	if memory[0] == (Block{}) {
		t.Error("Lane 0, block 0 is zero")
	}
	if memory[1] == (Block{}) {
		t.Error("Lane 0, block 1 is zero")
	}

	// Verify lane 1, blocks 0 and 1
	if memory[32] == (Block{}) {
		t.Error("Lane 1, block 0 is zero")
	}
	if memory[33] == (Block{}) {
		t.Error("Lane 1, block 1 is zero")
	}

	// Verify lanes are different
	if memory[0] == memory[32] {
		t.Error("Lane 0 and Lane 1 block 0 are identical")
	}
}

// Benchmark initializeMemory.
func BenchmarkInitializeMemory(b *testing.B) {
	h0 := initialHash(1, 32, 256*1024, 3, []byte("password"), []byte("salt"), nil, nil)
	memory := make([]Block, 262144) // 256 MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		initializeMemory(memory, 1, h0)
	}
}

// TestFinalizeHash_Basic verifies finalization produces expected output.
func TestFinalizeHash_Basic(t *testing.T) {
	// Create test memory
	const numBlocks = 32
	memory := make([]Block, numBlocks)

	// Initialize with test pattern
	for i := range memory {
		for j := range memory[i] {
			memory[i][j] = uint64(i*128 + j)
		}
	}

	// Finalize with tag length 32
	result := finalizeHash(memory, 1, 32)

	// Should produce 32 bytes
	if len(result) != 32 {
		t.Errorf("finalizeHash produced %d bytes, expected 32", len(result))
	}

	// Should not be all zeros
	allZero := true
	for _, b := range result {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("finalizeHash produced all zeros")
	}
}

// TestFinalizeHash_Deterministic verifies consistent output.
func TestFinalizeHash_Deterministic(t *testing.T) {
	const numBlocks = 32
	memory := make([]Block, numBlocks)

	// Initialize with test data
	for i := range memory {
		for j := range memory[i] {
			memory[i][j] = uint64(i + j*7)
		}
	}

	result1 := finalizeHash(memory, 1, 32)
	result2 := finalizeHash(memory, 1, 32)

	if !bytes.Equal(result1, result2) {
		t.Error("finalizeHash is not deterministic")
	}
}

// TestFinalizeHash_DifferentMemory verifies different memory produces different output.
func TestFinalizeHash_DifferentMemory(t *testing.T) {
	const numBlocks = 32
	memory1 := make([]Block, numBlocks)
	memory2 := make([]Block, numBlocks)

	// Initialize with different patterns
	for i := range memory1 {
		for j := range memory1[i] {
			memory1[i][j] = uint64(i + j)
			memory2[i][j] = uint64(i + j + 1) // Different!
		}
	}

	result1 := finalizeHash(memory1, 1, 32)
	result2 := finalizeHash(memory2, 1, 32)

	if bytes.Equal(result1, result2) {
		t.Error("Different memory produced identical hashes")
	}
}

// TestFinalizeHash_DifferentTagLength verifies variable output length.
func TestFinalizeHash_DifferentTagLength(t *testing.T) {
	const numBlocks = 32
	memory := make([]Block, numBlocks)

	// Initialize with test data
	for i := range memory {
		for j := range memory[i] {
			memory[i][j] = uint64(i*128 + j)
		}
	}

	// Test different tag lengths
	result16 := finalizeHash(memory, 1, 16)
	result32 := finalizeHash(memory, 1, 32)
	result64 := finalizeHash(memory, 1, 64)

	if len(result16) != 16 {
		t.Errorf("Tag length 16 produced %d bytes", len(result16))
	}
	if len(result32) != 32 {
		t.Errorf("Tag length 32 produced %d bytes", len(result32))
	}
	if len(result64) != 64 {
		t.Errorf("Tag length 64 produced %d bytes", len(result64))
	}

	// Different lengths should produce different results (first 16 bytes)
	if bytes.Equal(result16, result32[:16]) {
		t.Error("Different tag lengths produced identical prefixes")
	}
}

// TestFinalizeHash_SingleBlockChange verifies avalanche effect.
func TestFinalizeHash_SingleBlockChange(t *testing.T) {
	const numBlocks = 32
	memory1 := make([]Block, numBlocks)
	memory2 := make([]Block, numBlocks)

	// Initialize identically
	for i := range memory1 {
		for j := range memory1[i] {
			memory1[i][j] = uint64(i*128 + j)
			memory2[i][j] = uint64(i*128 + j)
		}
	}

	// Change single bit in memory2
	memory2[15][63] ^= 1

	result1 := finalizeHash(memory1, 1, 32)
	result2 := finalizeHash(memory2, 1, 32)

	// Should produce completely different hashes (avalanche effect)
	if bytes.Equal(result1, result2) {
		t.Error("Single bit change did not affect final hash")
	}

	// Count different bytes (should be many)
	differentBytes := 0
	for i := range result1 {
		if result1[i] != result2[i] {
			differentBytes++
		}
	}

	if differentBytes < 10 {
		t.Errorf("Only %d/32 bytes differ, expected strong avalanche effect", differentBytes)
	}
}

// Benchmark finalizeHash.
func BenchmarkFinalizeHash(b *testing.B) {
	memory := make([]Block, 262144) // 256 MB

	// Initialize with test data
	for i := range memory {
		for j := range memory[i] {
			memory[i][j] = uint64(i*128 + j)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = finalizeHash(memory, 1, 32)
	}
}

// TestArgon2d_Basic verifies the complete Argon2d algorithm works.
func TestArgon2d_Basic(t *testing.T) {
	password := []byte("password")
	salt := []byte("somesalt")

	// Use small parameters for testing (256 blocks = 256 KB)
	result := Argon2d(password, salt, 1, 256, 1, 32)

	if len(result) != 32 {
		t.Errorf("Argon2d produced %d bytes, expected 32", len(result))
	}

	// Should not be all zeros
	allZero := true
	for _, b := range result {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("Argon2d produced all zeros")
	}
}

// TestArgon2d_Deterministic verifies Argon2d is deterministic.
func TestArgon2d_Deterministic(t *testing.T) {
	password := []byte("test-password")
	salt := []byte("test-salt")

	result1 := Argon2d(password, salt, 1, 256, 1, 32)
	result2 := Argon2d(password, salt, 1, 256, 1, 32)

	if !bytes.Equal(result1, result2) {
		t.Error("Argon2d is not deterministic")
	}
}

// TestArgon2d_DifferentPasswords verifies password sensitivity.
func TestArgon2d_DifferentPasswords(t *testing.T) {
	salt := []byte("salt")

	result1 := Argon2d([]byte("password1"), salt, 1, 256, 1, 32)
	result2 := Argon2d([]byte("password2"), salt, 1, 256, 1, 32)

	if bytes.Equal(result1, result2) {
		t.Error("Different passwords produced identical hashes")
	}
}

// TestArgon2d_DifferentSalts verifies salt sensitivity.
func TestArgon2d_DifferentSalts(t *testing.T) {
	password := []byte("password")

	result1 := Argon2d(password, []byte("salt1"), 1, 256, 1, 32)
	result2 := Argon2d(password, []byte("salt2"), 1, 256, 1, 32)

	if bytes.Equal(result1, result2) {
		t.Error("Different salts produced identical hashes")
	}
}

// TestArgon2d_DifferentParameters verifies parameter sensitivity.
func TestArgon2d_DifferentParameters(t *testing.T) {
	password := []byte("password")
	salt := []byte("salt")

	result1 := Argon2d(password, salt, 1, 256, 1, 32)
	result2 := Argon2d(password, salt, 2, 256, 1, 32) // Different time cost
	result3 := Argon2d(password, salt, 1, 512, 1, 32) // Different memory

	if bytes.Equal(result1, result2) {
		t.Error("Different time costs produced identical hashes")
	}
	if bytes.Equal(result1, result3) {
		t.Error("Different memory sizes produced identical hashes")
	}
}

// TestArgon2d_VariableOutputLength verifies variable tag length.
func TestArgon2d_VariableOutputLength(t *testing.T) {
	password := []byte("password")
	salt := []byte("salt")

	result16 := Argon2d(password, salt, 1, 256, 1, 16)
	result32 := Argon2d(password, salt, 1, 256, 1, 32)
	result64 := Argon2d(password, salt, 1, 256, 1, 64)

	if len(result16) != 16 {
		t.Errorf("Tag length 16 produced %d bytes", len(result16))
	}
	if len(result32) != 32 {
		t.Errorf("Tag length 32 produced %d bytes", len(result32))
	}
	if len(result64) != 64 {
		t.Errorf("Tag length 64 produced %d bytes", len(result64))
	}
}

// TestArgon2d_MultiPass verifies multiple passes work correctly.
func TestArgon2d_MultiPass(t *testing.T) {
	password := []byte("password")
	salt := []byte("salt")

	result1 := Argon2d(password, salt, 1, 256, 1, 32)
	result3 := Argon2d(password, salt, 3, 256, 1, 32)

	// Different number of passes should produce different results
	if bytes.Equal(result1, result3) {
		t.Error("Different pass counts produced identical results")
	}
}

// TestArgon2dCache_Basic verifies the RandomX cache generation wrapper.
func TestArgon2dCache_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Argon2d cache test in short mode")
	}

	key := []byte("RandomX test key")

	cache := Argon2dCache(key)

	// RandomX cache should be 256 KB = 262144 bytes
	if len(cache) != 262144 {
		t.Errorf("Argon2dCache produced %d bytes, expected 262144", len(cache))
	}

	// Should not be all zeros
	allZero := true
	for _, b := range cache {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("Argon2dCache produced all zeros")
	}
}

// TestArgon2dCache_Deterministic verifies cache generation is deterministic.
func TestArgon2dCache_Deterministic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Argon2d cache test in short mode")
	}

	key := []byte("test-key")

	cache1 := Argon2dCache(key)
	cache2 := Argon2dCache(key)

	if !bytes.Equal(cache1, cache2) {
		t.Error("Argon2dCache is not deterministic")
	}
}

// TestArgon2dCache_DifferentKeys verifies different keys produce different caches.
func TestArgon2dCache_DifferentKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Argon2d cache test in short mode")
	}

	cache1 := Argon2dCache([]byte("key1"))
	cache2 := Argon2dCache([]byte("key2"))

	if bytes.Equal(cache1, cache2) {
		t.Error("Different keys produced identical caches")
	}
}

// Benchmark Argon2d with small parameters.
func BenchmarkArgon2d_Small(b *testing.B) {
	password := []byte("benchmark-password")
	salt := []byte("benchmark-salt")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Argon2d(password, salt, 1, 256, 1, 32)
	}
}

// Benchmark Argon2dCache (full RandomX parameters).
// This is expensive - 256 MB memory, 3 passes.
func BenchmarkArgon2dCache(b *testing.B) {
	key := []byte("benchmark-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Argon2dCache(key)
	}
}
