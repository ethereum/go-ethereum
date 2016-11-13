// Copyright 2015 The go-ethereum Authors
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
	"errors"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"golang.org/x/net/context"
)

// Assert at compile time that this implementation has all optional API features.
var (
	_ ethapi.TransactionInclusionBlock = (*Ethereum)(nil)
	_ ethapi.PendingState              = (*Ethereum)(nil)
	_ bind.ContractBackend             = (*Ethereum)(nil)
	_ bind.PendingContractCaller       = (*Ethereum)(nil)
)

var (
	ErrBlockNotFound = errors.New("block not found")
	ErrTxNotFound    = errors.New("transaction not found")
)

// HeaderByNumber returns headers from the canonical chain.
func (eth *Ethereum) HeaderByNumber(ctx context.Context, num *big.Int) (*types.Header, error) {
	if num == nil {
		return eth.blockchain.CurrentBlock().Header(), nil
	}
	if h := eth.blockchain.GetHeaderByNumber(num.Uint64()); h != nil {
		return h, nil
	}
	return nil, ErrBlockNotFound
}

// HeaderByHash returns the header with the given hash.
func (eth *Ethereum) HeaderByHash(ctx context.Context, blockhash common.Hash) (*types.Header, error) {
	if h := eth.blockchain.GetHeaderByHash(blockhash); h != nil {
		return h, nil
	}
	return nil, ErrBlockNotFound
}

// BlockByNumber returns blocks from the canonical chain.
func (eth *Ethereum) BlockByNumber(ctx context.Context, num *big.Int) (*types.Block, error) {
	if num == nil {
		return eth.blockchain.CurrentBlock(), nil
	}
	if b := eth.blockchain.GetBlockByNumber(num.Uint64()); b != nil {
		return b, nil
	}
	return nil, ErrBlockNotFound
}

// BlockByHash returns the block with the given hash.
func (eth *Ethereum) BlockByHash(ctx context.Context, blockhash common.Hash) (*types.Block, error) {
	if b := eth.blockchain.GetBlockByHash(blockhash); b != nil {
		return b, nil
	}
	return nil, ErrBlockNotFound
}

// BlockReceipts returns all receipts contained in the given block.
func (eth *Ethereum) BlockReceipts(ctx context.Context, blockhash common.Hash, number uint64) ([]*types.Receipt, error) {
	r := core.GetBlockReceipts(eth.chainDb, blockhash, number)
	if r == nil {
		return nil, errors.New("database has no valid receipts for the given block")
	}
	return r, nil
}

// TransactionCount returns the number of transactions in a block.
func (eth *Ethereum) TransactionCount(ctx context.Context, blockhash common.Hash) (uint, error) {
	b := eth.blockchain.GetBlockByHash(blockhash)
	if b == nil {
		return 0, nil
	}
	return uint(len(b.Transactions())), nil
}

// TransactionInBlock returns the i'th transaction in the given block.
func (eth *Ethereum) TransactionInBlock(ctx context.Context, blockhash common.Hash, i uint) (*types.Transaction, error) {
	b := eth.blockchain.GetBlockByHash(blockhash)
	if b == nil {
		return nil, fmt.Errorf("transaction index %d out of range for non-existent block", i)
	}
	if i >= uint(len(b.Transactions())) {
		return nil, fmt.Errorf("transaction index %d out of range", i)
	}
	return b.Transactions()[i], nil
}

// TransactionByHash returns the transaction with the given hash.
func (eth *Ethereum) TransactionByHash(ctx context.Context, txhash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	if tx = eth.txPool.Get(txhash); tx != nil {
		return tx, true, nil
	}
	if tx, _, _, _ = core.GetTransaction(eth.chainDb, txhash); tx != nil {
		return tx, false, nil
	}
	return nil, false, ErrTxNotFound
}

// TransactionInclusionBlock returns the block in which the given transaction was included.
func (eth *Ethereum) TransactionInclusionBlock(txhash common.Hash) (bhash common.Hash, bnum uint64, index int, err error) {
	var tx *types.Transaction
	if tx, bhash, bnum, index = core.GetTransaction(eth.chainDb, txhash); tx == nil {
		err = ErrTxNotFound
	}
	return bhash, bnum, index, err
}

// TransactionReceipt returns the receipt of a transaction.
func (eth *Ethereum) TransactionReceipt(ctx context.Context, txhash common.Hash) (*types.Receipt, error) {
	r := core.GetReceipt(eth.chainDb, txhash)
	if r == nil {
		return nil, ErrTxNotFound
	}
	return r, nil
}

// BlockTD returns the total difficulty of a certain block.
func (eth *Ethereum) BlockTD(blockhash common.Hash) *big.Int {
	return eth.blockchain.GetTdByHash(blockhash)
}

// BalanceAt returns the balance of the given account.
func (eth *Ethereum) BalanceAt(ctx context.Context, addr common.Address, block *big.Int) (bal *big.Int, err error) {
	err = eth.withStateAt(ctx, block, false, func(st *state.StateDB) { bal = st.GetBalance(addr) })
	return bal, err
}

// CodeAt returns the code of the given account.
func (eth *Ethereum) CodeAt(ctx context.Context, addr common.Address, block *big.Int) (code []byte, err error) {
	err = eth.withStateAt(ctx, block, false, func(st *state.StateDB) { code = st.GetCode(addr) })
	return code, err
}

// NonceAt returns the nonce of the given account.
func (eth *Ethereum) NonceAt(ctx context.Context, addr common.Address, block *big.Int) (nonce uint64, err error) {
	err = eth.withStateAt(ctx, block, false, func(st *state.StateDB) { nonce = st.GetNonce(addr) })
	return nonce, err
}

// StorageAt returns a storage value of the given account.
func (eth *Ethereum) StorageAt(ctx context.Context, addr common.Address, key common.Hash, block *big.Int) (val []byte, err error) {
	err = eth.withStateAt(ctx, block, false, func(st *state.StateDB) { v := st.GetState(addr, key); val = v[:] })
	return val, err
}

// PendingBalanceAt returns the balance of the given account in the pending state.
func (eth *Ethereum) PendingBalanceAt(ctx context.Context, addr common.Address) (bal *big.Int, err error) {
	err = eth.withStateAt(ctx, nil, true, func(st *state.StateDB) { bal = st.GetBalance(addr) })
	return bal, err
}

// PendingBalanceAt returns the code of the given account in the pending state.
func (eth *Ethereum) PendingCodeAt(ctx context.Context, addr common.Address) (code []byte, err error) {
	err = eth.withStateAt(ctx, nil, true, func(st *state.StateDB) { code = st.GetCode(addr) })
	return code, err
}

// PendingBalanceAt returns a storage value of the given account in the pending state.
func (eth *Ethereum) PendingStorageAt(ctx context.Context, addr common.Address, key common.Hash) (val []byte, err error) {
	err = eth.withStateAt(ctx, nil, true, func(st *state.StateDB) { v := st.GetState(addr, key); val = v[:] })
	return val, err
}

// PendingTransaction count returns the number of transactions in the pending block.
func (eth *Ethereum) PendingTransactionCount(ctx context.Context) (uint, error) {
	b, _ := eth.miner.Pending()
	return uint(len(b.Transactions())), nil
}

func (eth *Ethereum) withStateAt(ctx context.Context, block *big.Int, pending bool, fn func(st *state.StateDB)) error {
	var st *state.StateDB
	if pending {
		_, st = eth.miner.Pending()
	} else {
		header, err := eth.HeaderByNumber(ctx, block)
		if err != nil {
			return err
		}
		st, err = eth.BlockChain().StateAt(header.Root)
		if err != nil {
			return err
		}
	}
	fn(st)
	return nil
}

// PendingBlock returns the next block as envisioned by the pending state.
func (eth *Ethereum) PendingBlock() (*types.Block, error) {
	b, _ := eth.miner.Pending()
	return b, nil
}

// PendingNonceAt returns the next valid nonce according to the local transaction pool.
func (eth *Ethereum) PendingNonceAt(ctx context.Context, addr common.Address) (uint64, error) {
	return eth.txPool.NonceAt(addr), nil
}

// PendingTransactions returns all known pending transactions.
func (eth *Ethereum) PendingTransactions() []*types.Transaction {
	eth.txMu.Lock()
	defer eth.txMu.Unlock()

	var txs types.Transactions
	for _, batch := range eth.txPool.Pending() {
		txs = append(txs, batch...)
	}
	return txs
}

// SendTransaction queues a transaction in the pool.
func (eth *Ethereum) SendTransaction(ctx context.Context, signedTx *types.Transaction) error {
	eth.txMu.Lock()
	defer eth.txMu.Unlock()

	eth.txPool.SetLocal(signedTx)
	return eth.txPool.Add(signedTx)
}

// SyncProgress returns sync status information.
func (b *Ethereum) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	return b.protocolManager.downloader.Progress(), nil
}

// SuggestGasPrice returns a suitable gas price based on the content of recently seen blocks.
func (eth *Ethereum) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return eth.gpo.SuggestPrice(), nil
}

// RemoveTransaction removes the given transaction from the local pool.
func (eth *Ethereum) RemoveTransaction(txHash common.Hash) {
	eth.txMu.Lock()
	defer eth.txMu.Unlock()

	eth.txPool.Remove(txHash)
}

// ProtocolVersion returns the active protocol version.
func (eth *Ethereum) ProtocolVersion() int {
	return int(eth.protocolManager.SubProtocols[0].Version)
}

// AccountManager returns the internal account manager that accesses the data directory.
// Deprecated: get the account manager through node.Node instead.
func (eth *Ethereum) AccountManager() *accounts.Manager {
	return eth.accountManager
}

// ResetHeadBlock resets the blockchain to the given number.
// Use this method if you know what you're doing.
func (eth *Ethereum) ResetHeadBlock(blocknum uint64) {
	eth.blockchain.SetHead(blocknum)
}

func (eth *Ethereum) EstimateGas(ctx context.Context, call ethereum.CallMsg) (usedGas *big.Int, err error) {
	block, state := eth.miner.Pending()
	_, gas, err := eth.callContract(call, block.Header(), state)
	return gas, err
}

func (eth *Ethereum) PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error) {
	block, state := eth.miner.Pending()
	val, _, err := eth.callContract(call, block.Header(), state)
	return val, err
}

func (eth *Ethereum) CallContract(ctx context.Context, call ethereum.CallMsg, blocknum *big.Int) ([]byte, error) {
	var head *types.Header
	if blocknum == nil {
		head = eth.blockchain.CurrentHeader()
	} else if head = eth.blockchain.GetHeaderByNumber(blocknum.Uint64()); head == nil {
		return nil, ErrBlockNotFound
	}
	state, err := eth.blockchain.StateAt(head.Root)
	if err != nil {
		return nil, err
	}
	val, _, err := eth.callContract(call, head, state)
	return val, err
}

func (eth *Ethereum) callContract(call ethereum.CallMsg, head *types.Header, state *state.StateDB) ([]byte, *big.Int, error) {
	return core.ApplyCallMessage(call, func(msg core.Message) vm.Environment {
		return core.NewEnv(state, eth.chainConfig, eth.blockchain, msg, head, vm.Config{})
	})
}
