package argon2d

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
)

// TestArgon2dCache_RandomXReference tests against known RandomX cache output.
// The RandomX reference implementation generates cache with "test key 000".
// The first uint64 at cache[0] should be 0x191e0e1d23c02186.
func TestArgon2dCache_RandomXReference(t *testing.T) {
	key := []byte("test key 000")

	cache := Argon2dCache(key)

	if len(cache) != 262144 {
		t.Fatalf("Cache size = %d, expected 262144", len(cache))
	}

	// Check first uint64
	actual := binary.LittleEndian.Uint64(cache[0:8])
	expected := uint64(0x191e0e1d23c02186)

	t.Logf("Cache[0] = 0x%016x (expected 0x%016x)", actual, expected)
	t.Logf("First 64 bytes: %s", hex.EncodeToString(cache[:64]))

	if actual != expected {
		t.Errorf("Cache[0] mismatch: got 0x%016x, want 0x%016x", actual, expected)
	}
}

// TestArgon2dParameters logs the exact parameters being used.
func TestArgon2dParameters(t *testing.T) {
	key := []byte("test key 000")

	t.Logf("Argon2d parameters for RandomX:")
	t.Logf("  Key (password): %q", key)
	t.Logf("  Salt: %q (same as key)", key)
	t.Logf("  Time cost: 3")
	t.Logf("  Memory: 262144 KB (256 MB)")
	t.Logf("  Lanes: 1")
	t.Logf("  Output: 262144 bytes (256 KB)")

	cache := Argon2dCache(key)
	t.Logf("  Generated cache size: %d bytes", len(cache))
	t.Logf("  First 32 bytes: %s", hex.EncodeToString(cache[:32]))
}
