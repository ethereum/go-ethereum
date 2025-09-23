// Copyright 2020 The go-ethereum Authors
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

package t8ntool

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

type Prestate struct {
	Env stEnv              `json:"env"`
	Pre types.GenesisAlloc `json:"pre"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutionResult -field-override executionResultMarshaling -out gen_execresult.go

// ExecutionResult contains the execution status after running a state test, any
// error that might have occurred and a dump of the final state if requested.
type ExecutionResult struct {
	StateRoot            common.Hash           `json:"stateRoot"`
	TxRoot               common.Hash           `json:"txRoot"`
	ReceiptRoot          common.Hash           `json:"receiptsRoot"`
	LogsHash             common.Hash           `json:"logsHash"`
	Bloom                types.Bloom           `json:"logsBloom"        gencodec:"required"`
	Receipts             types.Receipts        `json:"receipts"`
	Rejected             []*rejectedTx         `json:"rejected,omitempty"`
	Difficulty           *math.HexOrDecimal256 `json:"currentDifficulty" gencodec:"required"`
	GasUsed              math.HexOrDecimal64   `json:"gasUsed"`
	BaseFee              *math.HexOrDecimal256 `json:"currentBaseFee,omitempty"`
	WithdrawalsRoot      *common.Hash          `json:"withdrawalsRoot,omitempty"`
	CurrentExcessBlobGas *math.HexOrDecimal64  `json:"currentExcessBlobGas,omitempty"`
	CurrentBlobGasUsed   *math.HexOrDecimal64  `json:"blobGasUsed,omitempty"`
	RequestsHash         *common.Hash          `json:"requestsHash,omitempty"`
	Requests             [][]byte              `json:"requests"`
}

type executionResultMarshaling struct {
	Requests []hexutil.Bytes `json:"requests"`
}

type ommer struct {
	Delta   uint64         `json:"delta"`
	Address common.Address `json:"address"`
}

//go:generate go run github.com/fjl/gencodec -type stEnv -field-override stEnvMarshaling -out gen_stenv.go
type stEnv struct {
	Coinbase              common.Address                      `json:"currentCoinbase"   gencodec:"required"`
	Difficulty            *big.Int                            `json:"currentDifficulty"`
	Random                *big.Int                            `json:"currentRandom"`
	ParentDifficulty      *big.Int                            `json:"parentDifficulty"`
	ParentBaseFee         *big.Int                            `json:"parentBaseFee,omitempty"`
	ParentGasUsed         uint64                              `json:"parentGasUsed,omitempty"`
	ParentGasLimit        uint64                              `json:"parentGasLimit,omitempty"`
	GasLimit              uint64                              `json:"currentGasLimit"   gencodec:"required"`
	Number                uint64                              `json:"currentNumber"     gencodec:"required"`
	Timestamp             uint64                              `json:"currentTimestamp"  gencodec:"required"`
	ParentTimestamp       uint64                              `json:"parentTimestamp,omitempty"`
	BlockHashes           map[math.HexOrDecimal64]common.Hash `json:"blockHashes,omitempty"`
	Ommers                []ommer                             `json:"ommers,omitempty"`
	Withdrawals           []*types.Withdrawal                 `json:"withdrawals,omitempty"`
	BaseFee               *big.Int                            `json:"currentBaseFee,omitempty"`
	ParentUncleHash       common.Hash                         `json:"parentUncleHash"`
	ExcessBlobGas         *uint64                             `json:"currentExcessBlobGas,omitempty"`
	ParentExcessBlobGas   *uint64                             `json:"parentExcessBlobGas,omitempty"`
	ParentBlobGasUsed     *uint64                             `json:"parentBlobGasUsed,omitempty"`
	ParentBeaconBlockRoot *common.Hash                        `json:"parentBeaconBlockRoot"`
}

type stEnvMarshaling struct {
	Coinbase            common.UnprefixedAddress
	Difficulty          *math.HexOrDecimal256
	Random              *math.HexOrDecimal256
	ParentDifficulty    *math.HexOrDecimal256
	ParentBaseFee       *math.HexOrDecimal256
	ParentGasUsed       math.HexOrDecimal64
	ParentGasLimit      math.HexOrDecimal64
	GasLimit            math.HexOrDecimal64
	Number              math.HexOrDecimal64
	Timestamp           math.HexOrDecimal64
	ParentTimestamp     math.HexOrDecimal64
	BaseFee             *math.HexOrDecimal256
	ExcessBlobGas       *math.HexOrDecimal64
	ParentExcessBlobGas *math.HexOrDecimal64
	ParentBlobGasUsed   *math.HexOrDecimal64
}

type rejectedTx struct {
	Index int    `json:"index"`
	Err   string `json:"error"`
}

// Apply applies a set of transactions to a pre-state
func (pre *Prestate) Apply(vmConfig vm.Config, chainConfig *params.ChainConfig, txIt txIterator, miningReward int64) (*state.StateDB, *ExecutionResult, []byte, error) {
	// Capture errors for BLOCKHASH operation, if we haven't been supplied the
	// required blockhashes
	var hashError error
	getHash := func(num uint64) common.Hash {
		if pre.Env.BlockHashes == nil {
			hashError = fmt.Errorf("getHash(%d) invoked, no blockhashes provided", num)
			return common.Hash{}
		}
		h, ok := pre.Env.BlockHashes[math.HexOrDecimal64(num)]
		if !ok {
			hashError = fmt.Errorf("getHash(%d) invoked, blockhash for that block not provided", num)
		}
		return h
	}
	var (
		statedb     = MakePreState(rawdb.NewMemoryDatabase(), pre.Pre)
		signer      = types.MakeSigner(chainConfig, new(big.Int).SetUint64(pre.Env.Number), pre.Env.Timestamp)
		gaspool     = new(core.GasPool)
		blockHash   = common.Hash{0x13, 0x37}
		rejectedTxs []*rejectedTx
		includedTxs types.Transactions
		gasUsed     = uint64(0)
		blobGasUsed = uint64(0)
		receipts    = make(types.Receipts, 0)
	)
	gaspool.AddGas(pre.Env.GasLimit)
	vmContext := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    pre.Env.Coinbase,
		BlockNumber: new(big.Int).SetUint64(pre.Env.Number),
		Time:        pre.Env.Timestamp,
		Difficulty:  pre.Env.Difficulty,
		GasLimit:    pre.Env.GasLimit,
		GetHash:     getHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	if pre.Env.BaseFee != nil {
		vmContext.BaseFee = new(big.Int).Set(pre.Env.BaseFee)
	}
	// If random is defined, add it to the vmContext.
	if pre.Env.Random != nil {
		rnd := common.BigToHash(pre.Env.Random)
		vmContext.Random = &rnd
	}
	// Calculate the BlobBaseFee
	var excessBlobGas uint64
	if pre.Env.ExcessBlobGas != nil {
		excessBlobGas = *pre.Env.ExcessBlobGas
		header := &types.Header{
			Time:          pre.Env.Timestamp,
			ExcessBlobGas: pre.Env.ExcessBlobGas,
		}
		vmContext.BlobBaseFee = eip4844.CalcBlobFee(chainConfig, header)
	} else {
		// If it is not explicitly defined, but we have the parent values, we try
		// to calculate it ourselves.
		parentExcessBlobGas := pre.Env.ParentExcessBlobGas
		parentBlobGasUsed := pre.Env.ParentBlobGasUsed
		if parentExcessBlobGas != nil && parentBlobGasUsed != nil {
			parent := &types.Header{
				Time:          pre.Env.ParentTimestamp,
				ExcessBlobGas: pre.Env.ParentExcessBlobGas,
				BlobGasUsed:   pre.Env.ParentBlobGasUsed,
				BaseFee:       pre.Env.ParentBaseFee,
			}
			header := &types.Header{
				Time:          pre.Env.Timestamp,
				ExcessBlobGas: &excessBlobGas,
			}
			excessBlobGas = eip4844.CalcExcessBlobGas(chainConfig, parent, header.Time)
			vmContext.BlobBaseFee = eip4844.CalcBlobFee(chainConfig, header)
		}
	}
	// If DAO is supported/enabled, we need to handle it here. In geth 'proper', it's
	// done in StateProcessor.Process(block, ...), right before transactions are applied.
	if chainConfig.DAOForkSupport &&
		chainConfig.DAOForkBlock != nil &&
		chainConfig.DAOForkBlock.Cmp(new(big.Int).SetUint64(pre.Env.Number)) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	evm := vm.NewEVM(vmContext, statedb, chainConfig, vmConfig)
	if beaconRoot := pre.Env.ParentBeaconBlockRoot; beaconRoot != nil {
		core.ProcessBeaconBlockRoot(*beaconRoot, evm)
	}
	if pre.Env.BlockHashes != nil && chainConfig.IsPrague(new(big.Int).SetUint64(pre.Env.Number), pre.Env.Timestamp) {
		var (
			prevNumber = pre.Env.Number - 1
			prevHash   = pre.Env.BlockHashes[math.HexOrDecimal64(prevNumber)]
		)
		core.ProcessParentBlockHash(prevHash, evm)
	}
	for i := 0; txIt.Next(); i++ {
		tx, err := txIt.Tx()
		if err != nil {
			log.Warn("rejected tx", "index", i, "error", err)
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
			continue
		}
		if tx.Type() == types.BlobTxType && vmContext.BlobBaseFee == nil {
			errMsg := "blob tx used but field env.ExcessBlobGas missing"
			log.Warn("rejected tx", "index", i, "hash", tx.Hash(), "error", errMsg)
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, errMsg})
			continue
		}
		msg, err := core.TransactionToMessage(tx, signer, pre.Env.BaseFee)
		if err != nil {
			log.Warn("rejected tx", "index", i, "hash", tx.Hash(), "error", err)
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
			continue
		}
		txBlobGas := uint64(0)
		if tx.Type() == types.BlobTxType {
			txBlobGas = uint64(params.BlobTxBlobGasPerBlob * len(tx.BlobHashes()))
			max := eip4844.MaxBlobGasPerBlock(chainConfig, pre.Env.Timestamp)
			if used := blobGasUsed + txBlobGas; used > max {
				err := fmt.Errorf("blob gas (%d) would exceed maximum allowance %d", used, max)
				log.Warn("rejected tx", "index", i, "err", err)
				rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
				continue
			}
		}
		statedb.SetTxContext(tx.Hash(), len(receipts))
		var (
			snapshot = statedb.Snapshot()
			prevGas  = gaspool.Gas()
		)
		receipt, err := core.ApplyTransactionWithEVM(msg, gaspool, statedb, vmContext.BlockNumber, blockHash, pre.Env.Timestamp, tx, &gasUsed, evm)
		if err != nil {
			statedb.RevertToSnapshot(snapshot)
			log.Info("rejected tx", "index", i, "hash", tx.Hash(), "from", msg.From, "error", err)
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
			gaspool.SetGas(prevGas)
			continue
		}
		includedTxs = append(includedTxs, tx)
		if hashError != nil {
			return nil, nil, nil, NewError(ErrorMissingBlockhash, hashError)
		}
		blobGasUsed += txBlobGas
		receipts = append(receipts, receipt)
	}

	statedb.IntermediateRoot(chainConfig.IsEIP158(vmContext.BlockNumber))

	// Add mining reward? (-1 means rewards are disabled)
	if miningReward >= 0 {
		// Add mining reward. The mining reward may be `0`, which only makes a difference in the cases
		// where
		// - the coinbase self-destructed, or
		// - there are only 'bad' transactions, which aren't executed. In those cases,
		//   the coinbase gets no txfee, so isn't created, and thus needs to be touched
		var (
			blockReward = big.NewInt(miningReward)
			minerReward = new(big.Int).Set(blockReward)
			perOmmer    = new(big.Int).Rsh(blockReward, 5)
		)
		for _, ommer := range pre.Env.Ommers {
			// Add 1/32th for each ommer included
			minerReward.Add(minerReward, perOmmer)
			// Add (8-delta)/8
			reward := big.NewInt(8)
			reward.Sub(reward, new(big.Int).SetUint64(ommer.Delta))
			reward.Mul(reward, blockReward)
			reward.Rsh(reward, 3)
			statedb.AddBalance(ommer.Address, uint256.MustFromBig(reward), tracing.BalanceIncreaseRewardMineUncle)
		}
		statedb.AddBalance(pre.Env.Coinbase, uint256.MustFromBig(minerReward), tracing.BalanceIncreaseRewardMineBlock)
	}
	// Apply withdrawals
	for _, w := range pre.Env.Withdrawals {
		// Amount is in gwei, turn into wei
		amount := new(big.Int).Mul(new(big.Int).SetUint64(w.Amount), big.NewInt(params.GWei))
		statedb.AddBalance(w.Address, uint256.MustFromBig(amount), tracing.BalanceIncreaseWithdrawal)
	}

	// Gather the execution-layer triggered requests.
	var requests [][]byte
	if chainConfig.IsPrague(vmContext.BlockNumber, vmContext.Time) {
		requests = [][]byte{}
		// EIP-6110
		var allLogs []*types.Log
		for _, receipt := range receipts {
			allLogs = append(allLogs, receipt.Logs...)
		}
		if err := core.ParseDepositLogs(&requests, allLogs, chainConfig); err != nil {
			return nil, nil, nil, NewError(ErrorEVM, fmt.Errorf("could not parse requests logs: %v", err))
		}
		// EIP-7002
		if err := core.ProcessWithdrawalQueue(&requests, evm); err != nil {
			return nil, nil, nil, NewError(ErrorEVM, fmt.Errorf("could not process withdrawal requests: %v", err))
		}
		// EIP-7251
		if err := core.ProcessConsolidationQueue(&requests, evm); err != nil {
			return nil, nil, nil, NewError(ErrorEVM, fmt.Errorf("could not process consolidation requests: %v", err))
		}
	}

	// Commit block
	root, err := statedb.Commit(vmContext.BlockNumber.Uint64(), chainConfig.IsEIP158(vmContext.BlockNumber), chainConfig.IsCancun(vmContext.BlockNumber, vmContext.Time))
	if err != nil {
		return nil, nil, nil, NewError(ErrorEVM, fmt.Errorf("could not commit state: %v", err))
	}
	execRs := &ExecutionResult{
		StateRoot:   root,
		TxRoot:      types.DeriveSha(includedTxs, trie.NewStackTrie(nil)),
		ReceiptRoot: types.DeriveSha(receipts, trie.NewStackTrie(nil)),
		Bloom:       types.MergeBloom(receipts),
		LogsHash:    rlpHash(statedb.Logs()),
		Receipts:    receipts,
		Rejected:    rejectedTxs,
		Difficulty:  (*math.HexOrDecimal256)(vmContext.Difficulty),
		GasUsed:     (math.HexOrDecimal64)(gasUsed),
		BaseFee:     (*math.HexOrDecimal256)(vmContext.BaseFee),
	}
	if pre.Env.Withdrawals != nil {
		h := types.DeriveSha(types.Withdrawals(pre.Env.Withdrawals), trie.NewStackTrie(nil))
		execRs.WithdrawalsRoot = &h
	}
	if vmContext.BlobBaseFee != nil {
		execRs.CurrentExcessBlobGas = (*math.HexOrDecimal64)(&excessBlobGas)
		execRs.CurrentBlobGasUsed = (*math.HexOrDecimal64)(&blobGasUsed)
	}
	if requests != nil {
		// Set requestsHash on block.
		h := types.CalcRequestsHash(requests)
		execRs.RequestsHash = &h
		for i := range requests {
			// remove prefix
			requests[i] = requests[i][1:]
		}
		execRs.Requests = requests
	}

	// Re-create statedb instance with new root upon the updated database
	// for accessing latest states.
	statedb, err = state.New(root, statedb.Database())
	if err != nil {
		return nil, nil, nil, NewError(ErrorEVM, fmt.Errorf("could not reopen state: %v", err))
	}
	body, _ := rlp.EncodeToBytes(includedTxs)
	return statedb, execRs, body, nil
}

func MakePreState(db ethdb.Database, accounts types.GenesisAlloc) *state.StateDB {
	tdb := triedb.NewDatabase(db, &triedb.Config{Preimages: true})
	sdb := state.NewDatabase(tdb, nil)
	statedb, _ := state.New(types.EmptyRootHash, sdb)
	for addr, a := range accounts {
		statedb.SetCode(addr, a.Code, tracing.CodeChangeUnspecified)
		statedb.SetNonce(addr, a.Nonce, tracing.NonceChangeGenesis)
		statedb.SetBalance(addr, uint256.MustFromBig(a.Balance), tracing.BalanceIncreaseGenesisBalance)
		for k, v := range a.Storage {
			statedb.SetState(addr, k, v)
		}
	}
	// Commit and re-open to start with a clean state.
	root, _ := statedb.Commit(0, false, false)
	statedb, _ = state.New(root, sdb)
	return statedb
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewLegacyKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

// calcDifficulty is based on ethash.CalcDifficulty. This method is used in case
// the caller does not provide an explicit difficulty, but instead provides only
// parent timestamp + difficulty.
// Note: this method only works for ethash engine.
func calcDifficulty(config *params.ChainConfig, number, currentTime, parentTime uint64,
	parentDifficulty *big.Int, parentUncleHash common.Hash) *big.Int {
	uncleHash := parentUncleHash
	if uncleHash == (common.Hash{}) {
		uncleHash = types.EmptyUncleHash
	}
	parent := &types.Header{
		ParentHash: common.Hash{},
		UncleHash:  uncleHash,
		Difficulty: parentDifficulty,
		Number:     new(big.Int).SetUint64(number - 1),
		Time:       parentTime,
	}
	return ethash.CalcDifficulty(config, currentTime, parent)
}
