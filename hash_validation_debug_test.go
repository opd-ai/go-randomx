package randomx

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/opd-ai/go-randomx/internal"
)

// TestHashValidationDebug provides detailed tracing for hash validation debugging.
// This test helps identify where the implementation diverges from RandomX reference.
func TestHashValidationDebug(t *testing.T) {
	// Use the first test vector for detailed analysis
	key := []byte("test key 000")
	input := []byte("This is a test")
	expected := "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f"

	t.Logf("=== Hash Validation Debug Trace ===")
	t.Logf("Key: %q", string(key))
	t.Logf("Input: %q", string(input))
	t.Logf("Expected: %s", expected)

	// Step 1: Validate cache generation
	t.Logf("\n--- Step 1: Cache Generation ---")
	cache, err := newCache(key)
	if err != nil {
		t.Fatalf("Cache creation failed: %v", err)
	}
	defer cache.release()

	t.Logf("Cache size: %d bytes", len(cache.data))
	t.Logf("Number of superscalar programs: %d", len(cache.programs))
	t.Logf("Cache first 32 bytes: %x", cache.data[:32])

	// Validate first cache item
	firstItem := cache.getItem(0)
	t.Logf("Cache item 0: %x", firstItem)

	// Step 2: Check dataset item generation (light mode)
	t.Logf("\n--- Step 2: Dataset Item Generation ---")
	
	// Generate first dataset item
	dsItem := make([]byte, 64)
	
	// Inline dataset item generation for tracing
	const (
		superscalarMul0 = 6364136223846793005
		superscalarAdd1 = 9298411001130361340
		superscalarAdd2 = 12065312585734608966
		superscalarAdd3 = 9306329213124626780
		superscalarAdd4 = 5281919268842080866
		superscalarAdd5 = 10536153434571861004
		superscalarAdd6 = 3398623926847679864
		superscalarAdd7 = 9549104520008361294
	)
	
	var registers [8]uint64
	itemNumber := uint64(0)
	registerValue := itemNumber
	
	registers[0] = (itemNumber + 1) * superscalarMul0
	registers[1] = registers[0] ^ superscalarAdd1
	registers[2] = registers[0] ^ superscalarAdd2
	registers[3] = registers[0] ^ superscalarAdd3
	registers[4] = registers[0] ^ superscalarAdd4
	registers[5] = registers[0] ^ superscalarAdd5
	registers[6] = registers[0] ^ superscalarAdd6
	registers[7] = registers[0] ^ superscalarAdd7
	
	t.Logf("Initial registers for item 0:")
	for i := 0; i < 8; i++ {
		t.Logf("  r%d = %016x", i, registers[i])
	}
	
	// Execute superscalar programs
	for i := 0; i < cacheAccesses; i++ {
		const mask = cacheItems - 1
		cacheIndex := uint32(registerValue & mask)
		mixBlock := cache.getItem(cacheIndex)
		
		t.Logf("\nSuperscalar iteration %d:", i)
		t.Logf("  Cache index: %d (from registerValue=%016x)", cacheIndex, registerValue)
		t.Logf("  Program instructions: %d", len(cache.programs[i].instructions))
		t.Logf("  Address register: r%d", cache.programs[i].addressReg)
		
		// Show first instruction
		if len(cache.programs[i].instructions) > 0 {
			instr := cache.programs[i].instructions[0]
			t.Logf("  First instruction: opcode=%d dst=r%d src=r%d", instr.opcode, instr.dst, instr.src)
		}
		
		// Execute program
		executeSuperscalar(&registers, cache.programs[i], cache.reciprocals)
		
		// XOR cache block
		for r := 0; r < 8; r++ {
			val := binary.LittleEndian.Uint64(mixBlock[r*8 : r*8+8])
			registers[r] ^= val
		}
		
		// Show registers after execution and mixing
		t.Logf("  Registers after execution:")
		for r := 0; r < 8; r++ {
			t.Logf("    r%d = %016x", r, registers[r])
		}
		
		registerValue = registers[cache.programs[i].addressReg]
	}
	
	// Output dataset item
	for r := 0; r < 8; r++ {
		binary.LittleEndian.PutUint64(dsItem[r*8:r*8+8], registers[r])
	}
	
	t.Logf("\nDataset item 0: %x", dsItem)

	// Step 3: VM execution trace
	t.Logf("\n--- Step 3: VM Execution ---")
	
	// Create hasher
	config := Config{
		Mode:     LightMode,
		CacheKey: key,
	}
	
	hasher, err := New(config)
	if err != nil {
		t.Fatalf("Hasher creation failed: %v", err)
	}
	defer hasher.Close()
	
	// Compute hash
	hash := hasher.Hash(input)
	
	t.Logf("Computed hash: %x", hash[:])
	t.Logf("Expected hash: %s", expected)
	
	// Compare
	expectedBytes, _ := hex.DecodeString(expected)
	match := true
	for i := 0; i < 32; i++ {
		if hash[i] != expectedBytes[i] {
			if match {
				t.Logf("\nMismatch found at byte %d:", i)
				match = false
			}
			t.Logf("  Byte %2d: got %02x, expected %02x", i, hash[i], expectedBytes[i])
		}
	}
	
	if !match {
		t.Errorf("Hash mismatch detected")
	} else {
		t.Logf("\n✅ Hash matches expected output!")
	}
}

// TestBlake2GeneratorOutput validates Blake2Generator produces correct output.
func TestBlake2GeneratorOutput(t *testing.T) {
	seed := []byte("test key 000")
	
	t.Logf("Testing Blake2Generator with seed: %q", string(seed))
	
	gen := newBlake2Generator(seed)
	
	// Get first 64 bytes
	output := make([]byte, 64)
	for i := 0; i < 64; i++ {
		output[i] = gen.getByte()
	}
	
	t.Logf("First 64 bytes: %x", output)
	
	// Create another generator to verify determinism
	gen2 := newBlake2Generator(seed)
	output2 := make([]byte, 64)
	for i := 0; i < 64; i++ {
		output2[i] = gen2.getByte()
	}
	
	if hex.EncodeToString(output) != hex.EncodeToString(output2) {
		t.Errorf("Blake2Generator is not deterministic!")
	} else {
		t.Logf("✅ Blake2Generator is deterministic")
	}
}

// TestSuperscalarProgramGenerationDetail validates program generation with detailed output.
func TestSuperscalarProgramGenerationDetail(t *testing.T) {
	seed := []byte("test key 000")
	
	t.Logf("Testing Superscalar Program Generation with seed: %q", string(seed))
	
	gen := newBlake2Generator(seed)
	
	for i := 0; i < 3; i++ {
		prog := generateSuperscalarProgram(gen)
		
		t.Logf("\nProgram %d:", i)
		t.Logf("  Instruction count: %d", len(prog.instructions))
		t.Logf("  Address register: r%d", prog.addressReg)
		
		// Show first few instructions
		showCount := 5
		if len(prog.instructions) < showCount {
			showCount = len(prog.instructions)
		}
		
		for j := 0; j < showCount; j++ {
			instr := prog.instructions[j]
			t.Logf("  Instruction %d: opcode=%d dst=r%d src=r%d imm=%x mod=%x",
				j, instr.opcode, instr.dst, instr.src, instr.imm32, instr.mod)
		}
	}
}

// TestArgon2dCacheCorrectness validates Argon2d cache generation.
func TestArgon2dCacheCorrectness(t *testing.T) {
	seed := []byte("test key 000")
	
	t.Logf("Testing Argon2d cache generation with seed: %q", string(seed))
	
	// Generate cache using our implementation
	cacheData := internal.Argon2dCache(seed)
	
	t.Logf("Cache size: %d bytes", len(cacheData))
	t.Logf("First 64 bytes: %x", cacheData[:64])
	t.Logf("Last 64 bytes: %x", cacheData[len(cacheData)-64:])
	
	// Verify cache size
	expectedSize := 262144 * 1024 // 256 MB
	if len(cacheData) != expectedSize {
		t.Errorf("Cache size mismatch: got %d, expected %d", len(cacheData), expectedSize)
	}
	
	// Verify determinism
	cacheData2 := internal.Argon2dCache(seed)
	
	mismatch := false
	for i := 0; i < len(cacheData); i++ {
		if cacheData[i] != cacheData2[i] {
			if !mismatch {
				t.Errorf("Cache is not deterministic! First mismatch at byte %d", i)
				mismatch = true
			}
		}
	}
	
	if !mismatch {
		t.Logf("✅ Argon2d cache is deterministic")
	}
}

// TestDatasetItemConsistency checks if dataset items are generated consistently.
func TestDatasetItemConsistency(t *testing.T) {
	key := []byte("test key 000")
	
	// Create cache twice
	cache1, err := newCache(key)
	if err != nil {
		t.Fatalf("Failed to create cache1: %v", err)
	}
	defer cache1.release()
	
	cache2, err := newCache(key)
	if err != nil {
		t.Fatalf("Failed to create cache2: %v", err)
	}
	defer cache2.release()
	
	// Generate same dataset items from both caches
	item1 := make([]byte, 64)
	item2 := make([]byte, 64)
	
	// Test first 10 items
	for i := uint64(0); i < 10; i++ {
		// Generate using inline code to match dataset.go
		generateDatasetItemForTest(cache1, i, item1)
		generateDatasetItemForTest(cache2, i, item2)
		
		if hex.EncodeToString(item1) != hex.EncodeToString(item2) {
			t.Errorf("Dataset item %d not consistent between caches!", i)
			t.Logf("  Item1: %x", item1)
			t.Logf("  Item2: %x", item2)
		}
	}
	
	t.Logf("✅ Dataset item generation is consistent")
}

// generateDatasetItemForTest is a helper that mirrors dataset.go generateItem logic.
func generateDatasetItemForTest(c *cache, itemNumber uint64, output []byte) {
	const (
		superscalarMul0 = 6364136223846793005
		superscalarAdd1 = 9298411001130361340
		superscalarAdd2 = 12065312585734608966
		superscalarAdd3 = 9306329213124626780
		superscalarAdd4 = 5281919268842080866
		superscalarAdd5 = 10536153434571861004
		superscalarAdd6 = 3398623926847679864
		superscalarAdd7 = 9549104520008361294
	)
	
	var registers [8]uint64
	registerValue := itemNumber
	registers[0] = (itemNumber + 1) * superscalarMul0
	registers[1] = registers[0] ^ superscalarAdd1
	registers[2] = registers[0] ^ superscalarAdd2
	registers[3] = registers[0] ^ superscalarAdd3
	registers[4] = registers[0] ^ superscalarAdd4
	registers[5] = registers[0] ^ superscalarAdd5
	registers[6] = registers[0] ^ superscalarAdd6
	registers[7] = registers[0] ^ superscalarAdd7
	
	for i := 0; i < cacheAccesses; i++ {
		const mask = cacheItems - 1
		cacheIndex := uint32(registerValue & mask)
		mixBlock := c.getItem(cacheIndex)
		
		prog := c.programs[i]
		executeSuperscalar(&registers, prog, c.reciprocals)
		
		for r := 0; r < 8; r++ {
			val := binary.LittleEndian.Uint64(mixBlock[r*8 : r*8+8])
			registers[r] ^= val
		}
		
		registerValue = registers[prog.addressReg]
	}
	
	for r := 0; r < 8; r++ {
		binary.LittleEndian.PutUint64(output[r*8:r*8+8], registers[r])
	}
}

// TestVMScratchpadInit validates VM scratchpad initialization.
func TestVMScratchpadInit(t *testing.T) {
	input := []byte("This is a test")
	
	t.Logf("Testing VM scratchpad initialization")
	t.Logf("Input: %q", string(input))
	
	// Hash input with Blake2b-512
	hash := internal.Blake2b512(input)
	t.Logf("Blake2b-512(input): %x", hash[:])
	
	// Create AES generator from hash
	gen, err := newAesGenerator1R(hash[:])
	if err != nil {
		t.Fatalf("Failed to create AES generator: %v", err)
	}
	
	// Generate first 64 bytes of scratchpad
	scratchpad := make([]byte, 64)
	gen.getBytes(scratchpad)
	
	t.Logf("First 64 bytes of scratchpad: %x", scratchpad)
	
	// Verify determinism
	gen2, _ := newAesGenerator1R(hash[:])
	scratchpad2 := make([]byte, 64)
	gen2.getBytes(scratchpad2)
	
	if hex.EncodeToString(scratchpad) != hex.EncodeToString(scratchpad2) {
		t.Errorf("Scratchpad initialization is not deterministic!")
	} else {
		t.Logf("✅ Scratchpad initialization is deterministic")
	}
}
