// Copyright 2019 The go-ethereum Authors
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

package les

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	errNoCheckpoint         = errors.New("no local checkpoint provided")
	errNotActivated         = errors.New("checkpoint registrar is not activated")
	errUnknownBenchmarkType = errors.New("unknown benchmark type")
	errClientNotConnected   = errors.New("client is not connected")
	errBalanceOverflow      = errors.New("balance overflow")
	errNoPriority           = errors.New("not enough priority")
)

const maxBalance = math.MaxInt64

type clientApiFields struct {
	balanceUpdatePeriod uint64
}

// PrivateLightServerAPI provides an API to access the LES light server.
type PrivateLightServerAPI struct {
	server                               *LesServer
	defaultPosFactors, defaultNegFactors priceFactors
	subs                                 map[*eventSub]struct{}
	lock                                 sync.Mutex
}

// NewPrivateLightServerAPI creates a new LES light server API.
func NewPrivateLightServerAPI(server *LesServer) *PrivateLightServerAPI {
	api := &PrivateLightServerAPI{
		server:            server,
		defaultPosFactors: server.clientPool.defaultPosFactors,
		defaultNegFactors: server.clientPool.defaultNegFactors,
		subs:              make(map[*eventSub]struct{}),
	}
	server.clientPool.eventHook = api.sendEvent
	return api
}

// ServerInfo returns global server parameters
func (api *PrivateLightServerAPI) ServerInfo() map[string]interface{} {
	res := make(map[string]interface{})
	res["minimumCapacity"] = api.server.minCapacity
	res["maximumCapacity"] = api.server.maxCapacity
	res["freeClientCapacity"] = api.server.freeCapacity
	res["totalCapacity"], res["totalConnectedCapacity"], res["priorityConnectedCapacity"] = api.server.clientPool.capacityInfo()
	return res
}

// ClientInfo returns information about clients listed in the ids list or matching the given tags
func (api *PrivateLightServerAPI) ClientInfo(ids []enode.ID) map[enode.ID]map[string]interface{} {
	res := make(map[enode.ID]map[string]interface{})
	api.server.clientPool.forClients(ids, func(client *clientInfo, id enode.ID) {
		res[id] = api.clientInfo(client, id)
	})
	return res
}

// PriorityClientInfo returns information about clients with a positive balance
// in the given ID range (stop excluded). If stop is null then the iterator stops
// only at the end of the ID space. MaxCount limits the number of results returned.
// If maxCount limit is applied but there are more potential results then the ID
// of the next potential result is included in the map with an empty structure
// assigned to it.
func (api *PrivateLightServerAPI) PriorityClientInfo(start, stop enode.ID, maxCount int) map[enode.ID]map[string]interface{} {
	res := make(map[enode.ID]map[string]interface{})
	ids := api.server.clientPool.ndb.getPosBalanceIDs(start, stop, maxCount+1)
	if len(ids) > maxCount {
		res[ids[maxCount]] = make(map[string]interface{})
		ids = ids[:maxCount]
	}
	api.server.clientPool.forClients(ids, func(client *clientInfo, id enode.ID) {
		res[id] = api.clientInfo(client, id)
	})
	return res
}

// clientInfo creates a client info data structure
func (api *PrivateLightServerAPI) clientInfo(c *clientInfo, id enode.ID) map[string]interface{} {
	info := make(map[string]interface{})
	if c != nil {
		info["isConnected"] = true
		info["capacity"] = c.capacity
		info["pricing/balance"], info["pricing/negBalance"] = c.balanceTracker.getBalance(mclock.Now())
		info["pricing/balanceMeta"] = c.balanceMetaInfo
	} else {
		info["isConnected"] = false
		pb := api.server.clientPool.getPosBalance(id)
		info["pricing/balance"], info["pricing/balanceMeta"] = pb.value, pb.meta
	}
	return info
}

// sendEvent sends an event to the subscribers interested in it. For global events client == nil.
func (api *PrivateLightServerAPI) sendEvent(clientEvent string, client *clientInfo) {
	if len(api.subs) == 0 {
		return
	}

	event := make(map[string]interface{})
	event["totalCapacity"], event["totalConnectedCapacity"], event["priorityConnectedCapacity"] = api.server.clientPool.capacityInfo()
	if client != nil {
		event["clientEvent"] = clientEvent
		event["clientId"] = client.id
		event["clientInfo"] = api.clientInfo(client, client.id)
	}

	for sub := range api.subs {
		select {
		case <-sub.rpcSub.Err():
			delete(api.subs, sub)
		case <-sub.notifier.Closed():
			delete(api.subs, sub)
		default:
			sub.notifier.Notify(sub.rpcSub.ID, event)
		}
	}
}

// setParams either sets the given parameters for a single client (if ID is specified)
// or the default parameters applicable to clients connected in the future
func (api *PrivateLightServerAPI) setParams(params map[string]interface{}, client *clientInfo, id enode.ID, posFactors, negFactors *priceFactors) (updateFactors bool, err error) {
	if client != nil {
		posFactors, negFactors = &client.posFactors, &client.negFactors
	}
	defParams := id == enode.ID{}
loop:
	for name, value := range params {
		errValue := func() error {
			return fmt.Errorf("invalid value for parameter '%s'", name)
		}
		setFactor := func(v *float64) {
			if posFactors != nil {
				if val, ok := value.(float64); ok && val >= 0 {
					*v = val / float64(time.Second)
					updateFactors = true
				} else {
					err = errValue()
				}
			} else {
				err = errClientNotConnected
			}
		}

		processed := true
		switch name {
		case "pricing/timeFactor":
			setFactor(&posFactors.timeFactor)
		case "pricing/capacityFactor":
			setFactor(&posFactors.capacityFactor)
		case "pricing/requestCostFactor":
			setFactor(&posFactors.requestFactor)
		case "pricing/negative/timeFactor":
			setFactor(&negFactors.timeFactor)
		case "pricing/negative/capacityFactor":
			setFactor(&negFactors.capacityFactor)
		case "pricing/negative/requestCostFactor":
			setFactor(&negFactors.requestFactor)
		default:
			processed = false
			if defParams {
				err = fmt.Errorf("invalid default parameter '%s'", name)
				continue loop
			}
		}
		if processed {
			continue loop
		}
		switch name {
		case "capacity":
			if client != nil {
				if capacity, ok := value.(float64); ok && (capacity == 0 || uint64(capacity) >= api.server.minCapacity) {
					err = api.server.clientPool.setCapacity(client, uint64(capacity))
					updateFactors = true
				} else {
					err = errValue()
				}
			} else {
				err = errClientNotConnected
			}
		case "pricing/alert":
			if client != nil {
				if val, ok := value.(float64); ok && val >= 0 {
					api.setBalanceUpdate(client, uint64(val), false)
				} else {
					err = errValue()
				}
			} else {
				err = errClientNotConnected
			}
		case "pricing/periodicUpdate":
			if client != nil {
				if val, ok := value.(float64); ok && val >= 0 {
					api.setBalanceUpdate(client, uint64(val), true)
				} else {
					err = errValue()
				}
			} else {
				err = errClientNotConnected
			}
		default:
			err = fmt.Errorf("invalid client parameter '%s'", name)
		}
	}
	return updateFactors, err
}

// UpdateBalance updates the balance of a client (either overwrites it or adds to it).
// It also updates the balance meta info string.
func (api *PrivateLightServerAPI) UpdateBalance(id enode.ID, value int64, meta string) error {
	return api.server.clientPool.updateBalance(id, value, meta)
}

// SetClientParams sets client parameters for all clients listed in the ids list
// or all connected clients if the list is empty
func (api *PrivateLightServerAPI) SetClientParams(ids []enode.ID, params map[string]interface{}) error {
	var finalErr error
	api.server.clientPool.forClients(ids, func(client *clientInfo, id enode.ID) {
		update, err := api.setParams(params, client, id, nil, nil)
		if err != nil {
			finalErr = err
		}
		if update {
			client.updatePriceFactors()
		}
	})
	return finalErr
}

// SetDefaultParams sets the default parameters applicable to clients connected in the future
func (api *PrivateLightServerAPI) SetDefaultParams(params map[string]interface{}) error {
	update, err := api.setParams(params, nil, enode.ID{}, &api.defaultPosFactors, &api.defaultNegFactors)
	if update {
		api.server.clientPool.setDefaultFactors(api.defaultPosFactors, api.defaultNegFactors)
	}
	return err
}

// balanceUpdate sends a price update client event and schedules a new update with the
// price tracker if necessary.
func (api *PrivateLightServerAPI) balanceUpdate(client *clientInfo) {
	api.lock.Lock()
	defer api.lock.Unlock()

	api.sendEvent("balanceUpdate", client)
	if client.balanceUpdatePeriod != 0 {
		api.setBalanceUpdate(client, client.balanceUpdatePeriod, true)
	}
}

// setBalanceUpdate schedules a price update when the balance reaches the given limit.
// If periodic is false then the limit is interpreted as an absolute value while if true
// it is relative to the current totalAmount value or the its value at the last future update.
func (api *PrivateLightServerAPI) setBalanceUpdate(client *clientInfo, value uint64, periodic bool) {
	balance := balance{pos: value}
	if periodic {
		client.balanceUpdatePeriod = value
		balance.pos, _ = client.balanceTracker.getBalance(mclock.Now())
		if balance.pos > value {
			balance.pos -= value
		} else {
			balance.pos = 0
		}
	} else {
		client.balanceUpdatePeriod = 0
	}
	client.balanceTracker.addCallback(balanceCallbackApi, client.balanceTracker.balanceToPriority(balance), func() { api.balanceUpdate(client) })
}

// eventSub represents an event subscription
type eventSub struct {
	notifier *rpc.Notifier
	rpcSub   *rpc.Subscription
}

// SubscribeEvent subscribes to global events and client events related to the clients matching the given tags.
// If totalCapUnderrun is true then totalCapacity updates are only sent when totalCapacity drops under totalConnectedCapacity.
func (api *PrivateLightServerAPI) SubscribeEvent(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()
	api.subs[&eventSub{notifier, rpcSub}] = struct{}{}
	return rpcSub, nil
}

// Benchmark runs a request performance benchmark with a given set of measurement setups
// in multiple passes specified by passCount. The measurement time for each setup in each
// pass is specified in milliseconds by length.
//
// Note: measurement time is adjusted for each pass depending on the previous ones.
// Therefore a controlled total measurement time is achievable in multiple passes.
func (api *PrivateLightServerAPI) Benchmark(setups []map[string]interface{}, passCount, length int) ([]map[string]interface{}, error) {
	benchmarks := make([]requestBenchmark, len(setups))
	for i, setup := range setups {
		if t, ok := setup["type"].(string); ok {
			getInt := func(field string, def int) int {
				if value, ok := setup[field].(float64); ok {
					return int(value)
				}
				return def
			}
			getBool := func(field string, def bool) bool {
				if value, ok := setup[field].(bool); ok {
					return value
				}
				return def
			}
			switch t {
			case "header":
				benchmarks[i] = &benchmarkBlockHeaders{
					amount:  getInt("amount", 1),
					skip:    getInt("skip", 1),
					byHash:  getBool("byHash", false),
					reverse: getBool("reverse", false),
				}
			case "body":
				benchmarks[i] = &benchmarkBodiesOrReceipts{receipts: false}
			case "receipts":
				benchmarks[i] = &benchmarkBodiesOrReceipts{receipts: true}
			case "proof":
				benchmarks[i] = &benchmarkProofsOrCode{code: false}
			case "code":
				benchmarks[i] = &benchmarkProofsOrCode{code: true}
			case "cht":
				benchmarks[i] = &benchmarkHelperTrie{
					bloom:    false,
					reqCount: getInt("amount", 1),
				}
			case "bloom":
				benchmarks[i] = &benchmarkHelperTrie{
					bloom:    true,
					reqCount: getInt("amount", 1),
				}
			case "txSend":
				benchmarks[i] = &benchmarkTxSend{}
			case "txStatus":
				benchmarks[i] = &benchmarkTxStatus{}
			default:
				return nil, errUnknownBenchmarkType
			}
		} else {
			return nil, errUnknownBenchmarkType
		}
	}
	rs := api.server.handler.runBenchmark(benchmarks, passCount, time.Millisecond*time.Duration(length))
	result := make([]map[string]interface{}, len(setups))
	for i, r := range rs {
		res := make(map[string]interface{})
		if r.err == nil {
			res["totalCount"] = r.totalCount
			res["avgTime"] = r.avgTime
			res["maxInSize"] = r.maxInSize
			res["maxOutSize"] = r.maxOutSize
		} else {
			res["error"] = r.err.Error()
		}
		result[i] = res
	}
	return result, nil
}

// PrivateDebugAPI provides an API to debug LES light server functionality.
type PrivateDebugAPI struct {
	server *LesServer
}

// NewPrivateDebugAPI creates a new LES light server debug API.
func NewPrivateDebugAPI(server *LesServer) *PrivateDebugAPI {
	return &PrivateDebugAPI{
		server: server,
	}
}

// FreezeClient forces a temporary client freeze which normally happens when the server is overloaded
func (api *PrivateDebugAPI) FreezeClient(id enode.ID) error {
	err := errClientNotConnected
	api.server.clientPool.forClients([]enode.ID{id}, func(c *clientInfo, id enode.ID) {
		c.peer.freezeClient()
		err = nil
	})
	return err
}

// PrivateLightAPI provides an API to access the LES light server or light client.
type PrivateLightAPI struct {
	backend *lesCommons
}

// NewPrivateLightAPI creates a new LES service API.
func NewPrivateLightAPI(backend *lesCommons) *PrivateLightAPI {
	return &PrivateLightAPI{backend: backend}
}

// LatestCheckpoint returns the latest local checkpoint package.
//
// The checkpoint package consists of 4 strings:
//   result[0], hex encoded latest section index
//   result[1], 32 bytes hex encoded latest section head hash
//   result[2], 32 bytes hex encoded latest section canonical hash trie root hash
//   result[3], 32 bytes hex encoded latest section bloom trie root hash
func (api *PrivateLightAPI) LatestCheckpoint() ([4]string, error) {
	var res [4]string
	cp := api.backend.latestLocalCheckpoint()
	if cp.Empty() {
		return res, errNoCheckpoint
	}
	res[0] = hexutil.EncodeUint64(cp.SectionIndex)
	res[1], res[2], res[3] = cp.SectionHead.Hex(), cp.CHTRoot.Hex(), cp.BloomRoot.Hex()
	return res, nil
}

// GetLocalCheckpoint returns the specific local checkpoint package.
//
// The checkpoint package consists of 3 strings:
//   result[0], 32 bytes hex encoded latest section head hash
//   result[1], 32 bytes hex encoded latest section canonical hash trie root hash
//   result[2], 32 bytes hex encoded latest section bloom trie root hash
func (api *PrivateLightAPI) GetCheckpoint(index uint64) ([3]string, error) {
	var res [3]string
	cp := api.backend.localCheckpoint(index)
	if cp.Empty() {
		return res, errNoCheckpoint
	}
	res[0], res[1], res[2] = cp.SectionHead.Hex(), cp.CHTRoot.Hex(), cp.BloomRoot.Hex()
	return res, nil
}

// GetCheckpointContractAddress returns the contract contract address in hex format.
func (api *PrivateLightAPI) GetCheckpointContractAddress() (string, error) {
	if api.backend.oracle == nil {
		return "", errNotActivated
	}
	return api.backend.oracle.config.Address.Hex(), nil
}
