package storage

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-colorable"
)

func TestEncryption(t *testing.T) {
	//	key := []byte("AES256Key-32Characters1234567890")
	//	plaintext := []byte(value)
	key := []byte("AES256Key-32Characters1234567890")
	plaintext := []byte("exampleplaintext")

	c, iv, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Ciphertext %x, nonce %x\n", c, iv)

	p, err := decrypt(key, iv, c)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Plaintext %v\n", string(p))
	if !bytes.Equal(plaintext, p) {
		t.Errorf("Failed: expected plaintext recovery, got %v expected %v", string(plaintext), string(p))
	}
}

func TestFileStorage(t *testing.T) {

	a := map[string]storedCredential{
		"secret": storedCredential{
			Iv:         common.Hex2Bytes("cdb30036279601aeee60f16b"),
			CipherText: common.Hex2Bytes("f311ac49859d7260c2c464c28ffac122daf6be801d3cfd3edcbde7e00c9ff74f"),
		},
		"secret2": storedCredential{
			Iv:         common.Hex2Bytes("afb8a7579bf971db9f8ceeed"),
			CipherText: common.Hex2Bytes("2df87baf86b5073ef1f03e3cc738de75b511400f5465bb0ddeacf47ae4dc267d"),
		},
	}
	d, err := ioutil.TempDir("", "eth-encrypted-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	stored := &AESEncryptedStorage{
		filename: fmt.Sprintf("%v/vault.json", d),
		key:      []byte("AES256Key-32Characters1234567890"),
	}
	stored.writeEncryptedStorage(a)
	read := &AESEncryptedStorage{
		filename: fmt.Sprintf("%v/vault.json", d),
		key:      []byte("AES256Key-32Characters1234567890"),
	}
	creds, err := read.readEncryptedStorage()
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range a {
		if v2, exist := creds[k]; !exist {
			t.Errorf("Missing entry %v", k)
		} else {
			if !bytes.Equal(v.CipherText, v2.CipherText) {
				t.Errorf("Wrong ciphertext, expected %x got %x", v.CipherText, v2.CipherText)
			}
			if !bytes.Equal(v.Iv, v2.Iv) {
				t.Errorf("Wrong iv")
			}
		}
	}
}
func TestEnd2End(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(3), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))

	d, err := ioutil.TempDir("", "eth-encrypted-storage-test")
	if err != nil {
		t.Fatal(err)
	}

	s1 := &AESEncryptedStorage{
		filename: fmt.Sprintf("%v/vault.json", d),
		key:      []byte("AES256Key-32Characters1234567890"),
	}
	s2 := &AESEncryptedStorage{
		filename: fmt.Sprintf("%v/vault.json", d),
		key:      []byte("AES256Key-32Characters1234567890"),
	}

	s1.Put("bazonk", "foobar")
	if v := s2.Get("bazonk"); v != "foobar" {
		t.Errorf("Expected bazonk->foobar, got '%v'", v)
	}
}
