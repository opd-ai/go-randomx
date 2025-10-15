# go-randomx Quick Reference

## Installation

```bash
go get github.com/opd-ai/go-randomx
```

## Basic Usage

```go
import "github.com/opd-ai/go-randomx"

// Create hasher
config := randomx.Config{
    Mode:     randomx.FastMode,    // or LightMode
    CacheKey: []byte("seed"),      // Required, non-empty
}

hasher, err := randomx.New(config)
if err != nil {
    panic(err)
}
defer hasher.Close()

// Compute hash
hash := hasher.Hash([]byte("data"))  // [32]byte
```

## API Reference

### Types

```go
// Hasher - Main type, safe for concurrent use
type Hasher struct { /* ... */ }

// Config - Configuration for hasher creation
type Config struct {
    Mode     Mode    // LightMode or FastMode
    Flags    Flags   // CPU feature flags (usually FlagDefault)
    CacheKey []byte  // Seed for cache generation (required)
}

// Mode - Operating mode
type Mode int
const (
    LightMode Mode = iota  // 256 MB memory, slower
    FastMode               // 2+ GB memory, faster
)

// Flags - CPU optimization flags
type Flags uint32
const (
    FlagDefault Flags = 0      // Automatic detection
    FlagAES     Flags = 1 << 0 // Hardware AES support
)
```

### Functions

```go
// Create new hasher
func New(config Config) (*Hasher, error)

// Compute hash (thread-safe)
func (h *Hasher) Hash(input []byte) [32]byte

// Update cache key (expensive operation)
func (h *Hasher) UpdateCacheKey(newKey []byte) error

// Check if hasher is ready
func (h *Hasher) IsReady() bool

// Release resources
func (h *Hasher) Close() error
```

## Common Patterns

### Single Hash

```go
func computeHash(data []byte) [32]byte {
    h, _ := randomx.New(randomx.Config{
        Mode:     randomx.LightMode,
        CacheKey: []byte("seed"),
    })
    defer h.Close()
    return h.Hash(data)
}
```

### Concurrent Mining

```go
hasher, _ := randomx.New(config)
defer hasher.Close()

for i := 0; i < numWorkers; i++ {
    go func(id int) {
        for nonce := id; ; nonce += numWorkers {
            hash := hasher.Hash(makeInput(nonce))
            if meetsTarget(hash) {
                reportSolution(nonce)
                return
            }
        }
    }(i)
}
```

### Epoch Updates (Monero-style)

```go
type Miner struct {
    hasher *randomx.Hasher
    mu     sync.RWMutex
}

func (m *Miner) UpdateEpoch(newSeed []byte) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.hasher.UpdateCacheKey(newSeed)
}

func (m *Miner) Hash(data []byte) [32]byte {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.hasher.Hash(data)
}
```

## Performance Tips

### Choose the Right Mode

```go
// For mining/high throughput
Mode: randomx.FastMode  // 2-4x faster, needs 2+ GB RAM

// For memory-constrained environments
Mode: randomx.LightMode // Uses only ~256 MB RAM
```

### Reuse Hashers

```go
// ❌ BAD: Create hasher per hash (slow)
for i := 0; i < 1000; i++ {
    h, _ := randomx.New(config)
    hash := h.Hash(data)
    h.Close()
}

// ✅ GOOD: Reuse hasher (fast)
h, _ := randomx.New(config)
defer h.Close()
for i := 0; i < 1000; i++ {
    hash := h.Hash(data)
}
```

### Concurrent Workers

```go
// Optimal: One worker per CPU core
numWorkers := runtime.NumCPU()

// All workers share same hasher (thread-safe)
hasher, _ := randomx.New(config)
defer hasher.Close()

for i := 0; i < numWorkers; i++ {
    go worker(hasher, i)
}
```

## Error Handling

```go
// Validate configuration
if err := config.Validate(); err != nil {
    return fmt.Errorf("invalid config: %w", err)
}

// Handle creation errors
hasher, err := randomx.New(config)
if err != nil {
    return fmt.Errorf("hasher creation failed: %w", err)
}
defer hasher.Close()

// Handle cache key update errors
if err := hasher.UpdateCacheKey(newKey); err != nil {
    return fmt.Errorf("cache key update failed: %w", err)
}
```

## Testing

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem

# Check coverage
go test -cover ./...
```

## Benchmarks

```bash
# Hash performance
go test -bench=BenchmarkHasher_Hash -benchmem

# Parallel performance
go test -bench=BenchmarkHasher_Hash_Parallel -benchmem

# Cache creation
go test -bench=BenchmarkCacheCreation -benchmem
```

## Examples

```bash
# Simple hasher
go run ./examples/simple -mode=light -input="test"

# Mining simulation
go run ./examples/mining -workers=4 -target="00000000"
```

## Memory Usage

| Mode | Cache | Dataset | Per Worker | Total |
|------|-------|---------|------------|-------|
| Light | 256 KB | - | 2 MB | ~2.3 MB + (2MB × workers) |
| Fast | 256 KB | 2080 MB | 2 MB | ~2082 MB + (2MB × workers) |

## Performance

| Mode | Single Core | Parallel (4 cores) |
|------|------------|-------------------|
| Light | 50-200 H/s | 200-800 H/s |
| Fast | 500-2000 H/s | 2000-8000 H/s |

*Performance varies by CPU. Values for reference only.*

## Common Errors

### "cache key must not be empty"
```go
// ❌ BAD
config := randomx.Config{Mode: randomx.FastMode}

// ✅ GOOD
config := randomx.Config{
    Mode:     randomx.FastMode,
    CacheKey: []byte("seed"),
}
```

### "Hash called on closed hasher" (panic)
```go
// ❌ BAD
hasher.Close()
hasher.Hash(data)  // PANIC!

// ✅ GOOD
if hasher.IsReady() {
    hash := hasher.Hash(data)
}
```

### Out of memory (fast mode)
```go
// If you don't have 2+ GB RAM available:
config := randomx.Config{
    Mode:     randomx.LightMode,  // Use light mode
    CacheKey: seed,
}
```

## Troubleshooting

### Slow Performance

1. **Use Fast Mode**: If you have 2+ GB RAM
2. **Reuse Hasher**: Don't create new hasher per hash
3. **Parallel Workers**: Use multiple goroutines
4. **Check CPU**: Ensure AES-NI support

### High Memory Usage

1. **Use Light Mode**: If memory constrained
2. **Limit Workers**: Each worker uses ~2 MB
3. **Monitor Pools**: VM and scratchpad are pooled

### Race Conditions

```bash
# Always test concurrent code with race detector
go test -race ./...
```

## Dependencies

| Package | Version | License | Purpose |
|---------|---------|---------|---------|
| golang.org/x/crypto/blake2b | v0.31.0 | BSD-3-Clause | Blake2b hashing |
| golang.org/x/crypto/argon2 | v0.31.0 | BSD-3-Clause | Cache generation |
| crypto/aes | stdlib | BSD-3-Clause | AES operations |

## Resources

- [Full Documentation](README.md)
- [Architecture Guide](ARCHITECTURE.md)
- [Contributing Guide](CONTRIBUTING.md)
- [RandomX Spec](https://github.com/tevador/RandomX/blob/master/doc/specs.md)

## License

MIT License - See [LICENSE](LICENSE) file

---

**Quick Questions?**
- Installation issues: Check Go version (need 1.21+)
- Performance issues: Use fast mode with 2+ GB RAM
- Memory issues: Use light mode
- Concurrency: Hasher is thread-safe, reuse it!

---

For detailed examples and usage patterns, see [README.md](README.md)
