// Mining simulation demonstrating concurrent RandomX hashingpackage mining

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opd-ai/go-randomx"
)

func main() {
	// Command-line flags
	workers := flag.Int("workers", runtime.NumCPU(), "Number of mining workers")
	target := flag.String("target", "00000000", "Target hash prefix (hex)")
	key := flag.String("key", "mining example", "Cache key")

	flag.Parse()

	fmt.Printf("RandomX Mining Simulation\n")
	fmt.Printf("Workers: %d\n", *workers)
	fmt.Printf("Target prefix: %s\n", *target)
	fmt.Printf("\n")

	// Create hasher
	fmt.Printf("Initializing hasher...\n")
	config := randomx.Config{
		Mode:     randomx.LightMode, // Use light mode for this example
		CacheKey: []byte(*key),
	}

	hasher, err := randomx.New(config)
	if err != nil {
		log.Fatalf("Failed to create hasher: %v", err)
	}
	defer hasher.Close()

	// Start mining
	fmt.Printf("Starting mining...\n\n")
	startTime := time.Now()

	var (
		hashCount uint64
		found     bool
		foundMu   sync.Mutex
	)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			nonce := uint64(workerID)
			input := make([]byte, 8)

			for {
				// Check if solution found
				foundMu.Lock()
				if found {
					foundMu.Unlock()
					return
				}
				foundMu.Unlock()

				// Create input with nonce
				binary.LittleEndian.PutUint64(input, nonce)

				// Compute hash
				hash := hasher.Hash(input)
				atomic.AddUint64(&hashCount, 1)

				// Check if hash meets target (simplified: check first byte)
				if hash[0] == 0x00 && hash[1] == 0x00 {
					foundMu.Lock()
					if !found {
						found = true
						duration := time.Since(startTime)
						hashes := atomic.LoadUint64(&hashCount)
						hashrate := float64(hashes) / duration.Seconds()

						fmt.Printf("✓ Solution found by worker %d!\n", workerID)
						fmt.Printf("  Nonce: %d\n", nonce)
						fmt.Printf("  Hash: %x\n", hash)
						fmt.Printf("  Time: %v\n", duration)
						fmt.Printf("  Total hashes: %d\n", hashes)
						fmt.Printf("  Hashrate: %.2f H/s\n", hashrate)
					}
					foundMu.Unlock()
					return
				}

				nonce += uint64(*workers)
			}
		}(i)
	}

	// Progress reporter
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			foundMu.Lock()
			if found {
				foundMu.Unlock()
				return
			}
			foundMu.Unlock()

			hashes := atomic.LoadUint64(&hashCount)
			duration := time.Since(startTime)
			hashrate := float64(hashes) / duration.Seconds()

			fmt.Printf("Mining... %d hashes (%.2f H/s)\n", hashes, hashrate)
		}
	}()

	// Wait for workers
	wg.Wait()
	ticker.Stop()
}
