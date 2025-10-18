package randomx

import (
	"encoding/binary"
	"testing"

	"github.com/opd-ai/go-randomx/internal"
)

// TestVMRegisterInitialization validates that registers are NOT initialized from Blake2b hash
// According to RandomX spec, registers start at zero and are only filled from scratchpad
func TestVMRegisterInitialization(t *testing.T) {
	key := []byte("test key 000")
	input := []byte("This is a test")

	// Create cache
	cache, err := newCache(key)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.release()

	// Create VM
	vm := &virtualMachine{
		c:   cache,
		mem: make([]byte, scratchpadL3Size),
	}

	// Initialize VM - this should NOT set registers from hash
	vm.initialize(input)

	t.Logf("=== Register State After VM Initialization ===")
	t.Logf("Registers should be filled from scratchpad, NOT from Blake2b hash")
	t.Logf("")

	// Check that registers are NOT set to Blake2b hash values
	hash := internal.Blake2b512(input)
	hashMatch := true
	for i := 0; i < 8; i++ {
		hashReg := binary.LittleEndian.Uint64(hash[i*8 : i*8+8])
		actualReg := vm.reg[i]
		
		if hashReg == actualReg {
			t.Logf("r%d = 0x%016x (matches Blake2b hash - WRONG!)", i, actualReg)
			hashMatch = true
		} else {
			t.Logf("r%d = 0x%016x (doesn't match Blake2b hash - correct)", i, actualReg)
			hashMatch = false
		}
	}

	if hashMatch {
		t.Error("✗ BUG: Registers are being initialized from Blake2b hash!")
		t.Error("   RandomX spec: registers start at 0 and are filled from scratchpad")
	} else {
		t.Log("✓ Registers correctly NOT initialized from Blake2b hash")
	}

	t.Logf("")
	t.Logf("Scratchpad first 64 bytes:")
	t.Logf("  %x", vm.mem[:64])
}

// TestFirstIterationTrace traces the first iteration step by step
func TestFirstIterationTrace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping detailed trace in short mode")
	}

	key := []byte("test key 000")
	input := []byte("This is a test")

	// Create cache
	cache, err := newCache(key)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.release()

	// Create VM
	vm := &virtualMachine{
		c:   cache,
		mem: make([]byte, scratchpadL3Size),
	}

	// Initialize VM
	vm.initialize(input)

	// Generate first program
	prog := vm.generateProgram()

	t.Logf("=== First Program Iteration Trace ===")
	t.Logf("")

	// Save initial state
	t.Logf("Initial state:")
	t.Logf("  spAddr0 = 0x%08x", vm.spAddr0)
	t.Logf("  spAddr1 = 0x%08x", vm.spAddr1)
	t.Logf("  mx = 0x%016x", vm.mx)
	t.Logf("  ma = 0x%016x", vm.ma)
	t.Logf("")

	t.Logf("Registers before iteration:")
	for i := 0; i < 8; i++ {
		t.Logf("  r%d = 0x%016x", i, vm.reg[i])
	}
	t.Logf("")

	t.Logf("First 5 instructions:")
	for i := 0; i < 5; i++ {
		instr := &prog.instructions[i]
		instrType := getInstructionType(instr.opcode)
		t.Logf("  [%d] opcode=0x%02x type=%v dst=r%d src=r%d mod=0x%02x imm=0x%08x",
			i, instr.opcode, instrType, instr.dst, instr.src, instr.mod, instr.imm)
	}
	t.Logf("")

	// Execute first iteration
	vm.executeIteration(prog)

	t.Logf("Registers after iteration:")
	for i := 0; i < 8; i++ {
		t.Logf("  r%d = 0x%016x", i, vm.reg[i])
	}
	t.Logf("")

	t.Logf("State after iteration:")
	t.Logf("  spAddr0 = 0x%08x", vm.spAddr0)
	t.Logf("  spAddr1 = 0x%08x", vm.spAddr1)
	t.Logf("  mx = 0x%016x", vm.mx)
	t.Logf("  ma = 0x%016x", vm.ma)
}

// TestFinalizationTrace traces the finalization step
func TestFinalizationTrace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping finalization trace in short mode")
	}

	key := []byte("test key 000")
	input := []byte("This is a test")

	// Create hasher and run full computation
	config := Config{
		Mode:     LightMode,
		CacheKey: key,
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create hasher: %v", err)
	}
	defer hasher.Close()

	// Get VM for inspection (not exposed, so we'll just run it)
	_ = hasher.Hash(input)

	// We can't directly inspect the VM state, but we can trace the finalization
	// by creating our own VM and running the full algorithm

	cache, _ := newCache(key)
	defer cache.release()

	vm := &virtualMachine{
		c:   cache,
		mem: make([]byte, scratchpadL3Size),
	}

	// Run full algorithm
	vm.run(input)

	// Note: finalize() is called at the end of run(), we can't trace it separately
	t.Log("✓ Full algorithm executed")
}

// TestVMConfigurationParsing validates configuration parsing from AesGenerator4R
func TestVMConfigurationParsing(t *testing.T) {
	input := []byte("This is a test")
	hash := internal.Blake2b512(input)
	
	gen1, _ := newAesGenerator1R(hash[:])
	gen4, _ := newAesGenerator4R(gen1.state[:])

	// Get configuration data
	configData := make([]byte, 128)
	gen4.getBytes(configData)

	vm := &virtualMachine{}
	vm.parseConfiguration(configData)

	t.Logf("=== Configuration Parsed from AesGenerator4R ===")
	t.Logf("readReg0 = %d (register for spAddr0 XOR)", vm.config.readReg0)
	t.Logf("readReg1 = %d (register for spAddr1 XOR)", vm.config.readReg1)
	t.Logf("readReg2 = %d (register for mx XOR)", vm.config.readReg2)
	t.Logf("readReg3 = %d (register for mx XOR)", vm.config.readReg3)
	t.Logf("")
	t.Logf("E-register masks:")
	for i := 0; i < 4; i++ {
		t.Logf("  eMask[%d] = 0x%016x", i, vm.config.eMask[i])
	}

	// Verify readReg values are in valid range
	if vm.config.readReg0 >= 8 {
		t.Errorf("readReg0 out of range: %d", vm.config.readReg0)
	}
	if vm.config.readReg1 >= 8 {
		t.Errorf("readReg1 out of range: %d", vm.config.readReg1)
	}
	if vm.config.readReg2 >= 8 {
		t.Errorf("readReg2 out of range: %d", vm.config.readReg2)
	}
	if vm.config.readReg3 >= 8 {
		t.Errorf("readReg3 out of range: %d", vm.config.readReg3)
	}
}

// TestEMaskDefault validates that eMask has proper default values
func TestEMaskDefault(t *testing.T) {
	// According to RandomX spec, eMask default should allow normal FP range
	// Default mask: 0x3FFFFFFFFFFFFFFF (limits exponent to prevent infinity)
	
	input := []byte("This is a test")
	hash := internal.Blake2b512(input)
	
	gen1, _ := newAesGenerator1R(hash[:])
	gen4, _ := newAesGenerator4R(gen1.state[:])

	configData := make([]byte, 128)
	gen4.getBytes(configData)

	vm := &virtualMachine{}
	vm.parseConfiguration(configData)

	t.Logf("=== E-Mask Configuration ===")
	for i := 0; i < 4; i++ {
		mask := vm.config.eMask[i]
		t.Logf("eMask[%d] = 0x%016x", i, mask)
		
		// Check if mask is reasonable (should be non-zero and limit exponent)
		if mask == 0 {
			t.Errorf("eMask[%d] is zero - this will zero out e%d register!", i, i)
		}
		
		// Check if mask has reasonable bit pattern
		// Bit 63 (sign bit) and bits 0-51 (mantissa) should be set
		// Bits 52-62 (exponent) should be partially masked
		signBit := (mask >> 63) & 1
		if signBit == 0 {
			t.Logf("  Warning: eMask[%d] clears sign bit", i)
		}
	}
}
