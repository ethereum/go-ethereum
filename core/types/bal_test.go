package types

import (
	"bytes"
	"fmt"
	"github.com/holiman/uint256"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// test that a populated access list can be encoded/decoded correctly
func TestBALEncoding(t *testing.T) {
	b := BlockAccessList{
		map[common.Address]*accountAccess{
			common.BytesToAddress([]byte{0x01}): {
				StorageWrites: map[common.Hash]slotWrites{
					common.BytesToHash([]byte{0x01}): map[uint64]common.Hash{
						1: common.BytesToHash([]byte{1, 2, 3, 4}),
						2: common.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
					},
					common.BytesToHash([]byte{0x10}): map[uint64]common.Hash{
						20: common.BytesToHash([]byte{1, 2, 3, 4}),
					},
				},
				StorageReads: map[common.Hash]struct{}{
					common.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7}): {},
				},
				BalanceChanges: balanceDiff{
					1: uint256.NewInt(100),
					2: uint256.NewInt(500),
				},
				NonceChanges: accountNonceDiffs{
					1: 2,
					2: 6,
				},
				CodeChange: &codeChange{
					TxIndex: 0,
					Code:    common.Hex2Bytes("deadbeef"),
				},
			},
			common.BytesToAddress([]byte{0x02}): {
				StorageWrites: map[common.Hash]slotWrites{
					common.BytesToHash([]byte{0x01}): map[uint64]common.Hash{
						2: common.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
						3: common.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
					},
					common.BytesToHash([]byte{0x10}): map[uint64]common.Hash{
						21: common.BytesToHash([]byte{1, 2, 3, 4, 5}),
					},
				},
				StorageReads: map[common.Hash]struct{}{
					common.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}): {},
				},
				BalanceChanges: balanceDiff{
					2: uint256.NewInt(100),
					3: uint256.NewInt(500),
				},
				NonceChanges: accountNonceDiffs{
					1: 2,
				},
			},
		},
	}
	fmt.Println(b.PrettyPrint())
	var buf bytes.Buffer
	err := b.EncodeRLP(&buf)
	if err != nil {
		t.Fatalf("encoding failed: %v\n", err)
	}
	var dec BlockAccessList
	if err := dec.DecodeRLP(rlp.NewStream(bytes.NewReader(buf.Bytes()), 10000000)); err != nil {
		t.Fatalf("decoding failed: %v\n", err)
	}
	if dec.Hash() != b.Hash() {
		t.Fatalf("encoded block hash doesn't match decoded")
	}
}

func TestBALDecoding(t *testing.T) {
	data, err := os.ReadFile("22615532_block_access_list_with_reads.txt")
	if err != nil {
		t.Fatal(err)
	}
	var b BlockAccessList
	if err := b.DecodeSSZ(data); err != nil {
		t.Fatal(err)
	}
}
