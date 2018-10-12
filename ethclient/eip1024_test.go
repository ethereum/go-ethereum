package ethclient

import (
	"context"
	"testing"
)

var client = &Client{}

var (
	bobEthereumPrivateKey  = "7e5374ec2ef0d91761a6e72fdf8f6ac665519bfdf6da0a2329cf0d804514b816"
	bobEncryptionPublicKey = "C5YMNdqE4kLgxQhJO1MfuQcHP5hjVSXzamzd/TxlR0U="
	secretMessage          = "My name is Satoshi Buterin"
	encryptedData          = EncryptedData{
		Version:        "x25519-xsalsa20-poly1305",
		Nonce:          "1dvWO7uOnBnO7iNDJ9kO9pTasLuKNlej",
		EphemPublicKey: "FBH1/pAEHOOW14Lu3FWkgV3qOEcuL78Zy+qW1RwzMXQ=",
		Ciphertext:     "f8kBcl/NCyf3sybfbwAKk/np2Bzt9lRVkZejr6uh5FgnNlH/ic62DZzy",
	}
)

func TestGetEncryptionPublicKey(t *testing.T) {
	result, err := client.GetEncryptionPublicKey(context.TODO(), bobEthereumPrivateKey)
	if err != nil {
		t.Errorf("test getEncryptionPublicKey: error: %v", err)
	}
	if result != bobEncryptionPublicKey {
		t.Errorf("test getEncryptionPublicKey: mismatch: expected %s, actual %s", bobEncryptionPublicKey, result)
	}
}

func TestEncrypt(t *testing.T) {
	version := "x25519-xsalsa20-poly1305"
	encrypted, err := client.Encrypt(
		context.TODO(),
		bobEncryptionPublicKey,
		[]byte(secretMessage),
		version,
	)

	if err != nil {
		t.Errorf("test encrypt: error: %v", err)
	}
	if version != encrypted.Version {
		t.Errorf("test encrypt: mismatch version: expected %s, actual %s", version, encrypted.Version)
	}
	if encrypted.Nonce == "" {
		t.Errorf("test encrypt: empty nonce")
	}
	if encrypted.Ciphertext == "" {
		t.Errorf("test encrypt: empty ciphertext")
	}
	if encrypted.EphemPublicKey == "" {
		t.Errorf("test encrypt: empty ephemPublicKey")
	}
}

func TestDecrypt(t *testing.T) {
	decrypted, err := client.Decrypt(context.TODO(), bobEthereumPrivateKey, &encryptedData)

	if err != nil {
		t.Errorf("test decrypt got error: %v", err)
	}
	if secretMessage != string(decrypted) {
		t.Errorf("test decrypt: mismatch: expected %s, actual %s", secretMessage, string(decrypted))
	}
}
