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

package light

import (
	"bytes"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/net/context"
)

var (
	testBankKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
	testBankFunds   = big.NewInt(100000000)

	acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc1Addr   = crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr   = crypto.PubkeyToAddress(acc2Key.PublicKey)

	testContractCode = common.Hex2Bytes("606060405260cc8060106000396000f360606040526000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146041578063c16431b914606b57603f565b005b6055600480803590602001909190505060a9565b6040518082815260200191505060405180910390f35b60886004808035906020019091908035906020019091905050608a565b005b80600060005083606481101560025790900160005b50819055505b5050565b6000600060005082606481101560025790900160005b5054905060c7565b91905056")
	testContractAddr common.Address
)

type testOdr struct {
	OdrBackend
	sdb, ldb ethdb.Database
	disable  bool
}

func (odr *testOdr) Database() ethdb.Database {
	return odr.ldb
}

var ErrOdrDisabled = errors.New("ODR disabled")

func (odr *testOdr) Retrieve(ctx context.Context, req OdrRequest) error {
	if odr.disable {
		return ErrOdrDisabled
	}
	switch req := req.(type) {
	case *BlockRequest:
		req.Rlp = core.GetBodyRLP(odr.sdb, req.Hash, core.GetBlockNumber(odr.sdb, req.Hash))
	case *ReceiptsRequest:
		req.Receipts = core.GetBlockReceipts(odr.sdb, req.Hash, core.GetBlockNumber(odr.sdb, req.Hash))
	case *TrieRequest:
		t, _ := trie.New(req.Id.Root, odr.sdb)
		req.Proof = t.Prove(req.Key)
	case *CodeRequest:
		req.Data, _ = odr.sdb.Get(req.Hash[:])
	}
	req.StoreResult(odr.ldb)
	return nil
}

type odrTestFn func(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) []byte

func TestOdrGetBlockLes1(t *testing.T) { testChainOdr(t, 1, 1, odrGetBlock) }

func odrGetBlock(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) []byte {
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

func TestOdrGetReceiptsLes1(t *testing.T) { testChainOdr(t, 1, 1, odrGetReceipts) }

func odrGetReceipts(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) []byte {
	var receipts types.Receipts
	if bc != nil {
		receipts = core.GetBlockReceipts(db, bhash, core.GetBlockNumber(db, bhash))
	} else {
		receipts, _ = GetBlockReceipts(ctx, lc.Odr(), bhash, core.GetBlockNumber(db, bhash))
	}
	if receipts == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(receipts)
	return rlp
}

func TestOdrAccountsLes1(t *testing.T) { testChainOdr(t, 1, 1, odrAccounts) }

func odrAccounts(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) []byte {
	dummyAddr := common.HexToAddress("1234567812345678123456781234567812345678")
	acc := []common.Address{testBankAddress, acc1Addr, acc2Addr, dummyAddr}

	var res []byte
	for _, addr := range acc {
		if bc != nil {
			header := bc.GetHeaderByHash(bhash)
			st, err := state.New(header.Root, db)
			if err == nil {
				bal := st.GetBalance(addr)
				rlp, _ := rlp.EncodeToBytes(bal)
				res = append(res, rlp...)
			}
		} else {
			header := lc.GetHeaderByHash(bhash)
			st := NewLightState(StateTrieID(header), lc.Odr())
			bal, err := st.GetBalance(ctx, addr)
			if err == nil {
				rlp, _ := rlp.EncodeToBytes(bal)
				res = append(res, rlp...)
			}
		}
	}

	return res
}

func TestOdrContractCallLes1(t *testing.T) { testChainOdr(t, 1, 2, odrContractCall) }

type callmsg struct {
	types.Message
}

func (callmsg) CheckNonce() bool { return false }

func odrContractCall(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) []byte {
	data := common.Hex2Bytes("60CD26850000000000000000000000000000000000000000000000000000000000000000")

	config := params.TestChainConfig

	var res []byte
	for i := 0; i < 3; i++ {
		data[35] = byte(i)
		if bc != nil {
			header := bc.GetHeaderByHash(bhash)
			statedb, err := state.New(header.Root, db)
			if err == nil {
				from := statedb.GetOrNewStateObject(testBankAddress)
				from.SetBalance(common.MaxBig)

				msg := callmsg{types.NewMessage(from.Address(), &testContractAddr, 0, new(big.Int), big.NewInt(1000000), new(big.Int), data, false)}

				context := core.NewEVMContext(msg, header, bc)
				vmenv := vm.NewEVM(context, statedb, config, vm.Config{})

				gp := new(core.GasPool).AddGas(common.MaxBig)
				ret, _, _ := core.ApplyMessage(vmenv, msg, gp)
				res = append(res, ret...)
			}
		} else {
			header := lc.GetHeaderByHash(bhash)
			state := NewLightState(StateTrieID(header), lc.Odr())
			vmstate := NewVMState(ctx, state)
			from, err := state.GetOrNewStateObject(ctx, testBankAddress)
			if err == nil {
				from.SetBalance(common.MaxBig)

				msg := callmsg{types.NewMessage(from.Address(), &testContractAddr, 0, new(big.Int), big.NewInt(1000000), new(big.Int), data, false)}
				context := core.NewEVMContext(msg, header, lc)
				vmenv := vm.NewEVM(context, vmstate, config, vm.Config{})
				gp := new(core.GasPool).AddGas(common.MaxBig)
				ret, _, _ := core.ApplyMessage(vmenv, msg, gp)
				if vmstate.Error() == nil {
					res = append(res, ret...)
				}
			}
		}
	}
	return res
}

func testChainGen(i int, block *core.BlockGen) {
	signer := types.HomesteadSigner{}
	switch i {
	case 0:
		// In block 1, the test bank sends account #1 some ether.
		tx, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), acc1Addr, big.NewInt(10000), params.TxGas, nil, nil), signer, testBankKey)
		block.AddTx(tx)
	case 1:
		// In block 2, the test bank sends some more ether to account #1.
		// acc1Addr passes it on to account #2.
		// acc1Addr creates a test contract.
		tx1, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), acc1Addr, big.NewInt(1000), params.TxGas, nil, nil), signer, testBankKey)
		nonce := block.TxNonce(acc1Addr)
		tx2, _ := types.SignTx(types.NewTransaction(nonce, acc2Addr, big.NewInt(1000), params.TxGas, nil, nil), signer, acc1Key)
		nonce++
		tx3, _ := types.SignTx(types.NewContractCreation(nonce, big.NewInt(0), big.NewInt(1000000), big.NewInt(0), testContractCode), signer, acc1Key)
		testContractAddr = crypto.CreateAddress(acc1Addr, nonce)
		block.AddTx(tx1)
		block.AddTx(tx2)
		block.AddTx(tx3)
	case 2:
		// Block 3 is empty but was mined by account #2.
		block.SetCoinbase(acc2Addr)
		block.SetExtra([]byte("yeehaw"))
		data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001")
		tx, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), testContractAddr, big.NewInt(0), big.NewInt(100000), nil, data), signer, testBankKey)
		block.AddTx(tx)
	case 3:
		// Block 4 includes blocks 2 and 3 as uncle headers (with modified extra data).
		b2 := block.PrevBlock(1).Header()
		b2.Extra = []byte("foo")
		block.AddUncle(b2)
		b3 := block.PrevBlock(2).Header()
		b3.Extra = []byte("foo")
		block.AddUncle(b3)
		data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000002")
		tx, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), testContractAddr, big.NewInt(0), big.NewInt(100000), nil, data), signer, testBankKey)
		block.AddTx(tx)
	}
}

func testChainOdr(t *testing.T, protocol int, expFail uint64, fn odrTestFn) {
	var (
		evmux   = new(event.TypeMux)
		pow     = new(core.FakePow)
		sdb, _  = ethdb.NewMemDatabase()
		ldb, _  = ethdb.NewMemDatabase()
		genesis = core.WriteGenesisBlockForTesting(sdb, core.GenesisAccount{Address: testBankAddress, Balance: testBankFunds})
	)
	core.WriteGenesisBlockForTesting(ldb, core.GenesisAccount{Address: testBankAddress, Balance: testBankFunds})
	// Assemble the test environment
	blockchain, _ := core.NewBlockChain(sdb, testChainConfig(), pow, evmux)
	chainConfig := &params.ChainConfig{HomesteadBlock: new(big.Int)}
	gchain, _ := core.GenerateChain(chainConfig, genesis, sdb, 4, testChainGen)
	if _, err := blockchain.InsertChain(gchain); err != nil {
		panic(err)
	}

	odr := &testOdr{sdb: sdb, ldb: ldb}
	lightchain, _ := NewLightChain(odr, testChainConfig(), pow, evmux)
	lightchain.SetValidator(bproc{})
	headers := make([]*types.Header, len(gchain))
	for i, block := range gchain {
		headers[i] = block.Header()
	}
	if _, err := lightchain.InsertHeaderChain(headers, 1); err != nil {
		panic(err)
	}

	test := func(expFail uint64) {
		for i := uint64(0); i <= blockchain.CurrentHeader().Number.Uint64(); i++ {
			bhash := core.GetCanonicalHash(sdb, i)
			b1 := fn(NoOdr, sdb, blockchain, nil, bhash)
			ctx, _ := context.WithTimeout(context.Background(), 200*time.Millisecond)
			b2 := fn(ctx, ldb, nil, lightchain, bhash)
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

	odr.disable = true
	// expect retrievals to fail (except genesis block) without a les peer
	test(expFail)
	odr.disable = false
	// expect all retrievals to pass
	test(5)
	odr.disable = true
	// still expect all retrievals to pass, now data should be cached locally
	test(5)
}
