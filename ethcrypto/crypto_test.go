package ethcrypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// These tests are sanity checks.
// They should ensure that we don't e.g. use Sha3-224 instead of Sha3-256
// and that the sha3 library uses keccak-f permutation.

func TestSha3(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	checkhash(t, "Sha3-256", Sha3, msg, exp)
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

func checkhash(t *testing.T, name string, f func([]byte) []byte, msg, exp []byte) {
	sum := f(msg)
	if bytes.Compare(exp, sum) != 0 {
		t.Errorf("hash %s returned wrong result.\ngot:  %x\nwant: %x", name, sum, exp)
	}
}
