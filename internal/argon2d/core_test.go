package argon2d

import (
	"testing"
)

// TestFillMemory_Basic verifies fillMemory processes memory correctly.
func TestFillMemory_Basic(t *testing.T) {
	// Create small memory for testing (32 blocks = 32 KB)
	const numBlocks = 32
	memory := make([]Block, numBlocks)

	// Initialize first two blocks using proper H0-based initialization
	// This matches how the real Argon2d algorithm initializes memory
	password := []byte("test password")
	salt := []byte("test salt")
	lanes := uint32(1)

	h0 := initialHash(lanes, 32, numBlocks, 1, password, salt, nil, nil)
	initializeMemory(memory, lanes, h0)

	// Fill memory with 1 pass
	passes := uint32(1)

	fillMemory(memory, passes, lanes)

	// Verify blocks beyond first two were modified
	allZero := true
	for i := 2; i < numBlocks; i++ {
		for j := range memory[i] {
			if memory[i][j] != 0 {
				allZero = false
				break
			}
		}
		if !allZero {
			break
		}
	}

	if allZero {
		t.Error("fillMemory did not modify blocks beyond first two")
	}
}

// TestFillMemory_Deterministic verifies fillMemory is deterministic.
func TestFillMemory_Deterministic(t *testing.T) {
	const numBlocks = 32
	passes := uint32(1)
	lanes := uint32(1)

	// Create and fill first memory
	memory1 := make([]Block, numBlocks)
	for i := range memory1[0] {
		memory1[0][i] = uint64(i * 13)
		memory1[1][i] = uint64(i * 17)
	}
	fillMemory(memory1, passes, lanes)

	// Create and fill second memory with same initialization
	memory2 := make([]Block, numBlocks)
	for i := range memory2[0] {
		memory2[0][i] = uint64(i * 13)
		memory2[1][i] = uint64(i * 17)
	}
	fillMemory(memory2, passes, lanes)

	// Results should be identical
	for i := 0; i < numBlocks; i++ {
		if memory1[i] != memory2[i] {
			t.Errorf("fillMemory not deterministic at block %d", i)
			break
		}
	}
}

// TestFillMemory_MultiPass verifies multiple passes work correctly.
func TestFillMemory_MultiPass(t *testing.T) {
	const numBlocks = 32
	passes := uint32(3) // RandomX uses 3 passes
	lanes := uint32(1)

	memory := make([]Block, numBlocks)

	// Initialize first two blocks using proper initialization
	password := []byte("test password")
	salt := []byte("test salt")
	h0 := initialHash(lanes, 32, numBlocks, 1, password, salt, nil, nil)
	initializeMemory(memory, lanes, h0)

	// Save initial state of block 2
	initialBlock2 := memory[2]

	// Fill with multiple passes
	fillMemory(memory, passes, lanes)

	// Block 2 should be different from initial state
	if memory[2] == initialBlock2 {
		t.Error("Multi-pass fillMemory did not modify block 2")
	}

	// All blocks should be non-zero after 3 passes
	for i := 2; i < numBlocks; i++ {
		allZero := true
		for j := range memory[i] {
			if memory[i][j] != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			t.Errorf("Block %d is all zeros after 3 passes", i)
		}
	}
}

// TestFillMemory_DifferentInitialization verifies different inputs produce different outputs.
func TestFillMemory_DifferentInitialization(t *testing.T) {
	const numBlocks = 32
	passes := uint32(1)
	lanes := uint32(1)

	// Create first memory with one initialization
	memory1 := make([]Block, numBlocks)
	for i := range memory1[0] {
		memory1[0][i] = uint64(i)
		memory1[1][i] = uint64(i * 2)
	}
	fillMemory(memory1, passes, lanes)

	// Create second memory with different initialization
	memory2 := make([]Block, numBlocks)
	for i := range memory2[0] {
		memory2[0][i] = uint64(i + 1) // Different!
		memory2[1][i] = uint64(i * 2)
	}
	fillMemory(memory2, passes, lanes)

	// Results should differ
	different := false
	for i := 2; i < numBlocks; i++ {
		if memory1[i] != memory2[i] {
			different = true
			break
		}
	}

	if !different {
		t.Error("Different initializations produced identical results")
	}
}

// TestFillSegment_Basic verifies fillSegment processes one segment.
func TestFillSegment_Basic(t *testing.T) {
	const numBlocks = 32
	memory := make([]Block, numBlocks)
	segmentLength := uint32(numBlocks / SyncPoints) // 8 blocks

	// Initialize first two blocks
	for i := range memory[0] {
		memory[0][i] = uint64(i)
		memory[1][i] = uint64(i * 2)
	}

	// Fill first segment (blocks 0-7, but skips 0-1)
	fillSegment(memory, 0, 0, 0, segmentLength, numBlocks)

	// Blocks 2-7 should be modified
	for i := uint32(2); i < segmentLength; i++ {
		allZero := true
		for j := range memory[i] {
			if memory[i][j] != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			t.Errorf("Block %d was not modified by fillSegment", i)
		}
	}

	// Blocks 8-31 should still be zero
	for i := segmentLength; i < numBlocks; i++ {
		allZero := true
		for j := range memory[i] {
			if memory[i][j] != 0 {
				allZero = false
				break
			}
		}
		if !allZero {
			t.Errorf("Block %d was modified but should not have been", i)
		}
	}
}

// TestFillSegment_FirstPassSkipsFirstTwo verifies first segment skips blocks 0-1.
func TestFillSegment_FirstPassSkipsFirstTwo(t *testing.T) {
	const numBlocks = 32
	memory := make([]Block, numBlocks)
	segmentLength := uint32(numBlocks / SyncPoints)

	// Initialize blocks 0 and 1 with known values
	for i := range memory[0] {
		memory[0][i] = 0xAAAAAAAAAAAAAAAA
		memory[1][i] = 0xBBBBBBBBBBBBBBBB
	}

	// Fill first segment
	fillSegment(memory, 0, 0, 0, segmentLength, numBlocks)

	// Blocks 0 and 1 should be unchanged
	for i := range memory[0] {
		if memory[0][i] != 0xAAAAAAAAAAAAAAAA {
			t.Errorf("Block 0[%d] was modified (should be skipped)", i)
		}
		if memory[1][i] != 0xBBBBBBBBBBBBBBBB {
			t.Errorf("Block 1[%d] was modified (should be skipped)", i)
		}
	}
}

// TestFillSegment_LaterSegments verifies later segments process all blocks.
func TestFillSegment_LaterSegments(t *testing.T) {
	const numBlocks = 32
	memory := make([]Block, numBlocks)
	segmentLength := uint32(numBlocks / SyncPoints) // 8 blocks per segment

	// Initialize all blocks with some data
	for i := 0; i < numBlocks; i++ {
		for j := range memory[i] {
			memory[i][j] = uint64(i*128 + j)
		}
	}

	// Fill second segment (blocks 8-15)
	slice := uint32(1)
	fillSegment(memory, 0, 0, slice, segmentLength, numBlocks)

	// All blocks in segment should be modified
	// (we can't easily verify exact values, but check they changed)
	startIndex := slice * segmentLength
	for i := startIndex; i < startIndex+segmentLength; i++ {
		// Check if block is different from initial pattern
		unchanged := true
		for j := range memory[i] {
			if memory[i][j] != uint64(int(i)*128+j) {
				unchanged = false
				break
			}
		}
		if unchanged {
			t.Errorf("Block %d in segment %d was not modified", i, slice)
		}
	}
}

// TestFillSegment_UsesDataDependentIndexing verifies pseudo-random from prev block.
func TestFillSegment_UsesDataDependentIndexing(t *testing.T) {
	const numBlocks = 32
	segmentLength := uint32(numBlocks / SyncPoints)

	// Create two memories with different prev block data
	memory1 := make([]Block, numBlocks)
	memory2 := make([]Block, numBlocks)

	// Initialize identically except for one value in block 1
	for i := 0; i < 2; i++ {
		for j := range memory1[i] {
			memory1[i][j] = uint64(j)
			memory2[i][j] = uint64(j)
		}
	}
	memory2[1][0] = 0xFFFFFFFFFFFFFFFF // Different pseudoRand source!

	// Fill both
	fillSegment(memory1, 0, 0, 0, segmentLength, numBlocks)
	fillSegment(memory2, 0, 0, 0, segmentLength, numBlocks)

	// Results should differ because pseudoRand was different
	different := false
	for i := uint32(2); i < segmentLength; i++ {
		if memory1[i] != memory2[i] {
			different = true
			break
		}
	}

	if !different {
		t.Error("Different pseudoRand values produced identical results")
	}
}

// TestFillMemory_XORModeAfterFirstPass verifies XOR behavior across passes.
func TestFillMemory_XORModeAfterFirstPass(t *testing.T) {
	const numBlocks = 32
	passes := uint32(2)
	lanes := uint32(1)

	// Create memory and initialize
	memory := make([]Block, numBlocks)
	for i := range memory[0] {
		memory[0][i] = uint64(i)
		memory[1][i] = uint64(i * 2)
	}

	// Fill with just first pass
	memory1Pass := make([]Block, numBlocks)
	copy(memory1Pass, memory)
	fillMemory(memory1Pass, 1, lanes)

	// Fill with two passes
	memory2Pass := make([]Block, numBlocks)
	copy(memory2Pass, memory)
	fillMemory(memory2Pass, passes, lanes)

	// Results should differ (second pass XORs with existing content)
	if memory1Pass[5] == memory2Pass[5] {
		t.Error("Second pass did not produce different results")
	}
}

// Benchmark fillMemory with small memory.
func BenchmarkFillMemory_Small(b *testing.B) {
	const numBlocks = 256 // 256 KB
	passes := uint32(3)
	lanes := uint32(1)

	memory := make([]Block, numBlocks)
	for i := range memory[0] {
		memory[0][i] = uint64(i)
		memory[1][i] = uint64(i * 2)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fillMemory(memory, passes, lanes)
	}
}

// Benchmark fillSegment.
func BenchmarkFillSegment(b *testing.B) {
	const numBlocks = 256
	segmentLength := uint32(numBlocks / SyncPoints)

	memory := make([]Block, numBlocks)
	for i := range memory[0] {
		memory[0][i] = uint64(i)
		memory[1][i] = uint64(i * 2)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fillSegment(memory, 0, 0, 0, segmentLength, numBlocks)
	}
}
