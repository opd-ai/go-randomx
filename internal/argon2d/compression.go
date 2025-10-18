// Package argon2d implements Argon2d (data-dependent mode) for RandomX.
// This file contains block compression functions using Blake2b mixing.
package argon2d

const (
	// BlockSize128 is the number of uint64 values in a Block (128 = 1024 bytes / 8)
	BlockSize128 = 128
)

// fillBlock performs Argon2 block compression using Blake2b rounds.
// It mixes prevBlock and refBlock into nextBlock using Blake2b-style compression.
//
// Parameters:
//   - prevBlock: The previous block in the sequence
//   - refBlock: The reference block (chosen by data-dependent indexing)
//   - nextBlock: The output block to fill
//   - withXOR: If true, XOR with existing nextBlock content (used after first pass)
//
// Algorithm per Argon2 specification (RFC 9106 Section 3.4):
//  1. R = refBlock XOR prevBlock
//  2. Q = R (save original XOR)
//  3. Apply permutation P (Blake2b rounds with fBlaMka) to R
//  4. nextBlock = R XOR Q
//  5. If withXOR: nextBlock = nextBlock XOR oldNextBlock
func fillBlock(prevBlock, refBlock, nextBlock *Block, withXOR bool) {
	var R, Q Block

	// Step 1: R = refBlock XOR prevBlock
	R = *refBlock
	R.XOR(prevBlock)

	// Step 2: Q = R (save for feed-forward)
	Q = R

	// Step 3: Apply permutation P as per Argon2 reference implementation
	// This consists of:
	// - 8 rounds of Blake2b on columns (groups of 16 consecutive uint64s)
	// - 8 rounds of Blake2b on rows (interleaved pattern)
	applyBlake2bRound(&R)

	// Step 4: Feed-forward - R = R XOR Q
	R.XOR(&Q)

	// Step 5: If second+ pass, XOR with old nextBlock content
	if withXOR {
		oldNext := *nextBlock
		R.XOR(&oldNext)
	}

	// Step 6: Write result to nextBlock
	*nextBlock = R
}

// applyBlake2bRound applies the Argon2 permutation P to a block.
// This matches the reference implementation exactly:
// - 8 rounds on columns (consecutive groups of 16 uint64s)
// - 8 rounds on rows (interleaved pattern)
//
// Reference: Argon2 reference implementation fill_block() in argon2_ref.c
func applyBlake2bRound(block *Block) {
	// Apply Blake2 on columns: (0,1,...,15), (16,17,...,31), ..., (112,113,...,127)
	for i := 0; i < 8; i++ {
		gRound(block[i*16 : (i+1)*16])
	}

	// Apply Blake2 on rows (interleaved pattern):
	// (0,1,16,17,32,33,48,49,64,65,80,81,96,97,112,113)
	// (2,3,18,19,34,35,50,51,66,67,82,83,98,99,114,115)
	// ...
	// (14,15,30,31,46,47,62,63,78,79,94,95,110,111,126,127)
	for i := 0; i < 8; i++ {
		// Extract interleaved elements into a temporary slice
		var row [16]uint64
		row[0] = block[2*i]
		row[1] = block[2*i+1]
		row[2] = block[2*i+16]
		row[3] = block[2*i+17]
		row[4] = block[2*i+32]
		row[5] = block[2*i+33]
		row[6] = block[2*i+48]
		row[7] = block[2*i+49]
		row[8] = block[2*i+64]
		row[9] = block[2*i+65]
		row[10] = block[2*i+80]
		row[11] = block[2*i+81]
		row[12] = block[2*i+96]
		row[13] = block[2*i+97]
		row[14] = block[2*i+112]
		row[15] = block[2*i+113]

		// Apply Blake2b round
		gRound(row[:])

		// Write back
		block[2*i] = row[0]
		block[2*i+1] = row[1]
		block[2*i+16] = row[2]
		block[2*i+17] = row[3]
		block[2*i+32] = row[4]
		block[2*i+33] = row[5]
		block[2*i+48] = row[6]
		block[2*i+49] = row[7]
		block[2*i+64] = row[8]
		block[2*i+65] = row[9]
		block[2*i+80] = row[10]
		block[2*i+81] = row[11]
		block[2*i+96] = row[12]
		block[2*i+97] = row[13]
		block[2*i+112] = row[14]
		block[2*i+113] = row[15]
	}
}
