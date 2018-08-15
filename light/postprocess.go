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

const (
	// CHTFrequencyClient is the block frequency for creating CHTs on the client side.
	CHTFrequencyClient = 32768

	// CHTFrequencyServer is the block frequency for creating CHTs on the server side.
	// Eventually this can be merged back with the client version, but that requires a
	// full database upgrade, so that should be left for a suitable moment.
	CHTFrequencyServer = 4096

	HelperTrieConfirmations        = 2048 // number of confirmations before a server is expected to have the given HelperTrie available
	HelperTrieProcessConfirmations = 256  // number of confirmations before a HelperTrie is generated
)

// TrustedCheckpoint represents a set of post-processed trie roots (CHT and BloomTrie) associated with
// the appropriate section index and head hash. It is used to start light syncing from this checkpoint
// and avoid downloading the entire header chain while still being able to securely access old headers/logs.
type TrustedCheckpoint struct {
	name                            string
	SectionIdx                      uint64
	SectionHead, CHTRoot, BloomRoot common.Hash
}

var (
	mainnetCheckpoint = TrustedCheckpoint{
		name:        "mainnet",
		SectionIdx:  179,
		SectionHead: common.HexToHash("ae778e455492db1183e566fa0c67f954d256fdd08618f6d5a393b0e24576d0ea"),
		CHTRoot:     common.HexToHash("646b338f9ca74d936225338916be53710ec84020b89946004a8605f04c817f16"),
		BloomRoot:   common.HexToHash("d0f978f5dbc86e5bf931d8dd5b2ecbebbda6dc78f8896af6a27b46a3ced0ac25"),
	}

	ropstenCheckpoint = TrustedCheckpoint{
		name:        "ropsten",
		SectionIdx:  107,
		SectionHead: common.HexToHash("e1988f95399debf45b873e065e5cd61b416ef2e2e5deec5a6f87c3127086e1ce"),
		CHTRoot:     common.HexToHash("15cba18e4de0ab1e95e202625199ba30147aec8b0b70384b66ebea31ba6a18e0"),
		BloomRoot:   common.HexToHash("e00fa6389b2e597d9df52172cd8e936879eed0fca4fa59db99e2c8ed682562f2"),
	}
)

// trustedCheckpoints associates each known checkpoint with the genesis hash of the chain it belongs to
var trustedCheckpoints = map[common.Hash]TrustedCheckpoint{
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

// GetChtRoot reads the CHT root assoctiated to the given section from the database
// Note that sectionIdx is specified according to LES/1 CHT section size
func GetChtRoot(db ethdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := db.Get(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...))
	return common.BytesToHash(data)
}

// GetChtV2Root reads the CHT root assoctiated to the given section from the database
// Note that sectionIdx is specified according to LES/2 CHT section size
func GetChtV2Root(db ethdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	return GetChtRoot(db, (sectionIdx+1)*(CHTFrequencyClient/CHTFrequencyServer)-1, sectionHead)
}

// StoreChtRoot writes the CHT root assoctiated to the given section into the database
// Note that sectionIdx is specified according to LES/1 CHT section size
func StoreChtRoot(db ethdb.Database, sectionIdx uint64, sectionHead, root common.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	db.Put(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

// ChtIndexerBackend implements core.ChainIndexerBackend
type ChtIndexerBackend struct {
	diskdb, trieTable    ethdb.Database
	odr                  OdrBackend
	triedb               *trie.Database
	section, sectionSize uint64
	lastHash             common.Hash
	trie                 *trie.Trie
}

// NewBloomTrieIndexer creates a BloomTrie chain indexer
func NewChtIndexer(db ethdb.Database, clientMode bool, odr OdrBackend) *core.ChainIndexer {
	var sectionSize, confirmReq uint64
	if clientMode {
		sectionSize = CHTFrequencyClient
		confirmReq = HelperTrieConfirmations
	} else {
		sectionSize = CHTFrequencyServer
		confirmReq = HelperTrieProcessConfirmations
	}
	idb := ethdb.NewTable(db, "chtIndex-")
	trieTable := ethdb.NewTable(db, ChtTablePrefix)
	backend := &ChtIndexerBackend{
		diskdb:      db,
		odr:         odr,
		trieTable:   trieTable,
		triedb:      trie.NewDatabase(trieTable),
		sectionSize: sectionSize,
	}
	return core.NewChainIndexer(db, idb, backend, sectionSize, confirmReq, time.Millisecond*100, "cht")
}

// fetchMissingNodes tries to retrieve the last entry of the latest trusted CHT from the
// ODR backend in order to be able to add new entries and calculate subsequent root hashes
func (c *ChtIndexerBackend) fetchMissingNodes(ctx context.Context, section uint64, root common.Hash) error {
	batch := c.trieTable.NewBatch()
	r := &ChtRequest{ChtRoot: root, ChtNum: section - 1, BlockNum: section*c.sectionSize - 1}
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

	if ((c.section+1)*c.sectionSize)%CHTFrequencyClient == 0 {
		log.Info("Storing CHT", "section", c.section*c.sectionSize/CHTFrequencyClient, "head", fmt.Sprintf("%064x", c.lastHash), "root", fmt.Sprintf("%064x", root))
	}
	StoreChtRoot(c.diskdb, c.section, c.lastHash, root)
	return nil
}

const (
	BloomTrieFrequency  = 32768
	ethBloomBitsSection = 4096
)

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
	diskdb, trieTable                          ethdb.Database
	odr                                        OdrBackend
	triedb                                     *trie.Database
	section, parentSectionSize, bloomTrieRatio uint64
	trie                                       *trie.Trie
	sectionHeads                               []common.Hash
}

// NewBloomTrieIndexer creates a BloomTrie chain indexer
func NewBloomTrieIndexer(db ethdb.Database, clientMode bool, odr OdrBackend) *core.ChainIndexer {
	trieTable := ethdb.NewTable(db, BloomTrieTablePrefix)
	backend := &BloomTrieIndexerBackend{
		diskdb:    db,
		odr:       odr,
		trieTable: trieTable,
		triedb:    trie.NewDatabase(trieTable),
	}
	idb := ethdb.NewTable(db, "bltIndex-")

	if clientMode {
		backend.parentSectionSize = BloomTrieFrequency
	} else {
		backend.parentSectionSize = ethBloomBitsSection
	}
	backend.bloomTrieRatio = BloomTrieFrequency / backend.parentSectionSize
	backend.sectionHeads = make([]common.Hash, backend.bloomTrieRatio)
	return core.NewChainIndexer(db, idb, backend, BloomTrieFrequency, 0, time.Millisecond*100, "bloomtrie")
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
				r := &BloomRequest{BloomTrieRoot: root, BloomTrieNum: section - 1, BitIdx: bitIndex, SectionIdxList: []uint64{section - 1}}
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
	num := header.Number.Uint64() - b.section*BloomTrieFrequency
	if (num+1)%b.parentSectionSize == 0 {
		b.sectionHeads[num/b.parentSectionSize] = header.Hash()
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
			decompData, err2 := bitutil.DecompressBytes(data, int(b.parentSectionSize/8))
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
