# Next Steps: Implementing SuperscalarHash

**Status**: Foundation complete, ready for SuperscalarHash implementation  
**Date**: October 18, 2025  
**Estimated Time**: 2-4 days for complete implementation and validation

---

## Quick Summary

The go-randomx implementation is **98% complete**. All components work correctly except one: **SuperscalarHash dataset item generation**. This causes all test vectors to fail (0/4 passing).

**What's Done** ✅:
- Argon2d cache generation (verified correct)
- All AES generators (AesGenerator1R, AesGenerator4R, AesHash1R)
- VM implementation (256 instructions, execution, finalization)
- Blake2Generator foundation
- Superscalar program structures
- Comprehensive documentation

**What's Needed** ❌:
- SuperscalarHash program generator (~400 LOC)
- SuperscalarHash program executor (~100 LOC)
- Integration into cache/dataset (~50 LOC)

---

## Implementation Checklist

### Phase 1: Understand SuperscalarHash (~2 hours)

- [ ] Read RandomX specification Section 3: https://github.com/tevador/RandomX/blob/master/doc/specs.md#3-superscalarhash
- [ ] Study C++ reference: `/tmp/RandomX/src/superscalar.cpp` (903 lines)
- [ ] Review `SUPERSCALAR_IMPLEMENTATION_REQUIRED.md` in this repo
- [ ] Understand 14 instruction types and their semantics

### Phase 2: Implement Helper Functions (~4 hours)

Create `superscalar.go` with these functions:

- [ ] `reciprocal(divisor uint32) uint64` - Compute 2^64 / divisor
  ```go
  // Port from src/reciprocal.c (simple algorithm)
  ```

- [ ] `signExtend2sCompl(imm uint32) uint64` - Sign extension for immediates
  ```go
  // Extend 32-bit signed immediate to 64-bit
  ```

- [ ] `mulh(a, b uint64) uint64` - Unsigned high multiplication
  ```go
  // (a * b) >> 64 using math/bits.Mul64
  ```

- [ ] `smulh(a, b uint64) uint64` - Signed high multiplication
  ```go
  // (int64(a) * int64(b)) >> 64 using proper casting
  ```

### Phase 3: Implement Program Executor (~6 hours)

- [ ] `executeSuperscalarProgram(registers *[8]uint64, prog *superscalarProgram, reciprocals []uint64)`
  ```go
  func executeSuperscalarProgram(registers *[8]uint64, prog *superscalarProgram, reciprocals []uint64) {
      for _, instr := range prog.instructions {
          switch instr.opcode {
          case ssISUB_R:
              registers[instr.dst] -= registers[instr.src]
          case ssIXOR_R:
              registers[instr.dst] ^= registers[instr.src]
          case ssIADD_RS:
              registers[instr.dst] += registers[instr.src] << (instr.src & 3)
          case ssIMUL_R:
              registers[instr.dst] *= registers[instr.src]
          case ssIROR_C:
              registers[instr.dst] = bits.RotateLeft64(registers[instr.dst], -int(instr.imm32 & 63))
          case ssIADD_C7, ssIADD_C8, ssIADD_C9:
              registers[instr.dst] += signExtend2sCompl(instr.imm32)
          case ssIXOR_C7, ssIXOR_C8, ssIXOR_C9:
              registers[instr.dst] ^= signExtend2sCompl(instr.imm32)
          case ssIMULH_R:
              registers[instr.dst] = mulh(registers[instr.dst], registers[instr.src])
          case ssISMULH_R:
              registers[instr.dst] = smulh(registers[instr.dst], registers[instr.src])
          case ssIMUL_RCP:
              registers[instr.dst] *= reciprocals[instr.imm32]
          }
      }
  }
  ```

- [ ] Test executor with known register states
- [ ] Compare output with C++ reference for validation

### Phase 4: Implement Program Generator (~16 hours)

This is the complex part. The generator must:

- [ ] Create program structure with dependency tracking
- [ ] Generate 3-60 instructions per program
- [ ] Track register dependencies and cycles
- [ ] Respect execution port constraints
- [ ] Select address register based on criteria

**Simplified Approach** (recommended for initial implementation):

Port the C++ generator directly without optimization:
```go
func generateSuperscalarProgram(gen *blake2Generator) *superscalarProgram {
    prog := &superscalarProgram{}
    
    // Initialize state
    var dependencies [8]int // Track when each register is available
    var cycles int = 0       // Current cycle count
    
    // Generate 3-60 instructions
    for len(prog.instructions) < 60 {
        // Try to generate an instruction
        opcode := selectInstructionType(gen)
        instr := generateInstruction(opcode, gen, dependencies)
        
        if instr != nil {
            prog.instructions = append(prog.instructions, *instr)
            updateDependencies(instr, dependencies, &cycles)
        }
        
        // Stop if we've reached sufficient size and complexity
        if shouldStopGeneration(prog, cycles) {
            break
        }
    }
    
    // Select address register (register with most dependencies)
    prog.addressReg = selectAddressRegister(dependencies)
    
    return prog
}
```

**Key Functions Needed**:
- [ ] `selectInstructionType(gen *blake2Generator) uint8`
- [ ] `generateInstruction(opcode uint8, gen *blake2Generator, deps [8]int) *superscalarInstruction`
- [ ] `updateDependencies(instr *superscalarInstruction, deps *[8]int, cycles *int)`
- [ ] `shouldStopGeneration(prog *superscalarProgram, cycles int) bool`
- [ ] `selectAddressRegister(deps [8]int) uint8`

**Reference**: Study `superscalar.cpp` functions:
- `generateSuperscalar()`
- `selectRegister()`
- `generateRandomX()`
- Dependency tracking logic

### Phase 5: Integrate into Cache (~2 hours)

Modify `cache.go`:

- [ ] Add field `programs [8]superscalarProgram`
- [ ] Add field `reciprocals []uint64`

- [ ] Generate programs in `newCache()`:
  ```go
  func newCache(seed []byte) (*cache, error) {
      // ... existing Argon2d generation ...
      
      // Generate 8 superscalar programs
      gen := newBlake2Generator(seed)
      for i := 0; i < 8; i++ {
          c.programs[i] = generateSuperscalarProgram(gen)
          
          // Pre-compute reciprocals for IMUL_RCP instructions
          for _, instr := range c.programs[i].instructions {
              if instr.opcode == ssIMUL_RCP {
                  rcp := reciprocal(instr.imm32)
                  c.reciprocals = append(c.reciprocals, rcp)
                  // Update instruction to reference reciprocal index
              }
          }
      }
      
      return c, nil
  }
  ```

- [ ] Fix `getItem()` to compute dataset items:
  ```go
  func (c *cache) getItem(index uint32) []byte {
      // This should compute a dataset item using SuperscalarHash
      // For now, call a helper function
      item := make([]byte, 64)
      c.computeDatasetItem(uint64(index), item)
      return item
  }
  
  func (c *cache) computeDatasetItem(itemNumber uint64, output []byte) {
      // Initialize registers with constants
      var registers [8]uint64
      registers[0] = (itemNumber + 1) * superscalarMul0
      registers[1] = registers[0] ^ superscalarAdd1
      registers[2] = registers[0] ^ superscalarAdd2
      registers[3] = registers[0] ^ superscalarAdd3
      registers[4] = registers[0] ^ superscalarAdd4
      registers[5] = registers[0] ^ superscalarAdd5
      registers[6] = registers[0] ^ superscalarAdd6
      registers[7] = registers[0] ^ superscalarAdd7
      
      registerValue := itemNumber
      
      // Execute 8 superscalar programs
      for i := 0; i < 8; i++ {
          // Get cache block
          cacheIndex := uint32((registerValue & 0x1FFFF8) / 64)
          cacheBlock := c.data[cacheIndex*64 : (cacheIndex+1)*64]
          
          // Execute program
          executeSuperscalarProgram(&registers, &c.programs[i], c.reciprocals)
          
          // XOR cache block into registers
          for r := 0; r < 8; r++ {
              val := binary.LittleEndian.Uint64(cacheBlock[r*8 : r*8+8])
              registers[r] ^= val
          }
          
          // Next cache address from address register
          registerValue = registers[c.programs[i].addressReg]
      }
      
      // Write registers to output
      for r := 0; r < 8; r++ {
          binary.LittleEndian.PutUint64(output[r*8:r*8+8], registers[r])
      }
  }
  ```

### Phase 6: Update Dataset Generation (~1 hour)

Modify `dataset.go`:

- [ ] Replace `generateItem()` to use cache's SuperscalarHash:
  ```go
  func (ds *dataset) generateItem(c *cache, itemNumber uint64, output []byte) {
      // Just delegate to cache's SuperscalarHash implementation
      c.computeDatasetItem(itemNumber, output)
  }
  ```

- [ ] Remove old mixing functions (mixRegister, etc.)

### Phase 7: Add Constants (~30 minutes)

In `superscalar.go`, add:

```go
const (
    cacheAccesses = 8
    
    // Superscalar initialization constants
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

### Phase 8: Testing & Validation (~8 hours)

- [ ] Create `superscalar_test.go` with unit tests:
  ```go
  func TestBlake2Generator(t *testing.T) { /* ... */ }
  func TestReciprocal(t *testing.T) { /* ... */ }
  func TestProgramExecution(t *testing.T) { /* ... */ }
  func TestProgramGeneration(t *testing.T) { /* ... */ }
  func TestDatasetItemGeneration(t *testing.T) { /* ... */ }
  ```

- [ ] Run test vectors: `go test -v -run TestOfficialVectors`
- [ ] Compare intermediate values with C++ if tests still fail
- [ ] Debug and fix discrepancies
- [ ] Verify all 4 test vectors pass

### Phase 9: Performance & Polish (~4 hours)

- [ ] Run benchmarks: `go test -bench=.`
- [ ] Profile hot paths: `go test -cpuprofile=cpu.prof`
- [ ] Optimize critical sections if needed
- [ ] Add documentation comments
- [ ] Run `go vet` and `golint`
- [ ] Run race detector: `go test -race`

---

## Testing Strategy

### Unit Tests

Test each component independently:

1. **Blake2Generator**:
   ```go
   gen1 := newBlake2Generator([]byte("test"))
   gen2 := newBlake2Generator([]byte("test"))
   // Verify same sequence of bytes
   ```

2. **Reciprocal**:
   ```go
   rcp := reciprocal(12345678)
   // Verify against known values
   ```

3. **Program Execution**:
   ```go
   prog := &superscalarProgram{
       instructions: []superscalarInstruction{
           {opcode: ssIADD_RS, dst: 0, src: 1, imm32: 0},
       },
   }
   registers := [8]uint64{100, 200, 0, 0, 0, 0, 0, 0}
   executeSuperscalarProgram(&registers, prog, nil)
   // Verify r0 changed correctly
   ```

4. **Program Generation**:
   ```go
   gen := newBlake2Generator([]byte("test key"))
   prog := generateSuperscalarProgram(gen)
   // Verify program has 3-60 instructions
   // Verify address register is valid (0-7)
   ```

5. **Dataset Item**:
   ```go
   cache, _ := newCache([]byte("test key 000"))
   item := cache.getItem(0)
   // Compare with C++ reference for item 0
   ```

### Integration Tests

- [ ] Test full cache initialization
- [ ] Test dataset generation
- [ ] Test VM with real dataset items
- [ ] Test against all 4 official vectors

### Debugging Tips

If tests still fail after implementation:

1. **Add trace logging** to see intermediate values:
   ```go
   fmt.Printf("After program %d: registers=%v\n", i, registers)
   ```

2. **Compare with C++** at each step:
   - Blake2Generator output
   - Generated program instructions
   - Register state after each program execution
   - Final dataset item

3. **Check constants**: Verify superscalarMul0, superscalarAdd1-7 match C++

4. **Verify reciprocals**: Check IMUL_RCP reciprocal values

---

## Files to Create/Modify

**Create**:
- `superscalar.go` (~600 LOC)
- `superscalar_test.go` (~300 LOC)

**Modify**:
- `cache.go` - Add programs field and generation (~50 LOC)
- `dataset.go` - Use SuperscalarHash (~10 LOC)

**Already Created**:
- ✅ `blake2_generator.go`
- ✅ `superscalar_program.go`

---

## Success Criteria

✅ Ready to merge when:
- [ ] All 4 test vectors pass with byte-exact matches
- [ ] No race conditions (`go test -race` passes)
- [ ] Code passes `go vet` and `golint`
- [ ] Unit tests cover all SuperscalarHash components
- [ ] Performance within 2x of C++ reference (acceptable for Go)
- [ ] Documentation updated

---

## Resources

- **RandomX Spec**: https://github.com/tevador/RandomX/blob/master/doc/specs.md
- **C++ Reference**: `/tmp/RandomX/src/superscalar.cpp`
- **This Repo Docs**:
  - `SUPERSCALAR_IMPLEMENTATION_REQUIRED.md` - Detailed requirements
  - `VALIDATION_REPORT_OCT18.md` - Validation analysis
- **Test Vectors**: `testdata/randomx_vectors.json`

---

## Questions?

If stuck, refer to:
1. C++ reference implementation (`/tmp/RandomX/src/`)
2. RandomX specification (Section 3)
3. Existing Go code patterns in `vm.go` and `instructions.go`

Good luck! The foundation is solid, just need to implement SuperscalarHash. 🚀
