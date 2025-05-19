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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

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
		map[common.Hash][]byte{
			{0xa}: {0xa0},
			{0xb}: {0xb0},
			{0xc}: {0xc0},
		},
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20},
			},
			{0xb}: {
				common.Hash{0x1}: {0x10},
			},
			{0xc}: {
				common.Hash{0x1}: {0x10},
			},
		},
		false,
	)
	b := newStates(
		map[common.Hash][]byte{
			{0xa}: {0xa1},
			{0xb}: {0xb1},
			{0xc}: nil, // delete account
		},
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: {0x11},
				common.Hash{0x2}: nil, // delete slot
				common.Hash{0x3}: {0x31},
			},
			{0xb}: {
				common.Hash{0x1}: {0x11},
			},
			{0xc}: {
				common.Hash{0x1}: nil, // delete slot
			},
		},
		false,
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
		map[common.Hash][]byte{
			{0xa}: {0xa0},
			{0xb}: {0xb0},
			{0xc}: {0xc0},
		},
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20},
			},
			{0xb}: {
				common.Hash{0x1}: {0x10},
			},
			{0xc}: {
				common.Hash{0x1}: {0x10},
			},
		},
		false,
	)
	b := newStates(
		map[common.Hash][]byte{
			{0xa}: {0xa1},
			{0xb}: {0xb1},
			{0xc}: nil,
		},
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: {0x11},
				common.Hash{0x2}: nil,
				common.Hash{0x3}: {0x31},
			},
			{0xb}: {
				common.Hash{0x1}: {0x11},
			},
			{0xc}: {
				common.Hash{0x1}: nil,
			},
		},
		false,
	)
	a.merge(b)
	a.revertTo(
		map[common.Hash][]byte{
			{0xa}: {0xa0},
			{0xb}: {0xb0},
			{0xc}: {0xc0},
		},
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20},
				common.Hash{0x3}: nil,
			},
			{0xb}: {
				common.Hash{0x1}: {0x10},
			},
			{0xc}: {
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
	blob, exist = a.storage(common.Hash{0xa}, common.Hash{0x3})
	if !exist || len(blob) != 0 {
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

// TestStateRevertAccountNullMarker tests the scenario that account x did not exist
// before and was created during transition w, reverting w will retain an x=nil
// entry in the set.
func TestStateRevertAccountNullMarker(t *testing.T) {
	a := newStates(nil, nil, false) // empty initial state
	b := newStates(
		map[common.Hash][]byte{
			{0xa}: {0xa},
		},
		nil,
		false,
	)
	a.merge(b) // create account 0xa
	a.revertTo(
		map[common.Hash][]byte{
			{0xa}: nil,
		},
		nil,
	) // revert the transition b

	blob, exist := a.account(common.Hash{0xa})
	if !exist {
		t.Fatal("null marker is not found")
	}
	if len(blob) != 0 {
		t.Fatalf("Unexpected value for account, %v", blob)
	}
}

// TestStateRevertStorageNullMarker tests the scenario that slot x did not exist
// before and was created during transition w, reverting w will retain an x=nil
// entry in the set.
func TestStateRevertStorageNullMarker(t *testing.T) {
	a := newStates(map[common.Hash][]byte{
		{0xa}: {0xa},
	}, nil, false) // initial state with account 0xa

	b := newStates(
		nil,
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: {0x1},
			},
		},
		false,
	)
	a.merge(b) // create slot 0x1
	a.revertTo(
		nil,
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: nil,
			},
		},
	) // revert the transition b

	blob, exist := a.storage(common.Hash{0xa}, common.Hash{0x1})
	if !exist {
		t.Fatal("null marker is not found")
	}
	if len(blob) != 0 {
		t.Fatalf("Unexpected value for storage slot, %v", blob)
	}
}

func TestStatesEncode(t *testing.T) {
	testStatesEncode(t, false)
	testStatesEncode(t, true)
}

func testStatesEncode(t *testing.T, rawStorageKey bool) {
	s := newStates(
		map[common.Hash][]byte{
			{0x1}: {0x1},
		},
		map[common.Hash]map[common.Hash][]byte{
			{0x1}: {
				common.Hash{0x1}: {0x1},
			},
		},
		rawStorageKey,
	)
	buf := bytes.NewBuffer(nil)
	if err := s.encode(buf); err != nil {
		t.Fatalf("Failed to encode states, %v", err)
	}
	var dec stateSet
	if err := dec.decode(rlp.NewStream(buf, 0)); err != nil {
		t.Fatalf("Failed to decode states, %v", err)
	}
	if !reflect.DeepEqual(s.accountData, dec.accountData) {
		t.Fatal("Unexpected account data")
	}
	if !reflect.DeepEqual(s.storageData, dec.storageData) {
		t.Fatal("Unexpected storage data")
	}
	if s.rawStorageKey != dec.rawStorageKey {
		t.Fatal("Unexpected rawStorageKey flag")
	}
}

func TestStateWithOriginEncode(t *testing.T) {
	testStateWithOriginEncode(t, false)
	testStateWithOriginEncode(t, true)
}

func testStateWithOriginEncode(t *testing.T, rawStorageKey bool) {
	s := NewStateSetWithOrigin(
		map[common.Hash][]byte{
			{0x1}: {0x1},
		},
		map[common.Hash]map[common.Hash][]byte{
			{0x1}: {
				common.Hash{0x1}: {0x1},
			},
		},
		map[common.Address][]byte{
			{0x1}: {0x1},
		},
		map[common.Address]map[common.Hash][]byte{
			{0x1}: {
				common.Hash{0x1}: {0x1},
			},
		},
		rawStorageKey,
	)
	buf := bytes.NewBuffer(nil)
	if err := s.encode(buf); err != nil {
		t.Fatalf("Failed to encode states, %v", err)
	}
	var dec StateSetWithOrigin
	if err := dec.decode(rlp.NewStream(buf, 0)); err != nil {
		t.Fatalf("Failed to decode states, %v", err)
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
	if s.rawStorageKey != dec.rawStorageKey {
		t.Fatal("Unexpected rawStorageKey flag")
	}
}

func TestStateSizeTracking(t *testing.T) {
	expSizeA := 3*(common.HashLength+1) + /* account data */
		2*(2*common.HashLength+1) + /* storage data of 0xa */
		2*common.HashLength + 3 + /* storage data of 0xb */
		2*common.HashLength + 1 /* storage data of 0xc */

	a := newStates(
		map[common.Hash][]byte{
			{0xa}: {0xa0}, // common.HashLength+1
			{0xb}: {0xb0}, // common.HashLength+1
			{0xc}: {0xc0}, // common.HashLength+1
		},
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: {0x10}, // 2*common.HashLength+1
				common.Hash{0x2}: {0x20}, // 2*common.HashLength+1
			},
			{0xb}: {
				common.Hash{0x1}: {0x10, 0x11, 0x12}, // 2*common.HashLength+3
			},
			{0xc}: {
				common.Hash{0x1}: {0x10}, // 2*common.HashLength+1
			},
		},
		false,
	)
	if a.size != uint64(expSizeA) {
		t.Fatalf("Unexpected size, want: %d, got: %d", expSizeA, a.size)
	}

	expSizeB := common.HashLength + 2 + common.HashLength + 3 + common.HashLength + /* account data */
		2*common.HashLength + 3 + 2*common.HashLength + 2 + /* storage data of 0xa */
		2*common.HashLength + 2 + 2*common.HashLength + 2 + /* storage data of 0xb */
		3*2*common.HashLength /* storage data of 0xc */
	b := newStates(
		map[common.Hash][]byte{
			{0xa}: {0xa1, 0xa1},       // common.HashLength+2
			{0xb}: {0xb1, 0xb1, 0xb1}, // common.HashLength+3
			{0xc}: nil,                // common.HashLength, account deletion
		},
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: {0x11, 0x11, 0x11}, // 2*common.HashLength+3
				common.Hash{0x3}: {0x31, 0x31},       // 2*common.HashLength+2, slot creation
			},
			{0xb}: {
				common.Hash{0x1}: {0x11, 0x11}, // 2*common.HashLength+2
				common.Hash{0x2}: {0x22, 0x22}, // 2*common.HashLength+2, slot creation
			},
			// The storage of 0xc is entirely removed
			{0xc}: {
				common.Hash{0x1}: nil, // 2*common.HashLength, slot deletion
				common.Hash{0x2}: nil, // 2*common.HashLength, slot deletion
				common.Hash{0x3}: nil, // 2*common.HashLength, slot deletion
			},
		},
		false,
	)
	if b.size != uint64(expSizeB) {
		t.Fatalf("Unexpected size, want: %d, got: %d", expSizeB, b.size)
	}

	a.merge(b)
	mergeSize := expSizeA + 1 /* account a data change */ + 2 /* account b data change */ - 1 /* account c data change */
	mergeSize += 2*common.HashLength + 2 + 2                                                  /* storage a change */
	mergeSize += 2*common.HashLength + 2 - 1                                                  /* storage b change */
	mergeSize += 2*2*common.HashLength - 1                                                    /* storage data removal of 0xc */

	if a.size != uint64(mergeSize) {
		t.Fatalf("Unexpected size, want: %d, got: %d", mergeSize, a.size)
	}

	// Revert the set to original status
	a.revertTo(
		map[common.Hash][]byte{
			{0xa}: {0xa0},
			{0xb}: {0xb0},
			{0xc}: {0xc0},
		},
		map[common.Hash]map[common.Hash][]byte{
			{0xa}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20},
				common.Hash{0x3}: nil, // revert slot creation
			},
			{0xb}: {
				common.Hash{0x1}: {0x10, 0x11, 0x12},
				common.Hash{0x2}: nil, // revert slot creation
			},
			{0xc}: {
				common.Hash{0x1}: {0x10},
				common.Hash{0x2}: {0x20}, // resurrected slot
				common.Hash{0x3}: {0x30}, // resurrected slot
			},
		},
	)
	revertSize := expSizeA + 2*common.HashLength + 2*common.HashLength // delete-marker of a.3 and b.2 slot
	revertSize += 2 * (2*common.HashLength + 1)                        // resurrected slot, c.2, c.3
	if a.size != uint64(revertSize) {
		t.Fatalf("Unexpected size, want: %d, got: %d", revertSize, a.size)
	}
}
