// Contains the node database, storing previously seen nodes and any collected
// metadata about them for QoS purposes.

package discover

import (
	"bytes"
	"encoding/binary"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

// nodeDB stores all nodes we know about.
type nodeDB struct {
	lvl *leveldb.DB
}

// Schema layout for the node database
var (
	nodeDBVersionKey = []byte("version") // Version of the database to flush if changes
	nodeDBStartupKey = []byte("startup") // Time when the node discovery started (seed selection)
	nodeDBItemPrefix = []byte("n:")      // Identifier to prefix node entries with

	nodeDBDiscoverRoot = ":discover"
	nodeDBDiscoverPing = nodeDBDiscoverRoot + ":lastping"
	nodeDBDiscoverBond = nodeDBDiscoverRoot + ":lastbond"
)

// newNodeDB creates a new node database for storing and retrieving infos about
// known peers in the network. If no path is given, an in-memory, temporary
// database is constructed.
func newNodeDB(path string) (*nodeDB, error) {
	if path == "" {
		return newMemoryNodeDB()
	}
	return newPersistentNodeDB(path)
}

// newMemoryNodeDB creates a new in-memory node database without a persistent
// backend.
func newMemoryNodeDB() (*nodeDB, error) {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return &nodeDB{lvl: db}, nil
}

// newPersistentNodeDB creates/opens a leveldb backed persistent node database,
// also flushing its contents in case of a version mismatch.
func newPersistentNodeDB(path string) (*nodeDB, error) {
	// Try to open the cache, recovering any corruption
	db, err := leveldb.OpenFile(path, nil)
	if _, iscorrupted := err.(leveldb.ErrCorrupted); iscorrupted {
		db, err = leveldb.RecoverFile(path, nil)
	}
	if err != nil {
		return nil, err
	}
	// The nodes contained in the cache correspond to a certain protocol version.
	// Flush all nodes if the version doesn't match.
	currentVer := make([]byte, binary.MaxVarintLen64)
	currentVer = currentVer[:binary.PutVarint(currentVer, Version)]

	blob, err := db.Get(nodeDBVersionKey, nil)
	switch err {
	case leveldb.ErrNotFound:
		// Version not found (i.e. empty cache), insert it
		if err := db.Put(nodeDBVersionKey, currentVer, nil); err != nil {
			db.Close()
			return nil, err
		}

	case nil:
		// Version present, flush if different
		if !bytes.Equal(blob, currentVer) {
			db.Close()
			if err = os.RemoveAll(path); err != nil {
				return nil, err
			}
			return newPersistentNodeDB(path)
		}
	}
	return &nodeDB{lvl: db}, nil
}

// makeKey generates the leveldb key-blob from a node id and its particular
// field of interest.
func makeKey(id NodeID, field string) []byte {
	return append(nodeDBItemPrefix, append(id[:], field...)...)
}

// splitKey tries to split a database key into a node id and a field part.
func splitKey(key []byte) (id NodeID, field string) {
	// If the key is not of a node, return it plainly
	if !bytes.HasPrefix(key, nodeDBItemPrefix) {
		return NodeID{}, string(key)
	}
	// Otherwise split the id and field
	item := key[len(nodeDBItemPrefix):]
	copy(id[:], item[:len(id)])
	field = string(item[len(id):])

	return id, field
}

// fetchInt64 retrieves an integer instance associated with a particular
// database key.
func (db *nodeDB) fetchInt64(key []byte) int64 {
	blob, err := db.lvl.Get(key, nil)
	if err != nil {
		return 0
	}
	val, read := binary.Varint(blob)
	if read <= 0 {
		return 0
	}
	return val
}

// storeInt64 update a specific database entry to the current time instance as a
// unix timestamp.
func (db *nodeDB) storeInt64(key []byte, n int64) error {
	blob := make([]byte, binary.MaxVarintLen64)
	blob = blob[:binary.PutVarint(blob, n)]

	return db.lvl.Put(key, blob, nil)
}

// startup retrieves the time instance when the bootstrapping last begun. Its
// purpose is to prevent contacting potential seed nodes multiple times in the
// same boot cycle.
func (db *nodeDB) startup() time.Time {
	return time.Unix(db.fetchInt64(nodeDBStartupKey), 0)
}

// updateStartup updates the bootstrap initiation time to the one specified.
func (db *nodeDB) updateStartup(instance time.Time) error {
	return db.storeInt64(nodeDBStartupKey, instance.Unix())
}

// node retrieves a node with a given id from the database.
func (db *nodeDB) node(id NodeID) *Node {
	blob, err := db.lvl.Get(makeKey(id, nodeDBDiscoverRoot), nil)
	if err != nil {
		return nil
	}
	node := new(Node)
	if err := rlp.DecodeBytes(blob, node); err != nil {
		return nil
	}
	return node
}

// updateNode inserts - potentially overwriting - a node into the peer database.
func (db *nodeDB) updateNode(node *Node) error {
	blob, err := rlp.EncodeToBytes(node)
	if err != nil {
		return err
	}
	return db.lvl.Put(makeKey(node.ID, nodeDBDiscoverRoot), blob, nil)
}

// lastPing retrieves the time of the last ping packet send to a remote node,
// requesting binding.
func (db *nodeDB) lastPing(id NodeID) time.Time {
	return time.Unix(db.fetchInt64(makeKey(id, nodeDBDiscoverPing)), 0)
}

// updateLastPing updates the last time we tried contacting a remote node.
func (db *nodeDB) updateLastPing(id NodeID, instance time.Time) error {
	return db.storeInt64(makeKey(id, nodeDBDiscoverPing), instance.Unix())
}

// lastBond retrieves the time of the last successful bonding with a remote node.
func (db *nodeDB) lastBond(id NodeID) time.Time {
	return time.Unix(db.fetchInt64(makeKey(id, nodeDBDiscoverBond)), 0)
}

// updateLastBond updates the last time we successfully bound to a remote node.
func (db *nodeDB) updateLastBond(id NodeID, instance time.Time) error {
	return db.storeInt64(makeKey(id, nodeDBDiscoverBond), instance.Unix())
}

// querySeeds retrieves a batch of nodes to be used as potential seed servers
// during bootstrapping the node into the network.
//
// Ideal seeds are the most recently seen nodes (highest probability to be still
// alive), but yet untried. However, since leveldb only supports dumb iteration
// we will instead start pulling in potential seeds that haven't been yet pinged
// since the start of the boot procedure.
//
// If the database runs out of potential seeds, we restart the startup counter
// and start iterating over the peers again.
func (db *nodeDB) querySeeds(n int) []*Node {
	startup := db.startup()

	it := db.lvl.NewIterator(nil, nil)
	defer it.Release()

	nodes := make([]*Node, 0, n)
	for len(nodes) < n && it.Next() {
		// Iterate until a discovery node is found
		id, field := splitKey(it.Key())
		if field != nodeDBDiscoverRoot {
			continue
		}
		// Retrieve the last ping time, and if older than startup, query
		lastPing := db.lastPing(id)
		if lastPing.Before(startup) {
			if node := db.node(id); node != nil {
				nodes = append(nodes, node)
			}
		}
	}
	// Reset the startup time if no seeds were found
	if len(nodes) == 0 {
		db.updateStartup(time.Now())
	}
	return nodes
}

// close flushes and closes the database files.
func (db *nodeDB) close() {
	db.lvl.Close()
}
