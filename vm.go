package randomx

import (
	"encoding/binary"
	"math"

	"github.com/opd-ai/go-randomx/internal"
)

// virtualMachine implements the RandomX virtual machine.
type virtualMachine struct {
	reg [8]uint64 // Integer register file (r0-r7)
	mem []byte    // Scratchpad memory (2 MB)
	ds  *dataset  // Dataset reference (fast mode)
	c   *cache    // Cache reference (light mode)
	ma  uint64    // Memory address register
	mx  uint64    // Memory multiplier
}

// init initializes the VM with dataset or cache.
func (vm *virtualMachine) init(ds *dataset, c *cache) {
	vm.ds = ds
	vm.c = c
	vm.reset()
}

// reset clears the VM state for reuse.
func (vm *virtualMachine) reset() {
	for i := range vm.reg {
		vm.reg[i] = 0
	}
	if vm.mem != nil {
		for i := range vm.mem {
			vm.mem[i] = 0
		}
	}
	vm.ma = 0
	vm.mx = 0
}

// run executes the RandomX algorithm on the input.
func (vm *virtualMachine) run(input []byte) [32]byte {
	// Initialize VM state from input
	vm.initialize(input)

	// Execute RandomX program iterations
	const iterations = 8
	for i := 0; i < iterations; i++ {
		// Generate program for this iteration
		prog := generateProgram(input)

		// Execute the program
		prog.execute(vm)

		// Mix dataset/cache into registers
		vm.mixDataset()
	}

	// Finalize hash
	return vm.finalize()
}

// initialize sets up the VM state from input data.
func (vm *virtualMachine) initialize(input []byte) {
	// Hash input to get initial state
	hash := internal.Blake2b512(input)

	// Initialize registers from hash
	for i := 0; i < 8; i++ {
		vm.reg[i] = binary.LittleEndian.Uint64(hash[i*8 : i*8+8])
	}

	// Initialize scratchpad from registers
	vm.fillScratchpad()

	// Set memory access parameters
	vm.ma = vm.reg[0]
	vm.mx = vm.reg[1] | 0x01 // Ensure odd for proper mixing
}

// fillScratchpad initializes scratchpad memory.
func (vm *virtualMachine) fillScratchpad() {
	// Fill scratchpad with AES encryption of register state
	if len(vm.mem) < scratchpadL3Size {
		return
	}

	// Use registers as AES keys
	key := make([]byte, 32)
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint64(key[i*8:], vm.reg[i])
	}

	// Fill memory in blocks
	aesEnc, err := internal.NewAESEncryptor(key[:16])
	if err != nil {
		return
	}

	block := make([]byte, 16)
	for i := 0; i < scratchpadL3Size; i += 16 {
		binary.LittleEndian.PutUint64(block[0:8], uint64(i))
		binary.LittleEndian.PutUint64(block[8:16], uint64(i+8))
		aesEnc.Encrypt(vm.mem[i:i+16], block)
	}
}

// mixDataset mixes dataset items into the register file.
func (vm *virtualMachine) mixDataset() {
	// Get dataset/cache item based on register state
	var itemData []byte

	if vm.ds != nil {
		// Fast mode: read from dataset
		index := vm.reg[0] % datasetItems
		itemData = vm.ds.getItem(index)
	} else if vm.c != nil {
		// Light mode: compute item from cache
		index := uint32(vm.reg[0] % cacheItems)
		itemData = vm.c.getItem(index)
	} else {
		return
	}

	// XOR dataset item into registers
	for i := 0; i < 8 && i*8 < len(itemData); i++ {
		val := binary.LittleEndian.Uint64(itemData[i*8 : i*8+8])
		vm.reg[i] ^= val
	}
}

// finalize produces the final hash output.
func (vm *virtualMachine) finalize() [32]byte {
	// Mix final register state
	for i := 0; i < 8; i++ {
		vm.reg[i] ^= vm.readMemory(uint32(i * 8))
	}

	// Hash register file to produce output
	output := make([]byte, 64)
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint64(output[i*8:i*8+8], vm.reg[i])
	}

	return internal.Blake2b256(output)
}

// executeInstruction executes a single VM instruction.
func (vm *virtualMachine) executeInstruction(instr *instruction) {
	// Decode destination and source registers
	dst := instr.dst & 0x07
	src := instr.src & 0x07

	// Execute based on opcode
	switch instr.opcode % 16 {
	case 0: // ADD
		vm.reg[dst] += vm.reg[src]
	case 1: // SUB
		vm.reg[dst] -= vm.reg[src]
	case 2: // MUL
		vm.reg[dst] *= vm.reg[src]
	case 3: // XOR
		vm.reg[dst] ^= vm.reg[src]
	case 4: // ROR (rotate right)
		vm.reg[dst] = rotateRight64(vm.reg[dst], uint(vm.reg[src]&63))
	case 5: // LOAD
		addr := vm.getMemoryAddress(instr)
		vm.reg[dst] = vm.readMemory(addr)
	case 6: // STORE
		addr := vm.getMemoryAddress(instr)
		vm.writeMemory(addr, vm.reg[src])
	case 7: // ADD immediate
		vm.reg[dst] += uint64(instr.imm)
	case 8: // SUB immediate
		vm.reg[dst] -= uint64(instr.imm)
	case 9: // MUL immediate
		vm.reg[dst] *= uint64(instr.imm)
	case 10: // XOR immediate
		vm.reg[dst] ^= uint64(instr.imm)
	case 11: // ROR immediate
		vm.reg[dst] = rotateRight64(vm.reg[dst], uint(instr.imm&63))
	case 12: // AND
		vm.reg[dst] &= vm.reg[src]
	case 13: // OR
		vm.reg[dst] |= vm.reg[src]
	case 14: // FPADD (floating point add)
		vm.reg[dst] = floatToUint64(uint64ToFloat(vm.reg[dst]) + uint64ToFloat(vm.reg[src]))
	case 15: // FPMUL (floating point multiply)
		vm.reg[dst] = floatToUint64(uint64ToFloat(vm.reg[dst]) * uint64ToFloat(vm.reg[src]))
	}
}

// getMemoryAddress computes memory address for load/store operations.
func (vm *virtualMachine) getMemoryAddress(instr *instruction) uint32 {
	addr := vm.reg[instr.src] + uint64(instr.imm)
	return uint32(addr % scratchpadL3Size)
}

// readMemory reads a 64-bit value from scratchpad memory.
func (vm *virtualMachine) readMemory(addr uint32) uint64 {
	addr = addr % uint32(len(vm.mem))
	addr &= ^uint32(7) // Align to 8 bytes
	if addr+8 > uint32(len(vm.mem)) {
		addr = uint32(len(vm.mem)) - 8
	}
	return binary.LittleEndian.Uint64(vm.mem[addr : addr+8])
}

// writeMemory writes a 64-bit value to scratchpad memory.
func (vm *virtualMachine) writeMemory(addr uint32, value uint64) {
	addr = addr % uint32(len(vm.mem))
	addr &= ^uint32(7) // Align to 8 bytes
	if addr+8 > uint32(len(vm.mem)) {
		addr = uint32(len(vm.mem)) - 8
	}
	binary.LittleEndian.PutUint64(vm.mem[addr:addr+8], value)
}

// Helper functions for bit operations and floating point

func rotateRight64(val uint64, shift uint) uint64 {
	shift &= 63
	return (val >> shift) | (val << (64 - shift))
}

func uint64ToFloat(val uint64) float64 {
	return math.Float64frombits(val)
}

func floatToUint64(val float64) uint64 {
	return math.Float64bits(val)
}
