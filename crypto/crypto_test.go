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
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

var testAddrHex = "970e8128ab834e8eac17ab8e3812f010678cf791"
var testPrivHex = "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"

// These tests are sanity checks.
// They should ensure that we don't e.g. use Sha3-224 instead of Sha3-256
// and that the sha3 library uses keccak-f permutation.
func TestSha3(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	checkhash(t, "Sha3-256", func(in []byte) []byte { return Keccak256(in) }, msg, exp)
}

func TestSha3Hash(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	checkhash(t, "Sha3-256-array", func(in []byte) []byte { h := Keccak256Hash(in); return h[:] }, msg, exp)
}

func TestSha256(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
	checkhash(t, "Sha256", Sha256, msg, exp)
}

func TestRipemd160(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("8eb208f7e05d987a9b044a8e98c6b087f15a0bfc")
	checkhash(t, "Ripemd160", Ripemd160, msg, exp)
}

func BenchmarkSha3(b *testing.B) {
	a := []byte("hello world")
	amount := 1000000
	start := time.Now()
	for i := 0; i < amount; i++ {
		Keccak256(a)
	}

	fmt.Println(amount, ":", time.Since(start))
}

func Test0Key(t *testing.T) {
	key := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")
	_, err := secp256k1.GeneratePubKey(key)
	if err == nil {
		t.Errorf("expected error due to zero privkey")
	}
}

func testSign(signfn func([]byte, *ecdsa.PrivateKey) ([]byte, error), t *testing.T) {
	key, _ := HexToECDSA(testPrivHex)
	addr := common.HexToAddress(testAddrHex)

	msg := Keccak256([]byte("foo"))
	sig, err := signfn(msg, key)
	if err != nil {
		t.Errorf("Sign error: %s", err)
	}

	// signfn can return a recover id of either [0,1] or [27,28].
	// In the latter case its an Ethereum signature, adjust recover id.
	if sig[64] == 27 || sig[64] == 28 {
		sig[64] -= 27
	}

	recoveredPub, err := Ecrecover(msg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	pubKey := ToECDSAPub(recoveredPub)
	recoveredAddr := PubkeyToAddress(*pubKey)
	if addr != recoveredAddr {
		t.Errorf("Address mismatch: want: %x have: %x", addr, recoveredAddr)
	}

	// should be equal to SigToPub
	recoveredPub2, err := SigToPub(msg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	recoveredAddr2 := PubkeyToAddress(*recoveredPub2)
	if addr != recoveredAddr2 {
		t.Errorf("Address mismatch: want: %x have: %x", addr, recoveredAddr2)
	}
}

func TestSign(t *testing.T) {
	testSign(Sign, t)
}

func TestSignEthereum(t *testing.T) {
	testSign(SignEthereum, t)
}

func testInvalidSign(signfn func([]byte, *ecdsa.PrivateKey) ([]byte, error), t *testing.T) {
	_, err := signfn(make([]byte, 1), nil)
	if err == nil {
		t.Errorf("expected sign with hash 1 byte to error")
	}

	_, err = signfn(make([]byte, 33), nil)
	if err == nil {
		t.Errorf("expected sign with hash 33 byte to error")
	}
}

func TestInvalidSign(t *testing.T) {
	testInvalidSign(Sign, t)
}

func TestInvalidSignEthereum(t *testing.T) {
	testInvalidSign(SignEthereum, t)
}

func TestNewContractAddress(t *testing.T) {
	key, _ := HexToECDSA(testPrivHex)
	addr := common.HexToAddress(testAddrHex)
	genAddr := PubkeyToAddress(key.PublicKey)
	// sanity check before using addr to create contract address
	checkAddr(t, genAddr, addr)

	caddr0 := CreateAddress(addr, 0)
	caddr1 := CreateAddress(addr, 1)
	caddr2 := CreateAddress(addr, 2)
	checkAddr(t, common.HexToAddress("333c3310824b7c685133f2bedb2ca4b8b4df633d"), caddr0)
	checkAddr(t, common.HexToAddress("8bda78331c916a08481428e4b07c96d3e916d165"), caddr1)
	checkAddr(t, common.HexToAddress("c9ddedf451bc62ce88bf9292afb13df35b670699"), caddr2)
}

func TestLoadECDSAFile(t *testing.T) {
	keyBytes := common.FromHex(testPrivHex)
	fileName0 := "test_key0"
	fileName1 := "test_key1"
	checkKey := func(k *ecdsa.PrivateKey) {
		checkAddr(t, PubkeyToAddress(k.PublicKey), common.HexToAddress(testAddrHex))
		loadedKeyBytes := FromECDSA(k)
		if !bytes.Equal(loadedKeyBytes, keyBytes) {
			t.Fatalf("private key mismatch: want: %x have: %x", keyBytes, loadedKeyBytes)
		}
	}

	ioutil.WriteFile(fileName0, []byte(testPrivHex), 0600)
	defer os.Remove(fileName0)

	key0, err := LoadECDSA(fileName0)
	if err != nil {
		t.Fatal(err)
	}
	checkKey(key0)

	// again, this time with SaveECDSA instead of manual save:
	err = SaveECDSA(fileName1, key0)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName1)

	key1, err := LoadECDSA(fileName1)
	if err != nil {
		t.Fatal(err)
	}
	checkKey(key1)
}

func TestValidateSignatureValues(t *testing.T) {
	check := func(expected bool, v byte, r, s *big.Int) {
		if ValidateSignatureValues(v, r, s, false) != expected {
			t.Errorf("mismatch for v: %d r: %d s: %d want: %v", v, r, s, expected)
		}
	}
	minusOne := big.NewInt(-1)
	one := common.Big1
	zero := common.Big0
	secp256k1nMinus1 := new(big.Int).Sub(secp256k1.N, common.Big1)

	// correct v,r,s
	check(true, 27, one, one)
	check(true, 28, one, one)
	// incorrect v, correct r,s,
	check(false, 30, one, one)
	check(false, 26, one, one)

	// incorrect v, combinations of incorrect/correct r,s at lower limit
	check(false, 0, zero, zero)
	check(false, 0, zero, one)
	check(false, 0, one, zero)
	check(false, 0, one, one)

	// correct v for any combination of incorrect r,s
	check(false, 27, zero, zero)
	check(false, 27, zero, one)
	check(false, 27, one, zero)

	check(false, 28, zero, zero)
	check(false, 28, zero, one)
	check(false, 28, one, zero)

	// correct sig with max r,s
	check(true, 27, secp256k1nMinus1, secp256k1nMinus1)
	// correct v, combinations of incorrect r,s at upper limit
	check(false, 27, secp256k1.N, secp256k1nMinus1)
	check(false, 27, secp256k1nMinus1, secp256k1.N)
	check(false, 27, secp256k1.N, secp256k1.N)

	// current callers ensures r,s cannot be negative, but let's test for that too
	// as crypto package could be used stand-alone
	check(false, 27, minusOne, one)
	check(false, 27, one, minusOne)
}

func checkhash(t *testing.T, name string, f func([]byte) []byte, msg, exp []byte) {
	sum := f(msg)
	if bytes.Compare(exp, sum) != 0 {
		t.Fatalf("hash %s mismatch: want: %x have: %x", name, exp, sum)
	}
}

func checkAddr(t *testing.T, addr0, addr1 common.Address) {
	if addr0 != addr1 {
		t.Fatalf("address mismatch: want: %x have: %x", addr0, addr1)
	}
}

// test to help Python team with integration of libsecp256k1
// skip but keep it after they are done
func TestPythonIntegration(t *testing.T) {
	kh := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	k0, _ := HexToECDSA(kh)
	k1 := FromECDSA(k0)

	msg0 := Keccak256([]byte("foo"))
	sig0, _ := secp256k1.Sign(msg0, k1)

	msg1 := common.FromHex("00000000000000000000000000000000")
	sig1, _ := secp256k1.Sign(msg0, k1)

	fmt.Printf("msg: %x, privkey: %x sig: %x\n", msg0, k1, sig0)
	fmt.Printf("msg: %x, privkey: %x sig: %x\n", msg1, k1, sig1)
}

// TestChecksumAddress performs 4 groups of tests: a) all caps,
// b) all lower, c) mixed lower and upper case, and
// d) examples taken from myetherwallet.com
func TestChecksumAddress(t *testing.T) {
	testcases := []string{
		"0x52908400098527886E0F7030069857D2E4169EE7",
		"0x8617E340B3D01FA5F11F306F4090FD50E238070D",

		"0xde709f2102306220921060314715629080e2fb77",
		"0x27b1fdb04752bbc536007a920d24acb045561c26",

		"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		"0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
		"0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB",
		"0xD1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb",

		"0x5A4EAB120fB44eb6684E5e32785702FF45ea344D",
		"0x5be4BDC48CeF65dbCbCaD5218B1A7D37F58A0741",
		"0xa7dD84573f5ffF821baf2205745f768F8edCDD58",
		"0x027a49d11d118c0060746F1990273FcB8c2fC196",
		"0x689E3fE51F45760Ab73D237d28fc1d2C8EaC6D71",
		"0x97D509F0b388daE6D000C33193F4645D1e71Dc54",
		"0xa4Fd5bD20Cf5A7CF1c5A6015D2b3e08A3eC1b1a7",
		"0x230AE42Daf56B494E4b9E6D8Cce99F5E14FE29c1",
		"0xC19D1EDB7FC943f2abbF576f6058c2425B347AB9",
		"0x4f936Bb00CaaD116adc3861146dd8f68BF66F4E6",
		"0xE74287ECA7B7151Fd194cdf7680EB50752671c47",
		"0x5d32a30FBc5bddF39293CE3a9D74E4505dEb621D",
		"0x27cBC66cbE3625c2857ce3CF77A9933e589545DF",
		"0xE2A5f301EA7e461880Fe9A6B4b7EC1aBD023129A",
		"0xe0DFdDA1D174aB7315C753EA198885ee88B52763",
		"0x843655C78939365298FD9515b489939bADca64Ec",
		"0x6bB7a54E4ef381e4C64009DDa0A9ED127aab852C",
	}

	failed, passed := 0, 0
	for i := 0; i < len(testcases); i++ {
		ca := ChecksumAddress(common.HexToAddress(testcases[i]))
		if testcases[i] == ca {
			passed++
		} else {
			failed++
			t.Errorf("Failed to compute ChecksumAddress for address " + testcases[i] + ": \n\tExpected=" + testcases[i] + "\n\tReceived=" + ca)
		}
	}
	fmt.Printf("Passed %d tests and failed %d tests out of %d.\n", passed, failed, passed+failed)
}

func TestChecksumAddressHex(t *testing.T) {
	testcases := []string{
		"0x52908400098527886E0F7030069857D2E4169EE7",
		"0x8617E340B3D01FA5F11F306F4090FD50E238070D",

		"0xde709f2102306220921060314715629080e2fb77",
		"0x27b1fdb04752bbc536007a920d24acb045561c26",

		"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		"0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
		"0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB",
		"0xD1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb",

		"0x5A4EAB120fB44eb6684E5e32785702FF45ea344D",
		"0x5be4BDC48CeF65dbCbCaD5218B1A7D37F58A0741",
		"0xa7dD84573f5ffF821baf2205745f768F8edCDD58",
		"0x027a49d11d118c0060746F1990273FcB8c2fC196",
		"0x689E3fE51F45760Ab73D237d28fc1d2C8EaC6D71",
		"0x97D509F0b388daE6D000C33193F4645D1e71Dc54",
		"0xa4Fd5bD20Cf5A7CF1c5A6015D2b3e08A3eC1b1a7",
		"0x230AE42Daf56B494E4b9E6D8Cce99F5E14FE29c1",
		"0xC19D1EDB7FC943f2abbF576f6058c2425B347AB9",
		"0x4f936Bb00CaaD116adc3861146dd8f68BF66F4E6",
		"0xE74287ECA7B7151Fd194cdf7680EB50752671c47",
		"0x5d32a30FBc5bddF39293CE3a9D74E4505dEb621D",
		"0x27cBC66cbE3625c2857ce3CF77A9933e589545DF",
		"0xE2A5f301EA7e461880Fe9A6B4b7EC1aBD023129A",
		"0xe0DFdDA1D174aB7315C753EA198885ee88B52763",
		"0x843655C78939365298FD9515b489939bADca64Ec",
		"0x6bB7a54E4ef381e4C64009DDa0A9ED127aab852C",
	}

	failed, passed := 0, 0
	for i := 0; i < len(testcases); i++ {
		ca := ChecksumAddressHex(testcases[i])
		if testcases[i] == ca {
			passed++
		} else {
			failed++
			t.Errorf("Failed to compute ChecksumAddressHex for address " + testcases[i] + ": \n\tExpected=" + testcases[i] + "\n\tReceived=" + ca)
		}
	}

	testcases = []string{
		"52908400098527886E0F7030069857D2E4169EE7",
		"8617E340B3D01FA5F11F306F4090FD50E238070D",

		"de709f2102306220921060314715629080e2fb77",
		"27b1fdb04752bbc536007a920d24acb045561c26",

		"5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		"fB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
		"dbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB",
		"D1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb",

		"5A4EAB120fB44eb6684E5e32785702FF45ea344D",
		"5be4BDC48CeF65dbCbCaD5218B1A7D37F58A0741",
		"a7dD84573f5ffF821baf2205745f768F8edCDD58",
		"027a49d11d118c0060746F1990273FcB8c2fC196",
		"689E3fE51F45760Ab73D237d28fc1d2C8EaC6D71",
		"97D509F0b388daE6D000C33193F4645D1e71Dc54",
		"a4Fd5bD20Cf5A7CF1c5A6015D2b3e08A3eC1b1a7",
		"230AE42Daf56B494E4b9E6D8Cce99F5E14FE29c1",
		"C19D1EDB7FC943f2abbF576f6058c2425B347AB9",
		"4f936Bb00CaaD116adc3861146dd8f68BF66F4E6",
		"E74287ECA7B7151Fd194cdf7680EB50752671c47",
		"5d32a30FBc5bddF39293CE3a9D74E4505dEb621D",
		"27cBC66cbE3625c2857ce3CF77A9933e589545DF",
		"E2A5f301EA7e461880Fe9A6B4b7EC1aBD023129A",
		"e0DFdDA1D174aB7315C753EA198885ee88B52763",
		"843655C78939365298FD9515b489939bADca64Ec",
		"6bB7a54E4ef381e4C64009DDa0A9ED127aab852C",
	}

	for i := 0; i < len(testcases); i++ {
		ca := ChecksumAddressHex(testcases[i])
		if "0x"+testcases[i] == ca {
			passed++
		} else {
			failed++
			t.Errorf("Failed to compute ChecksumAddressHex for address " + testcases[i] + ": \n\tExpected=" + testcases[i] + "\n\tReceived=" + ca)
		}
	}
	fmt.Printf("Passed %d tests and failed %d tests out of %d.\n", passed, failed, passed+failed)
}
