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
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie/utils"
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

	for i, key := range presentKeys {
		root.Insert(key, values[i], nil)
	}

	proof, Cs, zis, yis, _ := verkle.MakeVerkleMultiProof(root, append(presentKeys, absentKeys...))
	cfg := verkle.GetConfig()
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
	t.Logf("serialized: %v", p)
	t.Logf("tree: %s\n%x\n", verkle.ToDot(root), root.Commitment().Bytes())
}

func TestChunkifyCodeTestnet(t *testing.T) {
	code, _ := hex.DecodeString("6080604052348015600f57600080fd5b506004361060285760003560e01c806381ca91d314602d575b600080fd5b60336047565b604051603e9190605a565b60405180910390f35b60005481565b6054816073565b82525050565b6000602082019050606d6000830184604d565b92915050565b600081905091905056fea264697066735822122000382db0489577c1646ea2147a05f92f13f32336a32f1f82c6fb10b63e19f04064736f6c63430008070033")
	chunks := ChunkifyCode(code)
	if len(chunks) != 32*(len(code)/31+1) {
		t.Fatalf("invalid length %d != %d", len(chunks), 32*(len(code)/31+1))
	}
	if chunks[0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0])
	}
	t.Logf("%x\n", chunks[0])
	for i := 32; i < len(chunks); i += 32 {
		chunk := chunks[i : 32+i]
		if chunk[0] != 0 && i != 5*32 {
			t.Fatalf("invalid offset in chunk #%d %d != 0", i+1, chunk[0])
		}
		if i == 4 && chunk[0] != 12 {
			t.Fatalf("invalid offset in chunk #%d %d != 0", i+1, chunk[0])
		}
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code, _ = hex.DecodeString("608060405234801561001057600080fd5b506004361061002b5760003560e01c8063f566852414610030575b600080fd5b61003861004e565b6040516100459190610146565b60405180910390f35b6000600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166381ca91d36040518163ffffffff1660e01b815260040160206040518083038186803b1580156100b857600080fd5b505afa1580156100cc573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906100f0919061010a565b905090565b60008151905061010481610170565b92915050565b6000602082840312156101205761011f61016b565b5b600061012e848285016100f5565b91505092915050565b61014081610161565b82525050565b600060208201905061015b6000830184610137565b92915050565b6000819050919050565b600080fd5b61017981610161565b811461018457600080fd5b5056fea2646970667358221220d8add45a339f741a94b4fe7f22e101b560dc8a5874cbd957a884d8c9239df86264736f6c63430008070033")
	chunks = ChunkifyCode(code)
	if len(chunks) != 32*((len(code)+30)/31) {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0])
	}
	t.Logf("%x\n", chunks[0])
	expected := []byte{0, 1, 0, 13, 0, 0, 1, 0, 0, 0, 0, 0, 0, 3}
	for i := 32; i < len(chunks); i += 32 {
		chunk := chunks[i : 32+i]
		t.Log(i, i/32, chunk[0])
		if chunk[0] != expected[i/32-1] {
			t.Fatalf("invalid offset in chunk #%d %d != %d", i/32-1, chunk[0], expected[i/32-1])
		}
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code, _ = hex.DecodeString("6080604052348015600f57600080fd5b506004361060285760003560e01c8063ab5ed15014602d575b600080fd5b60336047565b604051603e9190605d565b60405180910390f35b60006001905090565b6057816076565b82525050565b6000602082019050607060008301846050565b92915050565b600081905091905056fea2646970667358221220163c79eab5630c3dbe22f7cc7692da08575198dda76698ae8ee2e3bfe62af3de64736f6c63430008070033")
	chunks = ChunkifyCode(code)
	if len(chunks) != 32*((len(code)+30)/31) {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0])
	}
	expected = []byte{0, 0, 0, 0, 13}
	for i := 32; i < len(chunks); i += 32 {
		chunk := chunks[i : 32+i]
		if chunk[0] != expected[i/32-1] {
			t.Fatalf("invalid offset in chunk #%d %d != %d", i/32-1, chunk[0], expected[i/32-1])
		}
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)
}

func TestChunkifyCodeSimple(t *testing.T) {
	code := []byte{
		0, PUSH4, 1, 2, 3, 4, PUSH3, 58, 68, 12, PUSH21, 1, 2, 3, 4, 5, 6,
		7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		// Second 31 bytes
		0, PUSH21, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
		PUSH7, 1, 2, 3, 4, 5, 6, 7,
		// Third 31 bytes
		PUSH30, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22,
		23, 24, 25, 26, 27, 28, 29, 30,
	}
	t.Logf("code=%x", code)
	chunks := ChunkifyCode(code)
	if len(chunks) != 96 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0])
	}
	if chunks[32] != 1 {
		t.Fatalf("invalid offset in second chunk %d != 1, chunk=%x", chunks[32], chunks[32:64])
	}
	if chunks[64] != 0 {
		t.Fatalf("invalid offset in third chunk %d != 0", chunks[64])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)
}

func TestChunkifyCodeFuzz(t *testing.T) {
	code := []byte{
		3, PUSH32, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
	}
	chunks := ChunkifyCode(code)
	if len(chunks) != 32 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code = []byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, PUSH32,
	}
	chunks = ChunkifyCode(code)
	if len(chunks) != 32 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code = []byte{
		PUSH4, PUSH32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	chunks = ChunkifyCode(code)
	if len(chunks) != 64 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0])
	}
	if chunks[32] != 0 {
		t.Fatalf("invalid offset in second chunk %d != 0, chunk=%x", chunks[32], chunks[32:64])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)

	code = []byte{
		PUSH4, PUSH32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	chunks = ChunkifyCode(code)
	if len(chunks) != 64 {
		t.Fatalf("invalid length %d", len(chunks))
	}
	if chunks[0] != 0 {
		t.Fatalf("invalid offset in first chunk %d != 0", chunks[0])
	}
	if chunks[32] != 0 {
		t.Fatalf("invalid offset in second chunk %d != 0, chunk=%x", chunks[32], chunks[32:64])
	}
	t.Logf("code=%x, chunks=%x\n", code, chunks)
}

// This test case checks what happens when two keys whose absence is being proven start with the
// same byte (0x0b in this case). Only one 'extension status' should be declared.
func TestReproduceCondrieuStemAggregationInProofOfAbsence(t *testing.T) {
	presentKeys := [][]byte{
		common.Hex2Bytes("6766d007d8fd90ea45b2ac9027ff04fa57e49527f11010a12a73f58ffa580800"),
		common.Hex2Bytes("6766d007d8fd90ea45b2ac9027ff04fa57e49527f11010a12a73f58ffa580801"),
		common.Hex2Bytes("6766d007d8fd90ea45b2ac9027ff04fa57e49527f11010a12a73f58ffa580802"),
		common.Hex2Bytes("6766d007d8fd90ea45b2ac9027ff04fa57e49527f11010a12a73f58ffa580803"),
		common.Hex2Bytes("6766d007d8fd90ea45b2ac9027ff04fa57e49527f11010a12a73f58ffa580804"),
		common.Hex2Bytes("9f2a59ea98d7cb610eff49447571e1610188937ce9266c6b4ded1b6ee37ecd00"),
		common.Hex2Bytes("9f2a59ea98d7cb610eff49447571e1610188937ce9266c6b4ded1b6ee37ecd01"),
		common.Hex2Bytes("9f2a59ea98d7cb610eff49447571e1610188937ce9266c6b4ded1b6ee37ecd02"),
		common.Hex2Bytes("9f2a59ea98d7cb610eff49447571e1610188937ce9266c6b4ded1b6ee37ecd03"),
	}

	absentKeys := [][]byte{
		common.Hex2Bytes("089783b59ef47adbdf85546c92d9b93ffd2f4803093ee93727bb42a1537dfb00"),
		common.Hex2Bytes("089783b59ef47adbdf85546c92d9b93ffd2f4803093ee93727bb42a1537dfb01"),
		common.Hex2Bytes("089783b59ef47adbdf85546c92d9b93ffd2f4803093ee93727bb42a1537dfb02"),
		common.Hex2Bytes("089783b59ef47adbdf85546c92d9b93ffd2f4803093ee93727bb42a1537dfb03"),
		common.Hex2Bytes("089783b59ef47adbdf85546c92d9b93ffd2f4803093ee93727bb42a1537dfb04"),
		common.Hex2Bytes("0b373ba3992dde5cfee854e1a786559ba0b6a13d376550c1ed58c00dc9706f00"),
		common.Hex2Bytes("0b373ba3992dde5cfee854e1a786559ba0b6a13d376550c1ed58c00dc9706f01"),
		common.Hex2Bytes("0b373ba3992dde5cfee854e1a786559ba0b6a13d376550c1ed58c00dc9706f02"),
		common.Hex2Bytes("0b373ba3992dde5cfee854e1a786559ba0b6a13d376550c1ed58c00dc9706f03"),
		common.Hex2Bytes("0b373ba3992dde5cfee854e1a786559ba0b6a13d376550c1ed58c00dc9706f04"),
		common.Hex2Bytes("0b373ba3992dde5cfee854e1a786559ba0b6a13d376550c1ed58c00dc9706f80"),
		common.Hex2Bytes("0b373ba3992dde5cfee854e1a786559ba0b6a13d376550c1ed58c00dc9706f81"),
		common.Hex2Bytes("0b373ba3992dde5cfee854e1a786559ba0b6a13d376550c1ed58c00dc9706f82"),
		common.Hex2Bytes("0b373ba3992dde5cfee854e1a786559ba0b6a13d376550c1ed58c00dc9706f83"),
		common.Hex2Bytes("0bb7fda24b2ea0de0f791b27f8a040fcc79f8e1e2dfe50443bc632543ba5e700"),
		common.Hex2Bytes("0bb7fda24b2ea0de0f791b27f8a040fcc79f8e1e2dfe50443bc632543ba5e702"),
		common.Hex2Bytes("0bb7fda24b2ea0de0f791b27f8a040fcc79f8e1e2dfe50443bc632543ba5e703"),
		common.Hex2Bytes("3aeba70b6afb762af4a507c8ec10747479d797c6ec11c14f92b5699634bd18d4"),
	}

	values := [][]byte{
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("53bfa56cfcaddf191e0200000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0700000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("389a890a6ce3e618843300000000000000000000000000000000000000000000"),
		common.Hex2Bytes("0200000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
	}

	root := verkle.New()

	for i, key := range presentKeys {
		root.Insert(key, values[i], nil)
	}

	proof, Cs, zis, yis, _ := verkle.MakeVerkleMultiProof(root, append(presentKeys, absentKeys...))
	cfg := verkle.GetConfig()
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
	t.Logf("serialized: %p", p)
	t.Logf("tree: %s\n%x\n", verkle.ToDot(root), root.Commitment().Bytes())

	t.Logf("%d", len(proof.ExtStatus))
	if len(proof.ExtStatus) != 5 {
		t.Fatalf("invalid number of declared stems: %d != 5", len(proof.ExtStatus))
	}
}

// Cover the case in which a stem is both used for a proof of absence, and for a proof of presence.
func TestReproduceCondrieuPoAStemConflictWithAnotherStem(t *testing.T) {
	presentKeys := [][]byte{
		common.Hex2Bytes("6766d007d8fd90ea45b2ac9027ff04fa57e49527f11010a12a73f58ffa580800"),
	}

	absentKeys := [][]byte{
		common.Hex2Bytes("6766d007d8fd90ea45b2ac9027ff04fa57e49527f11010a12a73008ffa580800"),
		// the key differs from the key present...                            ^^ here
	}

	values := [][]byte{
		common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
	}

	root := verkle.New()

	for i, key := range presentKeys {
		root.Insert(key, values[i], nil)
	}

	proof, Cs, zis, yis, _ := verkle.MakeVerkleMultiProof(root, append(presentKeys, absentKeys...))
	cfg := verkle.GetConfig()
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
	t.Logf("serialized: %p", p)
	t.Logf("tree: %s\n%x\n", verkle.ToDot(root), root.Commitment().Bytes())

	t.Logf("%d", len(proof.ExtStatus))
	if len(proof.PoaStems) != 0 {
		t.Fatal("a proof-of-absence stem was declared, when there was no need")
	}
}

func TestEmptyKeySetInProveAndSerialize(t *testing.T) {
	tree := verkle.New()
	verkle.MakeVerkleMultiProof(tree, [][]byte{})
}

func TestGetTreeKeys(t *testing.T) {
	addr := common.Hex2Bytes("71562b71999873DB5b286dF957af199Ec94617f7")
	target := common.Hex2Bytes("274cde18dd9dbb04caf16ad5ee969c19fe6ca764d5688b5e1d419f4ac6cd1600")
	key := utils.GetTreeKeyVersion(addr)
	t.Logf("key=%x", key)
	t.Logf("actualKey=%x", target)
	if !bytes.Equal(key, target) {
		t.Fatalf("differing output %x != %x", key, target)
	}
}
