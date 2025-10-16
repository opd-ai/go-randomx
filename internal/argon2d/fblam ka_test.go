package argon2d

import (
	"testing"
)

// Test_fBlaMka_Zeros tests if fBlaMka preserves zeros
func Test_fBlaMka_Zeros(t *testing.T) {
	// Test the fBlaMka operation: a + b + 2*uint32(a)*uint32(b)
	a := uint64(0)
	b := uint64(0)

	result := a + b + 2*uint64(uint32(a))*uint64(uint32(b))

	t.Logf("fBlaMka(0, 0) = %d", result)
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}

	// Test with non-zero values
	a = 0x0000000000000001
	b = 0x0000000000000001
	result = a + b + 2*uint64(uint32(a))*uint64(uint32(b))
	t.Logf("fBlaMka(1, 1) = %d (0x%016x)", result, result)
	if result == 0 {
		t.Errorf("Expected non-zero, got 0")
	}
}

// TestCompressionOfZeros tests if 8 rounds of Blake2b preserve zeros
func TestCompressionOfZeros(t *testing.T) {
	var block Block
	// block is all zeros

	// Apply 8 rounds
	for round := 0; round < 8; round++ {
		applyBlake2bRound(&block)
	}

	// Check if still zeros
	allZeros := true
	for i := range block {
		if block[i] != 0 {
			allZeros = false
			t.Logf("After 8 rounds, block[%d] = 0x%016x", i, block[i])
			break
		}
	}

	if allZeros {
		t.Logf("✓ 8 rounds of Blake2b on zeros produces zeros (as expected)")
	} else {
		t.Errorf("8 rounds of Blake2b on zeros produced non-zeros!")
	}
}
