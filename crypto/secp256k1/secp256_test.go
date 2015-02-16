package secp256k1

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/ethereum/go-ethereum/crypto/randentropy"
)

const TESTS = 10000 // how many tests
const SigSize = 65  //64+1

func Test_Secp256_00(t *testing.T) {

	var nonce []byte = randentropy.GetEntropyMixed(32) //going to get bitcoins stolen!

	if len(nonce) != 32 {
		t.Fatal()
	}

}

//tests for Malleability
//highest bit of S must be 0; 32nd byte
func CompactSigTest(sig []byte) {

	var b int = int(sig[32])
	if b < 0 {
		log.Panic()
	}
	if ((b >> 7) == 1) != ((b & 0x80) == 0x80) {
		log.Panic("b= %v b2= %v \n", b, b>>7)
	}
	if (b & 0x80) == 0x80 {
		log.Panic("b= %v b2= %v \n", b, b&0x80)
	}
}

//test pubkey/private generation
func Test_Secp256_01(t *testing.T) {
	pubkey, seckey := GenerateKeyPair()
	if err := VerifySeckeyValidity(seckey); err != nil {
		t.Fatal()
	}
	if err := VerifyPubkeyValidity(pubkey); err != nil {
		t.Fatal()
	}
}

//test size of messages
func Test_Secp256_02s(t *testing.T) {
	pubkey, seckey := GenerateKeyPair()
	msg := randentropy.GetEntropyMixed(32)
	sig, _ := Sign(msg, seckey)
	CompactSigTest(sig)
	if sig == nil {
		t.Fatal("Signature nil")
	}
	if len(pubkey) != 65 {
		t.Fail()
	}
	if len(seckey) != 32 {
		t.Fail()
	}
	if len(sig) != 64+1 {
		t.Fail()
	}
	if int(sig[64]) > 4 {
		t.Fail()
	} //should be 0 to 4
}

//test signing message
func Test_Secp256_02(t *testing.T) {
	pubkey1, seckey := GenerateKeyPair()
	msg := randentropy.GetEntropyMixed(32)
	sig, _ := Sign(msg, seckey)
	if sig == nil {
		t.Fatal("Signature nil")
	}

	pubkey2, _ := RecoverPubkey(msg, sig)
	if pubkey2 == nil {
		t.Fatal("Recovered pubkey invalid")
	}
	if bytes.Equal(pubkey1, pubkey2) == false {
		t.Fatal("Recovered pubkey does not match")
	}

	err := VerifySignature(msg, sig, pubkey1)
	if err != nil {
		t.Fatal("Signature invalid")
	}
}

//test pubkey recovery
func Test_Secp256_02a(t *testing.T) {
	pubkey1, seckey1 := GenerateKeyPair()
	msg := randentropy.GetEntropyMixed(32)
	sig, _ := Sign(msg, seckey1)

	if sig == nil {
		t.Fatal("Signature nil")
	}
	err := VerifySignature(msg, sig, pubkey1)
	if err != nil {
		t.Fatal("Signature invalid")
	}

	pubkey2, _ := RecoverPubkey(msg, sig)
	if len(pubkey1) != len(pubkey2) {
		t.Fatal()
	}
	for i, _ := range pubkey1 {
		if pubkey1[i] != pubkey2[i] {
			t.Fatal()
		}
	}
	if bytes.Equal(pubkey1, pubkey2) == false {
		t.Fatal()
	}
}

//test random messages for the same pub/private key
func Test_Secp256_03(t *testing.T) {
	_, seckey := GenerateKeyPair()
	for i := 0; i < TESTS; i++ {
		msg := randentropy.GetEntropyMixed(32)
		sig, _ := Sign(msg, seckey)
		CompactSigTest(sig)

		sig[len(sig)-1] %= 4
		pubkey2, _ := RecoverPubkey(msg, sig)
		if pubkey2 == nil {
			t.Fail()
		}
	}
}

//test random messages for different pub/private keys
func Test_Secp256_04(t *testing.T) {
	for i := 0; i < TESTS; i++ {
		pubkey1, seckey := GenerateKeyPair()
		msg := randentropy.GetEntropyMixed(32)
		sig, _ := Sign(msg, seckey)
		CompactSigTest(sig)

		if sig[len(sig)-1] >= 4 {
			t.Fail()
		}
		pubkey2, _ := RecoverPubkey(msg, sig)
		if pubkey2 == nil {
			t.Fail()
		}
		if bytes.Equal(pubkey1, pubkey2) == false {
			t.Fail()
		}
	}
}

//test random signatures against fixed messages; should fail

//crashes:
//	-SIPA look at this

func randSig() []byte {
	sig := randentropy.GetEntropyMixed(65)
	sig[32] &= 0x70
	sig[64] %= 4
	return sig
}

func Test_Secp256_06a_alt0(t *testing.T) {
	pubkey1, seckey := GenerateKeyPair()
	msg := randentropy.GetEntropyMixed(32)
	sig, _ := Sign(msg, seckey)

	if sig == nil {
		t.Fail()
	}
	if len(sig) != 65 {
		t.Fail()
	}
	for i := 0; i < TESTS; i++ {
		sig = randSig()
		pubkey2, _ := RecoverPubkey(msg, sig)

		if bytes.Equal(pubkey1, pubkey2) == true {
			t.Fail()
		}

		if pubkey2 != nil && VerifySignature(msg, sig, pubkey2) != nil {
			t.Fail()
		}

		if VerifySignature(msg, sig, pubkey1) == nil {
			t.Fail()
		}
	}
}

//test random messages against valid signature: should fail

func Test_Secp256_06b(t *testing.T) {
	pubkey1, seckey := GenerateKeyPair()
	msg := randentropy.GetEntropyMixed(32)
	sig, _ := Sign(msg, seckey)

	fail_count := 0
	for i := 0; i < TESTS; i++ {
		msg = randentropy.GetEntropyMixed(32)
		pubkey2, _ := RecoverPubkey(msg, sig)
		if bytes.Equal(pubkey1, pubkey2) == true {
			t.Fail()
		}

		if pubkey2 != nil && VerifySignature(msg, sig, pubkey2) != nil {
			t.Fail()
		}

		if VerifySignature(msg, sig, pubkey1) == nil {
			t.Fail()
		}
	}
	if fail_count != 0 {
		fmt.Printf("ERROR: Accepted signature for %v of %v random messages\n", fail_count, TESTS)
	}
}

func TestInvalidKey(t *testing.T) {
	p1 := make([]byte, 32)
	err := VerifySeckeyValidity(p1)
	if err == nil {
		t.Errorf("pvk %x varify sec key should have returned error", p1)
	}
}
