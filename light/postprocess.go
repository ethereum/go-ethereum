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
	"encoding/binary"
	"errors"
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
	ChtConfirm uint64

	// The block frequency for creating new bloom bits.
	BloomSize uint64

	// The number of confirmation needed before a bloom section is considered probably final and its rotated bits
	// are calculated.
	BloomConfirm uint64

	// The block frequency for creating BloomTrie.
	BloomTrieSize uint64

	// The number of confirmations needed to generate/accept a bloom trie.
	BloomTrieConfirm uint64
}

var (
	// DefaultServerIndexerConfig wraps a set of configs as a default indexer config for server side.
	DefaultServerIndexerConfig = &IndexerConfig{
		ChtSize:          params.CHTFrequencyServer,
		PairChtSize:      params.CHTFrequencyClient,
		ChtConfirm:       params.HelperTrieProcessConfirmations,
		BloomSize:        params.BloomBitsBlocks,
		BloomConfirm:     params.BloomConfirms,
		BloomTrieSize:    params.BloomTrieFrequency,
		BloomTrieConfirm: params.HelperTrieProcessConfirmations,
	}
	// DefaultClientIndexerConfig wraps a set of configs as a default indexer config for client side.
	DefaultClientIndexerConfig = &IndexerConfig{
		ChtSize:          params.CHTFrequencyClient,
		PairChtSize:      params.CHTFrequencyServer,
		ChtConfirm:       params.HelperTrieConfirmations,
		BloomSize:        params.BloomBitsBlocksClient,
		BloomConfirm:     params.HelperTrieConfirmations,
		BloomTrieSize:    params.BloomTrieFrequency,
		BloomTrieConfirm: params.HelperTrieConfirmations,
	}
	// TestServerIndexerConfig wraps a set of configs as a test indexer config for server side.
	TestServerIndexerConfig = &IndexerConfig{
		ChtSize:          256,
		PairChtSize:      2048,
		ChtConfirm:       16,
		BloomSize:        256,
		BloomConfirm:     16,
		BloomTrieSize:    2048,
		BloomTrieConfirm: 16,
	}
	// TestClientIndexerConfig wraps a set of configs as a test indexer config for client side.
	TestClientIndexerConfig = &IndexerConfig{
		ChtSize:          2048,
		PairChtSize:      256,
		ChtConfirm:       128,
		BloomSize:        2048,
		BloomConfirm:     128,
		BloomTrieSize:    2048,
		BloomTrieConfirm: 128,
	}
)

// trustedCheckpoint represents a set of post-processed trie roots (CHT and BloomTrie) associated with
// the appropriate section index and head hash. It is used to start light syncing from this checkpoint
// and avoid downloading the entire header chain while still being able to securely access old headers/logs.
type trustedCheckpoint struct {
	name                                string
	sectionIdx                          uint64
	sectionHead, chtRoot, bloomTrieRoot common.Hash
}

var (
	mainnetCheckpoint = trustedCheckpoint{
		name:          "mainnet",
		sectionIdx:    179,
		sectionHead:   common.HexToHash("ae778e455492db1183e566fa0c67f954d256fdd08618f6d5a393b0e24576d0ea"),
		chtRoot:       common.HexToHash("646b338f9ca74d936225338916be53710ec84020b89946004a8605f04c817f16"),
		bloomTrieRoot: common.HexToHash("d0f978f5dbc86e5bf931d8dd5b2ecbebbda6dc78f8896af6a27b46a3ced0ac25"),
	}

	ropstenCheckpoint = trustedCheckpoint{
		name:          "ropsten",
		sectionIdx:    107,
		sectionHead:   common.HexToHash("e1988f95399debf45b873e065e5cd61b416ef2e2e5deec5a6f87c3127086e1ce"),
		chtRoot:       common.HexToHash("15cba18e4de0ab1e95e202625199ba30147aec8b0b70384b66ebea31ba6a18e0"),
		bloomTrieRoot: common.HexToHash("e00fa6389b2e597d9df52172cd8e936879eed0fca4fa59db99e2c8ed682562f2"),
	}
)

// trustedCheckpoints associates each known checkpoint with the genesis hash of the chain it belongs to
var trustedCheckpoints = map[common.Hash]trustedCheckpoint{
	params.MainnetGenesisHash: mainnetCheckpoint,
	params.TestnetGenesisHash: ropstenCheckpoint,
}

var (
	ErrNoTrustedCht       = errors.New("No trusted canonical hash trie")
	ErrNoTrustedBloomTrie = errors.New("No trusted bloom trie")
	ErrNoHeader           = errors.New("Header not found")
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
	diskdb      ethdb.Database
	triedb      *trie.Database
	section     uint64
	sectionSize uint64
	lastHash    common.Hash
	trie        *trie.Trie
}

// NewChtIndexer creates a Cht chain indexer.
func NewChtIndexer(db ethdb.Database, size, confirms uint64) *core.ChainIndexer {
	idb := ethdb.NewTable(db, "chtIndex-")
	backend := &ChtIndexerBackend{
		diskdb:      db,
		triedb:      trie.NewDatabase(ethdb.NewTable(db, ChtTablePrefix)),
		sectionSize: size,
	}
	return core.NewChainIndexer(db, idb, backend, size, confirms, time.Millisecond*100, "cht")
}

// Reset implements core.ChainIndexerBackend.
func (c *ChtIndexerBackend) Reset(section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetChtRoot(c.diskdb, section-1, lastSectionHead)
	}
	var err error
	c.trie, err = trie.New(root, c.triedb)
	c.section = section
	return err
}

// Process implements core.ChainIndexerBackend
func (c *ChtIndexerBackend) Process(header *types.Header) {
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
}

// Commit implements core.ChainIndexerBackend
func (c *ChtIndexerBackend) Commit() error {
	root, err := c.trie.Commit(nil)
	if err != nil {
		return err
	}
	c.triedb.Commit(root, false)

	if ((c.section+1)*c.sectionSize)%params.CHTFrequencyClient == 0 {
		log.Info("Storing CHT", "section", c.section*c.sectionSize/params.CHTFrequencyClient, "head", c.lastHash, "root", root)
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
	diskdb         ethdb.Database
	triedb         *trie.Database
	section        uint64
	parentSize     uint64
	size           uint64
	bloomTrieRatio uint64
	trie           *trie.Trie
	sectionHeads   []common.Hash
}

// NewBloomTrieIndexer creates a BloomTrie chain indexer
func NewBloomTrieIndexer(db ethdb.Database, parentSize, parentConfirms, size, confirms uint64) *core.ChainIndexer {
	backend := &BloomTrieIndexerBackend{
		diskdb:     db,
		triedb:     trie.NewDatabase(ethdb.NewTable(db, BloomTrieTablePrefix)),
		parentSize: parentSize,
		size:       size,
	}
	idb := ethdb.NewTable(db, "bltIndex-")

	backend.bloomTrieRatio = size / backend.parentSize
	backend.sectionHeads = make([]common.Hash, backend.bloomTrieRatio)
	return core.NewChainIndexer(db, idb, backend, size, confirms-parentConfirms, time.Millisecond*100, "bloomtrie")
}

// Reset implements core.ChainIndexerBackend
func (b *BloomTrieIndexerBackend) Reset(section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetBloomTrieRoot(b.diskdb, section-1, lastSectionHead)
	}
	var err error
	b.trie, err = trie.New(root, b.triedb)
	b.section = section
	return err
}

// Process implements core.ChainIndexerBackend
func (b *BloomTrieIndexerBackend) Process(header *types.Header) {
	num := header.Number.Uint64() - b.section*b.size
	if (num+1)%b.parentSize == 0 {
		b.sectionHeads[num/b.parentSize] = header.Hash()
	}
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
	log.Info("Storing bloom trie", "section", b.section, "head", sectionHead, "root", root, "compression", float64(compSize)/float64(decompSize))
	StoreBloomTrieRoot(b.diskdb, b.section, sectionHead, root)

	return nil
}
