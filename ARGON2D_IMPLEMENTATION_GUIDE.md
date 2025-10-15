# Argon2d Implementation Guide

**Date**: October 15, 2025  
**Purpose**: Step-by-step guide for implementing Argon2d in pure Go  
**Status**: Research Complete - Ready for Implementation  

---

## Overview

This document provides a detailed roadmap for implementing Argon2d (data-dependent mode) in pure Go, as required by the RandomX specification.

## Background

### Why Argon2d?

RandomX uses Argon2**d** for cache generation because:
- **Memory-hard**: Resistant to GPU/ASIC attacks
- **Data-dependent**: Provides better protection against time-memory tradeoffs
- **Fast**: Single-threaded performance optimized for sequential hashing

### Argon2 Variants Comparison

| Variant | Memory Access | Use Case | Available in Go? |
|---------|---------------|----------|------------------|
| Argon2**i** | Data-independent | Password hashing (side-channel resistant) | ✅ golang.org/x/crypto |
| Argon2**id** | Hybrid (first half i, second half d) | General password hashing | ✅ golang.org/x/crypto |
| Argon2**d** | Data-dependent | RandomX, faster hashing | ❌ NOT AVAILABLE |

## RandomX Argon2d Parameters

From `RandomX/src/configuration.h`:

```c
#define RANDOMX_ARGON_MEMORY       262144  // 256 MB memory usage
#define RANDOMX_ARGON_ITERATIONS   3       // 3 passes over memory
#define RANDOMX_ARGON_LANES        1       // Single-threaded (1 lane)
#define RANDOMX_ARGON_SALT         "RandomX\x03"  // Fixed salt
#define ARGON2_VERSION             0x13    // Argon2 version 1.3
```

**Output**: 256 KB cache (262144 bytes)

## Architecture

### Argon2d Memory Layout

```
Memory organized as blocks:
- Block size: 1024 bytes (128 uint64 values)
- Total blocks: (256 MB / 1024) = 262,144 blocks
- Cache output: First 256 KB = 256 blocks
```

### Core Algorithm Structure

```
1. Initial Hash (H0)
   - Blake2b hash of: password || salt || parameters
   
2. Memory Filling (fill_memory_blocks)
   - Initialize first block from H0
   - For each pass (3 iterations):
     - For each segment:
       - For each block:
         - Compute pseudo-random index using current block data (DATA-DEPENDENT!)
         - XOR current block with referenced block
         - Apply compression function (fill_block)

3. Finalization
   - XOR all blocks in lane
   - Extract first 256 KB as output
```

## Key Functions to Port

### 1. Blake2b Long Hash (`blake2b_long`)

**Purpose**: Generate variable-length output from Blake2b

**Location**: `RandomX/src/blake2/blake2b.c`

**Pseudocode**:
```
if outlen <= 64:
    return blake2b(input, outlen)
else:
    output = []
    V = blake2b(input || uint32_le(outlen), 64)
    output.append(V[:32])
    
    while len(output) < outlen:
        V = blake2b(V, 64)
        copy_len = min(64, outlen - len(output))
        output.append(V[:copy_len])
    
    return output
```

**Go Implementation Complexity**: Low - can use `golang.org/x/crypto/blake2b`

### 2. Compression Function (`fill_block`)

**Purpose**: Mix two 1024-byte blocks using Blake2b-based permutation

**Location**: `RandomX/src/argon2_core.c`

**Structure**:
```c
void fill_block(const block *prev_block,  // Previous block
                const block *ref_block,    // Referenced block (data-dependent!)
                block *next_block,         // Output block
                int with_xor) {            // XOR mode flag
    
    // Copy reference block
    block R = *ref_block;
    block Z = *prev_block;
    
    // XOR if needed
    if (with_xor) {
        xor_block(&R, next_block);
    }
    
    // Apply 8 rounds of column and row mixing using G function
    for (i = 0; i < 8; i++) {
        BLAKE2_ROUND_NOMSG(R.v[16*i+0], R.v[16*i+1], ...);
    }
    
    // Final XOR
    xor_block(&R, &Z);
    xor_block(&R, next_block);
}
```

**Key Insight**: Uses Blake2b's G function for mixing

### 3. Data-Dependent Indexing (`randomx_argon2_index_alpha`)

**Purpose**: Compute which block to reference (DATA-DEPENDENT!)

**Location**: `RandomX/src/argon2_core.c:106`

**This is the CRITICAL difference between Argon2i and Argon2d!**

```c
uint32_t randomx_argon2_index_alpha(
    const argon2_instance_t *instance,
    const argon2_position_t *position,
    uint32_t pseudo_rand,  // ← Derived from CURRENT BLOCK DATA
    int same_lane) {
    
    // Compute reference area size based on pass and segment
    uint32_t reference_area_size = ...;
    
    // Map pseudo_rand to block index (non-uniform distribution)
    uint64_t relative_position = pseudo_rand;
    relative_position = relative_position * relative_position >> 32;
    relative_position = reference_area_size - 1 - 
                       (reference_area_size * relative_position >> 32);
    
    return compute_absolute_position(position, relative_position);
}
```

**Key Insight**: `pseudo_rand` comes from the block data itself (data-dependent addressing)

### 4. Memory Filling (`fill_memory_blocks`)

**Purpose**: Main loop that fills memory with blocks

**Location**: `RandomX/src/argon2_core.c`

**Structure**:
```c
void fill_memory_blocks(argon2_instance_t *instance) {
    for (pass = 0; pass < instance->passes; pass++) {
        for (slice = 0; slice < ARGON2_SYNC_POINTS; slice++) {
            for (index = 0; index < instance->segment_length; index++) {
                // Get pseudo-random value from CURRENT block (data-dependent!)
                pseudo_rand = instance->memory[prev_offset].v[0];
                
                // Compute reference index using pseudo_rand
                ref_index = randomx_argon2_index_alpha(..., pseudo_rand, ...);
                
                // Mix blocks
                fill_block(&instance->memory[prev_offset],
                          &instance->memory[ref_index],
                          &instance->memory[curr_offset],
                          pass != 0);  // XOR mode after first pass
            }
        }
    }
}
```

## Implementation Plan

### Phase 1: Blake2b Utilities (Day 3, Morning - 2 hours)

**File**: `internal/argon2/blake2b_long.go`

```go
package argon2

import "golang.org/x/crypto/blake2b"

// Blake2bLong generates variable-length output using Blake2b.
// Implements the Argon2 specification for long outputs.
func Blake2bLong(input []byte, outlen uint32) []byte {
    if outlen <= 64 {
        h, _ := blake2b.New(int(outlen), nil)
        h.Write(input)
        return h.Sum(nil)
    }
    
    // Variable-length output (see algorithm above)
    // ... implementation
}
```

**Tests**: Validate against known Argon2 test vectors

### Phase 2: Block Structures (Day 3, Afternoon - 2 hours)

**File**: `internal/argon2/block.go`

```go
package argon2

const (
    BlockSize       = 1024  // bytes
    QWordsInBlock   = 128   // uint64 values
)

// Block represents a 1024-byte Argon2 memory block
type Block [QWordsInBlock]uint64

// XOR performs in-place XOR of two blocks
func (b *Block) XOR(other *Block) {
    for i := range b {
        b[i] ^= other[i]
    }
}

// Copy copies data from another block
func (b *Block) Copy(other *Block) {
    copy(b[:], other[:])
}

// Zero clears the block
func (b *Block) Zero() {
    for i := range b {
        b[i] = 0
    }
}
```

### Phase 3: Blake2b G Function (Day 4, Morning - 3 hours)

**File**: `internal/argon2/compression.go`

Port the Blake2b G mixing function:

```go
package argon2

// G is the Blake2b mixing function used in Argon2 compression
func G(a, b, c, d uint64) (uint64, uint64, uint64, uint64) {
    a += b
    d = rotr64(d^a, 32)
    c += d
    b = rotr64(b^c, 24)
    a += b
    d = rotr64(d^a, 16)
    c += d
    b = rotr64(b^c, 63)
    return a, b, c, d
}

func rotr64(x uint64, n uint) uint64 {
    return (x >> n) | (x << (64 - n))
}
```

### Phase 4: Block Compression (Day 4, Afternoon - 4 hours)

**File**: `internal/argon2/compression.go`

```go
// fillBlock mixes prev_block and ref_block into next_block
func fillBlock(prevBlock, refBlock, nextBlock *Block, withXOR bool) {
    var R, Z Block
    
    R.Copy(refBlock)
    Z.Copy(prevBlock)
    
    if withXOR {
        R.XOR(nextBlock)
    }
    
    // Apply Blake2b rounds (8 rounds of column + row mixing)
    // Each round processes 16 uint64 values
    for i := 0; i < 8; i++ {
        // Column mixing
        // ... apply G function
        
        // Row mixing  
        // ... apply G function
    }
    
    // Final XOR
    R.XOR(&Z)
    if withXOR {
        R.XOR(nextBlock)
    }
    nextBlock.Copy(&R)
}
```

### Phase 5: Data-Dependent Indexing (Day 5, Morning - 3 hours)

**File**: `internal/argon2/indexing.go`

```go
// Position tracks location in Argon2 memory
type Position struct {
    Pass    uint32
    Lane    uint32
    Slice   uint32
    Index   uint32
}

// indexAlpha computes the reference block index (DATA-DEPENDENT!)
func indexAlpha(memory []Block, pos *Position, pseudoRand uint64, 
                segmentLength, laneLength uint32) uint32 {
    
    // Compute reference area size
    var referenceAreaSize uint32
    if pos.Pass == 0 {
        if pos.Slice == 0 {
            referenceAreaSize = pos.Index - 1
        } else {
            referenceAreaSize = pos.Slice*segmentLength + pos.Index - 1
        }
    } else {
        referenceAreaSize = laneLength - segmentLength + pos.Index - 1
    }
    
    // Map pseudoRand to block index (quadratic distribution)
    relativePosition := pseudoRand & 0xFFFFFFFF
    relativePosition = (relativePosition * relativePosition) >> 32
    relativePosition = referenceAreaSize - 1 - 
                      ((uint64(referenceAreaSize) * relativePosition) >> 32)
    
    // Compute absolute position
    startPosition := uint32(0)
    if pos.Pass != 0 && pos.Slice != SyncPoints-1 {
        startPosition = (pos.Slice + 1) * segmentLength
    }
    
    return (startPosition + uint32(relativePosition)) % laneLength
}
```

### Phase 6: Memory Filling (Day 5, Afternoon - 4 hours)

**File**: `internal/argon2/core.go`

```go
// fillMemory is the main Argon2d algorithm
func fillMemory(memory []Block, passes, lanes, segmentLength uint32) {
    laneLength := uint32(len(memory)) / lanes
    
    for pass := uint32(0); pass < passes; pass++ {
        for slice := uint32(0); slice < SyncPoints; slice++ {
            for lane := uint32(0); lane < lanes; lane++ {
                for index := uint32(0); index < segmentLength; index++ {
                    pos := Position{Pass: pass, Lane: lane, Slice: slice, Index: index}
                    
                    // Current and previous block offsets
                    currOffset := lane*laneLength + slice*segmentLength + index
                    prevOffset := currOffset - 1
                    if index == 0 && slice == 0 {
                        prevOffset = lane*laneLength + laneLength - 1
                    }
                    
                    // Get pseudo-random value from CURRENT block (DATA-DEPENDENT!)
                    pseudoRand := memory[prevOffset][0]
                    
                    // Compute reference index
                    refIndex := indexAlpha(memory, &pos, pseudoRand, 
                                          segmentLength, laneLength)
                    refOffset := lane*laneLength + refIndex
                    
                    // Mix blocks
                    fillBlock(&memory[prevOffset], &memory[refOffset], 
                             &memory[currOffset], pass != 0)
                }
            }
        }
    }
}
```

### Phase 7: Public API (Day 6, Morning - 2 hours)

**File**: `internal/argon2/argon2d.go`

```go
package argon2

// Argon2d computes Argon2d hash with RandomX parameters
func Argon2d(password, salt []byte, timeCost, memoryCost uint32, 
             outputLen uint32) []byte {
    
    // 1. Initial hash H0
    h0 := initialHash(password, salt, timeCost, memoryCost, 1, outputLen)
    
    // 2. Allocate memory blocks
    memoryBlocks := (memoryCost + BlockSize - 1) / BlockSize
    memory := make([]Block, memoryBlocks)
    
    // 3. Initialize first blocks from H0
    initializeMemory(memory, h0)
    
    // 4. Fill memory (3 passes for RandomX)
    fillMemory(memory, timeCost, 1, uint32(len(memory)))
    
    // 5. Finalization - XOR all blocks
    final := &Block{}
    for i := range memory {
        final.XOR(&memory[i])
    }
    
    // 6. Extract output
    output := make([]byte, outputLen)
    for i := 0; i < int(outputLen)/8; i++ {
        binary.LittleEndian.PutUint64(output[i*8:], final[i])
    }
    
    return output
}

// Argon2dCache is the convenience function for RandomX cache generation
func Argon2dCache(seed []byte) []byte {
    return Argon2d(seed, []byte("RandomX\x03"), 3, 262144, 262144)
}
```

### Phase 8: Validation (Day 6, Afternoon - 4 hours)

**File**: `internal/argon2/argon2d_test.go`

```go
func TestArgon2dCache(t *testing.T) {
    // Test against RandomX reference values
    seed := []byte("test key 000")
    cache := Argon2dCache(seed)
    
    // Check length
    if len(cache) != 262144 {
        t.Errorf("cache length = %d, want 262144", len(cache))
    }
    
    // Check specific uint64 values from RandomX tests.cpp
    cacheU64 := make([]uint64, len(cache)/8)
    for i := range cacheU64 {
        cacheU64[i] = binary.LittleEndian.Uint64(cache[i*8:])
    }
    
    // From RandomX tests.cpp:
    // assert(cacheMemory[0] == 0x191e0e1d23c02186);
    if cacheU64[0] != 0x191e0e1d23c02186 {
        t.Errorf("cache[0] = 0x%x, want 0x191e0e1d23c02186", cacheU64[0])
    }
    
    // More validation points...
}
```

## Testing Strategy

### Unit Tests

1. **Blake2bLong**: Test against Argon2 spec test vectors
2. **Block Operations**: XOR, Copy, Zero
3. **G Function**: Validate against Blake2b test vectors
4. **fillBlock**: Compare with C reference for specific inputs
5. **indexAlpha**: Verify pseudo-random distribution
6. **Full Argon2d**: Validate against RandomX cache values

### Integration Tests

1. Replace placeholder in `internal/argon2.go` with real implementation
2. Run `TestOfficialVectors` - should PASS all 4 vectors
3. Run `TestCacheReferenceValues` - should PASS
4. Verify determinism (same input → same output)

### Performance Tests

```bash
go test -bench=BenchmarkArgon2dCache -benchmem
# Target: <5 seconds for cache generation
# Target: <1 GB peak memory usage
```

## Validation Criteria

Before marking complete:

- [ ] All unit tests pass
- [ ] `TestOfficialVectors` passes (all 4 vectors)
- [ ] Cache values match RandomX reference exactly
- [ ] Performance within 2x of C implementation
- [ ] No race conditions (`go test -race`)
- [ ] Memory doesn't leak
- [ ] Code reviewed for correctness
- [ ] Documentation complete

## Common Pitfalls

1. **Endianness**: Argon2 uses little-endian - use `binary.LittleEndian`
2. **Block Indexing**: Off-by-one errors are easy in segment/slice calculations
3. **XOR Mode**: First pass doesn't XOR, subsequent passes do
4. **Data Dependency**: Must use `memory[prevOffset][0]` for pseudoRand, not external source
5. **Memory Layout**: Blocks are 128 uint64, not 1024 bytes as array

## References

- **Argon2 Spec**: https://github.com/P-H-C/phc-winner-argon2/blob/master/argon2-specs.pdf
- **RandomX Argon2**: https://github.com/tevador/RandomX/tree/master/src (argon2_core.c)
- **Blake2b Spec**: https://blake2.net/blake2.pdf
- **Go Blake2b**: https://pkg.go.dev/golang.org/x/crypto/blake2b

## Estimated Timeline

| Phase | Task | Hours | Day |
|-------|------|-------|-----|
| 1 | Blake2b utilities | 2 | Day 3 AM |
| 2 | Block structures | 2 | Day 3 PM |
| 3 | G function | 3 | Day 4 AM |
| 4 | Block compression | 4 | Day 4 PM |
| 5 | Data-dependent indexing | 3 | Day 5 AM |
| 6 | Memory filling | 4 | Day 5 PM |
| 7 | Public API | 2 | Day 6 AM |
| 8 | Validation | 4 | Day 6 PM |
| **Total** | | **24 hours** | **3-4 days** |

## Success Definition

Implementation is complete when:
1. ✅ All unit tests pass
2. ✅ `TestOfficialVectors` shows all 4 vectors PASS
3. ✅ Hashes match RandomX reference implementation exactly
4. ✅ Code is clean, documented, and maintainable
5. ✅ No performance regressions
6. ✅ PLAN.md updated, warnings removed from README.md

---

**Status**: Ready to Begin Implementation  
**Next Step**: Phase 1 - Blake2b Utilities  
**Blocker**: None - all prerequisites met
