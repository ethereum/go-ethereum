// Copyright 2021 The go-ethereum Authors
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

package catalyst

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type syncer struct {
	running   bool
	newBlocks map[common.Hash]*types.Block
	lock      sync.Mutex
}

func newSyncer() *syncer {
	return &syncer{
		newBlocks: make(map[common.Hash]*types.Block),
	}
}

// onNewBlock is the action for receiving new block event
func (s *syncer) onNewBlock(block *types.Block) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.running {
		return
	}
	s.newBlocks[block.Hash()] = block
}

func (s *syncer) hasBlock(hash common.Hash) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, present := s.newBlocks[hash]
	return present
}

// onNewHead is the action for receiving new head event
func (s *syncer) onNewHead(head common.Hash) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.running {
		return
	}
	_, present := s.newBlocks[head]
	if !present {
		log.Error("Chain head is set with an unknown header")
		return
	}
	s.running = true

	// todo call the SetHead function exposed by the downloader
}
