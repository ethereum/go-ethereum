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

package light

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	// checkpointKey tracks the latest stable checkpoint.
	checkpointKey = []byte("Checkpoint")
)

// TrustedCheckpoint represents a set of post-processed trie roots (CHT and BloomTrie) associated with
// the appropriate section index and head hash.
//
// It is used to start light syncing from this checkpoint and avoid downloading the entire header chain
// while still being able to securely access old headers/logs.
type TrustedCheckpoint struct {
	Name          string      // Indicator which chain the checkpoint belongs to
	SectionIdx    uint64      // Section index
	SectionHead   common.Hash // Block Hash for the last block in the section
	ChtRoot       common.Hash // CHT(Canonical Hash Trie) root associated to the section
	BloomTrieRoot common.Hash // Bloom Trie root associated to the section
}

type trustCheckpointRLP struct {
	SectionIdx    uint64
	SectionHead   common.Hash
	ChtRoot       common.Hash
	BloomTrieRoot common.Hash
}

// EncodeRLP implements rlp.Encoder, and flattens the necessary fields of a checkpoint
// into an RLP stream.
func (c *TrustedCheckpoint) EncodeRLP(w io.Writer) (err error) {
	return rlp.Encode(w, &trustCheckpointRLP{c.SectionIdx, c.SectionHead, c.ChtRoot, c.BloomTrieRoot})
}

// DecodeRLP implements rlp.Decoder, and loads the necessary fields of a checkpoint
// from an RLP stream.
func (c *TrustedCheckpoint) DecodeRLP(s *rlp.Stream) error {
	var dec trustCheckpointRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	c.SectionIdx, c.SectionHead, c.ChtRoot, c.BloomTrieRoot = dec.SectionIdx, dec.SectionHead, dec.ChtRoot, dec.BloomTrieRoot
	return nil
}

var (
	// Hardcode checkpoint for mainnet and testnet(ropsten). Will be deleted eventually once checkpoint contract
	// works.
	mainnetCheckpoint = TrustedCheckpoint{
		Name:          "mainnet",
		SectionIdx:    179,
		SectionHead:   common.HexToHash("ae778e455492db1183e566fa0c67f954d256fdd08618f6d5a393b0e24576d0ea"),
		ChtRoot:       common.HexToHash("646b338f9ca74d936225338916be53710ec84020b89946004a8605f04c817f16"),
		BloomTrieRoot: common.HexToHash("d0f978f5dbc86e5bf931d8dd5b2ecbebbda6dc78f8896af6a27b46a3ced0ac25"),
	}

	ropstenCheckpoint = TrustedCheckpoint{
		Name:          "ropsten",
		SectionIdx:    107,
		SectionHead:   common.HexToHash("e1988f95399debf45b873e065e5cd61b416ef2e2e5deec5a6f87c3127086e1ce"),
		ChtRoot:       common.HexToHash("15cba18e4de0ab1e95e202625199ba30147aec8b0b70384b66ebea31ba6a18e0"),
		BloomTrieRoot: common.HexToHash("e00fa6389b2e597d9df52172cd8e936879eed0fca4fa59db99e2c8ed682562f2"),
	}
)

// TrustedCheckpoints associates each known checkpoint with the genesis hash of the chain it belongs to.
var TrustedCheckpoints = map[common.Hash]TrustedCheckpoint{
	params.MainnetGenesisHash: mainnetCheckpoint,
	params.TestnetGenesisHash: ropstenCheckpoint,
}

// ReadTrustedCheckpoint retrieves the checkpoint from the database.
func ReadTrustedCheckpoint(db ethdb.Database) *TrustedCheckpoint {
	data, err := db.Get(checkpointKey)
	if err != nil {
		return nil
	}
	c := new(TrustedCheckpoint)
	if err := rlp.DecodeBytes(data, c); err != nil {
		log.Error("Invalid checkpoint RLP", "err", err)
		return nil
	}
	return c
}

// WriteTrustedCheckpoint stores an RLP encoded checkpoint into the database.
func WriteTrustedCheckpoint(db ethdb.Putter, checkpoint *TrustedCheckpoint) {
	data, err := rlp.EncodeToBytes(checkpoint)
	if err != nil {
		log.Crit("Failed to RLP encode checkpoint", err)
	}
	if err := db.Put(checkpointKey, data); err != nil {
		log.Crit("Failed to store checkpoint", "err", err)
	}
}
