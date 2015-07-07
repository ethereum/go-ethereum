// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package crypto

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBox(t *testing.T) {
	prv1 := ToECDSA(common.Hex2Bytes("4b50fa71f5c3eeb8fdc452224b2395af2fcc3d125e06c32c82e048c0559db03f"))
	prv2 := ToECDSA(common.Hex2Bytes("d0b043b4c5d657670778242d82d68a29d25d7d711127d17b8e299f156dad361a"))
	pub2 := ToECDSAPub(common.Hex2Bytes("04bd27a63c91fe3233c5777e6d3d7b39204d398c8f92655947eb5a373d46e1688f022a1632d264725cbc7dc43ee1cfebde42fa0a86d08b55d2acfbb5e9b3b48dc5"))

	message := []byte("Hello, world.")
	ct, err := Encrypt(pub2, message)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	pt, err := Decrypt(prv2, ct)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !bytes.Equal(pt, message) {
		fmt.Println("ecies: plaintext doesn't match message")
		t.FailNow()
	}

	_, err = Decrypt(prv1, pt)
	if err == nil {
		fmt.Println("ecies: encryption should not have succeeded")
		t.FailNow()
	}

}
