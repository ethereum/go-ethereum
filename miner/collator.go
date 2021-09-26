package miner

import (
	"errors"
	//	"math"
	//"math/big"
	"context"
	"os"
	"plugin"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/naoina/toml"
)

type CollatorAPI interface {
	Version() string
	Service() interface{}
}

// Pool is an interface to the transaction pool
type Pool interface {
	Pending(bool) (map[common.Address]types.Transactions, error)
	Locals() []common.Address
}

func LoadCollator(filepath string, configPath string) (Collator, CollatorAPI, error) {
	p, err := plugin.Open(filepath)
	if err != nil {
		return nil, nil, err
	}

	v, err := p.Lookup("PluginConstructor")
	if err != nil {
		return nil, nil, errors.New("symbol 'APIExport' not found")
	}

	pluginConstructor, ok := v.(func(config *map[string]interface{}) (Collator, CollatorAPI, error))
	if !ok {
		return nil, nil, errors.New("expected symbol 'API' to be of type 'CollatorAPI")
	}

	f, err := os.Open(configPath)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	config := make(map[string]interface{})
	if err := toml.NewDecoder(f).Decode(&config); err != nil {
		return nil, nil, err
	}

	collator, collatorAPI, err := pluginConstructor(&config)
	if err != nil {
		return nil, nil, err
	}

	return collator, collatorAPI, nil
}

/*
	BlockState represents an under-construction block.  An instance of
	BlockState is passed to CollateBlock where it can be filled with transactions
	via BlockState.AddTransaction() and submitted for sealing via
	BlockState.Commit().
	Operations on a single BlockState instance are not threadsafe.  However,
	instances can be copied with BlockState.Copy().
*/
type BlockState interface {
	/*
		adds a single transaction to the blockState.  Returned errors include ..,..,..
		which signify that the transaction was invalid for the current EVM/chain state.
		ErrRecommit signals that the recommit timer has elapsed.
		ErrNewHead signals that the client has received a new canonical chain head.
		All subsequent calls to AddTransaction fail if either newHead or the recommit timer
		have occured.
		If the recommit interval has elapsed, the BlockState can still be committed to the sealer.
	*/
	AddTransactions(tx types.Transactions) (error, types.Receipts)

	/*
		removes a number of transactions from the block resetting the state to what
		it was before the transactions were added.  If count is greater than the number
		of transactions in the block,  returns
	*/
	RevertTransactions(count uint) error

	/*
		returns true if the Block has been made the current sealing block.
		returns false if the newHead interrupt has been triggered.
		can also return false if the BlockState is no longer valid (the call to CollateBlock
		which the original instance was passed has returned).
	*/
	Commit() bool
	Copy() BlockState
	State() vm.StateReader
	Signer() types.Signer
	Header() *types.Header
	/*
		the account which will receive the block reward.
	*/
	Etherbase() common.Address
	GasPool() core.GasPool
}

type collatorBlockState struct {
	state     *state.StateDB
	txs       []*types.Transaction
	receipts  []*types.Receipt
	env       *environment
	committed bool
	tcount    int           // tx count in cycle
	gasPool   *core.GasPool // available gas used to pack transactions
	logs      []*types.Log
	snapshots []int
	header    *types.Header
}

type MinerState interface {
	IsRunning() bool
	ChainConfig() *params.ChainConfig
}

type minerState struct {
	// keep a copy of ChainConfig here, if collator chooses (erroneously) to modify chainConfig, the chainConfig used by the miner doesn't get changed
	chainConfig *params.ChainConfig
	worker      *worker
}

func (m *minerState) ChainConfig() *params.ChainConfig {
	return m.chainConfig
}

func (m *minerState) IsRunning() bool {
	return m.worker.isRunning()
}

type BlockCollatorWork struct {
	Ctx   context.Context
	Block BlockState
}

type Collator interface {
	CollateBlocks(miner MinerState, pool Pool, blockCh <-chan BlockCollatorWork, exitCh <-chan struct{})
	CollateBlock(bs BlockState, pool Pool)
}

var (
	ErrAlreadyCommitted = errors.New("can't mutate BlockState after calling Commit()")

	// errors which indicate that a given transaction cannot be
	// added at a given block or chain configuration.
	ErrGasLimitReached    = errors.New("gas limit reached")
	ErrNonceTooLow        = errors.New("tx nonce too low")
	ErrNonceTooHigh       = errors.New("tx nonce too high")
	ErrTxTypeNotSupported = errors.New("tx type not supported")
	ErrGasFeeCapTooLow    = errors.New("gas fee cap too low")
	ErrZeroTxs            = errors.New("zero transactions")
	ErrTooManyTxs         = errors.New("applying txs to block would go over the block gas limit")
	// error which encompasses all other reasons a given transaction
	// could not be added to the block.
	ErrStrange = errors.New("strange error")
)

func (bs *collatorBlockState) Commit() bool {
	bs.env.worker.curEnvMu.Lock()
	defer bs.env.worker.curEnvMu.Unlock()

	if bs.env.cycleCtx != nil {
		select {
		case <-bs.env.cycleCtx.Done():
			return false
		default:
		}
	}

	bs.env.current = bs
	// TODO apply FinalizeAndAssemble with our state, then copy and send it to sealer?
	// that way the post-block-processing state could be inspected from a BlockState
	bs.env.worker.commit(bs.env.copy(), nil, true, time.Now())

	return true
}

func copyLogs(logs []*types.Log) []*types.Log {
	result := make([]*types.Log, len(logs))
	for _, l := range logs {
		logCopy := types.Log{
			Address:     l.Address,
			BlockNumber: l.BlockNumber,
			TxHash:      l.TxHash,
			TxIndex:     l.TxIndex,
			Index:       l.Index,
			Removed:     l.Removed,
		}
		for _, t := range l.Topics {
			logCopy.Topics = append(logCopy.Topics, t)
		}
		logCopy.Data = make([]byte, len(l.Data))
		copy(logCopy.Data[:], l.Data[:])

		result = append(result, &logCopy)
	}

	return result
}

// copyReceipts makes a deep copy of the given receipts.
func copyReceipts(receipts []*types.Receipt) []*types.Receipt {
	result := make([]*types.Receipt, len(receipts))
	for i, l := range receipts {
		cpy := *l
		result[i] = &cpy
	}
	return result
}

func (bs *collatorBlockState) Copy() BlockState {
	return bs.copy()
}

func (bs *collatorBlockState) copy() *collatorBlockState {
	cpy := collatorBlockState{
		env:      bs.env,
		state:    bs.state.Copy(),
		tcount:   bs.tcount,
		logs:     copyLogs(bs.logs),
		receipts: copyReceipts(bs.receipts),
		header:   types.CopyHeader(bs.header),
	}

	if bs.gasPool != nil {
		cpy.gasPool = new(core.GasPool)
		*cpy.gasPool = *bs.gasPool
	}
	cpy.txs = make([]*types.Transaction, len(bs.txs))
	copy(cpy.txs, bs.txs)
	cpy.snapshots = make([]int, len(bs.snapshots))
	copy(cpy.snapshots, bs.snapshots)

	return &cpy
}

func (bs *collatorBlockState) commitTransaction(tx *types.Transaction) ([]*types.Log, error) {
	snap := bs.state.Snapshot()

	receipt, err := core.ApplyTransaction(bs.env.worker.chainConfig, bs.env.worker.chain, &bs.env.coinbase, bs.gasPool, bs.state, bs.header, tx, &bs.header.GasUsed, *bs.env.worker.chain.GetVMConfig())
	if err != nil {
		bs.state.RevertToSnapshot(snap)
		return nil, err
	}
	bs.txs = append(bs.txs, tx)
	bs.receipts = append(bs.receipts, receipt)

	return receipt.Logs, nil
}

func (bs *collatorBlockState) AddTransactions(txs types.Transactions) (error, types.Receipts) {
	tcount := 0
	var retErr error = nil

	if len(txs) == 0 {
		return ErrZeroTxs, nil
	}

	for _, tx := range txs {
		if bs.gasPool.Gas() < params.TxGas {
			retErr = ErrGasLimitReached
			break
		}

		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !bs.env.worker.chainConfig.IsEIP155(bs.header.Number) {
			retErr = ErrTxTypeNotSupported
			break
		}

		// TODO can this error also be returned by commitTransaction below?
		_, err := tx.EffectiveGasTip(bs.header.BaseFee)
		if err != nil {
			retErr = ErrGasFeeCapTooLow
			break
		}

		snapshot := bs.state.Snapshot()
		bs.snapshots = append(bs.snapshots, snapshot)

		bs.state.Prepare(tx.Hash(), bs.tcount+tcount)
		receipt, err := core.ApplyTransaction(bs.env.worker.chainConfig, bs.env.worker.chain, &bs.env.coinbase, bs.gasPool, bs.state, bs.header, tx, &bs.header.GasUsed, *bs.env.worker.chain.GetVMConfig())
		if err != nil {
			switch {
			case errors.Is(err, core.ErrGasLimitReached):
				// this should never be reached.
				// should be caught above
				retErr = ErrGasLimitReached
			case errors.Is(err, core.ErrNonceTooLow):
				retErr = ErrNonceTooLow
			case errors.Is(err, core.ErrNonceTooHigh):
				retErr = ErrNonceTooHigh
			case errors.Is(err, core.ErrTxTypeNotSupported):
				// TODO check that this unspported tx type is the same as the one caught above
				retErr = ErrTxTypeNotSupported
			default:
				retErr = ErrStrange
			}

			break
		} else {
			bs.logs = append(bs.logs, receipt.Logs...)
			bs.txs = append(bs.txs, tx)
			bs.receipts = append(bs.receipts, receipt)
			tcount++
		}
	}

	var retReceipts []*types.Receipt

	if retErr != nil {
		bs.logs = bs.logs[:len(bs.logs)-tcount]
		bs.state.RevertToSnapshot(bs.snapshots[len(bs.snapshots)-(tcount+1)])

		bs.snapshots = bs.snapshots[:len(bs.snapshots)-(tcount+1)]
		bs.txs = bs.txs[:len(bs.txs)-(tcount+1)]
		bs.receipts = bs.receipts[:len(bs.receipts)-(tcount+1)]

		return retErr, nil
	} else {
		retReceipts = bs.receipts[bs.tcount:]
		bs.tcount += tcount
	}
	return nil, retReceipts
}

func (bs *collatorBlockState) RevertTransactions(count uint) error {
	if int(count) > len(bs.snapshots) {
		return ErrTooManyTxs
	} else if count == 0 {
		return ErrZeroTxs
	}
	bs.state.RevertToSnapshot(bs.snapshots[len(bs.snapshots)-int(count)])
	bs.snapshots = bs.snapshots[:len(bs.snapshots)-int(count)]
	return nil
}

func (bs *collatorBlockState) State() vm.StateReader {
	return bs.state
}

func (bs *collatorBlockState) Signer() types.Signer {
	return bs.env.signer
}

func (bs *collatorBlockState) Etherbase() common.Address {
	return bs.env.coinbase
}

func (bs *collatorBlockState) GasPool() core.GasPool {
	return *bs.gasPool
}

func (bs *collatorBlockState) discard() {
	bs.state.StopPrefetcher()
}

func (bs *collatorBlockState) Header() *types.Header {
	return types.CopyHeader(bs.header)
}
