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

type chtTestFn func(ctx context.Context, bc *core.BlockChain, lc *light.LightChain, number uint64) []byte

func TestChtGetHeadersLes1(t *testing.T) { testCht(t, 1, chtGetHeader) }

func TestChtGetHeadersLes2(t *testing.T) { testCht(t, 2, chtGetHeader) }

func chtGetHeader(ctx context.Context, bc *core.BlockChain, lc *light.LightChain, number uint64) []byte {
	var header *types.Header
	if bc != nil {
		header = bc.GetHeaderByNumber(number)
	} else {
		header, _ = lc.GetHeaderByNumberOdr(ctx, number)
	}
	if header == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(header)
	return rlp
}

type odrTestFn func(ctx context.Context, db ethdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte

func TestOdrGetBlockLes1(t *testing.T) { testOdr(t, 1, 1, odrGetBlock) }

func TestOdrGetBlockLes2(t *testing.T) { testOdr(t, 2, 1, odrGetBlock) }

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

func TestOdrGetReceiptsLes1(t *testing.T) { testOdr(t, 1, 1, odrGetReceipts) }

func TestOdrGetReceiptsLes2(t *testing.T) { testOdr(t, 2, 1, odrGetReceipts) }

func odrGetReceipts(ctx context.Context, db ethdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	var receipts types.Receipts
	if bc != nil {
		if number := rawdb.ReadHeaderNumber(db, bhash); number != nil {
			receipts = rawdb.ReadReceipts(db, bhash, *number)
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

func TestOdrAccountsLes1(t *testing.T) { testOdr(t, 1, 1, odrAccounts) }

func TestOdrAccountsLes2(t *testing.T) { testOdr(t, 2, 1, odrAccounts) }

func odrAccounts(ctx context.Context, db ethdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	dummyAddr := common.HexToAddress("1234567812345678123456781234567812345678")
	acc := []common.Address{testBankAddress, acc1Addr, acc2Addr, dummyAddr}

	var (
		res []byte
		st  *state.StateDB
		err error
	)
	for _, addr := range acc {
		if bc != nil {
			header := bc.GetHeaderByHash(bhash)
			st, err = state.New(header.Root, state.NewDatabase(db))
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

func TestOdrContractCallLes1(t *testing.T) { testOdr(t, 1, 2, odrContractCall) }

func TestOdrContractCallLes2(t *testing.T) { testOdr(t, 2, 2, odrContractCall) }

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
			statedb, err := state.New(header.Root, state.NewDatabase(db))

			if err == nil {
				from := statedb.GetOrNewStateObject(testBankAddress)
				from.SetBalance(math.MaxBig256)

				msg := callmsg{types.NewMessage(from.Address(), &testContractAddr, 0, new(big.Int), 100000, new(big.Int), data, false)}

				context := core.NewEVMContext(msg, header, bc, nil)
				vmenv := vm.NewEVM(context, statedb, config, vm.Config{})

				//vmenv := core.NewEnv(statedb, config, bc, msg, header, vm.Config{})
				gp := new(core.GasPool).AddGas(math.MaxUint64)
				ret, _, _, _ := core.ApplyMessage(vmenv, msg, gp)
				res = append(res, ret...)
			}
		} else {
			header := lc.GetHeaderByHash(bhash)
			state := light.NewState(ctx, header, lc.Odr())
			state.SetBalance(testBankAddress, math.MaxBig256)
			msg := callmsg{types.NewMessage(testBankAddress, &testContractAddr, 0, new(big.Int), 100000, new(big.Int), data, false)}
			context := core.NewEVMContext(msg, header, lc, nil)
			vmenv := vm.NewEVM(context, state, config, vm.Config{})
			gp := new(core.GasPool).AddGas(math.MaxUint64)
			ret, _, _, _ := core.ApplyMessage(vmenv, msg, gp)
			if state.Error() == nil {
				res = append(res, ret...)
			}
		}
	}
	return res
}

// testCht tests cht requests whose validation guaranteed by calculated cht root.
func testCht(t *testing.T, protocol int, fn chtTestFn) {
	// Assemble the test environment
	config := light.TestServerIndexerConfig
	waitIndexers := func(cIndexer, bIndexer, btIndexer *core.ChainIndexer) {
		for {
			cs, _, _ := cIndexer.Sections()
			bs, _, _ := bIndexer.Sections()
			bts, _, _ := btIndexer.Sections()
			if cs >= 8 && bs >= 8 && bts >= 1 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	server, client, tearDown := newClientServerEnv(t, int(config.ChtSize*8+config.ChtConfirm), protocol, waitIndexers, false)
	defer func() {
		if tearDown != nil {
			tearDown()
		}
	}()

	// Add trusted checkpoint for client side indexers.
	cs, _, head := server.chtIndexer.Sections()
	light.StoreChtRoot(client.db, cs/8-1, head, light.GetChtRoot(server.db, cs-1, head))
	client.chtIndexer.AddKnownSectionHead(cs/8-1, head)
	bts, _, head := server.bloomTrieIndexer.Sections()
	light.StoreBloomTrieRoot(client.db, bts-1, head, light.GetBloomTrieRoot(server.db, bts-1, head))
	client.bloomTrieIndexer.AddKnownSectionHead(bts-1, head)

	// Create connected peer pair.
	peer, err1, lPeer, err2 := newTestPeerPair("peer", protocol, server.pm, client.pm)
	select {
	case <-time.After(time.Millisecond * 100):
	case err := <-err1:
		t.Fatalf("peer 1 handshake error: %v", err)
	case err := <-err2:
		t.Fatalf("peer 2 handshake error: %v", err)
	}
	server.rPeer, client.rPeer = peer, lPeer

	test := func() {
		for i := uint64(0); i <= config.ChtSize*8-1; i++ {
			h1 := fn(light.NoOdr, server.pm.blockchain.(*core.BlockChain), nil, i)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			h2 := fn(ctx, nil, client.pm.blockchain.(*light.LightChain), i)
			if !bytes.Equal(h1, h2) {
				t.Error("cht mismatch")
			}
			cancel()
		}
	}
	test()
}

// testOdr tests odr requests whose validation guaranteed by block headers.
func testOdr(t *testing.T, protocol int, expFail uint64, fn odrTestFn) {
	// Assemble the test environment
	server, client, tearDown := newClientServerEnv(t, 4, protocol, nil, true)
	defer tearDown()
	client.pm.synchronise(client.rPeer)

	test := func(expFail uint64) {
		for i := uint64(0); i <= server.pm.blockchain.CurrentHeader().Number.Uint64(); i++ {
			bhash := rawdb.ReadCanonicalHash(server.db, i)
			b1 := fn(light.NoOdr, server.db, server.pm.chainConfig, server.pm.blockchain.(*core.BlockChain), nil, bhash)

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			b2 := fn(ctx, client.db, client.pm.chainConfig, nil, client.pm.blockchain.(*light.LightChain), bhash)

			eq := bytes.Equal(b1, b2)
			exp := i < expFail
			if exp && !eq {
				t.Errorf("odr mismatch")
			}
			if !exp && eq {
				t.Errorf("unexpected odr match")
			}
		}
	}
	// temporarily remove peer to test odr fails
	// expect retrievals to fail (except genesis block) without a les peer
	client.peers.Unregister(client.rPeer.id)
	time.Sleep(time.Millisecond * 10) // ensure that all peerSetNotify callbacks are executed
	test(expFail)
	// expect all retrievals to pass
	client.peers.Register(client.rPeer)
	time.Sleep(time.Millisecond * 10) // ensure that all peerSetNotify callbacks are executed
	client.peers.lock.Lock()
	client.rPeer.hasBlock = func(common.Hash, uint64) bool { return true }
	client.peers.lock.Unlock()
	test(5)
	// still expect all retrievals to pass, now data should be cached locally
	client.peers.Unregister(client.rPeer.id)
	time.Sleep(time.Millisecond * 10) // ensure that all peerSetNotify callbacks are executed
	test(5)
}
