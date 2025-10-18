package randomx

// Superscalar instruction types
// These correspond to the SuperscalarInstructionType enum in the C++ reference
const (
	ssISUB_R = iota // r[dst] -= r[src]
	ssIXOR_R        // r[dst] ^= r[src]
	ssIADD_RS       // r[dst] += r[src] << shift
	ssIMUL_R        // r[dst] *= r[src]
	ssIROR_C        // r[dst] = rotate_right(r[dst], imm)
	ssIADD_C7       // r[dst] += imm (7-byte immediate)
	ssIXOR_C7       // r[dst] ^= imm (7-byte immediate)
	ssIADD_C8       // r[dst] += imm (8-byte immediate)
	ssIXOR_C8       // r[dst] ^= imm (8-byte immediate)
	ssIADD_C9       // r[dst] += imm (9-byte immediate)
	ssIXOR_C9       // r[dst] ^= imm (9-byte immediate)
	ssIMULH_R       // r[dst] = (r[dst] * r[src]) >> 64 (unsigned high multiplication)
	ssISMULH_R      // r[dst] = (int64(r[dst]) * int64(r[src])) >> 64 (signed high multiplication)
	ssIMUL_RCP      // r[dst] *= reciprocal(imm)
	
	ssCount = 14
)

// superscalarInstruction represents a single instruction in a superscalar program.
type superscalarInstruction struct {
	opcode uint8  // Instruction type (0-13)
	dst    uint8  // Destination register (0-7)
	src    uint8  // Source register (0-7) or shift amount
	imm32  uint32 // 32-bit immediate value
}

// superscalarProgram represents a sequence of superscalar instructions
// that compute a dataset item from cache data.
type superscalarProgram struct {
	instructions []superscalarInstruction // Instruction sequence (3-60 instructions)
	addressReg   uint8                    // Register that determines next cache address (0-7)
}

// size returns the number of instructions in the program.
func (p *superscalarProgram) size() int {
	return len(p.instructions)
}
