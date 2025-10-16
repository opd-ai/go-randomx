package argon2d

import (
	"testing"
)

// TestFillBlock_WithFBlaMka tests fillBlock with the fBlaMka G function.
func TestFillBlock_WithFBlaMka(t *testing.T) {
	var prev, ref, next Block

	// Case 1: Different blocks
	for i := range prev {
		prev[i] = uint64(i + 1)
		ref[i] = uint64((i + 1) * 2)
	}

	t.Logf("=== Case 1: Different blocks ===")
	t.Logf("prev[0] = 0x%016x", prev[0])
	t.Logf("ref[0] = 0x%016x", ref[0])

	fillBlock(&prev, &ref, &next, false)

	t.Logf("next[0] = 0x%016x", next[0])

	allZero := true
	for i := range next {
		if next[i] != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		t.Error("fillBlock produced all zeros with different blocks")
	} else {
		t.Log("✓ Different blocks: non-zero output")
	}

	// Case 2: Same blocks (self-reference)
	// When prev and ref have identical content, their XOR produces all zeros.
	// The fBlaMka compression function correctly outputs all zeros from all-zero input.
	// This is expected behavior per Argon2 spec - fBlaMka(0,0) = 0.
	// In practice, the indexing algorithm prevents this case from occurring.
	for i := range prev {
		prev[i] = uint64(i + 1)
	}
	ref = prev
	next = Block{} // Reset

	t.Logf("\n=== Case 2: Same blocks (self-reference) ===")
	t.Logf("prev[0] = 0x%016x", prev[0])
	t.Logf("ref[0] = 0x%016x", ref[0])

	fillBlock(&prev, &ref, &next, false)

	t.Logf("next[0] = 0x%016x", next[0])

	allZero = true
	for i := range next {
		if next[i] != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		t.Log("✓ Self-reference: all-zero output (correct - XOR of identical blocks = 0)")
	} else {
		t.Error("fillBlock produced non-zero output with self-reference (unexpected - should be all zeros)")
	}

	// Case 3: All-zero input blocks
	prev = Block{}
	ref = Block{}
	next = Block{}

	t.Logf("\n=== Case 3: All-zero input blocks ===")
	t.Logf("prev[0] = 0x%016x", prev[0])
	t.Logf("ref[0] = 0x%016x", ref[0])

	fillBlock(&prev, &ref, &next, false)

	t.Logf("next[0] = 0x%016x", next[0])

	allZero = true
	for i := range next {
		if next[i] != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		t.Log("⚠️  All-zero inputs produce all-zero output (expected behavior)")
	} else {
		t.Log("✓ All-zero inputs: non-zero output")
	}
}
