// Copyright 2017 The go-ethereum Authors
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
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// IndexerConfig includes a set of configs for chain indexers.
type IndexerConfig struct {
	// The block frequency for creating CHTs.
	ChtSize uint64

	// The number of confirmations needed to generate/accept a canonical hash help trie.
	ChtConfirms uint64

	// The block frequency for creating new bloom bits.
	BloomSize uint64

	// The number of confirmation needed before a bloom section is considered probably final and its rotated bits
	// are calculated.
	BloomConfirms uint64

	// The block frequency for creating BloomTrie.
	BloomTrieSize uint64

	// The number of confirmations needed to generate/accept a bloom trie.
	BloomTrieConfirms uint64
}

var (
	// DefaultServerIndexerConfig wraps a set of configs as a default indexer config for server side.
	DefaultServerIndexerConfig = &IndexerConfig{
		ChtSize:           params.CHTFrequency,
		ChtConfirms:       params.HelperTrieProcessConfirmations,
		BloomSize:         params.BloomBitsBlocks,
		BloomConfirms:     params.BloomConfirms,
		BloomTrieSize:     params.BloomTrieFrequency,
		BloomTrieConfirms: params.HelperTrieProcessConfirmations,
	}
	// DefaultClientIndexerConfig wraps a set of configs as a default indexer config for client side.
	DefaultClientIndexerConfig = &IndexerConfig{
		ChtSize:           params.CHTFrequency,
		ChtConfirms:       params.HelperTrieConfirmations,
		BloomSize:         params.BloomBitsBlocksClient,
		BloomConfirms:     params.HelperTrieConfirmations,
		BloomTrieSize:     params.BloomTrieFrequency,
		BloomTrieConfirms: params.HelperTrieConfirmations,
	}
	// TestServerIndexerConfig wraps a set of configs as a test indexer config for server side.
	TestServerIndexerConfig = &IndexerConfig{
		ChtSize:           128,
		ChtConfirms:       1,
		BloomSize:         16,
		BloomConfirms:     1,
		BloomTrieSize:     128,
		BloomTrieConfirms: 1,
	}
	// TestClientIndexerConfig wraps a set of configs as a test indexer config for client side.
	TestClientIndexerConfig = &IndexerConfig{
		ChtSize:           128,
		ChtConfirms:       8,
		BloomSize:         128,
		BloomConfirms:     8,
		BloomTrieSize:     128,
		BloomTrieConfirms: 8,
	}
)

var (
	errNoTrustedCht       = errors.New("no trusted canonical hash trie")
	errNoTrustedBloomTrie = errors.New("no trusted bloom trie")
	errNoHeader           = errors.New("header not found")
	chtPrefix             = []byte("chtRootV2-") // chtPrefix + chtNum (uint64 big endian) -> trie root hash
	ChtTablePrefix        = "cht-"
)

// ChtNode structures are stored in the Canonical Hash Trie in an RLP encoded format
type ChtNode struct {
	Hash common.Hash
	Td   *big.Int
}

// GetChtRoot reads the CHT root associated to the given section from the database
func GetChtRoot(db ethdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := db.Get(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...))
	return common.BytesToHash(data)
}

// StoreChtRoot writes the CHT root associated to the given section into the database
func StoreChtRoot(db ethdb.Database, sectionIdx uint64, sectionHead, root common.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	db.Put(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

// ChtIndexerBackend implements core.ChainIndexerBackend.
type ChtIndexerBackend struct {
	disablePruning       bool
	diskdb, trieTable    ethdb.Database
	odr                  OdrBackend
	triedb               *trie.Database
	trieset              mapset.Set
	section, sectionSize uint64
	lastHash             common.Hash
	trie                 *trie.Trie
}

// NewChtIndexer creates a Cht chain indexer
func NewChtIndexer(db ethdb.Database, odr OdrBackend, size, confirms uint64, disablePruning bool) *core.ChainIndexer {
	trieTable := rawdb.NewTable(db, ChtTablePrefix)
	backend := &ChtIndexerBackend{
		diskdb:         db,
		odr:            odr,
		trieTable:      trieTable,
		triedb:         trie.NewDatabaseWithConfig(trieTable, &trie.Config{Cache: 1}), // Use a tiny cache only to keep memory down
		trieset:        mapset.NewSet(),
		sectionSize:    size,
		disablePruning: disablePruning,
	}
	return core.NewChainIndexer(db, rawdb.NewTable(db, "chtIndexV2-"), backend, size, confirms, time.Millisecond*100, "cht")
}

// fetchMissingNodes tries to retrieve the last entry of the latest trusted CHT from the
// ODR backend in order to be able to add new entries and calculate subsequent root hashes
func (c *ChtIndexerBackend) fetchMissingNodes(ctx context.Context, section uint64, root common.Hash) error {
	batch := c.trieTable.NewBatch()
	r := &ChtRequest{ChtRoot: root, ChtNum: section - 1, BlockNum: section*c.sectionSize - 1, Config: c.odr.IndexerConfig()}
	for {
		err := c.odr.Retrieve(ctx, r)
		switch err {
		case nil:
			r.Proof.Store(batch)
			return batch.Write()
		case ErrNoPeers:
			// if there are no peers to serve, retry later
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second * 10):
				// stay in the loop and try again
			}
		default:
			return err
		}
	}
}

// Reset implements core.ChainIndexerBackend
func (c *ChtIndexerBackend) Reset(ctx context.Context, section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetChtRoot(c.diskdb, section-1, lastSectionHead)
	}
	var err error
	c.trie, err = trie.New(common.Hash{}, root, c.triedb)

	if err != nil && c.odr != nil {
		err = c.fetchMissingNodes(ctx, section, root)
		if err == nil {
			c.trie, err = trie.New(common.Hash{}, root, c.triedb)
		}
	}
	c.section = section
	return err
}

// Process implements core.ChainIndexerBackend
func (c *ChtIndexerBackend) Process(ctx context.Context, header *types.Header) error {
	hash, num := header.Hash(), header.Number.Uint64()
	c.lastHash = hash

	td := rawdb.ReadTd(c.diskdb, hash, num)
	if td == nil {
		panic(nil)
	}
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], num)
	data, _ := rlp.EncodeToBytes(ChtNode{hash, td})
	c.trie.Update(encNumber[:], data)
	return nil
}

// Commit implements core.ChainIndexerBackend
func (c *ChtIndexerBackend) Commit() error {
	root, _, err := c.trie.Commit(nil)
	if err != nil {
		return err
	}
	// Pruning historical trie nodes if necessary.
	if !c.disablePruning {
		// Flush the triedb and track the latest trie nodes.
		c.trieset.Clear()
		c.triedb.Commit(root, false, func(hash common.Hash) { c.trieset.Add(hash) })

		it := c.trieTable.NewIterator(nil, nil)
		defer it.Release()

		var (
			deleted   int
			remaining int
			t         = time.Now()
		)
		for it.Next() {
			trimmed := bytes.TrimPrefix(it.Key(), []byte(ChtTablePrefix))
			if !c.trieset.Contains(common.BytesToHash(trimmed)) {
				c.trieTable.Delete(trimmed)
				deleted += 1
			} else {
				remaining += 1
			}
		}
		log.Debug("Prune historical CHT trie nodes", "deleted", deleted, "remaining", remaining, "elapsed", common.PrettyDuration(time.Since(t)))
	} else {
		c.triedb.Commit(root, false, nil)
	}
	log.Info("Storing CHT", "section", c.section, "head", fmt.Sprintf("%064x", c.lastHash), "root", fmt.Sprintf("%064x", root))
	StoreChtRoot(c.diskdb, c.section, c.lastHash, root)
	return nil
}

// Prune implements core.ChainIndexerBackend which deletes all chain data
// (except hash<->number mappings) older than the specified threshold.
func (c *ChtIndexerBackend) Prune(threshold uint64) error {
	// Short circuit if the light pruning is disabled.
	if c.disablePruning {
		return nil
	}
	t := time.Now()
	// Always keep genesis header in database.
	start, end := uint64(1), (threshold+1)*c.sectionSize

	var batch = c.diskdb.NewBatch()
	for {
		numbers, hashes := rawdb.ReadAllCanonicalHashes(c.diskdb, start, end, 10240)
		if len(numbers) == 0 {
			break
		}
		for i := 0; i < len(numbers); i++ {
			// Keep hash<->number mapping in database otherwise the hash based
			// API(e.g. GetReceipt, GetLogs) will be broken.
			//
			// Storage size wise, the size of a mapping is ~41bytes. For one
			// section is about 1.3MB which is acceptable.
			//
			// In order to totally get rid of this index, we need an additional
			// flag to specify how many historical data light client can serve.
			rawdb.DeleteCanonicalHash(batch, numbers[i])
			rawdb.DeleteBlockWithoutNumber(batch, hashes[i], numbers[i])
		}
		if batch.ValueSize() > ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return err
			}
			batch.Reset()
		}
		start = numbers[len(numbers)-1] + 1
	}
	if err := batch.Write(); err != nil {
		return err
	}
	log.Debug("Prune history headers", "threshold", threshold, "elapsed", common.PrettyDuration(time.Since(t)))
	return nil
}

var (
	bloomTriePrefix      = []byte("bltRoot-") // bloomTriePrefix + bloomTrieNum (uint64 big endian) -> trie root hash
	BloomTrieTablePrefix = "blt-"
)

// GetBloomTrieRoot reads the BloomTrie root assoctiated to the given section from the database
func GetBloomTrieRoot(db ethdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := db.Get(append(append(bloomTriePrefix, encNumber[:]...), sectionHead.Bytes()...))
	return common.BytesToHash(data)
}

// StoreBloomTrieRoot writes the BloomTrie root assoctiated to the given section into the database
func StoreBloomTrieRoot(db ethdb.Database, sectionIdx uint64, sectionHead, root common.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	db.Put(append(append(bloomTriePrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

// BloomTrieIndexerBackend implements core.ChainIndexerBackend
type BloomTrieIndexerBackend struct {
	disablePruning    bool
	diskdb, trieTable ethdb.Database
	triedb            *trie.Database
	trieset           mapset.Set
	odr               OdrBackend
	section           uint64
	parentSize        uint64
	size              uint64
	bloomTrieRatio    uint64
	trie              *trie.Trie
	sectionHeads      []common.Hash
}

// NewBloomTrieIndexer creates a BloomTrie chain indexer
func NewBloomTrieIndexer(db ethdb.Database, odr OdrBackend, parentSize, size uint64, disablePruning bool) *core.ChainIndexer {
	trieTable := rawdb.NewTable(db, BloomTrieTablePrefix)
	backend := &BloomTrieIndexerBackend{
		diskdb:         db,
		odr:            odr,
		trieTable:      trieTable,
		triedb:         trie.NewDatabaseWithConfig(trieTable, &trie.Config{Cache: 1}), // Use a tiny cache only to keep memory down
		trieset:        mapset.NewSet(),
		parentSize:     parentSize,
		size:           size,
		disablePruning: disablePruning,
	}
	backend.bloomTrieRatio = size / parentSize
	backend.sectionHeads = make([]common.Hash, backend.bloomTrieRatio)
	return core.NewChainIndexer(db, rawdb.NewTable(db, "bltIndex-"), backend, size, 0, time.Millisecond*100, "bloomtrie")
}

// fetchMissingNodes tries to retrieve the last entries of the latest trusted bloom trie from the
// ODR backend in order to be able to add new entries and calculate subsequent root hashes
func (b *BloomTrieIndexerBackend) fetchMissingNodes(ctx context.Context, section uint64, root common.Hash) error {
	indexCh := make(chan uint, types.BloomBitLength)
	type res struct {
		nodes *NodeSet
		err   error
	}
	resCh := make(chan res, types.BloomBitLength)
	for i := 0; i < 20; i++ {
		go func() {
			for bitIndex := range indexCh {
				r := &BloomRequest{BloomTrieRoot: root, BloomTrieNum: section - 1, BitIdx: bitIndex, SectionIndexList: []uint64{section - 1}, Config: b.odr.IndexerConfig()}
				for {
					if err := b.odr.Retrieve(ctx, r); err == ErrNoPeers {
						// if there are no peers to serve, retry later
						select {
						case <-ctx.Done():
							resCh <- res{nil, ctx.Err()}
							return
						case <-time.After(time.Second * 10):
							// stay in the loop and try again
						}
					} else {
						resCh <- res{r.Proofs, err}
						break
					}
				}
			}
		}()
	}
	for i := uint(0); i < types.BloomBitLength; i++ {
		indexCh <- i
	}
	close(indexCh)
	batch := b.trieTable.NewBatch()
	for i := uint(0); i < types.BloomBitLength; i++ {
		res := <-resCh
		if res.err != nil {
			return res.err
		}
		res.nodes.Store(batch)
	}
	return batch.Write()
}

// Reset implements core.ChainIndexerBackend
func (b *BloomTrieIndexerBackend) Reset(ctx context.Context, section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetBloomTrieRoot(b.diskdb, section-1, lastSectionHead)
	}
	var err error
	b.trie, err = trie.New(common.Hash{}, root, b.triedb)
	if err != nil && b.odr != nil {
		err = b.fetchMissingNodes(ctx, section, root)
		if err == nil {
			b.trie, err = trie.New(common.Hash{}, root, b.triedb)
		}
	}
	b.section = section
	return err
}

// Process implements core.ChainIndexerBackend
func (b *BloomTrieIndexerBackend) Process(ctx context.Context, header *types.Header) error {
	num := header.Number.Uint64() - b.section*b.size
	if (num+1)%b.parentSize == 0 {
		b.sectionHeads[num/b.parentSize] = header.Hash()
	}
	return nil
}

// Commit implements core.ChainIndexerBackend
func (b *BloomTrieIndexerBackend) Commit() error {
	var compSize, decompSize uint64

	for i := uint(0); i < types.BloomBitLength; i++ {
		var encKey [10]byte
		binary.BigEndian.PutUint16(encKey[0:2], uint16(i))
		binary.BigEndian.PutUint64(encKey[2:10], b.section)
		var decomp []byte
		for j := uint64(0); j < b.bloomTrieRatio; j++ {
			data, err := rawdb.ReadBloomBits(b.diskdb, i, b.section*b.bloomTrieRatio+j, b.sectionHeads[j])
			if err != nil {
				return err
			}
			decompData, err2 := bitutil.DecompressBytes(data, int(b.parentSize/8))
			if err2 != nil {
				return err2
			}
			decomp = append(decomp, decompData...)
		}
		comp := bitutil.CompressBytes(decomp)

		decompSize += uint64(len(decomp))
		compSize += uint64(len(comp))
		if len(comp) > 0 {
			b.trie.Update(encKey[:], comp)
		} else {
			b.trie.Delete(encKey[:])
		}
	}
	root, _, err := b.trie.Commit(nil)
	if err != nil {
		return err
	}
	// Pruning historical trie nodes if necessary.
	if !b.disablePruning {
		// Flush the triedb and track the latest trie nodes.
		b.trieset.Clear()
		b.triedb.Commit(root, false, func(hash common.Hash) { b.trieset.Add(hash) })

		it := b.trieTable.NewIterator(nil, nil)
		defer it.Release()

		var (
			deleted   int
			remaining int
			t         = time.Now()
		)
		for it.Next() {
			trimmed := bytes.TrimPrefix(it.Key(), []byte(BloomTrieTablePrefix))
			if !b.trieset.Contains(common.BytesToHash(trimmed)) {
				b.trieTable.Delete(trimmed)
				deleted += 1
			} else {
				remaining += 1
			}
		}
		log.Debug("Prune historical bloom trie nodes", "deleted", deleted, "remaining", remaining, "elapsed", common.PrettyDuration(time.Since(t)))
	} else {
		b.triedb.Commit(root, false, nil)
	}
	sectionHead := b.sectionHeads[b.bloomTrieRatio-1]
	StoreBloomTrieRoot(b.diskdb, b.section, sectionHead, root)
	log.Info("Storing bloom trie", "section", b.section, "head", fmt.Sprintf("%064x", sectionHead), "root", fmt.Sprintf("%064x", root), "compression", float64(compSize)/float64(decompSize))

	return nil
}

// Prune implements core.ChainIndexerBackend which deletes all
// bloombits which older than the specified threshold.
func (b *BloomTrieIndexerBackend) Prune(threshold uint64) error {
	// Short circuit if the light pruning is disabled.
	if b.disablePruning {
		return nil
	}
	start := time.Now()
	for i := uint(0); i < types.BloomBitLength; i++ {
		rawdb.DeleteBloombits(b.diskdb, i, 0, threshold*b.bloomTrieRatio+b.bloomTrieRatio)
	}
	log.Debug("Prune history bloombits", "threshold", threshold, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}
