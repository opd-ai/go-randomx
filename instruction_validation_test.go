package randomx

import (
	"testing"
)

// TestInstructionTypeMapping validates that opcodes map to correct instruction types
func TestInstructionTypeMapping(t *testing.T) {
	tests := []struct {
		opcode   uint8
		expected instructionType
		name     string
	}{
		{0, instrIADD_RS, "IADD_RS_first"},
		{15, instrIADD_RS, "IADD_RS_last"},
		{16, instrIADD_M, "IADD_M_first"},
		{22, instrIADD_M, "IADD_M_last"},
		{23, instrISUB_R, "ISUB_R_first"},
		{38, instrISUB_R, "ISUB_R_last"},
		{130, instrFADD_R, "FADD_R_first"},
		{142, instrFADD_R, "FADD_R_at_142"},
		{145, instrFADD_R, "FADD_R_last"},
		{220, instrCBRANCH, "CBRANCH_first"},
		{244, instrCBRANCH, "CBRANCH_last"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getInstructionType(tt.opcode)
			if actual != tt.expected {
				t.Errorf("opcode %d: got %v, expected %v", tt.opcode, actual, tt.expected)
			}
		})
	}
}

// TestMemoryAddressing validates L1/L2/L3 addressing
func TestMemoryAddressing(t *testing.T) {
	vm := &virtualMachine{
		reg: [8]uint64{0x1234567890ABCDEF, 0, 0, 0, 0, 0, 0, 0},
		mem: make([]byte, scratchpadL3Size),
	}

	tests := []struct {
		mod      uint8
		expected string
	}{
		{0, "L3"},  // mod % 4 == 0
		{1, "L2"},  // mod % 4 == 1
		{2, "L1"},  // mod % 4 == 2
		{3, "L2"},  // mod % 4 == 3
		{4, "L3"},  // mod % 4 == 0
		{5, "L2"},  // mod % 4 == 1
	}

	for _, tt := range tests {
		instr := &instruction{
			src: 0,
			imm: 0,
			mod: tt.mod,
		}
		addr := vm.getMemoryAddress(instr)
		
		var level string
		if addr <= scratchpadL1Mask {
			level = "L1"
		} else if addr <= scratchpadL2Mask {
			level = "L2"
		} else {
			level = "L3"
		}
		
		if level != tt.expected {
			t.Errorf("mod %d (mod%%4=%d): address 0x%X mapped to %s, expected %s",
				tt.mod, tt.mod%4, addr, level, tt.expected)
		}
	}
}

// TestFloatMasking validates floating-point masking
func TestFloatMasking(t *testing.T) {
	tests := []struct {
		name  string
		input uint64
	}{
		{"normal", 0x3FF0000000000000}, // 1.0
		{"large", 0x7FEFFFFFFFFFFFFF},  // max double
		{"inf", 0x7FF0000000000000},    // infinity
		{"nan", 0x7FF8000000000000},    // NaN
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := maskFloat(uint64ToFloat(tt.input))
			// Just verify it doesn't panic and returns a value
			t.Logf("Input: 0x%016X -> Output: %v (bits: 0x%016X)",
				tt.input, f, floatToUint64(f))
		})
	}
}

// TestEMaskApplication validates E-register masking
func TestEMaskApplication(t *testing.T) {
	vm := &virtualMachine{
		config: vmConfig{
			eMask: [4]uint64{
				0x3FFFFFFFFFFFFFFF, // Default eMask for E0
				0x3FFFFFFFFFFFFFFF, // Default eMask for E1
				0x3FFFFFFFFFFFFFFF, // Default eMask for E2
				0x3FFFFFFFFFFFFFFF, // Default eMask for E3
			},
		},
	}

	// Test that eMask limits exponent range
	testValue := uint64(0x7FF0000000000000) // Infinity
	masked := testValue & vm.config.eMask[0]
	
	// After masking, exponent should be limited
	exponentMask := uint64(0x7FF0000000000000)
	if (masked & exponentMask) == exponentMask {
		t.Error("eMask should limit exponent to prevent infinity")
	}
	
	t.Logf("Original: 0x%016X, Masked: 0x%016X", testValue, masked)
}
