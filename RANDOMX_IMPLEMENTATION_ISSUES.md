# RandomX Implementation Issues - Detailed Analysis

## Summary

The current go-randomx implementation has **fundamental algorithmic errors** in the RandomX VM execution. While the Argon2d cache generation is correct, the VM does not follow the RandomX specification.

## Verified Correct Components

✅ **Argon2d Cache Generation**
- Cache[0] = 0x191e0e1d23c02186 (matches reference)
- All 256 MB cache data correctly generated
- Salt "RandomX\x03" correctly used

✅ **AES Generators** (newly implemented)
- AesGenerator1R with correct keys
- AesGenerator4R with correct keys
- Proper encrypt/decrypt per column

## Critical Issues in Current Implementation

### Issue #1: Wrong Program Generation Source

**Current Code (vm.go:50):**
```go
prog := generateProgram(input)  // WRONG: Uses same input for all 8 iterations
```

**RandomX Spec:**
- Programs must be generated from AesGenerator4R
- Generator state updated between programs using RegisterFile hash
- Each of 8 programs is DIFFERENT

**Fix Required:**
- Use AesGenerator4R to generate 128 + 2048 bytes per program
- Update generator state = Hash512(RegisterFile) between programs

### Issue #2: Missing Configuration Data

**Current Code:**
- No configuration data parsing
- Registers initialized from Blake2b(input) only
- Missing register masks, address registers, dataset offset

**RandomX Spec (Section 4.5):**
- First 128 bytes from generator = configuration data
- Configures: a0-a3 registers, ma/mx registers, address registers, E register masks
- Next 2048 bytes = 256 instructions (8 bytes each)

**Fix Required:**
- Parse 128 bytes of config data
- Initialize registers per Table 4.5.1
- Then read 2048 bytes for program instructions

### Issue #3: Wrong Iteration Count

**Current Code (vm.go:47-48):**
```go
const iterations = 8
for i := 0; i < iterations; i++ {
```

This runs 8 iterations total.

**RandomX Spec:**
- `RANDOMX_PROGRAM_COUNT` = 8 (number of programs)
- `RANDOMX_PROGRAM_ITERATIONS` = 2048 (iterations per program)
- Total: 8 programs × 2048 iterations = 16,384 instruction loop iterations

**Fix Required:**
```go
const programCount = 8
const programIterations = 2048

for progNum := 0; progNum < programCount; progNum++ {
    // Generate and load program
    // Execute program loop 2048 times
    for iter := 0; iter < programIterations; iter++ {
        // Execute all 256 instructions
    }
    // Hash RegisterFile and update generator
}
```

### Issue #4: Incomplete VM Execution Loop

**Current Code:**
- Simple program execution
- Basic dataset mixing
- Missing many steps

**RandomX Spec (Section 4.6.2 - Loop Execution):**

Each of 2048 iterations must:
1. XOR readReg0/readReg1, update spAddr0/spAddr1
2. Read 64 bytes from Scratchpad[spAddr0], XOR with r0-r7
3. Read 64 bytes from Scratchpad[spAddr1], init f0-f3, e0-e3
4. Execute all 256 instructions in program
5. XOR mx with readReg2/readReg3
6. Prefetch Dataset item
7. Read Dataset item, XOR with r0-r7
8. Swap mx and ma
9. Write r0-r7 to Scratchpad[spAddr1]
10. XOR f0-f3 with e0-e3
11. Write f0-f3 to Scratchpad[spAddr0]
12. Reset spAddr0 and spAddr1

**Fix Required:**
- Implement full 12-step loop per iteration
- Track spAddr0, spAddr1 properly
- Implement proper scratchpad reads/writes
- Implement dataset prefetch and read

### Issue #5: Missing Scratchpad Initialization

**Current Code:**
- fillScratchpad() uses AES with register-derived keys
- No use of AesGenerator1R

**RandomX Spec (Section 2, Steps 3-4):**
- gen1 = AesGenerator1R(Hash512(input))
- Fill entire scratchpad (2 MB) with gen1 output

**Fix Required:**
- Create AesGenerator1R from Hash512(input)
- Fill scratchpad with generator output
- Then create gen4 from gen1.state

### Issue #6: Wrong Finalization

**Current Code (vm.go:132-145):**
```go
func (vm *virtualMachine) finalize() [32]byte {
    // Mix with memory
    for i := 0; i < 8; i++ {
        vm.reg[i] ^= vm.readMemory(uint32(i * 8))
    }
    // Hash registers
    output := make([]byte, 64)
    for i := 0; i < 8; i++ {
        binary.LittleEndian.PutUint64(output[i*8:i*8+8], vm.reg[i])
    }
    return internal.Blake2b256(output)
}
```

**RandomX Spec (Section 2, Step 12):**
- A = AesHash1R(Scratchpad)  // Hash entire 2MB scratchpad
- R = Hash256(A || RegisterFile)  // Concatenate and hash

**Fix Required:**
- Implement AesHash1R to fingerprint scratchpad
- Concatenate AesHash1R output (64 bytes) with RegisterFile (256 bytes)
- Hash with Blake2b-256

## Implementation Plan

### Phase 1: Implement Missing Components
1. ✅ AesGenerator1R
2. ✅ AesGenerator4R  
3. ⏳ AesHash1R
4. ⏳ RegisterFile serialization

### Phase 2: Rewrite VM Initialization
1. Initialize with AesGenerator1R
2. Fill scratchpad from generator
3. Create AesGenerator4R from gen1.state
4. Parse configuration data (128 bytes)
5. Initialize all registers properly

### Phase 3: Rewrite VM Execution
1. Outer loop: 8 programs
2. Generate each program from gen4 (128 + 2048 bytes)
3. Inner loop: 2048 iterations per program
4. Implement full 12-step execution per iteration
5. Update gen4.state between programs

### Phase 4: Fix Finalization
1. Implement AesHash1R
2. Hash entire scratchpad
3. Concatenate with RegisterFile
4. Final Blake2b-256 hash

### Phase 5: Testing & Validation
1. Test each component individually
2. Run official test vectors
3. Debug any remaining mismatches
4. Performance optimization

## Estimated Effort

- **Phase 1:** 2 hours (mostly done)
- **Phase 2:** 3 hours
- **Phase 3:** 5 hours (most complex)
- **Phase 4:** 2 hours
- **Phase 5:** 3 hours

**Total:** ~15 hours of focused implementation

## Next Steps

1. Implement AesHash1R (Section 3.4 of spec)
2. Create new vm_new.go with correct algorithm
3. Gradually replace vm.go functions
4. Test incrementally
5. Validate against test vectors

## References

- RandomX Spec: https://github.com/tevador/RandomX/blob/master/doc/specs.md
- Test Vectors: testdata/randomx_vectors.json
- Reference Implementation: https://github.com/tevador/RandomX
