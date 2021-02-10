package core

import (
	"testing"
)

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
