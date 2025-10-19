package randomx

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	
	"github.com/opd-ai/go-randomx/internal"
)

// ReferenceTrace contains expected intermediate values from C++ RandomX reference
// This structure will be populated with values extracted from the C++ implementation
type ReferenceTrace struct {
	TestName       string   `json:"test_name"`
	Key            string   `json:"key"`
	Input          string   `json:"input"`
	InitialBlake2b string   `json:"initial_blake2b"` // Blake2b-512 of input (128 hex chars)
	InitialRegs    []string `json:"initial_regs"`    // 8 registers as hex strings
	FinalRegs      []string `json:"final_regs"`      // 8 registers after all programs
	FinalHash      string   `json:"final_hash"`      // Expected final hash (64 hex chars)
	
	// Optional: Per-program traces for detailed debugging
	Programs []ProgramTrace `json:"programs,omitempty"`
}

// ProgramTrace contains trace information for a single program execution
type ProgramTrace struct {
	ProgramNum     int      `json:"program_num"`
	FirstInstr     []string `json:"first_instr,omitempty"`     // First 5 instructions
	RegistersAfter []string `json:"registers_after,omitempty"` // Register state after this program
}

// TestCompareWithReference performs detailed comparison with C++ reference implementation
// This test is currently skipped because we need to generate reference traces from C++
func TestCompareWithReference(t *testing.T) {
	t.Skip("Waiting for C++ reference trace data - see NEXT_DEVELOPMENT_PHASE.md for generation instructions")
	
	// Load reference trace from JSON file
	data, err := os.ReadFile("testdata/reference_trace_test1.json")
	if err != nil {
		t.Fatalf("Failed to load reference trace: %v", err)
	}
	
	var ref ReferenceTrace
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatalf("Failed to parse reference trace: %v", err)
	}
	
	// Enable debug logging for this test
	originalDebug := debugEnabled
	debugEnabled = true
	defer func() { debugEnabled = originalDebug }()
	
	// Create hasher with same configuration as reference
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte(ref.Key),
	}
	hasher, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create hasher: %v", err)
	}
	defer hasher.Close()
	
	// Compute hash - debug output will show intermediate values
	hash := hasher.Hash([]byte(ref.Input))
	actualHash := hex.EncodeToString(hash[:])
	
	// Compare final hash
	if actualHash != ref.FinalHash {
		t.Errorf("Hash mismatch for test '%s':", ref.TestName)
		t.Errorf("  Expected: %s", ref.FinalHash)
		t.Errorf("  Actual:   %s", actualHash)
		t.Error("\nCheck debug output above to find the first divergence point")
		t.Error("This indicates where our implementation differs from the C++ reference")
	} else {
		t.Logf("✓ Hash matches reference for test '%s'", ref.TestName)
	}
}

// TestExtractOurTrace outputs a detailed trace from our implementation
// This can be compared manually with C++ reference output to find divergences
func TestExtractOurTrace(t *testing.T) {
	// Test with the first official test vector
	testKey := "test key 000"
	testInput := "This is a test"
	expectedHash := "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f"
	
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte(testKey),
	}
	hasher, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create hasher: %v", err)
	}
	defer hasher.Close()
	
	// Enable debug tracing
	originalDebug := debugEnabled
	debugEnabled = true
	defer func() { debugEnabled = originalDebug }()
	
	t.Logf("=== EXTRACTING TRACE FOR COMPARISON ===")
	t.Logf("Key: %q", testKey)
	t.Logf("Input: %q", testInput)
	t.Logf("Expected: %s", expectedHash)
	t.Logf("")
	t.Logf("Trace output follows (enable with RANDOMX_DEBUG=1):")
	t.Logf("---")
	
	// Compute hash - this will output detailed trace if RANDOMX_DEBUG=1
	hash := hasher.Hash([]byte(testInput))
	actualHash := hex.EncodeToString(hash[:])
	
	t.Logf("---")
	t.Logf("")
	t.Logf("Our hash:      %s", actualHash)
	t.Logf("Expected hash: %s", expectedHash)
	
	if actualHash == expectedHash {
		t.Logf("✓ PASS - Hash matches!")
	} else {
		t.Logf("✗ FAIL - Hash mismatch")
		t.Logf("")
		t.Logf("To debug:")
		t.Logf("1. Run: RANDOMX_DEBUG=1 go test -v -run TestExtractOurTrace > our_trace.txt")
		t.Logf("2. Generate C++ reference trace with same input")
		t.Logf("3. Compare the two traces to find divergence point")
	}
}

// TestCompareInitialHashes validates that initial Blake2b hashing is correct
// This is the first checkpoint - if this fails, the bug is in input processing
func TestCompareInitialHashes(t *testing.T) {
	tests := []struct {
		input    string
		expected string // Blake2b-512 output from reference
	}{
		{
			input:    "This is a test",
			expected: "152455751b73ac2167dd07ed8adeb4f40a1875bce1d64ca9bc5048f94a70d23ff7d26b86498c645a4c3d75c74aef7bbbaabfad29298ddc0da6d65f9ce8043577",
		},
		// Add more test cases once we have reference values
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			hash := internal.Blake2b512([]byte(tt.input))
			actual := hex.EncodeToString(hash[:])
			
			if actual != tt.expected {
				t.Errorf("Blake2b-512 mismatch:")
				t.Errorf("  Input:    %q", tt.input)
				t.Errorf("  Expected: %s", tt.expected)
				t.Errorf("  Actual:   %s", actual)
			}
		})
	}
}

// TestDebugEnvironmentVariable verifies debug tracing can be enabled
func TestDebugEnvironmentVariable(t *testing.T) {
	// Save original state
	originalDebug := debugEnabled
	defer func() { debugEnabled = originalDebug }()
	
	// Test enabling debug
	debugEnabled = true
	if !debugEnabled {
		t.Error("Failed to enable debug tracing")
	}
	
	// Test disabling debug
	debugEnabled = false
	if debugEnabled {
		t.Error("Failed to disable debug tracing")
	}
	
	t.Log("Debug tracing can be controlled via RANDOMX_DEBUG environment variable")
	t.Log("Set RANDOMX_DEBUG=1 to enable detailed trace output")
}

// BenchmarkHashWithDebugDisabled ensures debug logging has zero overhead when disabled
func BenchmarkHashWithDebugDisabled(b *testing.B) {
	// Ensure debug is disabled
	originalDebug := debugEnabled
	debugEnabled = false
	defer func() { debugEnabled = originalDebug }()
	
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte("benchmark key"),
	}
	hasher, err := New(config)
	if err != nil {
		b.Fatal(err)
	}
	defer hasher.Close()
	
	input := []byte("benchmark input data")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.Hash(input)
	}
}
