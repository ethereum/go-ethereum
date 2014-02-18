package ethutil

import (
	"code.google.com/p/go.crypto/ripemd160"
	"crypto/sha256"
	"encoding/hex"
	"github.com/obscuren/sha3"
	"strconv"
)

func Uitoa(i uint32) string {
	return strconv.FormatUint(uint64(i), 10)
}

func Sha256Bin(data []byte) []byte {
	hash := sha256.Sum256(data)

	return hash[:]
}

func Ripemd160(data []byte) []byte {
	ripemd := ripemd160.New()
	ripemd.Write(data)

	return ripemd.Sum(nil)
}

func Sha3Bin(data []byte) []byte {
	d := sha3.NewKeccak256()
	d.Write(data)

	return d.Sum(nil)
}

// Helper function for comparing slices
func CompareIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Returns the amount of nibbles that match each other from 0 ...
func MatchingNibbleLength(a, b []int) int {
	i := 0
	for CompareIntSlice(a[:i+1], b[:i+1]) && i < len(b) {
		i += 1
	}

	return i
}

func Hex(d []byte) string {
	return hex.EncodeToString(d)
}
func ToHex(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}
