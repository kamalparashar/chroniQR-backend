package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

var encryptionKey []byte

// InitCrypto initializes the AES-GCM encryption key.
func InitCrypto(key string) {
	hash := sha256.Sum256([]byte(key))
	encryptionKey = hash[:]
}

// Encrypt encrypts plaintext using AES-GCM and base64 encodes the result.
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if len(encryptionKey) == 0 {
		return plaintext, nil
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64 encoded AES-GCM ciphertext.
// If the input is not base64 or decryption fails, it returns the original string
// (useful for migrating legacy unencrypted data).
func Decrypt(encodedCiphertext string) (string, error) {
	if encodedCiphertext == "" {
		return "", nil
	}
	if len(encryptionKey) == 0 {
		return encodedCiphertext, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return encodedCiphertext, nil // Fallback for unencrypted data
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return encodedCiphertext, nil // Fallback
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return encodedCiphertext, nil // Fallback
	}

	return string(plaintext), nil
}
