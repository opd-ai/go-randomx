# Implementation Plan: Resolving go-randomx Gaps

**Document Version**: 1.0  
**Date**: October 15, 2025  
**Status**: Active Development Plan  

## Executive Summary

This document provides a comprehensive, step-by-step plan for resolving the remaining implementation gaps in go-randomx and preparing the project for production use. The primary focus is on **validating hash correctness against the RandomX specification** through comprehensive test vectors.

**Current State**: 
- 7 of 8 audit gaps resolved
- 1 critical gap remains: Test vector validation
- Additional optimization opportunities identified

**Goal**: Achieve production-ready status with verified hash compatibility.

---

## Priority Overview

### P0: CRITICAL - Required for Production
1. **Implement RandomX Test Vectors** (Gap #2)

### P1: HIGH - Performance & Optimization
2. **Optimize Hash() Allocations** (Improve Gap #4)
3. **Implement CPU Feature Detection** (Enhance Gap #1)

### P2: MEDIUM - Enhancements
4. **Custom Memory Allocators**
5. **Comprehensive Fuzzing Suite**
6. **Performance Profiling Tools**

---

## P0: CRITICAL - Test Vector Validation

### Overview
**Status**: ⚠️ NOT PRODUCTION READY  
**Risk Level**: CRITICAL  
**Estimated Effort**: 2-3 days  
**Prerequisite**: Access to RandomX reference implementation or official test vectors

### Problem Statement
The go-randomx implementation has no validation against the official RandomX specification. Hash outputs are unverified, meaning they may not match Monero or other RandomX-based systems, leading to:
- Blockchain consensus failures
- Incompatible mining pool shares
- Block validation errors

### Solution: Implement Comprehensive Test Vectors

#### Step 1: Obtain Official Test Vectors

**Option A: Extract from RandomX Reference Implementation**

```bash
# Clone the official RandomX repository
git clone https://github.com/tevador/RandomX.git
cd RandomX

# Build the reference implementation
mkdir build && cd build
cmake -DARCH=native ..
make

# Generate test vectors
./randomx-tests --help
./randomx-tests > test_vectors.txt
```

**Option B: Use Existing Test Data**

Check for official test vectors in:
- `RandomX/src/tests/` directory
- RandomX specification documentation
- Monero project test data

**Option C: Generate Test Vectors**

Create a test vector generator using the C++ reference:

```cpp
// test_vector_generator.cpp
#include "randomx.h"
#include <iostream>
#include <iomanip>
#include <vector>

struct TestVector {
    std::string key;
    std::string input;
    std::string expected_hash;
};

void generateVector(const std::string& key, const std::string& input) {
    randomx_flags flags = randomx_get_flags();
    randomx_cache* cache = randomx_alloc_cache(flags);
    randomx_init_cache(cache, key.c_str(), key.size());
    
    randomx_vm* vm = randomx_create_vm(flags, cache, NULL);
    
    char hash[RANDOMX_HASH_SIZE];
    randomx_calculate_hash(vm, input.c_str(), input.size(), hash);
    
    std::cout << "{\n";
    std::cout << "  key: \"" << key << "\",\n";
    std::cout << "  input: \"" << input << "\",\n";
    std::cout << "  hash: \"";
    for (int i = 0; i < RANDOMX_HASH_SIZE; i++) {
        std::cout << std::hex << std::setw(2) << std::setfill('0') 
                  << (int)(unsigned char)hash[i];
    }
    std::cout << "\"\n}\n";
    
    randomx_destroy_vm(vm);
    randomx_release_cache(cache);
}

int main() {
    // Generate various test vectors
    generateVector("test key 000", "This is a test");
    generateVector("", "");  // Empty key and input
    generateVector("RandomX example key", "RandomX example input");
    generateVector("Monero", "block data here");
    // Add more vectors...
    
    return 0;
}
```

Compile and run:
```bash
g++ -std=c++11 -o gen_vectors test_vector_generator.cpp -lrandomx
./gen_vectors > go_test_vectors.json
```

#### Step 2: Create Test Vector Data File

Create `testdata/randomx_vectors.json`:

```json
{
  "version": "1.1.10",
  "description": "Official RandomX test vectors from reference implementation",
  "vectors": [
    {
      "name": "empty_input",
      "mode": "light",
      "key": "",
      "input": "",
      "expected": "0000000000000000000000000000000000000000000000000000000000000000"
    },
    {
      "name": "simple_test",
      "mode": "light",
      "key": "test key 000",
      "input": "This is a test",
      "expected": "ACTUAL_HASH_FROM_REFERENCE_IMPLEMENTATION"
    },
    {
      "name": "quick_start_example",
      "mode": "fast",
      "key": "RandomX example key",
      "input": "RandomX example input",
      "expected": "6e2fae47ac7365c1008c046f88dcb5243a7cc8d500616a4a9afcc881f470fb3b"
    },
    {
      "name": "monero_compatible",
      "mode": "fast",
      "key": "Monero seed hash example",
      "input": "block header data",
      "expected": "ACTUAL_HASH_FROM_MONEROD"
    },
    {
      "name": "long_input",
      "mode": "light",
      "key": "test key",
      "input": "Lorem ipsum dolor sit amet... (1MB of data)",
      "expected": "ACTUAL_HASH"
    },
    {
      "name": "binary_data",
      "mode": "fast",
      "key": "binary test",
      "input_hex": "deadbeef000102030405060708090a0b0c0d0e0f",
      "expected": "ACTUAL_HASH"
    }
  ]
}
```

#### Step 3: Implement Test Vector Loading

Create `randomx_test_vectors.go`:

```go
package randomx

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

// TestVector represents a single RandomX test case
type TestVector struct {
	Name     string `json:"name"`
	Mode     string `json:"mode"`
	Key      string `json:"key"`
	Input    string `json:"input"`
	InputHex string `json:"input_hex,omitempty"`
	Expected string `json:"expected"`
}

// TestVectorSuite contains all test vectors
type TestVectorSuite struct {
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Vectors     []TestVector `json:"vectors"`
}

// loadTestVectors loads test vectors from JSON file
func loadTestVectors(t *testing.T) *TestVectorSuite {
	data, err := os.ReadFile("testdata/randomx_vectors.json")
	if err != nil {
		t.Fatalf("Failed to load test vectors: %v", err)
	}

	var suite TestVectorSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		t.Fatalf("Failed to parse test vectors: %v", err)
	}

	return &suite
}
```

#### Step 4: Implement Comprehensive Test

Update `randomx_test.go`:

```go
// TestOfficialVectors validates against official RandomX test vectors.
// This is the CRITICAL test that verifies hash compatibility.
func TestOfficialVectors(t *testing.T) {
	suite := loadTestVectors(t)
	
	t.Logf("Testing against RandomX version: %s", suite.Version)
	t.Logf("Description: %s", suite.Description)
	
	for _, tv := range suite.Vectors {
		t.Run(tv.Name, func(t *testing.T) {
			// Determine mode
			mode := LightMode
			if tv.Mode == "fast" {
				mode = FastMode
			}
			
			// Create hasher
			config := Config{
				Mode:     mode,
				CacheKey: []byte(tv.Key),
			}
			
			hasher, err := New(config)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}
			defer hasher.Close()
			
			// Prepare input
			var input []byte
			if tv.InputHex != "" {
				input, err = hex.DecodeString(tv.InputHex)
				if err != nil {
					t.Fatalf("Invalid input hex: %v", err)
				}
			} else {
				input = []byte(tv.Input)
			}
			
			// Compute hash
			hash := hasher.Hash(input)
			actual := hex.EncodeToString(hash[:])
			
			// Validate
			if actual != tv.Expected {
				t.Errorf("Hash mismatch for '%s':", tv.Name)
				t.Errorf("  Got:      %s", actual)
				t.Errorf("  Expected: %s", tv.Expected)
				t.Errorf("  Mode:     %s", tv.Mode)
				t.Errorf("  Key:      %q", tv.Key)
				t.Errorf("  Input:    %q (len=%d)", tv.Input, len(input))
			}
		})
	}
}

// TestMoneroCompatibility specifically tests Monero-related vectors
func TestMoneroCompatibility(t *testing.T) {
	// Test vectors from actual Monero blocks
	vectors := []struct {
		name       string
		seedHash   string // Block height's seed hash
		blockData  string // Serialized block header
		expectHash string // Known block hash
	}{
		// TODO: Add real Monero block test cases
		// These can be extracted from monerod using:
		// monerod --regtest --offline
		// Then query specific blocks
	}
	
	for _, tv := range vectors {
		t.Run(tv.name, func(t *testing.T) {
			seedHash, _ := hex.DecodeString(tv.seedHash)
			blockData, _ := hex.DecodeString(tv.blockData)
			expectedHash, _ := hex.DecodeString(tv.expectHash)
			
			hasher, _ := New(Config{
				Mode:     FastMode,
				CacheKey: seedHash,
			})
			defer hasher.Close()
			
			hash := hasher.Hash(blockData)
			
			if !bytes.Equal(hash[:], expectedHash) {
				t.Errorf("Monero block hash mismatch")
			}
		})
	}
}
```

#### Step 5: Validate and Document Results

```bash
# Run the test vectors
go test -v -run TestOfficialVectors

# If tests fail, investigate differences:
# 1. Check VM instruction implementation
# 2. Verify cache/dataset generation
# 3. Compare program generation logic
# 4. Validate finalization step

# Once tests pass:
# 1. Update README.md to remove warnings
# 2. Change "Test Vectors Needed" to "Verified"
# 3. Update AUDIT.md status
# 4. Tag release as v1.0.0-validated
```

#### Step 6: Remove Production Warnings

Once all test vectors pass:

```markdown
# In README.md:
- Remove: "⚠️ WARNING: This implementation has not been validated..."
- Change: "⚠️ Test Vectors Needed" → "✅ Verified Against Reference"
- Update: "NOT PRODUCTION READY" → "Production Ready"

# In AUDIT.md:
- Update Gap #2 status from "DOCUMENTED" to "RESOLVED"
```

#### Success Criteria

- [ ] All official test vectors pass (100% match)
- [ ] Monero-specific test cases pass
- [ ] Edge cases covered (empty input, large input, binary data)
- [ ] Fast mode and light mode both validated
- [ ] Documentation updated to reflect production-ready status

---

## P1: HIGH - Optimize Hash() Allocations

### Overview
**Current State**: ~18 allocations per Hash() call  
**Goal**: Reduce to ≤2 allocations per call  
**Estimated Effort**: 1-2 days  
**Performance Impact**: 10-20% faster hashing, reduced GC pressure

### Problem Analysis

Current allocations come from:
1. `generateProgram()` returns `*program` (8× per Hash)
2. `hashProgramEntropy()` allocates `make([]byte, programSize)` (8× per Hash)
3. Internal slice operations

### Solution: Implement Program Pooling

#### Step 1: Add Program Pool

```go
// memory.go

var programPool = sync.Pool{
	New: func() interface{} {
		return &program{
			instructions: [programLength]instruction{},
		}
	},
}

func poolGetProgram() *program {
	return programPool.Get().(*program)
}

func poolPutProgram(p *program) {
	// Clear sensitive data
	for i := range p.instructions {
		p.instructions[i] = instruction{}
	}
	programPool.Put(p)
}
```

#### Step 2: Add Entropy Buffer Pool

```go
// memory.go

var entropyPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, programSize)
	},
}

func poolGetEntropyBuffer() []byte {
	return entropyPool.Get().([]byte)
}

func poolPutEntropyBuffer(buf []byte) {
	// Zero the buffer for security
	for i := range buf {
		buf[i] = 0
	}
	entropyPool.Put(buf)
}
```

#### Step 3: Update Program Generation

```go
// program.go

// generateProgram creates a RandomX program using pooled resources.
func generateProgram(input []byte) *program {
	p := poolGetProgram()
	entropy := poolGetEntropyBuffer()
	defer poolPutEntropyBuffer(entropy)

	// Generate program entropy using Blake2b
	hashProgramEntropyInto(input, entropy)

	// Decode instructions from entropy
	for i := 0; i < programLength; i++ {
		offset := i * 8
		p.instructions[i] = decodeInstruction(entropy[offset : offset+8])
	}

	return p
}

// hashProgramEntropyInto generates entropy into provided buffer.
func hashProgramEntropyInto(input []byte, output []byte) {
	hash := internal.Blake2b512(input)
	copy(output, hash[:])

	for i := 64; i < len(output); i += 64 {
		hash = internal.Blake2b512(hash[:])
		remaining := len(output) - i
		if remaining > 64 {
			remaining = 64
		}
		copy(output[i:], hash[:remaining])
	}
}
```

#### Step 4: Update VM to Return Programs

```go
// vm.go

func (vm *virtualMachine) run(input []byte) [32]byte {
	vm.initialize(input)

	const iterations = 8
	for i := 0; i < iterations; i++ {
		prog := generateProgram(input)
		prog.execute(vm)
		poolPutProgram(prog) // Return to pool after use
		vm.mixDataset()
	}

	return vm.finalize()
}
```

#### Step 5: Verify Allocation Reduction

```go
// Update test in randomx_test.go

func TestHasherZeroAllocations(t *testing.T) {
	// ... existing setup ...
	
	allocs := testing.AllocsPerRun(10, func() {
		_ = hasher.Hash(input)
	})

	t.Logf("Hash() allocations per call: %.2f", allocs)
	
	// Target: ≤2 allocations per call
	if allocs > 2 {
		t.Errorf("Hash() allocated %.2f times per run, target ≤2", allocs)
	}
}
```

#### Success Criteria

- [ ] Allocations reduced from ~18 to ≤2 per Hash()
- [ ] All tests still pass
- [ ] Benchmark shows performance improvement
- [ ] Memory usage remains stable under load

---

## P1: HIGH - CPU Feature Detection

### Overview
**Current State**: Flags field unused  
**Goal**: Implement runtime CPU feature detection  
**Estimated Effort**: 1-2 days  
**Performance Impact**: Optional optimizations, better control

### Solution: Implement Feature Detection

#### Step 1: Add CPU Detection Package

```go
// internal/cpu/features.go

package cpu

import (
	"runtime"
	"golang.org/x/sys/cpu"
)

// Features represents detected CPU capabilities
type Features struct {
	AES      bool // AES-NI support
	AVX2     bool // AVX2 SIMD support
	SHA      bool // SHA extensions
	Platform string
}

// Detect probes the CPU for supported features
func Detect() Features {
	return Features{
		AES:      cpu.X86.HasAES,
		AVX2:     cpu.X86.HasAVX2,
		SHA:      cpu.X86.HasSHA,
		Platform: runtime.GOARCH,
	}
}

// String returns a human-readable feature list
func (f Features) String() string {
	var features []string
	if f.AES {
		features = append(features, "AES-NI")
	}
	if f.AVX2 {
		features = append(features, "AVX2")
	}
	if f.SHA {
		features = append(features, "SHA")
	}
	return strings.Join(features, ", ")
}
```

#### Step 2: Update Config Validation

```go
// randomx.go

// Validate checks if the configuration is valid and applies feature detection
func (c *Config) Validate() error {
	if len(c.CacheKey) == 0 {
		return errors.New("randomx: cache key must not be empty")
	}

	if c.Mode != LightMode && c.Mode != FastMode {
		return fmt.Errorf("randomx: invalid mode: %v", c.Mode)
	}

	// Apply automatic feature detection if Flags is default
	if c.Flags == FlagDefault {
		features := cpu.Detect()
		if features.AES {
			c.Flags |= FlagAES
		}
		// Future: Add more flags as optimizations are implemented
	}

	return nil
}
```

#### Step 3: Add Feature Logging

```go
// randomx.go

// New creates a new RandomX hasher with the specified configuration.
func New(config Config) (*Hasher, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Log detected features in debug mode
	if debugMode := os.Getenv("RANDOMX_DEBUG"); debugMode != "" {
		features := cpu.Detect()
		log.Printf("RandomX CPU features: %s", features.String())
		log.Printf("RandomX flags enabled: %v", config.Flags)
	}

	// ... rest of initialization ...
}
```

#### Step 4: Document Feature Usage

Update README.md:

```markdown
## CPU Feature Detection

go-randomx automatically detects CPU features and uses hardware acceleration when available:

- **AES-NI**: Automatic via Go's crypto/aes (Intel/AMD)
- **AVX2**: Reserved for future SIMD optimizations
- **SHA**: Reserved for future hash optimizations

You can control feature usage via the `Flags` field:

```go
// Automatic detection (recommended)
config := randomx.Config{
    Mode:     randomx.FastMode,
    Flags:    randomx.FlagDefault,
    CacheKey: seed,
}

// Explicitly disable features (testing/compatibility)
config := randomx.Config{
    Mode:     randomx.FastMode,
    Flags:    0, // No optimizations
    CacheKey: seed,
}
```

Check detected features:
```bash
RANDOMX_DEBUG=1 go run your_program.go
# Output: RandomX CPU features: AES-NI, AVX2
```
```

#### Success Criteria

- [ ] CPU feature detection working on x86-64, ARM64
- [ ] Flags field functional and documented
- [ ] Debug logging available
- [ ] No performance regression

---

## P2: MEDIUM - Custom Memory Allocators

### Overview
**Goal**: Support custom allocators for dataset/cache  
**Use Case**: Huge pages, NUMA-aware allocation  
**Estimated Effort**: 2-3 days

### Implementation Plan

#### Step 1: Define Allocator Interface

```go
// memory.go

// Allocator provides custom memory allocation
type Allocator interface {
	// Alloc allocates size bytes with specified alignment
	Alloc(size, alignment int) ([]byte, error)
	
	// Free releases allocated memory
	Free([]byte) error
}

// DefaultAllocator uses standard Go allocation
type DefaultAllocator struct{}

func (a *DefaultAllocator) Alloc(size, alignment int) ([]byte, error) {
	// Standard allocation
	return make([]byte, size), nil
}

func (a *DefaultAllocator) Free(buf []byte) error {
	// Go GC handles this
	return nil
}
```

#### Step 2: Add HugePageAllocator (Linux)

```go
// memory_linux.go

// HugePageAllocator uses Linux huge pages for better TLB efficiency
type HugePageAllocator struct{}

func (a *HugePageAllocator) Alloc(size, alignment int) ([]byte, error) {
	// Use mmap with MAP_HUGETLB
	// Implementation requires syscall package
	// See: https://man7.org/linux/man-pages/man2/mmap.2.html
}
```

#### Step 3: Update Config

```go
// randomx.go

type Config struct {
	Mode      Mode
	Flags     Flags
	CacheKey  []byte
	Allocator Allocator // Optional custom allocator
}
```

---

## P2: MEDIUM - Fuzzing Suite

### Overview
**Goal**: Discover edge cases and bugs  
**Estimated Effort**: 1-2 days

### Implementation

```go
// fuzz_test.go

func FuzzHasher(f *testing.F) {
	// Seed corpus
	f.Add([]byte("test"), []byte("input"))
	f.Add([]byte(""), []byte(""))
	f.Add([]byte("key"), []byte("data"))
	
	f.Fuzz(func(t *testing.T, key, input []byte) {
		if len(key) == 0 {
			return // Skip invalid configs
		}
		
		config := Config{
			Mode:     LightMode,
			CacheKey: key,
		}
		
		hasher, err := New(config)
		if err != nil {
			return
		}
		defer hasher.Close()
		
		// Should never panic
		_ = hasher.Hash(input)
	})
}
```

Run fuzzing:
```bash
go test -fuzz=FuzzHasher -fuzztime=1h
```

---

## P2: MEDIUM - Performance Profiling

### Overview
**Goal**: Identify and optimize hot paths  
**Estimated Effort**: 1 day

### Implementation

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=BenchmarkHasher_Hash
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=BenchmarkHasher_Hash
go tool pprof mem.prof

# Generate flame graph
go test -cpuprofile=cpu.prof -bench=.
go tool pprof -http=:8080 cpu.prof
```

Create profiling helper:

```go
// tools/profile.go

package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"
	
	"github.com/opd-ai/go-randomx"
)

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile")
	memprofile := flag.String("memprofile", "", "write memory profile")
	flag.Parse()

	if *cpuprofile != "" {
		f, _ := os.Create(*cpuprofile)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Run hash operations...

	if *memprofile != "" {
		f, _ := os.Create(*memprofile)
		pprof.WriteHeapProfile(f)
		f.Close()
	}
}
```

---

## Timeline and Milestones

### Phase 1: Critical (Week 1-2)
- [x] Day 1: Obtain and implement test vectors infrastructure ✅ **COMPLETED October 15, 2025**
  - Extracted official test vectors from RandomX reference implementation (github.com/tevador/RandomX)
  - Created `testdata/randomx_vectors.json` with 4 official test vectors
  - Implemented `testvectors.go` with LoadTestVectors, TestVector helpers
  - Created `testvectors_test.go` with comprehensive tests (>80% coverage)
  - Verified TestOfficialVectors correctly identifies hash mismatches
  - Confirmed implementation is deterministic (same input → same output)
  - All existing tests still pass (no regressions)
  
- [x] Day 2: Root cause analysis ✅ **COMPLETED October 15, 2025**
  - 🔴 **CRITICAL ISSUE IDENTIFIED**: golang.org/x/crypto/argon2 provides Argon2i/id, NOT Argon2d
  - RandomX specifically requires Argon2d (data-dependent mode)
  - This is the ROOT CAUSE of all hash mismatches
  - Documented in `ARGON2D_ISSUE.md` - full analysis and solution options
  - Documented in `DEBUGGING_SESSION_OCT15.md` - investigation notes
  - Created `ARGON2D_IMPLEMENTATION_GUIDE.md` - detailed implementation roadmap with 8 phases
  - Placeholder implementation added to allow continued development
  - Research completed: No suitable existing Go Argon2d libraries found
  - Path forward: Port Argon2d from RandomX C implementation (24 hours estimated)
  
- [ ] Days 3-6: **Implement proper Argon2d** (CRITICAL BLOCKER) ⏳ **READY TO START**
  - [x] Research existing Go Argon2d implementations → None suitable
  - [x] Analyze RandomX argon2_core.c structure → Documented
  - [x] Create detailed implementation guide → `ARGON2D_IMPLEMENTATION_GUIDE.md`
  - [ ] Phase 1: Blake2b utilities (2 hours)
  - [ ] Phase 2: Block structures (2 hours)
  - [ ] Phase 3: Blake2b G function (3 hours)
  - [ ] Phase 4: Block compression (4 hours)
  - [ ] Phase 5: Data-dependent indexing (3 hours)
  - [ ] Phase 6: Memory filling (4 hours)
  - [ ] Phase 7: Public API (2 hours)
  - [ ] Phase 8: Validation against reference (4 hours)
  
- [ ] Day 7: Validate all test vectors pass
- [ ] Day 8: Update documentation, remove warnings

### Phase 2: Optimization (Week 2)
- [ ] Day 1-2: Implement program pooling
- [ ] Day 3: CPU feature detection
- [ ] Day 4-5: Benchmark and validate improvements

### Phase 3: Enhancement (Week 3)
- [ ] Day 1-2: Custom allocators
- [ ] Day 3: Fuzzing suite
- [ ] Day 4-5: Performance profiling and documentation

---

## Validation Checklist

Before marking production-ready:

### Correctness
- [ ] All official RandomX test vectors pass
- [ ] Monero blockchain compatibility verified
- [ ] Cross-platform testing (Linux, macOS, Windows)
- [ ] Architecture testing (amd64, arm64)

### Performance
- [ ] Hash rate within 70% of C++ reference
- [ ] Allocations ≤2 per Hash() call
- [ ] Memory usage stable over time
- [ ] No memory leaks detected

### Security
- [ ] Fuzzing discovers no crashes
- [ ] Race detector passes
- [ ] Memory sanitizer clean (if available)
- [ ] Security audit completed

### Documentation
- [ ] All warnings removed
- [ ] API fully documented
- [ ] Examples updated and tested
- [ ] Migration guide available

---

## Resources

### Reference Implementations
- **Official RandomX**: https://github.com/tevador/RandomX
- **Monero Integration**: https://github.com/monero-project/monero
- **Test Vectors**: RandomX/src/tests/

### Documentation
- **RandomX Spec**: https://github.com/tevador/RandomX/blob/master/doc/specs.md
- **Design Doc**: https://github.com/tevador/RandomX/blob/master/doc/design.md

### Tools
- **Go Profiler**: `go tool pprof`
- **Benchmark**: `go test -bench`
- **Fuzzer**: `go test -fuzz`

---

## Contact and Support

For implementation questions:
- Open GitHub Discussion: https://github.com/opd-ai/go-randomx/discussions
- Create Issue: https://github.com/opd-ai/go-randomx/issues

---

**Next Steps**: Begin with Phase 1, Day 1 - Obtaining official test vectors. This is the critical blocker for production readiness.
