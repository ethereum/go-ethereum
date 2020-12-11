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
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

type odrTestFn func(ctx context.Context, db ethdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte

func TestOdrGetBlockLes2(t *testing.T) { testOdr(t, 2, 1, true, odrGetBlock) }
func TestOdrGetBlockLes3(t *testing.T) { testOdr(t, 3, 1, true, odrGetBlock) }

func odrGetBlock(ctx context.Context, db ethdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	var block *types.Block
	if bc != nil {
		block = bc.GetBlockByHash(bhash)
	} else {
		block, _ = lc.GetBlockByHash(ctx, bhash)
	}
	if block == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(block)
	return rlp
}

func TestOdrGetReceiptsLes2(t *testing.T) { testOdr(t, 2, 1, true, odrGetReceipts) }
func TestOdrGetReceiptsLes3(t *testing.T) { testOdr(t, 3, 1, true, odrGetReceipts) }

func odrGetReceipts(ctx context.Context, db ethdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	var receipts types.Receipts
	if bc != nil {
		if number := rawdb.ReadHeaderNumber(db, bhash); number != nil {
			receipts = rawdb.ReadReceipts(db, bhash, *number, config)
		}
	} else {
		if number := rawdb.ReadHeaderNumber(db, bhash); number != nil {
			receipts, _ = light.GetBlockReceipts(ctx, lc.Odr(), bhash, *number)
		}
	}
	if receipts == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(receipts)
	return rlp
}

func TestOdrAccountsLes2(t *testing.T) { testOdr(t, 2, 1, true, odrAccounts) }
func TestOdrAccountsLes3(t *testing.T) { testOdr(t, 3, 1, true, odrAccounts) }

func odrAccounts(ctx context.Context, db ethdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	dummyAddr := common.HexToAddress("1234567812345678123456781234567812345678")
	acc := []common.Address{bankAddr, userAddr1, userAddr2, dummyAddr}

	var (
		res []byte
		st  *state.StateDB
		err error
	)
	for _, addr := range acc {
		if bc != nil {
			header := bc.GetHeaderByHash(bhash)
			st, err = state.New(header.Root, state.NewDatabase(db), nil)
		} else {
			header := lc.GetHeaderByHash(bhash)
			st = light.NewState(ctx, header, lc.Odr())
		}
		if err == nil {
			bal := st.GetBalance(addr)
			rlp, _ := rlp.EncodeToBytes(bal)
			res = append(res, rlp...)
		}
	}
	return res
}

func TestOdrContractCallLes2(t *testing.T) { testOdr(t, 2, 2, true, odrContractCall) }
func TestOdrContractCallLes3(t *testing.T) { testOdr(t, 3, 2, true, odrContractCall) }

type callmsg struct {
	types.Message
}

func (callmsg) CheckNonce() bool { return false }

func odrContractCall(ctx context.Context, db ethdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	data := common.Hex2Bytes("60CD26850000000000000000000000000000000000000000000000000000000000000000")

	var res []byte
	for i := 0; i < 3; i++ {
		data[35] = byte(i)
		if bc != nil {
			header := bc.GetHeaderByHash(bhash)
			statedb, err := state.New(header.Root, state.NewDatabase(db), nil)

			if err == nil {
				from := statedb.GetOrNewStateObject(bankAddr)
				from.SetBalance(math.MaxBig256)

				msg := callmsg{types.NewMessage(from.Address(), &testContractAddr, 0, new(big.Int), 100000, new(big.Int), data, false)}

				context := core.NewEVMBlockContext(header, bc, nil)
				txContext := core.NewEVMTxContext(msg)
				vmenv := vm.NewEVM(context, txContext, statedb, config, vm.Config{})

				//vmenv := core.NewEnv(statedb, config, bc, msg, header, vm.Config{})
				gp := new(core.GasPool).AddGas(math.MaxUint64)
				result, _ := core.ApplyMessage(vmenv, msg, gp)
				res = append(res, result.Return()...)
			}
		} else {
			header := lc.GetHeaderByHash(bhash)
			state := light.NewState(ctx, header, lc.Odr())
			state.SetBalance(bankAddr, math.MaxBig256)
			msg := callmsg{types.NewMessage(bankAddr, &testContractAddr, 0, new(big.Int), 100000, new(big.Int), data, false)}
			context := core.NewEVMBlockContext(header, lc, nil)
			txContext := core.NewEVMTxContext(msg)
			vmenv := vm.NewEVM(context, txContext, state, config, vm.Config{})
			gp := new(core.GasPool).AddGas(math.MaxUint64)
			result, _ := core.ApplyMessage(vmenv, msg, gp)
			if state.Error() == nil {
				res = append(res, result.Return()...)
			}
		}
	}
	return res
}

func TestOdrTxStatusLes2(t *testing.T) { testOdr(t, 2, 1, false, odrTxStatus) }
func TestOdrTxStatusLes3(t *testing.T) { testOdr(t, 3, 1, false, odrTxStatus) }

func odrTxStatus(ctx context.Context, db ethdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	var txs types.Transactions
	if bc != nil {
		block := bc.GetBlockByHash(bhash)
		txs = block.Transactions()
	} else {
		if block, _ := lc.GetBlockByHash(ctx, bhash); block != nil {
			btxs := block.Transactions()
			txs = make(types.Transactions, len(btxs))
			for i, tx := range btxs {
				var err error
				txs[i], _, _, _, err = light.GetTransaction(ctx, lc.Odr(), tx.Hash())
				if err != nil {
					return nil
				}
			}
		}
	}
	rlp, _ := rlp.EncodeToBytes(txs)
	return rlp
}

// testOdr tests odr requests whose validation guaranteed by block headers.
func testOdr(t *testing.T, protocol int, expFail uint64, checkCached bool, fn odrTestFn) {
	// Assemble the test environment
	server, client, tearDown := newClientServerEnv(t, 4, protocol, nil, nil, 0, false, true, true)
	defer tearDown()

	// Ensure the client has synced all necessary data.
	clientHead := client.handler.backend.blockchain.CurrentHeader()
	if clientHead.Number.Uint64() != 4 {
		t.Fatalf("Failed to sync the chain with server, head: %v", clientHead.Number.Uint64())
	}
	// Disable the mechanism that we will wait a few time for request
	// even there is no suitable peer to send right now.
	waitForPeers = 0

	test := func(expFail uint64) {
		// Mark this as a helper to put the failures at the correct lines
		t.Helper()

		for i := uint64(0); i <= server.handler.blockchain.CurrentHeader().Number.Uint64(); i++ {
			bhash := rawdb.ReadCanonicalHash(server.db, i)
			b1 := fn(light.NoOdr, server.db, server.handler.server.chainConfig, server.handler.blockchain, nil, bhash)

			// Set the timeout as 1 second here, ensure there is enough time
			// for travis to make the action.
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			b2 := fn(ctx, client.db, client.handler.backend.chainConfig, nil, client.handler.backend.blockchain, bhash)
			cancel()

			eq := bytes.Equal(b1, b2)
			exp := i < expFail
			if exp && !eq {
				t.Fatalf("odr mismatch: have %x, want %x", b2, b1)
			}
			if !exp && eq {
				t.Fatalf("unexpected odr match")
			}
		}
	}

	// expect retrievals to fail (except genesis block) without a les peer
	client.handler.backend.peers.lock.Lock()
	client.peer.speer.hasBlockHook = func(common.Hash, uint64, bool) bool { return false }
	client.handler.backend.peers.lock.Unlock()
	test(expFail)

	// expect all retrievals to pass
	client.handler.backend.peers.lock.Lock()
	client.peer.speer.hasBlockHook = func(common.Hash, uint64, bool) bool { return true }
	client.handler.backend.peers.lock.Unlock()
	test(5)

	// still expect all retrievals to pass, now data should be cached locally
	if checkCached {
		client.handler.backend.peers.unregister(client.peer.speer.id)
		time.Sleep(time.Millisecond * 10) // ensure that all peerSetNotify callbacks are executed
		test(5)
	}
}
