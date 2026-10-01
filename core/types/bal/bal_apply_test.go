package bal

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func TestDecodeApplyRLP(t *testing.T) {
	addr := common.HexToAddress("0x1234")
	input := BlockAccessList{{
		Address: addr,
		StorageChanges: []encodingSlotChanges{{
			Slot: uint256.NewInt(1),
			SlotChanges: []encodingStorageWrite{
				{BlockAccessIndex: 1, PostValue: uint256.NewInt(10)},
				{BlockAccessIndex: 4, PostValue: uint256.NewInt(20)},
			},
		}},
		StorageReads: []*uint256.Int{uint256.NewInt(99)},
		BalanceChanges: []encodingBalanceChange{
			{BlockAccessIndex: 2, PostBalance: uint256.NewInt(100)},
			{BlockAccessIndex: 5, PostBalance: uint256.NewInt(200)},
		},
		NonceChanges: []encodingAccountNonce{
			{BlockAccessIndex: 3, PostNonce: 7},
			{BlockAccessIndex: 6, PostNonce: 8},
		},
		CodeChanges: []encodingCodeChange{
			{BlockAccessIndex: 4, NewCode: []byte{0xaa}},
			{BlockAccessIndex: 7, NewCode: []byte{0xbb, 0xcc}},
		},
	}}
	var buf bytes.Buffer
	if err := input.EncodeRLP(&buf); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeApplyRLP(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	want := &BlockAccessList{{
		Address: addr,
		StorageChanges: []encodingSlotChanges{{
			Slot: uint256.NewInt(1),
			SlotChanges: []encodingStorageWrite{{
				BlockAccessIndex: 4,
				PostValue:        uint256.NewInt(20),
			}},
		}},
		BalanceChanges: []encodingBalanceChange{{
			BlockAccessIndex: 5,
			PostBalance:      uint256.NewInt(200),
		}},
		NonceChanges: []encodingAccountNonce{{
			BlockAccessIndex: 6,
			PostNonce:        8,
		}},
		CodeChanges: []encodingCodeChange{{
			BlockAccessIndex: 7,
			NewCode:          []byte{0xbb, 0xcc},
		}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("apply view mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestDecodeApplyRLPRejectsTrailingBytes(t *testing.T) {
	input := BlockAccessList{{Address: common.HexToAddress("0x1234")}}
	var buf bytes.Buffer
	if err := input.EncodeRLP(&buf); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeApplyRLP(append(buf.Bytes(), 0x80)); err == nil {
		t.Fatal("expected trailing-data error")
	}
}

func BenchmarkDecodeApplyRLP(b *testing.B) {
	input := makeTestBAL(true)
	var buf bytes.Buffer
	if err := input.EncodeRLP(&buf); err != nil {
		b.Fatal(err)
	}
	data := append([]byte(nil), buf.Bytes()...)

	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var out BlockAccessList
			if err := rlp.DecodeBytes(data, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("apply", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := DecodeApplyRLP(data); err != nil {
				b.Fatal(err)
			}
		}
	})
}
