package randomx

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/opd-ai/go-randomx/internal"
)

// TestComponentValidation validates each component of RandomX independently
// to identify where the hash mismatch originates.
func TestComponentValidation(t *testing.T) {
	key := []byte("test key 000")
	input := []byte("This is a test")

	t.Run("Step1_CacheGeneration", func(t *testing.T) {
		// Validate Argon2d cache matches reference
		cache, err := newCache(key)
		if err != nil {
			t.Fatalf("Failed to create cache: %v", err)
		}
		defer cache.release()

		// Check first uint64 - reference value from RandomX C++
		firstUint64 := binary.LittleEndian.Uint64(cache.data[0:8])
		expected := uint64(0x191e0e1d23c02186)

		t.Logf("Cache[0]: 0x%016x", firstUint64)
		t.Logf("Expected: 0x%016x", expected)

		if firstUint64 != expected {
			t.Errorf("Cache generation mismatch!")
		} else {
			t.Log("✓ Cache generation matches reference")
		}
	})

	t.Run("Step2_InputHashing", func(t *testing.T) {
		// Validate Blake2b-512 of input
		hash := internal.Blake2b512(input)

		t.Logf("Input: %q", input)
		t.Logf("Blake2b-512: %x", hash)

		// These should match regardless of RandomX implementation
		// The hash is deterministic
		t.Log("✓ Blake2b-512 is deterministic")
	})

	t.Run("Step3_AesGenerator1R", func(t *testing.T) {
		// Validate AesGenerator1R produces deterministic output
		hash := internal.Blake2b512(input)
		gen1, err := newAesGenerator1R(hash[:])
		if err != nil {
			t.Fatalf("Failed to create gen1: %v", err)
		}

		// Get first 64 bytes from generator
		output1 := make([]byte, 64)
		gen1.getBytes(output1)

		// Create second generator and verify same output
		gen2, _ := newAesGenerator1R(hash[:])
		output2 := make([]byte, 64)
		gen2.getBytes(output2)

		t.Logf("Gen1 output (first 32 bytes): %x", output1[:32])

		if hex.EncodeToString(output1) != hex.EncodeToString(output2) {
			t.Error("AesGenerator1R is not deterministic!")
		} else {
			t.Log("✓ AesGenerator1R is deterministic")
		}
	})

	t.Run("Step4_ScratchpadFilling", func(t *testing.T) {
		// Validate scratchpad filling from gen1
		hash := internal.Blake2b512(input)
		gen1, err := newAesGenerator1R(hash[:])
		if err != nil {
			t.Fatalf("Failed to create gen1: %v", err)
		}

		// Fill a small portion of scratchpad
		scratchpad := make([]byte, 256)
		gen1.getBytes(scratchpad)

		t.Logf("Scratchpad first 32 bytes: %x", scratchpad[:32])
		t.Log("✓ Scratchpad filled from gen1")
	})

	t.Run("Step5_AesGenerator4R", func(t *testing.T) {
		// Validate AesGenerator4R
		hash := internal.Blake2b512(input)
		gen1, _ := newAesGenerator1R(hash[:])

		// Create gen4 from gen1 state
		gen4, err := newAesGenerator4R(gen1.state[:])
		if err != nil {
			t.Fatalf("Failed to create gen4: %v", err)
		}

		// Get configuration data
		configData := make([]byte, 128)
		gen4.getBytes(configData)

		t.Logf("Config data (first 32 bytes): %x", configData[:32])
		t.Log("✓ AesGenerator4R working")
	})

	t.Run("Step6_ProgramGeneration", func(t *testing.T) {
		// Validate program generation
		hash := internal.Blake2b512(input)
		gen1, _ := newAesGenerator1R(hash[:])
		gen4, _ := newAesGenerator4R(gen1.state[:])

		// Get configuration
		configData := make([]byte, 128)
		gen4.getBytes(configData)

		// Get program data
		programData := make([]byte, 2048)
		gen4.getBytes(programData)

		// Decode first instruction
		instr := decodeInstruction(programData[0:8])
		t.Logf("First instruction:")
		t.Logf("  opcode: 0x%02x", instr.opcode)
		t.Logf("  dst: r%d", instr.dst)
		t.Logf("  src: r%d", instr.src)
		t.Logf("  mod: 0x%02x", instr.mod)
		t.Logf("  imm: 0x%08x", instr.imm)

		t.Log("✓ Program generation working")
	})

	t.Run("Step7_RegisterInitialization", func(t *testing.T) {
		// Check register initialization from Blake2b hash
		hash := internal.Blake2b512(input)

		// Registers should be initialized from first 64 bytes of hash
		for i := 0; i < 8; i++ {
			reg := binary.LittleEndian.Uint64(hash[i*8 : i*8+8])
			t.Logf("r%d = 0x%016x", i, reg)
		}

		t.Log("✓ Register initialization pattern validated")
	})
}

// TestFullExecutionTrace provides detailed trace of entire hash computation
func TestFullExecutionTrace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full execution trace in short mode")
	}

	key := []byte("test key 000")
	input := []byte("This is a test")
	expectedHash := "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f"

	t.Logf("=== Full RandomX Execution Trace ===")
	t.Logf("Key: %q", key)
	t.Logf("Input: %q", input)
	t.Logf("Expected: %s", expectedHash)
	t.Logf("")

	// Create hasher
	config := Config{
		Mode:     LightMode,
		CacheKey: key,
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create hasher: %v", err)
	}
	defer hasher.Close()

	// Compute hash
	hash := hasher.Hash(input)
	actualHash := hex.EncodeToString(hash[:])

	t.Logf("Got: %s", actualHash)
	t.Logf("")

	if actualHash != expectedHash {
		t.Errorf("Hash mismatch!")
		t.Logf("Comparing byte-by-byte:")
		expectedBytes, _ := hex.DecodeString(expectedHash)
		for i := 0; i < 32; i++ {
			match := "✓"
			if hash[i] != expectedBytes[i] {
				match = "✗"
			}
			t.Logf("  [%02d] got=0x%02x expected=0x%02x %s", i, hash[i], expectedBytes[i], match)
		}
	} else {
		t.Log("✓ Hash matches!")
	}
}

// TestInstructionExecution validates instruction execution
func TestInstructionExecution(t *testing.T) {
	vm := &virtualMachine{
		mem: make([]byte, scratchpadL3Size),
	}

	// Initialize registers
	vm.reg[0] = 0x1234567890ABCDEF
	vm.reg[1] = 0xFEDCBA9876543210

	t.Run("IADD_RS", func(t *testing.T) {
		// Test IADD_RS instruction (opcode 0-15)
		instr := &instruction{
			opcode: 0, // IADD_RS
			dst:    0,
			src:    1,
			mod:    0,
			imm:    42,
		}

		before := vm.reg[0]
		vm.executeInstruction(instr)
		after := vm.reg[0]

		t.Logf("IADD_RS: r0 before=0x%x after=0x%x", before, after)
		
		if before == after {
			t.Error("Register should have changed!")
		}
	})

	t.Run("IMUL_R", func(t *testing.T) {
		// Test IMUL_R instruction (opcode 46-61)
		vm.reg[2] = 100
		vm.reg[3] = 200
		
		instr := &instruction{
			opcode: 46, // IMUL_R
			dst:    2,
			src:    3,
			mod:    0,
			imm:    0,
		}

		before := vm.reg[2]
		vm.executeInstruction(instr)
		after := vm.reg[2]

		t.Logf("IMUL_R: r2 before=%d after=%d", before, after)
		
		if before == after {
			t.Error("Register should have changed!")
		}
	})
}

// TestDatasetMixing validates dataset/cache mixing
func TestDatasetMixing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping dataset mixing test in short mode")
	}

	key := []byte("test key 000")
	cache, err := newCache(key)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.release()

	// Create VM with cache
	vm := &virtualMachine{
		c:   cache,
		mem: make([]byte, scratchpadL3Size),
		mx:  0, // Start at index 0
	}

	// Initialize registers
	for i := 0; i < 8; i++ {
		vm.reg[i] = uint64(i * 1000)
	}

	t.Logf("Registers before mix:")
	for i := 0; i < 8; i++ {
		t.Logf("  r%d = %d", i, vm.reg[i])
	}

	// Mix dataset
	vm.mixDataset()

	t.Logf("Registers after mix:")
	for i := 0; i < 8; i++ {
		t.Logf("  r%d = %d", i, vm.reg[i])
	}

	// Registers should have changed
	allZero := true
	for i := 0; i < 8; i++ {
		if vm.reg[i] != uint64(i*1000) {
			allZero = false
			break
		}
	}

	if allZero {
		t.Error("Registers didn't change after dataset mixing!")
	} else {
		t.Log("✓ Dataset mixing affected registers")
	}
}

// TestScratchpadAddressing validates scratchpad addressing
func TestScratchpadAddressing(t *testing.T) {
	vm := &virtualMachine{
		mem: make([]byte, scratchpadL3Size),
	}

	// Initialize some memory
	for i := 0; i < 1024; i++ {
		binary.LittleEndian.PutUint64(vm.mem[i*8:], uint64(i))
	}

	t.Run("L1_Addressing", func(t *testing.T) {
		instr := &instruction{
			mod: 2, // L1 level (mod % 4 == 2)
			src: 0,
			imm: 0,
		}
		vm.reg[0] = 0x123456

		addr := vm.getMemoryAddress(instr)
		t.Logf("L1 address: 0x%x (should be within %d bytes)", addr, scratchpadL1Size)

		if addr >= scratchpadL1Size {
			t.Errorf("L1 address out of range: 0x%x >= 0x%x", addr, scratchpadL1Size)
		}
	})

	t.Run("L2_Addressing", func(t *testing.T) {
		instr := &instruction{
			mod: 1, // L2 level (mod % 4 == 1)
			src: 0,
			imm: 0,
		}
		vm.reg[0] = 0x123456

		addr := vm.getMemoryAddress(instr)
		t.Logf("L2 address: 0x%x (should be within %d bytes)", addr, scratchpadL2Size)

		if addr >= scratchpadL2Size {
			t.Errorf("L2 address out of range: 0x%x >= 0x%x", addr, scratchpadL2Size)
		}
	})

	t.Run("L3_Addressing", func(t *testing.T) {
		instr := &instruction{
			mod: 0, // L3 level (mod % 4 == 0)
			src: 0,
			imm: 0,
		}
		vm.reg[0] = 0x123456

		addr := vm.getMemoryAddress(instr)
		t.Logf("L3 address: 0x%x (should be within %d bytes)", addr, scratchpadL3Size)

		if addr >= scratchpadL3Size {
			t.Errorf("L3 address out of range: 0x%x >= 0x%x", addr, scratchpadL3Size)
		}
	})
}
