// Copyright 2020 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

func TestStackTrieInsertAndHash(t *testing.T) {
	type KeyValueHash struct {
		K string // Hex string for key.
		V string // Value, directly converted to bytes.
		H string // Expected root hash after insert of (K, V) to an existing trie.
	}
	tests := [][]KeyValueHash{
		{ // {0:0, 7:0, f:0}
			{"00", "v_______________________0___0", "5cb26357b95bb9af08475be00243ceb68ade0b66b5cd816b0c18a18c612d2d21"},
			{"70", "v_______________________0___1", "8ff64309574f7a437a7ad1628e690eb7663cfde10676f8a904a8c8291dbc1603"},
			{"f0", "v_______________________0___2", "9e3a01bd8d43efb8e9d4b5506648150b8e3ed1caea596f84ee28e01a72635470"},
		},
		{ // {1:0cc, e:{1:fc, e:fc}}
			{"10cc", "v_______________________1___0", "233e9b257843f3dfdb1cce6676cdaf9e595ac96ee1b55031434d852bc7ac9185"},
			{"e1fc", "v_______________________1___1", "39c5e908ae83d0c78520c7c7bda0b3782daf594700e44546e93def8f049cca95"},
			{"eefc", "v_______________________1___2", "d789567559fd76fe5b7d9cc42f3750f942502ac1c7f2a466e2f690ec4b6c2a7c"},
		},
		{ // {b:{a:ac, b:ac}, d:acc}
			{"baac", "v_______________________2___0", "8be1c86ba7ec4c61e14c1a9b75055e0464c2633ae66a055a24e75450156a5d42"},
			{"bbac", "v_______________________2___1", "8495159b9895a7d88d973171d737c0aace6fe6ac02a4769fff1bc43bcccce4cc"},
			{"dacc", "v_______________________2___2", "9bcfc5b220a27328deb9dc6ee2e3d46c9ebc9c69e78acda1fa2c7040602c63ca"},
		},
		{ // {0:0cccc, 2:456{0:0, 2:2}
			{"00cccc", "v_______________________3___0", "e57dc2785b99ce9205080cb41b32ebea7ac3e158952b44c87d186e6d190a6530"},
			{"245600", "v_______________________3___1", "0335354adbd360a45c1871a842452287721b64b4234dfe08760b243523c998db"},
			{"245622", "v_______________________3___2", "9e6832db0dca2b5cf81c0e0727bfde6afc39d5de33e5720bccacc183c162104e"},
		},
		{ // {1:4567{1:1c, 3:3c}, 3:0cccccc}
			{"1456711c", "v_______________________4___0", "f2389e78d98fed99f3e63d6d1623c1d4d9e8c91cb1d585de81fbc7c0e60d3529"},
			{"1456733c", "v_______________________4___1", "101189b3fab852be97a0120c03d95eefcf984d3ed639f2328527de6def55a9c0"},
			{"30cccccc", "v_______________________4___2", "3780ce111f98d15751dfde1eb21080efc7d3914b429e5c84c64db637c55405b3"},
		},
		{ // 8800{1:f, 2:e, 3:d}
			{"88001f", "v_______________________5___0", "e817db50d84f341d443c6f6593cafda093fc85e773a762421d47daa6ac993bd5"},
			{"88002e", "v_______________________5___1", "d6e3e6047bdc110edd296a4d63c030aec451bee9d8075bc5a198eee8cda34f68"},
			{"88003d", "v_______________________5___2", "b6bdf8298c703342188e5f7f84921a402042d0e5fb059969dd53a6b6b1fb989e"},
		},
		{ // 0{1:fc, 2:ec, 4:dc}
			{"01fc", "v_______________________6___0", "693268f2ca80d32b015f61cd2c4dba5a47a6b52a14c34f8e6945fad684e7a0d5"},
			{"02ec", "v_______________________6___1", "e24ddd44469310c2b785a2044618874bf486d2f7822603a9b8dce58d6524d5de"},
			{"04dc", "v_______________________6___2", "33fc259629187bbe54b92f82f0cd8083b91a12e41a9456b84fc155321e334db7"},
		},
		{ // f{0:fccc, f:ff{0:f, f:f}}
			{"f0fccc", "v_______________________7___0", "b0966b5aa469a3e292bc5fcfa6c396ae7a657255eef552ea7e12f996de795b90"},
			{"ffff0f", "v_______________________7___1", "3b1ca154ec2a3d96d8d77bddef0abfe40a53a64eb03cecf78da9ec43799fa3d0"},
			{"ffffff", "v_______________________7___2", "e75463041f1be8252781be0ace579a44ea4387bf5b2739f4607af676f7719678"},
		},
		{ // ff{0:f{0:f, f:f}, f:fcc}
			{"ff0f0f", "v_______________________8___0", "0928af9b14718ec8262ab89df430f1e5fbf66fac0fed037aff2b6767ae8c8684"},
			{"ff0fff", "v_______________________8___1", "d870f4d3ce26b0bf86912810a1960693630c20a48ba56be0ad04bc3e9ddb01e6"},
			{"ffffcc", "v_______________________8___2", "4239f10dd9d9915ecf2e047d6a576bdc1733ed77a30830f1bf29deaf7d8e966f"},
		},
		{
			{"123d", "x___________________________0", "fc453d88b6f128a77c448669710497380fa4588abbea9f78f4c20c80daa797d0"},
			{"123e", "x___________________________1", "5af48f2d8a9a015c1ff7fa8b8c7f6b676233bd320e8fb57fd7933622badd2cec"},
			{"123f", "x___________________________2", "1164d7299964e74ac40d761f9189b2a3987fae959800d0f7e29d3aaf3eae9e15"},
		},
		{
			{"123d", "x___________________________0", "fc453d88b6f128a77c448669710497380fa4588abbea9f78f4c20c80daa797d0"},
			{"123e", "x___________________________1", "5af48f2d8a9a015c1ff7fa8b8c7f6b676233bd320e8fb57fd7933622badd2cec"},
			{"124a", "x___________________________2", "661a96a669869d76b7231380da0649d013301425fbea9d5c5fae6405aa31cfce"},
		},
		{
			{"123d", "x___________________________0", "fc453d88b6f128a77c448669710497380fa4588abbea9f78f4c20c80daa797d0"},
			{"123e", "x___________________________1", "5af48f2d8a9a015c1ff7fa8b8c7f6b676233bd320e8fb57fd7933622badd2cec"},
			{"13aa", "x___________________________2", "6590120e1fd3ffd1a90e8de5bb10750b61079bb0776cca4414dd79a24e4d4356"},
		},
		{
			{"123d", "x___________________________0", "fc453d88b6f128a77c448669710497380fa4588abbea9f78f4c20c80daa797d0"},
			{"123e", "x___________________________1", "5af48f2d8a9a015c1ff7fa8b8c7f6b676233bd320e8fb57fd7933622badd2cec"},
			{"2aaa", "x___________________________2", "f869b40e0c55eace1918332ef91563616fbf0755e2b946119679f7ef8e44b514"},
		},
		{
			{"1234da", "x___________________________0", "1c4b4462e9f56a80ca0f5d77c0d632c41b0102290930343cf1791e971a045a79"},
			{"1234ea", "x___________________________1", "2f502917f3ba7d328c21c8b45ee0f160652e68450332c166d4ad02d1afe31862"},
			{"1234fa", "x___________________________2", "4f4e368ab367090d5bc3dbf25f7729f8bd60df84de309b4633a6b69ab66142c0"},
		},
		{
			{"1234da", "x___________________________0", "1c4b4462e9f56a80ca0f5d77c0d632c41b0102290930343cf1791e971a045a79"},
			{"1234ea", "x___________________________1", "2f502917f3ba7d328c21c8b45ee0f160652e68450332c166d4ad02d1afe31862"},
			{"1235aa", "x___________________________2", "21840121d11a91ac8bbad9a5d06af902a5c8d56a47b85600ba813814b7bfcb9b"},
		},
		{
			{"1234da", "x___________________________0", "1c4b4462e9f56a80ca0f5d77c0d632c41b0102290930343cf1791e971a045a79"},
			{"1234ea", "x___________________________1", "2f502917f3ba7d328c21c8b45ee0f160652e68450332c166d4ad02d1afe31862"},
			{"124aaa", "x___________________________2", "ea4040ddf6ae3fbd1524bdec19c0ab1581015996262006632027fa5cf21e441e"},
		},
		{
			{"1234da", "x___________________________0", "1c4b4462e9f56a80ca0f5d77c0d632c41b0102290930343cf1791e971a045a79"},
			{"1234ea", "x___________________________1", "2f502917f3ba7d328c21c8b45ee0f160652e68450332c166d4ad02d1afe31862"},
			{"13aaaa", "x___________________________2", "e4beb66c67e44f2dd8ba36036e45a44ff68f8d52942472b1911a45f886a34507"},
		},
		{
			{"1234da", "x___________________________0", "1c4b4462e9f56a80ca0f5d77c0d632c41b0102290930343cf1791e971a045a79"},
			{"1234ea", "x___________________________1", "2f502917f3ba7d328c21c8b45ee0f160652e68450332c166d4ad02d1afe31862"},
			{"2aaaaa", "x___________________________2", "5f5989b820ff5d76b7d49e77bb64f26602294f6c42a1a3becc669cd9e0dc8ec9"},
		},
		{
			{"000000", "x___________________________0", "3b32b7af0bddc7940e7364ee18b5a59702c1825e469452c8483b9c4e0218b55a"},
			{"1234da", "x___________________________1", "3ab152a1285dca31945566f872c1cc2f17a770440eda32aeee46a5e91033dde2"},
			{"1234ea", "x___________________________2", "0cccc87f96ddef55563c1b3be3c64fff6a644333c3d9cd99852cb53b6412b9b8"},
			{"1234fa", "x___________________________3", "65bb3aafea8121111d693ffe34881c14d27b128fd113fa120961f251fe28428d"},
		},
		{
			{"000000", "x___________________________0", "3b32b7af0bddc7940e7364ee18b5a59702c1825e469452c8483b9c4e0218b55a"},
			{"1234da", "x___________________________1", "3ab152a1285dca31945566f872c1cc2f17a770440eda32aeee46a5e91033dde2"},
			{"1234ea", "x___________________________2", "0cccc87f96ddef55563c1b3be3c64fff6a644333c3d9cd99852cb53b6412b9b8"},
			{"1235aa", "x___________________________3", "f670e4d2547c533c5f21e0045442e2ecb733f347ad6d29ef36e0f5ba31bb11a8"},
		},
		{
			{"000000", "x___________________________0", "3b32b7af0bddc7940e7364ee18b5a59702c1825e469452c8483b9c4e0218b55a"},
			{"1234da", "x___________________________1", "3ab152a1285dca31945566f872c1cc2f17a770440eda32aeee46a5e91033dde2"},
			{"1234ea", "x___________________________2", "0cccc87f96ddef55563c1b3be3c64fff6a644333c3d9cd99852cb53b6412b9b8"},
			{"124aaa", "x___________________________3", "c17464123050a9a6f29b5574bb2f92f6d305c1794976b475b7fb0316b6335598"},
		},
		{
			{"000000", "x___________________________0", "3b32b7af0bddc7940e7364ee18b5a59702c1825e469452c8483b9c4e0218b55a"},
			{"1234da", "x___________________________1", "3ab152a1285dca31945566f872c1cc2f17a770440eda32aeee46a5e91033dde2"},
			{"1234ea", "x___________________________2", "0cccc87f96ddef55563c1b3be3c64fff6a644333c3d9cd99852cb53b6412b9b8"},
			{"13aaaa", "x___________________________3", "aa8301be8cb52ea5cd249f5feb79fb4315ee8de2140c604033f4b3fff78f0105"},
		},
		{
			{"0000", "x___________________________0", "cb8c09ad07ae882136f602b3f21f8733a9f5a78f1d2525a8d24d1c13258000b2"},
			{"123d", "x___________________________1", "8f09663deb02f08958136410dc48565e077f76bb6c9d8c84d35fc8913a657d31"},
			{"123e", "x___________________________2", "0d230561e398c579e09a9f7b69ceaf7d3970f5a436fdb28b68b7a37c5bdd6b80"},
			{"123f", "x___________________________3", "80f7bad1893ca57e3443bb3305a517723a74d3ba831bcaca22a170645eb7aafb"},
		},
		{
			{"0000", "x___________________________0", "cb8c09ad07ae882136f602b3f21f8733a9f5a78f1d2525a8d24d1c13258000b2"},
			{"123d", "x___________________________1", "8f09663deb02f08958136410dc48565e077f76bb6c9d8c84d35fc8913a657d31"},
			{"123e", "x___________________________2", "0d230561e398c579e09a9f7b69ceaf7d3970f5a436fdb28b68b7a37c5bdd6b80"},
			{"124a", "x___________________________3", "383bc1bb4f019e6bc4da3751509ea709b58dd1ac46081670834bae072f3e9557"},
		},
		{
			{"0000", "x___________________________0", "cb8c09ad07ae882136f602b3f21f8733a9f5a78f1d2525a8d24d1c13258000b2"},
			{"123d", "x___________________________1", "8f09663deb02f08958136410dc48565e077f76bb6c9d8c84d35fc8913a657d31"},
			{"123e", "x___________________________2", "0d230561e398c579e09a9f7b69ceaf7d3970f5a436fdb28b68b7a37c5bdd6b80"},
			{"13aa", "x___________________________3", "ff0dc70ce2e5db90ee42a4c2ad12139596b890e90eb4e16526ab38fa465b35cf"},
		},
	}
	st := NewStackTrie(nil)
	for i, test := range tests {
		// The StackTrie does not allow Insert(), Hash(), Insert(), ...
		// so we will create new trie for every sequence length of inserts.
		for l := 1; l <= len(test); l++ {
			st.Reset()
			for j := 0; j < l; j++ {
				kv := &test[j]
				if err := st.TryUpdate(common.FromHex(kv.K), []byte(kv.V)); err != nil {
					t.Fatal(err)
				}
			}
			expected := common.HexToHash(test[l-1].H)
			if h := st.Hash(); h != expected {
				t.Errorf("%d(%d): root hash mismatch: %x, expected %x", i, l, h, expected)
			}
		}
	}
}

func TestSizeBug(t *testing.T) {
	st := NewStackTrie(nil)
	nt := NewEmpty(NewDatabase(memorydb.New()))

	leaf := common.FromHex("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	value := common.FromHex("94cf40d0d2b44f2b66e07cace1372ca42b73cf21a3")

	nt.TryUpdate(leaf, value)
	st.TryUpdate(leaf, value)

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

func TestEmptyBug(t *testing.T) {
	st := NewStackTrie(nil)
	nt := NewEmpty(NewDatabase(memorydb.New()))

	//leaf := common.FromHex("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	//value := common.FromHex("94cf40d0d2b44f2b66e07cace1372ca42b73cf21a3")
	kvs := []struct {
		K string
		V string
	}{
		{K: "405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace", V: "9496f4ec2bf9dab484cac6be589e8417d84781be08"},
		{K: "40edb63a35fcf86c08022722aa3287cdd36440d671b4918131b2514795fefa9c", V: "01"},
		{K: "b10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6", V: "947a30f7736e48d6599356464ba4c150d8da0302ff"},
		{K: "c2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b", V: "02"},
	}

	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

func TestValLength56(t *testing.T) {
	st := NewStackTrie(nil)
	nt := NewEmpty(NewDatabase(memorydb.New()))

	//leaf := common.FromHex("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	//value := common.FromHex("94cf40d0d2b44f2b66e07cace1372ca42b73cf21a3")
	kvs := []struct {
		K string
		V string
	}{
		{K: "405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace", V: "1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"},
	}

	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

// TestUpdateSmallNodes tests a case where the leaves are small (both key and value),
// which causes a lot of node-within-node. This case was found via fuzzing.
func TestUpdateSmallNodes(t *testing.T) {
	st := NewStackTrie(nil)
	nt := NewEmpty(NewDatabase(memorydb.New()))

	kvs := []struct {
		K string
		V string
	}{
		{"63303030", "3041"}, // stacktrie.Update
		{"65", "3000"},       // stacktrie.Update
	}
	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}
	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

// TestUpdateVariableKeys contains a case which stacktrie fails: when keys of different
// sizes are used, and the second one has the same prefix as the first, then the
// stacktrie fails, since it's unable to 'expand' on an already added leaf.
// For all practical purposes, this is fine, since keys are fixed-size length
// in account and storage tries.
//
// The test is marked as 'skipped', and exists just to have the behaviour documented.
// This case was found via fuzzing.
func TestUpdateVariableKeys(t *testing.T) {
	t.SkipNow()
	st := NewStackTrie(nil)
	nt := NewEmpty(NewDatabase(memorydb.New()))

	kvs := []struct {
		K string
		V string
	}{
		{"0x33303534636532393561313031676174", "303030"},
		{"0x3330353463653239356131303167617430", "313131"},
	}
	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}
	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

// TestStacktrieNotModifyValues checks that inserting blobs of data into the
// stacktrie does not mutate the blobs
func TestStacktrieNotModifyValues(t *testing.T) {
	st := NewStackTrie(nil)
	{ // Test a very small trie
		// Give it the value as a slice with large backing alloc,
		// so if the stacktrie tries to append, it won't have to realloc
		value := make([]byte, 1, 100)
		value[0] = 0x2
		want := common.CopyBytes(value)
		st.TryUpdate([]byte{0x01}, value)
		st.Hash()
		if have := value; !bytes.Equal(have, want) {
			t.Fatalf("tiny trie: have %#x want %#x", have, want)
		}
		st = NewStackTrie(nil)
	}
	// Test with a larger trie
	keyB := big.NewInt(1)
	keyDelta := big.NewInt(1)
	var vals [][]byte
	getValue := func(i int) []byte {
		if i%2 == 0 { // large
			return crypto.Keccak256(big.NewInt(int64(i)).Bytes())
		} else { //small
			return big.NewInt(int64(i)).Bytes()
		}
	}
	for i := 0; i < 1000; i++ {
		key := common.BigToHash(keyB)
		value := getValue(i)
		st.TryUpdate(key.Bytes(), value)
		vals = append(vals, value)
		keyB = keyB.Add(keyB, keyDelta)
		keyDelta.Add(keyDelta, common.Big1)
	}
	st.Hash()
	for i := 0; i < 1000; i++ {
		want := getValue(i)

		have := vals[i]
		if !bytes.Equal(have, want) {
			t.Fatalf("item %d, have %#x want %#x", i, have, want)
		}
	}
}

// TestStacktrieSerialization tests that the stacktrie works well if we
// serialize/unserialize it a lot
func TestStacktrieSerialization(t *testing.T) {
	var (
		st       = NewStackTrie(nil)
		nt       = NewEmpty(NewDatabase(memorydb.New()))
		keyB     = big.NewInt(1)
		keyDelta = big.NewInt(1)
		vals     [][]byte
		keys     [][]byte
	)
	getValue := func(i int) []byte {
		if i%2 == 0 { // large
			return crypto.Keccak256(big.NewInt(int64(i)).Bytes())
		} else { //small
			return big.NewInt(int64(i)).Bytes()
		}
	}
	for i := 0; i < 10; i++ {
		vals = append(vals, getValue(i))
		keys = append(keys, common.BigToHash(keyB).Bytes())
		keyB = keyB.Add(keyB, keyDelta)
		keyDelta.Add(keyDelta, common.Big1)
	}
	for i, k := range keys {
		nt.TryUpdate(k, common.CopyBytes(vals[i]))
	}

	for i, k := range keys {
		blob, err := st.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		newSt, err := NewFromBinary(blob, nil)
		if err != nil {
			t.Fatal(err)
		}
		st = newSt
		st.TryUpdate(k, common.CopyBytes(vals[i]))
	}
	if have, want := st.Hash(), nt.Hash(); have != want {
		t.Fatalf("have %#x want %#x", have, want)
	}
}
