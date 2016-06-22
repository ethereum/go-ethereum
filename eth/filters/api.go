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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/net/context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicFilterAPI allows clients to retrieve or get notified of data such as blocks,
// transactions and logs.
type PublicFilterAPI struct {
	db ethdb.Database
	m  *Manager

	subscriptionsMu sync.Mutex
	subscriptions   map[string]FilterID // mapping between subscriptions and the underlying filters
}

// NewPublicFilterAPI returns a new PublicFilterAPI instance.
// It uses the given database to retrieve stored logs and the mux to listen for events.
func NewPublicFilterAPI(chainDb ethdb.Database, mux *event.TypeMux) *PublicFilterAPI {
	return &PublicFilterAPI{
		db:            chainDb,
		m:             NewManager(mux),
		subscriptions: make(map[string]FilterID),
	}
}

// NewBlockFilter creates a filter that returns hashes for new blocks.
// To check if the state has changed, call GetFilterChanges.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newblockfilter
func (api *PublicFilterAPI) NewBlockFilter() (FilterID, error) {
	return api.m.NewBlockFilter()
}

// NewPendingTransactionFilter creates a filter, to notify when new pending transactions arrive.
// To check if the state has changed, call GetFilterChanges.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newpendingtransactionfilter
func (api *PublicFilterAPI) NewPendingTransactionFilter() (FilterID, error) {
	return api.m.NewPendingTransactionFilter()
}

// GetFilterChanges returns new logs for the given filter since last time is was called.
// This can be used for polling.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterchanges
func (api *PublicFilterAPI) GetFilterChanges(id FilterID) (interface{}, error) {
	// filter might be deleted between retrieving filter type and calling the XXXFilterChanges
	// this is not an issue, XXXFilterChanges will return an errFilterNotFound error.
	switch api.m.FilterType(id) {
	case BlockFilter:
		hashes, err := api.m.GetBlockFilterChanges(id)
		return toRPCHashes(hashes), err
	case PendingTxFilter:
		hashes, err := api.m.GetPendingTxFilterChanges(id)
		return toRPCHashes(hashes), err
	case LogFilter:
		logs, err := api.m.GetLogFilterChanges(id)
		return toRPCLogs(logs), err
	}

	return nil, errFilterNotFound
}

// NewFilter creates a filter that can be used to fetch new logs.
// Note: as specification dictates, it can be used to fetch logs when the state changes.
// Older already created logs cannot be fetched using this method, use GetLogs.
// Therefore the fromBlock and toBlock parameters are ignored.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newfilter
func (api *PublicFilterAPI) NewFilter(crit FilterCriteria) (FilterID, error) {
	return api.m.NewLogFilter(crit, nil)
}

// GetLogs returns logs matching the given criteria.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (api *PublicFilterAPI) GetLogs(crit FilterCriteria) []Log {
	filter := New(api.db)
	filter.SetBeginBlock(crit.FromBlock.Int64())
	filter.SetEndBlock(crit.ToBlock.Int64())
	filter.SetAddresses(crit.Addresses)
	filter.SetTopics(crit.Topics)

	return toRPCLogs(filter.Find())
}

// GetFilterLogs return all logs that match for the given (log) filter.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterlogs
func (api *PublicFilterAPI) GetFilterLogs(id FilterID) ([]Log, error) {
	crit, err := api.m.GetLogFilterCriteria(id)
	if err != nil {
		return nil, err
	}
	return api.GetLogs(crit), nil
}

// Logs creates a subscription that fires each time a log is created that matches the given criteria.
// Note, in case of chain reorganisations logs with the Removed true property can be returned indicating
// a previously sent log is reverted.
func (api *PublicFilterAPI) Logs(ctx context.Context, crit FilterCriteria) (subscription rpc.Subscription, err error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	onUnsubscribe := func(subid string) {
		api.subscriptionsMu.Lock()
		defer api.subscriptionsMu.Unlock()
		if fid, found := api.subscriptions[subid]; found {
			delete(api.subscriptions, subid)
			api.m.Uninstall(fid) // uninstall associated filter
		}
	}

	subscription, err = notifier.NewSubscription(onUnsubscribe)
	if err != nil {
		return nil, err
	}

	send := func(id FilterID, logs []Log) {
		rpcLogs := toRPCLogs(logs)
		for _, l := range rpcLogs {
			subscription.Notify(l)
		}
	}

	fid, err := api.m.NewLogFilterWithNoTimeout(crit, send)
	if err != nil {
		return nil, err
	}

	api.subscriptionsMu.Lock()
	api.subscriptions[subscription.ID()] = fid
	api.subscriptionsMu.Unlock()

	return subscription, err
}

// UninstallFilter deletes a filter by its id. If successful true is returned,
// otherwise false.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_uninstallfilter
func (api *PublicFilterAPI) UninstallFilter(id FilterID) bool {
	return api.m.Uninstall(id) == nil
}

// toRPCLogs is a helper that will return an empty slice of logs when nil is given.
// Otherwise the given logs are returned. This is necessary for the RPC interface.
func toRPCLogs(logs []Log) []Log {
	if logs == nil {
		return []Log{}
	}
	return logs
}

// toRPCHashes is a helper that will return an empty hash array case the given hash
// array is nil, otherwise is will return the given hashes. The RPC interfaces defines
// that always an array is returned.
func toRPCHashes(hashes []common.Hash) []common.Hash {
	if hashes == nil {
		return []common.Hash{}
	}
	return hashes
}

// FilterCriteria represents a request to create a new filter.
type FilterCriteria struct {
	FromBlock rpc.BlockNumber
	ToBlock   rpc.BlockNumber
	Addresses []common.Address
	Topics    [][]common.Hash
}

// UnmarshalJSON sets *args fields with filter criteria JSON encoded in the given data.
func (args *FilterCriteria) UnmarshalJSON(data []byte) error {
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
