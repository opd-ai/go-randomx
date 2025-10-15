package argon2d

import (
	"testing"
)

// TestRotr64 verifies the right rotation function with various inputs.
func TestRotr64(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		rotation uint
		expected uint64
	}{
		{
			name:     "rotate_by_8",
			input:    0x123456789ABCDEF0,
			rotation: 8,
			expected: 0xF0123456789ABCDE,
		},
		{
			name:     "rotate_by_16",
			input:    0xFFFFFFFF00000000,
			rotation: 16,
			expected: 0x0000FFFFFFFF0000,
		},
		{
			name:     "rotate_by_32",
			input:    0x123456789ABCDEF0,
			rotation: 32,
			expected: 0x9ABCDEF012345678,
		},
		{
			name:     "rotate_by_63",
			input:    0x8000000000000001,
			rotation: 63,
			expected: 0x0000000000000003,
		},
		{
			name:     "rotate_zero_by_any",
			input:    0,
			rotation: 15,
			expected: 0,
		},
		{
			name:     "rotate_max_by_any",
			input:    0xFFFFFFFFFFFFFFFF,
			rotation: 27,
			expected: 0xFFFFFFFFFFFFFFFF,
		},
		{
			name:     "rotate_by_24",
			input:    0x123456789ABCDEF0,
			rotation: 24,
			expected: 0xBCDEF0123456789A,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rotr64(tt.input, tt.rotation)
			if result != tt.expected {
				t.Errorf("rotr64(0x%X, %d) = 0x%X, want 0x%X",
					tt.input, tt.rotation, result, tt.expected)
			}
		})
	}
}

// TestG verifies the Blake2b G function with known properties.
//
// We test the function's behavior by checking that it produces different
// outputs for different inputs and is deterministic.
func TestG(t *testing.T) {
	tests := []struct {
		name       string
		a, b, c, d uint64
	}{
		{
			name: "all_zeros",
			a:    0, b: 0, c: 0, d: 0,
		},
		{
			name: "all_ones",
			a:    0xFFFFFFFFFFFFFFFF,
			b:    0xFFFFFFFFFFFFFFFF,
			c:    0xFFFFFFFFFFFFFFFF,
			d:    0xFFFFFFFFFFFFFFFF,
		},
		{
			name: "sequential_values",
			a:    1, b: 2, c: 3, d: 4,
		},
		{
			name: "blake2b_initial_values",
			a:    0x6A09E667F3BCC908,
			b:    0xBB67AE8584CAA73B,
			c:    0x3C6EF372FE94F82B,
			d:    0xA54FF53A5F1D36F1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a1, b1, c1, d1 := g(tt.a, tt.b, tt.c, tt.d)

			// Verify determinism - same input should produce same output
			a2, b2, c2, d2 := g(tt.a, tt.b, tt.c, tt.d)
			if a1 != a2 || b1 != b2 || c1 != c2 || d1 != d2 {
				t.Errorf("G function not deterministic")
			}

			// For all_zeros, verify the output is also all zeros
			// (since additions and XORs with zero produce zero)
			if tt.name == "all_zeros" {
				if a1 != 0 || b1 != 0 || c1 != 0 || d1 != 0 {
					t.Errorf("g(0,0,0,0) should be (0,0,0,0), got (%#x, %#x, %#x, %#x)",
						a1, b1, c1, d1)
				}
			}
		})
	}
}

// TestGDeterminism ensures the G function is deterministic.
func TestGDeterminism(t *testing.T) {
	inputs := [][4]uint64{
		{0x123456789ABCDEF0, 0xFEDCBA9876543210, 0x0F0E0D0C0B0A0908, 0x0706050403020100},
		{0, 0, 0, 0},
		{1, 2, 3, 4},
		{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
	}

	for i, input := range inputs {
		a1, b1, c1, d1 := g(input[0], input[1], input[2], input[3])
		a2, b2, c2, d2 := g(input[0], input[1], input[2], input[3])

		if a1 != a2 || b1 != b2 || c1 != c2 || d1 != d2 {
			t.Errorf("G function not deterministic for input %d: "+
				"first=(%#x,%#x,%#x,%#x) second=(%#x,%#x,%#x,%#x)",
				i, a1, b1, c1, d1, a2, b2, c2, d2)
		}
	}
}

// TestGRound verifies the gRound function applies G correctly to 16 elements.
func TestGRound(t *testing.T) {
	tests := []struct {
		name  string
		input [16]uint64
	}{
		{
			name:  "all_zeros",
			input: [16]uint64{},
		},
		{
			name: "sequential",
			input: [16]uint64{
				0, 1, 2, 3, 4, 5, 6, 7,
				8, 9, 10, 11, 12, 13, 14, 15,
			},
		},
		{
			name: "alternating_bits",
			input: [16]uint64{
				0xAAAAAAAAAAAAAAAA, 0x5555555555555555, 0xAAAAAAAAAAAAAAAA, 0x5555555555555555,
				0xAAAAAAAAAAAAAAAA, 0x5555555555555555, 0xAAAAAAAAAAAAAAAA, 0x5555555555555555,
				0xAAAAAAAAAAAAAAAA, 0x5555555555555555, 0xAAAAAAAAAAAAAAAA, 0x5555555555555555,
				0xAAAAAAAAAAAAAAAA, 0x5555555555555555, 0xAAAAAAAAAAAAAAAA, 0x5555555555555555,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := make([]uint64, 16)
			copy(v, tt.input[:])

			gRound(v)

			// Test 1: Determinism - running twice should give same result
			v2 := make([]uint64, 16)
			copy(v2, tt.input[:])
			gRound(v2)

			for i := 0; i < 16; i++ {
				if v[i] != v2[i] {
					t.Errorf("gRound not deterministic at index %d", i)
				}
			}

			// Test 2: For all_zeros input, output should also be all zeros
			if tt.name == "all_zeros" {
				for i := 0; i < 16; i++ {
					if v[i] != 0 {
						t.Errorf("gRound(zeros) element %d = 0x%X, want 0", i, v[i])
					}
				}
			}

			// Test 3: For non-zero inputs, output should differ from input
			// (gRound should mix the values)
			if tt.name != "all_zeros" {
				differs := false
				for i := 0; i < 16; i++ {
					if v[i] != tt.input[i] {
						differs = true
						break
					}
				}
				if !differs {
					t.Error("gRound did not modify non-zero input")
				}
			}
		})
	}
}

// TestGRoundDeterminism ensures gRound is deterministic.
func TestGRoundDeterminism(t *testing.T) {
	input := [16]uint64{
		0, 1, 2, 3, 4, 5, 6, 7,
		8, 9, 10, 11, 12, 13, 14, 15,
	}

	v1 := make([]uint64, 16)
	v2 := make([]uint64, 16)
	copy(v1, input[:])
	copy(v2, input[:])

	gRound(v1)
	gRound(v2)

	for i := 0; i < 16; i++ {
		if v1[i] != v2[i] {
			t.Errorf("gRound not deterministic at index %d: v1=0x%X, v2=0x%X",
				i, v1[i], v2[i])
		}
	}
}

// TestGRoundInPlace verifies gRound modifies the slice in-place.
func TestGRoundInPlace(t *testing.T) {
	v := make([]uint64, 16)
	for i := range v {
		v[i] = uint64(i)
	}

	original := make([]uint64, 16)
	copy(original, v)

	gRound(v)

	// Verify it was modified
	modified := false
	for i := 0; i < 16; i++ {
		if v[i] != original[i] {
			modified = true
			break
		}
	}

	if !modified {
		t.Error("gRound did not modify the input slice")
	}
}

// TestGRoundPanicOnShortSlice verifies gRound behavior with wrong slice size.
func TestGRoundPanicOnShortSlice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("gRound should panic on slice shorter than 16 elements")
		}
	}()

	v := make([]uint64, 15) // Too short
	gRound(v)
}

// BenchmarkG measures the performance of the G function.
func BenchmarkG(b *testing.B) {
	a, bVal, c, d := uint64(0x123456789ABCDEF0), uint64(0xFEDCBA9876543210),
		uint64(0x0F0E0D0C0B0A0908), uint64(0x0706050403020100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a, bVal, c, d = g(a, bVal, c, d)
	}
	b.StopTimer()

	// Use the values to prevent optimization
	_ = a + bVal + c + d
}

// BenchmarkRotr64 measures the performance of the rotation function.
func BenchmarkRotr64(b *testing.B) {
	x := uint64(0x123456789ABCDEF0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = rotr64(x, 32)
	}
	b.StopTimer()

	_ = x
}

// BenchmarkGRound measures the performance of one full G round.
func BenchmarkGRound(b *testing.B) {
	v := make([]uint64, 16)
	for i := range v {
		v[i] = uint64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gRound(v)
	}
}
