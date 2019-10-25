// Copyright 2019 The go-ethereum Authors
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

package light

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// headTiledSection tracks the latest known tiled section index.
	headTiledSection = []byte("LastSection")
	tablePrefix      = "cht-tile-v10" // tablePrefix is the namespace of tile database.
	tilePrefix       = []byte("t")    // tilePrefix + level(uint8) + position(uint64 big endian) -> tile
	fullnodeChildren = 16             // Each completed full node has 16 children(not include the value of itself)
	nibbleLen        = 16             // The length of nibbles of cht path(term is not included).

	// levelDivisors is the divisors of each tile level. levelDivisors can be used
	// for calculating tile number in each level.
	//
	// e.g. if the size of section is N, the level0 tile number is N/16, level1 tile
	// number is N/4096.
	//
	// We only maintain three level divisors here, since it's enough.
	levelDivisors = []int{16, 4096, 1048576}
)

var errNoCommittedCHT = errors.New("no committed cht for tiles generation")

// chtTiler is reponsible for creating CHT tiles whenever a new section
// is committed.
type chtTiler struct {
	size     uint64         // The number of records in one section
	levels   int            // The number of levels we can build immutable tiles
	db       ethdb.Database // The main database used to store all other data.
	table    ethdb.Database // The database used to store all tiles relative records
	chtTable ethdb.Database // The database used to store all cht nodes.

	taskCh  chan uint64
	wg      sync.WaitGroup
	closeCh chan struct{}
}

func newCHTTiler(db ethdb.Database, size uint64, sectionCount uint64) *chtTiler {
	// Ensure we can build some complete tiles.
	if size < uint64(fullnodeChildren) {
		return nil
	}
	tiler := &chtTiler{
		size:     size,
		levels:   MaxTileLevels(size),
		db:       db,
		table:    rawdb.NewTable(db, tablePrefix),
		chtTable: rawdb.NewTable(db, ChtTablePrefix),
		taskCh:   make(chan uint64, 64),
		closeCh:  make(chan struct{}),
	}
	tiler.wg.Add(1)
	go tiler.run(sectionCount)
	return tiler
}

func (t *chtTiler) run(sectionCount uint64) {
	defer t.wg.Done()
	defer log.Debug("chtTiler stopped")

	var (
		// head is the lastest known tiled section index, nil means no one.
		head = readHeadSection(t.table)

		// taskQueue contains all un-processed sections.
		taskQueue []uint64
	)
	// Generate initial tiling tasks.
	if head == nil {
		for i := uint64(0); i < sectionCount; i++ {
			taskQueue = append(taskQueue, i)
		}
	} else {
		// Double check whether some committed sections have
		// been reverted. If so, re-run the tile task.
		for i := uint64(0); i <= *head; i++ {
			num := (i+1)*t.size - 1
			hash := rawdb.ReadCanonicalHash(t.db, num)
			if tile, _ := readCHTTile(t.table, GetChtRoot(t.db, num, hash)); len(tile) == 0 {
				taskQueue = append(taskQueue, i)
				log.Debug("Readd commited section for tiling", "section", i)
			}
		}
		for i := *head + 1; i < sectionCount; i++ {
			taskQueue = append(taskQueue, i)
		}
	}
	// newTrie initialises the cht trie of given section.
	newTrie := func(section uint64) (*trie.Trie, error) {
		// Calculate the number of the last block in the specified section
		number := (section+1)*t.size - 1
		hash := rawdb.ReadCanonicalHash(t.db, number)

		root := GetChtRoot(t.db, section, hash)
		if root == (common.Hash{}) {
			return nil, errNoCommittedCHT
		}
		t, err := trie.New(root, trie.NewDatabaseWithCache(t.chtTable, 1))
		if err != nil {
			return nil, err
		}
		return t, nil
	}
	// createTiles creates tiles for new generated cht branches in new section.
	createTiles := func(section uint64) error {
		defer func(start time.Time) {
			log.Info("Created tiles", "section", section, "elasped", common.PrettyDuration(time.Since(start)))
		}(time.Now())

		var iter trie.NodeIterator
		if section == 0 {
			curTrie, err := newTrie(section)
			if err != nil {
				return err
			}
			iter = curTrie.NodeIterator(nil) // Create a iterator for traversing the whole CHT
		} else {
			prevTrie, err := newTrie(section - 1)
			if err != nil {
				return err
			}
			prevIter := prevTrie.NodeIterator(nil)
			curTrie, err := newTrie(section)
			if err != nil {
				return err
			}
			curIter := curTrie.NodeIterator(nil)
			iter, _ = trie.NewDifferenceIterator(prevIter, curIter) // Create a diff iterator for traversing the diff
		}
		// The concrete algorithm for tiling CHT.
		//
		// We divide CHT into several tiles, each has 16 trie nodes + 1 parent
		// (except the topmost one). We can also call tile as node group.
		//
		// For a section, there are "size" number records, specifically in CHT
		// the size is 32768. So that we will have 2048 level0 tiles, 8 level1
		// tiles.
		//
		// All generated tiles are stored in the local database, the key is root
		// of tile. Also all tiles(except topmost one) are immutable, since they
		// are children of a complete full node.
		//
		// The order of trie traverse here is deep first. So the concrete tiling
		// algorithm is pretty simple.
		// If the path length > 14, pack all trie nodes into level0 tile
		// If the path length > 12, pack all trie nodes into level1 tile.
		// For all other nodes, pack them into topmost tile.
		type topnode struct {
			path []byte
			blob []byte
		}
		var (
			tophashes []common.Hash
			topnodes  = make(map[common.Hash]topnode)

			tiles  = make([][][]byte, t.levels)
			heads  = make([]common.Hash, t.levels)
			triedb = trie.NewDatabaseWithCache(t.chtTable, 1)
		)
		pack := func(index int, hash common.Hash) error {
			node, err := triedb.Node(iter.Hash())
			if err != nil {
				return err
			}
			// Record the head trie node hash if tile is empty.
			// Becase order of trie traverse here is deep first,
			// we can always get parent of tile packed first.
			if index < t.levels && len(tiles[index]) == 0 {
				heads[index] = hash
			}
			tiles[index] = append(tiles[index], node)
			if index < t.levels && len(tiles[index]) == fullnodeChildren+1 {
				writeCHTTile(t.table, heads[index], tiles[index])
				tiles[index] = nil
			}
			return nil
		}
		for iter.Next(true) {
			path := iter.Path()
			// Ignore the value node here, since it will be
			// embedded in the parent short node.
			if len(path) == nibbleLen+1 {
				continue
			}
			// If in this level we can't assmble a complete tile,
			// pack all of them into topmost level.
			if (nibbleLen-len(path))/2 >= t.levels {
				node, err := triedb.Node(iter.Hash())
				if err != nil {
					return err
				}
				topnodes[iter.Hash()] = topnode{path: common.CopyBytes(iter.Path()), blob: node}
				tophashes = append(tophashes, iter.Hash())
				continue
			}
			// For the bottom level, we can assemble some complete
			// tiles, pack nodes into corresponding tiles.
			level := (nibbleLen - len(path)) / 2
			if err := pack(level, iter.Hash()); err != nil {
				return err
			}
		}
		// Genrate the topmost tile and persist.
		hash := rawdb.ReadCanonicalHash(t.db, (section+1)*t.size-1)
		root := GetChtRoot(t.db, section, hash)

		// Fetch all old but immutable children and add into the topmost tile.
		curhashes, curnodes := tophashes, topnodes
		for {
			var hashes []common.Hash
			var nodes = make(map[common.Hash]topnode)
			for _, h := range curhashes {
				node := curnodes[h]
				trie.IterateRefs(node.blob, func(path []byte, hash common.Hash) error {
					path = append(node.path, path...)
					if _, exist := topnodes[hash]; !exist && (nibbleLen-len(path))/2 >= t.levels {
						blob, err := triedb.Node(hash)
						if err != nil {
							return err
						}
						nodes[hash] = topnode{path: path, blob: blob}
						hashes = append(hashes, hash)
					}
					return nil
				})
			}
			// Nothing to expand
			if len(hashes) == 0 {
				break
			}
			curhashes, curnodes = hashes, nodes
			for _, h := range hashes {
				topnodes[h] = nodes[h]
			}
		}
		var toptile [][]byte
		for _, node := range topnodes {
			toptile = append(toptile, node.blob)
		}
		writeCHTTile(t.table, root, toptile)
		writeHeadSection(t.table, section)
		return nil
	}
	runTask := func() {
		for len(taskQueue) > 0 {
			createTiles(taskQueue[0])
			taskQueue = taskQueue[1:]
		}
	}
	for {
		runTask()
		select {
		case section := <-t.taskCh:
			taskQueue = append(taskQueue, section)
		case <-t.closeCh:
			return
		}
	}
}

func (t *chtTiler) commit(section uint64) {
	select {
	case t.taskCh <- section:
	case <-t.closeCh:
	}
}

func (t *chtTiler) close() {
	close(t.closeCh)
	t.wg.Wait()
}

// readHeadSection reads the last known tiled section index from database.
func readHeadSection(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(headTiledSection)
	if len(data) != 8 {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

// writeHeadSection writes the lastest known tiled section index to database.
func writeHeadSection(db ethdb.KeyValueWriter, section uint64) {
	var enc [8]byte
	binary.BigEndian.PutUint64(enc[:], section)
	if err := db.Put(headTiledSection, enc[:]); err != nil {
		log.Crit("Failed to store head tiled section", "err", err)
	}
}

// readCHTTile retrieves the relative tiles from database.
func readCHTTile(db ethdb.KeyValueReader, tileKey common.Hash) ([][]byte, error) {
	var key []byte
	key = append(key, tilePrefix...)
	key = append(key, tileKey.Bytes()...)
	enc, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	var tiles [][]byte
	err = rlp.DecodeBytes(enc, &tiles)
	if err != nil {
		return nil, err
	}
	return tiles, nil
}

// writeCHTTile writes generated tiles into the database.
func writeCHTTile(db ethdb.KeyValueWriter, tileKey common.Hash, tile [][]byte) {
	var key []byte
	key = append(key, tilePrefix...)
	key = append(key, tileKey.Bytes()...)
	enc, err := rlp.EncodeToBytes(tile)
	if err != nil {
		log.Crit("Failed to rlp encode the tile blob", "err", err)
	}
	if err := db.Put(key, enc[:]); err != nil {
		log.Crit("Failed to store tile", "err", err)
	}
}

// ReadCHTTile retrieves the relative tiles from database based on the given key.
func ReadCHTTile(db ethdb.Database, tileKey common.Hash) ([][]byte, error) {
	table := rawdb.NewTable(db, tablePrefix)
	return readCHTTile(table, tileKey)
}

// MaxTileLevels calculates the levels of immutable tiles.
func MaxTileLevels(size uint64) int {
	var levels int
	for i := 1; i <= len(levelDivisors); i++ {
		if size/uint64(levelDivisors[i-1]) == 0 {
			levels = i - 1
			break
		}
	}
	return levels
}
