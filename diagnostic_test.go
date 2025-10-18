package randomx

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/opd-ai/go-randomx/internal"
)

// TestDiagnosticVMSteps performs step-by-step validation of VM execution
func TestDiagnosticVMSteps(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping diagnostic test in short mode")
	}

	// Test vector: basic_test_1
	key := []byte("test key 000")
	input := []byte("This is a test")
	expectedHash := "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f"

	t.Logf("=== DIAGNOSTIC VALIDATION ===")
	t.Logf("Key: %q", key)
	t.Logf("Input: %q", input)
	t.Logf("Expected: %s", expectedHash)
	t.Logf("")

	// Step 1: Cache generation
	t.Logf("Step 1: Cache Generation")
	cache, err := newCache(key)
	if err != nil {
		t.Fatalf("Cache creation failed: %v", err)
	}
	defer cache.release()

	// Verify cache matches reference
	cacheData := cache.data
	firstQword := binary.LittleEndian.Uint64(cacheData[0:8])
	t.Logf("  Cache[0] = 0x%016x", firstQword)
	if firstQword != 0x191e0e1d23c02186 {
		t.Errorf("  ❌ Cache mismatch! Expected 0x191e0e1d23c02186")
	} else {
		t.Logf("  ✅ Cache matches reference")
	}
	t.Logf("")

	// Step 2: Initial hash
	t.Logf("Step 2: Blake2b-512(input)")
	hash := internal.Blake2b512(input)
	t.Logf("  Hash: %x", hash)
	t.Logf("")

	// Step 3: AesGenerator1R
	t.Logf("Step 3: AesGenerator1R initialization")
	gen1, err := newAesGenerator1R(hash[:])
	if err != nil {
		t.Fatalf("AesGenerator1R creation failed: %v", err)
	}

	// Get first few bytes to verify
	testBytes := make([]byte, 64)
	gen1Copy, _ := newAesGenerator1R(hash[:])
	gen1Copy.getBytes(testBytes)
	t.Logf("  First 64 bytes: %x", testBytes)
	t.Logf("")

	// Step 4: Scratchpad filling
	t.Logf("Step 4: Fill scratchpad (2 MB)")
	scratchpad := make([]byte, scratchpadL3Size)
	gen1.getBytes(scratchpad)
	t.Logf("  Scratchpad[0:8] = 0x%016x", binary.LittleEndian.Uint64(scratchpad[0:8]))
	t.Logf("  Scratchpad[8:16] = 0x%016x", binary.LittleEndian.Uint64(scratchpad[8:16]))
	t.Logf("")

	// Step 5: AesGenerator4R
	t.Logf("Step 5: AesGenerator4R from gen1.state")
	gen4, err := newAesGenerator4R(gen1.state[:])
	if err != nil {
		t.Fatalf("AesGenerator4R creation failed: %v", err)
	}
	t.Logf("  gen4 initialized with state from gen1")
	t.Logf("")

	// Step 6: First program generation
	t.Logf("Step 6: Generate first program")
	configData := make([]byte, 128)
	gen4.getBytes(configData)
	t.Logf("  Configuration (first 32 bytes): %x", configData[:32])

	programData := make([]byte, 2048)
	gen4.getBytes(programData)
	t.Logf("  Program data (first 32 bytes): %x", programData[:32])

	// Decode first instruction
	instr0 := decodeInstruction(programData[0:8])
	t.Logf("  First instruction: opcode=%d dst=r%d src=r%d imm=0x%08x",
		instr0.opcode, instr0.dst, instr0.src, instr0.imm)
	t.Logf("")

	// Step 7: Execute full hash
	t.Logf("Step 7: Execute full RandomX algorithm")
	hasher, err := New(Config{
		Mode:     LightMode,
		CacheKey: key,
	})
	if err != nil {
		t.Fatalf("Hasher creation failed: %v", err)
	}
	defer hasher.Close()

	result := hasher.Hash(input)
	t.Logf("  Result: %x", result[:])
	t.Logf("  Expected: %s", expectedHash)
	t.Logf("")

	// Compare
	expectedBytes, _ := hex.DecodeString(expectedHash)
	if hex.EncodeToString(result[:]) == expectedHash {
		t.Logf("✅ PASS: Hash matches!")
	} else {
		t.Logf("❌ FAIL: Hash mismatch")
		t.Logf("")
		t.Logf("Byte-by-byte comparison (first 16 bytes):")
		for i := 0; i < 16; i++ {
			match := "✓"
			if result[i] != expectedBytes[i] {
				match = "✗"
			}
			t.Logf("  [%02d] %02x vs %02x %s", i, result[i], expectedBytes[i], match)
		}
	}
}

// TestInstructionDecoding validates instruction decoding
func TestInstructionDecoding(t *testing.T) {
	// Test with known bytes
	data := []byte{0x15, 0x75, 0xED, 0x67, 0x1B, 0x73, 0xAC, 0x21}
	instr := decodeInstruction(data)

	t.Logf("Raw bytes: %x", data)
	t.Logf("Opcode: 0x%02x (%d)", instr.opcode, instr.opcode)
	t.Logf("Dst: r%d (from byte 0x%02x)", instr.dst, data[1])
	t.Logf("Src: r%d (from byte 0x%02x)", instr.src, data[2])
	t.Logf("Mod: 0x%02x", instr.mod)
	t.Logf("Imm: 0x%08x", instr.imm)

	// Verify decoding
	raw := binary.LittleEndian.Uint64(data)
	expectedDst := uint8((raw >> 8) & 0x07)
	expectedSrc := uint8((raw >> 16) & 0x07)

	if instr.dst != expectedDst {
		t.Errorf("Dst mismatch: got r%d, expected r%d", instr.dst, expectedDst)
	}
	if instr.src != expectedSrc {
		t.Errorf("Src mismatch: got r%d, expected r%d", instr.src, expectedSrc)
	}
}

// TestRegisterInitialization validates that registers are properly initialized
func TestRegisterInitialization(t *testing.T) {
	vm := poolGetVM()
	defer poolPutVM(vm)

	input := []byte("This is a test")

	// Initialize VM
	vm.initialize(input)

	// Check that scratchpad is filled
	if len(vm.mem) != scratchpadL3Size {
		t.Errorf("Scratchpad size: got %d, expected %d", len(vm.mem), scratchpadL3Size)
	}

	// Check that gen4 is initialized
	if vm.gen4 == nil {
		t.Error("gen4 should be initialized")
	}

	// Verify scratchpad has non-zero data
	nonZero := false
	for i := 0; i < 64; i++ {
		if vm.mem[i] != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Error("Scratchpad should contain non-zero data")
	}

	t.Logf("✅ VM initialization looks correct")
}

// TestConfigurationParsing validates configuration data parsing
func TestConfigurationParsing(t *testing.T) {
	vm := &virtualMachine{}

	// Create test configuration data
	configData := make([]byte, 128)
	// Set readReg values (individual bytes, masked with 7)
	configData[0] = 3  // readReg0 = 3 & 7 = 3
	configData[1] = 10 // readReg1 = 10 & 7 = 2
	configData[2] = 15 // readReg2 = 15 & 7 = 7
	configData[3] = 22 // readReg3 = 22 & 7 = 6

	// Set E masks (consecutive uint64 values starting at byte 8)
	// Set with high bit (bit 62) set to avoid default mask substitution
	for i := 0; i < 4; i++ {
		offset := 8 + i*8
		val := uint64(i*1000) | (1 << 62) // Set bit 62 to use parsed value
		binary.LittleEndian.PutUint64(configData[offset:offset+8], val)
	}

	vm.parseConfiguration(configData)

	// Verify
	if vm.config.readReg0 != 3 {
		t.Errorf("readReg0: got %d, expected 3", vm.config.readReg0)
	}
	if vm.config.readReg1 != 2 {
		t.Errorf("readReg1: got %d, expected 2", vm.config.readReg1)
	}
	if vm.config.readReg2 != 7 {
		t.Errorf("readReg2: got %d, expected 7", vm.config.readReg2)
	}
	if vm.config.readReg3 != 6 {
		t.Errorf("readReg3: got %d, expected 6", vm.config.readReg3)
	}

	for i := 0; i < 4; i++ {
		expected := uint64(i*1000) | (1 << 62) // Expected has bit 62 set
		if vm.config.eMask[i] != expected {
			t.Errorf("eMask[%d]: got %d, expected %d", i, vm.config.eMask[i], expected)
		}
	}

	t.Logf("✅ Configuration parsing correct")
}
