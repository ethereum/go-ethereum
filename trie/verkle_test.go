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
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
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

func TestChunkifyCodeTestnet(t *testing.T) {
	code, _ := hex.DecodeString("6080604052348015600f57600080fd5b506004361060285760003560e01c806381ca91d314602d575b600080fd5b60336047565b604051603e9190605a565b60405180910390f35b60005481565b6054816073565b82525050565b6000602082019050606d6000830184604d565b92915050565b600081905091905056fea264697066735822122000382db0489577c1646ea2147a05f92f13f32336a32f1f82c6fb10b63e19f04064736f6c63430008070033")
	chunks, err := ChunkifyCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != (len(code)+30)/31 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0][0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0][0])
	}
	t.Logf("%x\n", chunks[0])
	for i, chunk := range chunks[1:] {
		if chunk[0] != 0 && i != 4 {
			t.Fatalf("invalid offset in chunk #%d %d != 0", i+1, chunk[0])
		}
		if i == 4 && chunk[0] != 12 {
			t.Fatalf("invalid offset in chunk #%d %d != 0", i+1, chunk[0])
		}
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code, _ = hex.DecodeString("608060405234801561001057600080fd5b506004361061002b5760003560e01c8063f566852414610030575b600080fd5b61003861004e565b6040516100459190610146565b60405180910390f35b6000600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166381ca91d36040518163ffffffff1660e01b815260040160206040518083038186803b1580156100b857600080fd5b505afa1580156100cc573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906100f0919061010a565b905090565b60008151905061010481610170565b92915050565b6000602082840312156101205761011f61016b565b5b600061012e848285016100f5565b91505092915050565b61014081610161565b82525050565b600060208201905061015b6000830184610137565b92915050565b6000819050919050565b600080fd5b61017981610161565b811461018457600080fd5b5056fea2646970667358221220d8add45a339f741a94b4fe7f22e101b560dc8a5874cbd957a884d8c9239df86264736f6c63430008070033")
	chunks, err = ChunkifyCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != (len(code)+30)/31 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0][0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0][0])
	}
	t.Logf("%x\n", chunks[0])
	expected := []byte{0, 1, 0, 13, 0, 0, 1, 0, 0, 0, 0, 0, 0, 3}
	for i, chunk := range chunks[1:] {
		if chunk[0] != expected[i] {
			t.Fatalf("invalid offset in chunk #%d %d != %d", i+1, chunk[0], expected[i])
		}
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code, _ = hex.DecodeString("6080604052348015600f57600080fd5b506004361060285760003560e01c8063ab5ed15014602d575b600080fd5b60336047565b604051603e9190605d565b60405180910390f35b60006001905090565b6057816076565b82525050565b6000602082019050607060008301846050565b92915050565b600081905091905056fea2646970667358221220163c79eab5630c3dbe22f7cc7692da08575198dda76698ae8ee2e3bfe62af3de64736f6c63430008070033")
	chunks, err = ChunkifyCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != (len(code)+30)/31 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0][0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0][0])
	}
	expected = []byte{0, 0, 0, 0, 13}
	for i, chunk := range chunks[1:] {
		if chunk[0] != expected[i] {
			t.Fatalf("invalid offset in chunk #%d %d != %d", i+1, chunk[0], expected[i])
		}
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)
}

func TestChunkifyCodeSimple(t *testing.T) {
	code := []byte{
		0, byte(vm.PUSH4), 1, 2, 3, 4, byte(vm.PUSH3), 58, 68, 12, byte(vm.PUSH21), 1, 2, 3, 4, 5, 6,
		7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		// Second 31 bytes
		0, byte(vm.PUSH21), 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
		byte(vm.PUSH7), 1, 2, 3, 4, 5, 6, 7,
		// Third 31 bytes
		byte(vm.PUSH30), 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22,
		23, 24, 25, 26, 27, 28, 29, 30,
	}
	t.Logf("code=%x", code)
	chunks, err := ChunkifyCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 3 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0][0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0][0])
	}
	if chunks[1][0] != 1 {
		t.Fatalf("invalid offset in second chunk %d != 1, chunk=%x", chunks[1][0], chunks[1])
	}
	if chunks[2][0] != 0 {
		t.Fatalf("invalid offset in third chunk %d != 0", chunks[2][0])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)
}

func TestChunkifyCodeFuzz(t *testing.T) {
	code := []byte{
		3, PUSH32, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
	}
	chunks, err := ChunkifyCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0][0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0][0])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code = []byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, PUSH32,
	}
	chunks, err = ChunkifyCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0][0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0][0])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code = []byte{
		byte(vm.PUSH4), PUSH32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	chunks, err = ChunkifyCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0][0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0][0])
	}
	if chunks[1][0] != 0 {
		t.Fatalf("invalid offset in second chunk %d != 0, chunk=%x", chunks[1][0], chunks[1])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code = []byte{
		byte(vm.PUSH4), PUSH32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	chunks, err = ChunkifyCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0][0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0][0])
	}
	if chunks[1][0] != 0 {
		t.Fatalf("invalid offset in second chunk %d != 0, chunk=%x", chunks[1][0], chunks[1])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)
}
