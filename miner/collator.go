package miner

import (
	"errors"
	//	"math"
	//"math/big"
	"os"
	"plugin"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/miner/collator"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/naoina/toml"
)

func LoadCollator(stack *node.Node, filepath string, configPath string) (collator.Collator, error) {
	p, err := plugin.Open(filepath)
	if err != nil {
		return nil, err
	}

	v, err := p.Lookup("PluginConstructor")
	if err != nil {
		return nil, errors.New("symbol 'APIExport' not found")
	}

	pluginConstructor, ok := v.(func(config *map[string]interface{}, stack *node.Node) (collator.Collator, error))
	if !ok {
		return nil, errors.New("invalid type signature for collator plugin constructor: expected 'func(config *map[string]interface{}, stack *node.Node) (collator.Collator, error)'")
	}

	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := make(map[string]interface{})
	if err := toml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	collator, err := pluginConstructor(&config, stack)
	if err != nil {
		return nil, err
	}

	return collator, nil
}

// collatorBlockState contains an under-construction pending block for sealing.
// a work-cycle (a round of pending block construction/sealing on top of a given canon chain head)
// is composed of an environment and one or more collatorBlockStates as a collator may choose
// to construct and seal many blocks during a given work cycle.
type collatorBlockState struct {
	state     *state.StateDB
	txs       []*types.Transaction
	receipts  []*types.Receipt
	env       *environment
	committed bool
	tcount    int           // tx count in cycle
	gasPool   *core.GasPool // available gas used to pack transactions
	logs      []*types.Log
	header    *types.Header
}

// minerState allows a collator implementation to read the current chain config and whether the miner is currently sealing.
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

func (bs *collatorBlockState) Commit() bool {
	return bs.env.worker.commitBlockState(bs)
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

func (bs *collatorBlockState) Copy() collator.BlockState {
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

	return &cpy
}

func (bs *collatorBlockState) commitTransaction(tx *types.Transaction) (*types.Receipt, error) {
	snap := bs.state.Snapshot()
	bs.state.Prepare(tx.Hash(), bs.tcount)
	receipt, err := core.ApplyTransaction(bs.env.worker.chainConfig, bs.env.worker.chain, &bs.env.coinbase, bs.gasPool, bs.state, bs.header, tx, &bs.header.GasUsed, *bs.env.worker.chain.GetVMConfig())
	if err != nil {
		bs.state.RevertToSnapshot(snap)
		return nil, err
	}
	bs.txs = append(bs.txs, tx)
	bs.receipts = append(bs.receipts, receipt)
	bs.logs = append(bs.logs, receipt.Logs...)
	bs.tcount++

	return receipt, nil
}

func (bs *collatorBlockState) AddTransaction(tx *types.Transaction) (*types.Receipt, error) {
	if bs.gasPool.Gas() < params.TxGas {
		return nil, collator.ErrGasLimitReached
	}

	// Check whether the tx is replay protected. If we're not in the EIP155 hf
	// phase, start ignoring the sender until we do.
	if tx.Protected() && !bs.env.worker.chainConfig.IsEIP155(bs.header.Number) {
		return nil, collator.ErrTxTypeNotSupported
	}

	// TODO can this error also be returned by commitTransaction below?
	_, err := tx.EffectiveGasTip(bs.header.BaseFee)
	if err != nil {
		return nil, collator.ErrGasFeeCapTooLow
	}

	receipt, err := bs.commitTransaction(tx)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			// this should never be reached.
			// should be caught above
			return nil, collator.ErrGasLimitReached
		case errors.Is(err, core.ErrNonceTooLow):
			return nil, collator.ErrNonceTooLow
		case errors.Is(err, core.ErrNonceTooHigh):
			return nil, collator.ErrNonceTooHigh
		case errors.Is(err, core.ErrTxTypeNotSupported):
			// TODO check that this unspported tx type is the same as the one caught above
			return nil, collator.ErrTxTypeNotSupported
		default:
			return nil, collator.ErrStrange
		}
	}

	receiptCpy := *receipt
	return &receiptCpy, nil
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
