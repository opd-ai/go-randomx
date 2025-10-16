// Package argon2d implements Argon2d (data-dependent mode) for RandomX.
// This file contains the main memory filling algorithm.
package argon2d

// fillMemory implements the core Argon2d memory filling algorithm.
// It performs multiple passes over memory, using data-dependent addressing
// to select reference blocks and compress them into current blocks.
//
// This is the main algorithm that makes Argon2d memory-hard and ASIC-resistant.
//
// Parameters:
//   - memory: Pre-allocated slice of blocks to fill
//   - passes: Number of passes to make over memory (3 for RandomX)
//   - lanes: Number of parallel lanes (1 for RandomX - single-threaded)
//
// Algorithm per Argon2 specification:
//
//	For each pass (0 to passes-1):
//	  For each slice (0 to SyncPoints-1):
//	    For each block in segment:
//	      1. Get pseudo-random from previous block (data-dependent!)
//	      2. Compute reference index using indexAlpha
//	      3. Mix prev, ref → current using fillBlock
//	      4. Use XOR mode after first pass
func fillMemory(memory []Block, passes, lanes uint32) {
	laneLength := uint32(len(memory)) / lanes
	segmentLength := laneLength / SyncPoints

	for pass := uint32(0); pass < passes; pass++ {
		for slice := uint32(0); slice < SyncPoints; slice++ {
			for lane := uint32(0); lane < lanes; lane++ {
				// Process each block in the segment
				fillSegment(memory, pass, lane, slice, segmentLength, laneLength)
			}
		}
	}
} // fillSegment processes one segment of memory in a lane.
// A segment is 1/4 of the lane (SyncPoints = 4).
//
// This function implements the inner loop of Argon2d, where:
// - Each block is filled by mixing previous and reference blocks
// - Reference blocks are selected using data-dependent indexing
// - First pass initializes, later passes use XOR mode
func fillSegment(memory []Block, pass, lane, slice, segmentLength, laneLength uint32) {
	// Compute starting index for this segment
	startIndex := slice * segmentLength

	// Process each block in the segment
	for i := uint32(0); i < segmentLength; i++ {
		currentIndex := startIndex + i

		// Special case: skip first two blocks in first segment of first pass
		// (they are initialized separately from H0)
		if pass == 0 && slice == 0 && currentIndex < 2 {
			continue
		}

		// Compute current block offset in memory
		currOffset := lane*laneLength + currentIndex

		// Compute previous block offset (wraps around lane)
		prevOffset := currOffset - 1
		if currentIndex == 0 {
			// First block of lane references last block
			prevOffset = lane*laneLength + laneLength - 1
		}

		// Get pseudo-random value from previous block's first uint64
		// THIS IS DATA-DEPENDENT - the key to Argon2d!
		pseudoRand := memory[prevOffset][0]

		// Create position for indexAlpha
		pos := Position{
			Pass:  pass,
			Lane:  lane,
			Slice: slice,
			Index: i, // Index within the segment
		}

		// Compute reference block index using data-dependent addressing
		refIndex := indexAlpha(&pos, pseudoRand, segmentLength, laneLength)
		refOffset := lane*laneLength + refIndex

		// Mix blocks: prev XOR ref → current
		// Use XOR mode after first pass (withXOR = pass != 0)
		fillBlock(&memory[prevOffset], &memory[refOffset], &memory[currOffset], pass != 0)
	}
}
