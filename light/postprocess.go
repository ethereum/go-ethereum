// Copyright 2016 The go-ethereum Authors
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
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	ChtFrequency                   = 32768
	ChtV1Frequency                 = 4096 // as long as we want to retain LES/1 compatibility, servers generate CHTs with the old, higher frequency
	HelperTrieConfirmations        = 2048 // number of confirmations before a server is expected to have the given HelperTrie available
	HelperTrieProcessConfirmations = 256  // number of confirmations before a HelperTrie is generated
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
		name:          "ETH mainnet",
		sectionIdx:    150,
		sectionHead:   common.HexToHash("1e2e67f289565cbe7bd4367f7960dbd73a3f7c53439e1047cd7ba331c8109e39"),
		chtRoot:       common.HexToHash("f2a6c9ca143d647b44523cc249f1072c8912358ab873a77a5fdc792b8df99e80"),
		bloomTrieRoot: common.HexToHash("c018952fa1513c97857e79fbb9a37acaf8432d5b85e52a78eca7dff5fd5900ee"),
	}

	ropstenCheckpoint = trustedCheckpoint{
		name:          "Ropsten testnet",
		sectionIdx:    75,
		sectionHead:   common.HexToHash("12e68324f4578ea3e8e7fb3968167686729396c9279287fa1f1a8b51bb2d05b4"),
		chtRoot:       common.HexToHash("3e51dc095c69fa654a4cac766e0afff7357515b4b3c3a379c675f810363e54be"),
		bloomTrieRoot: common.HexToHash("33e3a70b33c1d73aa698d496a80615e98ed31fa8f56969876180553b32333339"),
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
	return GetChtRoot(db, (sectionIdx+1)*(ChtFrequency/ChtV1Frequency)-1, sectionHead)
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
	db, cdb              ethdb.Database
	section, sectionSize uint64
	lastHash             common.Hash
	trie                 *trie.Trie
}

// NewBloomTrieIndexer creates a BloomTrie chain indexer
func NewChtIndexer(db ethdb.Database, clientMode bool) *core.ChainIndexer {
	cdb := ethdb.NewTable(db, ChtTablePrefix)
	idb := ethdb.NewTable(db, "chtIndex-")
	var sectionSize, confirmReq uint64
	if clientMode {
		sectionSize = ChtFrequency
		confirmReq = HelperTrieConfirmations
	} else {
		sectionSize = ChtV1Frequency
		confirmReq = HelperTrieProcessConfirmations
	}
	return core.NewChainIndexer(db, idb, &ChtIndexerBackend{db: db, cdb: cdb, sectionSize: sectionSize}, sectionSize, confirmReq, time.Millisecond*100, "cht")
}

// Reset implements core.ChainIndexerBackend
func (c *ChtIndexerBackend) Reset(section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetChtRoot(c.db, section-1, lastSectionHead)
	}
	var err error
	c.trie, err = trie.New(root, c.cdb)
	c.section = section
	return err
}

// Process implements core.ChainIndexerBackend
func (c *ChtIndexerBackend) Process(header *types.Header) {
	hash, num := header.Hash(), header.Number.Uint64()
	c.lastHash = hash

	td := core.GetTd(c.db, hash, num)
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
	batch := c.cdb.NewBatch()
	root, err := c.trie.CommitTo(batch)
	if err != nil {
		return err
	} else {
		batch.Write()
		if ((c.section+1)*c.sectionSize)%ChtFrequency == 0 {
			log.Info("Storing CHT", "idx", c.section*c.sectionSize/ChtFrequency, "sectionHead", fmt.Sprintf("%064x", c.lastHash), "root", fmt.Sprintf("%064x", root))
		}
		StoreChtRoot(c.db, c.section, c.lastHash, root)
	}
	return nil
}

const (
	BloomTrieFrequency        = 32768
	ethBloomBitsSection       = 4096
	ethBloomBitsConfirmations = 256
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
	db, cdb                                    ethdb.Database
	section, parentSectionSize, bloomTrieRatio uint64
	trie                                       *trie.Trie
	sectionHeads                               []common.Hash
}

// NewBloomTrieIndexer creates a BloomTrie chain indexer
func NewBloomTrieIndexer(db ethdb.Database, clientMode bool) *core.ChainIndexer {
	cdb := ethdb.NewTable(db, BloomTrieTablePrefix)
	idb := ethdb.NewTable(db, "bltIndex-")
	backend := &BloomTrieIndexerBackend{db: db, cdb: cdb}
	var confirmReq uint64
	if clientMode {
		backend.parentSectionSize = BloomTrieFrequency
		confirmReq = HelperTrieConfirmations
	} else {
		backend.parentSectionSize = ethBloomBitsSection
		confirmReq = HelperTrieProcessConfirmations
	}
	backend.bloomTrieRatio = BloomTrieFrequency / backend.parentSectionSize
	backend.sectionHeads = make([]common.Hash, backend.bloomTrieRatio)
	return core.NewChainIndexer(db, idb, backend, BloomTrieFrequency, confirmReq-ethBloomBitsConfirmations, time.Millisecond*100, "bloomtrie")
}

// Reset implements core.ChainIndexerBackend
func (b *BloomTrieIndexerBackend) Reset(section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetBloomTrieRoot(b.db, section-1, lastSectionHead)
	}
	var err error
	b.trie, err = trie.New(root, b.cdb)
	b.section = section
	return err
}

// Process implements core.ChainIndexerBackend
func (b *BloomTrieIndexerBackend) Process(header *types.Header) {
	num := header.Number.Uint64() - b.section*BloomTrieFrequency
	if (num+1)%b.parentSectionSize == 0 {
		b.sectionHeads[num/b.parentSectionSize] = header.Hash()
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
			data, err := core.GetBloomBits(b.db, i, b.section*b.bloomTrieRatio+j, b.sectionHeads[j])
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

	batch := b.cdb.NewBatch()
	root, err := b.trie.CommitTo(batch)
	if err != nil {
		return err
	} else {
		batch.Write()
		sectionHead := b.sectionHeads[b.bloomTrieRatio-1]
		log.Info("Storing BloomTrie", "section", b.section, "sectionHead", fmt.Sprintf("%064x", sectionHead), "root", fmt.Sprintf("%064x", root), "compression ratio", float64(compSize)/float64(decompSize))
		StoreBloomTrieRoot(b.db, b.section, sectionHead, root)
	}

	return nil
}
