// Copyright 2015 The go-ethereum Authors
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

package discv5

import (
	"encoding/binary"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
)

const powSize = 8

type pow interface {
	valid(packetHash common.Hash) bool
}

type simplePoW struct {
	compare uint64
}

func newSimplePoW(difficulty float64) *simplePoW {
	compare := ^uint64(0)
	if difficulty > 1 {
		compare = uint64(float64(compare) / difficulty)
	}
	return &simplePoW{compare}
}

func (s *simplePoW) valid(packetHash common.Hash) bool {
	return binary.BigEndian.Uint64(packetHash[0:powSize]) <= s.compare
}

func findPoW(targetHash common.Hash, packet []byte, pow pow, maxCount int) bool {
	hashBytes := targetHash.Bytes()
	data := append(hashBytes, packet...)
	nonceBytes := data[len(hashBytes) : len(hashBytes)+powSize]
	rand.Read(nonceBytes)
	nonce := binary.BigEndian.Uint64(nonceBytes)
	for i := 0; i < maxCount; i++ {
		packetHash := crypto.Keccak256Hash(data)
		if pow.valid(packetHash) {
			binary.BigEndian.PutUint64(packet[:powSize], nonce)
			return true
		}
		nonce++
		binary.BigEndian.PutUint64(nonceBytes, nonce)
	}
	return false
}

// powRequest represents a PoW to be calculated for an outgoing message
// PoWs are processed by powProcessor and are selected by common.WeightedRandomSelect
// for processing (powRequest implements wrsItem).
type powRequest struct {
	targetHash common.Hash
	packet     []byte
	pow        pow
	weight     int64
	done       chan bool
	// these fields are set by the processor
	timeout mclock.AbsTime
	next    *powRequest
}

func (p *powRequest) Weight() int64 {
	return p.weight
}

const (
	powQueueTimeout = time.Second * 10
	powTryCount     = 1000000
	powCpuRatio     = 0.1
)

// powProcessor starts a global processing loop for PoWs that ensures that only a
// certain percentage of a single CPU's time is assigned for PoW search globally
func powProcessor() chan *powRequest {
	wrs := common.NewWeightedRandomSelect()
	powCh := make(chan *powRequest, 100)
	go func() {
		var (
			first, last              *powRequest
			removeFirst, processNext <-chan time.Time
		)

		for {
			select {
			case pr, ok := <-powCh:
				if !ok {
					return
				}
				wrs.Update(pr)
				pr.timeout = mclock.Now() + mclock.AbsTime(powQueueTimeout)
				if first == nil {
					first = pr
					removeFirst = time.After(powQueueTimeout)
				}
				if last != nil {
					last.next = pr
				}
				last = pr
				if processNext == nil {
					processNext = time.After(0)
				}
			case <-removeFirst:
				wrs.Remove(first)
				select {
				case first.done <- false:
				default:
				}
				first = first.next
				if first != nil {
					removeFirst = time.After(time.Duration(first.timeout - mclock.Now()))
				}
			case <-processNext:
				p := wrs.Choose()
				if p != nil {
					pr := p.(*powRequest)
					start := mclock.Now()
					if findPoW(pr.targetHash, pr.packet, pr.pow, powTryCount) {
						wrs.Remove(pr)
						select {
						case pr.done <- true:
						default:
						}
					}
					d := time.Duration(mclock.Now() - start)
					processNext = time.After(d * (1/powCpuRatio - 1))
				}
			}
		}
	}()
	return powCh
}

// hashReplayFilter rejects replayed packets by packet hash, remembering only the
// recent received packet hashes. Intro packets are filtered by hash after checking
// their PoW.
// Note: general packets are also filtered by hash first even though they are later
// filtered by the node specific serial filter too in order to avoid decryption costs
// in case of packet resending. This is realized with a separate instance of
// hashReplayFilter so that processed intro packets are remembered for as long as possible.
type hashReplayFilter struct {
	indexToHash            map[uint64]common.Hash
	hashToIndex            map[common.Hash]uint64
	nextIndex, deleteIndex uint64
}

const hashReplayFilterSize = 10000

func newHashReplayFilter() *hashReplayFilter {
	return &hashReplayFilter{
		indexToHash: make(map[uint64]common.Hash),
		hashToIndex: make(map[common.Hash]uint64),
	}
}

func (f *hashReplayFilter) accept(hash common.Hash) bool {
	if oldIndex, ok := f.hashToIndex[hash]; ok {
		// pow already known, move to the front of the queue and reject
		if f.nextIndex != oldIndex {
			f.hashToIndex[hash] = f.nextIndex
			delete(f.indexToHash, oldIndex)
			f.indexToHash[f.nextIndex] = hash
			f.nextIndex++
		}
		return false
	}
	// pow not seen recently, add to the front of the queue and accept
	f.hashToIndex[hash] = f.nextIndex
	f.indexToHash[f.nextIndex] = hash
	f.nextIndex++
	// delete least recently received hash if entry count has reached the limit
	if len(f.indexToHash) > hashReplayFilterSize {
		for {
			if hash, ok := f.indexToHash[f.deleteIndex]; ok {
				delete(f.indexToHash, f.deleteIndex)
				delete(f.hashToIndex, hash)
				f.deleteIndex++
				break
			}
			f.deleteIndex++
			if f.deleteIndex >= f.nextIndex {
				panic(nil)
			}
		}
	}
	return true
}
