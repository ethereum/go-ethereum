// Copyright 2016 The go-ethereum Authors
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

package ens

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Tests for the decoding of the example ENS
func TestEIPSpecCidDecode(t *testing.T) {
	const (
		eipSpecHash = "e3010170122029f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f"
		eipHash     = "29f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f"
		dagPb       = 0x70
		sha2256     = 0x12
	)
	b, err := hex.DecodeString(eipSpecHash)
	if err != nil {
		t.Fatal(err)
	}
	hashBytes, err := hex.DecodeString(eipHash)

	if err != nil {
		t.Fatal(err)
	}

	storageNs, contentType, hashType, hashLength, decodedHashBytes, err := decodeEIP1577ContentHash(b)

	if err != nil {
		t.Fatal(err)
	}
	if storageNs != nsIpfs {
		t.Fatal("wrong ns")
	}
	if contentType != dagPb {
		t.Fatal("should be ipfs typecode")
	}
	if hashType != sha2256 {
		t.Fatal("should be sha2-256")
	}
	if hashLength != 32 {
		t.Fatal("should be 32")
	}
	if !bytes.Equal(hashBytes, decodedHashBytes) {
		t.Fatal("should be equal")
	}

}
func TestManualCidDecode(t *testing.T) {
	// call cid encode method with hash. expect byte slice returned, compare according to spec

	for _, v := range []struct {
		name        string
		headerBytes []byte
		wantErr     bool
	}{
		{
			name:        "values correct, should not fail",
			headerBytes: []byte{0xe4, 0x01, 0xfa, 0x1b, 0x20},
			wantErr:     false,
		},
		{
			name:        "cid version wrong, should fail",
			headerBytes: []byte{0xe4, 0x00, 0xfa, 0x1b, 0x20},
			wantErr:     true,
		},
		{
			name:        "hash length wrong, should fail",
			headerBytes: []byte{0xe4, 0x01, 0xfa, 0x1b, 0x1f},
			wantErr:     true,
		},
		{
			name:        "values correct for ipfs, should fail",
			headerBytes: []byte{0xe3, 0x01, 0x70, 0x12, 0x20},
			wantErr:     true,
		},
		{
			name:        "loose values for swarm, todo remove, should not fail",
			headerBytes: []byte{0xe4, 0x01, 0x70, 0x12, 0x20},
			wantErr:     false,
		},
		{
			name:        "loose values for swarm, todo remove, should not fail",
			headerBytes: []byte{0xe4, 0x01, 0x99, 0x99, 0x20},
			wantErr:     false,
		},
	} {
		t.Run(v.name, func(t *testing.T) {
			const eipHash = "29f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f"

			var bb []byte
			buf := make([]byte, binary.MaxVarintLen64)
			for _, vv := range v.headerBytes {
				n := binary.PutUvarint(buf, uint64(vv))
				bb = append(bb, buf[:n]...)
			}

			h := common.HexToHash(eipHash)
			bb = append(bb, h[:]...)
			str := hex.EncodeToString(bb)
			fmt.Println(str)
			decodedHash, e := extractContentHash(bb)
			switch v.wantErr {
			case true:
				if e == nil {
					t.Fatal("the decode should fail")
				}
			case false:
				if e != nil {
					t.Fatalf("the deccode shouldnt fail: %v", e)
				}
				if !bytes.Equal(decodedHash[:], h[:]) {
					t.Fatal("hashes not equal")
				}
			}
		})
	}
}

func TestManuelCidEncode(t *testing.T) {
	// call cid encode method with hash. expect byte slice returned, compare according to spec
	const eipHash = "29f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f"
	cidBytes, err := EncodeSwarmHash(common.HexToHash(eipHash))
	if err != nil {
		t.Fatal(err)
	}

	// logic in extractContentHash is unit tested thoroughly
	// hence we just check that the returned hash is equal
	h, err := extractContentHash(cidBytes)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(h[:], cidBytes) {
		t.Fatal("should be equal")
	}
}
