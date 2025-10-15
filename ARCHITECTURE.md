# go-randomx Architecture and Implementation Guide

## Overview

This document provides detailed information about the architecture, implementation decisions, and usage patterns for the go-randomx library.

## Architecture

### Package Structure

```
randomx/
├── randomx.go          # Public API: Hasher, Config, Mode types
├── cache.go            # Cache generation and management
├── dataset.go          # Dataset generation (fast mode)
├── vm.go               # Virtual machine implementation
├── program.go          # Program generator
├── memory.go           # Memory management and pooling
├── internal/
│   ├── blake2b.go      # Blake2b cryptographic wrapper
│   ├── argon2.go       # Argon2 wrapper for cache generation
│   └── aes.go          # AES operations wrapper
├── examples/
│   ├── simple/         # Basic hashing example
│   └── mining/         # Mining simulation
└── *_test.go           # Test files
```

## Core Components

### 1. Hasher (randomx.go)

The main entry point for library users. Provides:

- **Configuration validation**: Ensures valid mode and non-empty cache keys
- **Lifecycle management**: Explicit initialization and cleanup with `Close()`
- **Thread safety**: Safe for concurrent `Hash()` calls via read locks
- **Cache key updates**: Dynamic regeneration support

**Design Pattern**: Resource management with explicit lifecycle

```go
type Hasher struct {
    config Config
    cache  *cache
    ds     *dataset
    closed bool
    mu     sync.RWMutex
}
```

### 2. Cache (cache.go)

256 KB cache generated from seed using Argon2d:

- **Generation**: Uses `golang.org/x/crypto/argon2` with RandomX parameters
  - Time: 3 iterations
  - Memory: 256 MB during generation
  - Output: 256 KB (32,768 items of 64 bytes each)
- **Access**: O(1) item retrieval with automatic wrapping
- **Thread safety**: Read-only after initialization

**Design Pattern**: Immutable data structure after initialization

### 3. Dataset (dataset.go)

2+ GB dataset for fast mode operation:

- **Generation**: Parallel computation using `runtime.NumCPU()` workers
- **Algorithm**: Superscalar hash mixing of cache items
- **Memory**: Large contiguous allocation (~2080 MB)
- **Thread safety**: Read-only after initialization, safe for concurrent access

**Design Pattern**: Parallel initialization with worker pools

```go
// Simplified generation flow
for item := 0; item < datasetItems; item++ {
    registers := initialize(itemNumber)
    for iteration := 0; iteration < 8; iteration++ {
        cacheItem := cache.getItem(registers[0])
        mixIntoRegisters(registers, cacheItem)
    }
    writeToDataset(item, registers)
}
```

### 4. Virtual Machine (vm.go)

RandomX bytecode interpreter:

- **Register file**: 8 × 64-bit integer registers (r0-r7)
- **Memory**: 2 MB scratchpad with aligned access
- **Instructions**: 16 basic opcodes including arithmetic, logic, memory, and FP operations
- **Execution**: Interprets 256-instruction programs

**Design Pattern**: Register-based virtual machine with memory pooling

**Key Operations**:
- Integer arithmetic (ADD, SUB, MUL, XOR, ROR)
- Memory operations (LOAD, STORE with address calculation)
- Floating-point operations (FPADD, FPMUL using IEEE-754)
- Bitwise operations (AND, OR)

### 5. Program Generator (program.go)

Deterministic program generation:

- **Input**: Arbitrary byte sequence
- **Process**: Blake2b-512 → entropy → 256 instructions
- **Encoding**: Each instruction is 8 bytes (opcode, registers, immediate value)

**Design Pattern**: Cryptographically deterministic code generation

```go
type instruction struct {
    opcode uint8   // Operation type
    dst    uint8   // Destination register (0-7)
    src    uint8   // Source register (0-7)
    mod    uint8   // Modifier byte
    imm    uint32  // Immediate value
}
```

### 6. Memory Management (memory.go)

Zero-allocation hot path optimization:

- **VM Pool**: Reuses `virtualMachine` instances via `sync.Pool`
- **Scratchpad Pool**: Reuses 2 MB memory buffers
- **Dataset Allocation**: Large single allocation with alignment
- **Security**: Zeroes sensitive data before returning to pool

**Design Pattern**: Object pooling for high-frequency allocations

```go
var vmPool = sync.Pool{
    New: func() interface{} {
        return &virtualMachine{
            reg: [8]uint64{},
            mem: allocateScratchpad(),
        }
    },
}
```

## Cryptographic Dependencies

All cryptographic operations use battle-tested libraries:

### Blake2b (internal/blake2b.go)

**Library**: `golang.org/x/crypto/blake2b`
**License**: BSD-3-Clause
**Usage**:
- Program entropy generation
- Final hash computation
- Dataset item mixing

**Functions**:
- `Blake2b256()`: 32-byte output (final hash)
- `Blake2b512()`: 64-byte output (program generation)
- `Blake2bStream`: Streaming interface

### Argon2 (internal/argon2.go)

**Library**: `golang.org/x/crypto/argon2`
**License**: BSD-3-Clause
**Usage**: Cache generation from seed

**RandomX Parameters**:
```go
Time:      3         // iterations
Memory:    262144    // 256 MB
Threads:   1         // single-threaded
OutputLen: 262144    // 256 KB
Salt:      "RandomX\x03"
```

### AES (internal/aes.go)

**Library**: `crypto/aes` (standard library)
**License**: BSD-3-Clause
**Usage**:
- Scratchpad initialization
- Dataset generation (future optimization)
- VM instruction operations

**Features**:
- Hardware acceleration (AES-NI) when available
- Constant-time operations
- Standard block cipher interface

## Performance Characteristics

### Memory Usage

| Mode       | Cache | Dataset | Scratchpad | Total      |
|------------|-------|---------|------------|------------|
| LightMode  | 256KB | -       | 2MB        | ~2.3 MB    |
| FastMode   | 256KB | 2080MB  | 2MB        | ~2082 MB   |

**Per-goroutine overhead**: ~2 MB (scratchpad from pool)

### CPU Performance

**Expected Performance (AMD64 with AES-NI)**:

- **Fast Mode**: 2,000-4,000 H/s per core
- **Light Mode**: 50-200 H/s per core
- **Initialization**:
  - Cache: 0.6-1.2 seconds
  - Dataset (Fast Mode): 20-30 seconds

**Performance vs C++ Reference**:
- Pure Go: ~50-70% of CGo/C++ performance
- Main bottlenecks:
  1. No SIMD intrinsics
  2. Go's abstraction overhead in crypto operations
  3. GC interaction with large datasets

### Scalability

**Concurrent Hashing**:
- Linear scaling up to `runtime.NumCPU()` workers
- VM pooling prevents allocation storms
- Read-only dataset/cache allows lock-free concurrent access

**Benchmark Results** (example):
```
BenchmarkHasher_Hash-8              200    6,000,000 ns/op    0 B/op    0 allocs/op
BenchmarkHasher_Hash_Parallel-8    1000    2,000,000 ns/op    0 B/op    0 allocs/op
```

## Design Decisions

### 1. Pure Go (No CGo)

**Rationale**:
- Cross-platform compatibility (any Go-supported architecture)
- Simplified build process (no C compiler required)
- Better integration with Go tooling (race detector, pprof, etc.)

**Trade-offs**:
- Performance: 30-50% slower than optimized C++
- No platform-specific SIMD optimizations
- Higher abstraction overhead

### 2. Explicit Lifecycle Management

**Pattern**: `New()` + `defer Close()`

**Rationale**:
- Clear resource ownership
- Deterministic cleanup (not GC-dependent)
- Panic on misuse (use-after-close)

**Alternative Rejected**: Finalizers (non-deterministic, GC pressure)

### 3. Foolproof API

**Validation**:
- Config validation in `New()` before resource allocation
- Panic on use-after-close (programmer error)
- Clear error messages

**Type Safety**:
- Mode enumeration (not string or int)
- Fixed-size output ([32]byte, not []byte)

### 4. Short Functions (<50 lines)

**Rationale**:
- Improves readability and maintainability
- Enables compiler inlining for hot paths
- Easier to test and review

**Example**: VM instruction execution split into 16 small handlers

### 5. Zero Allocations in Hot Path

**Techniques**:
- Object pooling (`sync.Pool` for VMs and scratchpads)
- Pre-allocated buffers
- Stack allocation for small structures

**Verification**: Use `-benchmem` flag to check allocations

```bash
go test -bench=. -benchmem
```

### 6. No Custom Cryptography

**Principle**: "Don't roll your own crypto"

**Implementation**:
- All primitives from standard library or `golang.org/x/crypto`
- No custom hash functions, key derivation, or ciphers
- Only RandomX-specific VM logic is custom

## Usage Patterns

### Pattern 1: Single-Use Hashing

```go
func hashData(data []byte) [32]byte {
    hasher, err := randomx.New(randomx.Config{
        Mode:     randomx.LightMode,
        CacheKey: []byte("seed"),
    })
    if err != nil {
        panic(err)
    }
    defer hasher.Close()
    
    return hasher.Hash(data)
}
```

### Pattern 2: Long-Lived Hasher

```go
type Miner struct {
    hasher *randomx.Hasher
}

func NewMiner(seed []byte) (*Miner, error) {
    hasher, err := randomx.New(randomx.Config{
        Mode:     randomx.FastMode,
        CacheKey: seed,
    })
    if err != nil {
        return nil, err
    }
    return &Miner{hasher: hasher}, nil
}

func (m *Miner) Close() error {
    return m.hasher.Close()
}

func (m *Miner) Mine(data []byte, target [32]byte) (nonce uint64, found bool) {
    for nonce = 0; ; nonce++ {
        input := append(data, encodeNonce(nonce)...)
        hash := m.hasher.Hash(input)
        if bytes.Compare(hash[:], target[:]) <= 0 {
            return nonce, true
        }
    }
}
```

### Pattern 3: Epoch-Based Mining (Monero-style)

```go
type EpochMiner struct {
    hasher      *randomx.Hasher
    currentSeed []byte
    mu          sync.RWMutex
}

func (m *EpochMiner) UpdateEpoch(newSeed []byte) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if err := m.hasher.UpdateCacheKey(newSeed); err != nil {
        return err
    }
    m.currentSeed = newSeed
    return nil
}

func (m *EpochMiner) Hash(data []byte) [32]byte {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.hasher.Hash(data)
}
```

### Pattern 4: Concurrent Worker Pool

```go
func parallelMining(hasher *randomx.Hasher, numWorkers int) {
    var wg sync.WaitGroup
    
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            nonce := uint64(workerID)
            for {
                input := makeInput(nonce)
                hash := hasher.Hash(input)
                
                if checkTarget(hash) {
                    reportSolution(nonce, hash)
                    return
                }
                
                nonce += uint64(numWorkers)
            }
        }(i)
    }
    
    wg.Wait()
}
```

## Testing Strategy

### Unit Tests

- **Configuration validation**: All validation rules
- **Component isolation**: Cache, dataset, VM, program generator independently
- **Edge cases**: Empty inputs, boundary values, wraparound
- **Error handling**: Invalid configs, nil parameters

### Integration Tests

- **End-to-end hashing**: Full pipeline from config to hash
- **Concurrency**: Race detector enabled, parallel hash operations
- **Lifecycle**: Create, use, close, reuse patterns

### Benchmark Tests

- **Single-threaded**: Raw hashing performance
- **Parallel**: Concurrent hashing scalability
- **Allocations**: Verify zero allocations in hot path

### Test Vectors

Reference vectors from RandomX specification ensure compatibility with reference implementation.

## Known Limitations

### 1. Performance Gap

**Issue**: 30-50% slower than C++ implementation
**Cause**: No SIMD, higher abstraction overhead
**Status**: **Inherent** to pure Go
**Mitigation**: Use fast mode, leverage concurrency

### 2. Memory Pressure (Fast Mode)

**Issue**: 2+ GB allocation per hasher
**Cause**: RandomX specification requirement
**Status**: **By Design**
**Mitigation**: Use light mode for memory-constrained environments

### 3. Dataset Generation Time

**Issue**: 20-30 seconds to initialize fast mode
**Cause**: Cryptographically intensive computation
**Status**: **By Design**
**Mitigation**: Generate once, reuse; use light mode for rapid startup

### 4. Floating-Point Determinism

**Issue**: Potential cross-platform FP variations
**Cause**: Go float64 maps to IEEE-754, but some edge cases
**Status**: **Mitigated** by using standard operations only
**Testing**: Validate across architectures (amd64, arm64)

## Future Optimizations

### Potential Improvements

1. **Assembly hot paths**: Hand-optimized assembly for critical loops
2. **SIMD intrinsics**: Using `golang.org/x/sys/cpu` for detection
3. **Huge pages**: Linux transparent huge pages for dataset
4. **Cache line optimization**: Align data structures to 64-byte boundaries

### Non-Goals

- CGo bindings (defeats pure-Go purpose)
- Custom cryptographic implementations
- Breaking API compatibility for marginal gains

## Contributing Guidelines

### Code Style

1. **Function length**: Maximum 50 lines
2. **Error handling**: Return errors, don't panic (except programmer errors)
3. **Comments**: Godoc format for exported symbols
4. **Testing**: Coverage >80% for new code

### Performance Changes

- Include benchmark comparisons
- Verify zero allocations with `-benchmem`
- Test with race detector enabled

### Cryptographic Changes

- **Never** implement custom crypto primitives
- Document library versions and licenses
- Provide rationale for any crypto library choice

## References

- [RandomX Specification](https://github.com/tevador/RandomX/blob/master/doc/specs.md)
- [Monero RandomX Integration](https://github.com/monero-project/monero/tree/master/src/crypto/randomx)
- [Go Cryptography Packages](https://pkg.go.dev/golang.org/x/crypto)
- [Go sync.Pool Documentation](https://pkg.go.dev/sync#Pool)

## License

MIT License - See LICENSE file for details.

All cryptographic dependencies (Blake2b, Argon2, AES) use BSD-3-Clause or compatible licenses.
