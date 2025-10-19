# SuperscalarHash Implementation Guide

**Date**: October 19, 2025  
**Status**: Implementation In Progress  
**Purpose**: Guide for completing the SuperscalarHash program generator

---

## Current Implementation Status

### ✅ Complete Components

1. **Execution Engine** (`superscalar.go`):
   - All 14 superscalar instruction types implemented
   - Arithmetic operations (mulh, smulh, rotr, sign extension)
   - Reciprocal multiplication support
   - Matches C++ reference execution semantics

2. **Data Structures** (`superscalar_program.go`):
   - Instruction types and encodings
   - Program structures
   - Helper functions (reciprocal, signExtend2sCompl, etc.)

3. **Integration** (`cache.go`, `dataset.go`):
   - Programs stored in cache
   - Dataset generation uses superscalar programs
   - Reciprocal pre-computation
   - Proper register initialization and mixing

### ⚠️ Incomplete Component

**Program Generator** (`superscalar_gen.go`):
- Current implementation is simplified
- Missing full CPU scheduling simulation
- Does not match C++ reference output
- Causes all test vectors to fail

---

## What the Generator Must Do

The superscalar program generator must simulate Intel Ivy Bridge CPU scheduling to create deterministic, complex instruction sequences that match the C++ reference exactly.

### Key Requirements

1. **Decoder Buffer Simulation**:
   - 16-byte instruction fetch window
   - Multiple buffer configurations (4-8-4, 7-3-3-3, etc.)
   - Macro-operation fusion rules
   - Decode bandwidth constraints (4 uOPs/cycle)

2. **Execution Port Scheduling**:
   - 3 execution ports (P0, P1, P5) with specific capabilities
   - Port saturation detection
   - Cycle-by-cycle resource tracking
   - Look-ahead for register dependencies

3. **Register Allocation**:
   - Dependency tracking across instructions
   - Latency calculation and availability windows
   - Forward lookahead (up to 4 cycles)
   - Address register selection (highest latency)

4. **Instruction Selection**:
   - Weighted random selection based on Blake2Generator
   - Resource availability checks
   - Multiplication limiting (typically 4 max)
   - Throw-away logic for unsuitable instructions
   - Program size constraints

### Expected Output Characteristics

For seed "test key 000", the C++ reference generates:
- **447 instructions** (varies by seed)
- Specific instruction sequence (deterministic)
- **Address register**: varies based on latency tracking
- Specific mix of instruction types respecting CPU constraints

Current Go implementation generates:
- **~60-100 instructions** (too few)
- Different instruction sequence
- Different address register
- Different instruction type distribution

---

## Implementation Approach

### Phase 1: Study C++ Reference (~4 hours)

1. Clone RandomX reference implementation:
   ```bash
   git clone https://github.com/tevador/RandomX.git
   cd RandomX
   ```

2. Locate key files:
   - `src/superscalar.cpp` - Main generation algorithm (~900 lines)
   - `src/superscalar.hpp` - Structures and constants
   - `src/instruction.hpp` - Instruction definitions

3. Study algorithm structure:
   - `generateSuperscalar()` - Main entry point
   - `selectRegister()` - Register allocation with dependencies
   - `scheduleUop()` - Port scheduling logic
   - Decoder buffer management
   - Instruction creation and validation

### Phase 2: Port Data Structures (~2 hours)

Create exact equivalents of C++ structures in `superscalar_gen.go`:

```go
// DecoderBuffer tracks pending micro-operations
type decoderBuffer struct {
    slots    [4]macroOp
    size     int
    index    int
}

// PortSchedule tracks execution port availability
type portSchedule struct {
    busy  [3]int  // P0, P1, P5
    cycle int
}

// RegisterInfo tracks register dependencies
type registerInfo struct {
    latency      int
    lastOpGroup  int
    lastOp       *macroOp
}

// ProgramState maintains generation state
type programState struct {
    registers    [8]registerInfo
    ports        portSchedule
    decoder      decoderBuffer
    instructions []superscalarInstruction
    cycle        int
    opGroup      int
    mulCount     int
}
```

### Phase 3: Port Helper Functions (~3 hours)

Implement supporting functions that match C++ behavior:

```go
// scheduleUop assigns a micro-op to an execution port
func (ps *portSchedule) scheduleUop(uop executionPort, latency int) bool

// selectRegister chooses a register based on dependencies
func selectRegister(state *programState, gen *blake2Generator, 
    needsSrc bool, needsDst bool) (uint8, bool)

// canIssueInstruction checks if instruction can be issued
func canIssueInstruction(info *superscalarInstrInfo, 
    state *programState) bool

// createInstruction builds an instruction with proper operands
func createInstruction(info *superscalarInstrInfo, 
    state *programState, gen *blake2Generator) (*superscalarInstruction, bool)
```

### Phase 4: Port Main Generation Loop (~5 hours)

Implement the main generation algorithm:

```go
func generateSuperscalarProgram(gen *blake2Generator) *superscalarProgram {
    state := &programState{}
    
    // Initialize state
    state.ports.cycle = 0
    state.decoder.size = 0
    
    // Main generation loop
    for state.cycle < superscalarLatency {
        // Try to schedule pending decoder buffer entries
        issued := trySchedulePendingOps(state)
        
        // Try to add new instruction to decoder
        if state.decoder.size < 4 {
            if tryIssueNewInstruction(state, gen) {
                issued = true
            }
        }
        
        // Advance cycle if nothing scheduled
        if !issued {
            state.cycle++
        }
        
        // Safety limits
        if len(state.instructions) >= superscalarMaxSize {
            break
        }
    }
    
    // Build final program
    return buildProgram(state)
}
```

### Phase 5: Validation (~4 hours)

Create comprehensive tests comparing Go vs C++ output:

```go
func TestSuperscalarProgramGeneration(t *testing.T) {
    // Test with known seed
    seed := []byte("test key 000")
    gen := newBlake2Generator(seed)
    prog := generateSuperscalarProgram(gen)
    
    // Expected values from C++ reference
    expectedSize := 447
    expectedAddrReg := uint8(4)  // Example - verify with C++
    
    if len(prog.instructions) != expectedSize {
        t.Errorf("Program size mismatch: got %d, want %d", 
            len(prog.instructions), expectedSize)
    }
    
    if prog.addressReg != expectedAddrReg {
        t.Errorf("Address register mismatch: got %d, want %d",
            prog.addressReg, expectedAddrReg)
    }
    
    // Verify first few instructions match C++ output
    // ... detailed instruction comparison
}
```

### Phase 6: Debugging (~6 hours)

Add extensive logging to compare with C++ reference:

```go
func (ps *programState) logState() {
    fmt.Printf("Cycle %d:\n", ps.cycle)
    fmt.Printf("  Registers: %v\n", ps.registers)
    fmt.Printf("  Ports: P0=%d P1=%d P5=%d\n", 
        ps.ports.busy[0], ps.ports.busy[1], ps.ports.busy[2])
    fmt.Printf("  Decoder: size=%d\n", ps.decoder.size)
    fmt.Printf("  Instructions: %d\n", len(ps.instructions))
}
```

Run both implementations side-by-side and compare:
- Blake2Generator output (should already match)
- Instruction selection at each step
- Register allocation decisions
- Port scheduling choices
- Final program structure

---

## Testing Strategy

### Unit Tests

1. **Blake2Generator**:
   ```go
   TestBlake2Generator_Determinism
   TestBlake2Generator_ByteOutput
   TestBlake2Generator_Uint32Output
   ```

2. **Program Generation**:
   ```go
   TestGenerateProgram_Size
   TestGenerateProgram_Determinism
   TestGenerateProgram_InstructionTypes
   TestGenerateProgram_AddressRegister
   ```

3. **Instruction Selection**:
   ```go
   TestSelectInstruction_Distribution
   TestSelectInstruction_MultiplicationLimit
   TestSelectInstruction_ResourceAvailability
   ```

4. **Port Scheduling**:
   ```go
   TestPortScheduling_Availability
   TestPortScheduling_Conflicts
   TestPortScheduling_CycleAdvancement
   ```

### Integration Tests

1. **Program Execution**:
   ```go
   TestExecuteProgram_KnownOutput
   TestExecuteProgram_RegisterState
   ```

2. **Dataset Generation**:
   ```go
   TestDatasetItem_WithReferenceProgram
   TestDatasetItem_Determinism
   ```

3. **Full Hash**:
   ```go
   TestHash_OfficialVectors
   ```

---

## Common Pitfalls

### 1. Blake2Generator State

**Problem**: Generator state must match C++ exactly at each byte request.

**Solution**: 
- Ensure generate() is called at exactly 64-byte boundaries
- Verify pos tracking matches C++ cursor
- Test generator output against C++ for same seed

### 2. Integer Overflow

**Problem**: Go and C++ handle integer overflow differently.

**Solution**:
- Use unsigned arithmetic where C++ uses unsigned
- Be careful with signed/unsigned conversions
- Test boundary cases

### 3. Instruction Type Selection

**Problem**: Weighted random selection must match C++ distribution.

**Solution**:
- Use exact same modulo arithmetic as C++
- Verify instruction type probabilities
- Test with multiple seeds

### 4. Register Dependencies

**Problem**: Dependency tracking is complex and error-prone.

**Solution**:
- Track both read-after-write and write-after-read dependencies
- Maintain operation group counters
- Log all dependency decisions

### 5. Port Scheduling

**Problem**: Port busy times must be calculated identically.

**Solution**:
- Match C++ port assignment logic exactly
- Respect port capabilities (some instructions can only use certain ports)
- Handle multi-uop instructions correctly

---

## Success Criteria

Implementation is complete when:

- [ ] All 4 official test vectors pass
- [ ] Generated programs match C++ reference byte-for-byte (same seed → same program)
- [ ] Program sizes match C++ reference (e.g., 447 instructions for "test key 000")
- [ ] Address register selection matches
- [ ] Instruction type distribution matches
- [ ] No race conditions (`go test -race`)
- [ ] Code passes `go vet` and `golint`
- [ ] Performance within 2x of C++ reference (acceptable for Go)

---

## Estimated Effort

**Total Time**: 20-24 hours for complete implementation and validation

- Phase 1 (Study): 4 hours
- Phase 2 (Structures): 2 hours
- Phase 3 (Helpers): 3 hours
- Phase 4 (Main Loop): 5 hours
- Phase 5 (Validation): 4 hours
- Phase 6 (Debugging): 6 hours

**Complexity**: HIGH - Requires careful attention to detail and extensive testing against C++ reference.

---

## Alternative: CGo Wrapper

If time is critical and pure Go is not strictly required:

```go
// #cgo LDFLAGS: -lrandomx
// #include <randomx.h>
import "C"

func generateSuperscalarProgram(gen *blake2Generator) *superscalarProgram {
    // Use C++ implementation for generation
    // Keep execution in pure Go
}
```

**Pros**: Immediate correctness, minimal code
**Cons**: CGo dependency, violates pure Go goal

---

## Resources

- RandomX GitHub: https://github.com/tevador/RandomX
- RandomX Specification: https://github.com/tevador/RandomX/blob/master/doc/specs.md
- C++ Reference Implementation: `RandomX/src/superscalar.cpp`
- This Implementation: `superscalar_gen.go`, `superscalar.go`

---

## Maintenance Notes

Once implemented:

1. **Keep in sync with RandomX updates**: Monitor RandomX repository for algorithm changes
2. **Test with multiple seeds**: Ensure determinism across different inputs
3. **Profile performance**: Identify and optimize hot paths
4. **Document deviations**: If any simplifications are made, document why and impact

---

**Last Updated**: October 19, 2025  
**Maintainer**: go-randomx team  
**Status**: Implementation guide - ready for development
