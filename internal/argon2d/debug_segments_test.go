package argon2d

import (
	"testing"
)

// TestFillMemory_AllSegments verifies all segments are processed.
func TestFillMemory_AllSegments(t *testing.T) {
	const numBlocks = 32
	passes := uint32(1)
	lanes := uint32(1)

	memory := make([]Block, numBlocks)

	// Initialize first two blocks
	password := []byte("test password")
	salt := []byte("test salt")
	h0 := initialHash(lanes, 32, numBlocks, 1, password, salt, nil, nil)
	initializeMemory(memory, lanes, h0)

	t.Logf("=== Before fillMemory ===")
	for i := 0; i < numBlocks; i++ {
		allZero := true
		for j := range memory[i] {
			if memory[i][j] != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			t.Logf("Block %d: ZERO", i)
		} else {
			t.Logf("Block %d: DATA (val[0]=0x%x)", i, memory[i][0])
		}
	}

	// Fill memory
	fillMemory(memory, passes, lanes)

	t.Logf("\n=== After fillMemory (1 pass) ===")
	segmentLength := numBlocks / SyncPoints
	for seg := 0; seg < SyncPoints; seg++ {
		startIdx := seg * segmentLength
		endIdx := startIdx + segmentLength

		filledCount := 0
		for i := startIdx; i < endIdx; i++ {
			hasData := false
			for j := range memory[i] {
				if memory[i][j] != 0 {
					hasData = true
					break
				}
			}
			if hasData {
				filledCount++
			}
		}
		t.Logf("Segment %d (blocks %d-%d): %d/%d blocks filled",
			seg, startIdx, endIdx-1, filledCount, segmentLength)
	}

	// Count total filled blocks
	totalFilled := 0
	for i := 0; i < numBlocks; i++ {
		for j := range memory[i] {
			if memory[i][j] != 0 {
				totalFilled++
				break
			}
		}
	}
	t.Logf("\nTotal: %d/%d blocks filled", totalFilled, numBlocks)

	if totalFilled < numBlocks {
		t.Errorf("Only %d/%d blocks filled after fillMemory", totalFilled, numBlocks)
	}
}
