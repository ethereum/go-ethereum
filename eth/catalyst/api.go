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

package catalyst

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	chainParams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

type consensusAPI struct {
	eth  *eth.Ethereum
	env  *eth2bpenv
	head common.Hash
}

func newConsensusAPI(eth *eth.Ethereum) *consensusAPI {
	return &consensusAPI{eth: eth}
}

type eth2bpenv struct {
	state   *state.StateDB
	tcount  int
	gasPool *core.GasPool

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt
}

func (api *consensusAPI) commitTransaction(tx *types.Transaction, coinbase common.Address) error {
	//snap := eth2rpc.current.state.Snapshot()

	chain := api.eth.BlockChain()
	receipt, err := core.ApplyTransaction(chain.Config(), chain, &coinbase, api.env.gasPool, api.env.state, api.env.header, tx, &api.env.header.GasUsed, *chain.GetVMConfig())
	if err != nil {
		//w.current.state.RevertToSnapshot(snap)
		return err
	}
	api.env.txs = append(api.env.txs, tx)
	api.env.receipts = append(api.env.receipts, receipt)

	return nil
}

func (api *consensusAPI) makeEnv(parent *types.Block, header *types.Header) error {
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
type AssembleBlockParams struct {
	ParentHash common.Hash `json:"parent_hash"`
	Timestamp  uint64      `json:"timestamp"`
}

// Structure described at https://notes.ethereum.org/@n0ble/rayonism-the-merge-spec#Parameters1
type ExecutableData struct {
	BlockHash    common.Hash          `json:"blockHash"`
	ParentHash   common.Hash          `json:"parentHash"`
	Miner        common.Address       `json:"miner"`
	StateRoot    common.Hash          `json:"stateRoot"`
	Number       uint64               `json:"number"`
	GasLimit     uint64               `json:"gasLimit"`
	GasUsed      uint64               `json:"gasUsed"`
	Timestamp    uint64               `json:"timestamp"`
	ReceiptRoot  common.Hash          `json:"receiptsRoot"`
	LogsBloom    []byte               `json:"logsBloom"`
	Transactions []*types.Transaction `json:"transactions"`
}

// AssembleBlock creates a new block, inserts it into the chain, and returns the "execution
// data" required for eth2 clients to process the new block.
func (api *consensusAPI) AssembleBlock(params AssembleBlockParams) (*ExecutableData, error) {
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
	signer := types.LatestSigner(bc.Config())
	txs := types.NewTransactionsByPriceAndNonce(signer, pending)

	var transactions []*types.Transaction

	for {
		if api.env.gasPool.Gas() < chainParams.TxGas {
			log.Trace("Not enough gas for further transactions", "have", api.env.gasPool, "want", chainParams.TxGas)
			break
		}

		tx := txs.Peek()
		if tx == nil {
			fmt.Println("no tx")
			break
		}

		from, _ := types.Sender(signer, tx)
		// XXX replay protection check is missing

		// Execute the transaction
		api.env.state.Prepare(tx.Hash(), common.Hash{}, api.env.tcount)
		err := api.commitTransaction(tx, coinbase)
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
		BlockHash:    block.Hash(),
		ParentHash:   block.ParentHash(),
		Miner:        block.Coinbase(),
		StateRoot:    block.Root(),
		Number:       block.NumberU64(),
		GasLimit:     block.GasLimit(),
		GasUsed:      block.GasUsed(),
		Timestamp:    block.Time(),
		ReceiptRoot:  block.ReceiptHash(),
		LogsBloom:    block.Bloom().Bytes(),
		Transactions: []*types.Transaction(block.Transactions()),
	}, nil
}

var zeroNonce [8]byte

func insertBlockParamsToBlock(params ExecutableData, number *big.Int) *types.Block {
	header := &types.Header{
		ParentHash:  params.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    params.Miner,
		Root:        params.StateRoot,
		TxHash:      types.DeriveSha(types.Transactions(params.Transactions), trie.NewStackTrie(nil)),
		ReceiptHash: params.ReceiptRoot,
		Bloom:       types.BytesToBloom(params.LogsBloom),
		Difficulty:  big.NewInt(1),
		Number:      number,
		GasLimit:    params.GasLimit,
		GasUsed:     params.GasUsed,
		Time:        params.Timestamp,
		Extra:       nil,
		MixDigest:   common.Hash{},
		Nonce:       zeroNonce,
	}
	block := types.NewBlockWithHeader(header).WithBody(params.Transactions, nil /* uncles */)

	return block
}

type NewBlockReturn struct {
	Valid bool `json:"valid"`
}

// NewBlock creates an Eth1 block, inserts it in the chain, and either returns true,
// or false + an error. This is a bit redundant for go, but simplifies things on the
// eth2 side.
func (api *consensusAPI) NewBlock(params ExecutableData) (*NewBlockReturn, error) {
	// compute block number as parent.number + 1
	parent := api.eth.BlockChain().GetBlockByHash(params.ParentHash)
	if parent == nil {
		return &NewBlockReturn{false}, fmt.Errorf("could not find parent %x", params.ParentHash)
	}

	number := big.NewInt(0)
	number.SetUint64(params.Number)
	block := insertBlockParamsToBlock(params, number)
	_, err := api.eth.BlockChain().InsertChainWithoutSealVerification(block)

	return &NewBlockReturn{err == nil}, err
}

// Used in tests to add a the list of transactions from a block to the tx pool.
func (api *consensusAPI) addBlockTxs(block *types.Block) error {
	for _, tx := range block.Transactions() {
		api.eth.TxPool().AddLocal(tx)
	}

	return nil
}

type GenericResponse struct {
	Success bool `json:"success"`
}

// FinalizeBlock is called to mark a block as synchronized, so
// that data that is no longer needed can be removed.
func (api *consensusAPI) FinalizeBlock(blockHash common.Hash) (*GenericResponse, error) {
	// Stubbed for now, it's not critical
	return &GenericResponse{false}, nil
}

// SetHead is called to perform a force choice.
func (api *consensusAPI) SetHead(newHead common.Hash) (*GenericResponse, error) {
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
	return &GenericResponse{false}, nil
}
