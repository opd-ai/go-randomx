package argon2d

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
)

// TestArgon2d_DetailedLogging shows all intermediate values for debugging.
func TestArgon2d_DetailedLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Argon2d debug test in short mode")
	}

	password := []byte("test key 000")
	salt := []byte("RandomX\x03")

	t.Logf("=== Argon2d Detailed Logging ===")
	t.Logf("Password: %q (%d bytes)", password, len(password))
	t.Logf("Salt: %q (%d bytes)", salt, len(salt))
	t.Logf("Parameters: time=3, memory=262144 KB, lanes=1, output=262144 bytes")

	// Step 1: Generate H0
	h0 := initialHash(1, 262144, 262144, 3, password, salt, nil, nil)
	t.Logf("\nStep 1: Initial Hash (H0)")
	t.Logf("H0 (64 bytes): %s", hex.EncodeToString(h0[:]))
	t.Logf("H0[0:8] as uint64: 0x%016x", binary.LittleEndian.Uint64(h0[0:8]))

	// Step 2: Initialize memory (just first 2 blocks for logging)
	memory := make([]Block, 4) // Just 4 blocks for quick test
	initializeMemory(memory, 1, h0)

	t.Logf("\nStep 2: Initialize Memory")
	t.Logf("Block 0[0:8]: %s", hex.EncodeToString(memory[0].ToBytes()[0:8]))
	t.Logf("Block 0[0] as uint64: 0x%016x", memory[0][0])
	t.Logf("Block 1[0:8]: %s", hex.EncodeToString(memory[1].ToBytes()[0:8]))
	t.Logf("Block 1[0] as uint64: 0x%016x", memory[1][0])

	// Step 3: Fill memory with 1 pass on small memory
	fillMemory(memory, 1, 1)

	t.Logf("\nStep 3: After fillMemory (1 pass)")
	for i := 0; i < 4; i++ {
		t.Logf("Block %d[0] as uint64: 0x%016x", i, memory[i][0])
	}

	// Now test with full parameters
	t.Logf("\n=== Full Argon2d Test ===")
	result := Argon2d(password, salt, 3, 262144, 1, 262144)
	t.Logf("Result length: %d bytes", len(result))
	t.Logf("First 64 bytes: %s", hex.EncodeToString(result[:64]))
	t.Logf("result[0:8] as uint64: 0x%016x", binary.LittleEndian.Uint64(result[0:8]))
	t.Logf("\nExpected result[0:8]: 0x191e0e1d23c02186")

	expected := uint64(0x191e0e1d23c02186)
	actual := binary.LittleEndian.Uint64(result[0:8])
	if actual == expected {
		t.Logf("✅ MATCH!")
	} else {
		t.Logf("❌ MISMATCH")
		t.Logf("Difference: 0x%016x", actual^expected)
	}
}

// TestArgon2d_H0Only tests just the H0 generation step.
func TestArgon2d_H0Only(t *testing.T) {
	password := []byte("test key 000")
	salt := []byte("RandomX\x03")

	h0 := initialHash(1, 262144, 262144, 3, password, salt, nil, nil)

	t.Logf("H0: %s", hex.EncodeToString(h0[:]))
	t.Logf("\nH0 as uint64 values (little-endian):")
	for i := 0; i < 8; i++ {
		offset := i * 8
		val := binary.LittleEndian.Uint64(h0[offset : offset+8])
		t.Logf("  H0[%d] = 0x%016x", i, val)
	}
}

// TestArgon2d_FirstTwoBlocks tests just blocks 0 and 1 initialization.
func TestArgon2d_FirstTwoBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Argon2d debug test in short mode")
	}

	password := []byte("test key 000")
	salt := []byte("RandomX\x03")

	h0 := initialHash(1, 262144, 262144, 3, password, salt, nil, nil)

	memory := make([]Block, 262144)
	initializeMemory(memory, 1, h0)

	t.Logf("Block 0 (first 64 bytes): %s", hex.EncodeToString(memory[0].ToBytes()[:64]))
	t.Logf("Block 0[0] = 0x%016x", memory[0][0])
	t.Logf("Block 0[1] = 0x%016x", memory[0][1])

	t.Logf("\nBlock 1 (first 64 bytes): %s", hex.EncodeToString(memory[1].ToBytes()[:64]))
	t.Logf("Block 1[0] = 0x%016x", memory[1][0])
	t.Logf("Block 1[1] = 0x%016x", memory[1][1])
}
