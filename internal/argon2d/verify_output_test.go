package argon2d

import (
"encoding/binary"
"encoding/hex"
"testing"
)

// TestVerifyFinalOutput checks what Argon2dCache actually outputs
func TestVerifyFinalOutput(t *testing.T) {
key := []byte("test key 000")

cache := Argon2dCache(key)

t.Logf("Cache output length: %d bytes", len(cache))
t.Logf("First 64 bytes: %s", hex.EncodeToString(cache[:64]))

// Check as uint64 values
t.Log("\nFirst 8 uint64 values:")
for i := 0; i < 8; i++ {
val := binary.LittleEndian.Uint64(cache[i*8:(i+1)*8])
t.Logf("cache[%d] = 0x%016x", i, val)
}

// Expected values from RandomX reference
expected0 := uint64(0x191e0e1d23c02186)
actual0 := binary.LittleEndian.Uint64(cache[0:8])

t.Logf("\nComparison:")
t.Logf("Expected cache[0] = 0x%016x", expected0)
t.Logf("Actual   cache[0] = 0x%016x", actual0)

if actual0 == expected0 {
t.Log("✓ Match!")
} else {
t.Log("✗ Mismatch")
}
}
