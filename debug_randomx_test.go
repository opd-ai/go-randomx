package randomx

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/opd-ai/go-randomx/internal"
)

// TestDebugRandomXFlow provides detailed step-by-step execution trace
// for the first test vector to identify where divergence occurs.
func TestDebugRandomXFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping debug test in short mode")
	}

	// Test vector 1 from official RandomX tests
	key := []byte("test key 000")
	input := []byte("This is a test")
	expectedHash := "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f"

	t.Logf("=== RandomX Debug Flow ===")
	t.Logf("Key: %q", key)
	t.Logf("Input: %q", input)
	t.Logf("Expected: %s", expectedHash)
	t.Logf("")

	// Step 1: Create cache
	t.Logf("Step 1: Creating cache from key...")
	config := Config{
		Mode:     LightMode,
		CacheKey: key,
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer hasher.Close()

	// Check cache first bytes
	if hasher.c != nil && len(hasher.c.data) >= 64 {
		t.Logf("Cache first 64 bytes: %s", hex.EncodeToString(hasher.c.data[:64]))
		firstUint64 := binary.LittleEndian.Uint64(hasher.c.data[0:8])
		t.Logf("Cache[0] as uint64: 0x%016x", firstUint64)
		if firstUint64 == 0x191e0e1d23c02186 {
			t.Logf("✓ Cache matches reference implementation")
		} else {
			t.Logf("✗ Cache MISMATCH (expected 0x191e0e1d23c02186)")
		}
	}
	t.Logf("")

	// Step 2: Hash the input
	t.Logf("Step 2: Hashing input...")
	hash := hasher.Hash(input)
	t.Logf("Got:      %s", hex.EncodeToString(hash[:]))
	t.Logf("Expected: %s", expectedHash)
	t.Logf("")

	// Compare
	expectedBytes, _ := hex.DecodeString(expectedHash)
	if string(hash[:]) == string(expectedBytes) {
		t.Logf("✓ PASS: Hash matches!")
	} else {
		t.Logf("✗ FAIL: Hash mismatch")
		t.Logf("")
		t.Logf("Byte-by-byte comparison:")
		for i := 0; i < 32; i++ {
			if hash[i] == expectedBytes[i] {
				t.Logf("  [%02d] %02x == %02x ✓", i, hash[i], expectedBytes[i])
			} else {
				t.Logf("  [%02d] %02x != %02x ✗ (MISMATCH at byte %d)", i, hash[i], expectedBytes[i], i)
			}
		}
	}
}

// TestDebugVMInitialization tests the VM initialization step
func TestDebugVMInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping debug test in short mode")
	}

	input := []byte("This is a test")

	t.Logf("=== VM Initialization Debug ===")
	t.Logf("Input: %q", input)
	t.Logf("")

	// Hash input with Blake2b-512
	hash := internal.Blake2b512(input)
	t.Logf("Blake2b-512 of input:")
	t.Logf("  %s", hex.EncodeToString(hash[:]))
	t.Logf("")

	// Extract registers
	t.Logf("Initial register values from hash:")
	for i := 0; i < 8; i++ {
		reg := binary.LittleEndian.Uint64(hash[i*8 : i*8+8])
		t.Logf("  r%d = 0x%016x", i, reg)
	}
}

// TestDebugProgramGeneration tests program generation
func TestDebugProgramGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping debug test in short mode")
	}

	input := []byte("This is a test")

	t.Logf("=== Program Generation Debug ===")
	t.Logf("Input: %q", input)
	t.Logf("")

	// Generate program
	prog := generateProgram(input)

	t.Logf("First 10 instructions:")
	for i := 0; i < 10 && i < len(prog.instructions); i++ {
		instr := prog.instructions[i]
		t.Logf("  [%03d] opcode=%02x dst=r%d src=r%d mod=%02x imm=0x%08x",
			i, instr.opcode, instr.dst, instr.src, instr.mod, instr.imm)
	}
}

// TestDebugCacheItemRetrieval tests cache item calculation
func TestDebugCacheItemRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping debug test in short mode")
	}

	key := []byte("test key 000")

	t.Logf("=== Cache Item Retrieval Debug ===")
	t.Logf("Key: %q", key)
	t.Logf("")

	config := Config{
		Mode:     LightMode,
		CacheKey: key,
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer hasher.Close()

	if hasher.c == nil {
		t.Fatal("Cache is nil")
	}

	// Get first few cache items
	t.Logf("First 3 cache items:")
	for i := uint32(0); i < 3; i++ {
		item := hasher.c.getItem(i)
		if len(item) >= 64 {
			t.Logf("  Item[%d] first 64 bytes: %s", i, hex.EncodeToString(item[:64]))
		} else {
			t.Logf("  Item[%d] (%d bytes): %s", i, len(item), hex.EncodeToString(item))
		}
	}
}

// TestDebugHashStepByStep provides a complete step-by-step trace
func TestDebugHashStepByStep(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping debug test in short mode")
	}

	key := []byte("test key 000")
	input := []byte("This is a test")

	t.Logf("=== Complete Step-by-Step Hash Trace ===")
	t.Logf("Key: %q", key)
	t.Logf("Input: %q", input)
	t.Logf("")

	// Create hasher
	config := Config{
		Mode:     LightMode,
		CacheKey: key,
	}

	hasher, err := New(config)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer hasher.Close()

	// Create a VM manually to trace execution
	vm := &virtualMachine{
		mem: make([]byte, scratchpadL3Size),
	}
	vm.init(nil, hasher.c)

	t.Logf("VM initialized with cache")
	t.Logf("Scratchpad size: %d bytes", len(vm.mem))
	t.Logf("")

	// Initialize VM from input
	t.Logf("Initializing VM from input...")
	hash := internal.Blake2b512(input)
	for i := 0; i < 8; i++ {
		vm.reg[i] = binary.LittleEndian.Uint64(hash[i*8 : i*8+8])
		t.Logf("  r%d = 0x%016x", i, vm.reg[i])
	}
	t.Logf("")

	// Fill scratchpad
	t.Logf("Filling scratchpad...")
	vm.fillScratchpad()
	t.Logf("Scratchpad first 64 bytes: %s", hex.EncodeToString(vm.mem[:64]))
	t.Logf("")

	// Set memory access parameters
	vm.ma = vm.reg[0]
	vm.mx = vm.reg[1] | 0x01
	t.Logf("Memory access parameters:")
	t.Logf("  ma = 0x%016x", vm.ma)
	t.Logf("  mx = 0x%016x", vm.mx)
	t.Logf("")

	// Execute iterations
	const iterations = 8
	for iter := 0; iter < iterations; iter++ {
		t.Logf("--- Iteration %d ---", iter)

		// Generate program
		prog := generateProgram(input)
		t.Logf("Generated program with %d instructions", len(prog.instructions))

		// Show register state before execution
		t.Logf("Registers before execution:")
		for i := 0; i < 8; i++ {
			t.Logf("  r%d = 0x%016x", i, vm.reg[i])
		}

		// Execute program
		prog.execute(vm)

		t.Logf("Registers after execution:")
		for i := 0; i < 8; i++ {
			t.Logf("  r%d = 0x%016x", i, vm.reg[i])
		}

		// Mix dataset
		vm.mixDataset()

		t.Logf("Registers after dataset mix:")
		for i := 0; i < 8; i++ {
			t.Logf("  r%d = 0x%016x", i, vm.reg[i])
		}
		t.Logf("")
	}

	// Finalize
	t.Logf("Finalizing hash...")
	finalHash := vm.finalize()
	t.Logf("Final hash: %s", hex.EncodeToString(finalHash[:]))
}
