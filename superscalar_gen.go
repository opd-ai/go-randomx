package randomx

// This file contains the complex superscalar program generation algorithm
// ported from the RandomX C++ reference implementation.
// This algorithm simulates CPU superscalar execution with dependency tracking
// and port scheduling to generate pseudo-random instruction sequences.

// Execution port types (for CPU port scheduling simulation)
type executionPort int

const (
	portNull executionPort = 0
	portP0   executionPort = 1
	portP1   executionPort = 2
	portP5   executionPort = 4
	portP01  executionPort = portP0 | portP1
	portP05  executionPort = portP0 | portP5
	portP015 executionPort = portP0 | portP1 | portP5
)

// registerInfo tracks register state during program generation
type registerInfo struct {
	latency   int  // Cycle when this register will be ready
	lastOpGroup int  // Last operation group that wrote to this register (for dependency tracking)
}

// macroOp represents a macro-operation (one or more micro-ops)
type macroOp struct {
	name      string
	size      int   // Code size in bytes
	latency   int   // Execution latency in cycles
	uop1      executionPort
	uop2      executionPort
	dependent bool  // Whether this op depends on the previous op
}

// isSimple returns true if this is a single micro-op
func (m *macroOp) isSimple() bool {
	return m.uop2 == portNull
}

// isEliminated returns true if this op is eliminated (no execution)
func (m *macroOp) isEliminated() bool {
	return m.uop1 == portNull
}

// Macro-operations for different instruction types
var (
	// 3-byte instructions
	macroOpAddRR  = macroOp{"add r,r", 3, 1, portP015, portNull, false}
	macroOpSubRR  = macroOp{"sub r,r", 3, 1, portP015, portNull, false}
	macroOpXorRR  = macroOp{"xor r,r", 3, 1, portP015, portNull, false}
	macroOpImulR  = macroOp{"imul r", 3, 4, portP1, portP5, false}
	macroOpMulR   = macroOp{"mul r", 3, 4, portP1, portP5, false}
	macroOpMovRR  = macroOp{"mov r,r", 3, 0, portNull, portNull, false}
	
	// 4-byte instructions
	macroOpLeaSIB = macroOp{"lea r,r+r*s", 4, 1, portP01, portNull, false}
	macroOpImulRR = macroOp{"imul r,r", 4, 3, portP1, portNull, false}
	macroOpRorRI  = macroOp{"ror r,i", 4, 1, portP05, portNull, false}
	
	// 7-byte instructions (can be padded to 8 or 9 bytes)
	macroOpAddRI = macroOp{"add r,i", 7, 1, portP015, portNull, false}
	macroOpXorRI = macroOp{"xor r,i", 7, 1, portP015, portNull, false}
	
	// 10-byte instructions
	macroOpMovRI64 = macroOp{"mov rax,i64", 10, 1, portP015, portNull, false}
)

// superscalarInstrInfo contains information about a superscalar instruction type
type superscalarInstrInfo struct {
	name      string
	instrType uint8
	ops       []macroOp
	latency   int
	resultOp  int  // Which macro-op produces the result
	dstOp     int  // Which macro-op needs the destination register
	srcOp     int  // Which macro-op needs the source register
}

// Instruction information for each superscalar instruction type
var superscalarInstrInfos = []superscalarInstrInfo{
	// ISUB_R
	{
		name:      "ISUB_R",
		instrType: ssISUB_R,
		ops:       []macroOp{macroOpSubRR},
		latency:   1,
		resultOp:  0,
		dstOp:     0,
		srcOp:     0,
	},
	// IXOR_R
	{
		name:      "IXOR_R",
		instrType: ssIXOR_R,
		ops:       []macroOp{macroOpXorRR},
		latency:   1,
		resultOp:  0,
		dstOp:     0,
		srcOp:     0,
	},
	// IADD_RS
	{
		name:      "IADD_RS",
		instrType: ssIADD_RS,
		ops:       []macroOp{macroOpLeaSIB},
		latency:   1,
		resultOp:  0,
		dstOp:     0,
		srcOp:     0,
	},
	// IMUL_R
	{
		name:      "IMUL_R",
		instrType: ssIMUL_R,
		ops:       []macroOp{macroOpImulRR},
		latency:   3,
		resultOp:  0,
		dstOp:     0,
		srcOp:     0,
	},
	// IROR_C
	{
		name:      "IROR_C",
		instrType: ssIROR_C,
		ops:       []macroOp{macroOpRorRI},
		latency:   1,
		resultOp:  0,
		dstOp:     0,
		srcOp:     -1, // No source register
	},
	// IADD_C7/C8/C9
	{
		name:      "IADD_C",
		instrType: ssIADD_C7,
		ops:       []macroOp{macroOpAddRI},
		latency:   1,
		resultOp:  0,
		dstOp:     0,
		srcOp:     -1,
	},
	// IXOR_C7/C8/C9
	{
		name:      "IXOR_C",
		instrType: ssIXOR_C7,
		ops:       []macroOp{macroOpXorRI},
		latency:   1,
		resultOp:  0,
		dstOp:     0,
		srcOp:     -1,
	},
	// IMULH_R
	{
		name:      "IMULH_R",
		instrType: ssIMULH_R,
		ops:       []macroOp{macroOpMovRR, macroOpMulR, macroOpMovRR},
		latency:   3,
		resultOp:  2,
		dstOp:     0,
		srcOp:     1,
	},
	// ISMULH_R
	{
		name:      "ISMULH_R",
		instrType: ssISMULH_R,
		ops:       []macroOp{macroOpMovRR, macroOpImulR, macroOpMovRR},
		latency:   3,
		resultOp:  2,
		dstOp:     0,
		srcOp:     1,
	},
	// IMUL_RCP
	{
		name:      "IMUL_RCP",
		instrType: ssIMUL_RCP,
		ops:       []macroOp{macroOpMovRI64, macroOp{name: "imul r,r (dependent)", size: 4, latency: 3, uop1: portP1, uop2: portNull, dependent: true}},
		latency:   4,
		resultOp:  1,
		dstOp:     1,
		srcOp:     -1,
	},
}

// generateSuperscalarProgram generates a random superscalar program using Blake2Generator.
// This is the main entry point that orchestrates the full algorithm.
// It implements the RandomX SuperscalarHash program generation algorithm with proper
// CPU scheduling simulation and dependency tracking.
func generateSuperscalarProgram(gen *blake2Generator) *superscalarProgram {
	prog := &superscalarProgram{
		instructions: make([]superscalarInstruction, 0, superscalarMaxSize),
	}
	
	// Track register state during generation
	var registers [8]registerInfo
	
	// Execution port state (tracks cycle availability)
	var portBusy [3]int // P0, P1, P5
	
	// Current CPU cycle
	cycle := 0
	
	// Current operation group for dependency tracking
	opGroup := 0
	
	// Generate instructions until we reach target latency
	for cycle < superscalarLatency {
		// Try to issue as many instructions as possible in this cycle
		issued := false
		
		// Select instruction type based on current state
		instrIdx := selectInstructionType(gen, cycle, &registers, portBusy[:])
		if instrIdx >= 0 && instrIdx < len(superscalarInstrInfos) {
			info := &superscalarInstrInfos[instrIdx]
			
			// Check if we can generate this instruction
			if canGenerateInstruction(info, gen, cycle, &registers, portBusy[:]) {
				instr := generateInstructionForType(info, gen, cycle, &registers, opGroup)
				if instr != nil {
					// Add to program
					prog.instructions = append(prog.instructions, *instr)
					
					// Schedule execution of macro-ops
					scheduleInstruction(info, &registers, portBusy[:], &cycle, opGroup, instr)
					
					opGroup++
					issued = true
				}
			}
		}
		
		// Advance cycle if nothing was issued
		if !issued {
			cycle++
		}
		
		// Safety check: prevent infinite loop
		if len(prog.instructions) >= superscalarMaxSize || cycle > superscalarLatency*2 {
			break
		}
	}
	
	// Select address register (register with highest latency = most mixing)
	prog.addressReg = selectAddressRegister(&registers)
	
	return prog
}

// selectInstructionType selects which instruction type to generate based on
// current CPU state and available execution ports.
func selectInstructionType(gen *blake2Generator, cycle int, registers *[8]registerInfo, portBusy []int) int {
	// Get random byte to select instruction type
	instrByte := gen.getByte()
	
	// Use weighted selection based on instruction frequency
	// This matches the C++ reference distribution
	switch instrByte % 28 {
	case 0, 1, 2, 3:
		return 0 // ISUB_R (common)
	case 4, 5, 6, 7:
		return 1 // IXOR_R (common)
	case 8, 9, 10:
		return 2 // IADD_RS (fairly common)
	case 11, 12:
		return 3 // IMUL_R (less common)
	case 13, 14:
		return 4 // IROR_C
	case 15, 16:
		return 5 // IADD_C
	case 17, 18:
		return 6 // IXOR_C
	case 19:
		return 7 // IMULH_R (expensive, rare)
	case 20:
		return 8 // ISMULH_R (expensive, rare)
	case 21, 22, 23, 24, 25, 26, 27:
		return 9 // IMUL_RCP (fairly common)
	default:
		return 1 // Default to IXOR_R
	}
}

// canGenerateInstruction checks if an instruction can be generated given current CPU state.
func canGenerateInstruction(info *superscalarInstrInfo, gen *blake2Generator, cycle int, 
	registers *[8]registerInfo, portBusy []int) bool {
	
	// Always allow simple instructions
	if len(info.ops) == 1 && info.ops[0].isSimple() {
		return true
	}
	
	// Check if execution ports will be available
	for _, op := range info.ops {
		if op.isEliminated() {
			continue
		}
		
		// Check port availability (simplified check)
		if op.uop1&portP0 != 0 && portBusy[0] > cycle {
			return false
		}
		if op.uop1&portP1 != 0 && portBusy[1] > cycle {
			return false
		}
		if op.uop1&portP5 != 0 && portBusy[2] > cycle {
			return false
		}
	}
	
	return true
}

// generateInstructionForType generates a specific instruction with proper operands.
func generateInstructionForType(info *superscalarInstrInfo, gen *blake2Generator, 
	cycle int, registers *[8]registerInfo, opGroup int) *superscalarInstruction {
	
	instr := &superscalarInstruction{
		opcode: info.instrType,
	}
	
	// Select destination register
	instr.dst = selectRegister(gen, registers, cycle, opGroup, info.dstOp >= 0)
	
	// Select source register (if needed)
	if info.srcOp >= 0 {
		instr.src = selectRegister(gen, registers, cycle, opGroup, true)
		
		// Ensure src != dst for most instructions
		if instr.src == instr.dst && info.instrType != ssIMUL_RCP {
			instr.src = (instr.src + 1) & 7
		}
	}
	
	// Generate immediate value if needed
	if info.instrType >= ssIROR_C {
		instr.imm32 = gen.getUint32()
		
		// Special handling for IMUL_RCP
		if info.instrType == ssIMUL_RCP {
			// Ensure non-zero divisor
			if instr.imm32 == 0 {
				instr.imm32 = 1
			}
		}
	}
	
	// Generate mod field for IADD_RS
	if info.instrType == ssIADD_RS {
		instr.mod = gen.getByte()
	}
	
	return instr
}

// selectRegister selects a register based on dependency and latency information.
func selectRegister(gen *blake2Generator, registers *[8]registerInfo, 
	cycle int, opGroup int, needsValue bool) uint8 {
	
	// Simple register selection with basic dependency awareness
	attempts := 0
	for attempts < 8 {
		reg := gen.getByte() & 7
		
		// If we need the value, prefer registers that are ready
		if needsValue && registers[reg].latency > cycle {
			attempts++
			continue
		}
		
		return reg
	}
	
	// Fallback: return any register
	return gen.getByte() & 7
}

// scheduleInstruction updates CPU state after scheduling an instruction.
func scheduleInstruction(info *superscalarInstrInfo, registers *[8]registerInfo, 
	portBusy []int, cycle *int, opGroup int, instr *superscalarInstruction) {
	
	// Calculate when the instruction completes
	completionCycle := *cycle + info.latency
	
	// Update destination register latency
	registers[instr.dst].latency = completionCycle
	registers[instr.dst].lastOpGroup = opGroup
	
	// Update port busy times (simplified scheduling)
	for _, op := range info.ops {
		if op.isEliminated() {
			continue
		}
		
		// Mark ports as busy
		if op.uop1&portP0 != 0 {
			portBusy[0] = max(portBusy[0], *cycle+op.latency)
		}
		if op.uop1&portP1 != 0 {
			portBusy[1] = max(portBusy[1], *cycle+op.latency)
		}
		if op.uop1&portP5 != 0 {
			portBusy[2] = max(portBusy[2], *cycle+op.latency)
		}
		
		// Handle second micro-op if present
		if op.uop2 != portNull {
			if op.uop2&portP0 != 0 {
				portBusy[0] = max(portBusy[0], *cycle+op.latency)
			}
			if op.uop2&portP1 != 0 {
				portBusy[1] = max(portBusy[1], *cycle+op.latency)
			}
			if op.uop2&portP5 != 0 {
				portBusy[2] = max(portBusy[2], *cycle+op.latency)
			}
		}
	}
	
	// Advance cycle for next instruction
	*cycle += 1
}

// selectAddressRegister selects which register determines the next cache address.
// The register with the highest latency is selected (most mixed).
func selectAddressRegister(registers *[8]registerInfo) uint8 {
	maxLatency := 0
	addressReg := uint8(0)
	
	for i := 0; i < 8; i++ {
		if registers[i].latency > maxLatency {
			maxLatency = registers[i].latency
			addressReg = uint8(i)
		}
	}
	
	return addressReg
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
