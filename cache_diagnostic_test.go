package randomx

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
)

// TestCacheReferenceValues validates cache generation against RandomX C++ reference.
// From RandomX/src/tests/tests.cpp cache initialization test.
func TestCacheReferenceValues(t *testing.T) {
	// Create cache with reference key
	cache, err := newCache([]byte("test key 000"))
	if err != nil {
		t.Fatalf("newCache() error = %v", err)
	}
	defer cache.release()

	// Reference values from RandomX C++ tests
	// These are specific uint64 values at known offsets in the cache
	tests := []struct {
		name     string
		offset   int      // Byte offset (index * 8 for uint64)
		expected uint64   // Expected value from C++ reference
	}{
		{
			name:     "cache[0]",
			offset:   0,
			expected: 0x191e0e1d23c02186,
		},
		{
			name:     "cache[1568413]",
			offset:   1568413 * 8,
			expected: 0xf1b62fe6210bf8b1,
		},
		{
			name:     "cache[33554431]",
			offset:   33554431 * 8,
			expected: 0x1f47f056d05cd99b,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.offset+8 > len(cache.data) {
				t.Fatalf("offset %d out of bounds (cache size: %d)", tt.offset, len(cache.data))
			}

			actual := binary.LittleEndian.Uint64(cache.data[tt.offset : tt.offset+8])

			if actual != tt.expected {
				t.Errorf("Cache value mismatch at offset %d:", tt.offset)
				t.Errorf("  Got:      0x%016x", actual)
				t.Errorf("  Expected: 0x%016x", tt.expected)
			}
		})
	}
}

// TestCacheFirstBytes shows the first few bytes of cache for debugging.
func TestCacheFirstBytes(t *testing.T) {
	cache, err := newCache([]byte("test key 000"))
	if err != nil {
		t.Fatalf("newCache() error = %v", err)
	}
	defer cache.release()

	// Show first 64 bytes
	t.Logf("First 64 bytes of cache:")
	t.Logf("  Hex: %s", hex.EncodeToString(cache.data[:64]))

	// Show as uint64 values (little-endian)
	t.Logf("  As uint64 values:")
	for i := 0; i < 8; i++ {
		offset := i * 8
		val := binary.LittleEndian.Uint64(cache.data[offset : offset+8])
		t.Logf("    cache[%d] = 0x%016x", i, val)
	}
}
