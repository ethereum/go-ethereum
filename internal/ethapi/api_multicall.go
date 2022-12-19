package ethapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"
)

type multiCallResp struct {
	Results []*callResult   `json:"results"`
	Stats   *multiCallStats `json:"stats"`
}

type callResult struct {
	Code      int           `json:"code"`
	Err       string        `json:"err"`
	FromCache bool          `json:"fromCache"`
	Result    hexutil.Bytes `json:"result"`
	GasUsed   int64         `json:"gasUsed"`
	TimeCost  float64       `json:"timeCost"`
}

type multiCallStats struct {
	BlockNum     int64       `json:"blockNum"`
	BlockHash    common.Hash `json:"blockHash"`
	BlockTime    int64       `json:"blockTime"`
	Success      bool        `json:"success"`
	CacheEnabled bool        `json:"cacheEnabled"`
	// gasUsed, excluding calls from cache
	GasUsed       int64 `json:"gasUsed"`
	OriginGasUsed int64 `json:"originGasUsed"`
	CacheHitCount int64 `json:"cacheHitCount"`
}

const (
	singleCallTimeout = 1 * time.Second
	multiCallLimit    = 50

	errParam     = -40001
	errConsensus = -40002 // error on consensus check
	errLogic     = -40003 // logic error
	errEVM       = -40004 // error on evm execution
)

var (
	ethMultiCallCacheHit   = metrics.GetOrRegisterMeter("rpc/ethmulticall/cache/hit", nil)
	ethMultiCallCacheCount = metrics.GetOrRegisterMeter("rpc/ethmulticall/cache/count", nil)

	errCancelled = fmt.Errorf("execution aborted (timeout = %v)", singleCallTimeout)
)

func ethCallCacheKey(b Backend, blockHash common.Hash, to *common.Address, input []byte) string {
	var sb strings.Builder

	h := sha256.New()
	h.Write(input)
	bs := h.Sum(nil)

	sb.Grow(len(bs) + len(to.Bytes()) + len(blockHash))
	sb.Write(blockHash[:])
	sb.Write(bytes.ToLower(to.Bytes()))
	sb.Write(bs)

	return sb.String()
}

func doOneCall(ctx context.Context, b Backend, state *state.StateDB, header *types.Header, arg TransactionArgs, disableCache bool) (*callResult, error) {
	var err error
	var result = &callResult{}

	start := time.Now()

	if !disableCache {
		// try load result from cache
		cacheKey := ethCallCacheKey(b, header.Hash(), arg.To, arg.data())
		ethMultiCallCacheCount.Mark(1)
		if r, ok := b.GetCallCache(cacheKey); ok {
			ethMultiCallCacheHit.Mark(1)
			if res, ok := r.(*callResult); ok {
				res.FromCache = true
				return res, nil
			}
		}
		defer func() {
			// `err` here specifics to non-evm error. Evm internal error won't prevent
			// caching the result
			if err == nil {
				b.SetCallCache(cacheKey, result, int64(len(result.Result)))
			}
		}()
	}
	// make sure this will be called prior to the SetCallCache defer func on returning
	defer func() {
		result.TimeCost = time.Since(start).Seconds()
	}()

	// Get a new instance of the EVM.
	msg, err := arg.ToMessage(b.RPCGasCap(), header.BaseFee)
	if err != nil {
		result.Code = errParam
		result.Err = err.Error()
		return result, err
	}

	evm, _, _ := b.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true}) // never return error

	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()
	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	evmRet, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		result.Code = errConsensus
		result.Err = err.Error()
		return result, err
	}

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		err = errCancelled
		result.Code = errLogic
		result.Err = err.Error()
		return result, err
	}

	if evmRet.Err != nil {
		e := evmRet.Err
		if len(evmRet.Revert()) > 0 {
			e = newRevertError(evmRet)
		}
		result.Code = errEVM
		result.Err = e.Error()
		return result, e
	}

	result.Result = evmRet.Return()
	result.GasUsed = int64(evmRet.UsedGas)

	return result, nil
}

func (s *BlockChainAPI) MultiCall(ctx context.Context, args []TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, pfastFail, puseParallel, pdisableCache *bool, overrides *StateOverride) (resp *multiCallResp, err error) {

	// maximum calls check
	if len(args) > multiCallLimit {
		return nil, fmt.Errorf("calls exceed limit, expected: <%v, actual: %v", multiCallLimit, len(args))
	}

	setb := func(p *bool, d bool) bool {
		if p == nil {
			return d
		}
		return *p
	}

	fastFail := setb(pfastFail, true)
	useParallel := setb(puseParallel, true)
	disableCache := setb(pdisableCache, false)

	// check block & state
	state, header, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	if err := overrides.Apply(state); err != nil {
		return nil, err
	}
	blockTime := header.Time

	ret := make([]*callResult, len(args))
	stats := &multiCallStats{
		BlockNum:     header.Number.Int64(),
		BlockHash:    header.Hash(),
		BlockTime:    int64(blockTime),
		Success:      true,
		CacheEnabled: !disableCache,
	}

	ctx, cancel := context.WithTimeout(ctx, singleCallTimeout)
	defer cancel()

	if useParallel {
		// run in parallel
		var wg sync.WaitGroup
		for i, arg := range args {
			wg.Add(1)
			go func(i int, arg TransactionArgs) {
				defer wg.Done()

				// state is not reentrancy in concurrent scenarios, so use a copy
				state, _, _ := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
				r, _ := doOneCall(ctx, s.b, state, header, arg, disableCache)
				ret[i] = r
				if r.Err != "" {
					stats.Success = false
					if fastFail {
						cancel()
					}
					return
				}

				if r.FromCache {
					stats.CacheHitCount++
				} else {
					stats.GasUsed += r.GasUsed
				}
				stats.OriginGasUsed += r.GasUsed
			}(i, arg)
		}
		wg.Wait()

		return &multiCallResp{Results: ret, Stats: stats}, nil
	}

	// run in sequence
	failedOnce := false
	for i, arg := range args {
		if failedOnce {
			ret[i] = nil
			continue
		}

		r, _ := doOneCall(ctx, s.b, state, header, arg, disableCache)
		ret[i] = r
		if r.Err != "" {
			stats.Success = false
			if fastFail {
				failedOnce = true
			}
			continue
		}

		if r.FromCache {
			stats.CacheHitCount++
		} else {
			stats.GasUsed += r.GasUsed
		}
		stats.OriginGasUsed += r.GasUsed
	}

	return &multiCallResp{Results: ret, Stats: stats}, nil
}
