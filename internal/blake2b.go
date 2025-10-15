// Package internal provides cryptographic primitives for RandomX.
// This package wraps golang.org/x/crypto and crypto/* packages.
package internal

import (
	"hash"

	"golang.org/x/crypto/blake2b"
)

// Blake2bConfig specifies Blake2b hashing configuration.
type Blake2bConfig struct {
	OutputSize int    // Hash output size in bytes
	Key        []byte // Optional key for keyed hashing
}

// Blake2bHash computes a Blake2b hash with the specified configuration.
func Blake2bHash(data []byte, config Blake2bConfig) ([]byte, error) {
	var hasher hash.Hash
	var err error

	if len(config.Key) > 0 {
		hasher, err = blake2b.New(config.OutputSize, config.Key)
	} else {
		hasher, err = blake2b.New(config.OutputSize, nil)
	}

	if err != nil {
		return nil, err
	}

	hasher.Write(data)
	return hasher.Sum(nil), nil
}

// Blake2b256 computes a 256-bit Blake2b hash (32 bytes).
func Blake2b256(data []byte) [32]byte {
	h := blake2b.Sum256(data)
	return h
}

// Blake2b512 computes a 512-bit Blake2b hash (64 bytes).
func Blake2b512(data []byte) [64]byte {
	h := blake2b.Sum512(data)
	return h
}

// Blake2bStream provides streaming Blake2b hashing.
type Blake2bStream struct {
	hasher hash.Hash
}

// NewBlake2bStream creates a new streaming Blake2b hasher.
func NewBlake2bStream(size int, key []byte) (*Blake2bStream, error) {
	hasher, err := blake2b.New(size, key)
	if err != nil {
		return nil, err
	}
	return &Blake2bStream{hasher: hasher}, nil
}

// Write adds data to the hash.
func (b *Blake2bStream) Write(data []byte) (int, error) {
	return b.hasher.Write(data)
}

// Sum returns the current hash value.
func (b *Blake2bStream) Sum() []byte {
	return b.hasher.Sum(nil)
}

// Reset resets the hasher to initial state.
func (b *Blake2bStream) Reset() {
	b.hasher.Reset()
}
