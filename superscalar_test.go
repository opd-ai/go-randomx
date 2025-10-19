package randomx

import (
	"encoding/hex"
	"testing"
)

// TestSuperscalarProgramGeneration tests that we generate programs correctly
func TestSuperscalarProgramGeneration(t *testing.T) {
	// Use the same seed as test vector
	seed := []byte("test key 000")
	gen := newBlake2Generator(seed)
	
	// Generate first program
	prog := generateSuperscalarProgram(gen)
	
	t.Logf("Generated program with %d instructions", len(prog.instructions))
	t.Logf("Address register: r%d", prog.addressReg)
	
	// Print first few instructions
	for i := 0; i < min(10, len(prog.instructions)); i++ {
		instr := prog.instructions[i]
		t.Logf("  instr[%d]: opcode=%d dst=r%d src=r%d mod=%d imm32=0x%08x",
			i, instr.opcode, instr.dst, instr.src, instr.mod, instr.imm32)
	}
}

// TestBlake2Generator tests the Blake2 generator
func TestBlake2Generator(t *testing.T) {
	seed := []byte("test key 000")
	gen := newBlake2Generator(seed)
	
	// Get first few bytes
	bytes := make([]byte, 64)
	for i := range bytes {
		bytes[i] = gen.getByte()
	}
	
	t.Logf("First 64 bytes from generator: %s", hex.EncodeToString(bytes))
	
	// Test uint32 generation
	gen2 := newBlake2Generator(seed)
	v1 := gen2.getUint32()
	v2 := gen2.getUint32()
	
	t.Logf("First uint32: 0x%08x", v1)
	t.Logf("Second uint32: 0x%08x", v2)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
