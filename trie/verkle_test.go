// Copyright 2021 go-ethereum Authors
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

package trie

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gballet/go-verkle"
)

func TestReproduceTree(t *testing.T) {
	presentKeys := [][]byte{
		common.Hex2Bytes("318dea512b6f3237a2d4763cf49bf26de3b617fb0cabe38a97807a5549df4d01"),
		common.Hex2Bytes("e6ed6c222e3985050b4fc574b136b0a42c63538e9ab970995cd418ba8e526400"),
		common.Hex2Bytes("18fb432d3b859ec3a1803854e8cceea75d092e52d0d4a4398d13022496745a02"),
		common.Hex2Bytes("318dea512b6f3237a2d4763cf49bf26de3b617fb0cabe38a97807a5549df4d02"),
		common.Hex2Bytes("18fb432d3b859ec3a1803854e8cceea75d092e52d0d4a4398d13022496745a04"),
		common.Hex2Bytes("e6ed6c222e3985050b4fc574b136b0a42c63538e9ab970995cd418ba8e526402"),
		common.Hex2Bytes("e6ed6c222e3985050b4fc574b136b0a42c63538e9ab970995cd418ba8e526403"),
		common.Hex2Bytes("18fb432d3b859ec3a1803854e8cceea75d092e52d0d4a4398d13022496745a00"),
		common.Hex2Bytes("18fb432d3b859ec3a1803854e8cceea75d092e52d0d4a4398d13022496745a03"),
		common.Hex2Bytes("e6ed6c222e3985050b4fc574b136b0a42c63538e9ab970995cd418ba8e526401"),
		common.Hex2Bytes("e6ed6c222e3985050b4fc574b136b0a42c63538e9ab970995cd418ba8e526404"),
		common.Hex2Bytes("318dea512b6f3237a2d4763cf49bf26de3b617fb0cabe38a97807a5549df4d00"),
		common.Hex2Bytes("18fb432d3b859ec3a1803854e8cceea75d092e52d0d4a4398d13022496745a01"),
	}

	absentKeys := [][]byte{
		common.Hex2Bytes("318dea512b6f3237a2d4763cf49bf26de3b617fb0cabe38a97807a5549df4d03"),
		common.Hex2Bytes("318dea512b6f3237a2d4763cf49bf26de3b617fb0cabe38a97807a5549df4d04"),
	}

	values := [][]byte{
		common.Hex2Bytes("320122e8584be00d000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0300000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
		common.Hex2Bytes("1bc176f2790c91e6000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("e703000000000000000000000000000000000000000000000000000000000000"),
	}

	root := verkle.New()
	kv := make(map[string][]byte)

	for i, key := range presentKeys {
		root.Insert(key, values[i], nil)
		kv[string(key)] = values[i]
	}

	proof, Cs, zis, yis := verkle.MakeVerkleMultiProof(root, append(presentKeys, absentKeys...), kv)
	cfg, _ := verkle.GetConfig()
	if !verkle.VerifyVerkleProof(proof, Cs, zis, yis, cfg) {
		t.Fatal("could not verify proof")
	}

	t.Log("commitments returned by proof:")
	for i, c := range Cs {
		t.Logf("%d %x", i, c.Bytes())
	}

	p, _, err := verkle.SerializeProof(proof)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("serialized: %x", p)
	t.Logf("tree: %s\n%x\n", verkle.ToDot(root), root.ComputeCommitment().Bytes())
}
