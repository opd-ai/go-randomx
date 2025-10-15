package argon2d

import (
	"testing"
)

// TestPosition_Structure verifies Position struct holds correct fields.
func TestPosition_Structure(t *testing.T) {
	pos := Position{
		Pass:  1,
		Lane:  0,
		Slice: 2,
		Index: 42,
	}

	if pos.Pass != 1 {
		t.Errorf("Pass = %d, want 1", pos.Pass)
	}
	if pos.Lane != 0 {
		t.Errorf("Lane = %d, want 0", pos.Lane)
	}
	if pos.Slice != 2 {
		t.Errorf("Slice = %d, want 2", pos.Slice)
	}
	if pos.Index != 42 {
		t.Errorf("Index = %d, want 42", pos.Index)
	}
}

// TestIndexAlpha_FirstPassFirstSlice verifies indexing in pass 0, slice 0.
func TestIndexAlpha_FirstPassFirstSlice(t *testing.T) {
	// In first pass, first slice, can only reference blocks before current index
	pos := Position{Pass: 0, Lane: 0, Slice: 0, Index: 10}
	segmentLength := uint32(100)
	laneLength := uint32(400)

	// Test with various pseudo-random values
	tests := []struct {
		name       string
		pseudoRand uint64
	}{
		{"low_value", 0x1000},
		{"mid_value", 0x80000000},
		{"high_value", 0xFFFFFFFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refIndex := indexAlpha(&pos, tt.pseudoRand, segmentLength, laneLength)

			// Must reference a block before current index (0 to Index-1)
			if refIndex >= pos.Index {
				t.Errorf("refIndex=%d, must be < Index=%d in first pass, first slice",
					refIndex, pos.Index)
			}

			// Must be within lane bounds
			if refIndex >= laneLength {
				t.Errorf("refIndex=%d exceeds laneLength=%d", refIndex, laneLength)
			}
		})
	}
}

// TestIndexAlpha_FirstPassLaterSlice verifies indexing in pass 0, slice > 0.
func TestIndexAlpha_FirstPassLaterSlice(t *testing.T) {
	// In first pass, later slices, can reference all previous slices + current progress
	pos := Position{Pass: 0, Lane: 0, Slice: 2, Index: 10}
	segmentLength := uint32(100)
	laneLength := uint32(400)

	pseudoRand := uint64(0x12345678)
	refIndex := indexAlpha(&pos, pseudoRand, segmentLength, laneLength)

	// Maximum reference: slice*segmentLength + index - 1
	maxRef := pos.Slice*segmentLength + pos.Index
	if refIndex >= maxRef {
		t.Errorf("refIndex=%d, must be < maxRef=%d in first pass, slice %d",
			refIndex, maxRef, pos.Slice)
	}

	// Must be within lane bounds
	if refIndex >= laneLength {
		t.Errorf("refIndex=%d exceeds laneLength=%d", refIndex, laneLength)
	}
}

// TestIndexAlpha_LaterPass verifies indexing in pass > 0.
func TestIndexAlpha_LaterPass(t *testing.T) {
	// In later passes, can reference entire lane except current segment
	pos := Position{Pass: 1, Lane: 0, Slice: 1, Index: 50}
	segmentLength := uint32(100)
	laneLength := uint32(400)

	pseudoRand := uint64(0xABCDEF01)
	refIndex := indexAlpha(&pos, pseudoRand, segmentLength, laneLength)

	// Must be within lane bounds
	if refIndex >= laneLength {
		t.Errorf("refIndex=%d exceeds laneLength=%d", refIndex, laneLength)
	}

	// Should be able to reference most blocks (except current segment typically)
	// This is a weak constraint but validates basic behavior
	if refIndex == pos.Slice*segmentLength+pos.Index {
		t.Errorf("refIndex should not equal current block position")
	}
}

// TestIndexAlpha_Deterministic verifies same inputs produce same output.
func TestIndexAlpha_Deterministic(t *testing.T) {
	pos := Position{Pass: 0, Lane: 0, Slice: 1, Index: 25}
	segmentLength := uint32(100)
	laneLength := uint32(400)
	pseudoRand := uint64(0xDEADBEEF)

	// Call multiple times with same inputs
	results := make([]uint32, 10)
	for i := 0; i < 10; i++ {
		results[i] = indexAlpha(&pos, pseudoRand, segmentLength, laneLength)
	}

	// All results should be identical
	for i := 1; i < 10; i++ {
		if results[i] != results[0] {
			t.Errorf("indexAlpha not deterministic: results[%d]=%d != results[0]=%d",
				i, results[i], results[0])
		}
	}
}

// TestIndexAlpha_DifferentPseudoRand verifies different pseudoRand gives different results.
func TestIndexAlpha_DifferentPseudoRand(t *testing.T) {
	pos := Position{Pass: 1, Lane: 0, Slice: 2, Index: 50}
	segmentLength := uint32(100)
	laneLength := uint32(400)

	// Generate indices with different pseudo-random values
	results := make(map[uint32]bool)
	for i := uint64(0); i < 100; i++ {
		pseudoRand := i * uint64(0x123456789ABCDEF)
		refIndex := indexAlpha(&pos, pseudoRand, segmentLength, laneLength)
		results[refIndex] = true
	}

	// Should have generated multiple different indices (not all the same)
	if len(results) < 10 {
		t.Errorf("indexAlpha produced only %d unique values from 100 inputs, expected >10",
			len(results))
	}
}

// TestIndexAlpha_QuadraticDistribution verifies the distribution favors recent blocks.
func TestIndexAlpha_QuadraticDistribution(t *testing.T) {
	// In Argon2d, the quadratic mapping should favor more recent blocks
	pos := Position{Pass: 0, Lane: 0, Slice: 0, Index: 1000}
	segmentLength := uint32(1000)
	laneLength := uint32(4000)

	// Sample many pseudo-random values
	const samples = 10000
	distribution := make([]int, 10) // Divide range into 10 bins

	for i := 0; i < samples; i++ {
		pseudoRand := uint64(i) * uint64(0x9E3779B97F4A7C15) // Good mixing multiplier
		refIndex := indexAlpha(&pos, pseudoRand, segmentLength, laneLength)

		// Place into bin (0=oldest, 9=most recent)
		bin := int(refIndex * 10 / pos.Index)
		if bin >= 10 {
			bin = 9
		}
		distribution[bin]++
	}

	// The most recent bins should have more samples (quadratic distribution)
	// Check that the last bin has more samples than the first bin
	if distribution[9] <= distribution[0] {
		t.Logf("Distribution: %v", distribution)
		t.Errorf("Quadratic distribution not working: recent bin(%d) <= oldest bin(%d)",
			distribution[9], distribution[0])
	}

	// At least the last 3 bins should have more than first 3 bins combined
	recentCount := distribution[7] + distribution[8] + distribution[9]
	oldestCount := distribution[0] + distribution[1] + distribution[2]
	if recentCount <= oldestCount {
		t.Logf("Distribution: %v", distribution)
		t.Errorf("Quadratic distribution weak: recent(%d) <= oldest(%d)",
			recentCount, oldestCount)
	}
}

// TestIndexAlpha_BoundaryConditions verifies edge cases.
func TestIndexAlpha_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name          string
		pos           Position
		segmentLength uint32
		laneLength    uint32
		pseudoRand    uint64
		description   string
	}{
		{
			name:          "first_block",
			pos:           Position{Pass: 0, Lane: 0, Slice: 0, Index: 1},
			segmentLength: 100,
			laneLength:    400,
			pseudoRand:    0,
			description:   "Very first block can only reference itself conceptually",
		},
		{
			name:          "last_block_first_pass",
			pos:           Position{Pass: 0, Lane: 0, Slice: 3, Index: 99},
			segmentLength: 100,
			laneLength:    400,
			pseudoRand:    0xFFFFFFFF,
			description:   "Last block of first pass",
		},
		{
			name:          "first_block_second_pass",
			pos:           Position{Pass: 1, Lane: 0, Slice: 0, Index: 0},
			segmentLength: 100,
			laneLength:    400,
			pseudoRand:    0x80000000,
			description:   "First block of second pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refIndex := indexAlpha(&tt.pos, tt.pseudoRand, tt.segmentLength, tt.laneLength)

			// Basic validation: must be within lane
			if refIndex >= tt.laneLength {
				t.Errorf("%s: refIndex=%d exceeds laneLength=%d",
					tt.description, refIndex, tt.laneLength)
			}

			// Must not be negative (always true for uint32, but good documentation)
			// The function should handle all edge cases gracefully
		})
	}
}

// TestIndexAlpha_NoSelfReference verifies block doesn't reference itself.
func TestIndexAlpha_NoSelfReference(t *testing.T) {
	// Test various positions to ensure no self-reference
	tests := []struct {
		pos           Position
		segmentLength uint32
		laneLength    uint32
	}{
		{Position{0, 0, 0, 10}, 100, 400},
		{Position{0, 0, 1, 50}, 100, 400},
		{Position{1, 0, 2, 75}, 100, 400},
		{Position{2, 0, 3, 99}, 100, 400},
	}

	for _, tt := range tests {
		currentBlock := tt.pos.Slice*tt.segmentLength + tt.pos.Index

		// Try many pseudo-random values
		for i := uint64(0); i < 100; i++ {
			pseudoRand := i * uint64(0x123456789)
			refIndex := indexAlpha(&tt.pos, pseudoRand, tt.segmentLength, tt.laneLength)

			// Reference should not be current block
			if refIndex == currentBlock {
				t.Errorf("Position %+v referenced itself (block %d) with pseudoRand=0x%x",
					tt.pos, currentBlock, pseudoRand)
			}
		}
	}
}

// TestSyncPoints_Constant verifies SyncPoints is set correctly.
func TestSyncPoints_Constant(t *testing.T) {
	// Argon2 uses 4 sync points (segments) per pass
	if SyncPoints != 4 {
		t.Errorf("SyncPoints = %d, want 4 per Argon2 specification", SyncPoints)
	}
}

// Benchmark indexAlpha performance.
func BenchmarkIndexAlpha(b *testing.B) {
	pos := Position{Pass: 1, Lane: 0, Slice: 2, Index: 50}
	segmentLength := uint32(100)
	laneLength := uint32(400)
	pseudoRand := uint64(0x123456789ABCDEF)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = indexAlpha(&pos, pseudoRand, segmentLength, laneLength)
	}
}

// Benchmark indexAlpha with varying pseudo-random values.
func BenchmarkIndexAlpha_VaryingInput(b *testing.B) {
	pos := Position{Pass: 1, Lane: 0, Slice: 2, Index: 50}
	segmentLength := uint32(100)
	laneLength := uint32(400)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pseudoRand := uint64(i) * uint64(0x9E3779B97F4A7C15)
		_ = indexAlpha(&pos, pseudoRand, segmentLength, laneLength)
	}
}
