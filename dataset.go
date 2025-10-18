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
// This implements the RandomX initDatasetItem function from the C++ reference.
func (ds *dataset) generateItem(c *cache, itemNumber uint64, output []byte) {
	// Superscalar constants (from RandomX C++ reference)
	const (
		superscalarMul0 = 6364136223846793005
		superscalarAdd1 = 9298411001130361340
		superscalarAdd2 = 12065312585734608966
		superscalarAdd3 = 9306329213124626780
		superscalarAdd4 = 5281919268842080866
		superscalarAdd5 = 10536153434571861004
		superscalarAdd6 = 3398623926847679864
		superscalarAdd7 = 9549104520008361294
	)
	
	// Initialize register file with specific constants based on item number
	var registers [8]uint64
	registerValue := itemNumber
	registers[0] = (itemNumber + 1) * superscalarMul0
	registers[1] = registers[0] ^ superscalarAdd1
	registers[2] = registers[0] ^ superscalarAdd2
	registers[3] = registers[0] ^ superscalarAdd3
	registers[4] = registers[0] ^ superscalarAdd4
	registers[5] = registers[0] ^ superscalarAdd5
	registers[6] = registers[0] ^ superscalarAdd6
	registers[7] = registers[0] ^ superscalarAdd7
	
	// Execute 8 superscalar programs (one per cache access)
	for i := 0; i < cacheAccesses; i++ {
		// Get cache block based on current register value
		// Mask to cache line size (64 bytes per item)
		const mask = cacheItems - 1
		cacheIndex := uint32(registerValue & mask)
		mixBlock := c.getItem(cacheIndex)
		
		// Execute the superscalar program on the register file
		prog := c.programs[i]
		executeSuperscalar(&registers, prog, c.reciprocals)
		
		// XOR cache block into registers
		for r := 0; r < 8; r++ {
			val := binary.LittleEndian.Uint64(mixBlock[r*8 : r*8+8])
			registers[r] ^= val
		}
		
		// Next cache address is determined by the address register
		registerValue = registers[prog.addressReg]
	}
	
	// Output is the final register state (64 bytes)
	for r := 0; r < 8; r++ {
		binary.LittleEndian.PutUint64(output[r*8:r*8+8], registers[r])
	}
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
