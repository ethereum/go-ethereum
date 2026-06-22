// Ported verbatim from github.com/QuarkChain/goquarkchain/common (byte-compatible).

package common

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestTokenCharEncode(t *testing.T) {
	EncodedValues := make(map[string]uint64)
	EncodedValues["0"] = 0
	EncodedValues["Z"] = 35
	EncodedValues["00"] = 36
	EncodedValues["0Z"] = 71
	EncodedValues["1Z"] = 107
	EncodedValues["20"] = 108
	EncodedValues["ZZ"] = 1331
	EncodedValues["QKC"] = 35760
	EncodedValues[TOKENMAX] = TOKENIDMAX

	for key, value := range EncodedValues {
		if value != TokenIDEncode(key) {
			t.Fatalf("key:%v should: %v is %v", key, value, TokenIDEncode(key))
		}
	}
}

func TestRandomToken(t *testing.T) {
	count := 100000
	for index := 0; index < count; index++ {
		data := rand.Intn(int(TOKENIDMAX))

		deData, err := TokenIdDecode(uint64(data))
		if err != nil {
			fmt.Println("data", data)
			panic(err)
		}
		newData := TokenIDEncode(deData)
		if newData != uint64(data) {
			t.Fatalf("data:%v newData:%v", data, newData)
		}
	}
}
