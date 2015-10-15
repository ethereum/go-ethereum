// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package crypto

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/randentropy"
)

func TestKeyStorePlain(t *testing.T) {
	ks := NewKeyStorePlain(common.DefaultDataDir())
	pass := "" // not used but required by API
	k1, err := ks.GenerateNewKey(randentropy.Reader, pass)
	if err != nil {
		t.Fatal(err)
	}

	k2 := new(Key)
	k2, err = ks.GetKey(k1.Address, pass)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(k1.Address, k2.Address) {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(k1.PrivateKey, k2.PrivateKey) {
		t.Fatal(err)
	}

	err = ks.DeleteKey(k2.Address, pass)
	if err != nil {
		t.Fatal(err)
	}
}

func TestKeyStorePassphrase(t *testing.T) {
	ks := NewKeyStorePassphrase(common.DefaultDataDir())
	pass := "foo"
	k1, err := ks.GenerateNewKey(randentropy.Reader, pass)
	if err != nil {
		t.Fatal(err)
	}
	k2 := new(Key)
	k2, err = ks.GetKey(k1.Address, pass)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(k1.Address, k2.Address) {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(k1.PrivateKey, k2.PrivateKey) {
		t.Fatal(err)
	}

	err = ks.DeleteKey(k2.Address, pass) // also to clean up created files
	if err != nil {
		t.Fatal(err)
	}
}

func TestKeyStorePassphraseDecryptionFail(t *testing.T) {
	ks := NewKeyStorePassphrase(common.DefaultDataDir())
	pass := "foo"
	k1, err := ks.GenerateNewKey(randentropy.Reader, pass)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ks.GetKey(k1.Address, "bar") // wrong passphrase
	if err == nil {
		t.Fatal(err)
	}

	err = ks.DeleteKey(k1.Address, "bar") // wrong passphrase
	if err == nil {
		t.Fatal(err)
	}

	err = ks.DeleteKey(k1.Address, pass) // to clean up
	if err != nil {
		t.Fatal(err)
	}
}

func TestImportPreSaleKey(t *testing.T) {
	// file content of a presale key file generated with:
	// python pyethsaletool.py genwallet
	// with password "foo"
	fileContent := "{\"encseed\": \"26d87f5f2bf9835f9a47eefae571bc09f9107bb13d54ff12a4ec095d01f83897494cf34f7bed2ed34126ecba9db7b62de56c9d7cd136520a0427bfb11b8954ba7ac39b90d4650d3448e31185affcd74226a68f1e94b1108e6e0a4a91cdd83eba\", \"ethaddr\": \"d4584b5f6229b7be90727b0fc8c6b91bb427821f\", \"email\": \"gustav.simonsson@gmail.com\", \"btcaddr\": \"1EVknXyFC68kKNLkh6YnKzW41svSRoaAcx\"}"
	ks := NewKeyStorePassphrase(common.DefaultDataDir())
	pass := "foo"
	_, err := ImportPreSaleKey(ks, []byte(fileContent), pass)
	if err != nil {
		t.Fatal(err)
	}
}

// Test and utils for the key store tests in the Ethereum JSON tests;
// tests/KeyStoreTests/basic_tests.json
type KeyStoreTestV3 struct {
	Json     encryptedKeyJSONV3
	Password string
	Priv     string
}

type KeyStoreTestV1 struct {
	Json     encryptedKeyJSONV1
	Password string
	Priv     string
}

func TestV3_PBKDF2_1(t *testing.T) {
	tests := loadKeyStoreTestV3("tests/v3_test_vector.json", t)
	testDecryptV3(tests["wikipage_test_vector_pbkdf2"], t)
}

func TestV3_PBKDF2_2(t *testing.T) {
	tests := loadKeyStoreTestV3("../tests/files/KeyStoreTests/basic_tests.json", t)
	testDecryptV3(tests["test1"], t)
}

func TestV3_PBKDF2_3(t *testing.T) {
	tests := loadKeyStoreTestV3("../tests/files/KeyStoreTests/basic_tests.json", t)
	testDecryptV3(tests["python_generated_test_with_odd_iv"], t)
}

func TestV3_PBKDF2_4(t *testing.T) {
	tests := loadKeyStoreTestV3("../tests/files/KeyStoreTests/basic_tests.json", t)
	testDecryptV3(tests["evilnonce"], t)
}

func TestV3_Scrypt_1(t *testing.T) {
	tests := loadKeyStoreTestV3("tests/v3_test_vector.json", t)
	testDecryptV3(tests["wikipage_test_vector_scrypt"], t)
}

func TestV3_Scrypt_2(t *testing.T) {
	tests := loadKeyStoreTestV3("../tests/files/KeyStoreTests/basic_tests.json", t)
	testDecryptV3(tests["test2"], t)
}

func TestV1_1(t *testing.T) {
	tests := loadKeyStoreTestV1("tests/v1_test_vector.json", t)
	testDecryptV1(tests["test1"], t)
}

func TestV1_2(t *testing.T) {
	ks := NewKeyStorePassphrase("tests/v1")
	addr := common.HexToAddress("cb61d5a9c4896fb9658090b597ef0e7be6f7b67e")
	k, err := ks.GetKey(addr, "g")
	if err != nil {
		t.Fatal(err)
	}
	if k.Address != addr {
		t.Fatal(fmt.Errorf("Unexpected address: %v, expected %v", k.Address, addr))
	}

	privHex := hex.EncodeToString(FromECDSA(k.PrivateKey))
	expectedHex := "d1b1178d3529626a1a93e073f65028370d14c7eb0936eb42abef05db6f37ad7d"
	if privHex != expectedHex {
		t.Fatal(fmt.Errorf("Unexpected privkey: %v, expected %v", privHex, expectedHex))
	}
}

func testDecryptV3(test KeyStoreTestV3, t *testing.T) {
	privBytes, _, err := decryptKeyV3(&test.Json, test.Password)
	if err != nil {
		t.Fatal(err)
	}
	privHex := hex.EncodeToString(privBytes)
	if test.Priv != privHex {
		t.Fatal(fmt.Errorf("Decrypted bytes not equal to test, expected %v have %v", test.Priv, privHex))
	}
}

func testDecryptV1(test KeyStoreTestV1, t *testing.T) {
	privBytes, _, err := decryptKeyV1(&test.Json, test.Password)
	if err != nil {
		t.Fatal(err)
	}
	privHex := hex.EncodeToString(privBytes)
	if test.Priv != privHex {
		t.Fatal(fmt.Errorf("Decrypted bytes not equal to test, expected %v have %v", test.Priv, privHex))
	}
}

func loadKeyStoreTestV3(file string, t *testing.T) map[string]KeyStoreTestV3 {
	tests := make(map[string]KeyStoreTestV3)
	err := common.LoadJSON(file, &tests)
	if err != nil {
		t.Fatal(err)
	}
	return tests
}

func loadKeyStoreTestV1(file string, t *testing.T) map[string]KeyStoreTestV1 {
	tests := make(map[string]KeyStoreTestV1)
	err := common.LoadJSON(file, &tests)
	if err != nil {
		t.Fatal(err)
	}
	return tests
}

func TestKeyForDirectICAP(t *testing.T) {
	key := NewKeyForDirectICAP(randentropy.Reader)
	if !strings.HasPrefix(key.Address.Hex(), "0x00") {
		t.Errorf("Expected first address byte to be zero, have: %s", key.Address.Hex())
	}
}
