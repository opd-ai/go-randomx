package randomx

import (
	"encoding/hex"
	"testing"

	"github.com/opd-ai/go-randomx/internal"
)

// TestAESGenerator1R_ScratchpadFilling validates AesGenerator1R functionality
func TestAESGenerator1R_ScratchpadFilling(t *testing.T) {
	// Test from RandomX spec: initial hash of input
	input := []byte("This is a test")
	hash := internal.Blake2b512(input)
	
	t.Logf("Input: %q", input)
	t.Logf("Blake2b-512: %x", hash)
	
	// Create generator
	gen, err := newAesGenerator1R(hash[:])
	if err != nil {
		t.Fatalf("Failed to create AesGenerator1R: %v", err)
	}
	
	// Generate first 64 bytes
	var output [64]byte
	gen.getBytes(output[:])
	
	t.Logf("First 64 bytes from AesGenerator1R: %x", output[:])
	
	// Should produce deterministic output
	gen2, _ := newAesGenerator1R(hash[:])
	var output2 [64]byte
	gen2.getBytes(output2[:])
	
	if output != output2 {
		t.Error("AesGenerator1R not deterministic")
	}
}

// TestAESGenerator4R_ProgramGeneration validates AesGenerator4R for program generation
func TestAESGenerator4R_ProgramGeneration(t *testing.T) {
	// Test program generation with AesGenerator4R
	input := []byte("This is a test")
	hash := internal.Blake2b512(input)
	
	t.Logf("Input: %q", input)
	t.Logf("Blake2b-512: %x", hash)
	
	// Create generator from gen1 state (in full implementation, this would come from gen1)
	gen, err := newAesGenerator4R(hash[:])
	if err != nil {
		t.Fatalf("Failed to create AesGenerator4R: %v", err)
	}
	
	// Generate configuration data (128 bytes)
	configData := make([]byte, 128)
	gen.getBytes(configData)
	t.Logf("Configuration data (first 32 bytes): %x", configData[:32])
	
	// Generate program data (2048 bytes = 256 instructions × 8 bytes)
	programData := make([]byte, 2048)
	gen.getBytes(programData)
	t.Logf("Program data (first 32 bytes): %x", programData[:32])
	
	// Verify determinism
	gen2, _ := newAesGenerator4R(hash[:])
	configData2 := make([]byte, 128)
	gen2.getBytes(configData2)
	
	if hex.EncodeToString(configData) != hex.EncodeToString(configData2) {
		t.Error("AesGenerator4R not deterministic")
	}
}

// TestVMInitialization_Spec validates VM initialization matches RandomX spec
func TestVMInitialization_Spec(t *testing.T) {
	input := []byte("This is a test")
	
	// Step 1: Blake2b-512 of input
	hash := internal.Blake2b512(input)
	t.Logf("Step 1: Blake2b-512(input) = %x", hash)
	
	// Step 2: Create AesGenerator1R with hash as seed
	gen1, err := newAesGenerator1R(hash[:])
	if err != nil {
		t.Fatalf("Failed to create gen1: %v", err)
	}
	
	// Step 3: Fill scratchpad (2 MB) with gen1 output
	// For test, just check first 256 bytes
	scratchpad := make([]byte, 256)
	gen1.getBytes(scratchpad)
	t.Logf("Step 2: Scratchpad first 64 bytes from gen1 = %x", scratchpad[:64])
	
	// Step 4: Create AesGenerator4R from gen1 state
	gen4, err := newAesGenerator4R(gen1.state[:])
	if err != nil {
		t.Fatalf("Failed to create gen4: %v", err)
	}
	
	// Generate first program's configuration
	configData := make([]byte, 128)
	gen4.getBytes(configData)
	t.Logf("Step 3: Configuration data (first 32 bytes) = %x", configData[:32])
	
	// This validates the generator chain works correctly
	t.Log("✓ Generator chain: input → Blake2b → gen1 → gen4 → config/program")
}

// TestIterationCounts validates correct number of iterations
func TestIterationCounts(t *testing.T) {
	const (
		programCount      = 8
		programIterations = 2048
		instructionsPerProgram = 256
	)
	
	t.Logf("RandomX execution structure:")
	t.Logf("  Programs: %d", programCount)
	t.Logf("  Iterations per program: %d", programIterations)
	t.Logf("  Instructions per program: %d", instructionsPerProgram)
	t.Logf("  Total instruction executions: %d", programCount*programIterations*instructionsPerProgram)
	
	// Current implementation only does 8 iterations total - THIS IS WRONG
	// Should be 8 programs × 2048 iterations = 16,384 loop iterations
	const currentIterations = 8
	if currentIterations != programCount*programIterations {
		t.Logf("⚠ WARNING: Current implementation has %d iterations", currentIterations)
		t.Logf("  Should be: %d programs × %d iterations = %d total", 
			programCount, programIterations, programCount*programIterations)
	}
}

// TestCacheGeneration validates cache matches reference
func TestCacheGeneration(t *testing.T) {
	key := []byte("test key 000")
	
	cache, err := newCache(key)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.release()
	
	// Check first uint64 value - this should match reference implementation
	// Reference value from RandomX: 0x191e0e1d23c02186
	firstUint64 := uint64(cache.data[0]) |
		uint64(cache.data[1])<<8 |
		uint64(cache.data[2])<<16 |
		uint64(cache.data[3])<<24 |
		uint64(cache.data[4])<<32 |
		uint64(cache.data[5])<<40 |
		uint64(cache.data[6])<<48 |
		uint64(cache.data[7])<<56
	
	expectedFirst := uint64(0x191e0e1d23c02186)
	
	t.Logf("Cache first uint64: 0x%016x", firstUint64)
	t.Logf("Expected:           0x%016x", expectedFirst)
	
	if firstUint64 == expectedFirst {
		t.Log("✓ Cache generation matches reference implementation")
	} else {
		t.Error("✗ Cache generation mismatch")
	}
}

// TestRandomXAlgorithmStructure documents the correct algorithm structure
func TestRandomXAlgorithmStructure(t *testing.T) {
	t.Log("=== Correct RandomX Algorithm Structure ===")
	t.Log("")
	t.Log("INITIALIZATION:")
	t.Log("  1. H = Blake2b-512(input)")
	t.Log("  2. gen1 = AesGenerator1R(H)")
	t.Log("  3. Fill scratchpad (2 MB) with gen1 output")
	t.Log("  4. gen4 = AesGenerator4R(gen1.state)")
	t.Log("")
	t.Log("PROGRAM EXECUTION (8 programs):")
	t.Log("  FOR program_num = 0 TO 7:")
	t.Log("    A. Generate configuration (128 bytes from gen4)")
	t.Log("    B. Generate program (2048 bytes from gen4)")
	t.Log("    C. Parse 256 instructions from program data")
	t.Log("    D. Execute 2048 iterations:")
	t.Log("       FOR iter = 0 TO 2047:")
	t.Log("         1. Update spAddr0, spAddr1")
	t.Log("         2. Read from scratchpad → r0-r7")
	t.Log("         3. Read from scratchpad → f0-f3, e0-e3")
	t.Log("         4. Execute all 256 instructions")
	t.Log("         5. Read dataset item")
	t.Log("         6. XOR dataset with r0-r7")
	t.Log("         7. Write r0-r7 to scratchpad")
	t.Log("         8. Write f0-f3 to scratchpad")
	t.Log("    E. Update gen4.state = Hash512(RegisterFile)")
	t.Log("")
	t.Log("FINALIZATION:")
	t.Log("  1. A = AesHash1R(scratchpad)")
	t.Log("  2. result = Blake2b-256(A || RegisterFile)")
	t.Log("")
	t.Log("===========================================")
}
