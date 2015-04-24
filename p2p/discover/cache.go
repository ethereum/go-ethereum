// Contains the discovery cache, storing previously seen nodes to act as seed
// servers during bootstrapping the network.

package discover

import (
	"bytes"
	"encoding/binary"
	"net"
	"os"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

// Cache stores all nodes we know about.
type Cache struct {
	db *leveldb.DB
}

// Cache version to allow dumping old data if it changes.
var cacheVersionKey = []byte("pv")

// NewMemoryCache creates a new in-memory peer cache without a persistent backend.
func NewMemoryCache() (*Cache, error) {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return &Cache{db: db}, nil
}

// NewPersistentCache creates/opens a leveldb backed persistent peer cache, also
// flushing its contents in case of a version mismatch.
func NewPersistentCache(path string) (*Cache, error) {
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

	blob, err := db.Get(cacheVersionKey, nil)
	switch err {
	case leveldb.ErrNotFound:
		// Version not found (i.e. empty cache), insert it
		err = db.Put(cacheVersionKey, currentVer, nil)

	case nil:
		// Version present, flush if different
		if !bytes.Equal(blob, currentVer) {
			db.Close()
			if err = os.RemoveAll(path); err != nil {
				return nil, err
			}
			return NewPersistentCache(path)
		}
	}
	// Clean up in case of an error
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Cache{db: db}, nil
}

// get retrieves a node with a given id from the seed da
func (c *Cache) get(id NodeID) *Node {
	blob, err := c.db.Get(id[:], nil)
	if err != nil {
		return nil
	}
	node := new(Node)
	if err := rlp.DecodeBytes(blob, node); err != nil {
		return nil
	}
	return node
}

// list retrieves a batch of nodes from the database.
func (c *Cache) list(n int) []*Node {
	it := c.db.NewIterator(nil, nil)
	defer it.Release()

	nodes := make([]*Node, 0, n)
	for i := 0; i < n && it.Next(); i++ {
		var id NodeID
		copy(id[:], it.Key())

		if node := c.get(id); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// update inserts - potentially overwriting - a node in the seed database.
func (c *Cache) update(node *Node) error {
	blob, err := rlp.EncodeToBytes(node)
	if err != nil {
		return err
	}
	return c.db.Put(node.ID[:], blob, nil)
}

// add inserts a new node into the seed database.
func (c *Cache) add(id NodeID, addr *net.UDPAddr, tcpPort uint16) *Node {
	node := &Node{
		ID:       id,
		IP:       addr.IP,
		DiscPort: addr.Port,
		TCPPort:  int(tcpPort),
	}
	c.update(node)

	return node
}

// delete removes a node from the database.
func (c *Cache) delete(id NodeID) error {
	return c.db.Delete(id[:], nil)
}

// Close flushes and closes the database files.
func (c *Cache) Close() {
	c.db.Close()
}
