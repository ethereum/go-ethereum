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
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

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

	// A special auxiliary field represents client's chtsize for server config, otherwise represents server's chtsize.
	PairChtSize uint64

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
		ChtSize:           params.CHTFrequencyServer,
		PairChtSize:       params.CHTFrequencyClient,
		ChtConfirms:       params.HelperTrieProcessConfirmations,
		BloomSize:         params.BloomBitsBlocks,
		BloomConfirms:     params.BloomConfirms,
		BloomTrieSize:     params.BloomTrieFrequency,
		BloomTrieConfirms: params.HelperTrieProcessConfirmations,
	}
	// DefaultClientIndexerConfig wraps a set of configs as a default indexer config for client side.
	DefaultClientIndexerConfig = &IndexerConfig{
		ChtSize:           params.CHTFrequencyClient,
		PairChtSize:       params.CHTFrequencyServer,
		ChtConfirms:       params.HelperTrieConfirmations,
		BloomSize:         params.BloomBitsBlocksClient,
		BloomConfirms:     params.HelperTrieConfirmations,
		BloomTrieSize:     params.BloomTrieFrequency,
		BloomTrieConfirms: params.HelperTrieConfirmations,
	}
	// TestServerIndexerConfig wraps a set of configs as a test indexer config for server side.
	TestServerIndexerConfig = &IndexerConfig{
		ChtSize:           256,
		PairChtSize:       2048,
		ChtConfirms:       16,
		BloomSize:         256,
		BloomConfirms:     16,
		BloomTrieSize:     2048,
		BloomTrieConfirms: 16,
	}
	// TestClientIndexerConfig wraps a set of configs as a test indexer config for client side.
	TestClientIndexerConfig = &IndexerConfig{
		ChtSize:           2048,
		PairChtSize:       256,
		ChtConfirms:       128,
		BloomSize:         2048,
		BloomConfirms:     128,
		BloomTrieSize:     2048,
		BloomTrieConfirms: 128,
	}
)

// trustedCheckpoint represents a set of post-processed trie roots (CHT and BloomTrie) associated with
// the appropriate section index and head hash. It is used to start light syncing from this checkpoint
// and avoid downloading the entire header chain while still being able to securely access old headers/logs.
type TrustedCheckpoint struct {
	name                            string
	SectionIdx                      uint64
	SectionHead, CHTRoot, BloomRoot common.Hash
}

// trustedCheckpoints associates each known checkpoint with the genesis hash of the chain it belongs to
var trustedCheckpoints = map[common.Hash]TrustedCheckpoint{
	params.MainnetGenesisHash: {
		name:        "mainnet",
		SectionIdx:  187,
		SectionHead: common.HexToHash("e6baa034efa31562d71ff23676512dec6562c1ad0301e08843b907e81958c696"),
		CHTRoot:     common.HexToHash("28001955219719cf06de1b08648969139d123a9835fc760547a1e4dabdabc15a"),
		BloomRoot:   common.HexToHash("395ca2373fc662720ac6b58b3bbe71f68aa0f38b63b2d3553dd32ff3c51eebc4"),
	},
	params.TestnetGenesisHash: {
		name:        "ropsten",
		SectionIdx:  117,
		SectionHead: common.HexToHash("9529b38631ae30783f56cbe4c3b9f07575b770ecba4f6e20a274b1e2f40fede1"),
		CHTRoot:     common.HexToHash("6f48e9f101f1fac98e7d74fbbcc4fda138358271ffd974d40d2506f0308bb363"),
		BloomRoot:   common.HexToHash("8242342e66e942c0cd893484e6736b9862ceb88b43ca344bb06a8285ac1b6d64"),
	},
	params.RinkebyGenesisHash: {
		name:        "rinkeby",
		SectionIdx:  85,
		SectionHead: common.HexToHash("92cfa67afc4ad8ab0dcbc6fa49efd14b5b19402442e7317e6bc879d85f89d64d"),
		CHTRoot:     common.HexToHash("2802ec92cd7a54a75bca96afdc666ae7b99e5d96cf8192dcfb09588812f51564"),
		BloomRoot:   common.HexToHash("ebefeb31a9a42866d8cf2d2477704b4c3d7c20d0e4e9b5aaa77f396e016a1263"),
	},
}

var (
	ErrNoTrustedCht       = errors.New("no trusted canonical hash trie")
	ErrNoTrustedBloomTrie = errors.New("no trusted bloom trie")
	ErrNoHeader           = errors.New("header not found")
	chtPrefix             = []byte("chtRoot-") // chtPrefix + chtNum (uint64 big endian) -> trie root hash
	ChtTablePrefix        = "cht-"
)

// ChtNode structures are stored in the Canonical Hash Trie in an RLP encoded format
type ChtNode struct {
	Hash common.Hash
	Td   *big.Int
}

// GetChtRoot reads the CHT root associated to the given section from the database
// Note that sectionIdx is specified according to LES/1 CHT section size.
func GetChtRoot(db ethdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := db.Get(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...))
	return common.BytesToHash(data)
}

// StoreChtRoot writes the CHT root associated to the given section into the database
// Note that sectionIdx is specified according to LES/1 CHT section size.
func StoreChtRoot(db ethdb.Database, sectionIdx uint64, sectionHead, root common.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	db.Put(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

// ChtIndexerBackend implements core.ChainIndexerBackend.
type ChtIndexerBackend struct {
	diskdb, trieTable    ethdb.Database
	odr                  OdrBackend
	triedb               *trie.Database
	section, sectionSize uint64
	lastHash             common.Hash
	trie                 *trie.Trie
}

// NewChtIndexer creates a Cht chain indexer
func NewChtIndexer(db ethdb.Database, odr OdrBackend, size, confirms uint64) *core.ChainIndexer {
	trieTable := ethdb.NewTable(db, ChtTablePrefix)
	backend := &ChtIndexerBackend{
		diskdb:      db,
		odr:         odr,
		trieTable:   trieTable,
		triedb:      trie.NewDatabase(trieTable),
		sectionSize: size,
	}
	return core.NewChainIndexer(db, ethdb.NewTable(db, "chtIndex-"), backend, size, confirms, time.Millisecond*100, "cht")
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
	c.trie, err = trie.New(root, c.triedb)

	if err != nil && c.odr != nil {
		err = c.fetchMissingNodes(ctx, section, root)
		if err == nil {
			c.trie, err = trie.New(root, c.triedb)
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
	root, err := c.trie.Commit(nil)
	if err != nil {
		return err
	}
	c.triedb.Commit(root, false)

	if ((c.section+1)*c.sectionSize)%params.CHTFrequencyClient == 0 {
		log.Info("Storing CHT", "section", c.section*c.sectionSize/params.CHTFrequencyClient, "head", fmt.Sprintf("%064x", c.lastHash), "root", fmt.Sprintf("%064x", root))
	}
	StoreChtRoot(c.diskdb, c.section, c.lastHash, root)
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
	diskdb, trieTable ethdb.Database
	triedb            *trie.Database
	odr               OdrBackend
	section           uint64
	parentSize        uint64
	size              uint64
	bloomTrieRatio    uint64
	trie              *trie.Trie
	sectionHeads      []common.Hash
}

// NewBloomTrieIndexer creates a BloomTrie chain indexer
func NewBloomTrieIndexer(db ethdb.Database, odr OdrBackend, parentSize, size uint64) *core.ChainIndexer {
	trieTable := ethdb.NewTable(db, BloomTrieTablePrefix)
	backend := &BloomTrieIndexerBackend{
		diskdb:     db,
		odr:        odr,
		trieTable:  trieTable,
		triedb:     trie.NewDatabase(trieTable),
		parentSize: parentSize,
		size:       size,
	}
	backend.bloomTrieRatio = size / parentSize
	backend.sectionHeads = make([]common.Hash, backend.bloomTrieRatio)
	return core.NewChainIndexer(db, ethdb.NewTable(db, "bltIndex-"), backend, size, 0, time.Millisecond*100, "bloomtrie")
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
				r := &BloomRequest{BloomTrieRoot: root, BloomTrieNum: section - 1, BitIdx: bitIndex, SectionIdxList: []uint64{section - 1}, Config: b.odr.IndexerConfig()}
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
	b.trie, err = trie.New(root, b.triedb)
	if err != nil && b.odr != nil {
		err = b.fetchMissingNodes(ctx, section, root)
		if err == nil {
			b.trie, err = trie.New(root, b.triedb)
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
	root, err := b.trie.Commit(nil)
	if err != nil {
		return err
	}
	b.triedb.Commit(root, false)

	sectionHead := b.sectionHeads[b.bloomTrieRatio-1]
	log.Info("Storing bloom trie", "section", b.section, "head", fmt.Sprintf("%064x", sectionHead), "root", fmt.Sprintf("%064x", root), "compression", float64(compSize)/float64(decompSize))
	StoreBloomTrieRoot(b.diskdb, b.section, sectionHead, root)
	return nil
}
