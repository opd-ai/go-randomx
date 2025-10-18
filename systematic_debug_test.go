package randomx

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"testing"

	"github.com/opd-ai/go-randomx/internal"
)

// TestSystematicDebug provides detailed component-by-component validation
// comparing Go implementation against C++ reference implementation behavior.
func TestSystematicDebug(t *testing.T) {
	key := []byte("test key 000")
	input := []byte("This is a test")
	expectedHash := "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f"

	t.Logf("=== SYSTEMATIC RANDOMX DEBUG ===")
	t.Logf("Key: %q", key)
	t.Logf("Input: %q", input)
	t.Logf("Expected: %s", expectedHash)
	t.Logf("")

	// Component 1: Cache generation (Argon2d)
	t.Run("Component1_CacheGeneration", func(t *testing.T) {
		cache, err := newCache(key)
		if err != nil {
			t.Fatalf("Cache creation failed: %v", err)
		}
		defer cache.release()

		// Verify first uint64 matches reference
		firstUint64 := binary.LittleEndian.Uint64(cache.data[0:8])
		expected := uint64(0x191e0e1d23c02186)

		t.Logf("Cache[0]: 0x%016x", firstUint64)
		t.Logf("Expected: 0x%016x", expected)
		
		if firstUint64 == expected {
			t.Logf("✓ Cache generation CORRECT")
		} else {
			t.Errorf("✗ Cache mismatch")
		}
	})

	// Component 2: Initial hash (Blake2b-512)
	t.Run("Component2_InitialHash", func(t *testing.T) {
		hash := internal.Blake2b512(input)
		t.Logf("Blake2b-512: %x", hash)
		
		// Extract register initial values
		t.Logf("Initial register values from hash:")
		for i := 0; i < 8; i++ {
			val := binary.LittleEndian.Uint64(hash[i*8 : (i+1)*8])
			t.Logf("  r%d = 0x%016x", i, val)
		}
		t.Logf("✓ Blake2b-512 completed")
	})

	// Component 3: AES Generator 1R (scratchpad filling)
	t.Run("Component3_AesGenerator1R", func(t *testing.T) {
		hash := internal.Blake2b512(input)
		gen, err := newAesGenerator1R(hash[:])
		if err != nil {
			t.Fatalf("Failed to create AesGenerator1R: %v", err)
		}

		// Get first 64 bytes
		output := make([]byte, 64)
		gen.getBytes(output)
		
		t.Logf("First 64 bytes from AesGenerator1R:")
		t.Logf("  %x", output)
		t.Logf("Gen state after: %x", gen.state[:64])
		t.Logf("✓ AesGenerator1R output generated")
	})

	// Component 4: AES Generator 4R (program generation)
	t.Run("Component4_AesGenerator4R", func(t *testing.T) {
		hash := internal.Blake2b512(input)
		gen1, _ := newAesGenerator1R(hash[:])
		
		gen4, err := newAesGenerator4R(gen1.state[:])
		if err != nil {
			t.Fatalf("Failed to create AesGenerator4R: %v", err)
		}

		// Get configuration data (128 bytes)
		configData := make([]byte, 128)
		gen4.getBytes(configData)
		
		t.Logf("Configuration data (all 128 bytes):")
		t.Logf("  [000-063] %x", configData[:64])
		t.Logf("  [064-127] %x", configData[64:])
		
		// Parse readReg configuration
		readReg0 := configData[0] & 7
		readReg1 := configData[1] & 7
		readReg2 := configData[2] & 7
		readReg3 := configData[3] & 7
		
		t.Logf("Configuration:")
		t.Logf("  readReg0 = %d", readReg0)
		t.Logf("  readReg1 = %d", readReg1)
		t.Logf("  readReg2 = %d", readReg2)
		t.Logf("  readReg3 = %d", readReg3)
		
		// Parse E-masks
		for i := 0; i < 4; i++ {
			eMask := binary.LittleEndian.Uint64(configData[8+i*8 : 8+(i+1)*8])
			t.Logf("  eMask[%d] = 0x%016x", i, eMask)
		}
		
		t.Logf("✓ AesGenerator4R configuration parsed")
	})

	// Component 5: Program parsing
	t.Run("Component5_ProgramParsing", func(t *testing.T) {
		hash := internal.Blake2b512(input)
		gen1, _ := newAesGenerator1R(hash[:])
		gen4, _ := newAesGenerator4R(gen1.state[:])
		
		// Skip configuration (128 bytes)
		configData := make([]byte, 128)
		gen4.getBytes(configData)
		
		// Get program data (2048 bytes for 256 instructions)
		programData := make([]byte, 2048)
		gen4.getBytes(programData)
		
		t.Logf("First 5 instructions:")
		for i := 0; i < 5; i++ {
			offset := i * 8
			opcode := programData[offset+7]
			dst := programData[offset+6] & 7
			src := programData[offset+5] & 7
			mod := programData[offset+4]
			imm := binary.LittleEndian.Uint32(programData[offset : offset+4])
			
			t.Logf("  [%d] opcode=0x%02x dst=r%d src=r%d mod=0x%02x imm=0x%08x",
				i, opcode, dst, src, mod, imm)
		}
		t.Logf("✓ Program parsing completed")
	})

	// Component 6: Full VM execution trace
	t.Run("Component6_VMExecution", func(t *testing.T) {
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
		result := hasher.Hash(input)
		
		t.Logf("Final hash: %x", result[:])
		t.Logf("Expected:   %s", expectedHash)
		
		// Compare byte by byte
		expected, _ := hex.DecodeString(expectedHash)
		t.Logf("")
		t.Logf("Byte-by-byte comparison:")
		firstMismatch := -1
		for i := 0; i < 32; i++ {
			match := "✓"
			if result[i] != expected[i] {
				match = "✗"
				if firstMismatch < 0 {
					firstMismatch = i
				}
			}
			t.Logf("  [%02d] got=0x%02x expected=0x%02x %s", i, result[i], expected[i], match)
		}
		
		if firstMismatch >= 0 {
			t.Logf("")
			t.Logf("✗ First mismatch at byte %d", firstMismatch)
		} else {
			t.Logf("")
			t.Logf("✓ ALL BYTES MATCH!")
		}
	})
}

// TestInstructionExecutionDetailed tests individual instruction execution
func TestInstructionExecutionDetailed(t *testing.T) {
	t.Run("IADD_RS_Detailed", func(t *testing.T) {
		// Test IADD_RS with shift=2
		r0 := uint64(0x1234567890ABCDEF)
		r1 := uint64(0x1111111111111111)
		imm := uint32(0x22222222)
		
		// IADD_RS: r0 = r0 + (r1 << shift) + imm
		shift := uint(2)
		result := r0 + (r1 << shift) + uint64(int32(imm))
		
		t.Logf("IADD_RS execution:")
		t.Logf("  r0 (before) = 0x%016x", r0)
		t.Logf("  r1          = 0x%016x", r1)
		t.Logf("  shift       = %d", shift)
		t.Logf("  imm         = 0x%08x (signed: %d)", imm, int32(imm))
		t.Logf("  r1 << shift = 0x%016x", r1<<shift)
		t.Logf("  result      = 0x%016x", result)
	})
	
	t.Run("IMUL_R_Detailed", func(t *testing.T) {
		r0 := uint64(0x123456789ABCDEF0)
		r1 := uint64(0x2)
		
		result := r0 * r1
		
		t.Logf("IMUL_R execution:")
		t.Logf("  r0 (before) = 0x%016x (%d)", r0, r0)
		t.Logf("  r1          = 0x%016x (%d)", r1, r1)
		t.Logf("  result      = 0x%016x (%d)", result, result)
	})
	
	t.Run("FADD_Detailed", func(t *testing.T) {
		// Test floating-point addition
		a0 := 1.5
		e0 := 2.5
		
		result := a0 + e0
		
		t.Logf("FADD execution:")
		t.Logf("  a0 = %f (bits: 0x%016x)", a0, math.Float64bits(a0))
		t.Logf("  e0 = %f (bits: 0x%016x)", e0, math.Float64bits(e0))
		t.Logf("  result = %f (bits: 0x%016x)", result, math.Float64bits(result))
	})
}

// TestMemoryAddressingDetailed tests memory address calculations
func TestMemoryAddressingDetailed(t *testing.T) {
	t.Run("L1_Mask", func(t *testing.T) {
		// L1 cache: 16 KB, mask should be 0x3FF8 (16384 - 64, aligned to 8)
		addr := uint64(0x123456789ABCDEF0)
		l1Addr := (addr & 0x3FF8)
		
		t.Logf("L1 addressing:")
		t.Logf("  raw addr  = 0x%016x", addr)
		t.Logf("  L1 mask   = 0x%04x", 0x3FF8)
		t.Logf("  L1 addr   = 0x%04x (%d)", l1Addr, l1Addr)
		t.Logf("  in range? = %v (should be < 16384)", l1Addr < 16384)
	})
	
	t.Run("L2_Mask", func(t *testing.T) {
		// L2 cache: 256 KB, mask should be 0x3FFF8
		addr := uint64(0x123456789ABCDEF0)
		l2Addr := (addr & 0x3FFF8)
		
		t.Logf("L2 addressing:")
		t.Logf("  raw addr  = 0x%016x", addr)
		t.Logf("  L2 mask   = 0x%05x", 0x3FFF8)
		t.Logf("  L2 addr   = 0x%05x (%d)", l2Addr, l2Addr)
		t.Logf("  in range? = %v (should be < 262144)", l2Addr < 262144)
	})
	
	t.Run("L3_Mask", func(t *testing.T) {
		// L3 cache: 2 MB, mask should be 0x1FFFF8
		addr := uint64(0x123456789ABCDEF0)
		l3Addr := (addr & 0x1FFFF8)
		
		t.Logf("L3 addressing:")
		t.Logf("  raw addr  = 0x%016x", addr)
		t.Logf("  L3 mask   = 0x%06x", 0x1FFFF8)
		t.Logf("  L3 addr   = 0x%06x (%d)", l3Addr, l3Addr)
		t.Logf("  in range? = %v (should be < 2097152)", l3Addr < 2097152)
	})
}

// TestFloatingPointBehavior tests Go-specific floating-point handling
func TestFloatingPointBehavior(t *testing.T) {
	t.Run("NaN_Handling", func(t *testing.T) {
		nan := math.NaN()
		t.Logf("NaN bits: 0x%016x", math.Float64bits(nan))
		t.Logf("IsNaN: %v", math.IsNaN(nan))
		
		// Test NaN in arithmetic
		result := nan + 1.0
		t.Logf("NaN + 1.0 = %f (IsNaN: %v)", result, math.IsNaN(result))
	})
	
	t.Run("Infinity_Handling", func(t *testing.T) {
		inf := math.Inf(1)
		t.Logf("Inf bits: 0x%016x", math.Float64bits(inf))
		t.Logf("IsInf: %v", math.IsInf(inf, 0))
		
		// Test Inf in arithmetic
		result := inf + 1.0
		t.Logf("Inf + 1.0 = %f (IsInf: %v)", result, math.IsInf(result, 0))
	})
	
	t.Run("Denormal_Handling", func(t *testing.T) {
		// Test denormal (subnormal) numbers
		denormal := math.Float64frombits(0x0000000000000001)
		t.Logf("Denormal: %e (bits: 0x%016x)", denormal, math.Float64bits(denormal))
		
		result := denormal * 2.0
		t.Logf("Denormal * 2 = %e (bits: 0x%016x)", result, math.Float64bits(result))
	})
	
	t.Run("E_Mask_Application", func(t *testing.T) {
		// Test E-mask application (RandomX default: 0x3FFFFFFFFFFFFFFF)
		eMask := uint64(0x3FFFFFFFFFFFFFFF)
		
		// Test with various bit patterns
		testValues := []uint64{
			0xFFFFFFFFFFFFFFFF, // All bits set
			0x7FFFFFFFFFFFFFFF, // Max positive
			0x8000000000000000, // Min value
			0x4000000000000000, // Normal value
		}
		
		for _, val := range testValues {
			masked := val & eMask
			f := math.Float64frombits(masked)
			
			t.Logf("Value: 0x%016x", val)
			t.Logf("  Masked: 0x%016x", masked)
			t.Logf("  Float:  %e (IsNaN: %v, IsInf: %v)",
				f, math.IsNaN(f), math.IsInf(f, 0))
		}
	})
}
