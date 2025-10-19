package randomx

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// CPPReferenceTrace contains expected values from C++ RandomX reference implementation
type CPPReferenceTrace struct {
	TestName string `json:"test_name"`
	Key      string `json:"key"`
	Input    string `json:"input"`
	FinalHash string `json:"final_hash"`
	Note     string `json:"note,omitempty"`
}

// TestCompareWithCPPReference performs detailed comparison with C++ reference implementation
// This test validates that our implementation produces the same final hash as the C++ reference
func TestCompareWithCPPReference(t *testing.T) {
	// Check if reference traces directory exists
	tracesDir := "testdata/reference_traces"
	if _, err := os.Stat(tracesDir); os.IsNotExist(err) {
		t.Skip("Reference traces not generated yet. Run: make generate-cpp-traces")
	}

	// Test files to process
	testFiles := map[string]struct{
		key    string
		input  string
		expect string
	}{
		"basic_test_1.json": {
			key:    "test key 000",
			input:  "This is a test",
			expect: "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f",
		},
		"basic_test_2.json": {
			key:    "test key 000",
			input:  "Lorem ipsum dolor sit amet",
			expect: "300a0adb47603dedb42228ccb2b211104f4da45af709cd7547cd049e9489c969",
		},
		"basic_test_3.json": {
			key:    "test key 000",
			input:  "sed do eiusmod tempor incididunt ut labore et dolore magna aliqua\n",
			expect: "c36d4ed4191e617309867ed66a443be4075014e2b061bcdaf9ce7b721d2b77a8",
		},
		"different_key.json": {
			key:    "test key 001",
			input:  "sed do eiusmod tempor incididunt ut labore et dolore magna aliqua\n",
			expect: "e9ff4503201c0c2cca26d285c93ae883f9b1d30c9eb240b820756f2d5a7905fc",
		},
	}

	for filename, testCase := range testFiles {
		t.Run(filename, func(t *testing.T) {
			// Try to load reference trace
			tracePath := filepath.Join(tracesDir, filename)
			data, err := os.ReadFile(tracePath)
			if err != nil {
				// If file doesn't exist, use expected values directly
				t.Logf("Warning: Reference trace not found, using known expected values")
				runComparisonTest(t, testCase.key, testCase.input, testCase.expect)
				return
			}

			// Parse reference trace
			var ref CPPReferenceTrace
			if err := json.Unmarshal(data, &ref); err != nil {
				t.Fatalf("Failed to parse reference trace: %v", err)
			}

			// Verify reference trace matches expected values
			if ref.Key != testCase.key {
				t.Errorf("Reference trace key mismatch: got %q, want %q", ref.Key, testCase.key)
			}
			if ref.Input != testCase.input {
				t.Errorf("Reference trace input mismatch: got %q, want %q", ref.Input, testCase.input)
			}
			if ref.FinalHash != testCase.expect {
				t.Errorf("Reference trace hash mismatch: got %q, want %q", ref.FinalHash, testCase.expect)
			}

			// Run comparison test
			runComparisonTest(t, ref.Key, ref.Input, ref.FinalHash)
		})
	}
}

// runComparisonTest runs a single test case and compares with expected hash
func runComparisonTest(t *testing.T, key, input, expectedHash string) {
	// Create hasher with same configuration as reference
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte(key),
	}
	hasher, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create hasher: %v", err)
	}
	defer hasher.Close()

	// Compute hash (debug tracing controlled by RANDOMX_DEBUG env var)
	hash := hasher.Hash([]byte(input))
	actualHash := hex.EncodeToString(hash[:])

	// Compare final hash
	if actualHash != expectedHash {
		t.Errorf("Hash mismatch:")
		t.Errorf("  Key:      %q", key)
		t.Errorf("  Input:    %q (len=%d)", input, len(input))
		t.Errorf("  Expected: %s", expectedHash)
		t.Errorf("  Actual:   %s", actualHash)
		t.Error("")
		t.Error("To see detailed trace, run:")
		t.Errorf("  RANDOMX_DEBUG=1 go test -v -run %s", t.Name())
	} else {
		t.Logf("✓ Hash matches C++ reference")
	}
}

// TestExtractGoTrace outputs a detailed trace from our implementation
// This can be compared manually with C++ reference output to find divergences
// Run with: RANDOMX_DEBUG=1 go test -v -run TestExtractGoTrace
func TestExtractGoTrace(t *testing.T) {
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

	// Enable debug tracing (if not already enabled by env var)
	if !debugEnabled {
		t.Log("Tip: Run with RANDOMX_DEBUG=1 to see detailed trace output")
	}

	// Compute hash
	hash := hasher.Hash([]byte(testInput))
	actualHash := hex.EncodeToString(hash[:])

	t.Logf("Test configuration:")
	t.Logf("  Key:      %q", testKey)
	t.Logf("  Input:    %q", testInput)
	t.Logf("  Expected: %s", expectedHash)
	t.Logf("  Actual:   %s", actualHash)

	if actualHash != expectedHash {
		t.Errorf("Hash mismatch - this is expected until bug is fixed")
		t.Error("Check debug output above to identify divergence point")
	}
}

// TestDeterministicOutput verifies that our implementation produces consistent output
// This is a sanity check - the output should be the same every time for the same input
func TestDeterministicOutput(t *testing.T) {
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte("test key 000"),
	}

	// Run same test 3 times
	var hashes [3][32]byte
	for i := 0; i < 3; i++ {
		hasher, err := New(config)
		if err != nil {
			t.Fatalf("Failed to create hasher: %v", err)
		}

		hashes[i] = hasher.Hash([]byte("This is a test"))
		hasher.Close()
	}

	// Verify all hashes are identical
	for i := 1; i < 3; i++ {
		if hashes[i] != hashes[0] {
			t.Errorf("Non-deterministic output detected:")
			t.Errorf("  Run 0: %x", hashes[0])
			t.Errorf("  Run %d: %x", i, hashes[i])
			t.Fatal("Implementation must be deterministic")
		}
	}

	t.Logf("✓ Implementation is deterministic: %x", hashes[0])
}
