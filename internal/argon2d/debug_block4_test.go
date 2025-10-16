package argon2d

import (
	"testing"
)

// TestFillSegment_Block4 specifically debugs what happens at block 4.
func TestFillSegment_Block4(t *testing.T) {
	const numBlocks = 32
	lanes := uint32(1)

	memory := make([]Block, numBlocks)

	// Initialize first two blocks
	password := []byte("test password")
	salt := []byte("test salt")
	h0 := initialHash(lanes, 32, numBlocks, 1, password, salt, nil, nil)
	initializeMemory(memory, lanes, h0)

	laneLength := uint32(numBlocks)
	segmentLength := laneLength / SyncPoints

	t.Logf("Lane length: %d, Segment length: %d", laneLength, segmentLength)

	// Manually process blocks 2, 3, 4 to see what happens
	pass := uint32(0)
	lane := uint32(0)
	slice := uint32(0)

	for i := uint32(2); i <= 4; i++ {
		currentIndex := i
		currOffset := lane*laneLength + currentIndex
		prevOffset := currOffset - 1

		pseudoRand := memory[prevOffset][0]

		pos := Position{
			Pass:  pass,
			Lane:  lane,
			Slice: slice,
			Index: i, // Index within segment
		}

		refIndex := indexAlpha(&pos, pseudoRand, segmentLength, laneLength)
		refOffset := lane*laneLength + refIndex

		t.Logf("\n=== Processing Block %d ===", i)
		t.Logf("prevOffset: %d, prevBlock[0]: 0x%016x", prevOffset, memory[prevOffset][0])
		t.Logf("pseudoRand: 0x%016x", pseudoRand)
		t.Logf("pos.Index: %d", pos.Index)
		t.Logf("refIndex: %d, refOffset: %d", refIndex, refOffset)

		// Check if ref has data
		refHasData := false
		for j := range memory[refOffset] {
			if memory[refOffset][j] != 0 {
				refHasData = true
				break
			}
		}
		t.Logf("refBlock has data: %v", refHasData)
		if refHasData {
			t.Logf("refBlock[0]: 0x%016x", memory[refOffset][0])
		}

		// Check if prev == ref
		if prevOffset == refOffset {
			t.Logf("⚠️  SELF-REFERENCE: prevOffset == refOffset")
		}

		// Call fillBlock
		fillBlock(&memory[prevOffset], &memory[refOffset], &memory[currOffset], false)

		// Check result
		currHasData := false
		for j := range memory[currOffset] {
			if memory[currOffset][j] != 0 {
				currHasData = true
				break
			}
		}
		t.Logf("After fillBlock, block %d has data: %v", i, currHasData)
		if currHasData {
			t.Logf("block[%d][0]: 0x%016x", i, memory[currOffset][0])
		} else {
			t.Logf("❌ Block %d is all zeros!", i)
		}
	}
}
