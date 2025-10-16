package randomx

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadTestVectors verifies test vector loading functionality.
func TestLoadTestVectors(t *testing.T) {
	// Test loading the official vectors
	suite, err := LoadTestVectors("testdata/randomx_vectors.json")
	if err != nil {
		t.Fatalf("LoadTestVectors() error = %v", err)
	}

	if suite.Version == "" {
		t.Error("suite.Version should not be empty")
	}

	if len(suite.Vectors) == 0 {
		t.Fatal("suite.Vectors should not be empty")
	}

	t.Logf("Loaded %d test vectors from version %s", len(suite.Vectors), suite.Version)
}

// TestLoadTestVectors_FileNotFound verifies error handling for missing files.
func TestLoadTestVectors_FileNotFound(t *testing.T) {
	_, err := LoadTestVectors("nonexistent.json")
	if err == nil {
		t.Error("LoadTestVectors() should return error for nonexistent file")
	}
}

// TestLoadTestVectors_InvalidJSON verifies error handling for invalid JSON.
func TestLoadTestVectors_InvalidJSON(t *testing.T) {
	// Create a temporary invalid JSON file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(tmpFile, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err = LoadTestVectors(tmpFile)
	if err == nil {
		t.Error("LoadTestVectors() should return error for invalid JSON")
	}
}

// TestTestVector_GetInput verifies input extraction from test vectors.
func TestTestVector_GetInput(t *testing.T) {
	tests := []struct {
		name    string
		tv      TestVector
		want    []byte
		wantErr bool
	}{
		{
			name: "string_input",
			tv: TestVector{
				Input: "test",
			},
			want:    []byte("test"),
			wantErr: false,
		},
		{
			name: "hex_input",
			tv: TestVector{
				InputHex: "deadbeef",
			},
			want:    []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr: false,
		},
		{
			name: "invalid_hex",
			tv: TestVector{
				InputHex: "invalid",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.tv.GetInput()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !bytes.Equal(got, tt.want) {
				t.Errorf("GetInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTestVector_GetExpected verifies expected hash extraction.
func TestTestVector_GetExpected(t *testing.T) {
	tests := []struct {
		name    string
		tv      TestVector
		wantLen int
		wantErr bool
	}{
		{
			name: "valid_hash",
			tv: TestVector{
				Expected: "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f",
			},
			wantLen: 32,
			wantErr: false,
		},
		{
			name: "invalid_hex",
			tv: TestVector{
				Expected: "invalid",
			},
			wantLen: 0,
			wantErr: true,
		},
		{
			name: "wrong_length",
			tv: TestVector{
				Expected: "deadbeef",
			},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.tv.GetExpected()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetExpected() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("GetExpected() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// TestTestVector_GetMode verifies mode parsing.
func TestTestVector_GetMode(t *testing.T) {
	tests := []struct {
		name    string
		tv      TestVector
		want    Mode
		wantErr bool
	}{
		{
			name:    "light_mode",
			tv:      TestVector{Mode: "light"},
			want:    LightMode,
			wantErr: false,
		},
		{
			name:    "fast_mode",
			tv:      TestVector{Mode: "fast"},
			want:    FastMode,
			wantErr: false,
		},
		{
			name:    "invalid_mode",
			tv:      TestVector{Mode: "invalid"},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.tv.GetMode()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GetMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOfficialVectors validates against official RandomX test vectors.
// This is the CRITICAL test that verifies hash compatibility with the reference implementation.
func TestOfficialVectors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping official test vectors in short mode")
	}

	suite, err := LoadTestVectors("testdata/randomx_vectors.json")
	if err != nil {
		t.Fatalf("Failed to load test vectors: %v", err)
	}

	t.Logf("Testing against RandomX version: %s", suite.Version)
	t.Logf("Description: %s", suite.Description)
	t.Logf("Running %d test vectors", len(suite.Vectors))

	for _, tv := range suite.Vectors {
		t.Run(tv.Name, func(t *testing.T) {
			// Parse mode
			mode, err := tv.GetMode()
			if err != nil {
				t.Fatalf("GetMode() failed: %v", err)
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

			// Get input
			input, err := tv.GetInput()
			if err != nil {
				t.Fatalf("GetInput() failed: %v", err)
			}

			// Get expected hash
			expected, err := tv.GetExpected()
			if err != nil {
				t.Fatalf("GetExpected() failed: %v", err)
			}

			// Compute hash
			hash := hasher.Hash(input)

			// Validate
			if !bytes.Equal(hash[:], expected) {
				t.Errorf("Hash mismatch for '%s':", tv.Name)
				t.Errorf("  Got:      %s", hex.EncodeToString(hash[:]))
				t.Errorf("  Expected: %s", hex.EncodeToString(expected))
				t.Errorf("  Mode:     %s", tv.Mode)
				t.Errorf("  Key:      %q", tv.Key)
				t.Errorf("  Input:    %q (len=%d)", string(input), len(input))
			}
		})
	}
}

// TestOfficialVectors_Determinism verifies that the same input always produces the same output.
func TestOfficialVectors_Determinism(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test vector determinism test in short mode")
	}

	suite, err := LoadTestVectors("testdata/randomx_vectors.json")
	if err != nil {
		t.Fatalf("Failed to load test vectors: %v", err)
	}

	// Use the first vector for determinism testing
	if len(suite.Vectors) == 0 {
		t.Fatal("No test vectors available")
	}

	tv := suite.Vectors[0]
	mode, _ := tv.GetMode()
	input, _ := tv.GetInput()

	config := Config{
		Mode:     mode,
		CacheKey: []byte(tv.Key),
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer hasher.Close()

	// Hash the same input 10 times
	const iterations = 10
	var hashes [][32]byte
	for i := 0; i < iterations; i++ {
		hash := hasher.Hash(input)
		hashes = append(hashes, hash)
	}

	// Verify all hashes are identical
	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			t.Errorf("Hash mismatch at iteration %d:", i)
			t.Errorf("  First:   %s", hex.EncodeToString(hashes[0][:]))
			t.Errorf("  Current: %s", hex.EncodeToString(hashes[i][:]))
		}
	}
}
