package randomx

import (
	"encoding/hex"
	"fmt"
	"os"
)

// debugEnabled controls whether debug tracing is enabled via RANDOMX_DEBUG env var
var debugEnabled = os.Getenv("RANDOMX_DEBUG") == "1"

// traceLog outputs a debug message if tracing is enabled
func traceLog(format string, args ...interface{}) {
	if debugEnabled {
		fmt.Printf("[TRACE] "+format+"\n", args...)
	}
}

// traceBytes outputs bytes in hex format with a descriptive name
func traceBytes(name string, data []byte) {
	if debugEnabled {
		fmt.Printf("[TRACE] %s (%d bytes): %s\n", name, len(data), hex.EncodeToString(data))
	}
}

// traceRegisters outputs the full integer register state
func traceRegisters(name string, regs [8]uint64) {
	if debugEnabled {
		fmt.Printf("[TRACE] %s:\n", name)
		for i, r := range regs {
			fmt.Printf("[TRACE]   r%d = 0x%016x (%d)\n", i, r, r)
		}
	}
}

// traceFRegisters outputs the full floating-point register state
func traceFRegisters(name string, regs [4]float64) {
	if debugEnabled {
		fmt.Printf("[TRACE] %s:\n", name)
		for i, r := range regs {
			fmt.Printf("[TRACE]   f%d = %e\n", i, r)
		}
	}
}

// traceUint64 outputs a single uint64 value
func traceUint64(name string, value uint64) {
	if debugEnabled {
		fmt.Printf("[TRACE] %s = 0x%016x (%d)\n", name, value, value)
	}
}

// compareTrace compares expected vs actual hex strings
// Returns true if they match, false otherwise
// Automatically logs the comparison result when debug is enabled
func compareTrace(stage, expected, actual string) bool {
	match := expected == actual
	if debugEnabled {
		status := "✓"
		if !match {
			status := "✗ MISMATCH"
			fmt.Printf("[TRACE] %s %s:\n", status, stage)
			fmt.Printf("[TRACE]   Expected: %s\n", expected)
			fmt.Printf("[TRACE]   Actual:   %s\n", actual)
		} else {
			fmt.Printf("[TRACE] %s %s matches\n", status, stage)
		}
	}
	return match
}

// traceSeparator prints a visual separator in debug output
func traceSeparator(title string) {
	if debugEnabled {
		fmt.Printf("[TRACE] ========== %s ==========\n", title)
	}
}

// traceSubsection prints a subsection header
func traceSubsection(title string) {
	if debugEnabled {
		fmt.Printf("[TRACE] --- %s ---\n", title)
	}
}
