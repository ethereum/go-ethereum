package types

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestBALEncoding(t *testing.T) {
	b := BlockAccessList{
		map[common.Address]*accountAccess{
			common.BytesToAddress([]byte{0x01}): {
				StorageWrites: map[common.Hash]slotWrites{
					common.BytesToHash([]byte{0x01}): map[uint64]common.Hash{},
				},
			},
		},
	}
	// TODO: populate the BAL with something
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
