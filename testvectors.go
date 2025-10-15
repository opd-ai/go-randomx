package randomx

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// TestVector represents a single RandomX test case from the reference implementation.
// These vectors are used to validate hash compatibility with the official RandomX C++ implementation.
type TestVector struct {
	Name     string `json:"name"`
	Mode     string `json:"mode"`
	Key      string `json:"key"`
	Input    string `json:"input"`
	InputHex string `json:"input_hex,omitempty"` // Alternative hex-encoded input
	Expected string `json:"expected"`            // Hex-encoded expected hash
}

// TestVectorSuite contains all test vectors with metadata about their source.
type TestVectorSuite struct {
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Source      string       `json:"source,omitempty"`
	License     string       `json:"license,omitempty"`
	Vectors     []TestVector `json:"vectors"`
}

// LoadTestVectors loads test vectors from a JSON file.
// Returns an error if the file cannot be read or parsed.
//
// This is used internally for testing but exported for potential external validation tools.
func LoadTestVectors(path string) (*TestVectorSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test vectors: %w", err)
	}

	var suite TestVectorSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse test vectors: %w", err)
	}

	return &suite, nil
}

// GetInput returns the decoded input bytes for a test vector.
// If InputHex is set, it decodes from hex, otherwise uses Input as UTF-8.
func (tv *TestVector) GetInput() ([]byte, error) {
	if tv.InputHex != "" {
		input, err := hex.DecodeString(tv.InputHex)
		if err != nil {
			return nil, fmt.Errorf("invalid input hex: %w", err)
		}
		return input, nil
	}
	return []byte(tv.Input), nil
}

// GetExpected returns the decoded expected hash bytes.
func (tv *TestVector) GetExpected() ([]byte, error) {
	expected, err := hex.DecodeString(tv.Expected)
	if err != nil {
		return nil, fmt.Errorf("invalid expected hash: %w", err)
	}
	if len(expected) != 32 {
		return nil, fmt.Errorf("expected hash must be 32 bytes, got %d", len(expected))
	}
	return expected, nil
}

// GetMode returns the Mode value for this test vector.
func (tv *TestVector) GetMode() (Mode, error) {
	switch tv.Mode {
	case "light":
		return LightMode, nil
	case "fast":
		return FastMode, nil
	default:
		return 0, fmt.Errorf("unknown mode: %s", tv.Mode)
	}
}
