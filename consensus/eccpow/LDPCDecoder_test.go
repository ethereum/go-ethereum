package eccpow

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/cryptoecc/ETH-ECC/core/types"
)

func TestNonceDecoding(t *testing.T) {
	LDPCNonce := generateRandomNonce()
	EncodedNonce := types.EncodeNonce(LDPCNonce)
	DecodedNonce := EncodedNonce.Uint64()

	if LDPCNonce == DecodedNonce {
		t.Logf("LDPCNonce : %v\n", LDPCNonce)
		t.Logf("Decoded Nonce : %v\n", DecodedNonce)
	} else {
		t.Errorf("LDPCNonce : %v\n", LDPCNonce)
		t.Errorf("Decoded Nonce : %v\n", DecodedNonce)
	}
}

func TestGenerateH(t *testing.T) {
	for i := 0; i < 10; i++ {
		header := new(types.Header)
		header.Difficulty = ProbToDifficulty(Table[0].miningProb)

		parameters, _ := setParameters(header)

		H1 := generateH(parameters)
		H2 := generateH(parameters)

		if !reflect.DeepEqual(H1, H2) {
			t.Error("Wrong")
		}
	}
}

func TestRandShuffle(t *testing.T) {
	for attempt := 0; attempt < 100; attempt++ {
		var hSeed int64
		var colOrder []int

		for i := 1; i < 4; i++ {
			colOrder = nil
			for j := 0; j < 32; j++ {
				colOrder = append(colOrder, j)
			}

			rand.Seed(hSeed)
			rand.Shuffle(len(colOrder), func(i, j int) {
				colOrder[i], colOrder[j] = colOrder[j], colOrder[i]
			})
			hSeed--
		}

		var hSeed2 int64
		var colOrder2 []int

		for i := 1; i < 4; i++ {
			colOrder2 = nil
			for j := 0; j < 32; j++ {
				colOrder2 = append(colOrder2, j)
			}

			rand.Seed(hSeed2)
			rand.Shuffle(len(colOrder2), func(i, j int) {
				colOrder2[i], colOrder2[j] = colOrder2[j], colOrder2[i]
			})
			hSeed2--
		}

		if !reflect.DeepEqual(colOrder, colOrder2) {
			t.Error("Wrong")
		}
	}
}
