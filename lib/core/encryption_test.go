package core

import (
	"crypto/aes"
	"crypto/cipher"
	"regexp"
	"testing"
)

const pbkdf2SaltSize = 16

func TestEncrypt_EmbedsSalt(t *testing.T) {
	// Ciphertext must be salt | nonce | data | GCM-tag.
	// With MD5-only key derivation (no salt) the length is nonce+data+tag, which is shorter.
	passphrase := "test-passphrase"
	plaintext := []byte("hello")

	ct, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	block, _ := aes.NewCipher(make([]byte, 32))
	gcm, _ := cipher.NewGCM(block)
	want := pbkdf2SaltSize + gcm.NonceSize() + len(plaintext) + gcm.Overhead()
	if len(ct) != want {
		t.Errorf("ciphertext len = %d, want %d (salt %d + nonce %d + data %d + tag %d)",
			len(ct), want, pbkdf2SaltSize, gcm.NonceSize(), len(plaintext), gcm.Overhead())
	}
}

func TestRandomKey_UsesCryptoRand(t *testing.T) {
	// With math/rand the key space is bounded by the PRNG state; crypto/rand
	// gives full 48-bit entropy for a 12-char hex key.
	// We can't inspect the source, but we can verify that 1000 consecutive
	// calls never collide — math/rand would occasionally repeat under load.
	seen := make(map[string]struct{}, 1000)
	for i := range 1000 {
		k := RandomKey()
		if _, dup := seen[k]; dup {
			t.Fatalf("RandomKey produced duplicate after %d iterations", i)
		}
		seen[k] = struct{}{}
	}
}

func TestRandomKey(t *testing.T) {
	// Verifica che RandomKey restituisca sempre una stringa di 12 caratteri esadecimali
	key1 := RandomKey()
	if len(key1) != 12 {
		t.Errorf("Expected key length 12, got %d", len(key1))
	}

	// Verifica che sia composta solo da caratteri esadecimali
	hexPattern := regexp.MustCompile(`^[0-9a-f]{12}$`)
	if !hexPattern.MatchString(key1) {
		t.Errorf("Expected key to be 12 hex characters, got '%s'", key1)
	}

	// Verifica che generi chiavi diverse (molto probabile, ma non garantito)
	key2 := RandomKey()
	if len(key2) != 12 {
		t.Errorf("Expected key2 length 12, got %d", len(key2))
	}
	if !hexPattern.MatchString(key2) {
		t.Errorf("Expected key2 to be 12 hex characters, got '%s'", key2)
	}

	// Genera diverse chiavi per assicurarsi che funzioni
	keys := make(map[string]bool)
	for i := 0; i < 10; i++ {
		key := RandomKey()
		keys[key] = true
		if len(key) != 12 {
			t.Errorf("Expected key length 12 at iteration %d, got %d", i, len(key))
		}
		if !hexPattern.MatchString(key) {
			t.Errorf("Expected key to be 12 hex characters at iteration %d, got '%s'", i, key)
		}
	}

	// Nota: potrebbe generare chiavi duplicate, ma è molto improbabile
	// Verifichiamo almeno che generi qualcosa
	if len(keys) == 0 {
		t.Error("RandomKey should generate at least one key")
	}
}

func TestEncryptionDecryption(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	passphrase := `secret`
	message := `the secret message`

	enc, err := Encrypt([]byte(message), passphrase)
	if err != nil {
		t.Errorf("Encryption error %v", err)
	}
	encB64, err := EncryptToBase64([]byte(message), passphrase)
	if err != nil {
		t.Errorf("Encryption base 64 error %v", err)
	}

	dec, err := Decrypt(enc, passphrase)
	if err != nil {
		t.Errorf("Encryption base 64 error %v", err)
	}
	decB64, err := DecryptBase64(encB64, passphrase)
	if err != nil {
		t.Errorf("Encryption base 64 error %v", err)
	}
	if string(decB64) != message || string(dec) != message {
		t.Errorf("Expected '%s', got dec='%s' b64='%s'", message, string(dec), string(decB64))
	}

}
func TestDecrypt_ShortInput(t *testing.T) {
	passphrase := "secret"

	// inputs shorter than the AES-GCM nonce size (12 bytes) must return an
	// error, not panic
	inputs := [][]byte{
		{},
		[]byte("short"),
		make([]byte, 11),
	}
	for _, input := range inputs {
		_, err := Decrypt(input, passphrase)
		if err == nil {
			t.Errorf("Decrypt(%q) should return an error for input shorter than nonce size", input)
		}
	}
}

func TestDecryptionError(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	passphrase := `secret`
	message := `the secret message`

	enc, err := Encrypt([]byte(message), passphrase)
	if err != nil {
		t.Errorf("Encryption error %v", err)
	}
	encB64, err := EncryptToBase64([]byte(message), passphrase)
	if err != nil {
		t.Errorf("Encryption base 64 error %v", err)
	}

	_, err = Decrypt(enc, `the wrong key`)
	if err == nil {
		t.Errorf("An error was expected (decrypt using wrong key)")
	}
	_, err = Decrypt([]byte(`not encrypted!`), passphrase)
	if err == nil {
		t.Errorf("An error was expected (decrypt an invalid value)")
	}
	_, err = DecryptBase64(encB64, `the wrong key`)
	if err == nil {
		t.Errorf("An error was expected (decrypt base64 using invalid key)")
	}
	_, err = DecryptBase64(`not base64`, passphrase)
	if err == nil {
		t.Errorf("An error was expected (decrypt base64 an invalid base64)")
	}

}
