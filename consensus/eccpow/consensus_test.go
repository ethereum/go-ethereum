// Copyright 2017 The go-ethereum Authors
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

package eccpow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/cryptoecc/ETH-ECC/common"
	"github.com/cryptoecc/ETH-ECC/common/math"
	"github.com/cryptoecc/ETH-ECC/core/types"
	"github.com/cryptoecc/ETH-ECC/params"
)

type diffTest struct {
	ParentTimestamp    uint64
	ParentDifficulty   *big.Int
	CurrentTimestamp   uint64
	CurrentBlocknumber *big.Int
	CurrentDifficulty  *big.Int
}

func (d *diffTest) UnmarshalJSON(b []byte) (err error) {
	var ext struct {
		ParentTimestamp    string
		ParentDifficulty   string
		CurrentTimestamp   string
		CurrentBlocknumber string
		CurrentDifficulty  string
	}
	if err := json.Unmarshal(b, &ext); err != nil {
		return err
	}

	d.ParentTimestamp = math.MustParseUint64(ext.ParentTimestamp)
	d.ParentDifficulty = math.MustParseBig256(ext.ParentDifficulty)
	d.CurrentTimestamp = math.MustParseUint64(ext.CurrentTimestamp)
	d.CurrentBlocknumber = math.MustParseBig256(ext.CurrentBlocknumber)
	d.CurrentDifficulty = math.MustParseBig256(ext.CurrentDifficulty)

	return nil
}

func TestCalcDifficulty(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "..", "tests", "testdata", "BasicTests", "difficulty.json"))
	if err != nil {
		t.Skip(err)
	}
	defer file.Close()

	tests := make(map[string]diffTest)
	err = json.NewDecoder(file).Decode(&tests)
	if err != nil {
		t.Fatal(err)
	}

	config := &params.ChainConfig{HomesteadBlock: big.NewInt(1150000)}

	for name, test := range tests {
		number := new(big.Int).Sub(test.CurrentBlocknumber, big.NewInt(1))
		diff := CalcDifficulty(config, test.CurrentTimestamp, &types.Header{
			Number:     number,
			Time:       test.ParentTimestamp,
			Difficulty: test.ParentDifficulty,
		})
		if diff.Cmp(test.CurrentDifficulty) != 0 {
			t.Error(name, "failed. Expected", test.CurrentDifficulty, "and calculated", diff)
		}
	}
}

func TestDecodingVerification(t *testing.T) {
	for i := 0; i < 8; i++ {
		ecc := ECC{}
		header := new(types.Header)
		header.Difficulty = ProbToDifficulty(Table[0].miningProb)
		hash := ecc.SealHash(header).Bytes()

		_, hashVector, outputWord, LDPCNonce, digest := RunOptimizedConcurrencyLDPC(header, hash)

		headerForTest := types.CopyHeader(header)
		headerForTest.MixDigest = common.BytesToHash(digest)
		headerForTest.Nonce = types.EncodeNonce(LDPCNonce)
		hashForTest := ecc.SealHash(headerForTest).Bytes()

		flag, hashVectorOfVerification, outputWordOfVerification, digestForValidation := VerifyOptimizedDecoding(headerForTest, hashForTest)

		encodedDigestForValidation := common.BytesToHash(digestForValidation)

		//fmt.Printf("%+v\n", header)
		//fmt.Printf("Hash : %v\n", hash)
		//fmt.Println()

		//fmt.Printf("%+v\n", headerForTest)
		//fmt.Printf("headerForTest : %v\n", headerForTest)
		//fmt.Println()

		// * means padding for compare easily
		if flag && bytes.Equal(headerForTest.MixDigest[:], encodedDigestForValidation[:]) {
			fmt.Printf("Hash vector ** ************ : %v\n", hashVector)
			fmt.Printf("Hash vector of verification : %v\n", hashVectorOfVerification)

			fmt.Printf("Outputword ** ************ : %v\n", outputWord)
			fmt.Printf("Outputword of verification : %v\n", outputWordOfVerification)

			fmt.Printf("LDPC Nonce : %v\n", LDPCNonce)
			fmt.Printf("Digest : %v\n", headerForTest.MixDigest[:])
			/*
				t.Logf("Hash vector : %v\n", hashVector)
				t.Logf("Outputword : %v\n", outputWord)
				t.Logf("LDPC Nonce : %v\n", LDPCNonce)
				t.Logf("Digest : %v\n", header.MixDigest[:])
			*/
		} else {
			fmt.Printf("Hash vector ** ************ : %v\n", hashVector)
			fmt.Printf("Hash vector of verification : %v\n", hashVectorOfVerification)

			fmt.Printf("Outputword ** ************ : %v\n", outputWord)
			fmt.Printf("Outputword of verification : %v\n", outputWordOfVerification)

			fmt.Printf("flag : %v\n", flag)
			fmt.Printf("Digest compare result : %v\n", bytes.Equal(headerForTest.MixDigest[:], encodedDigestForValidation[:]))
			fmt.Printf("Digest *** ********** : %v\n", headerForTest.MixDigest[:])
			fmt.Printf("Digest for validation : %v\n", encodedDigestForValidation)

			t.Errorf("Test Fail")
			/*
				t.Errorf("flag : %v\n", flag)
				t.Errorf("Digest compare result : %v", bytes.Equal(header.MixDigest[:], digestForValidation)
				t.Errorf("Digest : %v\n", digest)
				t.Errorf("Digest for validation : %v\n", digestForValidation)
			*/
		}
		//t.Logf("\n")
		fmt.Println()
	}
}
