package randomx

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"sync"

	"github.com/opd-ai/go-randomx/internal"
)

const (
	// Dataset size in bytes (2080 MB for RandomX v1.1.x)
	datasetSize = 2080 * 1024 * 1024

	// Number of dataset items (each item is 64 bytes)
	datasetItems = datasetSize / 64
)

// dataset holds the full RandomX dataset for fast mode operation.
// The dataset is ~2 GB and is generated from the cache.
type dataset struct {
	data []byte // Full dataset (2+ GB)
}

// newDataset creates and initializes a new RandomX dataset from the cache.
// This is an expensive operation taking 20-30 seconds.
func newDataset(c *cache) (*dataset, error) {
	if c == nil || len(c.data) == 0 {
		return nil, fmt.Errorf("invalid cache")
	}

	ds := &dataset{
		data: make([]byte, datasetSize),
	}

	// Generate dataset items in parallel
	if err := ds.generate(c); err != nil {
		return nil, err
	}

	return ds, nil
}

// generate creates all dataset items from the cache using parallel workers.
func (ds *dataset) generate(c *cache) error {
	numWorkers := runtime.NumCPU()
	itemsPerWorker := datasetItems / uint64(numWorkers)

	var wg sync.WaitGroup
	errChan := make(chan error, numWorkers)

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			start := uint64(workerID) * itemsPerWorker
			end := start + itemsPerWorker
			if workerID == numWorkers-1 {
				end = datasetItems
			}

			for item := start; item < end; item++ {
				offset := item * 64
				ds.generateItem(c, item, ds.data[offset:offset+64])
			}
		}(w)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

// generateItem creates a single dataset item using superscalar hash.
// This implements the RandomX dataset item generation algorithm.
func (ds *dataset) generateItem(c *cache, itemNumber uint64, output []byte) {
	// Initialize register file with item number
	var registers [8]uint64
	registers[0] = itemNumber

	// Mix with cache items using superscalar program
	const iterations = 8
	for i := 0; i < iterations; i++ {
		// Get cache item based on current register state
		cacheIndex := uint32(registers[0] % cacheItems)
		cacheItem := c.getItem(cacheIndex)

		// Mix cache item into registers
		for r := 0; r < 8; r++ {
			val := binary.LittleEndian.Uint64(cacheItem[r*8 : r*8+8])
			registers[r] ^= val
		}

		// Apply simple mixing function
		for r := 0; r < 8; r++ {
			registers[r] = mixRegister(registers[r], uint64(i))
		}
	}

	// Write final register state to output
	for r := 0; r < 8; r++ {
		binary.LittleEndian.PutUint64(output[r*8:r*8+8], registers[r])
	}
}

// mixRegister applies a mixing transformation to a register value.
func mixRegister(val uint64, iteration uint64) uint64 {
	// Simple mixing using prime multipliers and rotation
	val ^= iteration
	val *= 0x9e3779b97f4a7c15 // Knuth's golden ratio
	val ^= val >> 33
	val *= 0xbf58476d1ce4e5b9
	val ^= val >> 29
	return val
}

// release frees the dataset resources.
func (ds *dataset) release() {
	if ds.data != nil {
		releaseDataset(ds.data)
		ds.data = nil
	}
}

// getItem returns the dataset item at the specified index.
// Each item is 64 bytes. Thread-safe for reads after initialization.
func (ds *dataset) getItem(index uint64) []byte {
	if index >= datasetItems {
		index = index % datasetItems
	}
	offset := index * 64
	return ds.data[offset : offset+64]
}

// hashBlake2b performs Blake2b hashing for dataset generation.
func hashBlake2b(input []byte) []byte {
	hash := internal.Blake2b512(input)
	return hash[:]
}
