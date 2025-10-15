package randomx

import (
	"encoding/binary"

	"github.com/opd-ai/go-randomx/internal"
)

const (
	// Program length in instructions
	programLength = 256

	// Program buffer size
	programSize = programLength * 8
)

// instruction represents a single RandomX VM instruction.
type instruction struct {
	opcode uint8
	dst    uint8
	src    uint8
	mod    uint8
	imm    uint32
}

// program represents a RandomX program (sequence of instructions).
type program struct {
	instructions [programLength]instruction
}

// generateProgram creates a RandomX program from input data.
// Uses Blake2b to deterministically generate instructions.
func generateProgram(input []byte) *program {
	p := &program{}

	// Generate program entropy using Blake2b
	entropy := hashProgramEntropy(input)

	// Decode instructions from entropy
	for i := 0; i < programLength; i++ {
		offset := i * 8
		if offset+8 > len(entropy) {
			// Need more entropy, re-hash
			entropy = hashProgramEntropy(entropy)
			offset = 0
		}

		p.instructions[i] = decodeInstruction(entropy[offset : offset+8])
	}

	return p
}

// hashProgramEntropy generates entropy for program generation.
func hashProgramEntropy(input []byte) []byte {
	// Use Blake2b-512 to generate 64 bytes of entropy
	hash := internal.Blake2b512(input)

	// Need more entropy, chain hashes
	output := make([]byte, programSize)
	copy(output, hash[:])

	for i := 64; i < programSize; i += 64 {
		hash = internal.Blake2b512(hash[:])
		remaining := programSize - i
		if remaining > 64 {
			remaining = 64
		}
		copy(output[i:], hash[:remaining])
	}

	return output
}

// decodeInstruction decodes an 8-byte sequence into an instruction.
func decodeInstruction(data []byte) instruction {
	raw := binary.LittleEndian.Uint64(data)

	return instruction{
		opcode: uint8(raw & 0xFF),
		dst:    uint8((raw >> 8) & 0x07),  // 3 bits: register 0-7
		src:    uint8((raw >> 16) & 0x07), // 3 bits: register 0-7
		mod:    uint8((raw >> 24) & 0xFF),
		imm:    uint32(raw >> 32),
	}
}

// execute runs the program on the VM.
func (p *program) execute(vm *virtualMachine) {
	for i := 0; i < programLength; i++ {
		vm.executeInstruction(&p.instructions[i])
	}
}
