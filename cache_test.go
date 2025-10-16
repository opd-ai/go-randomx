package randomx

import (
	"testing"

	"github.com/opd-ai/go-randomx/internal"
)

func TestCacheCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache creation test in short mode")
	}

	seed := []byte("test seed")
	cache, err := newCache(seed)
	if err != nil {
		t.Fatalf("newCache() error = %v", err)
	}
	defer cache.release()

	if len(cache.data) != cacheSize {
		t.Errorf("cache size = %d, want %d", len(cache.data), cacheSize)
	}

	if !bytesEqual(cache.key, seed) {
		t.Error("cache key should match seed")
	}
}

func TestCacheEmptySeed(t *testing.T) {
	_, err := newCache([]byte{})
	if err == nil {
		t.Error("newCache() with empty seed should fail")
	}
}

func TestCacheGetItem(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	cache, err := newCache([]byte("test"))
	if err != nil {
		t.Fatalf("newCache() error = %v", err)
	}
	defer cache.release()

	// Test getting valid items
	for i := uint32(0); i < 10; i++ {
		item := cache.getItem(i)
		if len(item) != 64 {
			t.Errorf("item length = %d, want 64", len(item))
		}
	}

	// Test boundary
	item := cache.getItem(cacheItems - 1)
	if len(item) != 64 {
		t.Errorf("boundary item length = %d, want 64", len(item))
	}

	// Test wrapping
	item = cache.getItem(cacheItems + 5)
	expectedItem := cache.getItem(5)
	if !bytesEqual(item, expectedItem) {
		t.Error("item wrapping not working correctly")
	}
}

func TestCacheRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	cache, err := newCache([]byte("test"))
	if err != nil {
		t.Fatalf("newCache() error = %v", err)
	}

	cache.release()

	if cache.data != nil {
		t.Error("cache data should be nil after release")
	}

	// Calling release again should be safe
	cache.release()
}

func TestCacheDeterminism(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	seed := []byte("determinism test")

	cache1, err := newCache(seed)
	if err != nil {
		t.Fatalf("newCache() error = %v", err)
	}
	defer cache1.release()

	cache2, err := newCache(seed)
	if err != nil {
		t.Fatalf("newCache() error = %v", err)
	}
	defer cache2.release()

	if !bytesEqual(cache1.data, cache2.data) {
		t.Error("cache generation should be deterministic")
	}
}

func TestCacheDifferentSeeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	cache1, err := newCache([]byte("seed1"))
	if err != nil {
		t.Fatalf("newCache() error = %v", err)
	}
	defer cache1.release()

	cache2, err := newCache([]byte("seed2"))
	if err != nil {
		t.Fatalf("newCache() error = %v", err)
	}
	defer cache2.release()

	if bytesEqual(cache1.data, cache2.data) {
		t.Error("different seeds should produce different caches")
	}
}

// Benchmark cache creation
func BenchmarkCacheCreation(b *testing.B) {
	seed := []byte("benchmark seed")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache, err := newCache(seed)
		if err != nil {
			b.Fatalf("newCache() error = %v", err)
		}
		cache.release()
	}
}

// Benchmark cache item access
func BenchmarkCacheGetItem(b *testing.B) {
	cache, err := newCache([]byte("benchmark"))
	if err != nil {
		b.Fatalf("newCache() error = %v", err)
	}
	defer cache.release()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.getItem(uint32(i % cacheItems))
	}
}

// Test internal Argon2 cache generation
func TestArgon2CacheGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow Argon2 test in short mode")
	}

	seed := []byte("argon2 test")
	result := internal.Argon2dCache(seed)

	if len(result) != cacheSize {
		t.Errorf("Argon2dCache output size = %d, want %d", len(result), cacheSize)
	}

	// Test determinism
	result2 := internal.Argon2dCache(seed)
	if !bytesEqual(result, result2) {
		t.Error("Argon2dCache should be deterministic")
	}
}
