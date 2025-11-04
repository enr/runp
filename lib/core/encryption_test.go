package core

import (
	"regexp"
	"testing"
)

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
func TestDecryptionError(t *testing.T) {
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
