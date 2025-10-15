// Package argon2d implements Argon2d (data-dependent mode) for RandomX.
// This file contains data-dependent indexing logic for block selection.
package argon2d

const (
	// SyncPoints is the number of segments per pass in Argon2.
	// Argon2 divides each pass into 4 segments for synchronization.
	SyncPoints = 4
)

// Position tracks the current location in Argon2 memory during processing.
// It represents where we are in the multi-dimensional memory space:
// - Pass: which iteration through memory (0-based)
// - Lane: which parallel lane (0 for single-threaded)
// - Slice: which segment within the pass (0 to SyncPoints-1)
// - Index: which block within the slice
type Position struct {
	Pass  uint32 // Current pass number (0 to timeCost-1)
	Lane  uint32 // Current lane number (0 to lanes-1)
	Slice uint32 // Current slice number (0 to SyncPoints-1)
	Index uint32 // Current index within slice
}

// indexAlpha computes the reference block index using data-dependent addressing.
//
// This is the KEY DIFFERENCE between Argon2d and Argon2i:
// - Argon2i uses pseudo-random counter (data-independent)
// - Argon2d uses pseudo-random from current block data (data-dependent)
//
// The function maps a pseudo-random value to a block index using a
// non-uniform (quadratic) distribution that favors recent blocks.
//
// Parameters:
//   - pos: Current position in memory
//   - pseudoRand: Pseudo-random value from current block's first uint64
//   - segmentLength: Number of blocks per segment
//   - laneLength: Total blocks in the lane
//
// Returns: Absolute block index to reference
//
// Algorithm per Argon2 specification (RFC 9106):
//  1. Compute reference area size based on pass and slice
//  2. Map pseudoRand to relative position using quadratic distribution
//  3. Convert relative position to absolute block index
func indexAlpha(pos *Position, pseudoRand uint64, segmentLength, laneLength uint32) uint32 {
	// Step 1: Determine the reference area size
	// This is the number of blocks we can reference from current position
	var referenceAreaSize uint32

	if pos.Pass == 0 {
		// First pass: can only reference blocks processed so far
		if pos.Slice == 0 {
			// First slice of first pass: only previous blocks in same slice
			referenceAreaSize = pos.Index
		} else {
			// Later slices: can reference all previous slices + current progress
			referenceAreaSize = pos.Slice*segmentLength + pos.Index
		}
	} else {
		// Later passes: can reference all blocks except current segment
		if pos.Slice == 0 {
			referenceAreaSize = laneLength - segmentLength + pos.Index
		} else {
			referenceAreaSize = laneLength - segmentLength + pos.Index
		}
	}

	// Handle edge case: must have at least one block to reference
	if referenceAreaSize == 0 {
		referenceAreaSize = 1
	}

	// Step 2: Map pseudoRand to relative position using quadratic distribution
	// This creates a distribution that favors more recent blocks
	// Formula: J2 = |J1| * |J1| / 2^32 where J1 is uniformly distributed
	relativePosition := pseudoRand & 0xFFFFFFFF // Use lower 32 bits

	// Apply quadratic mapping: x^2 / 2^32
	relativePosition = (relativePosition * relativePosition) >> 32

	// Invert to favor recent blocks: (size - 1) - (size * x^2 / 2^32)
	relativePosition = uint64(referenceAreaSize-1) -
		(uint64(referenceAreaSize) * relativePosition >> 32)

	// Step 3: Compute absolute block index
	// Start position depends on pass and slice
	var startPosition uint32
	if pos.Pass != 0 && pos.Slice != SyncPoints-1 {
		// Later passes: start after current segment
		startPosition = (pos.Slice + 1) * segmentLength
	} else {
		// First pass or last slice: start at beginning
		startPosition = 0
	}

	// Compute absolute index in lane
	absolutePosition := (startPosition + uint32(relativePosition)) % laneLength

	return absolutePosition
}
