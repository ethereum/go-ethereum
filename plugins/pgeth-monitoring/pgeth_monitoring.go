package pgeth_monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
  "github.com/redis/go-redis/v9"
  "github.com/attestantio/go-eth2-client/http"
	"github.com/rs/zerolog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pgeth/toolkit"
	"github.com/ethereum/go-ethereum/plugins/pgeth-monitoring/pkg/tracer"
	"github.com/ethereum/go-ethereum/rpc"
)

func Version() {
	fmt.Println("pgeth-monitoring: v0.0.1")
}

// plugin.so entrypoint
func Start(pt *toolkit.PluginToolkit, cfg map[string]interface{}, ctx context.Context, errChan chan error) {
	var redisEndpointRaw interface{}
	var redisEndpoint string
	var beaconEndpointRaw interface{}
	var beaconEndpoint string
	var ok bool

	if redisEndpointRaw, ok = cfg["REDIS_ENDPOINT"]; !ok {
		pt.Logger.Error("missing REDIS_ENDPOINT config var")
		return
	}

	if redisEndpoint, ok = redisEndpointRaw.(string); !ok {
		pt.Logger.Error("invalid REDIS_ENDPOINT value")
		return
	}

	if beaconEndpointRaw, ok = cfg["BEACON_ENDPOINT"]; !ok {
		pt.Logger.Error("missing BEACON_ENDPOINT config var")
		return
	}

	if beaconEndpoint, ok = beaconEndpointRaw.(string); !ok {
		pt.Logger.Error("invalid BEACON_ENDPOINT value")
		return
	}

	client, err := http.New(ctx,
		http.WithAddress(beaconEndpoint),
		http.WithLogLevel(zerolog.WarnLevel),
	)
	if err != nil {
		errChan <- err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisEndpoint,
		Password: "",
		DB:       0,
	})
	me := NewMonitoringEngine(pt, rdb, client, beaconEndpoint, errChan)

	me.Start(ctx)

}

// MonitoringEngine is the main plugin struct
// It contains the plugin toolkit, the redis client, and the eth2 client
type MonitoringEngine struct {
	ptk *toolkit.PluginToolkit

	backend     *eth.EthAPIBackend
	chainConfig *params.ChainConfig
	coinbase    common.Address
	state       *state.StateDB
	header      *types.Header

	latestBlock *types.Block

	beaconEndpoint string

	errChan chan error

	rdb  *redis.Client
	eth2 eth2client.Service
}

// AnalyzedTransaction is a transaction with its traces
// It is used to encode the transaction and its traces in a redis key
// Then the entire struct is sent to redis
type AnalyzedTransaction struct {
	Transaction *types.Transaction `json:"transaction"`
	From        common.Address     `json:"from"`
	Receipt     *types.Receipt     `json:"receipt"`
	Traces      tracer.Action      `json:"traces"`
}

// CachedBlockSimulation is a block simulation that is cached in memory
// It is used to avoid simulating the same block twice for finalized blocks
// Two hours after the block is cached, it is removed from memory
// When a cache block is found as finalized, it is removed from memory
type CachedBlockSimulation struct {
	Time                 time.Time
	AnalyzedTransactions []AnalyzedTransaction
}

func NewMonitoringEngine(pt *toolkit.PluginToolkit, rdb *redis.Client, eth2 eth2client.Service, beaconEndpoint string, errChan chan error) *MonitoringEngine {

	return &MonitoringEngine{
		ptk:            pt,
		backend:        pt.Backend.(*eth.EthAPIBackend),
		chainConfig:    pt.Backend.ChainConfig(),
		beaconEndpoint: beaconEndpoint,
		errChan:        errChan,
		rdb:            rdb,
		eth2:           eth2,
	}
}

func (me *MonitoringEngine) Start(ctx context.Context) {
	me.startHeadListener(ctx)
}

func (me *MonitoringEngine) update(ctx context.Context, parent *types.Block) {

	state, _, err := me.backend.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHashWithHash(parent.Hash(), true))
	if err != nil {
		me.errChan <- err
		return
	}
	me.header = &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		GasLimit:   parent.GasLimit(),
		Time:       parent.Time() + 12,
		Coinbase:   parent.Coinbase(),
		BaseFee:    eip1559.CalcBaseFee(me.chainConfig, parent.Header()),
		Difficulty: parent.Difficulty(),
	}
	me.coinbase = parent.Coinbase()
	me.state = state
}

func (me *MonitoringEngine) encodeAndBroadcastCallTrace(ctx context.Context, at *AnalyzedTransaction, channel string) {
	var topic string
	if at.Transaction.To() == nil {
		topic = fmt.Sprintf("/%s/tx/%s/%s/null/%s", channel, at.Transaction.Hash(), at.From, encodeActionCalls(at.Traces))
	} else {
		topic = fmt.Sprintf("/%s/tx/%s/%s/%s/%s", channel, at.Transaction.Hash(), at.From, at.Transaction.To(), encodeActionCalls(at.Traces))

	}

	jsoned, err := json.Marshal(*at)
	if err != nil {
		me.errChan <- err
		return
	}

	err = me.rdb.Publish(ctx, topic, jsoned).Err()
	if err != nil {
		me.errChan <- err
	}

	err = me.rdb.Expire(ctx, topic, 1*time.Hour).Err()
	if err != nil {
		me.errChan <- err
	}
}

func (me *MonitoringEngine) encodeAndBroadcast(ctx context.Context, ats []AnalyzedTransaction, channel string) {
	for _, analyzedTx := range ats {
		me.encodeAndBroadcastCallTrace(ctx, &analyzedTx, channel)
	}
}

// analyze is the main function of the plugin
// It takes a block and returns a list of AnalyzedTransaction
// It simulates the block and gets the traces of all the transactions
func (me *MonitoringEngine) analyze(ctx context.Context, block *types.Block, scope string) []AnalyzedTransaction {
	// We retrieve the parent block
	parentBlk, err := me.backend.BlockByHash(ctx, block.ParentHash())
	if err != nil {
		me.errChan <- err
		return nil
	}
	// We retrieve the state of the parent block
	state, _, err := me.backend.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHashWithHash(parentBlk.Hash(), true))
	if err != nil {
		me.errChan <- err
		return nil
	}
	// We configure the vm to use our monitoring tracer
	gp := new(core.GasPool).AddGas(block.Header().GasLimit)
	mt := tracer.MonitoringTracer{}
	var vmConfig vm.Config = vm.Config{
		Tracer:                  &mt,
		NoBaseFee:               false,
		EnablePreimageRecording: false,
		ExtraEips:               []int{},
	}
	analyzedTransactions := []AnalyzedTransaction{}
	// We simulate all the transactions of the block
	for idx, tx := range block.Transactions() {
		state.SetTxContext(tx.Hash(), idx)
		receipt, err := core.ApplyTransaction(me.chainConfig, me.backend.Ethereum().BlockChain(), &block.Header().Coinbase, gp, state, block.Header(), tx, &block.Header().GasUsed, vmConfig)
		if err != nil {
			me.errChan <- err
			return nil
		}
		receipt.EffectiveGasPrice = getEffectiveGasPrice(tx, parentBlk.BaseFee())
		signer := types.MakeSigner(me.backend.Ethereum().BlockChain().Config(), receipt.BlockNumber, block.Time())
		from, _ := types.Sender(signer, tx)
		analyzedTransactions = append(analyzedTransactions, AnalyzedTransaction{
			Transaction: tx,
			From:        from,
			Receipt:     receipt,
			Traces:      mt.Action,
		})
		mt.Clear()
	}
	me.ptk.Logger.Info("Simulated txs", "count", len(block.Transactions()), "scope", scope, "number", block.Number())
	// We encode and broadcast the traces
	me.encodeAndBroadcast(ctx, analyzedTransactions, scope)
	me.ptk.Logger.Info("Broadcasted txs", "count", len(block.Transactions()), "scope", scope, "number", block.Number())
	me.latestBlock = block
	return analyzedTransactions
}

func (me *MonitoringEngine) analyzePending(ctx context.Context, txs []*types.Transaction) {
	gp := new(core.GasPool).AddGas(me.header.GasLimit)
	mt := tracer.MonitoringTracer{}
	var vmConfig vm.Config = vm.Config{
		Tracer:                  &mt,
		NoBaseFee:               true,
		EnablePreimageRecording: false,
		ExtraEips:               []int{},
	}
	analyzedTransactions := []AnalyzedTransaction{}
	for _, tx := range txs {
		// we copy the state at the head
		stateClone := me.state.Copy()
		stateClone.SetTxContext(tx.Hash(), 0)
		// we simulate the pending tx on top of it
		receipt, err := core.ApplyTransaction(me.chainConfig, me.backend.Ethereum().BlockChain(), &me.header.Coinbase, gp, stateClone, me.header, tx, &me.header.GasUsed, vmConfig)
		if err != nil {
			continue
		}
		receipt.EffectiveGasPrice = getEffectiveGasPrice(tx, me.header.BaseFee)
		signer := types.MakeSigner(me.backend.Ethereum().BlockChain().Config(), receipt.BlockNumber, me.header.Time)
		from, _ := types.Sender(signer, tx)
		analyzedTransactions = append(analyzedTransactions, AnalyzedTransaction{
			Transaction: tx,
			From:        from,
			Receipt:     receipt,
			Traces:      mt.Action,
		})
		mt.Clear()
	}
	if len(analyzedTransactions) > 0 {
		// we encode and broadcast the traces under the pending topic
		me.encodeAndBroadcast(ctx, analyzedTransactions, "pending")
	}
}

func (me *MonitoringEngine) startHeadListener(ctx context.Context) {

	headChan := make(chan core.ChainHeadEvent)
	headSubscription := me.backend.SubscribeChainHeadEvent(headChan)
	pendingChan := make(chan core.NewTxsEvent)
	pendingSubscription := me.backend.SubscribeNewTxsEvent(pendingChan)
	ticker := time.NewTicker(30 * time.Second)
	cache := make(map[common.Hash]CachedBlockSimulation)
	analyzedPendingTxs := 0

	highestFinalized := uint64(0)

	defer headSubscription.Unsubscribe()
	defer pendingSubscription.Unsubscribe()
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-headSubscription.Err():
			if err != nil {
				me.errChan <- err
			}
			return
		case err := <-pendingSubscription.Err():
			if err != nil {
				me.errChan <- err
			}
			return
		case newHead := <-headChan:
			me.update(ctx, newHead.Block)
			me.ptk.Logger.Info(fmt.Sprintf("Head was updated, number: %d, hash: %s, root: %s", newHead.Block.NumberU64(), newHead.Block.Hash(), newHead.Block.Root()))

			analyzedTransactions := me.analyze(ctx, newHead.Block, "head")
			cache[newHead.Block.Hash()] = CachedBlockSimulation{
				Time:                 time.Now(),
				AnalyzedTransactions: analyzedTransactions,
			}
		case newTxs := <-pendingChan:
			if me.header == nil {
				me.ptk.Logger.Warn("Skipping pending tx simulation, not ready")
			} else {
				me.analyzePending(ctx, newTxs.Txs)
				analyzedPendingTxs += len(newTxs.Txs)
			}
		case <-ticker.C:
			if analyzedPendingTxs > 0 {
				me.ptk.Logger.Info("Analyzed pending txs", "count", analyzedPendingTxs, "rate", fmt.Sprintf("%f/s", float64(analyzedPendingTxs)/float64(30)))
				analyzedPendingTxs = 0
			}

			if me.eth2 == nil {
				client, err := http.New(ctx,
					http.WithAddress(me.beaconEndpoint),
					http.WithLogLevel(zerolog.WarnLevel),
				)
				if err != nil {
					me.errChan <- err
					continue
				}
				me.eth2 = client
			}

			res, err := me.eth2.(eth2client.SignedBeaconBlockProvider).SignedBeaconBlock(ctx, "finalized")
			if err != nil {
				me.errChan <- err
				continue
			}

			newHighestFinalized := res.Capella.Message.Body.ExecutionPayload.BlockNumber
			if highestFinalized == 0 {
				highestFinalized = res.Capella.Message.Body.ExecutionPayload.BlockNumber
				me.ptk.Logger.Info("Finalized head was updated", "number", highestFinalized)
			} else if newHighestFinalized > highestFinalized {
				me.ptk.Logger.Info("Updating finalized head", "from", highestFinalized, "to", newHighestFinalized)
				breaked := false
				for i := highestFinalized + 1; i <= newHighestFinalized; i++ {
					blk, err := me.backend.BlockByNumber(ctx, rpc.BlockNumber(i))
					if err != nil {
						me.errChan <- err
						breaked = true
						break
					}
					if blk == nil {
						breaked = true
						break
					}
					if cachedValue, ok := cache[blk.Hash()]; ok {
						me.encodeAndBroadcast(ctx, cachedValue.AnalyzedTransactions, "finalized")
						me.ptk.Logger.Info("Broadcasted txs", "count", len(cachedValue.AnalyzedTransactions), "scope", "finalized", "number", i)
						delete(cache, blk.Hash())
					}
				}
				if breaked {
					continue
				}
				me.ptk.Logger.Info("Finalized head was updated", "number", newHighestFinalized)
				highestFinalized = newHighestFinalized

			}

			for k, v := range cache {
				if time.Since(v.Time) > 2*time.Hour {
					delete(cache, k)
				}
			}
			if len(cache) > 0 {
				me.ptk.Logger.Info("Cached blocks", "count", len(cache))
			}
		}
	}
}

func getEffectiveGasPrice(tx *types.Transaction, baseFee *big.Int) *big.Int {
	switch tx.Type() {
	case types.DynamicFeeTxType:
		if baseFee == nil {
			return tx.GasFeeCap()
		}
		tip := new(big.Int).Sub(tx.GasFeeCap(), baseFee)
		if tip.Cmp(tx.GasTipCap()) > 0 {
			tip.Set(tx.GasTipCap())
		}
		return tip.Add(tip, baseFee)
	case types.AccessListTxType:
		return tx.GasPrice()
	case types.LegacyTxType:
		return tx.GasPrice()
	default:
		return big.NewInt(0)
	}
}

func callTypeToPrefix(c *tracer.Call) string {
	switch c.Type() {
	case "call":
		return "C"
	case "staticcall":
		return "S"
	case "delegatecall":
		return "D"
	case "initial_call":
		return "C"
	}
	return "X"
}

func eventTypeToPrefix(e *tracer.Event) string {
	switch e.LogType {
	case "log0":
		fallthrough
	case "log1":
		fallthrough
	case "log2":
		fallthrough
	case "log3":
		fallthrough
	case "log4":
		return "L"
	}
	return "X"
}

func revertTypeToPrefix(r *tracer.Revert) string {
	switch r.ErrorType {
	case "revert":
		return "R"
	case "panic":
		return "P"
	}
	return "X"
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func encodeSelector(c *tracer.Call) string {
	selector := fmt.Sprintf("%x", c.In[0:minInt(4, len(c.In))])
	for len(selector) < 8 {
		selector += "X"
	}
	return selector
}

func encodeRevertSelector(c *tracer.Revert) string {
	selector := fmt.Sprintf("%x", c.Data[0:minInt(4, len(c.Data))])
	for len(selector) < 8 {
		selector += "X"
	}
	return selector
}

func encodeActionCalls(a tracer.Action) string {
	res := ""
	if c, ok := a.(*tracer.Call); ok {
		prefix := callTypeToPrefix(c)
		selector := encodeSelector(c)
		res = fmt.Sprintf("%s@%s_%s", prefix, c.To.String(), selector)
		if len(a.Children()) > 0 {
			chldArr := []string{}
			for _, chld := range a.Children() {
				chldRes := encodeActionCalls(chld)
				if len(chldRes) > 0 {
					chldArr = append(chldArr, chldRes)
				}
			}
			if len(chldArr) > 0 {
				joinedChldRes := strings.Join(chldArr[:], ",")
				res = fmt.Sprintf("%s[%s]", res, joinedChldRes)
			}
		}
	}
	if e, ok := a.(*tracer.Event); ok {
		concatenatedTopics := ""
		for _, topic := range e.Topics {
			concatenatedTopics += topic.String()[2:]
		}
		dataLength := len(e.Data)
		res = fmt.Sprintf("%s@%s_%s_%d", eventTypeToPrefix(e), e.ContextValue.String(), concatenatedTopics, dataLength)
	}
	if r, ok := a.(*tracer.Revert); ok {
		prefix := revertTypeToPrefix(r)
		selector := encodeRevertSelector(r)
		res = fmt.Sprintf("%s@%s_%s", prefix, r.ContextValue.String(), selector)
	}

	return res
}
