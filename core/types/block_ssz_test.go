// Copyright 2026 The go-ethereum Authors
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

package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/ssz"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// emptyListRoot is the root of an empty ProgressiveList: mix_in_length(0, 0).
const emptyListRoot = "0xf5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b"

// patternedPayload builds a payload from five parameters, the way the
// fellowship's vector generator does: every scalar is a fill byte or an
// offset of s, transaction i is the byte i+1 repeated (37i+5) % 200 times,
// withdrawal i the byte 0x80+i repeated 44 times, the block access list
// counts bytes modulo 251, and the extra data is 0xab repeated.
func patternedPayload(s byte, ntx, nwd, nbal, nextra int) *ExecutionPayload {
	fill := func(b byte, n int) []byte { return bytes.Repeat([]byte{b}, n) }
	hash := func(b byte) common.Hash { return common.Hash(fill(b, 32)) }
	p := &ExecutionPayload{
		ParentHash:            hash(0x10 + s),
		Miner:                 common.Address(fill(0x70+s, 20)),
		StateRoot:             hash(0x20 + s),
		Transactions:          make([][]byte, ntx),
		ReceiptsRoot:          hash(0x30 + s),
		Number:                1000 + uint64(s),
		GasLimits:             GasAmounts{30_000_000 + uint64(s), 786_432 + uint64(s)},
		GasUsed:               GasAmounts{21_000 + uint64(s), 131_072 + uint64(s)},
		Timestamp:             1_700_000_000 + uint64(s),
		ExtraData:             fill(0xab, nextra),
		MixHash:               hash(0x40 + s),
		Withdrawals:           make([][]byte, nwd),
		ExcessGas:             GasAmounts{uint64(s), 4096 + uint64(s)},
		ParentBeaconBlockRoot: hash(0x50 + s),
		RequestsHash:          hash(0x60 + s),
		BlockAccessList:       make([]byte, nbal),
		SlotNumber:            9_000_000 + uint64(s),
	}
	p.BaseFeesPerGas.Regular.Lsh(uint256.NewInt(1), 255)
	p.BaseFeesPerGas.Regular.Or(&p.BaseFeesPerGas.Regular, uint256.NewInt(uint64(s)))
	p.BaseFeesPerGas.Blob.SetUint64(1_000_000_000 + uint64(s))
	for i := range p.Transactions {
		p.Transactions[i] = fill(byte(i+1), (37*i+5)%200)
	}
	for i := range p.Withdrawals {
		p.Withdrawals[i] = fill(0x80+byte(i), 44)
	}
	for i := range p.BlockAccessList {
		p.BlockAccessList[i] = byte(i % 251)
	}
	return p
}

// maxPayload is every field at its maximum, with short lists of 0xff bytes.
func maxPayload() *ExecutionPayload {
	ff := func(n int) []byte { return bytes.Repeat([]byte{0xff}, n) }
	var max uint256.Int
	max.SetAllOne()
	return &ExecutionPayload{
		ParentHash:            common.Hash(ff(32)),
		Miner:                 common.Address(ff(20)),
		StateRoot:             common.Hash(ff(32)),
		Transactions:          [][]byte{ff(1), ff(32), ff(33)},
		ReceiptsRoot:          common.Hash(ff(32)),
		Number:                ^uint64(0),
		GasLimits:             GasAmounts{^uint64(0), ^uint64(0)},
		GasUsed:               GasAmounts{^uint64(0), ^uint64(0)},
		Timestamp:             ^uint64(0),
		ExtraData:             ff(32),
		MixHash:               common.Hash(ff(32)),
		BaseFeesPerGas:        BlobFeesPerGas{max, max},
		Withdrawals:           [][]byte{ff(44), ff(44)},
		ExcessGas:             GasAmounts{^uint64(0), ^uint64(0)},
		ParentBeaconBlockRoot: common.Hash(ff(32)),
		RequestsHash:          common.Hash(ff(32)),
		BlockAccessList:       ff(100),
		SlotNumber:            ^uint64(0),
	}
}

// referenceCase is a payload the test rebuilds from a few parameters, and
// what the SSZ executable reference (eth-ssz-specs 0.1.0) computed for it
// from the three containers as the EIP declares them: the roots of the list
// fields the summary replaces, and the payload root. Eighteen of the
// generator's thirty-three cases are pinned here; the generator and the full
// set live with the fellowship tooling, as the proof vectors of core/ssz do.
type referenceCase struct {
	name    string
	payload *ExecutionPayload
	lists   [3]string // transactions, withdrawals, block access list
	root    string
}

var referenceCases = []referenceCase{
	{
		name:    "zero",
		payload: &ExecutionPayload{},
		lists:   [3]string{emptyListRoot, emptyListRoot, emptyListRoot},
		root:    "0x050901020f828b1635abb9e1e8a97ec904eff2bea58fd78213359dad2bb83d70",
	},
	{
		name:    "max",
		payload: maxPayload(),
		lists: [3]string{
			"0x9b27ec861debee615dc21a31cf78477b7e65a447487c7d247db9ebda52881b7e",
			"0xebedde8fbe6c2103a0cc0a1ace9307f0cde3f7f6ba5d3ef342f818628f0dc6ac",
			"0xa9cff8578daacdf500d6faf3e2dc924ae15623461b25aa32a11e6e78381fbc4e",
		},
		root: "0xd82737dc9f60ab9c140244cde03865ad79fb6657bdf04e9934dc69336e9e61d0",
	},
	// An empty list and a list holding one empty element differ.
	{
		name:    "one-empty-transaction",
		payload: &ExecutionPayload{Transactions: [][]byte{{}}},
		lists:   [3]string{"0x03b28799b6c001d2a449bee88f2423870c1aa47bbcc60c3a00b3da008571ac43", emptyListRoot, emptyListRoot},
		root:    "0x9b2fca1702993c79ec017ce6b7edccc6fa5cfa4c4aebb7d4f9e71fbdc0e22c80",
	},
	{
		name:    "one-empty-withdrawal",
		payload: &ExecutionPayload{Withdrawals: [][]byte{{}}},
		lists:   [3]string{emptyListRoot, "0x03b28799b6c001d2a449bee88f2423870c1aa47bbcc60c3a00b3da008571ac43", emptyListRoot},
		root:    "0xb1156cc324664b27115f641a769a347305895073f7a26924b360467543ee1ce9",
	},
	// One element of 88 chunks, past the 85-chunk subtree boundary.
	{
		name:    "long-transaction",
		payload: &ExecutionPayload{Transactions: [][]byte{bytes.Repeat([]byte{0x99}, 2800)}},
		lists:   [3]string{"0xb71022f1212661aaec4769faa3653e8020f977d3bd7b9e3ed1e075e52776f045", emptyListRoot, emptyListRoot},
		root:    "0x3ad372ff654825601747a215a958065258996f7eaae841e33d8efbc432e87438",
	},
	{
		name:    "extra-data-1",
		payload: patternedPayload(1, 0, 0, 0, 1),
		lists:   [3]string{emptyListRoot, emptyListRoot, emptyListRoot},
		root:    "0x866cce596c587893bdc260b01749b64cc97cb309646889a416fb58ce0059813f",
	},
	{
		name:    "extra-data-32",
		payload: patternedPayload(1, 0, 0, 0, 32),
		lists:   [3]string{emptyListRoot, emptyListRoot, emptyListRoot},
		root:    "0x0dadc2aded36ebc22a93fa33a57432c5a99f628a2ff8a078476cbd5cf07ec00f",
	},
	// Element counts around the progressive subtree boundaries 1, 5 and 21.
	{
		name:    "transactions-1",
		payload: patternedPayload(2, 1, 0, 0, 0),
		lists:   [3]string{"0x65f37c681332a07dc47261fc93dd3c26b2534495fe716a3f27ba78aea659813a", emptyListRoot, emptyListRoot},
		root:    "0xfea7046a022a4ef8716212affe7ba4cc8eab4228ff0eefd34d531dde57520d6b",
	},
	{
		name:    "transactions-5",
		payload: patternedPayload(2, 5, 0, 0, 0),
		lists:   [3]string{"0x4a3671166caf50190b9e3bef494172fdcc3cc6a47efee332f3f78bc263f1eaf0", emptyListRoot, emptyListRoot},
		root:    "0x79d13f13dfbf90b581b704342b6330e5d9e7c35fd71c41a0a7f48213241ff2fc",
	},
	{
		name:    "transactions-21",
		payload: patternedPayload(2, 21, 0, 0, 0),
		lists:   [3]string{"0x399fac6765b159b915dce6c6de2a5b34e420f42fa0cd9621297b3e7715726e0a", emptyListRoot, emptyListRoot},
		root:    "0x1cf5d2aeed24e8fe6ad3bb352ab04564d365a5c399e5da0bf6c9d4ecce2d515b",
	},
	{
		name:    "transactions-22",
		payload: patternedPayload(2, 22, 0, 0, 0),
		lists:   [3]string{"0x4dd8bfa3c7114854ebfeeb059172b00c5d7e2c9a292d6e01de2e10ad7bb71d2b", emptyListRoot, emptyListRoot},
		root:    "0x21b28f8e1d8a4bf44a13ef4ffe5c39c4aea1c848f1a10d9500dd7c106b8c9106",
	},
	{
		name:    "withdrawals-4",
		payload: patternedPayload(3, 0, 4, 0, 0),
		lists:   [3]string{emptyListRoot, "0xa90a9a4fd5ef94f2a30840f64a4b0c6b0c79866663acaf2bc51f813221149e54", emptyListRoot},
		root:    "0x2dfa2f8e69f5d25f800e9248eebd9831d12ea6ed7f834f4b1d2840483cc15a3a",
	},
	{
		name:    "withdrawals-5",
		payload: patternedPayload(3, 0, 5, 0, 0),
		lists:   [3]string{emptyListRoot, "0x2d2f70c311664b6cfbefee82962d0708801f6ebcab80cfeb5e68cfe4a6b33fd1", emptyListRoot},
		root:    "0x5af1a19c6f63a78eaa3c100cd67d031531876aa602df1b89c3ad7e6375e3a389",
	},
	// Byte lengths past the chunk boundary and the 21- and 85-chunk subtree
	// boundaries of one progressive byte list.
	{
		name:    "block-access-list-33",
		payload: patternedPayload(4, 0, 0, 33, 0),
		lists:   [3]string{emptyListRoot, emptyListRoot, "0x43cd474d3b097438f5185868b2d6622d73ef7fa200719864508c0f3f24854cf2"},
		root:    "0x7cf45c98aee9cc764f93d7a617de5bb296ccc9393f1fdedde45f8a50a5c14d4a",
	},
	{
		name:    "block-access-list-673",
		payload: patternedPayload(4, 0, 0, 673, 0),
		lists:   [3]string{emptyListRoot, emptyListRoot, "0xeaa87bad13081655c15bcb59e7aadb8cfd72f5f7cd04ae67878a93ff00d76358"},
		root:    "0xf9e0414c9843e9c96f06e8e280759d38ed5d04e1a38aa78698b185a8a8052504",
	},
	{
		name:    "block-access-list-2800",
		payload: patternedPayload(4, 0, 0, 2800, 0),
		lists:   [3]string{emptyListRoot, emptyListRoot, "0x181b9b13ca61f5a88eeefed035265c4da2b56c80b3d5076d7fb9017ed601d22d"},
		root:    "0x92ed2ec050e1f67106b7ef552424823a8711c04fd26db0d019cd36493ac81411",
	},
	// Every field populated at once.
	{
		name:    "full-5",
		payload: patternedPayload(5, 22, 21, 2800, 32),
		lists: [3]string{
			"0x4dd8bfa3c7114854ebfeeb059172b00c5d7e2c9a292d6e01de2e10ad7bb71d2b",
			"0xf746c4efc193d828e29f7a881c5dd3d2e90aae69c9404ff5449de0105ae94ddf",
			"0x181b9b13ca61f5a88eeefed035265c4da2b56c80b3d5076d7fb9017ed601d22d",
		},
		root: "0x8668287f572c05814828385343234cc1643a6df2b68e25cf8b6e97384333b88d",
	},
	{
		name:    "full-6",
		payload: patternedPayload(6, 6, 5, 161, 7),
		lists: [3]string{
			"0xb9166db20e0212889e7ee993f274be78956544939240cf97b29aa9b537e39a56",
			"0x2d2f70c311664b6cfbefee82962d0708801f6ebcab80cfeb5e68cfe4a6b33fd1",
			"0x078c011197ebff63d41d23b4eb7233af5c5477c2dcb721210a4468f7169919a9",
		},
		root: "0x4efa089aecb1cbe9bb3bc52b2950fe53f426d7527e96388637d952a16311e061",
	},
}

// TestExecutionPayloadReference checks both computation paths against the
// reference: the payload root from the full lists, and the summary's list
// roots and root, which the reference computes by hashing the lists in
// place.
func TestExecutionPayloadReference(t *testing.T) {
	for _, c := range referenceCases {
		t.Run(c.name, func(t *testing.T) {
			root, err := c.payload.Root()
			if err != nil {
				t.Fatal(err)
			}
			if want := common.HexToHash(c.root); root != want {
				t.Errorf("payload root %x, want %x", root, want)
			}
			s := c.payload.Summary()
			for i, got := range []common.Hash{s.TransactionsRoot, s.WithdrawalsRoot, s.BlockAccessListRoot} {
				if want := common.HexToHash(c.lists[i]); got != want {
					t.Errorf("list %d root %x, want %x", i, got, want)
				}
			}
			if got, err := s.Root(); err != nil || got != root {
				t.Errorf("summary root %x, %v, want %x", got, err, root)
			}
		})
	}
}

// TestExecutionPayloadErigonFixture rebuilds the fixture of erigontech/erigon
// PR 23854 from Geth's types and checks every encoding and root it pins, the
// one payload root published by another client for the current EIP text.
// The requests hash is Erigon's Electra ExecutionRequests root of the
// fixture's single withdrawal request, an input here as in the payload.
func TestExecutionPayloadErigonFixture(t *testing.T) {
	to := common.HexToAddress("0x1111111111111111111111111111111111111111")
	txs := []*Transaction{
		NewTx(&LegacyTx{Nonce: 1, GasPrice: big.NewInt(3), Gas: 21_000, To: &to, Value: big.NewInt(2), Data: []byte{0xaa}}),
		NewTx(&DynamicFeeTx{ChainID: big.NewInt(1), Nonce: 2, GasTipCap: big.NewInt(5), GasFeeCap: big.NewInt(6), Gas: 50_000, To: &to, Value: big.NewInt(4), Data: []byte{0xbb, 0xcc}}),
	}
	receipts := []*Receipt{
		{Type: LegacyTxType, Status: ReceiptStatusSuccessful, CumulativeGasUsed: 21_000},
		{Type: DynamicFeeTxType, Status: ReceiptStatusFailed, CumulativeGasUsed: 42_000},
	}
	withdrawal := &Withdrawal{Index: 1, Validator: 2, Address: to, Amount: 3}

	// The element encodings the fixture hashes: unsigned transactions carry
	// empty v, r, s; receipts carry the zero bloom.
	var txBytes, receiptBytes [][]byte
	for _, tx := range txs {
		b, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		txBytes = append(txBytes, b)
	}
	for _, r := range receipts {
		b, err := r.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		receiptBytes = append(receiptBytes, b)
	}
	withdrawalBytes, err := rlp.EncodeToBytes(withdrawal)
	if err != nil {
		t.Fatal(err)
	}
	accessList, err := rlp.EncodeToBytes(&bal.BlockAccessList{})
	if err != nil {
		t.Fatal(err)
	}
	zeroBloom := make([]byte, 256)
	for i, want := range [][]byte{
		common.FromHex("e001038252089411111111111111111111111111111111111111110281aa808080"),
		common.FromHex("02e40102050682c3509411111111111111111111111111111111111111110482bbccc0808080"),
	} {
		if !bytes.Equal(txBytes[i], want) {
			t.Fatalf("transaction %d encodes as %x, want %x", i, txBytes[i], want)
		}
	}
	for i, want := range [][]byte{
		append(append(common.FromHex("f9010801825208b90100"), zeroBloom...), 0xc0),
		append(append(common.FromHex("02f901088082a410b90100"), zeroBloom...), 0xc0),
	} {
		if !bytes.Equal(receiptBytes[i], want) {
			t.Fatalf("receipt %d encodes as %x, want %x", i, receiptBytes[i], want)
		}
	}
	if want := common.FromHex("d8010294111111111111111111111111111111111111111103"); !bytes.Equal(withdrawalBytes, want) {
		t.Fatalf("withdrawal encodes as %x, want %x", withdrawalBytes, want)
	}
	if !bytes.Equal(accessList, []byte{0xc0}) {
		t.Fatalf("empty block access list encodes as %x, want c0", accessList)
	}

	// Blob schedule {target 1, max 2, update fraction 10000}: blob gas limit
	// 2 * 131072, blob base fee fake_exponential(1, 4096, 10000) = 1.
	// excess_gas.regular has no source in a header; the fixture sets 7.
	p := &ExecutionPayload{
		ParentHash:            common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		Miner:                 common.HexToAddress("0x2222222222222222222222222222222222222222"),
		StateRoot:             common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
		Transactions:          txBytes,
		ReceiptsRoot:          ByteListsRoot(receiptBytes),
		Number:                99,
		GasLimits:             GasAmounts{Regular: 30_000_000, Blob: 262144},
		GasUsed:               GasAmounts{Regular: 42_000, Blob: 131072},
		Timestamp:             1,
		ExtraData:             common.FromHex("deadbeef"),
		MixHash:               common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555"),
		BaseFeesPerGas:        BlobFeesPerGas{Regular: *uint256.NewInt(1_000_000_000), Blob: *uint256.NewInt(1)},
		Withdrawals:           [][]byte{withdrawalBytes},
		ExcessGas:             GasAmounts{Regular: 7, Blob: 4096},
		ParentBeaconBlockRoot: common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444"),
		RequestsHash:          common.HexToHash("0xb1c4c413201f39ef0e4528edbd42e595a5258578a22e8634a9dee79684fdf245"),
		BlockAccessList:       accessList,
		SlotNumber:            12345,
	}
	if want := common.HexToHash("0x4e93e5e6c10dd15a002b4447369b75d15ead60ee8caf9186ae9850f07a9c4088"); p.ReceiptsRoot != want {
		t.Errorf("receipts root %x, want %x", p.ReceiptsRoot, want)
	}
	s := p.Summary()
	if want := common.HexToHash("0x3339f9d76dc622a3c9d6da079a1479a499f40e6154f88ad8111c15889aa0430c"); s.TransactionsRoot != want {
		t.Errorf("transactions root %x, want %x", s.TransactionsRoot, want)
	}
	if want := common.HexToHash("0x76b3fc6c3855cc906e98c42aed803a8e2c9c883cef824f72b9c02fb2581dc7f2"); s.WithdrawalsRoot != want {
		t.Errorf("withdrawals root %x, want %x", s.WithdrawalsRoot, want)
	}
	if want := common.HexToHash("0xb404136294c9c48bd2a7ce3494551538bfbdfef3385e2ca1542ad640a8476612"); s.BlockAccessListRoot != want {
		t.Errorf("block access list root %x, want %x", s.BlockAccessListRoot, want)
	}
	root, err := p.Root()
	if err != nil {
		t.Fatal(err)
	}
	if want := common.HexToHash("0x51951516710af90e8c20092e086f37232c69b558d6ba0035333982e115e1f627"); root != want {
		t.Errorf("payload root %x, want %x", root, want)
	}
}

// TestExecutionPayloadSummaryReth checks the summary path against the
// ExecutionSummary root pinned by the Reth fellows' pureth branch
// (ArshLabs/reth@pureth-integration, crates/pureth/block), which takes the
// list roots as inputs.
func TestExecutionPayloadSummaryReth(t *testing.T) {
	rep := func(b byte) (h common.Hash) {
		for i := range h {
			h[i] = b
		}
		return h
	}
	s := &ExecutionPayloadSummary{
		ParentHash:            rep(1),
		Miner:                 common.Address(bytes.Repeat([]byte{2}, 20)),
		StateRoot:             rep(3),
		TransactionsRoot:      rep(4),
		ReceiptsRoot:          rep(5),
		Number:                6,
		GasLimits:             GasAmounts{Regular: 30_000_000, Blob: 786_432},
		GasUsed:               GasAmounts{Regular: 21_000, Blob: 131_072},
		Timestamp:             9,
		ExtraData:             []byte{10, 11},
		MixHash:               rep(11),
		BaseFeesPerGas:        BlobFeesPerGas{Regular: *new(uint256.Int).SetAllOne(), Blob: *uint256.NewInt(12)},
		WithdrawalsRoot:       rep(13),
		ExcessGas:             GasAmounts{Regular: 0, Blob: 14},
		ParentBeaconBlockRoot: rep(15),
		RequestsHash:          rep(16),
		BlockAccessListRoot:   rep(17),
		SlotNumber:            18,
	}
	root, err := s.Root()
	if err != nil {
		t.Fatal(err)
	}
	if want := common.HexToHash("0x77cd9751aa6a2ed05e728b9a96037f4f3c57eb1946900fb06b7140bc9c60d35b"); root != want {
		t.Errorf("summary root %x, want %x", root, want)
	}
}

// hashPair is H(a ‖ b) over crypto/sha256, for the hand computations below.
func hashPair(a, b [32]byte) [32]byte {
	return sha256.Sum256(append(a[:], b[:]...))
}

// naivePairRoot is ProgressiveContainer(active_fields=[1, 1]) over two leaf
// chunks, worked by hand from the EIPs:
// merkleize_progressive([a, b]) = hash(merkleize([a], 1), hash(merkleize([b], 4), 0)),
// then mix_in_active_fields with pack_bits([1, 1]) = 0x03.
func naivePairRoot(a, b [32]byte) [32]byte {
	var zero, word [32]byte
	word[0] = 0x03
	sub4 := hashPair(hashPair(b, zero), hashPair(zero, zero))
	return hashPair(hashPair(a, hashPair(sub4, zero)), word)
}

// TestInnerContainerRoots checks GasAmounts and BlobFeesPerGas against the
// hand computation and against the two roots the Reth fellows pin.
func TestInnerContainerRoots(t *testing.T) {
	var le [32]byte
	binary.LittleEndian.PutUint64(le[:], 0x0102030405060708)
	gas := GasAmounts{Regular: 0x0102030405060708, Blob: 42}
	var blob [32]byte
	blob[0] = 42
	if got, want := gas.Root(), common.Hash(naivePairRoot(le, blob)); got != want {
		t.Errorf("GasAmounts root %x, want %x", got, want)
	}
	gas = GasAmounts{Regular: 1, Blob: ^uint64(0)}
	if got, want := gas.Root(), common.HexToHash("0x9f9a1cf07aff54f3197a2cf6f38a6c9d4f96dc1edd352b69d00cdb66e9c7b7e4"); got != want {
		t.Errorf("GasAmounts{1, max} root %x, want %x", got, want)
	}

	var max, top [32]byte
	for i := range max {
		max[i] = 0xff
	}
	top[31] = 0x80 // 1 << 255, little-endian
	fees := BlobFeesPerGas{Regular: *new(uint256.Int).SetAllOne(), Blob: *new(uint256.Int).Lsh(uint256.NewInt(1), 255)}
	if got, want := fees.Root(), common.Hash(naivePairRoot(max, top)); got != want {
		t.Errorf("BlobFeesPerGas root %x, want %x", got, want)
	}
	if got, want := fees.Root(), common.HexToHash("0xe1e98c0ea24eb951bece89b1deb89c40bf1d574cb58f23c80293731c35decb16"); got != want {
		t.Errorf("BlobFeesPerGas{max, 1<<255} root %x, want %x", got, want)
	}
}

// TestExecutionPayloadFieldCount guards the hand-written Root methods against
// a field added to a struct but not to its active_fields configuration. The
// container helper already panics when a Root lists more or fewer roots than
// the configuration has fields, so with this the struct, the configuration
// and the roots list stay in step.
func TestExecutionPayloadFieldCount(t *testing.T) {
	for _, c := range []struct {
		name   string
		value  any
		fields int
	}{
		{"ExecutionPayload", ExecutionPayload{}, payloadFields.NumFields()},
		{"ExecutionPayloadSummary", ExecutionPayloadSummary{}, payloadFields.NumFields()},
		{"GasAmounts", GasAmounts{}, pairFields.NumFields()},
		{"BlobFeesPerGas", BlobFeesPerGas{}, pairFields.NumFields()},
	} {
		if n := reflect.TypeOf(c.value).NumField(); n != c.fields {
			t.Errorf("%s has %d fields, its configuration %d", c.name, n, c.fields)
		}
	}
}

// TestExecutionPayloadExtraData covers the one value rule of the container:
// extra data is bounded by MAX_EXTRA_DATA_BYTES on both paths.
func TestExecutionPayloadExtraData(t *testing.T) {
	p := &ExecutionPayload{ExtraData: make([]byte, 32)}
	if _, err := p.Root(); err != nil {
		t.Errorf("32 bytes of extra data: %v", err)
	}
	p.ExtraData = make([]byte, 33)
	if _, err := p.Root(); err == nil {
		t.Error("33 bytes of extra data accepted by the payload")
	}
	if _, err := p.Summary().Root(); err == nil {
		t.Error("33 bytes of extra data accepted by the summary")
	}
}

// naivePayloadRoot transcribes the EIP's ExecutionPayload definition from its
// field values: each leaf encoded by hand, the lists as progressive lists of
// progressive byte lists, and the eighteen chunks folded as
// mix_in_active_fields(merkleize_progressive(chunks), [1] * 18).
func naivePayloadRoot(p *ExecutionPayload) [32]byte {
	u64 := func(v uint64) (c [32]byte) {
		binary.LittleEndian.PutUint64(c[:], v)
		return c
	}
	u256 := func(v *uint256.Int) (c [32]byte) {
		be := v.Bytes32()
		for i := range c {
			c[i] = be[31-i]
		}
		return c
	}
	pair := func(a, b [32]byte) [32]byte { return naivePairRoot(a, b) }
	byteList := func(b []byte) [32]byte {
		return ssz.MixInLength(ssz.MerkleizeProgressive(ssz.Pack(b)), uint64(len(b)))
	}
	byteLists := func(elems [][]byte) [32]byte {
		roots := make([][32]byte, len(elems))
		for i, e := range elems {
			roots[i] = byteList(e)
		}
		return ssz.MixInLength(ssz.MerkleizeProgressive(roots), uint64(len(elems)))
	}
	var miner [32]byte
	copy(miner[:], p.Miner[:])
	extra := ssz.MixInLength(ssz.Merkleize(ssz.Pack(p.ExtraData), 1), uint64(len(p.ExtraData)))
	chunks := [][32]byte{
		p.ParentHash, miner, p.StateRoot, byteLists(p.Transactions), p.ReceiptsRoot,
		u64(p.Number), pair(u64(p.GasLimits.Regular), u64(p.GasLimits.Blob)),
		pair(u64(p.GasUsed.Regular), u64(p.GasUsed.Blob)), u64(p.Timestamp), extra,
		p.MixHash, pair(u256(&p.BaseFeesPerGas.Regular), u256(&p.BaseFeesPerGas.Blob)),
		byteLists(p.Withdrawals), pair(u64(p.ExcessGas.Regular), u64(p.ExcessGas.Blob)),
		p.ParentBeaconBlockRoot, p.RequestsHash, byteList(p.BlockAccessList), u64(p.SlotNumber),
	}
	var word [32]byte // pack_bits([1] * 18)
	word[0], word[1], word[2] = 0xff, 0xff, 0x03
	return hashPair(ssz.MerkleizeProgressive(chunks), word)
}

// fuzzPayload derives a payload from fuzz input, reading fields off the
// front and leaving the rest zero. Extra data runs to 33 bytes so the fuzzer
// also reaches the rejection.
func fuzzPayload(data []byte) *ExecutionPayload {
	r := bytes.NewReader(data)
	next := func(n int) []byte {
		buf := make([]byte, n)
		r.Read(buf)
		return buf
	}
	u64 := func() uint64 { return binary.LittleEndian.Uint64(next(8)) }
	list := func() [][]byte {
		elems := make([][]byte, next(1)[0]%24)
		for i := range elems {
			elems[i] = next(int(binary.LittleEndian.Uint16(next(2)) % 600))
		}
		return elems
	}
	p := &ExecutionPayload{
		ParentHash:            common.Hash(next(32)),
		Miner:                 common.Address(next(20)),
		StateRoot:             common.Hash(next(32)),
		Transactions:          list(),
		ReceiptsRoot:          common.Hash(next(32)),
		Number:                u64(),
		GasLimits:             GasAmounts{u64(), u64()},
		GasUsed:               GasAmounts{u64(), u64()},
		Timestamp:             u64(),
		ExtraData:             next(int(next(1)[0] % 34)),
		MixHash:               common.Hash(next(32)),
		Withdrawals:           list(),
		ExcessGas:             GasAmounts{u64(), u64()},
		ParentBeaconBlockRoot: common.Hash(next(32)),
		RequestsHash:          common.Hash(next(32)),
		BlockAccessList:       next(int(binary.LittleEndian.Uint16(next(2)) % 3000)),
		SlotNumber:            u64(),
	}
	p.BaseFeesPerGas.Regular.SetBytes(next(32))
	p.BaseFeesPerGas.Blob.SetBytes(next(32))
	return p
}

// FuzzExecutionPayloadRoot checks arbitrary payloads against the EIP
// transcription, and that the summary has the payload's root.
func FuzzExecutionPayloadRoot(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte{0xff}, 400))
	f.Add(bytes.Repeat([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22}, 60))
	f.Fuzz(func(t *testing.T, data []byte) {
		p := fuzzPayload(data)
		root, err := p.Root()
		if len(p.ExtraData) > 32 {
			if err == nil {
				t.Fatal("extra data over 32 bytes accepted")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if want := common.Hash(naivePayloadRoot(p)); root != want {
			t.Fatalf("payload root %x, EIP transcription %x", root, want)
		}
		if summary, err := p.Summary().Root(); err != nil || summary != root {
			t.Fatalf("summary root %x, %v, payload root %x", summary, err, root)
		}
	})
}
