package storage

import (
	"bytes"
	"testing"
)

func TestEncryption(t *testing.T) {
	//	key := []byte("AES256Key-32Characters1234567890")
	//	plaintext := []byte(value)
	key := []byte("AES256Key-32Characters1234567890")
	plaintext := []byte("exampleplaintext")

	c, iv, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Ciphertext %x, nonce %x\n", c, iv)

	p, err := Decrypt(key, iv, c, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Plaintext %v\n", string(p))
	if !bytes.Equal(plaintext, p) {
		t.Errorf("Failed: expected plaintext recovery, got %v expected %v", string(plaintext), string(p))
	}
}
