package argon2d

import (
"encoding/binary"
"encoding/hex"
"testing"
)

// TestDetailedTrace creates a detailed trace of Argon2d execution
// to help identify where our implementation diverges from RandomX reference.
func TestDetailedTrace(t *testing.T) {
if testing.Short() {
t.Skip("skipping detailed trace in short mode")
}

key := []byte("test key 000")
salt := []byte("RandomX\x03")

const (
lanes        = 1
tagLength    = 262144
memorySizeKB = 262144
timeCost     = 3
)

// Step 1: Compute H0
t.Log("=== Step 1: Computing initialHash (H0) ===")
h0 := initialHash(lanes, tagLength, memorySizeKB, timeCost, key, salt, nil, nil)
t.Logf("H0 (first 32 bytes): %s", hex.EncodeToString(h0[:32]))
t.Logf("H0 (last 32 bytes): %s", hex.EncodeToString(h0[32:]))

// Step 2: Allocate memory
t.Log("\n=== Step 2: Allocating memory ===")
numBlocks := memorySizeKB
memory := make([]Block, numBlocks)
t.Logf("Allocated %d blocks (%d KB)", numBlocks, numBlocks)

// Step 3: Initialize first two blocks
t.Log("\n=== Step 3: Initializing first two blocks ===")
initializeMemory(memory, lanes, h0)

// Check first block
t.Logf("Block[0][0] = 0x%016x", memory[0][0])
t.Logf("Block[0] (first 32 bytes): %s", hex.EncodeToString(memory[0].ToBytes()[:32]))

// Check second block
t.Logf("Block[1][0] = 0x%016x", memory[1][0])
t.Logf("Block[1] (first 32 bytes): %s", hex.EncodeToString(memory[1].ToBytes()[:32]))

// Step 4: Fill memory (just first pass, first few blocks)
t.Log("\n=== Step 4: Filling memory (first pass, first segment) ===")

laneLength := uint32(numBlocks / lanes)
segmentLength := uint32(laneLength / SyncPoints)

// Manually process first few blocks to trace
for i := uint32(0); i < 5 && i < segmentLength; i++ {
currentIndex := i

// Skip first two blocks
if currentIndex < 2 {
t.Logf("Block[%d]: Skipped (initialized from H0)", currentIndex)
continue
}

prevIndex := currentIndex - 1
pseudoRand := memory[prevIndex][0]

pos := Position{
Pass:  0,
Lane:  0,
Slice: 0,
Index: i,
}

refIndex := indexAlpha(&pos, pseudoRand, segmentLength, laneLength)

t.Logf("Block[%d]:", currentIndex)
t.Logf("  prev=%d, ref=%d", prevIndex, refIndex)
t.Logf("  pseudoRand=0x%016x", pseudoRand)

// Fill block
fillBlock(&memory[prevIndex], &memory[refIndex], &memory[currentIndex], false)

t.Logf("  result[0]=0x%016x", memory[currentIndex][0])
}

// Now run full fillMemory and check final results
t.Log("\n=== Step 5: Running full fillMemory ===")
fillMemory(memory, timeCost, lanes)

// Check specific values from RandomX reference
t.Log("\n=== Step 6: Checking against RandomX reference values ===")

cache0 := binary.LittleEndian.Uint64(memory[0].ToBytes()[0:8])
t.Logf("Cache[0] = 0x%016x", cache0)
t.Logf("Expected = 0x191e0e1d23c02186")

if cache0 != 0x191e0e1d23c02186 {
t.Errorf("Cache[0] mismatch!")
}

// Step 7: Finalize
t.Log("\n=== Step 7: Finalizing hash ===")
result := finalizeHash(memory, lanes, tagLength)
t.Logf("Final result length: %d bytes", len(result))
t.Logf("First 32 bytes: %s", hex.EncodeToString(result[:32]))
}

// TestBlake2bLongOutput tests Blake2bLong specifically
func TestBlake2bLongOutput(t *testing.T) {
key := []byte("test key 000")
salt := []byte("RandomX\x03")

// Create H0
h0 := initialHash(1, 262144, 262144, 3, key, salt, nil, nil)

// Test Blake2bLong for block 0
input := make([]byte, 72)
copy(input[0:64], h0[:])
binary.LittleEndian.PutUint32(input[64:68], 0) // block index 0
binary.LittleEndian.PutUint32(input[68:72], 0) // lane index 0

output := Blake2bLong(input, 1024)

t.Logf("Blake2bLong output (first 32 bytes): %s", hex.EncodeToString(output[:32]))
t.Logf("Blake2bLong output (last 32 bytes): %s", hex.EncodeToString(output[1024-32:]))

// Convert to Block and check first uint64
var block Block
block.FromBytes(output)
t.Logf("Block[0] as uint64 = 0x%016x", block[0])
}
