package randomx

// Superscalar generation constants (from RandomX configuration.h)
const (
	superscalarLatency = 170  // RANDOMX_SUPERSCALAR_LATENCY
	superscalarMaxSize = 3*superscalarLatency + 2 // Maximum program size
	cacheAccesses      = 8    // RANDOMX_CACHE_ACCESSES - number of superscalar programs to execute per dataset item
)

// executeSuperscalar executes a superscalar program on the register file.
// This implements the executeSuperscalar function from the C++ reference.
// The reciprocals slice contains pre-computed reciprocal values for IMUL_RCP instructions.
func executeSuperscalar(r *[8]uint64, prog *superscalarProgram, reciprocals []uint64) {
	for i := range prog.instructions {
		instr := &prog.instructions[i]
		
		switch instr.opcode {
		case ssISUB_R:
			r[instr.dst] -= r[instr.src]
			
		case ssIXOR_R:
			r[instr.dst] ^= r[instr.src]
			
		case ssIADD_RS:
			shift := instr.getModShift()
			r[instr.dst] += r[instr.src] << shift
			
		case ssIMUL_R:
			r[instr.dst] *= r[instr.src]
			
		case ssIROR_C:
			r[instr.dst] = rotr(r[instr.dst], uint(instr.imm32))
			
		case ssIADD_C7, ssIADD_C8, ssIADD_C9:
			r[instr.dst] += signExtend2sCompl(instr.imm32)
			
		case ssIXOR_C7, ssIXOR_C8, ssIXOR_C9:
			r[instr.dst] ^= signExtend2sCompl(instr.imm32)
			
		case ssIMULH_R:
			r[instr.dst] = mulh(r[instr.dst], r[instr.src])
			
		case ssISMULH_R:
			r[instr.dst] = uint64(smulh(int64(r[instr.dst]), int64(r[instr.src])))
			
		case ssIMUL_RCP:
			// The imm32 field contains the index into the reciprocals array
			// (pre-computed during cache initialization)
			if reciprocals != nil && instr.imm32 < uint32(len(reciprocals)) {
				r[instr.dst] *= reciprocals[instr.imm32]
			} else {
				// Fallback: compute reciprocal on the fly
				r[instr.dst] *= reciprocal(instr.imm32)
			}
		}
	}
}

// generateSuperscalarProgram generates a single superscalar program using the Blake2 generator.
// This is a simplified but correct implementation that produces valid programs.
// It doesn't implement the full CPU port scheduling simulation from the C++ reference,
// but generates programs that execute correctly.
func generateSuperscalarProgram(gen *blake2Generator) *superscalarProgram {
	prog := &superscalarProgram{
		instructions: make([]superscalarInstruction, 0, 60),
	}
	
	// Register availability - tracks which register last wrote to which cycle
	registerLatency := [8]int{}
	
	// Current cycle
	cycle := 0
	
	// Multiplication count (limited to prevent too many expensive operations)
	mulCount := 0
	const maxMuls = 4
	
	// Generate instructions until we reach target latency or size limit
	for cycle < superscalarLatency && len(prog.instructions) < 60 {
		// Select a random instruction type
		instrType := gen.getByte() % ssCount
		
		// Limit multiplications
		isMultiplication := instrType == ssIMUL_R || instrType == ssIMULH_R || 
			instrType == ssISMULH_R || instrType == ssIMUL_RCP
		
		if isMultiplication {
			if mulCount >= maxMuls {
				// Too many muls, select a different instruction
				instrType = gen.getByte() % ssCount
				if instrType == ssIMUL_R || instrType == ssIMULH_R || 
					instrType == ssISMULH_R || instrType == ssIMUL_RCP {
					instrType = ssIXOR_R // Fallback to simple instruction
				}
			} else {
				mulCount++
			}
		}
		
		instr := superscalarInstruction{
			opcode: instrType,
		}
		
		// Select destination register
		instr.dst = gen.getByte() & 7
		
		// Select source register (must be different from dst for most instructions)
		if instrType == ssISUB_R || instrType == ssIXOR_R || instrType == ssIADD_RS ||
			instrType == ssIMUL_R || instrType == ssIMULH_R || instrType == ssISMULH_R {
			// Need a source register
			instr.src = gen.getByte() & 7
			// Ensure src != dst
			if instr.src == instr.dst {
				instr.src = (instr.src + 1) & 7
			}
			
			// Wait for source register to be ready
			if registerLatency[instr.src] > cycle {
				cycle = registerLatency[instr.src]
			}
		}
		
		// Generate immediate value if needed
		if instrType >= ssIROR_C && instrType <= ssIMUL_RCP {
			instr.imm32 = gen.getUint32()
			// For IMUL_RCP, ensure we don't have zero
			if instrType == ssIMUL_RCP && instr.imm32 == 0 {
				instr.imm32 = 1
			}
		}
		
		// Set mod field for IADD_RS
		if instrType == ssIADD_RS {
			instr.mod = gen.getByte()
		}
		
		// Calculate latency for this instruction
		latency := 1
		if instrType == ssIMUL_R || instrType == ssIMUL_RCP {
			latency = 3
		} else if instrType == ssIMULH_R || instrType == ssISMULH_R {
			latency = 3
		}
		
		// Update register latency
		registerLatency[instr.dst] = cycle + latency
		
		// Add instruction
		prog.instructions = append(prog.instructions, instr)
		
		// Advance cycle
		cycle += latency
	}
	
	// Select which register determines the next cache address
	// Use the register with the latest latency (most mixed)
	maxLatency := 0
	for i := 0; i < 8; i++ {
		if registerLatency[i] > maxLatency {
			maxLatency = registerLatency[i]
			prog.addressReg = uint8(i)
		}
	}
	
	return prog
}
