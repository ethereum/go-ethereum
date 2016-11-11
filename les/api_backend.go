// Copyright 2016 The go-ethereum Authors
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

package les

import (
	"errors"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/light"
	"golang.org/x/net/context"
)

var (
	ErrBlockNotFound   = errors.New("block not found")
	ErrTxNotFound      = errors.New("the light client cannot retrieve past transactions by hash")
	ErrReceiptNotFound = errors.New("the light client cannot retrieve receipts by hash")
)

func (eth *LightEthereum) HeaderByHash(ctx context.Context, blockhash common.Hash) (*types.Header, error) {
	if h := eth.blockchain.GetHeaderByHash(blockhash); h != nil {
		return h, nil
	}
	return nil, ErrBlockNotFound
}

func (eth *LightEthereum) HeaderByNumber(ctx context.Context, blocknum *big.Int) (*types.Header, error) {
	if blocknum == nil {
		return eth.blockchain.CurrentHeader(), nil
	}
	h, err := eth.blockchain.GetHeaderByNumberOdr(ctx, uint64(blocknum.Uint64()))
	if err != nil {
		return nil, err
	}
	if h == nil {
		return nil, ErrBlockNotFound
	}
	return h, nil
}

func (eth *LightEthereum) BlockByNumber(ctx context.Context, blocknum *big.Int) (*types.Block, error) {
	header, err := eth.HeaderByNumber(ctx, blocknum)
	if err != nil {
		return nil, err
	}
	return eth.BlockByHash(ctx, header.Hash())
}

func (eth *LightEthereum) BlockByHash(ctx context.Context, blockhash common.Hash) (*types.Block, error) {
	return eth.blockchain.GetBlockByHash(ctx, blockhash)
}

func (eth *LightEthereum) BlockReceipts(ctx context.Context, blockhash common.Hash, blocknum uint64) ([]*types.Receipt, error) {
	return light.GetBlockReceipts(ctx, eth.odr, blockhash, blocknum)
}

func (eth *LightEthereum) BlockTD(blockHash common.Hash) *big.Int {
	return eth.blockchain.GetTdByHash(blockHash)
}

func (eth *LightEthereum) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	if tx = eth.txPool.GetTransaction(txHash); tx != nil {
		return tx, true, nil
	}
	return nil, false, ErrTxNotFound
}

func (eth *LightEthereum) TransactionInBlock(ctx context.Context, blockhash common.Hash, index uint) (*types.Transaction, error) {
	b, err := eth.blockchain.GetBlockByHash(ctx, blockhash)
	if err != nil {
		return nil, err
	}
	if index >= uint(len(b.Transactions())) {
		return nil, fmt.Errorf("transaction index out of range")
	}
	return b.Transactions()[index], nil
}

func (eth *LightEthereum) TransactionReceipt(ctx context.Context, txhash common.Hash) (*types.Receipt, error) {
	return nil, ErrReceiptNotFound
}

func (eth *LightEthereum) TransactionCount(ctx context.Context, blockhash common.Hash) (uint, error) {
	b, err := eth.blockchain.GetBlockByHash(ctx, blockhash)
	if err != nil {
		return 0, err
	}
	return uint(len(b.Transactions())), nil
}

func (eth *LightEthereum) BalanceAt(ctx context.Context, addr common.Address, blocknum *big.Int) (*big.Int, error) {
	st, err := eth.state(ctx, blocknum)
	if err != nil {
		return nil, err
	}
	return st.GetBalance(ctx, addr)
}

func (eth *LightEthereum) CodeAt(ctx context.Context, addr common.Address, blocknum *big.Int) ([]byte, error) {
	st, err := eth.state(ctx, blocknum)
	if err != nil {
		return nil, err
	}
	return st.GetCode(ctx, addr)
}

func (eth *LightEthereum) NonceAt(ctx context.Context, addr common.Address, blocknum *big.Int) (uint64, error) {
	st, err := eth.state(ctx, blocknum)
	if err != nil {
		return 0, err
	}
	return st.GetNonce(ctx, addr)
}

func (eth *LightEthereum) StorageAt(ctx context.Context, addr common.Address, key common.Hash, blocknum *big.Int) ([]byte, error) {
	st, err := eth.state(ctx, blocknum)
	if err != nil {
		return nil, err
	}
	v, err := st.GetState(ctx, addr, key)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}

func (eth *LightEthereum) PendingCodeAt(ctx context.Context, addr common.Address) ([]byte, error) {
	// TODO(fjl): find a way to get rid of PendingCodeAt here. CodeAt is a bad emulation
	// of PendingCodeAt because it forces users to wait for transactions to get mined.
	return eth.CodeAt(ctx, addr, nil)
}

func (eth *LightEthereum) state(ctx context.Context, blocknum *big.Int) (*light.LightState, error) {
	header, err := eth.HeaderByNumber(ctx, blocknum)
	if err != nil {
		return nil, err
	}
	return light.NewLightState(light.StateTrieID(header), eth.odr), nil
}

func (eth *LightEthereum) SendTransaction(ctx context.Context, signedTx *types.Transaction) error {
	return eth.txPool.Add(ctx, signedTx)
}

func (eth *LightEthereum) RemoveTransaction(txHash common.Hash) {
	eth.txPool.RemoveTx(txHash)
}

func (eth *LightEthereum) PendingTransactions() []*types.Transaction {
	return eth.txPool.GetTransactions()
}

func (eth *LightEthereum) PendingNonceAt(ctx context.Context, addr common.Address) (uint64, error) {
	return eth.txPool.GetNonce(ctx, addr)
}

func (eth *LightEthereum) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	return eth.protocolManager.downloader.Progress(), nil
}

func (eth *LightEthereum) ProtocolVersion() int {
	return int(eth.protocolManager.SubProtocols[0].Version) + 10000
}

func (eth *LightEthereum) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return eth.gpo.SuggestPrice(ctx)
}

func (eth *LightEthereum) AccountManager() *accounts.Manager {
	return eth.accountManager
}

func (eth *LightEthereum) ResetHeadBlock(number uint64) {
	eth.blockchain.SetHead(number)
}

func (eth *LightEthereum) EstimateGas(ctx context.Context, call ethereum.CallMsg) (usedGas *big.Int, err error) {
	_, gas, err := eth.callContract(ctx, call, nil)
	return gas, err
}

func (eth *LightEthereum) CallContract(ctx context.Context, call ethereum.CallMsg, blocknum *big.Int) ([]byte, error) {
	val, _, err := eth.callContract(ctx, call, blocknum)
	return val, err
}

func (eth *LightEthereum) callContract(ctx context.Context, call ethereum.CallMsg, blocknum *big.Int) ([]byte, *big.Int, error) {
	var head *types.Header
	if blocknum == nil {
		head = eth.blockchain.CurrentHeader()
	} else if head = eth.blockchain.GetHeaderByNumber(blocknum.Uint64()); head == nil {
		return nil, nil, ErrBlockNotFound
	}
	state := light.NewLightState(light.StateTrieID(head), eth.odr)
	return core.ApplyCallMessage(call, func(msg core.Message) vm.Environment {
		return light.NewEnv(ctx, state, eth.chainConfig, eth.blockchain, msg, head, vm.Config{})
	})
}
