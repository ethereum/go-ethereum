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

package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestRecoverSanity(t *testing.T) {
	msg, _ := hex.DecodeString("ce0677bb30baa8cf067c88db9811f4333d131bf8bcf12fe7065d211dce971008")
	sig, _ := hex.DecodeString("90f27b8b488db00b00606796d2987f6a5f59ae62ea05effe84fef5b8b0e549984a691139ad57a3f0b906637673aa2f63d1f55cb1a69199d4009eea23ceaddc9301")
	pubkey1, _ := hex.DecodeString("04e32df42865e97135acfb65f3bae71bdc86f4d49150ad6a440b6f15878109880a0a2b2667f7e725ceea70c673093bf67663e0312623c8e091b13cf2c0f11ef652")
	pubkey2, err := Ecrecover(msg, sig)
	if err != nil {
		t.Fatalf("recover error: %s", err)
	}
	if !bytes.Equal(pubkey1, pubkey2) {
		t.Errorf("pubkey mismatch: want: %x have: %x", pubkey1, pubkey2)
	}
}
