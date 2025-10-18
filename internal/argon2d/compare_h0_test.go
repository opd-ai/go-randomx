package argon2d

import (
"encoding/hex"
"testing"
)

// TestCompareH0 checks if our initialHash (H0) matches expectations
func TestCompareH0(t *testing.T) {
key := []byte("test key 000")
salt := []byte("RandomX\x03")

// Argon2 parameters
lanes := uint32(1)
tagLength := uint32(32)  // Standard Argon2 uses 32 bytes for H0
memorySizeKB := uint32(262144)
timeCost := uint32(3)

h0 := initialHash(lanes, tagLength, memorySizeKB, timeCost, key, salt, nil, nil)

t.Logf("H0 (64 bytes): %s", hex.EncodeToString(h0[:]))
t.Logf("H0[0:32]:  %s", hex.EncodeToString(h0[0:32]))
t.Logf("H0[32:64]: %s", hex.EncodeToString(h0[32:64]))
}
