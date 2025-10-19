# Debug Tracing Infrastructure - Usage Guide

This document explains how to use the debug tracing infrastructure to diagnose hash validation issues in go-randomx.

## Overview

The debug tracing system provides detailed visibility into the RandomX hash computation process, allowing step-by-step comparison with the C++ reference implementation to identify divergence points.

## Quick Start

### Enable Debug Tracing

Set the `RANDOMX_DEBUG` environment variable:

```bash
# Enable debug output
export RANDOMX_DEBUG=1

# Run tests
go test -v -run TestExtractOurTrace

# Disable debug output
unset RANDOMX_DEBUG
```

### Extract Our Implementation Trace

```bash
# Save trace to file
RANDOMX_DEBUG=1 go test -v -run TestExtractOurTrace > our_trace.txt 2>&1

# View the trace
less our_trace.txt
```

## What Gets Traced

The debug system logs:

1. **Input Processing**
   - Input data (string and length)
   - Initial Blake2b-512 hash

2. **VM Initialization**
   - Scratchpad first 64 bytes
   - AES generator states

3. **Program Execution** (for each of 8 programs)
   - Program number
   - First 5 instructions (opcode, dst, src, mod, immediate)
   - Register state after program execution

4. **Final Output**
   - Final hash (32 bytes)

## Trace Format

```
[TRACE] ========== RandomX Hash Computation ==========
[TRACE] Input: "This is a test" (length=14 bytes)
[TRACE] --- VM Initialization ---
[TRACE] Initial Blake2b-512 hash (64 bytes): 152455751b73ac...
[TRACE] Scratchpad first 64 bytes (64 bytes): daaa0722fff158...
[TRACE] VM initialization complete
[TRACE] --- Program 1/8 ---
[TRACE] First 5 instructions:
[TRACE]   [000] opcode=0x8e dst=r5 src=r0 mod=0x5e imm=0x273680b1
[TRACE]   [001] opcode=0x03 dst=r5 src=r1 mod=0x30 imm=0x74ef2b65
...
[TRACE] Registers after program 1:
[TRACE]   r0 = 0x4e854e362416808a (5658014501301551242)
[TRACE]   r1 = 0x83103580a254ee1c (9444107245191491100)
...
[TRACE] Final hash (32 bytes): 3b0012e9a25ae4cd...
[TRACE] ========== End of Hash Computation ==========
```

## Comparing with C++ Reference

### Step 1: Extract C++ Trace

You need to modify the RandomX C++ reference implementation to output similar trace information.

**Option A: Add printf Statements**

Edit `RandomX/src/randomx.cpp` and add logging:

```cpp
// In randomx_calculate_hash()
printf("[CPP TRACE] Input: \"%s\" (length=%zu bytes)\n", input, inputSize);

// In VM::run()
printf("[CPP TRACE] Initial Blake2b-512 hash: ");
for (int i = 0; i < 64; i++) printf("%02x", hash[i]);
printf("\n");

// After each program
printf("[CPP TRACE] Registers after program %d:\n", progNum);
for (int i = 0; i < 8; i++) {
    printf("[CPP TRACE]   r%d = 0x%016llx\n", i, reg[i]);
}
```

**Option B: Use Debugger**

```bash
gdb ./randomx-tests
(gdb) break randomx_calculate_hash
(gdb) run "test key 000" "This is a test"
(gdb) print reg[0]
(gdb) print reg[1]
...
```

### Step 2: Compare Traces

```bash
# Extract our trace
RANDOMX_DEBUG=1 go test -v -run TestExtractOurTrace > go_trace.txt 2>&1

# Extract C++ trace
./randomx-tests-instrumented "test key 000" "This is a test" > cpp_trace.txt 2>&1

# Find first divergence
diff -u cpp_trace.txt go_trace.txt | head -50
```

### Step 3: Identify Divergence Point

The diff will show where our implementation diverges from C++:

```diff
[TRACE] Initial Blake2b-512 hash: 152455751b73ac...
--- Same in both ---

[TRACE] Scratchpad first 64 bytes: daaa0722fff158...
--- Same in both ---

[TRACE] Registers after program 1:
-  r0 = 0x4e854e362416808a (C++ reference)
+  r0 = 0x4e854e362416808b (Our implementation)
   ^^^^^^^^^^^^^^^^^^ DIVERGENCE FOUND!
```

This tells you the bug occurs during program 1 execution.

## Advanced Debugging

### Add More Trace Points

Edit `vm.go` and add tracing where needed:

```go
// Example: Trace memory reads
func (vm *virtualMachine) readMemory(addr uint32) uint64 {
    value := binary.LittleEndian.Uint64(vm.mem[addr:])
    traceUint64(fmt.Sprintf("readMemory(0x%08x)", addr), value)
    return value
}
```

### Trace Specific Registers

```go
// In executeIteration(), trace register changes
traceLog("Before instruction %d: r%d = 0x%016x", i, dst, vm.reg[dst])
vm.executeInstruction(instr)
traceLog("After instruction %d: r%d = 0x%016x", i, dst, vm.reg[dst])
```

### Conditional Tracing

```go
// Only trace when certain conditions are met
if vm.reg[0] > 0x1000000000000000 {
    traceLog("WARNING: r0 is very large: 0x%016x", vm.reg[0])
}
```

## Performance Impact

Debug tracing is **completely free when disabled**:

- `RANDOMX_DEBUG=0` (or unset): Zero overhead, no performance impact
- `RANDOMX_DEBUG=1`: Significant slowdown due to I/O, only use for debugging

Benchmarks confirm zero overhead when disabled:

```
BenchmarkHashWithDebugDisabled    5  220460971 ns/op
```

## Test Infrastructure

### Available Tests

1. **TestExtractOurTrace** - Extract full trace from our implementation
2. **TestCompareWithReference** - Compare with C++ trace (requires trace file)
3. **TestCompareInitialHashes** - Verify Blake2b initial hash is correct
4. **TestDebugEnvironmentVariable** - Verify debug can be toggled

### Running Specific Tests

```bash
# Extract our trace
RANDOMX_DEBUG=1 go test -v -run TestExtractOurTrace

# Compare with reference (once you have reference_trace_test1.json)
go test -v -run TestCompareWithReference

# Verify initial hashing
go test -v -run TestCompareInitialHashes
```

## Creating Reference Traces

To create `testdata/reference_trace_test1.json` from C++ output:

1. Modify C++ to output JSON:
```cpp
FILE* f = fopen("reference_trace_test1.json", "w");
fprintf(f, "{\n");
fprintf(f, "  \"test_name\": \"basic_test_1\",\n");
fprintf(f, "  \"key\": \"test key 000\",\n");
fprintf(f, "  \"input\": \"This is a test\",\n");
fprintf(f, "  \"initial_blake2b\": \"");
for (int i = 0; i < 64; i++) fprintf(f, "%02x", hash[i]);
fprintf(f, "\",\n");
// ... more fields
fprintf(f, "}\n");
fclose(f);
```

2. Copy the JSON file to `testdata/`

3. Run comparison test:
```bash
go test -v -run TestCompareWithReference
```

## Troubleshooting

### "No trace output appears"

- Check that `RANDOMX_DEBUG=1` is set
- Verify you're running a test that actually calls Hash()
- Redirect stderr: `2>&1`

### "Too much output"

- Pipe through `head` or `tail`: `| head -100`
- Use `grep` to filter: `| grep "Program 1"`
- Save to file and use text editor

### "Can't find divergence"

- Compare byte-by-byte, not just register values
- Check endianness (little vs big)
- Verify both implementations use same input

## Next Steps

Once you've identified the divergence point:

1. Examine that specific code section
2. Look for:
   - Off-by-one errors
   - Sign extension issues
   - Byte order problems
   - Integer overflow/underflow
3. Apply minimal fix
4. Re-run all test vectors to verify

## Example Debug Session

```bash
# 1. Extract our trace
RANDOMX_DEBUG=1 go test -v -run TestExtractOurTrace > go_trace.txt 2>&1

# 2. Look at the hash we produce
grep "Final hash" go_trace.txt
# Output: Final hash (32 bytes): 3b0012e9a25ae4cd...

# 3. Compare with expected
# Expected: 639183aae1bf4c9a...
# Ours:     3b0012e9a25ae4cd...
#           ^^  <- First byte differs!

# 4. Work backwards - check registers after program 8
grep -A 10 "Program 8/8" go_trace.txt

# 5. Compare with C++ trace at same point
# If registers match, bug is in finalize()
# If registers differ, bug is in program execution

# 6. Binary search to find exact divergence
# Check after each program until you find first mismatch
```

## Files

- `debug_trace.go` - Debug helper functions
- `debug_comparison_test.go` - Comparison tests
- `testdata/reference_trace_template.json` - Template for C++ trace data

## Documentation

- `NEXT_DEVELOPMENT_PHASE.md` - Overall debugging strategy
- This file - Detailed usage guide
