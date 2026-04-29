package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	kdfSaltSize = 16
	kdfIter     = 100_000
	kdfKeyLen   = 32
)

func deriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, kdfIter, kdfKeyLen, sha256.New)
}

// RandomKey generates a random string usable as key to encrypt secrets
func RandomKey() string {
	b := make([]byte, 6)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// Encrypt a secret using passphrase. Ciphertext format: salt | nonce | ciphertext+tag.
func Encrypt(data []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, kdfSaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(deriveKey(passphrase, salt))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 0, kdfSaltSize+gcm.NonceSize()+len(data)+gcm.Overhead())
	out = append(out, salt...)
	out = append(out, nonce...)
	out = gcm.Seal(out, nonce, data, nil)
	return out, nil
}

// EncryptToBase64 encrypts secret and encodes in base64
func EncryptToBase64(data []byte, passphrase string) (string, error) {
	encrypted, err := Encrypt(data, passphrase)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt a secret using passphrase. Expects ciphertext format: salt | nonce | ciphertext+tag.
func Decrypt(data []byte, passphrase string) ([]byte, error) {
	if len(data) < kdfSaltSize {
		return nil, errors.New("ciphertext too short")
	}
	salt := data[:kdfSaltSize]
	data = data[kdfSaltSize:]

	block, err := aes.NewCipher(deriveKey(passphrase, salt))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// DecryptBase64 decrypts a secret encoded in base64
func DecryptBase64(encoded string, passphrase string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		ui.WriteLinef("Failed to decode base64 encrypted secret: %v", err)
		return []byte{}, err
	}
	return Decrypt(decoded, passphrase)
}
