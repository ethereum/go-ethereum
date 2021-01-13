// Copyright 2020 The go-ethereum Authors
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

package eth

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	chainParams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

type Eth2API struct {
	eth  *Ethereum
	env  *eth2bpenv
	head common.Hash
}

// NewEth2API creates a new API definition for the eth2 prototype.
func NewEth2API(eth *Ethereum) *Eth2API {
	return &Eth2API{eth: eth}
}

type eth2bpenv struct {
	state   *state.StateDB
	tcount  int
	gasPool *core.GasPool

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt
}

func (api *Eth2API) commitTransaction(tx *types.Transaction, coinbase common.Address, bcParentRoots []common.Hash, randao common.Hash) error {
	//snap := eth2rpc.current.state.Snapshot()

	chain := api.eth.BlockChain()
	receipt, err := core.ApplyTransaction(chain.Config(), chain, &coinbase, api.env.gasPool, api.env.state, api.env.header, tx, &api.env.header.GasUsed, *chain.GetVMConfig(), &vm.BeaconChainContext{bcParentRoots, randao})
	if err != nil {
		//w.current.state.RevertToSnapshot(snap)
		return err
	}
	api.env.txs = append(api.env.txs, tx)
	api.env.receipts = append(api.env.receipts, receipt)

	return nil
}

func (api *Eth2API) makeEnv(parent *types.Block, header *types.Header) error {
	state, err := api.eth.BlockChain().StateAt(parent.Root())
	if err != nil {
		return err
	}
	api.env = &eth2bpenv{
		state:   state,
		header:  header,
		gasPool: new(core.GasPool).AddGas(header.GasLimit),
	}
	return nil
}

// Structure described at https://hackmd.io/T9x2mMA4S7us8tJwEB3FDQ
type ProduceBlockParams struct {
	ParentHash             common.Hash   `json:"parent_hash"`
	RandaoMix              common.Hash   `json:"randao_mix"`
	Slot                   uint64        `json:"slot"`
	Timestamp              uint64        `json:"timestamp"`
	RecentBeaconBlockRoots []common.Hash `json:"recent_beacon_block_roots"`
}

// Structure described at https://ethresear.ch/t/executable-beacon-chain/8271
type ExecutableData struct {
	Coinbase     common.Address       `json:"coinbase"`
	StateRoot    common.Hash          `json:"state_root"`
	GasLimit     uint64               `json:"gas_limit"`
	GasUsed      uint64               `json:"gas_used"`
	Transactions []*types.Transaction `json:"transactions"`
	ReceiptRoot  common.Hash          `json:"receipt_root"`
	LogsBloom    []byte               `json:"logs_bloom"`
	BlockHash    common.Hash          `json:"block_hash"`
	ParentHash   common.Hash          `json:"parent_hash"`
	Difficulty   *big.Int             `json:"difficulty"`
}

// ProduceBlock creates a new block, inserts it into the chain, and returns the "execution
// data" required for eth2 clients to process the new block.
func (api *Eth2API) ProduceBlock(params ProduceBlockParams) (*ExecutableData, error) {
	log.Info("Produce block", "parentHash", params.ParentHash)

	bc := api.eth.BlockChain()
	parent := bc.GetBlockByHash(params.ParentHash)
	pool := api.eth.TxPool()

	if parent.Time() >= params.Timestamp {
		return nil, fmt.Errorf("child timestamp lower than parent's: %d >= %d", parent.Time(), params.Timestamp)
	}
	// this will ensure we're not going off too far in the future
	if now := uint64(time.Now().Unix()); params.Timestamp > now+1 {
		wait := time.Duration(params.Timestamp-now) * time.Second
		log.Info("Producing block too far in the future", "wait", common.PrettyDuration(wait))
		time.Sleep(wait)
	}

	pending, err := pool.Pending()
	if err != nil {
		return nil, err
	}

	coinbase, err := api.eth.Etherbase()
	if err != nil {
		return nil, err
	}
	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		Coinbase:   coinbase,
		GasLimit:   parent.GasLimit(), // Keep the gas limit constant in this prototype
		Extra:      []byte{},
		Time:       params.Timestamp,
	}
	err = api.eth.Engine().Prepare(bc, header)
	if err != nil {
		return nil, err
	}

	err = api.makeEnv(parent, header)
	if err != nil {
		return nil, err
	}
	signer := types.NewEIP155Signer(bc.Config().ChainID)
	txs := types.NewTransactionsByPriceAndNonce(signer, pending)

	var transactions []*types.Transaction

	for {
		if api.env.gasPool.Gas() < chainParams.TxGas {
			log.Trace("Not enough gas for further transactions", "have", api.env.gasPool, "want", chainParams.TxGas)
			break
		}

		tx := txs.Peek()
		if tx == nil {
			break
		}

		from, _ := types.Sender(signer, tx)
		// XXX replay protection check is missing

		// Execute the transaction
		api.env.state.Prepare(tx.Hash(), common.Hash{}, api.env.tcount)
		err := api.commitTransaction(tx, coinbase, params.RecentBeaconBlockRoots, params.RandaoMix)
		switch err {
		case core.ErrGasLimitReached:
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case core.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case core.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with high nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case nil:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			api.env.tcount++
			txs.Shift()
			transactions = append(transactions, tx)

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}

	block, err := api.eth.Engine().FinalizeAndAssemble(bc, header, api.env.state, transactions, nil /* uncles */, api.env.receipts)
	if err != nil {
		return nil, err
	}

	var logs []*types.Log
	var receipts = make(types.Receipts, len(api.env.receipts))
	hash := block.Hash()
	for i, receipt := range api.env.receipts {
		// add block location fields
		receipt.BlockHash = hash
		receipt.BlockNumber = block.Number()
		receipt.TransactionIndex = uint(i)

		receipts[i] = new(types.Receipt)
		*receipts[i] = *receipt
		// Update the block hash in all logs since it is now available and not when the
		// receipt/log of individual transactions were created.
		for _, log := range receipt.Logs {
			log.BlockHash = hash
		}
		logs = append(logs, receipt.Logs...)
	}

	block.Header().ReceiptHash = types.DeriveSha(receipts, new(trie.Trie))

	return &ExecutableData{
		Coinbase:     block.Coinbase(),
		StateRoot:    block.Root(),
		GasLimit:     block.GasLimit(),
		GasUsed:      block.GasUsed(),
		Transactions: []*types.Transaction(block.Transactions()),
		ReceiptRoot:  block.ReceiptHash(),
		LogsBloom:    block.Bloom().Bytes(),
		BlockHash:    block.Hash(),
		Difficulty:   block.Difficulty(),
		ParentHash:   block.ParentHash(),
	}, nil
}

// Structure described at https://hackmd.io/T9x2mMA4S7us8tJwEB3FDQ
type InsertBlockParams struct {
	RandaoMix              common.Hash    `json:"randao_mix"`
	Slot                   uint64         `json:"slot"`
	Timestamp              uint64         `json:"timestamp"`
	RecentBeaconBlockRoots []common.Hash  `json:"recent_beacon_block_roots"`
	ExecutableData         ExecutableData `json:"executable_data"`
}

var zeroNonce [8]byte

func insertBlockParamsToBlock(params InsertBlockParams, number *big.Int) *types.Block {
	header := &types.Header{
		ParentHash:  params.ExecutableData.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    params.ExecutableData.Coinbase,
		Root:        params.ExecutableData.StateRoot,
		TxHash:      types.DeriveSha(types.Transactions(params.ExecutableData.Transactions), trie.NewStackTrie(nil)),
		ReceiptHash: params.ExecutableData.ReceiptRoot,
		Bloom:       types.BytesToBloom(params.ExecutableData.LogsBloom),
		Difficulty:  params.ExecutableData.Difficulty,
		Number:      number,
		GasLimit:    params.ExecutableData.GasLimit,
		GasUsed:     params.ExecutableData.GasUsed,
		Time:        params.Timestamp,
		Extra:       nil,
		MixDigest:   common.Hash{},
		Nonce:       zeroNonce,
	}
	block := types.NewBlockWithHeader(header).WithBody(params.ExecutableData.Transactions, nil /* uncles */)

	return block
}

// InsertBlock creates an Eth1 block, inserts it in the chain, and either returns true,
// or false + an error. This is a bit redundant for go, but simplifies things on the
// eth2 side.
func (api *Eth2API) InsertBlock(params InsertBlockParams) (bool, error) {
	// compute block number as parent.number + 1
	parent := api.eth.BlockChain().GetBlockByHash(params.ExecutableData.ParentHash)
	if parent == nil {
		return false, fmt.Errorf("could not find parent %x", params.ExecutableData.ParentHash)
	}
	number := big.NewInt(0)
	number.Add(parent.Number(), big.NewInt(1))

	block := insertBlockParamsToBlock(params, number)
	_, err := api.eth.BlockChain().InsertChainWithoutSealVerification(types.Blocks([]*types.Block{block}))

	return (err == nil), err
}

func (api *Eth2API) AddBlockTxs(block *types.Block) error {
	for _, tx := range block.Transactions() {
		api.eth.txPool.AddLocal(tx)
	}

	return nil
}

//func (api *Eth2API) SetHead(newHead common.Hash) error {
//oldBlock := api.eth.BlockChain().CurrentBlock()

//if oldBlock.Hash() == newHead {
//return nil
//}

//newBlock := api.eth.BlockChain().GetBlockByHash(newHead)

//err := api.eth.BlockChain().Reorg(oldBlock, newBlock)
//if err != nil {
//return err
//}
//api.head = newHead
//return nil
//}
