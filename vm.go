package randomx

import (
	"encoding/binary"
	"math"

	"github.com/opd-ai/go-randomx/internal"
)

// vmConfig holds configuration data parsed from AesGenerator4R output.
type vmConfig struct {
	readReg0 uint8    // Register for spAddr0 XOR
	readReg1 uint8    // Register for spAddr1 XOR
	readReg2 uint8    // Register for mx XOR
	readReg3 uint8    // Register for mx XOR
	eMask    [4]uint64 // Masks for E registers
}

// virtualMachine implements the RandomX virtual machine.
type virtualMachine struct {
	reg  [8]uint64 // Integer register file (r0-r7)
	regF [4]float64 // Floating-point register file (f0-f3)
	regE [4]float64 // E register file (e0-e3)
	mem  []byte     // Scratchpad memory (2 MB)
	ds   *dataset   // Dataset reference (fast mode)
	c    *cache     // Cache reference (light mode)
	ma   uint64     // Memory address register
	mx   uint64     // Memory multiplier

	// Program generation and configuration
	gen4     *aesGenerator4R // Generator for programs
	config   vmConfig        // Current configuration
	spAddr0  uint32          // Scratchpad address 0
	spAddr1  uint32          // Scratchpad address 1
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
	for i := range vm.regF {
		vm.regF[i] = 0
	}
	for i := range vm.regE {
		vm.regE[i] = 0
	}
	if vm.mem != nil {
		for i := range vm.mem {
			vm.mem[i] = 0
		}
	}
	vm.ma = 0
	vm.mx = 0
	vm.spAddr0 = 0
	vm.spAddr1 = 0
}

// run executes the RandomX algorithm on the input.
func (vm *virtualMachine) run(input []byte) [32]byte {
	// Initialize VM state from input
	vm.initialize(input)

	// RandomX algorithm: 8 programs, each executed 2048 times
	const (
		programCount      = 8
		programIterations = 2048
	)

	for progNum := 0; progNum < programCount; progNum++ {
		// Generate new program from AesGenerator4R
		prog := vm.generateProgram()

		// Execute this program 2048 times
		for iter := 0; iter < programIterations; iter++ {
			vm.executeIteration(prog)
		}

		// Update generator state for next program
		// Hash the register file and use as new generator state
		regData := vm.serializeRegisters()
		newState := internal.Blake2b512(regData)
		vm.gen4.setState(newState[:])
	}

	// Finalize hash
	return vm.finalize()
}

// initialize sets up the VM state from input data using the RandomX algorithm.
func (vm *virtualMachine) initialize(input []byte) {
	// Step 1: Hash input to get initial state
	hash := internal.Blake2b512(input)

	// Step 2: Create AesGenerator1R from hash
	gen1, err := newAesGenerator1R(hash[:])
	if err != nil {
		panic("failed to create AesGenerator1R: " + err.Error())
	}

	// Step 3: Fill scratchpad (2 MB) from generator
	// Ensure mem is allocated
	if len(vm.mem) == 0 {
		vm.mem = make([]byte, scratchpadL3Size)
	}
	gen1.getBytes(vm.mem)

	// Step 4: Create AesGenerator4R from gen1 state for program generation
	gen4, err := newAesGenerator4R(gen1.state[:])
	if err != nil {
		panic("failed to create AesGenerator4R: " + err.Error())
	}
	vm.gen4 = gen4
}

// parseConfiguration parses 128 bytes of configuration data from AesGenerator4R.
// This sets up the VM's configuration according to RandomX spec Table 4.5.1.
func (vm *virtualMachine) parseConfiguration(data []byte) {
	if len(data) < 128 {
		panic("configuration data must be at least 128 bytes")
	}

	// Parse readReg values (which registers to use for address calculations)
	// RandomX spec: take individual bytes and mask with 7 to get register index (0-7)
	// BUG FIX: Was incorrectly reading uint32 values, should read individual bytes
	vm.config.readReg0 = data[0] & 7
	vm.config.readReg1 = data[1] & 7
	vm.config.readReg2 = data[2] & 7
	vm.config.readReg3 = data[3] & 7

	// Parse E register masks (used for floating-point operations)
	// BUG FIX: E-masks are consecutive uint64 values starting at byte 8, not byte 16 with stride 16
	// Layout: bytes 0-7 (readReg + padding), bytes 8-39 (4 E-masks), bytes 40-127 (other config)
	const defaultEMask = uint64(0x3FFFFFFFFFFFFFFF) // Default mask to prevent infinity/NaN
	
	for i := 0; i < 4; i++ {
		offset := 8 + i*8 // E masks start at byte 8, each is 8 bytes
		mask := binary.LittleEndian.Uint64(data[offset : offset+8])
		
		// RandomX spec: if bit 62 (sign bit of exponent) is 0, use default mask
		// This ensures E registers contain valid floating-point values
		if (mask & (1 << 62)) == 0 {
			vm.config.eMask[i] = defaultEMask
		} else {
			vm.config.eMask[i] = mask
		}
	}
}

// generateProgram creates a RandomX program from AesGenerator4R output.
func (vm *virtualMachine) generateProgram() *program {
	p := &program{}

	// Step 1: Read and parse configuration data (128 bytes)
	configData := make([]byte, 128)
	vm.gen4.getBytes(configData)
	vm.parseConfiguration(configData)

	// Step 2: Read program data (2048 bytes = 256 instructions × 8 bytes)
	programData := make([]byte, 2048)
	vm.gen4.getBytes(programData)

	// Step 3: Decode instructions
	for i := 0; i < programLength; i++ {
		p.instructions[i] = decodeInstruction(programData[i*8 : i*8+8])
	}

	return p
}

// executeIteration executes one iteration of the VM program loop.
// This implements the 12-step process per RandomX spec Section 4.6.2.
func (vm *virtualMachine) executeIteration(prog *program) {
	// Step 1: Update scratchpad addresses with register values
	vm.spAddr0 ^= uint32(vm.reg[vm.config.readReg0])
	vm.spAddr1 ^= uint32(vm.reg[vm.config.readReg1])

	// Align to 64-byte cache lines
	vm.spAddr0 &= 0x1FFFC0 // Mask to align to 64-byte boundary in scratchpad
	vm.spAddr1 &= 0x1FFFC0

	// Step 2: Read 64 bytes from Scratchpad[spAddr0] and XOR with r0-r7
	for i := 0; i < 8; i++ {
		vm.reg[i] ^= vm.readMemory(vm.spAddr0 + uint32(i*8))
	}

	// Step 3: Read 64 bytes from Scratchpad[spAddr1] to initialize f0-f3 and e0-e3
	for i := 0; i < 4; i++ {
		// Load f registers (first 32 bytes) - apply float mask
		fVal := vm.readMemory(vm.spAddr1 + uint32(i*8))
		vm.regF[i] = maskFloat(math.Float64frombits(fVal))

		// Load e registers (next 32 bytes) - apply eMask from configuration
		eVal := vm.readMemory(vm.spAddr1 + 32 + uint32(i*8))
		// Apply eMask to limit exponent range
		eValMasked := eVal & vm.config.eMask[i]
		vm.regE[i] = maskFloat(math.Float64frombits(eValMasked))
	}

	// Step 4: Execute all 256 instructions in the program
	for i := 0; i < programLength; i++ {
		vm.executeInstruction(&prog.instructions[i])
	}

	// Step 5: XOR mx with readReg2 and readReg3
	vm.mx ^= vm.reg[vm.config.readReg2]
	vm.mx ^= vm.reg[vm.config.readReg3]

	// Step 6-7: Read dataset item and XOR with registers
	vm.mixDataset()

	// Step 8: Swap mx and ma
	vm.mx, vm.ma = vm.ma, vm.mx

	// Step 9: Write r0-r7 to Scratchpad[spAddr1]
	for i := 0; i < 8; i++ {
		vm.writeMemory(vm.spAddr1+uint32(i*8), vm.reg[i])
	}

	// Step 10: XOR f0-f3 with e0-e3
	for i := 0; i < 4; i++ {
		vm.regF[i] += vm.regE[i]
	}

	// Step 11: Write f0-f3 to Scratchpad[spAddr0]
	for i := 0; i < 4; i++ {
		vm.writeMemory(vm.spAddr0+uint32(i*8), floatToUint64(vm.regF[i]))
	}

	// Step 12: Update spAddr0 (this happens automatically on next iteration)
}

// serializeRegisters serializes the register file for hashing.
// This is used to update the generator state between programs.
func (vm *virtualMachine) serializeRegisters() []byte {
	// Serialize: r0-r7 (64 bytes) + f0-f3 (32 bytes) + e0-e3 (32 bytes) = 128 bytes
	data := make([]byte, 128)

	// Integer registers
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint64(data[i*8:], vm.reg[i])
	}

	// Floating-point registers
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint64(data[64+i*8:], floatToUint64(vm.regF[i]))
	}

	// E registers
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint64(data[96+i*8:], floatToUint64(vm.regE[i]))
	}

	return data
}

// mixDataset mixes dataset items into the register file.
func (vm *virtualMachine) mixDataset() {
	// Use mx to select dataset item
	var itemData [64]byte

	if vm.ds != nil {
		// Fast mode: read from dataset
		index := vm.mx % datasetItems
		copy(itemData[:], vm.ds.getItem(index))
	} else if vm.c != nil {
		// Light mode: compute dataset item on-demand from cache
		// BUG FIX: Was incorrectly returning raw cache item instead of computing dataset item
		index := vm.mx % datasetItems
		vm.computeDatasetItem(index, itemData[:])
	} else {
		return
	}

	// XOR dataset item (64 bytes) into registers r0-r7
	for i := 0; i < 8; i++ {
		val := binary.LittleEndian.Uint64(itemData[i*8 : i*8+8])
		vm.reg[i] ^= val
	}

	// Update ma for next iteration
	vm.ma = vm.mx
}

// computeDatasetItem generates a single dataset item on-demand from the cache.
// This is used in light mode and implements dataset item generation.
// 
// NOTE: This is a simplified implementation that doesn't use superscalar programs.
// For full RandomX compatibility, superscalar program generation and execution
// would be required. This implementation uses the constants and structure from
// the RandomX specification to approximate the correct behavior.
func (vm *virtualMachine) computeDatasetItem(itemNumber uint64, output []byte) {
	// RandomX constants for dataset item initialization (from spec)
	const (
		superscalarMul0  = 6364136223846793005
		superscalarAdd1  = 9298411001130361340
		superscalarAdd2  = 12065312585734608966
		superscalarAdd3  = 9306329213124626780
		superscalarAdd4  = 5281919268842080866
		superscalarAdd5  = 10536153434571861004
		superscalarAdd6  = 3398623926847679864
		superscalarAdd7  = 9549104520008361294
	)
	
	// Initialize register file according to RandomX spec
	var registers [8]uint64
	registers[0] = (itemNumber + 1) * superscalarMul0
	registers[1] = registers[0] ^ superscalarAdd1
	registers[2] = registers[0] ^ superscalarAdd2
	registers[3] = registers[0] ^ superscalarAdd3
	registers[4] = registers[0] ^ superscalarAdd4
	registers[5] = registers[0] ^ superscalarAdd5
	registers[6] = registers[0] ^ superscalarAdd6
	registers[7] = registers[0] ^ superscalarAdd7

	// Mix with cache items (8 iterations as per RandomX spec)
	registerValue := itemNumber
	const iterations = 8
	
	for i := 0; i < iterations; i++ {
		// Get cache item based on register value
		cacheIndex := uint32(registerValue % cacheItems)
		cacheItem := vm.c.getItem(cacheIndex)

		// XOR cache item into registers
		for r := 0; r < 8; r++ {
			val := binary.LittleEndian.Uint64(cacheItem[r*8 : r*8+8])
			registers[r] ^= val
		}
		
		// Apply simple mixing to simulate superscalar program effect
		// This is a placeholder for proper superscalar program execution
		for r := 0; r < 8; r++ {
			registers[r] = mixRegister(registers[r], uint64(i))
		}
		
		// Update register value for next cache access
		// Use r0 as the address register (simplified)
		registerValue = registers[0]
	}

	// Write final register state to output
	for r := 0; r < 8; r++ {
		binary.LittleEndian.PutUint64(output[r*8:r*8+8], registers[r])
	}
}

// finalize produces the final hash output using the RandomX finalization algorithm.
func (vm *virtualMachine) finalize() [32]byte {
	// Step 1: Hash the scratchpad with AesHash1R
	hasher, err := newAesHash1R()
	if err != nil {
		panic("failed to create AesHash1R: " + err.Error())
	}
	scratchpadHash := hasher.hash(vm.mem)

	// Step 2: Serialize register file (256 bytes)
	// Include integer registers, floating-point registers, and E registers
	regData := make([]byte, 256)

	// Integer registers (r0-r7): 64 bytes
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint64(regData[i*8:], vm.reg[i])
	}

	// Floating-point registers (f0-f3): 32 bytes
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint64(regData[64+i*8:], floatToUint64(vm.regF[i]))
	}

	// E registers (e0-e3): 32 bytes
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint64(regData[96+i*8:], floatToUint64(vm.regE[i]))
	}

	// Add ma and mx: 16 bytes
	binary.LittleEndian.PutUint64(regData[128:], vm.ma)
	binary.LittleEndian.PutUint64(regData[136:], vm.mx)

	// Pad remaining bytes to 256
	// (rest is left as zeros)

	// Step 3: Concatenate scratchpad hash (64 bytes) + register file (256 bytes)
	combined := make([]byte, 320)
	copy(combined[0:64], scratchpadHash[:])
	copy(combined[64:], regData)

	// Step 4: Final Blake2b-256 hash
	return internal.Blake2b256(combined)
}

// executeInstruction executes a single VM instruction using the full RandomX instruction set.
func (vm *virtualMachine) executeInstruction(instr *instruction) {
	// Use the full instruction executor from instructions.go
	vm.executeInstructionFull(instr)
}

// getMemoryAddress computes memory address for load/store operations.
// The mod field determines which scratchpad level (L1/L2/L3) is accessed.
func (vm *virtualMachine) getMemoryAddress(instr *instruction) uint32 {
	// Calculate base address from src register + immediate
	addr := vm.reg[instr.src] + uint64(instr.imm)
	
	// Determine scratchpad level based on mod % 4
	// mod % 4 == 0: L3 (full 2 MB)
	// mod % 4 == 1: L2 (256 KB)
	// mod % 4 == 2: L1 (16 KB)
	// mod % 4 == 3: L2 (256 KB)
	switch instr.mod % 4 {
	case 0:
		// L3 level - full scratchpad
		return uint32(addr & scratchpadL3Mask)
	case 1, 3:
		// L2 level - 256 KB
		return uint32(addr & scratchpadL2Mask)
	case 2:
		// L1 level - 16 KB
		return uint32(addr & scratchpadL1Mask)
	default:
		return uint32(addr & scratchpadL3Mask)
	}
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
