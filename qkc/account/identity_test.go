// Ported verbatim from github.com/QuarkChain/goquarkchain/account (byte-compatible).

package account

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

type IdentityTestStruct struct {
	Key       string `json:"key"`
	IDKey     string `json:"idkey"`
	Recipient string `json:"recipient"`
}

func CheckIdentityUnitTest(data IdentityTestStruct) bool {
	key, err := hex.DecodeString(data.Key)
	if err != nil {
		fmt.Println("DecodeString failed err", err)
		return false
	}
	keyType := BytesToIdentityKey(key)
	identity, err := CreatIdentityFromKey(keyType)
	if err := checkPublicToRecipient(keyType, identity.recipient); err != nil {
		fmt.Println("checkPublicToRecipient err", err)
		return false
	}

	if err != nil {
		fmt.Println("creatFromKey Failed err", err)
		return false
	}
	if hex.EncodeToString(identity.recipient.Bytes()) != data.Recipient { //checkRecipent
		fmt.Printf("recipient is not match : unexcepted:%s , excepted %s", hex.EncodeToString(identity.recipient.Bytes()), data.Recipient)
		return false
	}
	return true
}

func checkPublicToRecipient(key Key, recipient Recipient) error {
	keyValue := big.NewInt(0)
	keyValue.SetBytes(key.Bytes())
	sk := new(ecdsa.PrivateKey)
	sk.PublicKey.Curve = crypto.S256()
	sk.D = keyValue
	sk.PublicKey.X, sk.PublicKey.Y = crypto.S256().ScalarBaseMult(keyValue.Bytes())

	recipientData := PublicKeyToRecipient(sk.PublicKey)
	if bytes.Equal(recipientData.Bytes(), recipient.Bytes()) {
		return nil
	}
	return errors.New("check public to recipient failed")
}

// 1.python generate testdata
//
//	1.1 identity from key
//
// 2.go.exe to check
//
//	2.1 checkRecipent
func TestIdentity(t *testing.T) {
	JSONParse := NewJSONStruct()
	v := []IdentityTestStruct{}
	err := JSONParse.Load("./testdata/testIdentity.json", &v) //analysis test data
	if err != nil {
		panic(err)
	}
	count := 0
	for _, v := range v {
		err := CheckIdentityUnitTest(v) //unit test
		if err == false {
			panic(err)
		}
		count++
	}
	fmt.Println("TestIdentity:success test num:", count)
}

type TestBytesTo struct {
	recipient         []byte
	key               []byte
	exceptedRecipient *big.Int
	exceptedKey       *big.Int
}

func TestBytesToIdentityKey(t *testing.T) {
	testCase := []TestBytesTo{
		{
			recipient:         []byte{0x1},
			key:               []byte{0x2},
			exceptedRecipient: new(big.Int).SetBytes([]byte{0x1}),
			exceptedKey:       new(big.Int).SetBytes([]byte{0x2}),
		},
		{
			recipient:         []byte{0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1},
			key:               []byte{0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2},
			exceptedRecipient: new(big.Int).SetBytes([]byte{0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1}),
			exceptedKey:       new(big.Int).SetBytes([]byte{0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2}),
		},
		{
			recipient:         []byte{0x66, 0x77, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x2, 0x3},
			key:               []byte{0x66, 0x77, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x3, 0x4},
			exceptedRecipient: new(big.Int).SetBytes([]byte{0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x2, 0x3}),
			exceptedKey:       new(big.Int).SetBytes([]byte{0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x3, 0x4}),
		},
	}

	for _, v := range testCase {
		recipientValue := BytesToIdentityRecipient(v.recipient)
		keyValue := BytesToIdentityKey(v.key)
		if new(big.Int).SetBytes(recipientValue.Bytes()).Cmp(v.exceptedRecipient) != 0 {
			t.Error("test BytesToIdentityKey failed", "excepted:", v.exceptedRecipient.Bytes(), "get", recipientValue.Bytes())
		}
		if new(big.Int).SetBytes(keyValue.Bytes()).Cmp(v.exceptedKey) != 0 {
			t.Error("test BytesToIdentityKey failed", "excepted:", v.exceptedKey, "get", keyValue.Bytes())
		}

	}

}

func TestCreatIdentity(t *testing.T) {
	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}
	id1, err := CreatRandomIdentity()
	if err != nil {
		t.Fatal("CreatRandomIdentity error: ", err)
	}
	id2, err := CreatIdentityFromKey(id1.key)
	if err != nil {
		t.Fatal("CreatIdentityFromKey error: ", err)
	}

	check("GetRecipient", id1.GetRecipient(), id2.GetRecipient())
	check("GetKey", id1.GetKey(), id2.GetKey())
}
