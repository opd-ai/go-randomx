package randomx

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Test basic configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid fast mode config",
			config: Config{
				Mode:     FastMode,
				CacheKey: []byte("test key"),
			},
			wantErr: false,
		},
		{
			name: "valid light mode config",
			config: Config{
				Mode:     LightMode,
				CacheKey: []byte("test key"),
			},
			wantErr: false,
		},
		{
			name: "empty cache key",
			config: Config{
				Mode:     FastMode,
				CacheKey: []byte{},
			},
			wantErr: true,
		},
		{
			name: "nil cache key",
			config: Config{
				Mode:     FastMode,
				CacheKey: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test Mode.String()
func TestModeString(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{LightMode, "LightMode"},
		{FastMode, "FastMode"},
		{Mode(99), "Mode(99)"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("Mode.String() = %v, want %v", got, tt.want)
		}
	}
}

// Test hasher creation and basic operations
func TestHasherNew(t *testing.T) {
	config := Config{
		Mode:     LightMode, // Use light mode for faster tests
		CacheKey: []byte("test seed"),
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer hasher.Close()

	if !hasher.IsReady() {
		t.Error("hasher should be ready after creation")
	}
}

// Test hashing functionality
func TestHasherHash(t *testing.T) {
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte("RandomX example key"),
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer hasher.Close()

	// Test hashing different inputs
	tests := []struct {
		name  string
		input []byte
	}{
		{"empty input", []byte{}},
		{"simple text", []byte("Hello, RandomX!")},
		{"longer text", []byte("This is a longer input to test the RandomX hashing function")},
		{"binary data", []byte{0x00, 0xFF, 0xAA, 0x55, 0x12, 0x34, 0x56, 0x78}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hasher.Hash(tt.input)

			// Verify hash is deterministic
			hash2 := hasher.Hash(tt.input)
			if hash != hash2 {
				t.Error("hash should be deterministic")
			}

			// Verify hash length
			if len(hash) != 32 {
				t.Errorf("hash length = %d, want 32", len(hash))
			}
		})
	}
}

// Test concurrent hashing
func TestHasherConcurrent(t *testing.T) {
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte("concurrent test"),
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer hasher.Close()

	const numGoroutines = 10
	const numHashes = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numHashes; j++ {
				input := []byte{byte(id), byte(j)}
				_ = hasher.Hash(input)
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// Test cache key update
func TestHasherUpdateCacheKey(t *testing.T) {
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte("initial key"),
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer hasher.Close()

	input := []byte("test input")
	hash1 := hasher.Hash(input)

	// Update cache key
	err = hasher.UpdateCacheKey([]byte("new key"))
	if err != nil {
		t.Fatalf("UpdateCacheKey() error = %v", err)
	}

	hash2 := hasher.Hash(input)

	// Hash should be different with different cache key
	if hash1 == hash2 {
		t.Error("hash should change after cache key update")
	}

	// Update to same key should be no-op
	err = hasher.UpdateCacheKey([]byte("new key"))
	if err != nil {
		t.Errorf("UpdateCacheKey() with same key error = %v", err)
	}

	hash3 := hasher.Hash(input)
	if hash2 != hash3 {
		t.Error("hash should be same when cache key doesn't change")
	}
}

// Test closing hasher
func TestHasherClose(t *testing.T) {
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte("close test"),
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = hasher.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if hasher.IsReady() {
		t.Error("hasher should not be ready after close")
	}

	// Closing again should be no-op
	err = hasher.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

// TestHasherTestVectors validates against RandomX reference implementation.
//
// CRITICAL: These test vectors must be validated against the official RandomX
// C++ reference implementation (github.com/tevador/RandomX).
//
// TODO: Add real test vectors from RandomX test suite. Until then, this test
// only verifies deterministic behavior (same input produces same output).
//
// Without proper test vectors, there is NO GUARANTEE that this implementation
// produces hashes compatible with Monero or other RandomX-based systems.
func TestHasherTestVectors(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		input    string
		expected string // hex encoded expected hash
	}{
		{
			name:  "test vector 1",
			key:   "test key 000",
			input: "This is a test",
			// TODO: Replace with actual RandomX reference output
			// Run: echo -n "This is a test" | randomx-tests --key "test key 000"
			expected: "",
		},
		// TODO: Add more test vectors covering:
		// - Empty input
		// - Long input (>1MB)
		// - Binary data
		// - Different key lengths
		// - Edge cases from RandomX specification
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Mode:     LightMode,
				CacheKey: []byte(tt.key),
			}

			hasher, err := New(config)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			defer hasher.Close()

			hash := hasher.Hash([]byte(tt.input))

			if tt.expected != "" {
				expectedBytes, err := hex.DecodeString(tt.expected)
				if err != nil {
					t.Fatalf("invalid expected hash: %v", err)
				}
				if !bytes.Equal(hash[:], expectedBytes) {
					t.Errorf("hash mismatch:\ngot:  %x\nwant: %x",
						hash, expectedBytes)
				}
			}

			// Verify determinism
			hash2 := hasher.Hash([]byte(tt.input))
			if hash != hash2 {
				t.Error("hash should be deterministic")
			}
		})
	}
}

// Test hasher usage after close panics
func TestHasherPanicAfterClose(t *testing.T) {
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte("panic test"),
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	hasher.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Hash() should panic after Close()")
		}
	}()

	hasher.Hash([]byte("test"))
}

// TestQuickStartExample validates the hash output from the README Quick Start example.
// This ensures the documented example produces the expected hash.
func TestQuickStartExample(t *testing.T) {
	config := Config{
		Mode:     FastMode,
		CacheKey: []byte("RandomX example key"),
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer hasher.Close()

	hash := hasher.Hash([]byte("RandomX example input"))
	expected := "6e2fae47ac7365c1008c046f88dcb5243a7cc8d500616a4a9afcc881f470fb3b"
	actual := hex.EncodeToString(hash[:])

	if actual != expected {
		t.Errorf("Quick Start example hash mismatch:\ngot:  %s\nwant: %s", actual, expected)
		t.Log("Note: This may indicate the example hash in README.md needs to be updated")
	}
}

// TestHasherZeroAllocations verifies allocation behavior of Hash().
// Documents current allocation count for tracking optimization progress.
func TestHasherZeroAllocations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping allocation test in short mode")
	}

	hasher, err := New(Config{
		Mode:     LightMode,
		CacheKey: []byte("allocation test"),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer hasher.Close()

	input := []byte("test input for allocation check")

	// Warmup: Allow VM pool to initialize
	for i := 0; i < 5; i++ {
		_ = hasher.Hash(input)
	}

	// Measure allocations
	allocs := testing.AllocsPerRun(10, func() {
		_ = hasher.Hash(input)
	})

	// Document current allocation behavior
	t.Logf("Hash() allocations per call: %.2f", allocs)

	// Current implementation allocates ~18 times per call due to:
	// - Program generation (program struct + entropy buffer)
	// - Internal Blake2b operations
	// Future optimization could reduce this through pooling.

	if allocs > 25 {
		t.Errorf("Hash() allocated %.2f times per run, expected ~18", allocs)
		t.Error("Allocation count has increased significantly - check for regressions")
	}
}

// Test bytesEqual helper function
func TestBytesEqual(t *testing.T) {
	tests := []struct {
		name string
		a    []byte
		b    []byte
		want bool
	}{
		{"equal slices", []byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{"different values", []byte{1, 2, 3}, []byte{1, 2, 4}, false},
		{"different lengths", []byte{1, 2, 3}, []byte{1, 2}, false},
		{"both empty", []byte{}, []byte{}, true},
		{"both nil", nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bytesEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("bytesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
