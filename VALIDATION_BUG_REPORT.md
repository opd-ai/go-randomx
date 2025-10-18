# RandomX Implementation Bug Report - Systematic Validation

**Date**: October 18, 2025  
**Status**: 🔍 IN PROGRESS  
**Task**: Validate and debug Go implementation of RandomX against C++ reference

## Executive Summary

The go-randomx implementation has a **correct Argon2d cache generation** that matches the C++ reference implementation byte-for-byte. However, the **RandomX VM execution** has multiple critical bugs that cause hash mismatches with the reference implementation.

**Testing Status**:
- ✅ Argon2d cache generation: **PASSES** (matches reference)
- ✅ AesGenerator1R: **IMPLEMENTED AND WORKING**
- ✅ AesGenerator4R: **IMPLEMENTED AND WORKING**
- ❌ RandomX VM: **FAILING** (all 4 test vectors produce wrong hashes)

**Root Causes Identified**: 6 major bugs in VM implementation

---

## Testing Results

### Test Vector Results

Running `TestOfficialVectors` against 4 official RandomX test vectors:

```
Test: basic_test_1
Key:      "test key 000"
Input:    "This is a test"
Expected: 639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f
Got:      10c3fd4f67097c15465d10ad8ac2e30cfb07762421bd8fd9eb4209c717aa8649
Result:   ❌ MISMATCH (every byte wrong)

Test: basic_test_2
Key:      "test key 000"
Input:    "Lorem ipsum dolor sit amet"
Expected: 300a0adb47603dedb42228ccb2b211104f4da45af709cd7547cd049e9489c969
Got:      f5e75c7494d37e585a4a137e94f23ead6834235d4ff292f4103a87973c5c7512
Result:   ❌ MISMATCH

Test: basic_test_3
Key:      "test key 000"
Input:    "sed do eiusmod tempor incididunt ut labore et dolore magna aliqua\n"
Expected: c36d4ed4191e617309867ed66a443be4075014e2b061bcdaf9ce7b721d2b77a8
Got:      ab19bf3a9bd3cbaaf45ae07bda011e846ae98e13cd1502db4e81fb53895bddd7
Result:   ❌ MISMATCH

Test: different_key
Key:      "test key 001"
Input:    "sed do eiusmod tempor incididunt ut labore et dolore magna aliqua\n"
Expected: e9ff4503201c0c2cca26d285c93ae883f9b1d30c9eb240b820756f2d5a7905fc
Got:      da648ff0e6f8b3721529fc0af572c6434c8eac705c544f4b91388976a55685d9
Result:   ❌ MISMATCH
```

**Pass Rate**: 0/4 (0%)

### Component Validation

```
✓ Argon2d cache generation: Cache[0] = 0x191e0e1d23c02186 (matches reference)
✓ AesGenerator1R: Deterministic output, working correctly
✓ AesGenerator4R: Deterministic output, working correctly
✗ VM initialization: Using wrong algorithm
✗ Program generation: Using wrong source
✗ VM execution: Wrong iteration count and structure
✗ Finalization: Wrong algorithm
```

---

## Bugs Discovered

### BUG #1: VM Initialization - Wrong Scratchpad Filling Algorithm

**Location**: `vm.go:82-106` - `fillScratchpad()`  
**Severity**: Critical  
**Category**: Algorithm Implementation Error

**Description**:
The VM initialization uses a custom AES-based algorithm to fill the scratchpad, rather than using the required AesGenerator1R as specified in the RandomX specification.

**Current (WRONG) Code**:
```go
func (vm *virtualMachine) fillScratchpad() {
    // Use registers as AES keys
    key := make([]byte, 32)
    for i := 0; i < 4; i++ {
        binary.LittleEndian.PutUint64(key[i*8:], vm.reg[i])
    }
    
    // Fill memory in blocks
    aesEnc, err := internal.NewAESEncryptor(key[:16])
    if err != nil {
        return
    }
    
    block := make([]byte, 16)
    for i := 0; i < scratchpadL3Size; i += 16 {
        binary.LittleEndian.PutUint64(block[0:8], uint64(i))
        binary.LittleEndian.PutUint64(block[8:16], uint64(i+8))
        aesEnc.Encrypt(vm.mem[i:i+16], block)
    }
}
```

**RandomX Specification** (Section 2, Steps 3-4):
```
3. Create AesGenerator1R initialized with Blake2b-512(input)
4. Fill entire scratchpad (2 MB) by reading from generator
```

**Root Cause**:
The implementation doesn't follow the RandomX specification. It creates a custom AES encryption scheme instead of using the standardized AesGenerator1R.

**Affected Test Cases**: All 4 test vectors

**Go-Specific Issue**:
None - this is a pure algorithm error independent of language.

**Required Fix**:
```go
func (vm *virtualMachine) initialize(input []byte) {
    // Hash input to get initial state
    hash := internal.Blake2b512(input)
    
    // Create AesGenerator1R from hash
    gen1, err := newAesGenerator1R(hash[:])
    if err != nil {
        panic("failed to create AesGenerator1R")
    }
    
    // Fill scratchpad from generator
    gen1.getBytes(vm.mem)
    
    // Create AesGenerator4R from gen1 state for program generation
    vm.gen4, err = newAesGenerator4R(gen1.state[:])
    if err != nil {
        panic("failed to create AesGenerator4R")
    }
}
```

---

### BUG #2: Program Generation - Wrong Source Data

**Location**: `vm.go:50` and `program.go:33-52` - `generateProgram()`  
**Severity**: Critical  
**Category**: Algorithm Implementation Error

**Description**:
Programs are generated from Blake2b hashing of the input, rather than from AesGenerator4R as required. Additionally, the same program is generated for all 8 iterations.

**Current (WRONG) Code**:
```go
// vm.go:50
for i := 0; i < iterations; i++ {
    prog := generateProgram(input)  // WRONG: Uses same input every time!
    // ...
}

// program.go:33
func generateProgram(input []byte) *program {
    p := &program{}
    
    // Generate program entropy using Blake2b
    entropy := hashProgramEntropy(input)  // WRONG: Should use AesGenerator4R
    
    // Decode instructions from entropy
    for i := 0; i < programLength; i++ {
        offset := i * 8
        if offset+8 > len(entropy) {
            entropy = hashProgramEntropy(entropy)
            offset = 0
        }
        p.instructions[i] = decodeInstruction(entropy[offset : offset+8])
    }
    
    return p
}
```

**RandomX Specification** (Section 4.5):
```
1. Read 128 bytes from AesGenerator4R for configuration
2. Read 2048 bytes from AesGenerator4R for program
3. Parse 256 instructions (8 bytes each)
4. After each program execution, update gen4.state = Hash512(RegisterFile)
```

**Root Cause**:
- Programs generated from wrong source (Blake2b instead of AesGenerator4R)
- Same program used for all 8 iterations (not updated)
- No configuration data parsing
- No generator state updates between programs

**Affected Test Cases**: All 4 test vectors

**Go-Specific Issue**:
None - this is an algorithm error.

**Required Fix**:
```go
func (vm *virtualMachine) generateProgram() *program {
    p := &program{}
    
    // Read configuration data (128 bytes)
    configData := make([]byte, 128)
    vm.gen4.getBytes(configData)
    
    // Parse configuration to set up registers, address registers, etc.
    // (Implementation details per RandomX spec Table 4.5.1)
    vm.parseConfiguration(configData)
    
    // Read program data (2048 bytes = 256 instructions)
    programData := make([]byte, 2048)
    vm.gen4.getBytes(programData)
    
    // Decode instructions
    for i := 0; i < programLength; i++ {
        p.instructions[i] = decodeInstruction(programData[i*8 : i*8+8])
    }
    
    return p
}
```

---

### BUG #3: Iteration Count - Wrong Number of Executions

**Location**: `vm.go:47-57` - `run()`  
**Severity**: Critical  
**Category**: Algorithm Implementation Error

**Description**:
The VM executes only 8 iterations total, but RandomX requires 8 programs × 2048 iterations per program = 16,384 loop iterations.

**Current (WRONG) Code**:
```go
const iterations = 8
for i := 0; i < iterations; i++ {
    prog := generateProgram(input)
    prog.execute(vm)
    vm.mixDataset()
}
```

**RandomX Specification**:
```
RANDOMX_PROGRAM_COUNT = 8
RANDOMX_PROGRAM_ITERATIONS = 2048

FOR program_num = 0 TO 7:
    Generate program from AesGenerator4R
    FOR iteration = 0 TO 2047:
        Execute program (all 256 instructions)
        Mix dataset
        Update scratchpad
    Update gen4.state = Hash512(RegisterFile)
```

**Root Cause**:
Misunderstanding of the spec - confused "8 programs" with "8 total iterations". Each program should run 2048 times.

**Affected Test Cases**: All 4 test vectors

**Impact**:
- Executes only 8 iterations instead of 16,384
- Missing ~99.95% of required computation
- Completely wrong hash output

**Go-Specific Issue**:
None - this is an algorithm error.

**Required Fix**:
```go
const (
    programCount      = 8
    programIterations = 2048
)

for progNum := 0; progNum < programCount; progNum++ {
    // Generate new program
    prog := vm.generateProgram()
    
    // Execute program 2048 times
    for iter := 0; iter < programIterations; iter++ {
        vm.executeIteration(prog)
    }
    
    // Update generator state
    regData := vm.serializeRegisters()
    newState := internal.Blake2b512(regData)
    vm.gen4.setState(newState[:])
}
```

---

### BUG #4: VM Execution Loop - Missing Steps

**Location**: `vm.go:42-61` - `run()` and `program.go:89-93` - `execute()`  
**Severity**: Critical  
**Category**: Algorithm Implementation Error

**Description**:
The current execution loop is too simplistic. RandomX requires a 12-step execution process per iteration (see Section 4.6.2), but the current code only does basic instruction execution.

**Current (WRONG) Code**:
```go
// program.go:89
func (p *program) execute(vm *virtualMachine) {
    for i := 0; i < programLength; i++ {
        vm.executeInstruction(&p.instructions[i])
    }
}

// vm.go:109
func (vm *virtualMachine) mixDataset() {
    // Simple dataset mixing - incomplete
    index := vm.reg[0] % datasetItems
    itemData := vm.ds.getItem(index)
    for i := 0; i < 8 && i*8 < len(itemData); i++ {
        val := binary.LittleEndian.Uint64(itemData[i*8 : i*8+8])
        vm.reg[i] ^= val
    }
}
```

**RandomX Specification** (Section 4.6.2 - Loop Execution):

Each of the 2048 iterations must perform these steps:
```
1. XOR readReg0/readReg1, update spAddr0/spAddr1
2. Read 64 bytes from Scratchpad[spAddr0], XOR with r0-r7
3. Read 64 bytes from Scratchpad[spAddr1], init f0-f3, e0-e3
4. Execute all 256 instructions in program
5. XOR mx with readReg2/readReg3
6. Prefetch dataset item at (ma % datasetItems)
7. Read dataset item at (mx % datasetItems), XOR with r0-r7
8. Swap mx and ma
9. Write r0-r7 to Scratchpad[spAddr1]
10. XOR f0-f3 with e0-e3
11. Write f0-f3 to Scratchpad[spAddr0]
12. Reset spAddr0 = (spAddr0 + 16) & CacheLineAlignMask
```

**Root Cause**:
The implementation has only a basic instruction loop. It's missing:
- Proper scratchpad address management (spAddr0, spAddr1)
- Pre-execution register initialization from scratchpad
- Post-execution scratchpad writes
- Proper dataset mixing sequence
- Memory address calculations

**Affected Test Cases**: All 4 test vectors

**Go-Specific Issue**:
None - this is an algorithm error.

**Required Fix**:
```go
func (vm *virtualMachine) executeIteration(prog *program) {
    // Step 1: Update scratchpad addresses
    vm.spAddr0 ^= vm.reg[vm.config.readReg0]
    vm.spAddr1 ^= vm.reg[vm.config.readReg1]
    
    // Step 2: Read from scratchpad → r0-r7
    for i := 0; i < 8; i++ {
        vm.reg[i] ^= vm.readMemory(vm.spAddr0 + uint32(i*8))
    }
    
    // Step 3: Read from scratchpad → f0-f3, e0-e3
    for i := 0; i < 4; i++ {
        vm.regF[i] = vm.readFloat(vm.spAddr1 + uint32(i*8))
        vm.regE[i] = vm.readFloat(vm.spAddr1 + uint32(32 + i*8))
    }
    
    // Step 4: Execute all instructions
    for i := 0; i < programLength; i++ {
        vm.executeInstruction(&prog.instructions[i])
    }
    
    // Step 5-12: Dataset mixing and scratchpad updates
    vm.finishIteration()
}
```

---

### BUG #5: Missing Configuration Data Parsing

**Location**: N/A - Not implemented  
**Severity**: Critical  
**Category**: Missing Implementation

**Description**:
The first 128 bytes from AesGenerator4R contain configuration data that must be parsed to initialize various VM parameters. This is completely missing.

**RandomX Specification** (Section 4.5, Table 4.5.1):

Configuration data structure:
```
Bytes 0-31:   readReg0, readReg1, readReg2, readReg3
Bytes 32-63:  E register masks
Bytes 64-127: Address register initialization
```

**Impact**:
- Wrong register initialization
- Wrong memory addressing
- Wrong floating-point mask values

**Required Fix**:
```go
type vmConfig struct {
    readReg0     uint32
    readReg1     uint32
    readReg2     uint32
    readReg3     uint32
    eMask        [4]uint64
    addressRegs  [8]uint64
    datasetOff   uint64
}

func (vm *virtualMachine) parseConfiguration(data []byte) {
    // Parse according to RandomX spec Table 4.5.1
    vm.config.readReg0 = binary.LittleEndian.Uint32(data[0:4]) % 8
    vm.config.readReg1 = binary.LittleEndian.Uint32(data[4:8]) % 8
    // ... etc
}
```

---

### BUG #6: Wrong Finalization Algorithm

**Location**: `vm.go:133-146` - `finalize()`  
**Severity**: Critical  
**Category**: Algorithm Implementation Error

**Description**:
The finalization simply XORs registers with memory and hashes. RandomX requires using AesHash1R to fingerprint the entire scratchpad.

**Current (WRONG) Code**:
```go
func (vm *virtualMachine) finalize() [32]byte {
    // Mix final register state
    for i := 0; i < 8; i++ {
        vm.reg[i] ^= vm.readMemory(uint32(i * 8))
    }
    
    // Hash register file to produce output
    output := make([]byte, 64)
    for i := 0; i < 8; i++ {
        binary.LittleEndian.PutUint64(output[i*8:i*8+8], vm.reg[i])
    }
    
    return internal.Blake2b256(output)
}
```

**RandomX Specification** (Section 2, Step 12):
```
1. A = AesHash1R(Scratchpad)  // Hash entire 2MB scratchpad
2. Serialize RegisterFile (256 bytes)
3. result = Blake2b-256(A || RegisterFile)
```

**Root Cause**:
Missing AesHash1R implementation and wrong finalization structure.

**Impact**:
Even if all other steps were correct, this would produce wrong output.

**Required Fix**:
```go
// First, implement AesHash1R (similar to AesGenerator1R but hashes data)
type aesHash1R struct {
    // Similar structure to aesGenerator1R
    // Processes scratchpad in chunks and produces 64-byte hash
}

func (vm *virtualMachine) finalize() [32]byte {
    // Step 1: Hash scratchpad with AesHash1R
    hasher := newAesHash1R()
    scratchpadHash := hasher.hash(vm.mem)  // 64 bytes
    
    // Step 2: Serialize register file (256 bytes = 8 registers × 8 bytes + floats)
    regData := make([]byte, 256)
    // ... serialize all VM state
    
    // Step 3: Concatenate and hash
    combined := append(scratchpadHash[:], regData...)
    return internal.Blake2b256(combined)
}
```

---

## Summary of Required Changes

### Phase 1: Implement Missing Components

1. **AesHash1R** - New implementation required
   - Similar to AesGenerator1R but for hashing
   - Processes 2 MB scratchpad
   - Produces 64-byte fingerprint

2. **VM Configuration Parsing** - New function required
   - Parse 128 bytes from AesGenerator4R
   - Initialize readReg0-3, E masks, address registers

3. **Extended VM State** - Add fields to `virtualMachine`
   ```go
   type virtualMachine struct {
       // Existing fields...
       reg  [8]uint64
       regF [4]float64  // NEW: Floating-point registers
       regE [4]float64  // NEW: E registers
       mem  []byte
       
       // NEW fields:
       gen4     *aesGenerator4R
       config   vmConfig
       spAddr0  uint32
       spAddr1  uint32
       ma       uint64
       mx       uint64
   }
   ```

### Phase 2: Rewrite VM Execution

1. **Fix `vm.initialize()`**
   - Use AesGenerator1R for scratchpad
   - Create AesGenerator4R for programs
   - Remove custom AES filling

2. **Fix `vm.run()`**
   - Implement 8 programs × 2048 iterations structure
   - Update generator between programs

3. **Implement `vm.executeIteration()`**
   - Full 12-step execution loop
   - Proper scratchpad addressing
   - Dataset mixing

4. **Fix `vm.finalize()`**
   - Implement AesHash1R
   - Proper concatenation and hashing

### Phase 3: Validation

Run tests to verify:
```bash
go test -v -run TestOfficialVectors
# Should show: PASS (4/4 vectors)
```

---

## Testing Strategy

### Unit Tests

1. Test AesHash1R independently
2. Test configuration parsing
3. Test iteration loop structure
4. Test finalization

### Integration Tests

1. Run with single program, single iteration
2. Gradually increase to full 8×2048
3. Compare intermediate states with C++ reference (if available)

### Final Validation

1. All 4 test vectors must pass
2. Determinism check (same input → same output)
3. Race detector must be clean
4. Performance within 2-5x of C++ (acceptable for pure Go)

---

## Estimated Effort

- **Phase 1**: Implement missing components - 4-6 hours
- **Phase 2**: Rewrite VM execution - 6-8 hours
- **Phase 3**: Testing and debugging - 4-6 hours

**Total**: 14-20 hours of focused development

---

## Go-Specific Considerations

### Memory Management

```go
// Use memory pools for VM instances
var vmPool = sync.Pool{
    New: func() interface{} {
        return &virtualMachine{
            mem: make([]byte, scratchpadL3Size),
        }
    },
}
```

### Floating Point

```go
// Go's float64 is IEEE-754 compliant
// Use math.Float64bits and math.Float64frombits for conversions
f := math.Float64frombits(vm.reg[i])
vm.reg[i] = math.Float64bits(f * 2.0)
```

### Race Conditions

- Current `Hash()` is safe (uses pooled VMs)
- Maintain thread-safety in fixes
- Run with `-race` flag

### Performance

- Go will be 2-5x slower than C++ (expected)
- Profile with `pprof` after correctness achieved
- Consider assembly for critical paths (future optimization)

---

## Next Steps

1. ✅ Create comprehensive test suite
2. ✅ Document all bugs with evidence
3. ⏳ Implement AesHash1R
4. ⏳ Rewrite VM initialization
5. ⏳ Rewrite program generation
6. ⏳ Implement proper execution loop
7. ⏳ Fix finalization
8. ⏳ Test against vectors
9. ⏳ Generate final report

**Current Status**: Bug analysis complete, ready to begin implementation.
