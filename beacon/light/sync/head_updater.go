// Copyright 2023 The go-ethereum Authors
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

package sync

import (
	"fmt"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
)

type HeadUpdater struct {
	headValidator  *light.HeadValidator
	chain          *light.CommitteeChain
	lock           sync.Mutex
	nextSyncPeriod uint64
	queuedHeads    map[*request.Server][]types.SignedHeader
}

func NewHeadUpdater(headValidator *light.HeadValidator, chain *light.CommitteeChain) *HeadUpdater {
	s := &HeadUpdater{
		headValidator:  headValidator,
		chain:          chain,
		nextSyncPeriod: math.MaxUint64,
		queuedHeads:    make(map[*request.Server][]types.SignedHeader),
	}
	return s
}

// SetupModuleTriggers implements request.Module
func (s *HeadUpdater) SetupModuleTriggers(trigger func(id string, subscribe bool) *request.ModuleTrigger) {
	trigger("newUpdate", true)
}

func (s *HeadUpdater) NewSignedHead(server *request.Server, signedHead types.SignedHeader) {
	nextPeriod, ok := s.chain.NextSyncPeriod()
	if !ok || signedHead.Header.SyncPeriod() > nextPeriod {
		s.lock.Lock()
		s.queuedHeads[server] = append(s.queuedHeads[server], signedHead) //TODO protect against future period spam
		s.lock.Unlock()
		return
	}
	if err := s.headValidator.Add(signedHead); err != nil {
		server.Fail(fmt.Sprintf("Invalid signed head: %v", err))
	}
}

// Process implements request.Module
func (s *HeadUpdater) Process(env *request.Environment) {
	s.lock.Lock()
	defer s.lock.Unlock()

	nextPeriod, ok := s.chain.NextSyncPeriod()
	if !ok || nextPeriod == s.nextSyncPeriod {
		return
	}
	s.nextSyncPeriod = nextPeriod

	for server, queued := range s.queuedHeads {
		j := len(queued)
		for i := len(queued) - 1; i >= 0; i-- {
			if signedHead := queued[i]; signedHead.Header.SyncPeriod() <= nextPeriod {
				if err := s.headValidator.Add(signedHead); err != nil {
					server.Fail(fmt.Sprintf("Invalid queued head: %v", err))
				}
			} else {
				j--
				if j != i {
					queued[j] = queued[i]
				}
			}
		}
		if j != 0 {
			s.queuedHeads[server] = queued[j:]
		}
	}
}
