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
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/net/context"
)

type odrTestFn func(ctx context.Context, db ethdb.Database, config *core.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte

func TestOdrGetBlockLes1(t *testing.T) { testOdr(t, 1, 1, odrGetBlock) }

func odrGetBlock(ctx context.Context, db ethdb.Database, config *core.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
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

func odrGetReceipts(ctx context.Context, db ethdb.Database, config *core.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	var receipts types.Receipts
	if bc != nil {
		receipts = core.GetBlockReceipts(db, bhash, core.GetBlockNumber(db, bhash))
	} else {
		receipts, _ = light.GetBlockReceipts(ctx, lc.Odr(), bhash, core.GetBlockNumber(db, bhash))
	}
	if receipts == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(receipts)
	return rlp
}

func TestOdrAccountsLes1(t *testing.T) { testOdr(t, 1, 1, odrAccounts) }

func odrAccounts(ctx context.Context, db ethdb.Database, config *core.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
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
			st := light.NewLightState(light.StateTrieID(header), lc.Odr())
			bal, err := st.GetBalance(ctx, addr)
			if err == nil {
				rlp, _ := rlp.EncodeToBytes(bal)
				res = append(res, rlp...)
			}
		}
	}

	return res
}

func TestOdrContractCallLes1(t *testing.T) { testOdr(t, 1, 2, odrContractCall) }

// fullcallmsg is the message type used for call transations.
type fullcallmsg struct {
	from          *state.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m fullcallmsg) From() (common.Address, error)         { return m.from.Address(), nil }
func (m fullcallmsg) FromFrontier() (common.Address, error) { return m.from.Address(), nil }
func (m fullcallmsg) Nonce() uint64                         { return 0 }
func (m fullcallmsg) CheckNonce() bool                      { return false }
func (m fullcallmsg) To() *common.Address                   { return m.to }
func (m fullcallmsg) GasPrice() *big.Int                    { return m.gasPrice }
func (m fullcallmsg) Gas() *big.Int                         { return m.gas }
func (m fullcallmsg) Value() *big.Int                       { return m.value }
func (m fullcallmsg) Data() []byte                          { return m.data }

// callmsg is the message type used for call transations.
type lightcallmsg struct {
	from          *light.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m lightcallmsg) From() (common.Address, error)         { return m.from.Address(), nil }
func (m lightcallmsg) FromFrontier() (common.Address, error) { return m.from.Address(), nil }
func (m lightcallmsg) Nonce() uint64                         { return 0 }
func (m lightcallmsg) CheckNonce() bool                      { return false }
func (m lightcallmsg) To() *common.Address                   { return m.to }
func (m lightcallmsg) GasPrice() *big.Int                    { return m.gasPrice }
func (m lightcallmsg) Gas() *big.Int                         { return m.gas }
func (m lightcallmsg) Value() *big.Int                       { return m.value }
func (m lightcallmsg) Data() []byte                          { return m.data }

func odrContractCall(ctx context.Context, db ethdb.Database, config *core.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	data := common.Hex2Bytes("60CD26850000000000000000000000000000000000000000000000000000000000000000")

	var res []byte
	for i := 0; i < 3; i++ {
		data[35] = byte(i)
		if bc != nil {
			header := bc.GetHeaderByHash(bhash)
			statedb, err := state.New(header.Root, db)
			if err == nil {
				from := statedb.GetOrNewStateObject(testBankAddress)
				from.SetBalance(common.MaxBig)

				msg := fullcallmsg{
					from:     from,
					gas:      big.NewInt(100000),
					gasPrice: big.NewInt(0),
					value:    big.NewInt(0),
					data:     data,
					to:       &testContractAddr,
				}

				vmenv := core.NewEnv(statedb, config, bc, msg, header, config.VmConfig)
				gp := new(core.GasPool).AddGas(common.MaxBig)
				ret, _, _ := core.ApplyMessage(vmenv, msg, gp)
				res = append(res, ret...)
			}
		} else {
			header := lc.GetHeaderByHash(bhash)
			state := light.NewLightState(light.StateTrieID(header), lc.Odr())
			from, err := state.GetOrNewStateObject(ctx, testBankAddress)
			if err == nil {
				from.SetBalance(common.MaxBig)

				msg := lightcallmsg{
					from:     from,
					gas:      big.NewInt(100000),
					gasPrice: big.NewInt(0),
					value:    big.NewInt(0),
					data:     data,
					to:       &testContractAddr,
				}

				vmenv := light.NewEnv(ctx, state, config, lc, msg, header, config.VmConfig)
				gp := new(core.GasPool).AddGas(common.MaxBig)
				ret, _, _ := core.ApplyMessage(vmenv, msg, gp)
				if vmenv.Error() == nil {
					res = append(res, ret...)
				}
			}
		}
	}
	return res
}

func testOdr(t *testing.T, protocol int, expFail uint64, fn odrTestFn) {
	// Assemble the test environment
	pm, db, odr := newTestProtocolManagerMust(t, false, 4, testChainGen)
	lpm, ldb, odr := newTestProtocolManagerMust(t, true, 0, nil)
	_, err1, lpeer, err2 := newTestPeerPair("peer", protocol, pm, lpm)
	select {
	case <-time.After(time.Millisecond * 100):
	case err := <-err1:
		t.Fatalf("peer 1 handshake error: %v", err)
	case err := <-err2:
		t.Fatalf("peer 1 handshake error: %v", err)
	}

	lpm.synchronise(lpeer)

	test := func(expFail uint64) {
		for i := uint64(0); i <= pm.blockchain.CurrentHeader().Number.Uint64(); i++ {
			bhash := core.GetCanonicalHash(db, i)
			b1 := fn(light.NoOdr, db, pm.chainConfig, pm.blockchain.(*core.BlockChain), nil, bhash)
			ctx, _ := context.WithTimeout(context.Background(), 200*time.Millisecond)
			b2 := fn(ctx, ldb, lpm.chainConfig, nil, lpm.blockchain.(*light.LightChain), bhash)
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
	odr.UnregisterPeer(lpeer)
	// expect retrievals to fail (except genesis block) without a les peer
	test(expFail)
	odr.RegisterPeer(lpeer)
	// expect all retrievals to pass
	test(5)
	odr.UnregisterPeer(lpeer)
	// still expect all retrievals to pass, now data should be cached locally
	test(5)
}
