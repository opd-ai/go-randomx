# Contributing to go-randomx

Thank you for considering contributing to go-randomx! This document provides guidelines and best practices for contributing to the project.

## Code of Conduct

Be respectful, constructive, and professional in all interactions.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Basic understanding of RandomX algorithm (see [RandomX specification](https://github.com/tevador/RandomX/blob/master/doc/specs.md))

### Getting Started

```bash
# Clone the repository
git clone https://github.com/opd-ai/go-randomx.git
cd go-randomx

# Download dependencies
go mod download

# Run tests
go test -v ./...

# Run tests with race detector
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem
```

## Contribution Guidelines

### Code Style

1. **Function Length**: Maximum 50 lines per function
   - Rationale: Improves readability, enables inlining
   - Exception: Complex algorithms may split across multiple functions

2. **Error Handling**:
   ```go
   // Good: Return errors
   func doSomething() error {
       if err := validate(); err != nil {
           return fmt.Errorf("validation failed: %w", err)
       }
       return nil
   }
   
   // Bad: Panic for recoverable errors
   func doSomething() {
       if err := validate(); err != nil {
           panic(err) // Only panic for programmer errors!
       }
   }
   ```

3. **Naming Conventions**:
   - Exported functions: `PascalCase`
   - Private functions: `camelCase`
   - Memory helpers: `verb + noun + detail` (e.g., `allocateScratchpad`, `poolGetVM`)
   - Constants: `camelCase` or `SCREAMING_SNAKE_CASE` for groups

4. **Documentation**:
   ```go
   // Good: Godoc-style comments
   // HashData computes the RandomX hash of data using the configured hasher.
   // It is safe to call concurrently from multiple goroutines.
   func (h *Hasher) Hash(data []byte) [32]byte
   
   // Bad: No documentation
   func (h *Hasher) Hash(data []byte) [32]byte
   ```

### Testing Requirements

1. **Unit Tests**:
   - Test each function in isolation
   - Cover happy path and error cases
   - Use table-driven tests for multiple scenarios

   ```go
   func TestConfigValidation(t *testing.T) {
       tests := []struct {
           name    string
           config  Config
           wantErr bool
       }{
           {"valid config", Config{...}, false},
           {"invalid config", Config{...}, true},
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               err := tt.config.Validate()
               if (err != nil) != tt.wantErr {
                   t.Errorf("unexpected error: %v", err)
               }
           })
       }
   }
   ```

2. **Benchmark Tests**:
   - Include benchmarks for performance-critical functions
   - Use `b.ReportAllocs()` to track allocations
   - Run with `-benchmem` flag

   ```go
   func BenchmarkHasher_Hash(b *testing.B) {
       hasher := setupHasher(b)
       defer hasher.Close()
       
       input := []byte("benchmark input")
       
       b.ResetTimer()
       b.ReportAllocs()
       
       for i := 0; i < b.N; i++ {
           _ = hasher.Hash(input)
       }
   }
   ```

3. **Race Detection**:
   - All concurrent code must pass race detector
   - Run: `go test -race ./...`

4. **Coverage**:
   - Aim for >80% coverage
   - Check: `go test -cover ./...`

### Cryptographic Guidelines

**CRITICAL**: Do NOT implement custom cryptographic primitives.

1. **Use Existing Libraries**:
   ```go
   // Good: Use golang.org/x/crypto
   import "golang.org/x/crypto/blake2b"
   hash := blake2b.Sum256(data)
   
   // Bad: Custom hash implementation
   func myBlake2b(data []byte) []byte {
       // Don't do this!
   }
   ```

2. **Library Selection Criteria**:
   - Prefer Go standard library (`crypto/*`)
   - Use `golang.org/x/crypto` for extended algorithms
   - Check library licenses (must be MIT-compatible)
   - Verify active maintenance and security track record

3. **Document Crypto Choices**:
   ```go
   // Blake2b512 computes a 512-bit Blake2b hash.
   // Uses golang.org/x/crypto/blake2b (BSD-3-Clause).
   // Hardware acceleration automatically enabled when available.
   func Blake2b512(data []byte) [64]byte
   ```

### Performance Guidelines

1. **Zero Allocations in Hot Paths**:
   ```go
   // Verify with: go test -bench=BenchmarkHasher_Hash -benchmem
   // Should show: 0 B/op  0 allocs/op (after warmup)
   ```

2. **Use Object Pooling**:
   ```go
   var vmPool = sync.Pool{
       New: func() interface{} {
           return &virtualMachine{
               mem: allocateScratchpad(),
           }
       },
   }
   ```

3. **Benchmark Before and After**:
   ```bash
   # Before changes
   go test -bench=BenchmarkHasher_Hash -benchmem > before.txt
   
   # Make changes
   
   # After changes
   go test -bench=BenchmarkHasher_Hash -benchmem > after.txt
   
   # Compare
   benchstat before.txt after.txt
   ```

### Memory Safety

1. **Avoid `unsafe` Package**:
   - Only use for essential alignment operations
   - Document why unsafe is necessary
   - Ensure bounds checks are in place

2. **Secure Memory Handling**:
   ```go
   // Good: Clear sensitive data
   func (c *cache) release() {
       if c.data != nil {
           zeroBytes(c.data)
           c.data = nil
       }
   }
   
   // Bad: Leave sensitive data in memory
   func (c *cache) release() {
       c.data = nil // Data still in memory!
   }
   ```

3. **Concurrent Access**:
   - Use `sync.RWMutex` for read-heavy workloads
   - Document thread-safety guarantees
   - Test with race detector

## Pull Request Process

### Before Submitting

1. **Run Full Test Suite**:
   ```bash
   go test -v -race -cover ./...
   go test -bench=. -benchmem ./...
   ```

2. **Format Code**:
   ```bash
   gofmt -w .
   go vet ./...
   ```

3. **Update Documentation**:
   - Update godoc comments for changed functions
   - Update README.md if API changes
   - Update ARCHITECTURE.md for design changes

4. **Add Tests**:
   - Unit tests for new functions
   - Integration tests for new features
   - Benchmarks for performance-critical code

### PR Description Template

```markdown
## Description
Brief description of changes.

## Motivation
Why is this change needed?

## Changes
- List of specific changes
- Each on its own line

## Testing
- [ ] Unit tests added/updated
- [ ] Benchmarks added/updated
- [ ] Race detector clean
- [ ] All tests pass

## Performance Impact
(Include benchmark comparison if applicable)

## Documentation
- [ ] Godoc comments updated
- [ ] README.md updated (if needed)
- [ ] ARCHITECTURE.md updated (if needed)

## Checklist
- [ ] Code follows style guidelines
- [ ] All functions <50 lines
- [ ] No custom cryptography
- [ ] Proper error handling
- [ ] Tests pass
```

### Review Process

1. Automated checks must pass (tests, linting)
2. Code review by maintainer(s)
3. Address review feedback
4. Maintainer approval
5. Merge to main branch

## Types of Contributions

### Bug Fixes

- Include test case demonstrating the bug
- Fix the issue
- Verify test now passes
- Add regression test if applicable

### Performance Improvements

- Include benchmark comparison
- Verify no allocations added to hot path
- Document trade-offs
- Ensure correctness not compromised

### New Features

- Discuss feature in issue first
- Ensure fits project scope (pure-Go, no custom crypto)
- Include comprehensive tests
- Update documentation
- Add examples if user-facing

### Documentation

- Fix typos, clarify explanations
- Add examples
- Improve godoc comments
- Update architecture docs

## Issue Reporting

### Bug Reports

Include:
1. Go version (`go version`)
2. OS and architecture
3. Minimal reproducible example
4. Expected behavior
5. Actual behavior
6. Stack trace if applicable

### Feature Requests

Include:
1. Use case description
2. Proposed API design
3. Alternatives considered
4. Willing to implement? (yes/no)

### Performance Issues

Include:
1. Benchmark results
2. Comparison with expected performance
3. Environment details (CPU, RAM)
4. Profiling data if available

## Development Tips

### Debugging

```bash
# Run single test
go test -v -run=TestHasherNew

# Run with verbose logging
go test -v -race -run=TestHasherConcurrent

# Profile CPU usage
go test -cpuprofile=cpu.prof -bench=BenchmarkHasher_Hash
go tool pprof cpu.prof

# Profile memory usage
go test -memprofile=mem.prof -bench=BenchmarkHasher_Hash
go tool pprof mem.prof
```

### IDE Setup

**VS Code**:
- Install Go extension
- Enable format on save
- Enable run tests on save
- Use gopls language server

**Vim/Neovim**:
- Use vim-go or coc-go
- Enable auto-format
- Use :GoTest for quick testing

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Questions?

- Open an issue for questions about contributing
- Check existing issues for similar discussions
- Read the [ARCHITECTURE.md](ARCHITECTURE.md) for design details

## Resources

- [RandomX Specification](https://github.com/tevador/RandomX/blob/master/doc/specs.md)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Monero RandomX Integration](https://github.com/monero-project/monero/tree/master/src/crypto/randomx)

---

Thank you for contributing to go-randomx! 🎉
