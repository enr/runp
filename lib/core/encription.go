package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	mr "math/rand"
	"strconv"
)

func toKey32(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// RandomKey generates a random string usable as key to encrypt secrets
func RandomKey() string {
	i := mr.Int()
	hasher := md5.New()
	hasher.Write([]byte(strconv.Itoa(i)))
	return hex.EncodeToString(hasher.Sum(nil))[:12]
}

// Encrypt a secret using passphrase
func Encrypt(data []byte, passphrase string) ([]byte, error) {
	block, _ := aes.NewCipher([]byte(toKey32(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return []byte{}, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// EncryptToBase64 encypt secret and encode in base 64
func EncryptToBase64(data []byte, passphrase string) (string, error) {
	encrypted, err := Encrypt(data, passphrase)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt a secret using passphrase
func Decrypt(data []byte, passphrase string) ([]byte, error) {
	key := []byte(toKey32(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte{}, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return []byte{}, err
	}
	return plaintext, nil
}

// DecryptBase64 decrypt a secret encoded in base 64
func DecryptBase64(encoded string, passphrase string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		ui.WriteLinef("Failed to decode base64 encrypted secret: %v", err)
		return []byte{}, err
	}
	return Decrypt(decoded, passphrase)
}
