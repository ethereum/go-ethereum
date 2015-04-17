package discover

import (
	"bytes"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

var nodeDBKeyTests = []struct {
	id    NodeID
	field string
	key   []byte
}{
	{
		id:    NodeID{},
		field: "version",
		key:   []byte{0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e}, // field
	},
	{
		id:    MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
		field: ":discover",
		key: []byte{0x6e, 0x3a, // prefix
			0x1d, 0xd9, 0xd6, 0x5c, 0x45, 0x52, 0xb5, 0xeb, // node id
			0x43, 0xd5, 0xad, 0x55, 0xa2, 0xee, 0x3f, 0x56, //
			0xc6, 0xcb, 0xc1, 0xc6, 0x4a, 0x5c, 0x8d, 0x65, //
			0x9f, 0x51, 0xfc, 0xd5, 0x1b, 0xac, 0xe2, 0x43, //
			0x51, 0x23, 0x2b, 0x8d, 0x78, 0x21, 0x61, 0x7d, //
			0x2b, 0x29, 0xb5, 0x4b, 0x81, 0xcd, 0xef, 0xb9, //
			0xb3, 0xe9, 0xc3, 0x7d, 0x7f, 0xd5, 0xf6, 0x32, //
			0x70, 0xbc, 0xc9, 0xe1, 0xa6, 0xf6, 0xa4, 0x39, //
			0x3a, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, // field
		},
	},
}

func TestNodeDBKeys(t *testing.T) {
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

func TestNodeDBInt64(t *testing.T) {
	db, _ := newNodeDB("", Version)
	defer db.close()

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

func TestNodeDBFetchStore(t *testing.T) {
	node := &Node{
		ID:  MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
		IP:  net.IP([]byte{192, 168, 0, 1}),
		UDP: 30303,
		TCP: 30303,
	}
	inst := time.Now()

	db, _ := newNodeDB("", Version)
	defer db.close()

	// Check fetch/store operations on a node ping object
	if stored := db.lastPing(node.ID); stored.Unix() != 0 {
		t.Errorf("ping: non-existing object: %v", stored)
	}
	if err := db.updateLastPing(node.ID, inst); err != nil {
		t.Errorf("ping: failed to update: %v", err)
	}
	if stored := db.lastPing(node.ID); stored.Unix() != inst.Unix() {
		t.Errorf("ping: value mismatch: have %v, want %v", stored, inst)
	}
	// Check fetch/store operations on a node pong object
	if stored := db.lastPong(node.ID); stored.Unix() != 0 {
		t.Errorf("pong: non-existing object: %v", stored)
	}
	if err := db.updateLastPong(node.ID, inst); err != nil {
		t.Errorf("pong: failed to update: %v", err)
	}
	if stored := db.lastPong(node.ID); stored.Unix() != inst.Unix() {
		t.Errorf("pong: value mismatch: have %v, want %v", stored, inst)
	}
	// Check fetch/store operations on an actual node object
	if stored := db.node(node.ID); stored != nil {
		t.Errorf("node: non-existing object: %v", stored)
	}
	if err := db.updateNode(node); err != nil {
		t.Errorf("node: failed to update: %v", err)
	}
	if stored := db.node(node.ID); stored == nil {
		t.Errorf("node: not found")
	} else if !reflect.DeepEqual(stored, node) {
		t.Errorf("node: data mismatch: have %v, want %v", stored, node)
	}
}

var nodeDBSeedQueryNodes = []struct {
	node Node
	pong time.Time
}{
	{
		node: Node{
			ID: MustHexID("0x01d9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			IP: []byte{127, 0, 0, 1},
		},
		pong: time.Now().Add(-2 * time.Second),
	},
	{
		node: Node{
			ID: MustHexID("0x02d9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			IP: []byte{127, 0, 0, 2},
		},
		pong: time.Now().Add(-3 * time.Second),
	},
	{
		node: Node{
			ID: MustHexID("0x03d9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			IP: []byte{127, 0, 0, 3},
		},
		pong: time.Now().Add(-1 * time.Second),
	},
}

func TestNodeDBSeedQuery(t *testing.T) {
	db, _ := newNodeDB("", Version)
	defer db.close()

	// Insert a batch of nodes for querying
	for i, seed := range nodeDBSeedQueryNodes {
		if err := db.updateNode(&seed.node); err != nil {
			t.Fatalf("node %d: failed to insert: %v", i, err)
		}
	}
	// Retrieve the entire batch and check for duplicates
	seeds := db.querySeeds(2 * len(nodeDBSeedQueryNodes))
	if len(seeds) != len(nodeDBSeedQueryNodes) {
		t.Errorf("seed count mismatch: have %v, want %v", len(seeds), len(nodeDBSeedQueryNodes))
	}
	have := make(map[NodeID]struct{})
	for _, seed := range seeds {
		have[seed.ID] = struct{}{}
	}
	want := make(map[NodeID]struct{})
	for _, seed := range nodeDBSeedQueryNodes {
		want[seed.node.ID] = struct{}{}
	}
	for id, _ := range have {
		if _, ok := want[id]; !ok {
			t.Errorf("extra seed: %v", id)
		}
	}
	for id, _ := range want {
		if _, ok := have[id]; !ok {
			t.Errorf("missing seed: %v", id)
		}
	}
	// Make sure the next batch is empty (seed EOF)
	seeds = db.querySeeds(2 * len(nodeDBSeedQueryNodes))
	if len(seeds) != 0 {
		t.Errorf("seed count mismatch: have %v, want %v", len(seeds), 0)
	}
}

func TestNodeDBSeedQueryContinuation(t *testing.T) {
	db, _ := newNodeDB("", Version)
	defer db.close()

	// Insert a batch of nodes for querying
	for i, seed := range nodeDBSeedQueryNodes {
		if err := db.updateNode(&seed.node); err != nil {
			t.Fatalf("node %d: failed to insert: %v", i, err)
		}
	}
	// Iteratively retrieve the batch, checking for an empty batch on reset
	for i := 0; i < len(nodeDBSeedQueryNodes); i++ {
		if seeds := db.querySeeds(1); len(seeds) != 1 {
			t.Errorf("1st iteration %d: seed count mismatch: have %v, want %v", i, len(seeds), 1)
		}
	}
	if seeds := db.querySeeds(1); len(seeds) != 0 {
		t.Errorf("reset: seed count mismatch: have %v, want %v", len(seeds), 0)
	}
	for i := 0; i < len(nodeDBSeedQueryNodes); i++ {
		if seeds := db.querySeeds(1); len(seeds) != 1 {
			t.Errorf("2nd iteration %d: seed count mismatch: have %v, want %v", i, len(seeds), 1)
		}
	}
}

func TestNodeDBPersistency(t *testing.T) {
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
	db, err := newNodeDB(filepath.Join("root", "database"), Version)
	if err != nil {
		t.Fatalf("failed to create persistent database: %v", err)
	}
	if err := db.storeInt64(testKey, testInt); err != nil {
		t.Fatalf("failed to store value: %v.", err)
	}
	db.close()

	// Reopen the database and check the value
	db, err = newNodeDB(filepath.Join("root", "database"), Version)
	if err != nil {
		t.Fatalf("failed to open persistent database: %v", err)
	}
	if val := db.fetchInt64(testKey); val != testInt {
		t.Fatalf("value mismatch: have %v, want %v", val, testInt)
	}
	db.close()

	// Change the database version and check flush
	db, err = newNodeDB(filepath.Join("root", "database"), Version+1)
	if err != nil {
		t.Fatalf("failed to open persistent database: %v", err)
	}
	if val := db.fetchInt64(testKey); val != 0 {
		t.Fatalf("value mismatch: have %v, want %v", val, 0)
	}
	db.close()
}

var nodeDBExpirationNodes = []struct {
	node Node
	pong time.Time
	exp  bool
}{
	{
		node: Node{
			ID: MustHexID("0x01d9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			IP: []byte{127, 0, 0, 1},
		},
		pong: time.Now().Add(-nodeDBNodeExpiration + time.Minute),
		exp:  false,
	}, {
		node: Node{
			ID: MustHexID("0x02d9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			IP: []byte{127, 0, 0, 2},
		},
		pong: time.Now().Add(-nodeDBNodeExpiration - time.Minute),
		exp:  true,
	},
}

func TestNodeDBExpiration(t *testing.T) {
	db, _ := newNodeDB("", Version)
	defer db.close()

	// Add all the test nodes and set their last pong time
	for i, seed := range nodeDBExpirationNodes {
		if err := db.updateNode(&seed.node); err != nil {
			t.Fatalf("node %d: failed to insert: %v", i, err)
		}
		if err := db.updateLastPong(seed.node.ID, seed.pong); err != nil {
			t.Fatalf("node %d: failed to update pong: %v", i, err)
		}
	}
	// Expire some of them, and check the rest
	if err := db.expireNodes(); err != nil {
		t.Fatalf("failed to expire nodes: %v", err)
	}
	for i, seed := range nodeDBExpirationNodes {
		node := db.node(seed.node.ID)
		if (node == nil && !seed.exp) || (node != nil && seed.exp) {
			t.Errorf("node %d: expiration mismatch: have %v, want %v", i, node, seed.exp)
		}
	}
}
