package argon2d

import (
	"fmt"
	"testing"
)

// TestG_Zeros tests the g function with zero inputs.
func TestG_Zeros(t *testing.T) {
	a, b, c, d := g(0, 0, 0, 0)
	t.Logf("g(0,0,0,0) = (%d, %d, %d, %d)", a, b, c, d)

	if a == 0 && b == 0 && c == 0 && d == 0 {
		t.Error("g(0,0,0,0) produced all zeros - this is the bug!")
	}
}

// TestFillBlock_SameBlocks tests fillBlock with identical prev and ref.
func TestFillBlock_SameBlocks(t *testing.T) {
	var prev, ref, next Block

	// Initialize prev and ref identically
	for i := range prev {
		prev[i] = uint64(i + 1)
		ref[i] = uint64(i + 1) // Same as prev!
	}

	t.Logf("Before fillBlock:")
	t.Logf("prev[0] = %d", prev[0])
	t.Logf("ref[0] = %d", ref[0])
	t.Logf("next[0] = %d", next[0])

	// Call fillBlock with identical blocks
	fillBlock(&prev, &ref, &next, false)

	t.Logf("After fillBlock:")
	t.Logf("next[0] = %d", next[0])
	t.Logf("next[1] = %d", next[1])
	t.Logf("next[127] = %d", next[127])

	// Result should still be non-zero!
	// Because: R = prev XOR ref = 0, then compression should produce non-zero
	if next[0] == 0 && next[1] == 0 && next[127] == 0 {
		t.Error("fillBlock produced all zeros when prev==ref")
	}
}

// TestFillBlock_Direct tests fillBlock directly.
func TestFillBlock_Direct(t *testing.T) {
	var prev, ref, next Block

	// Initialize prev and ref with some data
	for i := range prev {
		prev[i] = uint64(i + 1)
		ref[i] = uint64((i + 1) * 2)
	}

	fmt.Printf("Before fillBlock:\n")
	fmt.Printf("prev[0] = %d\n", prev[0])
	fmt.Printf("ref[0] = %d\n", ref[0])
	fmt.Printf("next[0] = %d\n", next[0])

	// Call fillBlock
	fillBlock(&prev, &ref, &next, false)

	fmt.Printf("\nAfter fillBlock:\n")
	fmt.Printf("next[0] = %d (should be non-zero)\n", next[0])
	fmt.Printf("next[1] = %d (should be non-zero)\n", next[1])

	if next[0] == 0 {
		t.Error("fillBlock did not fill next[0]")
	}
	if next[1] == 0 {
		t.Error("fillBlock did not fill next[1]")
	}
}

// TestFillSegment_Inline tests by inlining the logic.
func TestFillSegment_Inline(t *testing.T) {
	const numBlocks = 8
	memory := make([]Block, numBlocks)

	// Initialize blocks 0 and 1
	for i := range memory[0] {
		memory[0][i] = uint64(i + 1)
		memory[1][i] = uint64((i + 1) * 2)
	}

	t.Logf("Block 0[0] = %d", memory[0][0])
	t.Logf("Block 1[0] = %d", memory[1][0])
	t.Logf("Block 2[0] before = %d", memory[2][0])

	// Inline what fillSegment should do for block 2
	pass := uint32(0)
	lane := uint32(0)
	slice := uint32(0)
	segmentLength := uint32(8)
	laneLength := uint32(8)

	i := uint32(2)                               // Processing block index 2 (skip 0 and 1)
	currentIndex := slice*segmentLength + i      // = 0*8 + 2 = 2
	currOffset := lane*laneLength + currentIndex // = 0*8 + 2 = 2
	prevOffset := currOffset - 1                 // = 1

	pseudoRand := memory[prevOffset][0]

	pos := Position{
		Pass:  pass,
		Lane:  lane,
		Slice: slice,
		Index: i,
	}

	refIndex := indexAlpha(&pos, pseudoRand, segmentLength, laneLength)
	refOffset := lane*laneLength + refIndex

	t.Logf("Processing block 2:")
	t.Logf("  currOffset=%d, prevOffset=%d, refOffset=%d", currOffset, prevOffset, refOffset)
	t.Logf("  pseudoRand=%d", pseudoRand)

	// Call fillBlock
	fillBlock(&memory[prevOffset], &memory[refOffset], &memory[currOffset], false)

	t.Logf("Block 2[0] after = %d", memory[2][0])

	if memory[2][0] == 0 {
		t.Error("Block 2 was not filled by manual inline")
	}
}

// TestFillSegment_Minimal tests the absolute minimum.
func TestFillSegment_Minimal(t *testing.T) {
	// Create tiny memory - just 8 blocks
	const numBlocks = 8
	memory := make([]Block, numBlocks)

	// Initialize blocks 0 and 1
	for i := range memory[0] {
		memory[0][i] = uint64(i + 1)
		memory[1][i] = uint64((i + 1) * 2)
	}

	t.Logf("Before fillSegment:")
	t.Logf("Block 0[0] = %d", memory[0][0])
	t.Logf("Block 1[0] = %d", memory[1][0])
	t.Logf("Block 2[0] = %d", memory[2][0])

	// Call fillSegment to process slice 0 (blocks 0-7, but will skip 0-1)
	// Parameters: memory, pass, lane, slice, segmentLength, laneLength
	fillSegment(memory, 0, 0, 0, 8, 8)

	t.Logf("After fillSegment:")
	for i := 0; i < numBlocks; i++ {
		t.Logf("Block %d[0] = %d", i, memory[i][0])
	}

	// Block 2 should be modified
	if memory[2][0] == 0 {
		t.Error("Block 2 was not filled")
	}
} // TestFillSegment_Debug helps debug the segment filling logic.
func TestFillSegment_Debug(t *testing.T) {
	const numBlocks = 32
	memory := make([]Block, numBlocks)
	segmentLength := uint32(numBlocks / SyncPoints) // 8 blocks

	// Initialize first two blocks
	for i := range memory[0] {
		memory[0][i] = uint64(i + 1)
		memory[1][i] = uint64((i + 1) * 2)
	}

	fmt.Printf("Before fillSegment:\n")
	fmt.Printf("Block 0[0] = %d\n", memory[0][0])
	fmt.Printf("Block 1[0] = %d\n", memory[1][0])
	fmt.Printf("Block 2[0] = %d\n", memory[2][0])

	// Manually try what fillSegment should do for block 2
	fmt.Printf("\nManual simulation for block 2 (index 0 in segment after skipping 0-1):\n")
	i := uint32(0)                // first iteration
	currentIndex := uint32(0) + i // = 0
	fmt.Printf("i=%d, currentIndex=%d\n", i, currentIndex)
	fmt.Printf("Should skip? pass==0 && slice==0 && currentIndex < 2: %v\n", currentIndex < 2)

	i = uint32(2)                // what should actually be processed first
	currentIndex = uint32(0) + i // = 2
	fmt.Printf("i=%d, currentIndex=%d\n", i, currentIndex)
	fmt.Printf("Should skip? pass==0 && slice==0 && currentIndex < 2: %v\n", currentIndex < 2)

	// Fill first segment (blocks 0-7, but should skip 0-1)
	fillSegment(memory, 0, 0, 0, segmentLength, numBlocks)

	fmt.Printf("\nAfter fillSegment:\n")
	fmt.Printf("Block 0[0] = %d (should be unchanged: 1)\n", memory[0][0])
	fmt.Printf("Block 1[0] = %d (should be unchanged: 2)\n", memory[1][0])
	fmt.Printf("Block 2[0] = %d (should be non-zero)\n", memory[2][0])
	fmt.Printf("Block 3[0] = %d (should be non-zero)\n", memory[3][0])

	// Check if blocks were actually modified
	if memory[2][0] == 0 {
		t.Error("Block 2 was not filled")
	}
	if memory[3][0] == 0 {
		t.Error("Block 3 was not filled")
	}
}

// TestIndexAlpha_Debug helps verify indexing works.
func TestIndexAlpha_Debug(t *testing.T) {
	pos := Position{
		Pass:  0,
		Lane:  0,
		Slice: 0,
		Index: 0, // Processing block 2 (index 0 in segment, after skipping 0-1)
	}

	pseudoRand := uint64(123456789)
	segmentLength := uint32(8)
	laneLength := uint32(32)

	refIndex := indexAlpha(&pos, pseudoRand, segmentLength, laneLength)
	fmt.Printf("indexAlpha returned: %d (for processing block 2)\n", refIndex)
	fmt.Printf("pseudoRand: %d\n", pseudoRand)
	fmt.Printf("segmentLength: %d\n", segmentLength)
	fmt.Printf("laneLength: %d\n", laneLength)

	// refIndex should be in range [0, 2) for first segment, first pass
	if refIndex >= 2 {
		t.Errorf("refIndex %d should be < 2 for first block", refIndex)
	}
}
