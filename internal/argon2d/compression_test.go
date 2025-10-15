package argon2d

import (
	"testing"
)

// TestFillBlock_Basic verifies fillBlock performs block compression.
func TestFillBlock_Basic(t *testing.T) {
	// Create test blocks with known patterns
	var prev, ref, next Block

	// Fill with test patterns
	for i := range prev {
		prev[i] = uint64(i)
		ref[i] = uint64(i * 2)
		next[i] = 0
	}

	// Apply fillBlock without XOR (first pass)
	fillBlock(&prev, &ref, &next, false)

	// Verify next block was modified
	allZero := true
	for i := range next {
		if next[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("fillBlock did not modify next block")
	}

	// Verify determinism - same inputs produce same output
	var next2 Block
	fillBlock(&prev, &ref, &next2, false)

	if next != next2 {
		t.Error("fillBlock is not deterministic")
	}
}

// TestFillBlock_WithXOR verifies fillBlock XOR behavior in later passes.
func TestFillBlock_WithXOR(t *testing.T) {
	var prev, ref, next, nextWithXOR Block

	// Initialize test blocks
	for i := range prev {
		prev[i] = uint64(i)
		ref[i] = uint64(i * 2)
		next[i] = uint64(i * 3)
		nextWithXOR[i] = uint64(i * 3)
	}

	// First call without XOR
	fillBlock(&prev, &ref, &next, false)

	// Second call with XOR
	fillBlock(&prev, &ref, &nextWithXOR, true)

	// Results should be different when withXOR changes
	if next == nextWithXOR {
		t.Error("fillBlock with/without XOR produced same result")
	}
}

// TestFillBlock_Deterministic verifies fillBlock is deterministic.
func TestFillBlock_Deterministic(t *testing.T) {
	var prev, ref Block

	// Initialize with pseudo-random pattern
	for i := range prev {
		prev[i] = uint64(i*7 + 13)
		ref[i] = uint64(i*11 + 17)
	}

	// Run fillBlock multiple times
	results := make([]Block, 10)
	for i := 0; i < 10; i++ {
		var next Block
		fillBlock(&prev, &ref, &next, false)
		results[i] = next
	}

	// All results should be identical
	for i := 1; i < 10; i++ {
		if results[0] != results[i] {
			t.Errorf("fillBlock not deterministic: result[0] != result[%d]", i)
		}
	}
}

// TestFillBlock_DifferentInputs verifies different inputs produce different outputs.
func TestFillBlock_DifferentInputs(t *testing.T) {
	var prev1, prev2, ref, next1, next2 Block

	// Create slightly different prev blocks
	for i := range prev1 {
		prev1[i] = uint64(i)
		prev2[i] = uint64(i + 1) // Different by 1
		ref[i] = uint64(i * 2)
	}

	// Fill both
	fillBlock(&prev1, &ref, &next1, false)
	fillBlock(&prev2, &ref, &next2, false)

	// Results should differ
	if next1 == next2 {
		t.Error("Different inputs produced identical outputs")
	}
}

// TestFillBlock_AvalancheEffect verifies small input changes cause large output changes.
func TestFillBlock_AvalancheEffect(t *testing.T) {
	var prev1, prev2, ref, next1, next2 Block

	// Create blocks differing by 1 bit
	for i := range prev1 {
		prev1[i] = uint64(i)
		prev2[i] = uint64(i)
		ref[i] = uint64(i * 2)
	}
	prev2[0] ^= 1 // Flip 1 bit

	// Fill both
	fillBlock(&prev1, &ref, &next1, false)
	fillBlock(&prev2, &ref, &next2, false)

	// Count differences
	diffCount := 0
	for i := range next1 {
		if next1[i] != next2[i] {
			diffCount++
		}
	}

	// At least 25% of values should differ (good avalanche)
	if diffCount < BlockSize128/4 {
		t.Errorf("Poor avalanche effect: only %d/%d values differ", diffCount, BlockSize128)
	}
}

// TestFillBlock_PreservesBlake2bStructure verifies Blake2b compression structure.
func TestFillBlock_PreservesBlake2bStructure(t *testing.T) {
	var prev, ref, next Block

	// Initialize with known pattern
	for i := range prev {
		prev[i] = 0x0123456789ABCDEF
		ref[i] = 0xFEDCBA9876543210
	}

	fillBlock(&prev, &ref, &next, false)

	// Verify the output has been thoroughly mixed
	// Check that values are not trivially related to inputs
	trivialCount := 0
	for i := range next {
		// Check if output is just XOR of inputs (too simple)
		if next[i] == (prev[i] ^ ref[i]) {
			trivialCount++
		}
	}

	// Should not be trivial XOR (Blake2b does more mixing)
	if trivialCount > 1 {
		t.Errorf("Too many trivial XOR values: %d/%d", trivialCount, BlockSize128)
	}
}

// TestFillBlock_XORIncorporatesExisting verifies withXOR=true uses existing content.
func TestFillBlock_XORIncorporatesExisting(t *testing.T) {
	var prev, ref, next1, next2 Block

	// Initialize blocks
	for i := range prev {
		prev[i] = uint64(i)
		ref[i] = uint64(i * 2)
		next1[i] = 0             // Start with zeros
		next2[i] = uint64(i * 3) // Start with pattern
	}

	// Fill with XOR (simulating second pass)
	fillBlock(&prev, &ref, &next1, true)
	fillBlock(&prev, &ref, &next2, true)

	// Results should differ because starting content differs
	if next1 == next2 {
		t.Error("withXOR=true did not incorporate existing content")
	}
}

// TestApplyBlake2bRound_Basic verifies applyBlake2bRound modifies block.
func TestApplyBlake2bRound_Basic(t *testing.T) {
	var block Block

	// Initialize with test pattern
	for i := range block {
		block[i] = uint64(i)
	}

	original := block

	// Apply one round
	applyBlake2bRound(&block)

	// Block should be modified
	if block == original {
		t.Error("applyBlake2bRound did not modify block")
	}
}

// TestApplyBlake2bRound_Deterministic verifies applyBlake2bRound is deterministic.
func TestApplyBlake2bRound_Deterministic(t *testing.T) {
	var block1, block2 Block

	// Initialize identically
	for i := range block1 {
		block1[i] = uint64(i*13 + 7)
		block2[i] = uint64(i*13 + 7)
	}

	// Apply round to both
	applyBlake2bRound(&block1)
	applyBlake2bRound(&block2)

	// Results should be identical
	if block1 != block2 {
		t.Error("applyBlake2bRound is not deterministic")
	}
}

// TestApplyBlake2bRound_Invertibility verifies round is not trivially invertible.
func TestApplyBlake2bRound_Invertibility(t *testing.T) {
	var block1, block2 Block

	// Initialize
	for i := range block1 {
		block1[i] = uint64(i)
		block2[i] = uint64(i)
	}

	// Apply round once
	applyBlake2bRound(&block1)

	// Apply round twice
	applyBlake2bRound(&block2)
	applyBlake2bRound(&block2)

	// Two rounds should not equal one round (not trivially invertible)
	if block1 == block2 {
		t.Error("applyBlake2bRound appears trivially invertible")
	}
}

// Benchmark fillBlock performance.
func BenchmarkFillBlock(b *testing.B) {
	var prev, ref, next Block

	// Initialize blocks
	for i := range prev {
		prev[i] = uint64(i)
		ref[i] = uint64(i * 2)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fillBlock(&prev, &ref, &next, false)
	}
}

// Benchmark fillBlock with XOR.
func BenchmarkFillBlock_WithXOR(b *testing.B) {
	var prev, ref, next Block

	// Initialize blocks
	for i := range prev {
		prev[i] = uint64(i)
		ref[i] = uint64(i * 2)
		next[i] = uint64(i * 3)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fillBlock(&prev, &ref, &next, true)
	}
}

// Benchmark applyBlake2bRound performance.
func BenchmarkApplyBlake2bRound(b *testing.B) {
	var block Block

	// Initialize block
	for i := range block {
		block[i] = uint64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applyBlake2bRound(&block)
	}
}
