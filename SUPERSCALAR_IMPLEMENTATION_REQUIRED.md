# SuperscalarHash Implementation Required

## Problem Statement

The go-randomx implementation is missing the **SuperscalarHash algorithm**, which is critical for generating dataset items from the cache. This causes all hash computations to be incorrect.

## Root Cause Analysis

### What SuperscalarHash Does

SuperscalarHash is a complex sub-algorithm within RandomX that:

1. Generates pseudo-random instruction sequences ("superscalar programs") using Blake2Generator
2. Executes these programs on a virtual register file
3. Mixes cache data into the registers during execution
4. Produces deterministic 64-byte dataset items from 64-byte cache items

### Current Implementation Issues

**Bug Location**: `cache.go:60-66` and `dataset.go:85-113`

**Current (Wrong) Code**:
```go
// cache.go - Just returns raw cache bytes
func (c *cache) getItem(index uint32) []byte {
    offset := index * 64
    return c.data[offset : offset+64]
}

// dataset.go - Uses simple XOR mixing instead of SuperscalarHash
func (ds *dataset) generateItem(c *cache, itemNumber uint64, output []byte) {
    // Simplified mixing - NOT the RandomX algorithm!
    for i := 0; i < 8; i++ {
        cacheIndex := uint32(registers[0] % cacheItems)
        cacheItem := c.getItem(cacheIndex)
        // ... simple XOR mixing ...
    }
}
```

**What Should Happen** (from RandomX C++ reference):

```cpp
// Cache initialization generates 8 superscalar programs
Blake2Generator gen(key, keySize);
for (int i = 0; i < RANDOMX_CACHE_ACCESSES; ++i) {
    generateSuperscalar(cache->programs[i], gen);
}

// Dataset item generation executes these programs
void initDatasetItem(randomx_cache* cache, uint8_t* out, uint64_t itemNumber) {
    int_reg_t rl[8];
    uint64_t registerValue = itemNumber;
    
    // Initialize registers with constants
    rl[0] = (itemNumber + 1) * superscalarMul0;
    rl[1] = rl[0] ^ superscalarAdd1;
    // ... etc for all 8 registers
    
    // Execute 8 superscalar programs
    for (unsigned i = 0; i < RANDOMX_CACHE_ACCESSES; ++i) {
        mixBlock = getMixBlock(registerValue, cache->memory);
        executeSuperscalar(rl, cache->programs[i]);
        
        // XOR cache block into registers
        for (unsigned q = 0; q < 8; ++q)
            rl[q] ^= load64(mixBlock + 8 * q);
            
        registerValue = rl[prog.getAddressRegister()];
    }
    
    memcpy(out, &rl, 64);
}
```

## Impact

**ALL hash computations are wrong** because:
- Light mode: VM mixes raw cache bytes instead of computed dataset items
- Fast mode: Dataset is pre-generated with wrong algorithm
- Test vectors: 0/4 passing (100% failure rate)

**Why components seem to work independently**:
- Argon2d cache generation: ✅ Correct (verified against reference)
- AesGenerator1R/4R: ✅ Correct (deterministic, matches spec)
- VM execution: ✅ Correct (instructions execute properly)
- **Dataset item generation**: ❌ WRONG (missing SuperscalarHash)

## Implementation Requirements

### 1. Blake2Generator

A PRNG that generates deterministic pseudo-random data using Blake2b:

```go
type blake2Generator struct {
    data [64]byte  // Current Blake2b output
    pos  int       // Position in current output
}

func newBlake2Generator(seed []byte) *blake2Generator
func (g *blake2Generator) getByte() byte
func (g *blake2Generator) getUint32() uint32
```

### 2. Superscalar Instruction Types

```go
const (
    ISUB_R = iota      // r[dst] -= r[src]
    IXOR_R             // r[dst] ^= r[src]
    IADD_RS            // r[dst] += r[src] << shift
    IMUL_R             // r[dst] *= r[src]
    IROR_C             // r[dst] = rotate_right(r[dst], imm)
    IADD_C7            // r[dst] += imm (7-byte imm)
    IXOR_C7            // r[dst] ^= imm (7-byte imm)
    IADD_C8            // r[dst] += imm (8-byte imm)
    IXOR_C8            // r[dst] ^= imm (8-byte imm)
    IADD_C9            // r[dst] += imm (9-byte imm)
    IXOR_C9            // r[dst] ^= imm (9-byte imm)
    IMULH_R            // r[dst] = (r[dst] * r[src]) >> 64
    ISMULH_R           // r[dst] = (int64(r[dst]) * int64(r[src])) >> 64
    IMUL_RCP           // r[dst] *= reciprocal(imm)
)
```

### 3. Superscalar Program Structure

```go
type superscalarInstruction struct {
    opcode  uint8
    dst     uint8
    src     uint8
    mod     uint8
    imm32   uint32
}

type superscalarProgram struct {
    instructions []superscalarInstruction
    addressReg   uint8  // Which register determines next cache address
}
```

### 4. Program Generation

```go
// Generate one superscalar program using Blake2Generator
func generateSuperscalarProgram(gen *blake2Generator) *superscalarProgram {
    // Complex algorithm:
    // - Generate instructions with dependency tracking
    // - Ensure instruction-level parallelism
    // - Respect execution port constraints
    // - Generate 3-60 instructions per program
    // - Select address register for cache access
}
```

### 5. Program Execution

```go
// Execute superscalar program on register file
func executeSuperscalarProgram(registers *[8]uint64, prog *superscalarProgram) {
    for _, instr := range prog.instructions {
        switch instr.opcode {
        case ISUB_R:
            registers[instr.dst] -= registers[instr.src]
        case IXOR_R:
            registers[instr.dst] ^= registers[instr.src]
        case IMUL_R:
            registers[instr.dst] *= registers[instr.src]
        // ... all 14 instruction types
        }
    }
}
```

### 6. Constants

```go
const (
    cacheAccesses = 8
    
    superscalarMul0 = 6364136223846793005
    superscalarAdd1 = 9298411001130361340
    superscalarAdd2 = 12065312585734608966
    superscalarAdd3 = 9306329213124626780
    superscalarAdd4 = 5281919268842080866
    superscalarAdd5 = 10536153434571861004
    superscalarAdd6 = 3398623926847679864
    superscalarAdd7 = 9549104520008361294
)
```

## Implementation Complexity

**Estimated Complexity**: HIGH
- ~500-800 lines of code
- Complex dependency tracking for program generation
- Must match C++ reference implementation exactly
- Subtle bugs will cause hash mismatches

**Key Challenges**:
1. **Program generation**: Must generate valid instruction sequences with proper dependencies
2. **Port constraints**: Instructions have execution port requirements (p0, p1, p5, p015, etc.)
3. **Cycle counting**: Track execution latency and throughput
4. **Reciprocal calculation**: IMUL_RCP needs pre-computed reciprocals
5. **Exact matching**: Any deviation from C++ causes wrong hashes

## Recommended Approach

### Option A: Full Implementation (Correct but Complex)

1. Study RandomX specification Section 3 (SuperscalarHash)
2. Port C++ `superscalar.cpp` to Go line-by-line
3. Implement Blake2Generator
4. Implement program generator with dependency tracking
5. Implement program executor
6. Validate each component against C++ intermediate values
7. Integrate into cache/dataset

**Timeline**: 2-4 days for careful implementation and testing

### Option B: Simplified Reference Port (Pragmatic)

1. Port only the essential parts from C++ reference
2. Use simpler dependency tracking (may be less efficient)
3. Skip JIT compilation (interpreted only)
4. Focus on correctness over performance

**Timeline**: 1-2 days

### Option C: Call C++ Library (Fast but defeats purpose)

Use CGo to call RandomX C++ library for dataset generation.

**Pros**: Works immediately, matches reference exactly
**Cons**: Defeats "pure Go" goal, adds CGo dependency

## Validation Strategy

Once implemented:

1. **Component tests**: Validate Blake2Generator output matches C++
2. **Program tests**: Generate programs and compare with C++ output
3. **Execution tests**: Execute programs and verify register states
4. **Dataset tests**: Generate dataset items and compare with C++
5. **Integration tests**: Run full RandomX and validate against test vectors

## Files to Modify

1. Create `blake2_generator.go` - Blake2 PRNG
2. Create `superscalar.go` - Superscalar program generation and execution
3. Create `superscalar_program.go` - Program structure
4. Modify `cache.go`:
   - Add `programs [8]superscalarProgram` field
   - Generate programs in `newCache()`
   - Implement `getItem()` to compute dataset items
5. Modify `dataset.go`:
   - Fix `generateItem()` to use superscalar programs

## Expected Outcome

After proper SuperscalarHash implementation:
- ✅ All 4 test vectors should pass
- ✅ Hash output should match C++ reference exactly
- ✅ Light mode and fast mode both work correctly
- ✅ Compatible with Monero network

## References

- RandomX specification: https://github.com/tevador/RandomX/blob/master/doc/specs.md#3-superscalarhash
- C++ reference: `/tmp/RandomX/src/superscalar.cpp`
- C++ reference: `/tmp/RandomX/src/dataset.cpp`
