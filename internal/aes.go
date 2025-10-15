package internal

import (
	"crypto/aes"
	"crypto/cipher"
)

// AESEncryptor provides AES encryption for RandomX operations.
type AESEncryptor struct {
	block cipher.Block
}

// NewAESEncryptor creates a new AES encryptor with the given key.
// Key must be 16, 24, or 32 bytes (AES-128, AES-192, or AES-256).
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &AESEncryptor{block: block}, nil
}

// Encrypt encrypts a single 16-byte block.
// dst and src must be exactly 16 bytes.
func (a *AESEncryptor) Encrypt(dst, src []byte) {
	a.block.Encrypt(dst, src)
}

// Decrypt decrypts a single 16-byte block.
// dst and src must be exactly 16 bytes.
func (a *AESEncryptor) Decrypt(dst, src []byte) {
	a.block.Decrypt(dst, src)
}

// EncryptBlocks encrypts multiple blocks using AES.
// Both dst and src must be multiples of 16 bytes.
func (a *AESEncryptor) EncryptBlocks(dst, src []byte) {
	if len(src)%aes.BlockSize != 0 {
		panic("aes: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("aes: output buffer too small")
	}

	for i := 0; i < len(src); i += aes.BlockSize {
		a.block.Encrypt(dst[i:i+aes.BlockSize], src[i:i+aes.BlockSize])
	}
}

// AES1R performs 1-round AES encryption (used in RandomX).
// This is a simplified AES used in dataset generation.
func AES1R(dst, src, key []byte) error {
	enc, err := NewAESEncryptor(key)
	if err != nil {
		return err
	}
	enc.Encrypt(dst, src)
	return nil
}

// AES4R performs 4-round AES encryption (used in RandomX).
// This implements the 4-round AES used in RandomX VM operations.
func AES4R(state, key1, key2, key3, key4 []byte) {
	enc1, _ := NewAESEncryptor(key1)
	enc2, _ := NewAESEncryptor(key2)
	enc3, _ := NewAESEncryptor(key3)
	enc4, _ := NewAESEncryptor(key4)

	temp := make([]byte, 16)

	enc1.Encrypt(temp, state)
	enc2.Encrypt(state, temp)
	enc3.Encrypt(temp, state)
	enc4.Encrypt(state, temp)
}
