// Copyright 2015 The go-ethereum Authors
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

package enode

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

var nodeDBKeyTests = []struct {
	id    ID
	field string
	key   []byte
}{
	{
		id:    ID{},
		field: "version",
		key:   []byte{0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e}, // field
	},
	{
		id:    HexID("51232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
		field: ":discover",
		key: []byte{
			0x6e, 0x3a, // prefix
			0x51, 0x23, 0x2b, 0x8d, 0x78, 0x21, 0x61, 0x7d, // node id
			0x2b, 0x29, 0xb5, 0x4b, 0x81, 0xcd, 0xef, 0xb9, //
			0xb3, 0xe9, 0xc3, 0x7d, 0x7f, 0xd5, 0xf6, 0x32, //
			0x70, 0xbc, 0xc9, 0xe1, 0xa6, 0xf6, 0xa4, 0x39, //
			0x3a, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, // field
		},
	},
}

func TestDBKeys(t *testing.T) {
	for i, tt := range nodeDBKeyTests {
		if key := makeKey(tt.id, tt.field); !bytes.Equal(key, tt.key) {
			t.Errorf("make test %d: key mismatch: have 0x%x, want 0x%x", i, key, tt.key)
		}
		id, field := splitKey(tt.key)
		if !bytes.Equal(id[:], tt.id[:]) {
			t.Errorf("split test %d: id mismatch: have 0x%x, want 0x%x", i, id, tt.id)
		}
		if field != tt.field {
			t.Errorf("split test %d: field mismatch: have 0x%x, want 0x%x", i, field, tt.field)
		}
	}
}

var nodeDBInt64Tests = []struct {
	key   []byte
	value int64
}{
	{key: []byte{0x01}, value: 1},
	{key: []byte{0x02}, value: 2},
	{key: []byte{0x03}, value: 3},
}

func TestDBInt64(t *testing.T) {
	db, _ := OpenDB("")
	defer db.Close()

	tests := nodeDBInt64Tests
	for i := 0; i < len(tests); i++ {
		// Insert the next value
		if err := db.storeInt64(tests[i].key, tests[i].value); err != nil {
			t.Errorf("test %d: failed to store value: %v", i, err)
		}
		// Check all existing and non existing values
		for j := 0; j < len(tests); j++ {
			num := db.fetchInt64(tests[j].key)
			switch {
			case j <= i && num != tests[j].value:
				t.Errorf("test %d, item %d: value mismatch: have %v, want %v", i, j, num, tests[j].value)
			case j > i && num != 0:
				t.Errorf("test %d, item %d: value mismatch: have %v, want %v", i, j, num, 0)
			}
		}
	}
}

func TestDBFetchStore(t *testing.T) {
	node := NewV4(
		hexPubkey("1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
		net.IP{192, 168, 0, 1},
		30303,
		30303,
	)
	inst := time.Now()
	num := 314

	db, _ := OpenDB("")
	defer db.Close()

	// Check fetch/store operations on a node ping object
	if stored := db.LastPingReceived(node.ID()); stored.Unix() != 0 {
		t.Errorf("ping: non-existing object: %v", stored)
	}
	if err := db.UpdateLastPingReceived(node.ID(), inst); err != nil {
		t.Errorf("ping: failed to update: %v", err)
	}
	if stored := db.LastPingReceived(node.ID()); stored.Unix() != inst.Unix() {
		t.Errorf("ping: value mismatch: have %v, want %v", stored, inst)
	}
	// Check fetch/store operations on a node pong object
	if stored := db.LastPongReceived(node.ID()); stored.Unix() != 0 {
		t.Errorf("pong: non-existing object: %v", stored)
	}
	if err := db.UpdateLastPongReceived(node.ID(), inst); err != nil {
		t.Errorf("pong: failed to update: %v", err)
	}
	if stored := db.LastPongReceived(node.ID()); stored.Unix() != inst.Unix() {
		t.Errorf("pong: value mismatch: have %v, want %v", stored, inst)
	}
	// Check fetch/store operations on a node findnode-failure object
	if stored := db.FindFails(node.ID()); stored != 0 {
		t.Errorf("find-node fails: non-existing object: %v", stored)
	}
	if err := db.UpdateFindFails(node.ID(), num); err != nil {
		t.Errorf("find-node fails: failed to update: %v", err)
	}
	if stored := db.FindFails(node.ID()); stored != num {
		t.Errorf("find-node fails: value mismatch: have %v, want %v", stored, num)
	}
	// Check fetch/store operations on an actual node object
	if stored := db.Node(node.ID()); stored != nil {
		t.Errorf("node: non-existing object: %v", stored)
	}
	if err := db.UpdateNode(node); err != nil {
		t.Errorf("node: failed to update: %v", err)
	}
	if stored := db.Node(node.ID()); stored == nil {
		t.Errorf("node: not found")
	} else if !reflect.DeepEqual(stored, node) {
		t.Errorf("node: data mismatch: have %v, want %v", stored, node)
	}
}

var nodeDBSeedQueryNodes = []struct {
	node *Node
	pong time.Time
}{
	// This one should not be in the result set because its last
	// pong time is too far in the past.
	{
		node: NewV4(
			hexPubkey("1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			net.IP{127, 0, 0, 3},
			30303,
			30303,
		),
		pong: time.Now().Add(-3 * time.Hour),
	},
	// This one shouldn't be in the result set because its
	// nodeID is the local node's ID.
	{
		node: NewV4(
			hexPubkey("ff93ff820abacd4351b0f14e47b324bc82ff014c226f3f66a53535734a3c150e7e38ca03ef0964ba55acddc768f5e99cd59dea95ddd4defbab1339c92fa319b2"),
			net.IP{127, 0, 0, 3},
			30303,
			30303,
		),
		pong: time.Now().Add(-4 * time.Second),
	},

	// These should be in the result set.
	{
		node: NewV4(
			hexPubkey("c2b5eb3f5dde05f815b63777809ee3e7e0cbb20035a6b00ce327191e6eaa8f26a8d461c9112b7ab94698e7361fa19fd647e603e73239002946d76085b6f928d6"),
			net.IP{127, 0, 0, 1},
			30303,
			30303,
		),
		pong: time.Now().Add(-2 * time.Second),
	},
	{
		node: NewV4(
			hexPubkey("6ca1d400c8ddf8acc94bcb0dd254911ad71a57bed5e0ae5aa205beed59b28c2339908e97990c493499613cff8ecf6c3dc7112a8ead220cdcd00d8847ca3db755"),
			net.IP{127, 0, 0, 2},
			30303,
			30303,
		),
		pong: time.Now().Add(-3 * time.Second),
	},
	{
		node: NewV4(
			hexPubkey("234dc63fe4d131212b38236c4c3411288d7bec61cbf7b120ff12c43dc60c96182882f4291d209db66f8a38e986c9c010ff59231a67f9515c7d1668b86b221a47"),
			net.IP{127, 0, 0, 3},
			30303,
			30303,
		),
		pong: time.Now().Add(-1 * time.Second),
	},
	{
		node: NewV4(
			hexPubkey("c013a50b4d1ebce5c377d8af8cb7114fd933ffc9627f96ad56d90fef5b7253ec736fd07ef9a81dc2955a997e54b7bf50afd0aa9f110595e2bec5bb7ce1657004"),
			net.IP{127, 0, 0, 3},
			30303,
			30303,
		),
		pong: time.Now().Add(-2 * time.Second),
	},
	{
		node: NewV4(
			hexPubkey("f141087e3e08af1aeec261ff75f48b5b1637f594ea9ad670e50051646b0416daa3b134c28788cbe98af26992a47652889cd8577ccc108ac02c6a664db2dc1283"),
			net.IP{127, 0, 0, 3},
			30303,
			30303,
		),
		pong: time.Now().Add(-2 * time.Second),
	},
}

func TestDBSeedQuery(t *testing.T) {
	// Querying seeds uses seeks an might not find all nodes
	// every time when the database is small. Run the test multiple
	// times to avoid flakes.
	const attempts = 15
	var err error
	for i := 0; i < attempts; i++ {
		if err = testSeedQuery(); err == nil {
			return
		}
	}
	if err != nil {
		t.Errorf("no successful run in %d attempts: %v", attempts, err)
	}
}

func testSeedQuery() error {
	db, _ := OpenDB("")
	defer db.Close()

	// Insert a batch of nodes for querying
	for i, seed := range nodeDBSeedQueryNodes {
		if err := db.UpdateNode(seed.node); err != nil {
			return fmt.Errorf("node %d: failed to insert: %v", i, err)
		}
		if err := db.UpdateLastPongReceived(seed.node.ID(), seed.pong); err != nil {
			return fmt.Errorf("node %d: failed to insert bondTime: %v", i, err)
		}
	}

	// Retrieve the entire batch and check for duplicates
	seeds := db.QuerySeeds(len(nodeDBSeedQueryNodes)*2, time.Hour)
	have := make(map[ID]struct{})
	for _, seed := range seeds {
		have[seed.ID()] = struct{}{}
	}
	want := make(map[ID]struct{})
	for _, seed := range nodeDBSeedQueryNodes[1:] {
		want[seed.node.ID()] = struct{}{}
	}
	if len(seeds) != len(want) {
		return fmt.Errorf("seed count mismatch: have %v, want %v", len(seeds), len(want))
	}
	for id := range have {
		if _, ok := want[id]; !ok {
			return fmt.Errorf("extra seed: %v", id)
		}
	}
	for id := range want {
		if _, ok := have[id]; !ok {
			return fmt.Errorf("missing seed: %v", id)
		}
	}
	return nil
}

func TestDBPersistency(t *testing.T) {
	root, err := ioutil.TempDir("", "nodedb-")
	if err != nil {
		t.Fatalf("failed to create temporary data folder: %v", err)
	}
	defer os.RemoveAll(root)

	var (
		testKey = []byte("somekey")
		testInt = int64(314)
	)

	// Create a persistent database and store some values
	db, err := OpenDB(filepath.Join(root, "database"))
	if err != nil {
		t.Fatalf("failed to create persistent database: %v", err)
	}
	if err := db.storeInt64(testKey, testInt); err != nil {
		t.Fatalf("failed to store value: %v.", err)
	}
	db.Close()

	// Reopen the database and check the value
	db, err = OpenDB(filepath.Join(root, "database"))
	if err != nil {
		t.Fatalf("failed to open persistent database: %v", err)
	}
	if val := db.fetchInt64(testKey); val != testInt {
		t.Fatalf("value mismatch: have %v, want %v", val, testInt)
	}
	db.Close()
}

var nodeDBExpirationNodes = []struct {
	node *Node
	pong time.Time
	exp  bool
}{
	{
		node: NewV4(
			hexPubkey("8d110e2ed4b446d9b5fb50f117e5f37fb7597af455e1dab0e6f045a6eeaa786a6781141659020d38bdc5e698ed3d4d2bafa8b5061810dfa63e8ac038db2e9b67"),
			net.IP{127, 0, 0, 1},
			30303,
			30303,
		),
		pong: time.Now().Add(-dbNodeExpiration + time.Minute),
		exp:  false,
	}, {
		node: NewV4(
			hexPubkey("913a205579c32425b220dfba999d215066e5bdbf900226b11da1907eae5e93eb40616d47412cf819664e9eacbdfcca6b0c6e07e09847a38472d4be46ab0c3672"),
			net.IP{127, 0, 0, 2},
			30303,
			30303,
		),
		pong: time.Now().Add(-dbNodeExpiration - time.Minute),
		exp:  true,
	},
}

func TestDBExpiration(t *testing.T) {
	db, _ := OpenDB("")
	defer db.Close()

	// Add all the test nodes and set their last pong time
	for i, seed := range nodeDBExpirationNodes {
		if err := db.UpdateNode(seed.node); err != nil {
			t.Fatalf("node %d: failed to insert: %v", i, err)
		}
		if err := db.UpdateLastPongReceived(seed.node.ID(), seed.pong); err != nil {
			t.Fatalf("node %d: failed to update bondTime: %v", i, err)
		}
	}
	// Expire some of them, and check the rest
	if err := db.expireNodes(); err != nil {
		t.Fatalf("failed to expire nodes: %v", err)
	}
	for i, seed := range nodeDBExpirationNodes {
		node := db.Node(seed.node.ID())
		if (node == nil && !seed.exp) || (node != nil && seed.exp) {
			t.Errorf("node %d: expiration mismatch: have %v, want %v", i, node, seed.exp)
		}
	}
}
