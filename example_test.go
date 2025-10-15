package randomx

import (
	"fmt"
	"testing"
)

// Example of basic usage
func ExampleNew() {
	config := Config{
		Mode:     FastMode,
		CacheKey: []byte("example key"),
	}

	hasher, err := New(config)
	if err != nil {
		panic(err)
	}
	defer hasher.Close()

	hash := hasher.Hash([]byte("Hello, RandomX!"))
	fmt.Printf("Hash length: %d bytes\n", len(hash))
	// Output: Hash length: 32 bytes
}

// Example of light mode usage
func ExampleNew_lightMode() {
	config := Config{
		Mode:     LightMode,
		CacheKey: []byte("light mode key"),
	}

	hasher, err := New(config)
	if err != nil {
		panic(err)
	}
	defer hasher.Close()

	hash := hasher.Hash([]byte("test input"))
	fmt.Printf("Hash length: %d bytes\n", len(hash))
	// Output: Hash length: 32 bytes
}

// Example of updating cache key
func ExampleHasher_UpdateCacheKey() {
	hasher, err := New(Config{
		Mode:     LightMode,
		CacheKey: []byte("initial key"),
	})
	if err != nil {
		panic(err)
	}
	defer hasher.Close()

	// Hash with initial key
	hash1 := hasher.Hash([]byte("data"))

	// Update to new key (e.g., new blockchain epoch)
	err = hasher.UpdateCacheKey([]byte("new epoch key"))
	if err != nil {
		panic(err)
	}

	// Hash with new key
	hash2 := hasher.Hash([]byte("data"))

	fmt.Printf("Hashes are different: %v\n", hash1 != hash2)
	// Output: Hashes are different: true
}

// Example of concurrent hashing
func ExampleHasher_Hash_concurrent() {
	hasher, err := New(Config{
		Mode:     LightMode,
		CacheKey: []byte("concurrent key"),
	})
	if err != nil {
		panic(err)
	}
	defer hasher.Close()

	// Spawn multiple goroutines
	done := make(chan bool, 4)
	for i := 0; i < 4; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				input := []byte(fmt.Sprintf("worker:%d:nonce:%d", id, j))
				_ = hasher.Hash(input)
			}
			done <- true
		}(i)
	}

	// Wait for all workers
	for i := 0; i < 4; i++ {
		<-done
	}

	fmt.Println("Concurrent hashing completed")
	// Output: Concurrent hashing completed
}

// Example showing mode selection
func ExampleMode() {
	modes := []Mode{LightMode, FastMode}

	for _, mode := range modes {
		fmt.Printf("%s\n", mode)
	}
	// Output:
	// LightMode
	// FastMode
}

// Benchmark example
func BenchmarkHasher_Hash(b *testing.B) {
	hasher, err := New(Config{
		Mode:     LightMode,
		CacheKey: []byte("benchmark key"),
	})
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	defer hasher.Close()

	input := []byte("benchmark input data")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = hasher.Hash(input)
	}
}

// Benchmark parallel hashing
func BenchmarkHasher_Hash_Parallel(b *testing.B) {
	hasher, err := New(Config{
		Mode:     LightMode,
		CacheKey: []byte("parallel benchmark"),
	})
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	defer hasher.Close()

	input := []byte("parallel benchmark data")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = hasher.Hash(input)
		}
	})
}
