// Copyright 2018 The go-ethereum Authors
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
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Keys in the node database.
const (
	dbVersionKey   = "version" // Version of the database to flush if changes
	dbNodePrefix   = "n:"      // Identifier to prefix node entries with
	dbLocalPrefix  = "local:"
	dbDiscoverRoot = "v4"
	dbDiscv5Root   = "v5"

	// These fields are stored per ID and IP, the full key is "n:<ID>:v4:<IP>:findfail".
	// Use nodeItemKey to create those keys.
	dbNodeFindFails = "findfail"
	dbNodePing      = "lastping"
	dbNodePong      = "lastpong"
	dbNodeSeq       = "seq"

	// Local information is keyed by ID only, the full key is "local:<ID>:seq".
	// Use localItemKey to create those keys.
	dbLocalSeq = "seq"
)

const (
	dbNodeExpiration = 24 * time.Hour // Time after which an unseen node should be dropped.
	dbCleanupCycle   = time.Hour      // Time period for running the expiration task.
	dbVersion        = 10
)

var (
	errInvalidIP = errors.New("invalid IP")
	errNotFound  = errors.New("not found")
)

var zeroIP = netip.IPv6Unspecified()

// kvDatabase is an internal interface that abstracts the underlying key-value database.
// This allows supporting both leveldb and pebble backends.
type kvDatabase interface {
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
	Delete(key []byte) error
	NewIterator(prefix []byte) dbIterator
	Close() error
}

// dbIterator is an internal interface for database iterators.
type dbIterator interface {
	Next() bool
	First() bool
	Seek(key []byte) bool
	Valid() bool
	Key() []byte
	Value() []byte
	Release()
}

// DB is the node database, storing previously seen nodes and any collected metadata about
// them for QoS purposes.
type DB struct {
	kv     kvDatabase    // Interface to the database itself (leveldb or pebble)
	runner sync.Once     // Ensures we can start at most one expirer
	quit   chan struct{} // Channel to signal the expiring thread to stop
}

// levelDB wraps a leveldb database to implement the kvDatabase interface.
type levelDB struct {
	db *leveldb.DB
}

func (l *levelDB) Get(key []byte) ([]byte, error) {
	val, err := l.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, errNotFound
	}
	return val, err
}

func (l *levelDB) Put(key, value []byte) error {
	return l.db.Put(key, value, nil)
}

func (l *levelDB) Delete(key []byte) error {
	return l.db.Delete(key, nil)
}

func (l *levelDB) NewIterator(prefix []byte) dbIterator {
	return &levelDBIterator{
		iter: l.db.NewIterator(util.BytesPrefix(prefix), nil),
	}
}

func (l *levelDB) Close() error {
	return l.db.Close()
}

// levelDBIterator wraps a leveldb iterator.
type levelDBIterator struct {
	iter iterator.Iterator
}

func (it *levelDBIterator) Next() bool {
	return it.iter.Next()
}

func (it *levelDBIterator) First() bool {
	return it.iter.First()
}

func (it *levelDBIterator) Seek(key []byte) bool {
	return it.iter.Seek(key)
}

func (it *levelDBIterator) Valid() bool {
	return it.iter.Valid()
}

func (it *levelDBIterator) Key() []byte {
	return it.iter.Key()
}

func (it *levelDBIterator) Value() []byte {
	return it.iter.Value()
}

func (it *levelDBIterator) Release() {
	it.iter.Release()
}

// pebbleDB wraps a pebble database to implement the kvDatabase interface.
type pebbleDB struct {
	db *pebble.DB
}

func (p *pebbleDB) Get(key []byte) ([]byte, error) {
	val, closer, err := p.db.Get(key)
	if err == pebble.ErrNotFound {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	ret := make([]byte, len(val))
	copy(ret, val)
	return ret, nil
}

func (p *pebbleDB) Put(key, value []byte) error {
	return p.db.Set(key, value, pebble.Sync)
}

func (p *pebbleDB) Delete(key []byte) error {
	return p.db.Delete(key, pebble.Sync)
}

func (p *pebbleDB) NewIterator(prefix []byte) dbIterator {
	iter, _ := p.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: common.UpperBound(prefix),
	})
	return &pebbleDBIterator{iter: iter}
}

func (p *pebbleDB) Close() error {
	return p.db.Close()
}

// pebbleDBIterator wraps a pebble iterator.
type pebbleDBIterator struct {
	iter *pebble.Iterator
}

func (it *pebbleDBIterator) Next() bool {
	return it.iter.Next()
}

func (it *pebbleDBIterator) First() bool {
	return it.iter.First()
}

func (it *pebbleDBIterator) Seek(key []byte) bool {
	return it.iter.SeekGE(key)
}

func (it *pebbleDBIterator) Valid() bool {
	return it.iter.Valid()
}

func (it *pebbleDBIterator) Key() []byte {
	return it.iter.Key()
}

func (it *pebbleDBIterator) Value() []byte {
	return it.iter.Value()
}

func (it *pebbleDBIterator) Release() {
	it.iter.Close()
}

// OpenDB opens a node database for storing and retrieving infos about known peers in the
// network. If no path is given an in-memory, temporary database is constructed.
// The dbType parameter can be "leveldb", "pebble", or "" for auto-detection.
func OpenDB(path string, dbType string) (*DB, error) {
	if path == "" {
		return newMemoryDB()
	}
	return newPersistentDB(path, dbType)
}

// newMemoryDB creates a new in-memory node database without a persistent backend.
// Defaults to pebble for new in-memory databases.
func newMemoryDB() (*DB, error) {
	db, err := pebble.Open("", &pebble.Options{
		FS: vfs.NewMem(),
	})
	if err != nil {
		return nil, err
	}
	return &DB{kv: &pebbleDB{db: db}, quit: make(chan struct{})}, nil
}

// newPersistentDB creates/opens a persistent node database.
// The dbType parameter can specify "leveldb", "pebble", or "" for auto-detection.
// When auto-detecting, it checks for existing databases and defaults to pebble for new databases.
// It also flushes the database contents in case of a version mismatch.
func newPersistentDB(path string, dbType string) (*DB, error) {
	if dbType != "" {
		switch dbType {
		case common.DBPebble:
			return openPebbleDB(path)
		case common.DBLeveldb:
			return openLevelDB(path)
		default:
			return nil, fmt.Errorf("unknown database engine: %s (valid options: leveldb, pebble)", dbType)
		}
	}
	detectedType := common.PreexistingDatabase(path)
	switch detectedType {
	case common.DBPebble:
		return openPebbleDB(path)
	case common.DBLeveldb:
		return openLevelDB(path)
	default:
		return openPebbleDB(path)
	}
}

// checkDBVersion checks the database version and handles version mismatches.
// If the version doesn't match, it removes the old database and reopens it using reopenFn.
func checkDBVersion(kv kvDatabase, path string, reopenFn func(string) (*DB, error)) (*DB, error) {
	currentVer := encodeVarint(int64(dbVersion))

	blob, err := kv.Get([]byte(dbVersionKey))
	switch err {
	case errNotFound:
		// Version not found (i.e. empty cache), insert it
		if err := kv.Put([]byte(dbVersionKey), currentVer); err != nil {
			kv.Close()
			return nil, err
		}

	case nil:
		// Version present, flush if different
		if !bytes.Equal(blob, currentVer) {
			kv.Close()
			if err = os.RemoveAll(path); err != nil {
				return nil, err
			}
			return reopenFn(path)
		}
	}
	return &DB{kv: kv, quit: make(chan struct{})}, nil
}

// openLevelDB opens a leveldb database at the given path.
func openLevelDB(path string) (*DB, error) {
	opts := &opt.Options{OpenFilesCacheCapacity: 5}
	db, err := leveldb.OpenFile(path, opts)
	if _, iscorrupted := err.(*errors.ErrCorrupted); iscorrupted {
		db, err = leveldb.RecoverFile(path, nil)
	}
	if err != nil {
		return nil, err
	}
	return checkDBVersion(&levelDB{db: db}, path, openLevelDB)
}

// openPebbleDB opens a pebble database at the given path.
func openPebbleDB(path string) (*DB, error) {
	opts := &pebble.Options{MaxOpenFiles: 5}
	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, err
	}
	return checkDBVersion(&pebbleDB{db: db}, path, openPebbleDB)
}

// nodeKey returns the database key for a node record.
func nodeKey(id ID) []byte {
	key := append([]byte(dbNodePrefix), id[:]...)
	key = append(key, ':')
	key = append(key, dbDiscoverRoot...)
	return key
}

// splitNodeKey returns the node ID of a key created by nodeKey.
func splitNodeKey(key []byte) (id ID, rest []byte) {
	if !bytes.HasPrefix(key, []byte(dbNodePrefix)) {
		return ID{}, nil
	}
	item := key[len(dbNodePrefix):]
	copy(id[:], item[:len(id)])
	return id, item[len(id)+1:]
}

// nodeItemKey returns the database key for a node metadata field.
func nodeItemKey(id ID, ip netip.Addr, field string) []byte {
	if !ip.IsValid() {
		panic("invalid IP")
	}
	ip16 := ip.As16()
	return bytes.Join([][]byte{nodeKey(id), ip16[:], []byte(field)}, []byte{':'})
}

// splitNodeItemKey returns the components of a key created by nodeItemKey.
func splitNodeItemKey(key []byte) (id ID, ip netip.Addr, field string) {
	id, key = splitNodeKey(key)
	// Skip discover root.
	if string(key) == dbDiscoverRoot {
		return id, netip.Addr{}, ""
	}
	key = key[len(dbDiscoverRoot)+1:]
	// Split out the IP.
	ip, _ = netip.AddrFromSlice(key[:16])
	key = key[16+1:]
	// Field is the remainder of key.
	field = string(key)
	return id, ip, field
}

func v5Key(id ID, ip netip.Addr, field string) []byte {
	ip16 := ip.As16()
	return bytes.Join([][]byte{
		[]byte(dbNodePrefix),
		id[:],
		[]byte(dbDiscv5Root),
		ip16[:],
		[]byte(field),
	}, []byte{':'})
}

// localItemKey returns the key of a local node item.
func localItemKey(id ID, field string) []byte {
	key := append([]byte(dbLocalPrefix), id[:]...)
	key = append(key, ':')
	key = append(key, field...)
	return key
}

// encodeVarint encodes an int64 as a varint.
func encodeVarint(n int64) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	buf = buf[:binary.PutVarint(buf, n)]
	return buf
}

// encodeUvarint encodes a uint64 as a varint.
func encodeUvarint(n uint64) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	buf = buf[:binary.PutUvarint(buf, n)]
	return buf
}

// fetchInt64 retrieves an integer associated with a particular key.
func (db *DB) fetchInt64(key []byte) int64 {
	blob, err := db.kv.Get(key)
	if err != nil {
		return 0
	}
	val, read := binary.Varint(blob)
	if read <= 0 {
		return 0
	}
	return val
}

// storeInt64 stores an integer in the given key.
func (db *DB) storeInt64(key []byte, n int64) error {
	return db.kv.Put(key, encodeVarint(n))
}

// fetchUint64 retrieves an integer associated with a particular key.
func (db *DB) fetchUint64(key []byte) uint64 {
	blob, err := db.kv.Get(key)
	if err != nil {
		return 0
	}
	val, _ := binary.Uvarint(blob)
	return val
}

// storeUint64 stores an integer in the given key.
func (db *DB) storeUint64(key []byte, n uint64) error {
	return db.kv.Put(key, encodeUvarint(n))
}

// Node retrieves a node with a given id from the database.
func (db *DB) Node(id ID) *Node {
	blob, err := db.kv.Get(nodeKey(id))
	if err != nil {
		return nil
	}
	return mustDecodeNode(id[:], blob)
}

func mustDecodeNode(id, data []byte) *Node {
	var r enr.Record
	if err := rlp.DecodeBytes(data, &r); err != nil {
		panic(fmt.Errorf("p2p/enode: can't decode node %x in DB: %v", id, err))
	}
	if len(id) != len(ID{}) {
		panic(fmt.Errorf("invalid id length %d", len(id)))
	}
	return newNodeWithID(&r, ID(id))
}

// UpdateNode inserts - potentially overwriting - a node into the peer database.
func (db *DB) UpdateNode(node *Node) error {
	if node.Seq() < db.NodeSeq(node.ID()) {
		return nil
	}
	blob, err := rlp.EncodeToBytes(&node.r)
	if err != nil {
		return err
	}
	if err := db.kv.Put(nodeKey(node.ID()), blob); err != nil {
		return err
	}
	return db.storeUint64(nodeItemKey(node.ID(), zeroIP, dbNodeSeq), node.Seq())
}

// NodeSeq returns the stored record sequence number of the given node.
func (db *DB) NodeSeq(id ID) uint64 {
	return db.fetchUint64(nodeItemKey(id, zeroIP, dbNodeSeq))
}

// Resolve returns the stored record of the node if it has a larger sequence
// number than n.
func (db *DB) Resolve(n *Node) *Node {
	if n.Seq() > db.NodeSeq(n.ID()) {
		return n
	}
	return db.Node(n.ID())
}

// DeleteNode deletes all information associated with a node.
func (db *DB) DeleteNode(id ID) {
	deleteRange(db.kv, nodeKey(id))
}

func deleteRange(kv kvDatabase, prefix []byte) {
	it := kv.NewIterator(prefix)
	defer it.Release()
	for it.First(); it.Valid(); it.Next() {
		kv.Delete(it.Key())
	}
}

// ensureExpirer is a small helper method ensuring that the data expiration
// mechanism is running. If the expiration goroutine is already running, this
// method simply returns.
//
// The goal is to start the data evacuation only after the network successfully
// bootstrapped itself (to prevent dumping potentially useful seed nodes). Since
// it would require significant overhead to exactly trace the first successful
// convergence, it's simpler to "ensure" the correct state when an appropriate
// condition occurs (i.e. a successful bonding), and discard further events.
func (db *DB) ensureExpirer() {
	db.runner.Do(func() { go db.expirer() })
}

// expirer should be started in a go routine, and is responsible for looping ad
// infinitum and dropping stale data from the database.
func (db *DB) expirer() {
	tick := time.NewTicker(dbCleanupCycle)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			db.expireNodes()
		case <-db.quit:
			return
		}
	}
}

// expireNodes iterates over the database and deletes all nodes that have not
// been seen (i.e. received a pong from) for some time.
func (db *DB) expireNodes() {
	it := db.kv.NewIterator([]byte(dbNodePrefix))
	defer it.Release()

	if !it.First() {
		return
	}

	var (
		threshold    = time.Now().Add(-dbNodeExpiration).Unix()
		youngestPong int64
		atEnd        = false
	)
	for !atEnd {
		id, ip, field := splitNodeItemKey(it.Key())
		if field == dbNodePong {
			time, _ := binary.Varint(it.Value())
			if time > youngestPong {
				youngestPong = time
			}
			if time < threshold {
				// Last pong from this IP older than threshold, remove fields belonging to it.
				deleteRange(db.kv, nodeItemKey(id, ip, ""))
			}
		}
		atEnd = !it.Next()
		nextID, _ := splitNodeKey(it.Key())
		if atEnd || nextID != id {
			// We've moved beyond the last entry of the current ID.
			// Remove everything if there was no recent enough pong.
			if youngestPong > 0 && youngestPong < threshold {
				deleteRange(db.kv, nodeKey(id))
			}
			youngestPong = 0
		}
	}
}

// LastPingReceived retrieves the time of the last ping packet received from
// a remote node.
func (db *DB) LastPingReceived(id ID, ip netip.Addr) time.Time {
	if !ip.IsValid() {
		return time.Time{}
	}
	return time.Unix(db.fetchInt64(nodeItemKey(id, ip, dbNodePing)), 0)
}

// UpdateLastPingReceived updates the last time we tried contacting a remote node.
func (db *DB) UpdateLastPingReceived(id ID, ip netip.Addr, instance time.Time) error {
	if !ip.IsValid() {
		return errInvalidIP
	}
	return db.storeInt64(nodeItemKey(id, ip, dbNodePing), instance.Unix())
}

// LastPongReceived retrieves the time of the last successful pong from remote node.
func (db *DB) LastPongReceived(id ID, ip netip.Addr) time.Time {
	if !ip.IsValid() {
		return time.Time{}
	}
	// Launch expirer
	db.ensureExpirer()
	return time.Unix(db.fetchInt64(nodeItemKey(id, ip, dbNodePong)), 0)
}

// UpdateLastPongReceived updates the last pong time of a node.
func (db *DB) UpdateLastPongReceived(id ID, ip netip.Addr, instance time.Time) error {
	if !ip.IsValid() {
		return errInvalidIP
	}
	return db.storeInt64(nodeItemKey(id, ip, dbNodePong), instance.Unix())
}

// FindFails retrieves the number of findnode failures since bonding.
func (db *DB) FindFails(id ID, ip netip.Addr) int {
	if !ip.IsValid() {
		return 0
	}
	return int(db.fetchInt64(nodeItemKey(id, ip, dbNodeFindFails)))
}

// UpdateFindFails updates the number of findnode failures since bonding.
func (db *DB) UpdateFindFails(id ID, ip netip.Addr, fails int) error {
	if !ip.IsValid() {
		return errInvalidIP
	}
	return db.storeInt64(nodeItemKey(id, ip, dbNodeFindFails), int64(fails))
}

// FindFailsV5 retrieves the discv5 findnode failure counter.
func (db *DB) FindFailsV5(id ID, ip netip.Addr) int {
	if !ip.IsValid() {
		return 0
	}
	return int(db.fetchInt64(v5Key(id, ip, dbNodeFindFails)))
}

// UpdateFindFailsV5 stores the discv5 findnode failure counter.
func (db *DB) UpdateFindFailsV5(id ID, ip netip.Addr, fails int) error {
	if !ip.IsValid() {
		return errInvalidIP
	}
	return db.storeInt64(v5Key(id, ip, dbNodeFindFails), int64(fails))
}

// localSeq retrieves the local record sequence counter, defaulting to the current
// timestamp if no previous exists. This ensures that wiping all data associated
// with a node (apart from its key) will not generate already used sequence nums.
func (db *DB) localSeq(id ID) uint64 {
	if seq := db.fetchUint64(localItemKey(id, dbLocalSeq)); seq > 0 {
		return seq
	}
	return uint64(time.Now().UnixMilli())
}

// storeLocalSeq stores the local record sequence counter.
func (db *DB) storeLocalSeq(id ID, n uint64) {
	db.storeUint64(localItemKey(id, dbLocalSeq), n)
}

// QuerySeeds retrieves random nodes to be used as potential seed nodes
// for bootstrapping.
func (db *DB) QuerySeeds(n int, maxAge time.Duration) []*Node {
	var (
		now   = time.Now()
		nodes = make([]*Node, 0, n)
		it    = db.kv.NewIterator(nil)
		id    ID
	)
	defer it.Release()

seek:
	for seeks := 0; len(nodes) < n && seeks < n*5; seeks++ {
		// Seek to a random entry. The first byte is incremented by a
		// random amount each time in order to increase the likelihood
		// of hitting all existing nodes in very small databases.
		ctr := id[0]
		rand.Read(id[:])
		id[0] = ctr + id[0]%16
		it.Seek(nodeKey(id))

		n := nextNode(it)
		if n == nil {
			id[0] = 0
			continue seek // iterator exhausted
		}
		if now.Sub(db.LastPongReceived(n.ID(), n.IPAddr())) > maxAge {
			continue seek
		}
		for i := range nodes {
			if nodes[i].ID() == n.ID() {
				continue seek // duplicate
			}
		}
		nodes = append(nodes, n)
	}
	return nodes
}

// reads the next node record from the iterator, skipping over other
// database entries.
func nextNode(it dbIterator) *Node {
	for it.Valid() {
		id, rest := splitNodeKey(it.Key())
		if string(rest) == dbDiscoverRoot {
			return mustDecodeNode(id[:], it.Value())
		}
		if !it.Next() {
			break
		}
	}
	return nil
}

// Close flushes and closes the database files.
func (db *DB) Close() {
	select {
	case <-db.quit: // already closed
	default:
		close(db.quit)
	}
	db.kv.Close()
}
