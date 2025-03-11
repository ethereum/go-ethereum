// Copyright 2023 The go-ethereum Authors
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
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
)

func startSimulatedBeaconEthService(t *testing.T, genesis *core.Genesis, period uint64) (*node.Node, *eth.Ethereum, *SimulatedBeacon) {
	t.Helper()

	n, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "127.0.0.1:0",
			NoDiscovery: true,
			MaxPeers:    0,
		},
	})
	if err != nil {
		t.Fatal("can't create node:", err)
	}

	ethcfg := &ethconfig.Config{Genesis: genesis, SyncMode: ethconfig.FullSync, TrieTimeout: time.Minute, TrieDirtyCache: 256, TrieCleanCache: 256, Miner: miner.DefaultConfig}
	ethservice, err := eth.New(n, ethcfg)
	if err != nil {
		t.Fatal("can't create eth service:", err)
	}

	simBeacon, err := NewSimulatedBeacon(period, ethservice)
	if err != nil {
		t.Fatal("can't create simulated beacon:", err)
	}

	n.RegisterLifecycle(simBeacon)

	if err := n.Start(); err != nil {
		t.Fatal("can't start node:", err)
	}

	ethservice.SetSynced()
	return n, ethservice, simBeacon
}

// send 20 transactions, >10 withdrawals and ensure they are included in order
// send enough transactions to fill multiple blocks
func TestSimulatedBeaconSendWithdrawals(t *testing.T) {
	var withdrawals []types.Withdrawal
	txs := make(map[common.Hash]*types.Transaction)

	var (
		// testKey is a private key to use for funding a tester account.
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

		// testAddr is the Ethereum address of the tester account.
		testAddr = crypto.PubkeyToAddress(testKey.PublicKey)
	)

	// short period (1 second) for testing purposes
	var gasLimit uint64 = 10_000_000
	genesis := core.DeveloperGenesisBlock(gasLimit, &testAddr)
	node, ethService, mock := startSimulatedBeaconEthService(t, genesis, 1)
	_ = mock
	defer node.Close()

	chainHeadCh := make(chan core.ChainHeadEvent, 10)
	subscription := ethService.BlockChain().SubscribeChainHeadEvent(chainHeadCh)
	defer subscription.Unsubscribe()

	// generate some withdrawals
	for i := 0; i < 20; i++ {
		withdrawals = append(withdrawals, types.Withdrawal{Index: uint64(i)})
		if err := mock.withdrawals.add(&withdrawals[i]); err != nil {
			t.Fatal("addWithdrawal failed", err)
		}
	}

	// generate a bunch of transactions
	signer := types.NewEIP155Signer(ethService.BlockChain().Config().ChainID)
	for i := 0; i < 20; i++ {
		tx, err := types.SignTx(types.NewTransaction(uint64(i), common.Address{}, big.NewInt(1000), params.TxGas, big.NewInt(params.InitialBaseFee), nil), signer, testKey)
		if err != nil {
			t.Fatalf("error signing transaction, err=%v", err)
		}
		txs[tx.Hash()] = tx

		if err := ethService.APIBackend.SendTx(context.Background(), tx); err != nil {
			t.Fatal("SendTx failed", err)
		}
	}

	includedTxs := make(map[common.Hash]struct{})
	var includedWithdrawals []uint64

	timer := time.NewTimer(12 * time.Second)
	for {
		select {
		case ev := <-chainHeadCh:
			block := ethService.BlockChain().GetBlock(ev.Header.Hash(), ev.Header.Number.Uint64())
			for _, includedTx := range block.Transactions() {
				includedTxs[includedTx.Hash()] = struct{}{}
			}
			for _, includedWithdrawal := range block.Withdrawals() {
				includedWithdrawals = append(includedWithdrawals, includedWithdrawal.Index)
			}
			// ensure all withdrawals/txs included. this will take two blocks b/c number of withdrawals > 10
			if len(includedTxs) == len(txs) && len(includedWithdrawals) == len(withdrawals) && ev.Header.Number.Cmp(big.NewInt(2)) == 0 {
				return
			}
		case <-timer.C:
			t.Fatal("timed out without including all withdrawals/txs")
		}
	}
}

// Tests that zero-period dev mode can handle a lot of simultaneous
// transactions/withdrawals
func TestOnDemandSpam(t *testing.T) {
	// This test is flaky, due to various causes, and the root cause is synchronicity.
	// We have optimistic timeouts here and there in the simulated becaon and the worker.
	// This test typically fails on 32-bit windows appveyor.
	t.Skip("flaky test")
	var (
		withdrawals     []types.Withdrawal
		txCount                = 20000
		wxCount                = 20
		testKey, _             = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		testAddr               = crypto.PubkeyToAddress(testKey.PublicKey)
		gasLimit        uint64 = 10_000_000
		genesis                = core.DeveloperGenesisBlock(gasLimit, &testAddr)
		node, eth, mock        = startSimulatedBeaconEthService(t, genesis, 0)
		_                      = newSimulatedBeaconAPI(mock)
		signer                 = types.LatestSigner(eth.BlockChain().Config())
		chainHeadCh            = make(chan core.ChainHeadEvent, 100)
		sub                    = eth.BlockChain().SubscribeChainHeadEvent(chainHeadCh)
	)
	defer node.Close()
	defer sub.Unsubscribe()

	// generate some withdrawals
	for i := 0; i < wxCount; i++ {
		withdrawals = append(withdrawals, types.Withdrawal{Index: uint64(i)})
		if err := mock.withdrawals.add(&withdrawals[i]); err != nil {
			t.Fatal("addWithdrawal failed", err)
		}
	}

	// generate a bunch of transactions
	go func() {
		for i := 0; i < txCount; i++ {
			tx, err := types.SignTx(types.NewTransaction(uint64(i), common.Address{byte(i), byte(1)}, big.NewInt(1000), params.TxGas, big.NewInt(params.InitialBaseFee*2), nil), signer, testKey)
			if err != nil {
				panic(fmt.Sprintf("error signing transaction: %v", err))
			}
			if err := eth.TxPool().Add([]*types.Transaction{tx}, false)[0]; err != nil {
				panic(fmt.Sprintf("error adding txs to pool: %v", err))
			}
		}
	}()
	var (
		includedTxs int
		includedWxs int
		abort       = time.NewTimer(10 * time.Second)
	)
	defer abort.Stop()
	for {
		select {
		case ev := <-chainHeadCh:
			block := eth.BlockChain().GetBlock(ev.Header.Hash(), ev.Header.Number.Uint64())
			includedTxs += len(block.Transactions())
			includedWxs += len(block.Withdrawals())
			// ensure all withdrawals/txs included. this will take two blocks b/c number of withdrawals > 10
			if includedTxs == txCount && includedWxs == wxCount {
				return
			}
			abort.Reset(10 * time.Second)
		case <-abort.C:
			t.Fatalf("timed out without including all withdrawals/txs: have txs %d, want %d, have wxs %d, want %d",
				includedTxs, txCount, includedWxs, wxCount)
		}
	}
}
