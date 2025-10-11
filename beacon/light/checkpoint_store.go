// Copyright 2025 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var checkpointKey = []byte("checkpoint-") // block root -> RLP(types.BootstrapData)

// CheckpointStore stores checkpoints in a database, identified by their hash.
type CheckpointStore struct {
	chain *CommitteeChain
	db    ethdb.KeyValueStore
}

func NewCheckpointStore(db ethdb.KeyValueStore, chain *CommitteeChain) *CheckpointStore {
	return &CheckpointStore{
		db:    db,
		chain: chain,
	}
}

func getCheckpointKey(checkpoint common.Hash) []byte {
	var (
		kl  = len(checkpointKey)
		key = make([]byte, kl+32)
	)
	copy(key[:kl], checkpointKey)
	copy(key[kl:], checkpoint[:])
	return key
}

func (cs *CheckpointStore) Get(checkpoint common.Hash) *types.BootstrapData {
	if enc, err := cs.db.Get(getCheckpointKey(checkpoint)); err == nil {
		c := new(types.BootstrapData)
		if err := rlp.DecodeBytes(enc, c); err != nil {
			log.Error("Error decoding stored checkpoint", "error", err)
			return nil
		}
		if committee := cs.chain.GetCommittee(c.Header.SyncPeriod()); committee != nil && committee.Root() == c.CommitteeRoot {
			c.Committee = committee
			return c
		}
		log.Error("Missing committee for stored checkpoint", "period", c.Header.SyncPeriod())
	}
	return nil
}

func (cs *CheckpointStore) Store(c *types.BootstrapData) {
	enc, err := rlp.EncodeToBytes(c)
	if err != nil {
		log.Error("Error encoding checkpoint for storage", "error", err)
	}
	if err := cs.db.Put(getCheckpointKey(c.Header.Hash()), enc); err != nil {
		log.Error("Error storing checkpoint in database", "error", err)
	}
}
