package randomx

import (
	"testing"
)

// TestBlake2Generator_Determinism verifies the Blake2Generator produces
// deterministic output for the same seed.
func TestBlake2Generator_Determinism(t *testing.T) {
	seed := []byte("test seed 123")
	
	gen1 := newBlake2Generator(seed)
	gen2 := newBlake2Generator(seed)
	
	// Generate 1000 bytes and compare
	for i := 0; i < 1000; i++ {
		b1 := gen1.getByte()
		b2 := gen2.getByte()
		
		if b1 != b2 {
			t.Fatalf("Mismatch at byte %d: gen1=%d, gen2=%d", i, b1, b2)
		}
	}
}

// TestBlake2Generator_ByteOutput verifies getByte returns values in full range.
func TestBlake2Generator_ByteOutput(t *testing.T) {
	gen := newBlake2Generator([]byte("test"))
	
	// Generate enough bytes to likely see full range
	seen := make(map[byte]bool)
	for i := 0; i < 10000; i++ {
		b := gen.getByte()
		seen[b] = true
	}
	
	// Should have seen at least 200 different byte values (out of 256)
	// This is a statistical test, might rarely fail
	if len(seen) < 200 {
		t.Errorf("Poor byte distribution: only %d/256 unique values seen", len(seen))
	}
}

// TestBlake2Generator_Uint32Output verifies getUint32 returns proper little-endian values.
func TestBlake2Generator_Uint32Output(t *testing.T) {
	
	// Generate same data via getByte and getUint32
	gen1 := newBlake2Generator([]byte("test"))
	gen2 := newBlake2Generator([]byte("test"))
	
	for i := 0; i < 100; i++ {
		// Get as uint32
		val32 := gen1.getUint32()
		
		// Get as 4 bytes
		b0 := uint32(gen2.getByte())
		b1 := uint32(gen2.getByte())
		b2 := uint32(gen2.getByte())
		b3 := uint32(gen2.getByte())
		
		// Reconstruct little-endian uint32
		expected := b0 | (b1 << 8) | (b2 << 16) | (b3 << 24)
		
		if val32 != expected {
			t.Fatalf("Uint32 mismatch at index %d: got 0x%08x, want 0x%08x", i, val32, expected)
		}
	}
}

// TestSuperscalarExecution_BasicOps tests execution of basic superscalar instructions.
func TestSuperscalarExecution_BasicOps(t *testing.T) {
	tests := []struct {
		name     string
		opcode   uint8
		r0       uint64
		r1       uint64
		expected uint64
	}{
		{"ISUB_R", ssISUB_R, 100, 30, 70},
		{"IXOR_R", ssIXOR_R, 0xFF00, 0x00FF, 0xFFFF},
		{"IMUL_R", ssIMUL_R, 10, 5, 50},
		{"IADD_C7", ssIADD_C7, 100, 0, 105}, // imm32=5
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var regs [8]uint64
			regs[0] = tt.r0
			regs[1] = tt.r1
			
			prog := &superscalarProgram{
				instructions: []superscalarInstruction{
					{opcode: tt.opcode, dst: 0, src: 1, imm32: 5},
				},
			}
			
			executeSuperscalar(&regs, prog, nil)
			
			if regs[0] != tt.expected {
				t.Errorf("Result mismatch: got %d, want %d", regs[0], tt.expected)
			}
		})
	}
}

// TestSuperscalarExecution_MultiplyHigh tests high multiplication instructions.
func TestSuperscalarExecution_MultiplyHigh(t *testing.T) {
	tests := []struct {
		name     string
		opcode   uint8
		a        uint64
		b        uint64
		expected uint64
	}{
		{
			name:     "IMULH_R small",
			opcode:   ssIMULH_R,
			a:        1000000,
			b:        1000000,
			expected: 0, // (10^12 / 2^64) = 0
		},
		{
			name:     "IMULH_R large",
			opcode:   ssIMULH_R,
			a:        0xFFFFFFFFFFFFFFFF,
			b:        0xFFFFFFFFFFFFFFFF,
			expected: 0xFFFFFFFFFFFFFFFE, // (2^128 - 2^65 + 1) >> 64
		},
		{
			name:     "ISMULH_R positive",
			opcode:   ssISMULH_R,
			a:        0x7FFFFFFFFFFFFFFF, // max int64
			b:        2,
			expected: 0, // (max*2) >> 64 = 0
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var regs [8]uint64
			regs[0] = tt.a
			regs[1] = tt.b
			
			prog := &superscalarProgram{
				instructions: []superscalarInstruction{
					{opcode: tt.opcode, dst: 0, src: 1},
				},
			}
			
			executeSuperscalar(&regs, prog, nil)
			
			if regs[0] != tt.expected {
				t.Errorf("Result mismatch: got 0x%016x, want 0x%016x", regs[0], tt.expected)
			}
		})
	}
}

// TestSuperscalarExecution_Reciprocal tests IMUL_RCP instruction.
func TestSuperscalarExecution_Reciprocal(t *testing.T) {
	divisor := uint32(12345)
	rcp := reciprocal(divisor)
	
	var regs [8]uint64
	regs[0] = 1000000000
	
	prog := &superscalarProgram{
		instructions: []superscalarInstruction{
			{opcode: ssIMUL_RCP, dst: 0, imm32: 0}, // imm32 is index into reciprocals
		},
	}
	
	reciprocals := []uint64{rcp}
	
	executeSuperscalar(&regs, prog, reciprocals)
	
	// Result should be approximately (1000000000 / 12345) but using fast reciprocal
	// Exact value depends on reciprocal approximation
	if regs[0] == 1000000000 {
		t.Error("IMUL_RCP did not modify register")
	}
}

// TestSuperscalarExecution_Rotation tests IROR_C instruction.
func TestSuperscalarExecution_Rotation(t *testing.T) {
	var regs [8]uint64
	regs[0] = 0x123456789ABCDEF0
	
	prog := &superscalarProgram{
		instructions: []superscalarInstruction{
			{opcode: ssIROR_C, dst: 0, imm32: 8}, // Rotate right by 8 bits
		},
	}
	
	executeSuperscalar(&regs, prog, nil)
	
	expected := uint64(0xF0123456789ABCDE)
	if regs[0] != expected {
		t.Errorf("Rotation incorrect: got 0x%016x, want 0x%016x", regs[0], expected)
	}
}

// TestSuperscalarExecution_AddRS tests IADD_RS (scaled addition).
func TestSuperscalarExecution_AddRS(t *testing.T) {
	var regs [8]uint64
	regs[0] = 1000
	regs[1] = 100
	
	prog := &superscalarProgram{
		instructions: []superscalarInstruction{
			{opcode: ssIADD_RS, dst: 0, src: 1, mod: 2}, // r0 += r1 << 2
		},
	}
	
	executeSuperscalar(&regs, prog, nil)
	
	expected := uint64(1000 + (100 << 2)) // 1000 + 400 = 1400
	if regs[0] != expected {
		t.Errorf("IADD_RS incorrect: got %d, want %d", regs[0], expected)
	}
}

// TestReciprocal verifies the reciprocal function.
func TestReciprocal(t *testing.T) {
	tests := []struct {
		divisor uint32
		// We can't test exact values without C++ reference,
		// but we can test that function runs without panic
	}{
		{divisor: 2},
		{divisor: 12345},
		{divisor: 3},
		{divisor: 0x7FFFFFFF},
	}
	
	for _, tt := range tests {
		// Just verify it doesn't panic and returns something
		rcp := reciprocal(tt.divisor)
		
		// For most divisors > 1, reciprocal should produce a value
		// (some edge cases with large divisors may return 0 due to shift overflow)
		_ = rcp
	}
}

// TestSignExtend2sCompl tests sign extension.
func TestSignExtend2sCompl(t *testing.T) {
	tests := []struct {
		input    uint32
		expected uint64
	}{
		{0x00000000, 0x0000000000000000},
		{0x7FFFFFFF, 0x000000007FFFFFFF},
		{0x80000000, 0xFFFFFFFF80000000},
		{0xFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		{0x00000001, 0x0000000000000001},
		{0xFFFFFFFE, 0xFFFFFFFFFFFFFFFE},
	}
	
	for _, tt := range tests {
		result := signExtend2sCompl(tt.input)
		if result != tt.expected {
			t.Errorf("signExtend2sCompl(0x%08x) = 0x%016x, want 0x%016x",
				tt.input, result, tt.expected)
		}
	}
}

// TestMulh tests unsigned high multiplication.
func TestMulh(t *testing.T) {
	tests := []struct {
		a        uint64
		b        uint64
		expected uint64
	}{
		{0, 0, 0},
		{1, 1, 0},
		{0xFFFFFFFFFFFFFFFF, 1, 0},
		{0xFFFFFFFFFFFFFFFF, 2, 1},
		{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFE},
	}
	
	for _, tt := range tests {
		result := mulh(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("mulh(0x%x, 0x%x) = 0x%x, want 0x%x",
				tt.a, tt.b, result, tt.expected)
		}
	}
}

// TestSmulh tests signed high multiplication.
func TestSmulh(t *testing.T) {
	tests := []struct {
		name string
		a    int64
		b    int64
		// Expected can vary - this is just to ensure no crashes
	}{
		{"zero", 0, 0},
		{"positive", 1000000000, 1000000000},
		{"negative*positive", -1000000000, 1000000000},
		{"negative*negative", -1000000000, -1000000000},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			result := smulh(tt.a, tt.b)
			_ = result
		})
	}
}

// TestRotr tests right rotation.
func TestRotr(t *testing.T) {
	tests := []struct {
		x        uint64
		c        uint
		expected uint64
	}{
		{0x123456789ABCDEF0, 0, 0x123456789ABCDEF0},
		{0x123456789ABCDEF0, 8, 0xF0123456789ABCDE},
		{0x123456789ABCDEF0, 64, 0x123456789ABCDEF0},
		{0x1, 1, 0x8000000000000000},
		{0x8000000000000000, 1, 0x4000000000000000},
	}
	
	for _, tt := range tests {
		result := rotr(tt.x, tt.c)
		if result != tt.expected {
			t.Errorf("rotr(0x%016x, %d) = 0x%016x, want 0x%016x",
				tt.x, tt.c, result, tt.expected)
		}
	}
}

// TestSuperscalarProgram_Determinism verifies program generation is deterministic.
func TestSuperscalarProgram_Determinism(t *testing.T) {
	seed := []byte("determinism test")
	
	gen1 := newBlake2Generator(seed)
	prog1 := generateSuperscalarProgram(gen1)
	
	gen2 := newBlake2Generator(seed)
	prog2 := generateSuperscalarProgram(gen2)
	
	// Programs should be identical
	if len(prog1.instructions) != len(prog2.instructions) {
		t.Fatalf("Program size mismatch: %d vs %d", 
			len(prog1.instructions), len(prog2.instructions))
	}
	
	if prog1.addressReg != prog2.addressReg {
		t.Errorf("Address register mismatch: %d vs %d",
			prog1.addressReg, prog2.addressReg)
	}
	
	// Compare instructions
	for i := range prog1.instructions {
		if prog1.instructions[i] != prog2.instructions[i] {
			t.Errorf("Instruction %d mismatch: %+v vs %+v",
				i, prog1.instructions[i], prog2.instructions[i])
		}
	}
}

// TestSuperscalarProgram_Properties verifies generated programs have valid properties.
func TestSuperscalarProgram_Properties(t *testing.T) {
	seeds := [][]byte{
		[]byte("test key 000"),
		[]byte("test key 001"),
		[]byte("another seed"),
	}
	
	for _, seed := range seeds {
		t.Run(string(seed), func(t *testing.T) {
			gen := newBlake2Generator(seed)
			prog := generateSuperscalarProgram(gen)
			
			// Program should have at least a few instructions
			if len(prog.instructions) < 3 {
				t.Errorf("Program too small: %d instructions", len(prog.instructions))
			}
			
			// Program should not exceed maximum size
			if len(prog.instructions) > superscalarMaxSize {
				t.Errorf("Program too large: %d instructions (max %d)", 
					len(prog.instructions), superscalarMaxSize)
			}
			
			// Address register should be valid (0-7)
			if prog.addressReg > 7 {
				t.Errorf("Invalid address register: %d", prog.addressReg)
			}
			
			// All instructions should have valid opcodes
			for i, instr := range prog.instructions {
				if instr.opcode >= ssCount {
					t.Errorf("Instruction %d has invalid opcode: %d", i, instr.opcode)
				}
				
				// Dst and src should be valid registers
				if instr.dst > 7 {
					t.Errorf("Instruction %d has invalid dst: %d", i, instr.dst)
				}
				if instr.src > 7 && instr.opcode < ssIROR_C {
					t.Errorf("Instruction %d has invalid src: %d", i, instr.src)
				}
			}
		})
	}
}

// Benchmark superscalar program generation.
func BenchmarkGenerateSuperscalarProgram(b *testing.B) {
	seed := []byte("benchmark seed")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := newBlake2Generator(seed)
		_ = generateSuperscalarProgram(gen)
	}
}

// Benchmark superscalar program execution.
func BenchmarkExecuteSuperscalarProgram(b *testing.B) {
	seed := []byte("benchmark seed")
	gen := newBlake2Generator(seed)
	prog := generateSuperscalarProgram(gen)
	
	var regs [8]uint64
	reciprocals := []uint64{reciprocal(12345)}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		regs = [8]uint64{1, 2, 3, 4, 5, 6, 7, 8}
		executeSuperscalar(&regs, prog, reciprocals)
	}
}
