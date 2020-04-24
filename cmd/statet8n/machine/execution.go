// Copyright 2012 The go-ethereum Authors
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

package machine

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type Prestate struct {
	Env stEnv             `json:"env"`
	Pre core.GenesisAlloc `json:"pre"`
}

// ExecutionResult contains the execution status after running a state test, any
// error that might have occurred and a dump of the final state if requested.
type ExecutionResult struct {
	StateRoot   common.Hash    `json:"postState"`
	TxHash      common.Hash    `json:"txHash"`
	ReceiptRoot common.Hash    `json:"receiptRoot"`
	Receipts    types.Receipts `json:"receipts,omitempty"`
	Rejected    []int          `json:"rejected,omitempty"`
}

//go:generate gencodec -type stEnv -field-override stEnvMarshaling -out gen_stenv.go
type stEnv struct {
	Coinbase   common.Address `json:"currentCoinbase"   gencodec:"required"`
	Difficulty *big.Int       `json:"currentDifficulty" gencodec:"required"`
	GasLimit   uint64         `json:"currentGasLimit"   gencodec:"required"`
	Number     uint64         `json:"currentNumber"     gencodec:"required"`
	Timestamp  uint64         `json:"currentTimestamp"  gencodec:"required"`
}

type stEnvMarshaling struct {
	Coinbase   common.UnprefixedAddress
	Difficulty *math.HexOrDecimal256
	GasLimit   math.HexOrDecimal64
	Number     math.HexOrDecimal64
	Timestamp  math.HexOrDecimal64
}

// Apply applies a set of transactions to a pre-state
func (pre *Prestate) Apply(vmConfig vm.Config, chainConfig *params.ChainConfig, txs types.Transactions, miningReward int64) (*state.StateDB, *ExecutionResult, error) {

	getHash := func(num uint64) common.Hash {
		panic(fmt.Sprintf("getHash(%d) invoked", num))
	}
	statedb := MakePreState(rawdb.NewMemoryDatabase(), pre.Pre)
	// Configure a signer with chainid 99
	signer := types.NewEIP155Signer(chainConfig.ChainID)

	//block := t.genesis(config).ToBlock(nil)
	gaspool := new(core.GasPool)
	gaspool.AddGas(pre.Env.GasLimit)

	blockHash := common.Hash{0x13, 0x37}

	vmContext := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    pre.Env.Coinbase,
		BlockNumber: new(big.Int).SetUint64(pre.Env.Number),
		Time:        new(big.Int).SetUint64(pre.Env.Timestamp),
		Difficulty:  pre.Env.Difficulty,
		GasLimit:    pre.Env.GasLimit,
		// This will cause a panic
		GetHash: getHash,
		// GasPrice and Origin needs to be set per transaction
	}
	var rejected []int
	gasUsed := uint64(0)
	var receipts types.Receipts
	for i, tx := range txs {
		msg, err := tx.AsMessage(signer)
		if err != nil {
			log.Info("rejected tx", "index", i, "hash", tx.Hash(), "error", err)
			rejected = append(rejected, i)
			continue
		}
		statedb.Prepare(tx.Hash(), blockHash, i)
		vmContext.GasPrice = msg.GasPrice()
		vmContext.Origin = msg.From()

		evm := vm.NewEVM(vmContext, statedb, chainConfig, vmConfig)
		snapshot := statedb.Snapshot()
		// (ret []byte, usedGas uint64, failed bool, err error)
		msgResult, err := core.ApplyMessage(evm, msg, gaspool)
		if err != nil {
			statedb.RevertToSnapshot(snapshot)
			log.Info("rejected tx", "index", i, "hash", tx.Hash(), "from", msg.From(), "error", err)
			rejected = append(rejected, i)
			continue
		}
		gasUsed += msgResult.UsedGas
		// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
		var root []byte
		receipt := types.NewReceipt(root, msgResult.Failed(), gasUsed)
		receipt.TxHash = tx.Hash()
		receipt.GasUsed = msgResult.UsedGas
		// if the transaction created a contract, store the creation address in the receipt.
		if msg.To() == nil {
			receipt.ContractAddress = crypto.CreateAddress(evm.Context.Origin, tx.Nonce())
		}
		// Set the receipt logs and create a bloom for filtering
		receipt.Logs = statedb.GetLogs(tx.Hash())
		receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
		receipts = append(receipts, receipt)
	}
	statedb.IntermediateRoot(chainConfig.IsEIP158(vmContext.BlockNumber))
	// Add mining reward?
	if miningReward > 0 {
		// Add mining reward. The mining reward may be `0`, which only makes a difference in the cases
		// where
		// - the coinbase suicided, or
		// - there are only 'bad' transactions, which aren't executed. In those cases,
		//   the coinbase gets no txfee, so isn't created, and thus needs to be touched
		statedb.AddBalance(pre.Env.Coinbase, big.NewInt(miningReward))
	}
	// Commit block
	root, err := statedb.Commit(chainConfig.IsEIP158(vmContext.BlockNumber))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not commit state: %v", err)
		return nil, nil, NewError(ErrorEVM, fmt.Errorf("could not commit state: %v", err))
	}
	execRs := &ExecutionResult{
		StateRoot:   root,
		TxHash:      types.DeriveSha(txs),
		ReceiptRoot: types.DeriveSha(receipts),
		Receipts:    receipts,
		Rejected:    rejected,
	}
	return statedb, execRs, nil
}

func MakePreState(db ethdb.Database, accounts core.GenesisAlloc) *state.StateDB {
	sdb := state.NewDatabase(db)
	statedb, _ := state.New(common.Hash{}, sdb, nil)
	for addr, a := range accounts {
		statedb.SetCode(addr, a.Code)
		statedb.SetNonce(addr, a.Nonce)
		statedb.SetBalance(addr, a.Balance)
		for k, v := range a.Storage {
			statedb.SetState(addr, k, v)
		}
	}
	// Commit and re-open to start with a clean state.
	root, _ := statedb.Commit(false)
	statedb, _ = state.New(root, sdb, nil)
	return statedb
}
