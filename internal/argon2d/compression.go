// Package argon2d implements Argon2d (data-dependent mode) for RandomX.
// This file contains block compression functions using Blake2b mixing.
package argon2d

const (
	// BlockSize128 is the number of uint64 values in a Block (128 = 1024 bytes / 8)
	BlockSize128 = 128
)

// fillBlock performs Argon2 block compression using Blake2b rounds.
// It mixes prevBlock and refBlock into nextBlock using 8 rounds of
// Blake2b-style compression (column mixing + row mixing).
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
//  3. Apply permutation P (8 rounds of Blake2b with fBlaMka) to R
//  4. nextBlock = R XOR Q
//  5. If withXOR: nextBlock = nextBlock XOR oldNextBlock
func fillBlock(prevBlock, refBlock, nextBlock *Block, withXOR bool) {
	var R, Q Block

	// Step 1: R = refBlock XOR prevBlock
	R = *refBlock
	R.XOR(prevBlock)

	// Step 2: Q = R (save for feed-forward)
	Q = R

	// Step 3: Apply 8 rounds of Blake2b compression with fBlaMka (permutation P)
	// Each round consists of:
	//   - Column mixing (4 applications of G)
	//   - Row mixing (4 applications of G)
	for round := 0; round < 8; round++ {
		applyBlake2bRound(&R)
	}

	// Step 4: Feed-forward - R = R XOR Q
	R.XOR(&Q)

	// Step 5: If second+ pass, XOR with old nextBlock content
	if withXOR {
		oldNext := *nextBlock
		R.XOR(&oldNext)
	}

	// Step 5: Write result to nextBlock
	*nextBlock = R
}

// applyBlake2bRound applies one round of Blake2b compression to a Block.
// This processes the entire 1024-byte block (128 uint64 values) using
// the Blake2b G function in the standard Blake2b pattern.
//
// The block is processed as 8 groups of 16 uint64 values, with each
// group undergoing the full Blake2b round (column + diagonal mixing).
func applyBlake2bRound(block *Block) {
	// Process block in 16-value chunks
	for i := 0; i < BlockSize128; i += 16 {
		// Apply one Blake2b round to this 16-value group
		gRound(block[i : i+16])
	}
}
