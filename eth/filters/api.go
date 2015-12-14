// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package filters

import (
	"sync"
	"time"

	"crypto/rand"
	"encoding/hex"
	"errors"

	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
)

var (
	filterTickerTime = 5 * time.Minute
)

// byte will be inferred
const (
	unknownFilterTy = iota
	blockFilterTy
	transactionFilterTy
	logFilterTy
)

// PublicFilterAPI offers support to create and manage filters. This will allow externa clients to retrieve various
// information related to the Ethereum protocol such als blocks, transactions and logs.
type PublicFilterAPI struct {
	mux *event.TypeMux

	quit    chan struct{}
	chainDb ethdb.Database

	filterManager *FilterSystem

	filterMapMu   sync.RWMutex
	filterMapping map[string]int // maps between filter internal filter identifiers and external filter identifiers

	logMu    sync.RWMutex
	logQueue map[int]*logQueue

	blockMu    sync.RWMutex
	blockQueue map[int]*hashQueue

	transactionMu    sync.RWMutex
	transactionQueue map[int]*hashQueue

	transactMu sync.Mutex
}

// NewPublicFilterAPI returns a new PublicFilterAPI instance.
func NewPublicFilterAPI(chainDb ethdb.Database, mux *event.TypeMux) *PublicFilterAPI {
	svc := &PublicFilterAPI{
		mux:              mux,
		chainDb:          chainDb,
		filterManager:    NewFilterSystem(mux),
		filterMapping:    make(map[string]int),
		logQueue:         make(map[int]*logQueue),
		blockQueue:       make(map[int]*hashQueue),
		transactionQueue: make(map[int]*hashQueue),
	}
	go svc.start()
	return svc
}

// Stop quits the work loop.
func (s *PublicFilterAPI) Stop() {
	close(s.quit)
}

// start the work loop, wait and process events.
func (s *PublicFilterAPI) start() {
	timer := time.NewTicker(2 * time.Second)
	defer timer.Stop()
done:
	for {
		select {
		case <-timer.C:
			s.logMu.Lock()
			for id, filter := range s.logQueue {
				if time.Since(filter.timeout) > filterTickerTime {
					s.filterManager.Remove(id)
					delete(s.logQueue, id)
				}
			}
			s.logMu.Unlock()

			s.blockMu.Lock()
			for id, filter := range s.blockQueue {
				if time.Since(filter.timeout) > filterTickerTime {
					s.filterManager.Remove(id)
					delete(s.blockQueue, id)
				}
			}
			s.blockMu.Unlock()

			s.transactionMu.Lock()
			for id, filter := range s.transactionQueue {
				if time.Since(filter.timeout) > filterTickerTime {
					s.filterManager.Remove(id)
					delete(s.transactionQueue, id)
				}
			}
			s.transactionMu.Unlock()
		case <-s.quit:
			break done
		}
	}

}

// NewBlockFilter create a new filter that returns blocks that are included into the canonical chain.
func (s *PublicFilterAPI) NewBlockFilter() (string, error) {
	externalId, err := newFilterId()
	if err != nil {
		return "", err
	}

	s.blockMu.Lock()
	filter := New(s.chainDb)
	id := s.filterManager.Add(filter)
	s.blockQueue[id] = &hashQueue{timeout: time.Now()}

	filter.BlockCallback = func(block *types.Block, logs vm.Logs) {
		s.blockMu.Lock()
		defer s.blockMu.Unlock()

		if queue := s.blockQueue[id]; queue != nil {
			queue.add(block.Hash())
		}
	}

	defer s.blockMu.Unlock()

	s.filterMapMu.Lock()
	s.filterMapping[externalId] = id
	s.filterMapMu.Unlock()

	return externalId, nil
}

// NewPendingTransactionFilter creates a filter that returns new pending transactions.
func (s *PublicFilterAPI) NewPendingTransactionFilter() (string, error) {
	externalId, err := newFilterId()
	if err != nil {
		return "", err
	}

	s.transactionMu.Lock()
	defer s.transactionMu.Unlock()

	filter := New(s.chainDb)
	id := s.filterManager.Add(filter)
	s.transactionQueue[id] = &hashQueue{timeout: time.Now()}

	filter.TransactionCallback = func(tx *types.Transaction) {
		s.transactionMu.Lock()
		defer s.transactionMu.Unlock()

		if queue := s.transactionQueue[id]; queue != nil {
			queue.add(tx.Hash())
		}
	}

	s.filterMapMu.Lock()
	s.filterMapping[externalId] = id
	s.filterMapMu.Unlock()

	return externalId, nil
}

// newLogFilter creates a new log filter.
func (s *PublicFilterAPI) newLogFilter(earliest, latest int64, addresses []common.Address, topics [][]common.Hash) int {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	filter := New(s.chainDb)
	id := s.filterManager.Add(filter)
	s.logQueue[id] = &logQueue{timeout: time.Now()}

	filter.SetBeginBlock(earliest)
	filter.SetEndBlock(latest)
	filter.SetAddresses(addresses)
	filter.SetTopics(topics)
	filter.LogsCallback = func(logs vm.Logs) {
		s.logMu.Lock()
		defer s.logMu.Unlock()

		if queue := s.logQueue[id]; queue != nil {
			queue.add(logs...)
		}
	}

	return id
}

// NewFilterArgs represents a request to create a new filter.
type NewFilterArgs struct {
	FromBlock rpc.BlockNumber
	ToBlock   rpc.BlockNumber
	Addresses []common.Address
	Topics    [][]common.Hash
}

func (args *NewFilterArgs) UnmarshalJSON(data []byte) error {
	type input struct {
		From      *rpc.BlockNumber `json:"fromBlock"`
		ToBlock   *rpc.BlockNumber `json:"toBlock"`
		Addresses interface{}      `json:"address"`
		Topics    interface{}      `json:"topics"`
	}

	var raw input
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.From == nil {
		args.FromBlock = rpc.LatestBlockNumber
	} else {
		args.FromBlock = *raw.From
	}

	if raw.ToBlock == nil {
		args.ToBlock = rpc.LatestBlockNumber
	} else {
		args.ToBlock = *raw.ToBlock
	}

	args.Addresses = []common.Address{}

	if raw.Addresses != nil {
		// raw.Address can contain a single address or an array of addresses
		var addresses []common.Address

		if strAddrs, ok := raw.Addresses.([]interface{}); ok {
			for i, addr := range strAddrs {
				if strAddr, ok := addr.(string); ok {
					if len(strAddr) >= 2 && strAddr[0] == '0' && (strAddr[1] == 'x' || strAddr[1] == 'X') {
						strAddr = strAddr[2:]
					}
					if decAddr, err := hex.DecodeString(strAddr); err == nil {
						addresses = append(addresses, common.BytesToAddress(decAddr))
					} else {
						fmt.Errorf("invalid address given")
					}
				} else {
					return fmt.Errorf("invalid address on index %d", i)
				}
			}
		} else if singleAddr, ok := raw.Addresses.(string); ok {
			if len(singleAddr) >= 2 && singleAddr[0] == '0' && (singleAddr[1] == 'x' || singleAddr[1] == 'X') {
				singleAddr = singleAddr[2:]
			}
			if decAddr, err := hex.DecodeString(singleAddr); err == nil {
				addresses = append(addresses, common.BytesToAddress(decAddr))
			} else {
				fmt.Errorf("invalid address given")
			}
		} else {
			errors.New("invalid address(es) given")
		}
		args.Addresses = addresses
	}

	topicConverter := func(raw string) (common.Hash, error) {
		if len(raw) == 0 {
			return common.Hash{}, nil
		}

		if len(raw) >= 2 && raw[0] == '0' && (raw[1] == 'x' || raw[1] == 'X') {
			raw = raw[2:]
		}

		if decAddr, err := hex.DecodeString(raw); err == nil {
			return common.BytesToHash(decAddr), nil
		}

		return common.Hash{}, errors.New("invalid topic given")
	}

	// topics is an array consisting of strings or arrays of strings
	if raw.Topics != nil {
		topics, ok := raw.Topics.([]interface{})
		if ok {
			parsedTopics := make([][]common.Hash, len(topics))
			for i, topic := range topics {
				if topic == nil {
					parsedTopics[i] = []common.Hash{common.StringToHash("")}
				} else if strTopic, ok := topic.(string); ok {
					if t, err := topicConverter(strTopic); err != nil {
						return fmt.Errorf("invalid topic on index %d", i)
					} else {
						parsedTopics[i] = []common.Hash{t}
					}
				} else if arrTopic, ok := topic.([]interface{}); ok {
					parsedTopics[i] = make([]common.Hash, len(arrTopic))
					for j := 0; j < len(parsedTopics[i]); i++ {
						if arrTopic[j] == nil {
							parsedTopics[i][j] = common.StringToHash("")
						} else if str, ok := arrTopic[j].(string); ok {
							if t, err := topicConverter(str); err != nil {
								return fmt.Errorf("invalid topic on index %d", i)
							} else {
								parsedTopics[i] = []common.Hash{t}
							}
						} else {
							fmt.Errorf("topic[%d][%d] not a string", i, j)
						}
					}
				} else {
					return fmt.Errorf("topic[%d] invalid", i)
				}
			}
			args.Topics = parsedTopics
		}
	}

	return nil
}

// NewFilter creates a new filter and returns the filter id. It can be uses to retrieve logs.
func (s *PublicFilterAPI) NewFilter(args NewFilterArgs) (string, error) {
	externalId, err := newFilterId()
	if err != nil {
		return "", err
	}

	var id int
	if len(args.Addresses) > 0 {
		id = s.newLogFilter(args.FromBlock.Int64(), args.ToBlock.Int64(), args.Addresses, args.Topics)
	} else {
		id = s.newLogFilter(args.FromBlock.Int64(), args.ToBlock.Int64(), nil, args.Topics)
	}

	s.filterMapMu.Lock()
	s.filterMapping[externalId] = id
	s.filterMapMu.Unlock()

	return externalId, nil
}

// GetLogs returns the logs matching the given argument.
func (s *PublicFilterAPI) GetLogs(args NewFilterArgs) vm.Logs {
	filter := New(s.chainDb)
	filter.SetBeginBlock(args.FromBlock.Int64())
	filter.SetEndBlock(args.ToBlock.Int64())
	filter.SetAddresses(args.Addresses)
	filter.SetTopics(args.Topics)

	return returnLogs(filter.Find())
}

// UninstallFilter removes the filter with the given filter id.
func (s *PublicFilterAPI) UninstallFilter(filterId string) bool {
	s.filterMapMu.Lock()
	defer s.filterMapMu.Unlock()

	id, ok := s.filterMapping[filterId]
	if !ok {
		return false
	}

	defer s.filterManager.Remove(id)
	delete(s.filterMapping, filterId)

	if _, ok := s.logQueue[id]; ok {
		s.logMu.Lock()
		defer s.logMu.Unlock()
		delete(s.logQueue, id)
		return true
	}
	if _, ok := s.blockQueue[id]; ok {
		s.blockMu.Lock()
		defer s.blockMu.Unlock()
		delete(s.blockQueue, id)
		return true
	}
	if _, ok := s.transactionQueue[id]; ok {
		s.transactionMu.Lock()
		defer s.transactionMu.Unlock()
		delete(s.transactionQueue, id)
		return true
	}

	return false
}

// getFilterType is a helper utility that determine the type of filter for the given filter id.
func (s *PublicFilterAPI) getFilterType(id int) byte {
	if _, ok := s.blockQueue[id]; ok {
		return blockFilterTy
	} else if _, ok := s.transactionQueue[id]; ok {
		return transactionFilterTy
	} else if _, ok := s.logQueue[id]; ok {
		return logFilterTy
	}

	return unknownFilterTy
}

// blockFilterChanged returns a collection of block hashes for the block filter with the given id.
func (s *PublicFilterAPI) blockFilterChanged(id int) []common.Hash {
	s.blockMu.Lock()
	defer s.blockMu.Unlock()

	if s.blockQueue[id] != nil {
		return s.blockQueue[id].get()
	}
	return nil
}

// transactionFilterChanged returns a collection of transaction hashes for the pending
// transaction filter with the given id.
func (s *PublicFilterAPI) transactionFilterChanged(id int) []common.Hash {
	s.blockMu.Lock()
	defer s.blockMu.Unlock()

	if s.transactionQueue[id] != nil {
		return s.transactionQueue[id].get()
	}
	return nil
}

// logFilterChanged returns a collection of logs for the log filter with the given id.
func (s *PublicFilterAPI) logFilterChanged(id int) vm.Logs {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	if s.logQueue[id] != nil {
		return s.logQueue[id].get()
	}
	return nil
}

// GetFilterLogs returns the logs for the filter with the given id.
func (s *PublicFilterAPI) GetFilterLogs(filterId string) vm.Logs {
	id, ok := s.filterMapping[filterId]
	if !ok {
		return returnLogs(nil)
	}

	if filter := s.filterManager.Get(id); filter != nil {
		return returnLogs(filter.Find())
	}

	return returnLogs(nil)
}

// GetFilterChanges returns the logs for the filter with the given id since last time is was called.
// This can be used for polling.
func (s *PublicFilterAPI) GetFilterChanges(filterId string) interface{} {
	s.filterMapMu.Lock()
	id, ok := s.filterMapping[filterId]
	s.filterMapMu.Unlock()

	if !ok { // filter not found
		return []interface{}{}
	}

	switch s.getFilterType(id) {
	case blockFilterTy:
		return returnHashes(s.blockFilterChanged(id))
	case transactionFilterTy:
		return returnHashes(s.transactionFilterChanged(id))
	case logFilterTy:
		return returnLogs(s.logFilterChanged(id))
	}

	return []interface{}{}
}

type logQueue struct {
	mu sync.Mutex

	logs    vm.Logs
	timeout time.Time
	id      int
}

func (l *logQueue) add(logs ...*vm.Log) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = append(l.logs, logs...)
}

func (l *logQueue) get() vm.Logs {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.timeout = time.Now()
	tmp := l.logs
	l.logs = nil
	return tmp
}

type hashQueue struct {
	mu sync.Mutex

	hashes  []common.Hash
	timeout time.Time
	id      int
}

func (l *hashQueue) add(hashes ...common.Hash) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.hashes = append(l.hashes, hashes...)
}

func (l *hashQueue) get() []common.Hash {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.timeout = time.Now()
	tmp := l.hashes
	l.hashes = nil
	return tmp
}

// newFilterId generates a new random filter identifier that can be exposed to the outer world. By publishing random
// identifiers it is not feasible for DApp's to guess filter id's for other DApp's and uninstall or poll for them
// causing the affected DApp to miss data.
func newFilterId() (string, error) {
	var subid [16]byte
	n, _ := rand.Read(subid[:])
	if n != 16 {
		return "", errors.New("Unable to generate filter id")
	}
	return "0x" + hex.EncodeToString(subid[:]), nil
}

// returnLogs is a helper that will return an empty logs array case the given logs is nil, otherwise is will return the
// given logs. The RPC interfaces defines that always an array is returned.
func returnLogs(logs vm.Logs) vm.Logs {
	if logs == nil {
		return vm.Logs{}
	}
	return logs
}

// returnHashes is a helper that will return an empty hash array case the given hash array is nil, otherwise is will
// return the given hashes. The RPC interfaces defines that always an array is returned.
func returnHashes(hashes []common.Hash) []common.Hash {
	if hashes == nil {
		return []common.Hash{}
	}
	return hashes
}
