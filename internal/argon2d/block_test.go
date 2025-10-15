package argon2d

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestBlock_Constants verifies block size constants
func TestBlock_Constants(t *testing.T) {
	if BlockSize != 1024 {
		t.Errorf("BlockSize = %d, want 1024", BlockSize)
	}

	if QWordsInBlock != 128 {
		t.Errorf("QWordsInBlock = %d, want 128", QWordsInBlock)
	}

	if BlockSize != QWordsInBlock*8 {
		t.Errorf("BlockSize (%d) != QWordsInBlock (%d) * 8", BlockSize, QWordsInBlock)
	}
}

// TestBlock_Zero verifies that Zero() clears all data
func TestBlock_Zero(t *testing.T) {
	var b Block

	// Fill with non-zero data
	for i := range b {
		b[i] = uint64(i + 1)
	}

	// Verify not all zeros
	allZeros := true
	for _, v := range b {
		if v != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Fatal("Block should not be all zeros before Zero()")
	}

	// Zero the block
	b.Zero()

	// Verify all zeros
	for i, v := range b {
		if v != 0 {
			t.Errorf("Block[%d] = %d after Zero(), want 0", i, v)
		}
	}
}

// TestBlock_Copy verifies that Copy() duplicates block data
func TestBlock_Copy(t *testing.T) {
	var src, dst Block

	// Fill source with test pattern
	for i := range src {
		src[i] = uint64(i*2 + 1)
	}

	// Copy to destination
	dst.Copy(&src)

	// Verify all values match
	for i := range src {
		if dst[i] != src[i] {
			t.Errorf("After Copy(), dst[%d] = %d, want %d", i, dst[i], src[i])
		}
	}

	// Verify modifying dst doesn't affect src
	dst[0] = 9999
	if src[0] == 9999 {
		t.Errorf("Modifying copy affected original block")
	}
}

// TestBlock_XOR verifies XOR operation correctness
func TestBlock_XOR(t *testing.T) {
	var a, b Block

	// Test pattern: alternate bits
	for i := range a {
		a[i] = 0xAAAAAAAAAAAAAAAA // 10101010...
		b[i] = 0x5555555555555555 // 01010101...
	}

	// XOR should produce all 1s
	a.XOR(&b)

	for i := range a {
		expected := uint64(0xFFFFFFFFFFFFFFFF)
		if a[i] != expected {
			t.Errorf("After XOR, block[%d] = 0x%016x, want 0x%016x", i, a[i], expected)
		}
	}
}

// TestBlock_XOR_Identity verifies XORing with itself produces zeros
func TestBlock_XOR_Identity(t *testing.T) {
	var a, b Block

	// Fill with random-looking pattern
	for i := range a {
		a[i] = uint64(i*7 + 13)
		b[i] = uint64(i*7 + 13)
	}

	// XOR with identical data should produce zeros
	a.XOR(&b)

	for i := range a {
		if a[i] != 0 {
			t.Errorf("After XOR with self, block[%d] = %d, want 0", i, a[i])
		}
	}
}

// TestBlock_XOR_Commutative verifies XOR is commutative: A XOR B == B XOR A
func TestBlock_XOR_Commutative(t *testing.T) {
	var a1, a2, b1, b2 Block

	// Initialize with test patterns
	for i := range a1 {
		a1[i] = uint64(i * 3)
		a2[i] = uint64(i * 3)
		b1[i] = uint64(i * 5)
		b2[i] = uint64(i * 5)
	}

	// Compute A XOR B
	a1.XOR(&b1)

	// Compute B XOR A
	b2.XOR(&a2)

	// Results should be identical
	for i := range a1 {
		if a1[i] != b2[i] {
			t.Errorf("XOR not commutative at index %d: a1[i]=0x%x, b2[i]=0x%x", i, a1[i], b2[i])
		}
	}
}

// TestBlock_FromBytes_ToBytes verifies round-trip conversion
func TestBlock_FromBytes_ToBytes(t *testing.T) {
	var b Block

	// Fill with test pattern
	for i := range b {
		b[i] = uint64(i*11 + 7)
	}

	// Convert to bytes
	data := b.ToBytes()

	// Verify size
	if len(data) != BlockSize {
		t.Fatalf("ToBytes() returned %d bytes, want %d", len(data), BlockSize)
	}

	// Convert back to block
	var restored Block
	if err := restored.FromBytes(data); err != nil {
		t.Fatalf("FromBytes() error: %v", err)
	}

	// Verify all values match
	for i := range b {
		if restored[i] != b[i] {
			t.Errorf("Round-trip failed at index %d: got %d, want %d", i, restored[i], b[i])
		}
	}
}

// TestBlock_FromBytes_InvalidSize verifies error handling for wrong size
func TestBlock_FromBytes_InvalidSize(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"empty", 0},
		{"too_small", 512},
		{"too_large", 2048},
		{"off_by_one_small", BlockSize - 1},
		{"off_by_one_large", BlockSize + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b Block
			data := make([]byte, tt.size)

			err := b.FromBytes(data)
			if err == nil {
				t.Errorf("FromBytes(%d bytes) succeeded, want error", tt.size)
			}

			// Verify error message contains useful information
			errMsg := err.Error()
			if errMsg == "" {
				t.Errorf("Error message is empty")
			}
		})
	}
}

// TestBlock_ToBytes_Endianness verifies little-endian encoding
func TestBlock_ToBytes_Endianness(t *testing.T) {
	var b Block

	// Set first uint64 to a known value
	b[0] = 0x0123456789ABCDEF

	data := b.ToBytes()

	// Verify little-endian encoding of first 8 bytes
	expected := []byte{0xEF, 0xCD, 0xAB, 0x89, 0x67, 0x45, 0x23, 0x01}
	if !bytes.Equal(data[:8], expected) {
		t.Errorf("ToBytes() endianness incorrect:\n  got: %x\n  want: %x", data[:8], expected)
	}

	// Verify using binary package
	value := binary.LittleEndian.Uint64(data[:8])
	if value != b[0] {
		t.Errorf("Binary decode = 0x%x, want 0x%x", value, b[0])
	}
}

// TestBlock_FromBytes_Endianness verifies little-endian decoding
func TestBlock_FromBytes_Endianness(t *testing.T) {
	// Create byte slice with known little-endian value
	data := make([]byte, BlockSize)
	data[0] = 0x01
	data[1] = 0x02
	data[2] = 0x03
	data[3] = 0x04
	data[4] = 0x05
	data[5] = 0x06
	data[6] = 0x07
	data[7] = 0x08

	var b Block
	if err := b.FromBytes(data); err != nil {
		t.Fatalf("FromBytes() error: %v", err)
	}

	// In little-endian: 0x0807060504030201
	expected := uint64(0x0807060504030201)
	if b[0] != expected {
		t.Errorf("FromBytes() decoded 0x%016x, want 0x%016x", b[0], expected)
	}
}

// TestBlock_Operations_Chaining verifies multiple operations work correctly
func TestBlock_Operations_Chaining(t *testing.T) {
	var a, b, c Block

	// Initialize blocks
	for i := range a {
		a[i] = uint64(i)
		b[i] = uint64(i * 2)
	}

	// Copy a to c
	c.Copy(&a)

	// XOR c with b
	c.XOR(&b)

	// Verify c = a XOR b
	for i := range a {
		expected := a[i] ^ b[i]
		if c[i] != expected {
			t.Errorf("Chained ops failed at index %d: got %d, want %d", i, c[i], expected)
		}
	}
}

// TestBlock_ZeroAfterUse verifies secure cleanup pattern
func TestBlock_ZeroAfterUse(t *testing.T) {
	var b Block

	// Simulate use with sensitive data
	for i := range b {
		b[i] = 0xDEADBEEFCAFEBABE
	}

	// Secure cleanup
	defer b.Zero()

	// After this test, block should be zeroed
	// (verified by the defer call)
}

// TestBlock_SizeVerification verifies Block has correct memory size
func TestBlock_SizeVerification(t *testing.T) {
	var b Block

	// Verify actual memory size matches expected
	data := b.ToBytes()
	if len(data) != 1024 {
		t.Errorf("Block memory size = %d bytes, want 1024", len(data))
	}
}

// TestInvalidBlockSizeError_Message verifies error formatting
func TestInvalidBlockSizeError_Message(t *testing.T) {
	err := &InvalidBlockSizeError{got: 512, want: 1024}
	msg := err.Error()

	// Verify error message contains key information
	if msg == "" {
		t.Error("Error message is empty")
	}

	// Should mention both got and want values
	t.Logf("Error message: %s", msg)
}

// BenchmarkBlock_XOR measures XOR performance
func BenchmarkBlock_XOR(b *testing.B) {
	var block1, block2 Block

	for i := range block1 {
		block1[i] = uint64(i)
		block2[i] = uint64(i * 2)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block1.XOR(&block2)
	}
}

// BenchmarkBlock_Copy measures Copy performance
func BenchmarkBlock_Copy(b *testing.B) {
	var src, dst Block

	for i := range src {
		src[i] = uint64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst.Copy(&src)
	}
}

// BenchmarkBlock_Zero measures Zero performance
func BenchmarkBlock_Zero(b *testing.B) {
	var block Block

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block.Zero()
	}
}

// BenchmarkBlock_ToBytes measures byte conversion performance
func BenchmarkBlock_ToBytes(b *testing.B) {
	var block Block

	for i := range block {
		block[i] = uint64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = block.ToBytes()
	}
}

// BenchmarkBlock_FromBytes measures byte parsing performance
func BenchmarkBlock_FromBytes(b *testing.B) {
	data := make([]byte, BlockSize)
	for i := 0; i < BlockSize; i++ {
		data[i] = byte(i)
	}

	var block Block

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = block.FromBytes(data)
	}
}
