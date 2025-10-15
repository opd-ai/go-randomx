// Simple RandomX hasher demonstration
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/opd-ai/go-randomx"
)

func main() {
	// Command-line flags
	mode := flag.String("mode", "light", "Operating mode: light or fast")
	key := flag.String("key", "RandomX example key", "Cache key (seed)")
	input := flag.String("input", "Hello, RandomX!", "Input data to hash")
	bench := flag.Bool("bench", false, "Run benchmark (1000 hashes)")

	flag.Parse()

	// Parse mode
	var rxMode randomx.Mode
	switch *mode {
	case "light":
		rxMode = randomx.LightMode
	case "fast":
		rxMode = randomx.FastMode
	default:
		log.Fatalf("Invalid mode: %s (use 'light' or 'fast')", *mode)
	}

	// Create hasher
	fmt.Printf("Creating RandomX hasher in %s mode...\n", *mode)
	start := time.Now()

	config := randomx.Config{
		Mode:     rxMode,
		CacheKey: []byte(*key),
	}

	hasher, err := randomx.New(config)
	if err != nil {
		log.Fatalf("Failed to create hasher: %v", err)
	}
	defer hasher.Close()

	initTime := time.Since(start)
	fmt.Printf("Initialization took: %v\n", initTime)

	if *bench {
		runBenchmark(hasher, []byte(*input))
	} else {
		runSingleHash(hasher, []byte(*input))
	}
}

func runSingleHash(hasher *randomx.Hasher, input []byte) {
	fmt.Printf("\nHashing input: %q\n", string(input))

	start := time.Now()
	hash := hasher.Hash(input)
	duration := time.Since(start)

	fmt.Printf("Hash: %s\n", hex.EncodeToString(hash[:]))
	fmt.Printf("Time: %v\n", duration)
}

func runBenchmark(hasher *randomx.Hasher, input []byte) {
	const numHashes = 1000

	fmt.Printf("\nBenchmarking: computing %d hashes...\n", numHashes)

	start := time.Now()
	for i := 0; i < numHashes; i++ {
		_ = hasher.Hash(input)
	}
	duration := time.Since(start)

	hashesPerSecond := float64(numHashes) / duration.Seconds()
	avgTime := duration / numHashes

	fmt.Printf("Total time: %v\n", duration)
	fmt.Printf("Average time per hash: %v\n", avgTime)
	fmt.Printf("Hashes per second: %.2f\n", hashesPerSecond)
}
