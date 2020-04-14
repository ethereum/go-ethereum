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

package eth2

import (
	"bytes"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// Validator validates eth1 blocks passed by the eth2 validator
type Eth2 struct {
	//mux      *event.TypeMux
	//worker *worker
	//coinbase common.Address
	eth    miner.Backend
	engine consensus.Engine
	exitCh chan struct{}

	coinbase common.Address

	//canStart    int32 // can start indicates whether we can start the mining operation
	//shouldStart int32 // should start indicates whether we should start after sync
}

func New(engine consensus.Engine) *Eth2 {
	return &Eth2{
		engine: engine,
		exitCh: make(chan struct{}),
	}
}

func (producer *Eth2) Start() {
}

func (producer *Eth2) Stop() {
}

func (producer *Eth2) Close() {
}

type env struct {
	state   *state.StateDB
	tcount  int
	gasPool *core.GasPool

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt
}

type Eth2RPC struct {
	eth2 *Eth2
	env  *env
}

func (eth2rpc *Eth2RPC) makeEnv(parent *types.Block, header *types.Header) error {
	state, err := eth2rpc.eth2.eth.BlockChain().StateAt(parent.Root())
	if err != nil {
		return err
	}
	eth2rpc.env = &env{
		state:  state,
		header: header,
	}
	return nil
}

func (eth2rpc *Eth2RPC) commitTransaction(tx *types.Transaction, coinbase common.Address) ([]*types.Log, error) {
	//snap := eth2rpc.current.state.Snapshot()

	chain := eth2rpc.eth2.eth.BlockChain()
	receipt, err := core.ApplyTransaction(chain.Config(), chain, &coinbase, eth2rpc.env.gasPool, eth2rpc.env.state, eth2rpc.env.header, tx, &eth2rpc.env.header.GasUsed, *chain.GetVMConfig())
	if err != nil {
		//w.current.state.RevertToSnapshot(snap)
		return nil, err
	}
	eth2rpc.env.txs = append(eth2rpc.env.txs, tx)
	eth2rpc.env.receipts = append(eth2rpc.env.receipts, receipt)

	return receipt.Logs, nil
}

// TODO provide parent's hash and check this is indeed the current block
func (eth2rpc *Eth2RPC) ProduceBlock() ([]byte, error) {
	eth := eth2rpc.eth2.eth
	bc := eth2rpc.eth2.eth.BlockChain()
	parent := bc.CurrentBlock()
	pool := eth.TxPool()
	pending, err := pool.Pending()
	if err != nil {
		return nil, err
	}

	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		GasLimit:   core.CalcGasLimit(parent, w.config.GasFloor, w.config.GasCeil),
		Extra:      []byte{},
		Time:       uint64(time.Now().UnixNano()),
	}
	err = eth2rpc.eth2.engine.Prepare(bc, header)
	if err != nil {
		return nil, err
	}

	err = eth2rpc.makeEnv(parent, header)
	if err != nil {
		return nil, err
	}
	signer := types.NewEIP155Signer(eth.BlockChain().Config().ChainID)
	txs := types.NewTransactionsByPriceAndNonce(signer, pending)

	for {
		if eth2rpc.env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", eth2rpc.env.gasPool, "want", params.TxGas)
			break
		}

		tx := txs.Peek()
		if tx == nil {
			break
		}

		from, _ := types.Sender(signer, tx)
		// XXX manque qqch pour le replay protection

		// Execute the transaction
		eth2rpc.env.state.Prepare(tx.Hash(), common.Hash{}, eth2rpc.env.tcount)
		_, err := eth2rpc.commitTransaction(tx, eth2rpc.eth2.coinbase)
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
			//coalescedLogs = append(coalescedLogs, logs...)
			//w.current.tcount++
			txs.Shift()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}

	block, err := eth2rpc.eth2.engine.FinalizeAndAssemble(bc, header, eth2rpc.env.state, txs, nil /* uncles */, nil /* receipts */)
	if err != nil {
		return nil, err
	}

	var rlpbuf bytes.Buffer
	block.EncodeRLP(&rlpbuf)
	return rlpbuf.Bytes(), nil
}

func (e2rpc *Eth2RPC) InsertBlock(blockRLP []byte) error {
	var block types.Block
	if err := rlp.DecodeBytes(blockRLP, &block); err != nil {
		return err
	}

	_, err := e2rpc.eth2.eth.BlockChain().InsertChain(types.Blocks([]*types.Block{&block}))
	return err
}

func (e2rpc *Eth2RPC) ValidateBlock(blockRLP []byte) (bool, error) {
	blockchain := e2rpc.eth2.worker.eth.BlockChain()
	// Deserializer le block
	var block types.Block
	if err := rlp.DecodeBytes(blockRLP, &block); err != nil {
		return false, err
	}

	// Validate the block using the consensus engine - this will
	// allow for testing against Rinkeby/Görli
	if err := e2rpc.eth2.engine.VerifyHeader(blockchain, block.Header(), false); err != nil {
		return false, err
	}

	if err := blockchain.Validator().ValidateBody(&block); err != nil {
		return false, err
	}

	parent := blockchain.GetBlockByHash(block.ParentHash())
	statedb, err := state.New(parent.Root(), blockchain.StateCache(), blockchain.Snapshot())
	if err != nil {
		return false, err
	}
	receipts, _, usedGas, err := blockchain.Processor().Process(&block, statedb, *blockchain.GetVMConfig())
	if err != nil {
		return false, err
	}

	if err := blockchain.Validator().ValidateState(&block, statedb, receipts, usedGas); err != nil {
		return false, err
	}

	return true, nil
}

func (e2rpc *Eth2RPC) SetHead(hash common.Hash) error {
	bc := e2rpc.eth2.eth.BlockChain()
	bc.SetHead()
	return nil
}

// TODO this is a temporary value, it should be stored
// into the DB and initialized by the `Start` method.
var lastFinalizedBlock uint64 = 0

// Donc ce qu'il faut, c'est un truc qui me permet de détruire les
// blocs d'avant ainsi que tous leurs oncles et toutes les autres
// branches. Il faut enfin que je puisse indiquer où est la head,
// comme ça je peux savoir d'où partir. Il faut enfin que je sois
// capable de modifier le code pour que les réorgs ne repartent pas
// du début. Peut-être une autre PR juste pour empêcher les réorgs.
func (e2rpc *Eth2RPC) FinalizeBlock(block common.Hash) error {
	bc := e2rpc.eth2.eth.BlockChain()
	bc.FinalizeBlock(block)
	return nil
}
