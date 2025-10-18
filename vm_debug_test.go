package randomx

import (
	"encoding/hex"
	"testing"

	"github.com/opd-ai/go-randomx/internal"
)

// TestVMInitialization_Detailed validates each step of VM initialization
func TestVMInitialization_Detailed(t *testing.T) {
	input := []byte("This is a test")

	// Step 1: Blake2b-512 of input
	hash := internal.Blake2b512(input)
	t.Logf("Step 1 - Blake2b-512(input):")
	t.Logf("  %s", hex.EncodeToString(hash[:]))

	// Step 2: Create AesGenerator1R
	gen1, err := newAesGenerator1R(hash[:])
	if err != nil {
		t.Fatalf("Failed to create gen1: %v", err)
	}

	// Get first 64 bytes from gen1
	var scratchpadStart [64]byte
	gen1.getBytes(scratchpadStart[:])
	t.Logf("Step 2 - First 64 bytes from AesGenerator1R:")
	t.Logf("  %s", hex.EncodeToString(scratchpadStart[:]))

	// Check gen1 state
	t.Logf("Step 3 - gen1.state after first generation:")
	t.Logf("  %s", hex.EncodeToString(gen1.state[:]))

	// Step 4: Create AesGenerator4R from gen1 state
	gen4, err := newAesGenerator4R(gen1.state[:])
	if err != nil {
		t.Fatalf("Failed to create gen4: %v", err)
	}

	// Get first 128 bytes (configuration) from gen4
	configData := make([]byte, 128)
	gen4.getBytes(configData)
	t.Logf("Step 4 - Configuration data (128 bytes) from gen4:")
	t.Logf("  %s...", hex.EncodeToString(configData[:32]))

	// Get program data (first 64 bytes)
	programData := make([]byte, 64)
	gen4.getBytes(programData)
	t.Logf("Step 5 - Program data (first 64 bytes):")
	t.Logf("  %s...", hex.EncodeToString(programData[:32]))
}

// TestProgramGeneration_FirstProgram validates first program generation
func TestProgramGeneration_FirstProgram(t *testing.T) {
	input := []byte("This is a test")

	// Create VM and initialize
	vm := &virtualMachine{
		mem: make([]byte, scratchpadL3Size),
	}
	vm.initialize(input)

	// Generate first program
	prog := vm.generateProgram()

	// Check first few instructions
	t.Log("First 5 instructions of program 0:")
	for i := 0; i < 5; i++ {
		instr := prog.instructions[i]
		t.Logf("  [%d] opcode=%02x dst=r%d src=r%d mod=%02x imm=0x%08x",
			i, instr.opcode, instr.dst, instr.src, instr.mod, instr.imm)
	}

	// Check configuration
	t.Logf("VM Configuration:")
	t.Logf("  readReg0=%d readReg1=%d", vm.config.readReg0, vm.config.readReg1)
	t.Logf("  readReg2=%d readReg3=%d", vm.config.readReg2, vm.config.readReg3)
}

// TestIterationExecution_FirstIteration validates first iteration execution
func TestIterationExecution_FirstIteration(t *testing.T) {
	input := []byte("This is a test")

	// Create VM and initialize
	vm := &virtualMachine{
		mem: make([]byte, scratchpadL3Size),
	}
	vm.initialize(input)

	// Capture scratchpad state before execution
	scratchpadBefore := make([]byte, 64)
	copy(scratchpadBefore, vm.mem[:64])
	t.Logf("Scratchpad first 64 bytes before execution:")
	t.Logf("  %s", hex.EncodeToString(scratchpadBefore[:32]))

	// Generate first program
	prog := vm.generateProgram()

	// Capture register state
	regsBefore := vm.reg
	t.Logf("Registers before first iteration: %v", regsBefore)

	// Execute first iteration
	vm.executeIteration(prog)

	// Check register state after
	t.Logf("Registers after first iteration: %v", vm.reg)

	// Check scratchpad state after
	scratchpadAfter := make([]byte, 64)
	copy(scratchpadAfter, vm.mem[:64])
	t.Logf("Scratchpad first 64 bytes after execution:")
	t.Logf("  %s", hex.EncodeToString(scratchpadAfter[:32]))

	// Check memory addresses
	t.Logf("Memory state:")
	t.Logf("  spAddr0=0x%08x spAddr1=0x%08x", vm.spAddr0, vm.spAddr1)
	t.Logf("  ma=0x%016x mx=0x%016x", vm.ma, vm.mx)
}

// TestFinalization_Components validates finalization components
func TestFinalization_Components(t *testing.T) {
	input := []byte("This is a test")

	// Create and run VM (just initialize for testing)
	vm := &virtualMachine{
		mem: make([]byte, scratchpadL3Size),
	}
	vm.initialize(input)

	// Set some register values for testing
	for i := 0; i < 8; i++ {
		vm.reg[i] = uint64(i * 0x1111111111111111)
	}

	// Test AesHash1R
	hasher, err := newAesHash1R()
	if err != nil {
		t.Fatalf("Failed to create AesHash1R: %v", err)
	}

	scratchpadHash := hasher.hash(vm.mem)
	t.Logf("AesHash1R output (64 bytes):")
	t.Logf("  %s", hex.EncodeToString(scratchpadHash[:32]))
	t.Logf("  %s...", hex.EncodeToString(scratchpadHash[32:64]))

	// Serialize registers
	regData := make([]byte, 256)
	for i := 0; i < 8; i++ {
		regData[i*8] = byte(vm.reg[i])
		regData[i*8+1] = byte(vm.reg[i] >> 8)
		regData[i*8+2] = byte(vm.reg[i] >> 16)
		regData[i*8+3] = byte(vm.reg[i] >> 24)
		regData[i*8+4] = byte(vm.reg[i] >> 32)
		regData[i*8+5] = byte(vm.reg[i] >> 40)
		regData[i*8+6] = byte(vm.reg[i] >> 48)
		regData[i*8+7] = byte(vm.reg[i] >> 56)
	}
	t.Logf("Register data (first 32 bytes):")
	t.Logf("  %s", hex.EncodeToString(regData[:32]))

	// Combine and hash
	combined := make([]byte, 320)
	copy(combined[0:64], scratchpadHash[:])
	copy(combined[64:], regData)

	finalHash := internal.Blake2b256(combined)
	t.Logf("Final hash:")
	t.Logf("  %s", hex.EncodeToString(finalHash[:]))
}

// TestFullExecution_WithDebug runs one full hash with detailed logging
func TestFullExecution_WithDebug(t *testing.T) {
	input := []byte("This is a test")
	key := []byte("test key 000")

	t.Logf("=== Full RandomX Execution Debug ===")
	t.Logf("Key: %q", key)
	t.Logf("Input: %q", input)

	// Create cache
	cache, err := newCache(key)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.release()

	// Create VM
	vm := poolGetVM()
	defer poolPutVM(vm)

	vm.init(nil, cache)

	// Log first cache item
	cacheItem0 := cache.getItem(0)
	t.Logf("Cache item 0 (first 32 bytes): %s", hex.EncodeToString(cacheItem0[:32]))

	// Run hash
	hash := vm.run(input)

	t.Logf("Result: %s", hex.EncodeToString(hash[:]))
	t.Logf("Expected: 639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f")

	// Compare
	expected, _ := hex.DecodeString("639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f")
	if hex.EncodeToString(hash[:]) == hex.EncodeToString(expected) {
		t.Log("✓ MATCH!")
	} else {
		t.Log("✗ MISMATCH")
		// Find first differing byte
		for i := 0; i < 32; i++ {
			if hash[i] != expected[i] {
				t.Logf("  First mismatch at byte %d: got 0x%02x, expected 0x%02x",
					i, hash[i], expected[i])
				break
			}
		}
	}
}
