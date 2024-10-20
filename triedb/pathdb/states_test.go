// Copyright 2024 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestStatesMerge(t *testing.T) {
	a := newStates(
		nil,
		map[common.Hash][]byte{
			common.Hash{0xa}: {0xa0},
			common.Hash{0xb}: {0xb0},
			common.Hash{0xc}: {0xc0},
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0xa}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20},
			},
			common.Hash{0xb}: {
				common.Hash{0x1}: {0x10},
			},
			common.Hash{0xc}: {
				common.Hash{0x1}: {0x10},
			},
		},
	)
	b := newStates(
		map[common.Hash]struct{}{
			common.Hash{0xa}: {},
			common.Hash{0xc}: {},
		},
		map[common.Hash][]byte{
			common.Hash{0xa}: {0xa1},
			common.Hash{0xb}: {0xb1},
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0xa}: {
				common.Hash{0x1}: {0x11},
				common.Hash{0x3}: {0x31},
			},
			common.Hash{0xb}: {
				common.Hash{0x1}: {0x11},
			},
		},
	)
	a.merge(b)

	blob, exist := a.account(common.Hash{0xa})
	if !exist || !bytes.Equal(blob, []byte{0xa1}) {
		t.Error("Unexpected value for account a")
	}
	blob, exist = a.account(common.Hash{0xb})
	if !exist || !bytes.Equal(blob, []byte{0xb1}) {
		t.Error("Unexpected value for account b")
	}
	blob, exist = a.account(common.Hash{0xc})
	if !exist || len(blob) != 0 {
		t.Error("Unexpected value for account c")
	}
	// unknown account
	blob, exist = a.account(common.Hash{0xd})
	if exist || len(blob) != 0 {
		t.Error("Unexpected value for account d")
	}

	blob, exist = a.storage(common.Hash{0xa}, common.Hash{0x1})
	if !exist || !bytes.Equal(blob, []byte{0x11}) {
		t.Error("Unexpected value for a's storage")
	}
	blob, exist = a.storage(common.Hash{0xa}, common.Hash{0x2})
	if !exist || len(blob) != 0 {
		t.Error("Unexpected value for a's storage")
	}
	blob, exist = a.storage(common.Hash{0xa}, common.Hash{0x3})
	if !exist || !bytes.Equal(blob, []byte{0x31}) {
		t.Error("Unexpected value for a's storage")
	}
	blob, exist = a.storage(common.Hash{0xb}, common.Hash{0x1})
	if !exist || !bytes.Equal(blob, []byte{0x11}) {
		t.Error("Unexpected value for b's storage")
	}
	blob, exist = a.storage(common.Hash{0xc}, common.Hash{0x1})
	if !exist || len(blob) != 0 {
		t.Error("Unexpected value for c's storage")
	}

	// unknown storage slots
	blob, exist = a.storage(common.Hash{0xd}, common.Hash{0x1})
	if exist || len(blob) != 0 {
		t.Error("Unexpected value for d's storage")
	}
}

func TestStatesRevert(t *testing.T) {
	a := newStates(
		nil,
		map[common.Hash][]byte{
			common.Hash{0xa}: {0xa0},
			common.Hash{0xb}: {0xb0},
			common.Hash{0xc}: {0xc0},
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0xa}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20},
			},
			common.Hash{0xb}: {
				common.Hash{0x1}: {0x10},
			},
			common.Hash{0xc}: {
				common.Hash{0x1}: {0x10},
			},
		},
	)
	b := newStates(
		map[common.Hash]struct{}{
			common.Hash{0xa}: {},
			common.Hash{0xc}: {},
		},
		map[common.Hash][]byte{
			common.Hash{0xa}: {0xa1},
			common.Hash{0xb}: {0xb1},
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0xa}: {
				common.Hash{0x1}: {0x11},
				common.Hash{0x3}: {0x31},
			},
			common.Hash{0xb}: {
				common.Hash{0x1}: {0x11},
			},
		},
	)
	a.merge(b)
	a.revert(
		map[common.Hash][]byte{
			common.Hash{0xa}: {0xa0},
			common.Hash{0xb}: {0xb0},
			common.Hash{0xc}: {0xc0},
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0xa}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20},
				common.Hash{0x3}: {},
			},
			common.Hash{0xb}: {
				common.Hash{0x1}: {0x10},
			},
			common.Hash{0xc}: {
				common.Hash{0x1}: {0x10},
			},
		},
	)

	blob, exist := a.account(common.Hash{0xa})
	if !exist || !bytes.Equal(blob, []byte{0xa0}) {
		t.Error("Unexpected value for account a")
	}
	blob, exist = a.account(common.Hash{0xb})
	if !exist || !bytes.Equal(blob, []byte{0xb0}) {
		t.Error("Unexpected value for account b")
	}
	blob, exist = a.account(common.Hash{0xc})
	if !exist || !bytes.Equal(blob, []byte{0xc0}) {
		t.Error("Unexpected value for account c")
	}
	// unknown account
	blob, exist = a.account(common.Hash{0xd})
	if exist || len(blob) != 0 {
		t.Error("Unexpected value for account d")
	}

	blob, exist = a.storage(common.Hash{0xa}, common.Hash{0x1})
	if !exist || !bytes.Equal(blob, []byte{0x10}) {
		t.Error("Unexpected value for a's storage")
	}
	blob, exist = a.storage(common.Hash{0xa}, common.Hash{0x2})
	if !exist || !bytes.Equal(blob, []byte{0x20}) {
		t.Error("Unexpected value for a's storage")
	}
	_, exist = a.storage(common.Hash{0xa}, common.Hash{0x3})
	if exist {
		t.Error("Unexpected value for a's storage")
	}
	blob, exist = a.storage(common.Hash{0xb}, common.Hash{0x1})
	if !exist || !bytes.Equal(blob, []byte{0x10}) {
		t.Error("Unexpected value for b's storage")
	}
	blob, exist = a.storage(common.Hash{0xc}, common.Hash{0x1})
	if !exist || !bytes.Equal(blob, []byte{0x10}) {
		t.Error("Unexpected value for c's storage")
	}
	// unknown storage slots
	blob, exist = a.storage(common.Hash{0xd}, common.Hash{0x1})
	if exist || len(blob) != 0 {
		t.Error("Unexpected value for d's storage")
	}
}

func TestDestructJournalEncode(t *testing.T) {
	var enc journal
	enc.add(nil)          // nil
	enc.add([]destruct{}) // zero size destructs
	enc.add([]destruct{
		{Hash: common.HexToHash("0xdeadbeef"), Exist: true},
		{Hash: common.HexToHash("0xcafebabe"), Exist: false},
	})
	var buf bytes.Buffer
	enc.encode(&buf)

	var dec journal
	if err := dec.decode(rlp.NewStream(&buf, 0)); err != nil {
		t.Fatalf("Failed to decode journal, %v", err)
	}
	if len(enc.destructs) != len(dec.destructs) {
		t.Fatalf("Unexpected destruct journal length, want: %d, got: %d", len(enc.destructs), len(dec.destructs))
	}
	for i := 0; i < len(enc.destructs); i++ {
		want := enc.destructs[i]
		got := dec.destructs[i]
		if len(want) == 0 && len(got) == 0 {
			continue
		}
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("Unexpected destruct, want: %v, got: %v", want, got)
		}
	}
}

func TestStatesEncode(t *testing.T) {
	s := newStates(
		map[common.Hash]struct{}{
			common.Hash{0x1}: {},
		},
		map[common.Hash][]byte{
			common.Hash{0x1}: {0x1},
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0x1}: {
				common.Hash{0x1}: {0x1},
			},
		},
	)
	buf := bytes.NewBuffer(nil)
	if err := s.encode(buf); err != nil {
		t.Fatalf("Failed to encode states, %v", err)
	}
	var dec stateSet
	if err := dec.decode(rlp.NewStream(buf, 0)); err != nil {
		t.Fatalf("Failed to decode states, %v", err)
	}
	if !reflect.DeepEqual(s.destructSet, dec.destructSet) {
		t.Fatal("Unexpected destruct set")
	}
	if !reflect.DeepEqual(s.accountData, dec.accountData) {
		t.Fatal("Unexpected account data")
	}
	if !reflect.DeepEqual(s.storageData, dec.storageData) {
		t.Fatal("Unexpected storage data")
	}
}

func TestStateWithOriginEncode(t *testing.T) {
	s := NewStateSetWithOrigin(
		map[common.Hash]struct{}{
			common.Hash{0x1}: {},
		},
		map[common.Hash][]byte{
			common.Hash{0x1}: {0x1},
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0x1}: {
				common.Hash{0x1}: {0x1},
			},
		},
		map[common.Address][]byte{
			common.Address{0x1}: {0x1},
		},
		map[common.Address]map[common.Hash][]byte{
			common.Address{0x1}: {
				common.Hash{0x1}: {0x1},
			},
		},
	)
	buf := bytes.NewBuffer(nil)
	if err := s.encode(buf); err != nil {
		t.Fatalf("Failed to encode states, %v", err)
	}
	var dec StateSetWithOrigin
	if err := dec.decode(rlp.NewStream(buf, 0)); err != nil {
		t.Fatalf("Failed to decode states, %v", err)
	}
	if !reflect.DeepEqual(s.destructSet, dec.destructSet) {
		t.Fatal("Unexpected destruct set")
	}
	if !reflect.DeepEqual(s.accountData, dec.accountData) {
		t.Fatal("Unexpected account data")
	}
	if !reflect.DeepEqual(s.storageData, dec.storageData) {
		t.Fatal("Unexpected storage data")
	}
	if !reflect.DeepEqual(s.accountOrigin, dec.accountOrigin) {
		t.Fatal("Unexpected account origin data")
	}
	if !reflect.DeepEqual(s.storageOrigin, dec.storageOrigin) {
		t.Fatal("Unexpected storage origin data")
	}
}

func TestStateSizeTracking(t *testing.T) {
	expSizeA := 3*(common.HashLength+1) + /* account data */
		2*(2*common.HashLength+1) + /* storage data of 0xa */
		2*common.HashLength + 3 + /* storage data of 0xb */
		2*common.HashLength + 1 /* storage data of 0xc */

	a := newStates(
		nil,
		map[common.Hash][]byte{
			common.Hash{0xa}: {0xa0}, // common.HashLength+1
			common.Hash{0xb}: {0xb0}, // common.HashLength+1
			common.Hash{0xc}: {0xc0}, // common.HashLength+1
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0xa}: {
				common.Hash{0x1}: {0x10}, // 2*common.HashLength+1
				common.Hash{0x2}: {0x20}, // 2*common.HashLength+1
			},
			common.Hash{0xb}: {
				common.Hash{0x1}: {0x10, 0x11, 0x12}, // 2*common.HashLength+3
			},
			common.Hash{0xc}: {
				common.Hash{0x1}: {0x10}, // 2*common.HashLength+1
			},
		},
	)
	if a.size != uint64(expSizeA) {
		t.Fatalf("Unexpected size, want: %d, got: %d", expSizeA, a.size)
	}

	expSizeB := 2*common.HashLength + /* destruct set data */
		common.HashLength + 2 + common.HashLength + 3 + /* account data */
		2*common.HashLength + 3 + 2*common.HashLength + 2 + /* storage data of 0xa */
		2*common.HashLength + 2 + 2*common.HashLength + 2 /* storage data of 0xb */
	b := newStates(
		map[common.Hash]struct{}{
			common.Hash{0xa}: {}, // common.HashLength
			common.Hash{0xc}: {}, // common.HashLength
		},
		map[common.Hash][]byte{
			common.Hash{0xa}: {0xa1, 0xa1},       // common.HashLength+2
			common.Hash{0xb}: {0xb1, 0xb1, 0xb1}, // common.HashLength+3
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0xa}: {
				common.Hash{0x1}: {0x11, 0x11, 0x11}, // 2*common.HashLength+3
				common.Hash{0x3}: {0x31, 0x31},       // 2*common.HashLength+1
			},
			common.Hash{0xb}: {
				common.Hash{0x1}: {0x11, 0x11}, // 2*common.HashLength+2
				common.Hash{0x2}: {0x22, 0x22}, // 2*common.HashLength+2
			},
		},
	)
	if b.size != uint64(expSizeB) {
		t.Fatalf("Unexpected size, want: %d, got: %d", expSizeB, b.size)
	}

	a.merge(b)
	mergeSize := expSizeA + 2*common.HashLength    /* destruct set data */
	mergeSize += 1 /* account a data change */ + 2 /* account b data change */
	mergeSize -= common.HashLength + 1             /* account data removal of 0xc */
	mergeSize += 2 + 1                             /* storage a change */
	mergeSize += 2*common.HashLength + 2 - 1       /* storage b change */
	mergeSize -= 2*common.HashLength + 1           /* storage data removal of 0xc */

	if a.size != uint64(mergeSize) {
		t.Fatalf("Unexpected size, want: %d, got: %d", mergeSize, a.size)
	}

	// Revert the set to original status
	a.revert(
		map[common.Hash][]byte{
			common.Hash{0xa}: {0xa0},
			common.Hash{0xb}: {0xb0},
			common.Hash{0xc}: {0xc0},
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0xa}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20},
				common.Hash{0x3}: {},
			},
			common.Hash{0xb}: {
				common.Hash{0x1}: {0x10, 0x11, 0x12},
				common.Hash{0x2}: {},
			},
			common.Hash{0xc}: {
				common.Hash{0x1}: {0x10},
			},
		},
	)
	if a.size != uint64(expSizeA) {
		t.Fatalf("Unexpected size, want: %d, got: %d", expSizeA, a.size)
	}

	// Revert state set a again, this time with additional slots which were
	// deleted in account destruction and re-created because of resurrection.
	a.merge(b)
	a.revert(
		map[common.Hash][]byte{
			common.Hash{0xa}: {0xa0},
			common.Hash{0xb}: {0xb0},
			common.Hash{0xc}: {0xc0},
		},
		map[common.Hash]map[common.Hash][]byte{
			common.Hash{0xa}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20},
				common.Hash{0x3}: {},
				common.Hash{0x4}: {0x40},       // this slot was not in the set a, but resurrected because of revert
				common.Hash{0x5}: {0x50, 0x51}, // this slot was not in the set a, but resurrected because of revert
			},
			common.Hash{0xb}: {
				common.Hash{0x1}: {0x10, 0x11, 0x12},
				common.Hash{0x2}: {},
			},
			common.Hash{0xc}: {
				common.Hash{0x1}: {0x10},
			},
		},
	)
	expSize := expSizeA + common.HashLength*2 + 1 + /* slot 4 */ +common.HashLength*2 + 2 /* slot 5 */
	if a.size != uint64(expSize) {
		t.Fatalf("Unexpected size, want: %d, got: %d", expSize, a.size)
	}
}
