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

		dag_pb   = 0x70
		sha2_256 = 0x12
	)

	b, err := hex.DecodeString(eipSpecHash)
	if err != nil {
		t.Fatal(err)
	}
	hashBytes, err := hex.DecodeString(eipHash)

	if err != nil {
		t.Fatal(err)
	}

	storageNs, contentType, hashType, hashLength, hashBytes, err := decodeEIP1577ContentHash(b)

	if err != nil {
		t.Fatal(err)
	}
	if storageNs != ns_ipfs {
		t.Fatal("wrong ns")
	}
	if contentType != dag_pb {
		t.Fatal("should be swarm typecode")
	}
	if hashType != sha2_256 {
		t.Fatal("should be sha2-256")
	}
	if hashLength != 32 {
		t.Fatal("should be 32")
	}
	if !bytes.Equal(hashBytes, hashBytes) {
		t.Fatal("should be equal")
	}

}
func TestManualCidDecode(t *testing.T) {
	// call cid encode method with hash. expect byte slice returned, compare according to spec
	bb := []byte{}

	for _, v := range []struct {
		name        string
		headerBytes []byte
		fails       bool
	}{
		{
			name:        "values correct, should not fail",
			headerBytes: []byte{0xe4, 0x01, 0x99, 0x1b, 0x20},
			fails:       false,
		},
		{
			name:        "cid version wrong, should fail",
			headerBytes: []byte{0xe4, 0x00, 0x99, 0x1b, 0x20},
			fails:       true,
		},
		{
			name:        "hash length wrong, should fail",
			headerBytes: []byte{0xe4, 0x01, 0x99, 0x1b, 0x1f},
			fails:       true,
		},
		{
			name:        "values correct for ipfs, should fail",
			headerBytes: []byte{0xe3, 0x01, 0x99, 0x1b, 0x20},
			fails:       true,
		},
	} {
		t.Run(v.name, func(t *testing.T) {
			buf := make([]byte, binary.MaxVarintLen64)
			for _, vv := range v.headerBytes {
				n := binary.PutUvarint(buf, uint64(vv))
				bb = append(bb, buf[:n]...)
			}

			h := common.HexToHash("29f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f")
			bb = append(bb, h[:]...)
			str := hex.EncodeToString(bb)
			fmt.Println(str)
			decodedHash, e := extractContentHash(bb)
			switch v.fails {
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

	/* from the EIP documentation
	   storage system: Swarm (0xe4)
	   CID version: 1 (0x01)
	   content type: swarm-manifest (0x??)
	   hash function: keccak-256 (0x1B)
	   hash length: 32 bytes (0x20)
	   hash: 29f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f
	*/

}

func TestManuelCidEncode(t *testing.T) {
	// call cid encode method with hash. expect byte slice returned, compare according to spec

	/* from the EIP documentation
	   storage system: Swarm (0xe4)
	   CID version: 1 (0x01)
	   content type: swarm-manifest (0x??)
	   hash function: keccak-256 (0x1B)
	   hash length: 32 bytes (0x20)
	   hash: 29f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f
	*/

}

/*
func TestCIDSanity(t *testing.T) {
	hashStr := "d1de9994b4d039f6548d191eb26786769f580809256b4685ef316805265ea162"
	hash := common.HexToHash(hashStr) //this always yields a 32 byte long hash
	cc, err := encodeCid(hash)
	if err != nil {
		t.Fatal(err)
	}

	if cc.Prefix().MhLength != 32 {
		t.Fatal("w00t")
	}
	decoded, err := mh.Decode(cc.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Length != 32 {
		t.Fatal("invalid length")
	}
	if !bytes.Equal(decoded.Digest, hash[:]) {
		t.Fatalf("hashes not equal")
	}

	if decoded.Length != 32 {
		t.Fatal("wrong length")
	}
	fmt.Println(cc.StringOfBase(multibase.Base16))

	bbbb, e := cc.StringOfBase(multibase.Base16)
	if e != nil {
		t.Fatal(e)
	}
	fmt.Println(bbbb)
	//create the CID string artificially
	hashStr = "f01551b20" + hashStr

	c, err := cid.Decode(hashStr)
	if err != nil {
		t.Fatalf("Error decoding CID: %v", err)
	}

	fmt.Sprintf("Got CID: %v", c)
	fmt.Println("Got CID:", c.Prefix())

}*/
