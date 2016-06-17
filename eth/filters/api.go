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

package filters

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"

	"golang.org/x/net/context"
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

// PublicFilterAPI offers support to create and manage filters. This will allow external clients to retrieve various
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
			s.filterManager.Lock() // lock order like filterLoop()
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
			s.filterManager.Unlock()
		case <-s.quit:
			break done
		}
	}

}

// NewBlockFilter create a new filter that returns blocks that are included into the canonical chain.
func (s *PublicFilterAPI) NewBlockFilter() (string, error) {
	// protect filterManager.Add() and setting of filter fields
	s.filterManager.Lock()
	defer s.filterManager.Unlock()

	externalId, err := newFilterId()
	if err != nil {
		return "", err
	}

	filter := New(s.chainDb)
	id, err := s.filterManager.Add(filter, ChainFilter)
	if err != nil {
		return "", err
	}

	s.blockMu.Lock()
	s.blockQueue[id] = &hashQueue{timeout: time.Now()}
	s.blockMu.Unlock()

	filter.BlockCallback = func(block *types.Block, logs vm.Logs) {
		s.blockMu.Lock()
		defer s.blockMu.Unlock()

		if queue := s.blockQueue[id]; queue != nil {
			queue.add(block.Hash())
		}
	}

	s.filterMapMu.Lock()
	s.filterMapping[externalId] = id
	s.filterMapMu.Unlock()

	return externalId, nil
}

// NewPendingTransactionFilter creates a filter that returns new pending transactions.
func (s *PublicFilterAPI) NewPendingTransactionFilter() (string, error) {
	// protect filterManager.Add() and setting of filter fields
	s.filterManager.Lock()
	defer s.filterManager.Unlock()

	externalId, err := newFilterId()
	if err != nil {
		return "", err
	}

	filter := New(s.chainDb)
	id, err := s.filterManager.Add(filter, PendingTxFilter)
	if err != nil {
		return "", err
	}

	s.transactionMu.Lock()
	s.transactionQueue[id] = &hashQueue{timeout: time.Now()}
	s.transactionMu.Unlock()

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
func (s *PublicFilterAPI) newLogFilter(earliest, latest int64, addresses []common.Address, topics [][]common.Hash, callback func(log *vm.Log, removed bool)) (int, error) {
	// protect filterManager.Add() and setting of filter fields
	s.filterManager.Lock()
	defer s.filterManager.Unlock()

	filter := New(s.chainDb)
	id, err := s.filterManager.Add(filter, LogFilter)
	if err != nil {
		return 0, err
	}

	s.logMu.Lock()
	s.logQueue[id] = &logQueue{timeout: time.Now()}
	s.logMu.Unlock()

	filter.SetBeginBlock(earliest)
	filter.SetEndBlock(latest)
	filter.SetAddresses(addresses)
	filter.SetTopics(topics)
	filter.LogCallback = func(log *vm.Log, removed bool) {
		if callback != nil {
			callback(log, removed)
		} else {
			s.logMu.Lock()
			defer s.logMu.Unlock()
			if queue := s.logQueue[id]; queue != nil {
				queue.add(vmlog{log, removed})
			}
		}
	}

	return id, nil
}

// Logs creates a subscription that fires for all new log that match the given filter criteria.
func (s *PublicFilterAPI) Logs(ctx context.Context, args NewFilterArgs) (rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	var (
		externalId   string
		subscription rpc.Subscription
		err          error
	)

	if externalId, err = newFilterId(); err != nil {
		return nil, err
	}

	// uninstall filter when subscription is unsubscribed/cancelled
	if subscription, err = notifier.NewSubscription(func(string) {
		s.UninstallFilter(externalId)
	}); err != nil {
		return nil, err
	}

	notifySubscriber := func(log *vm.Log, removed bool) {
		rpcLog := toRPCLogs(vm.Logs{log}, removed)
		if err := subscription.Notify(rpcLog); err != nil {
			subscription.Cancel()
		}
	}

	// from and to block number are not used since subscriptions don't allow you to travel to "time"
	var id int
	if len(args.Addresses) > 0 {
		id, err = s.newLogFilter(-1, -1, args.Addresses, args.Topics, notifySubscriber)
	} else {
		id, err = s.newLogFilter(-1, -1, nil, args.Topics, notifySubscriber)
	}

	if err != nil {
		subscription.Cancel()
		return nil, err
	}

	s.filterMapMu.Lock()
	s.filterMapping[externalId] = id
	s.filterMapMu.Unlock()

	return subscription, err
}

// NewFilterArgs represents a request to create a new filter.
type NewFilterArgs struct {
	FromBlock rpc.BlockNumber
	ToBlock   rpc.BlockNumber
	Addresses []common.Address
	Topics    [][]common.Hash
}

// UnmarshalJSON sets *args fields with given data.
func (args *NewFilterArgs) UnmarshalJSON(data []byte) error {
	type input struct {
		From      *rpc.BlockNumber `json:"fromBlock"`
		ToBlock   *rpc.BlockNumber `json:"toBlock"`
		Addresses interface{}      `json:"address"`
		Topics    []interface{}    `json:"topics"`
	}

	var raw input
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.From == nil || raw.From.Int64() < 0 {
		args.FromBlock = rpc.LatestBlockNumber
	} else {
		args.FromBlock = *raw.From
	}

	if raw.ToBlock == nil || raw.ToBlock.Int64() < 0 {
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
						return fmt.Errorf("invalid address given")
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
				return fmt.Errorf("invalid address given")
			}
		} else {
			return errors.New("invalid address(es) given")
		}
		args.Addresses = addresses
	}

	// helper function which parses a string to a topic hash
	topicConverter := func(raw string) (common.Hash, error) {
		if len(raw) == 0 {
			return common.Hash{}, nil
		}
		if len(raw) >= 2 && raw[0] == '0' && (raw[1] == 'x' || raw[1] == 'X') {
			raw = raw[2:]
		}
		if len(raw) != 2*common.HashLength {
			return common.Hash{}, errors.New("invalid topic(s)")
		}
		if decAddr, err := hex.DecodeString(raw); err == nil {
			return common.BytesToHash(decAddr), nil
		}
		return common.Hash{}, errors.New("invalid topic(s)")
	}

	// topics is an array consisting of strings and/or arrays of strings.
	// JSON null values are converted to common.Hash{} and ignored by the filter manager.
	if len(raw.Topics) > 0 {
		args.Topics = make([][]common.Hash, len(raw.Topics))
		for i, t := range raw.Topics {
			if t == nil { // ignore topic when matching logs
				args.Topics[i] = []common.Hash{common.Hash{}}
			} else if topic, ok := t.(string); ok { // match specific topic
				top, err := topicConverter(topic)
				if err != nil {
					return err
				}
				args.Topics[i] = []common.Hash{top}
			} else if topics, ok := t.([]interface{}); ok { // or case e.g. [null, "topic0", "topic1"]
				for _, rawTopic := range topics {
					if rawTopic == nil {
						args.Topics[i] = append(args.Topics[i], common.Hash{})
					} else if topic, ok := rawTopic.(string); ok {
						parsed, err := topicConverter(topic)
						if err != nil {
							return err
						}
						args.Topics[i] = append(args.Topics[i], parsed)
					} else {
						return fmt.Errorf("invalid topic(s)")
					}
				}
			} else {
				return fmt.Errorf("invalid topic(s)")
			}
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
		id, err = s.newLogFilter(args.FromBlock.Int64(), args.ToBlock.Int64(), args.Addresses, args.Topics, nil)
	} else {
		id, err = s.newLogFilter(args.FromBlock.Int64(), args.ToBlock.Int64(), nil, args.Topics, nil)
	}
	if err != nil {
		return "", err
	}

	s.filterMapMu.Lock()
	s.filterMapping[externalId] = id
	s.filterMapMu.Unlock()

	return externalId, nil
}

// GetLogs returns the logs matching the given argument.
func (s *PublicFilterAPI) GetLogs(args NewFilterArgs) []vmlog {
	filter := New(s.chainDb)
	filter.SetBeginBlock(args.FromBlock.Int64())
	filter.SetEndBlock(args.ToBlock.Int64())
	filter.SetAddresses(args.Addresses)
	filter.SetTopics(args.Topics)

	return toRPCLogs(filter.Find(), false)
}

// UninstallFilter removes the filter with the given filter id.
func (s *PublicFilterAPI) UninstallFilter(filterId string) bool {
	s.filterManager.Lock()
	defer s.filterManager.Unlock()

	s.filterMapMu.Lock()
	id, ok := s.filterMapping[filterId]
	if !ok {
		s.filterMapMu.Unlock()
		return false
	}
	delete(s.filterMapping, filterId)
	s.filterMapMu.Unlock()

	s.filterManager.Remove(id)

	s.logMu.Lock()
	if _, ok := s.logQueue[id]; ok {
		delete(s.logQueue, id)
		s.logMu.Unlock()
		return true
	}
	s.logMu.Unlock()

	s.blockMu.Lock()
	if _, ok := s.blockQueue[id]; ok {
		delete(s.blockQueue, id)
		s.blockMu.Unlock()
		return true
	}
	s.blockMu.Unlock()

	s.transactionMu.Lock()
	if _, ok := s.transactionQueue[id]; ok {
		delete(s.transactionQueue, id)
		s.transactionMu.Unlock()
		return true
	}
	s.transactionMu.Unlock()

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
func (s *PublicFilterAPI) logFilterChanged(id int) []vmlog {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	if s.logQueue[id] != nil {
		return s.logQueue[id].get()
	}
	return nil
}

// GetFilterLogs returns the logs for the filter with the given id.
func (s *PublicFilterAPI) GetFilterLogs(filterId string) []vmlog {
	s.filterMapMu.RLock()
	id, ok := s.filterMapping[filterId]
	s.filterMapMu.RUnlock()
	if !ok {
		return toRPCLogs(nil, false)
	}

	if filter := s.filterManager.Get(id); filter != nil {
		return toRPCLogs(filter.Find(), false)
	}

	return toRPCLogs(nil, false)
}

// GetFilterChanges returns the logs for the filter with the given id since last time is was called.
// This can be used for polling.
func (s *PublicFilterAPI) GetFilterChanges(filterId string) interface{} {
	s.filterMapMu.RLock()
	id, ok := s.filterMapping[filterId]
	s.filterMapMu.RUnlock()

	if !ok { // filter not found
		return []interface{}{}
	}

	switch s.getFilterType(id) {
	case blockFilterTy:
		return returnHashes(s.blockFilterChanged(id))
	case transactionFilterTy:
		return returnHashes(s.transactionFilterChanged(id))
	case logFilterTy:
		return s.logFilterChanged(id)
	}

	return []interface{}{}
}

type vmlog struct {
	*vm.Log
	Removed bool `json:"removed"`
}

type logQueue struct {
	mu sync.Mutex

	logs    []vmlog
	timeout time.Time
	id      int
}

func (l *logQueue) add(logs ...vmlog) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = append(l.logs, logs...)
}

func (l *logQueue) get() []vmlog {
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

// toRPCLogs is a helper that will convert a vm.Logs array to an structure which
// can hold additional information about the logs such as whether it was deleted.
// Additionally when nil is given it will by default instead create an empty slice
// instead. This is required by the RPC specification.
func toRPCLogs(logs vm.Logs, removed bool) []vmlog {
	convertedLogs := make([]vmlog, len(logs))
	for i, log := range logs {
		convertedLogs[i] = vmlog{Log: log, Removed: removed}
	}
	return convertedLogs
}

// returnHashes is a helper that will return an empty hash array case the given hash array is nil, otherwise is will
// return the given hashes. The RPC interfaces defines that always an array is returned.
func returnHashes(hashes []common.Hash) []common.Hash {
	if hashes == nil {
		return []common.Hash{}
	}
	return hashes
}
