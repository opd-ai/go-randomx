package randomx

import (
	"math/bits"
)

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
	mod    uint8  // Modifier byte (for shift amount in IADD_RS)
	imm32  uint32 // 32-bit immediate value
}

// getModShift extracts the shift amount from the mod field for IADD_RS instruction.
func (i *superscalarInstruction) getModShift() uint8 {
	return i.mod % 4
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

// reciprocal computes a fast reciprocal approximation for IMUL_RCP instruction.
// This matches the randomx_reciprocal function from the C++ reference.
// divisor must not be 0.
func reciprocal(divisor uint32) uint64 {
	if divisor == 0 {
		divisor = 1 // Avoid division by zero
	}
	
	const p2exp63 = uint64(1) << 63
	q := p2exp63 / uint64(divisor)
	r := p2exp63 % uint64(divisor)
	
	// Count leading zeros to determine shift
	shift := uint32(64 - bits.LeadingZeros32(divisor))
	
	return (q << shift) + ((r << shift) / uint64(divisor))
}

// signExtend2sCompl sign-extends a 32-bit value to 64-bit using two's complement.
func signExtend2sCompl(x uint32) uint64 {
	// If the sign bit (bit 31) is set, extend with 1s, otherwise with 0s
	if x&0x80000000 != 0 {
		return uint64(x) | 0xFFFFFFFF00000000
	}
	return uint64(x)
}

// mulh computes the high 64 bits of the unsigned 128-bit product of a and b.
func mulh(a, b uint64) uint64 {
	hi, _ := bits.Mul64(a, b)
	return hi
}

// smulh computes the high 64 bits of the signed 128-bit product of a and b.
func smulh(a, b int64) int64 {
	// Convert to unsigned for multiplication
	ua, ub := uint64(a), uint64(b)
	hi := mulh(ua, ub)
	
	// Adjust for signed multiplication
	if a < 0 {
		hi -= ub
	}
	if b < 0 {
		hi -= ua
	}
	
	return int64(hi)
}

// rotr performs a right rotation of x by c bits.
func rotr(x uint64, c uint) uint64 {
	return bits.RotateLeft64(x, -int(c))
}
