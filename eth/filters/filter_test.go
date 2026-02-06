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

package filters

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts/abi"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rpc"
)

func makeReceipt(addr common.Address) *types.Receipt {
	receipt := types.NewReceipt(nil, false, 0)
	receipt.Logs = []*types.Log{
		{Address: addr},
	}
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt
}

func BenchmarkFilters(b *testing.B) {
	dir := b.TempDir()

	var (
		db, _   = rawdb.NewLevelDBDatabase(dir, 0, 0, "", false)
		_, sys  = newTestFilterSystem(b, db, Config{})
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = common.BytesToAddress([]byte("jeff"))
		addr3   = common.BytesToAddress([]byte("ethereum"))
		addr4   = common.BytesToAddress([]byte("random addresses please"))

		gspec = core.Genesis{
			Alloc:   types.GenesisAlloc{addr1: {Balance: big.NewInt(1000000)}},
			BaseFee: big.NewInt(params.InitialBaseFee),
			Config:  params.TestChainConfig,
		}
		genesis = gspec.ToBlock()
	)
	defer db.Close()

	gspec.MustCommit(db)

	chain, receipts := core.GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 100010, func(i int, gen *core.BlockGen) {
		switch i {
		case 2403:
			receipt := makeReceipt(addr1)
			gen.AddUncheckedReceipt(receipt)
		case 1034:
			receipt := makeReceipt(addr2)
			gen.AddUncheckedReceipt(receipt)
		case 34:
			receipt := makeReceipt(addr3)
			gen.AddUncheckedReceipt(receipt)
		case 99999:
			receipt := makeReceipt(addr4)
			gen.AddUncheckedReceipt(receipt)
		}
	})
	for i, block := range chain {
		rawdb.WriteBlock(db, block)
		rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
		rawdb.WriteHeadBlockHash(db, block.Hash())
		rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), receipts[i])
	}
	b.ResetTimer()

	filter := sys.NewRangeFilter(0, -1, []common.Address{addr1, addr2, addr3, addr4}, nil, 0)

	for i := 0; i < b.N; i++ {
		logs, _ := filter.Logs(context.Background())
		if len(logs) != 4 {
			b.Fatal("expected 4 logs, got", len(logs))
		}
	}
}

func TestFilters(t *testing.T) {
	config := *params.TestChainConfig
	config.Eip1559Block = big.NewInt(0)

	var (
		db     = rawdb.NewMemoryDatabase()
		_, sys = newTestFilterSystem(t, db, Config{})
		// Sender account
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr    = crypto.PubkeyToAddress(key1.PublicKey)
		signer  = types.NewLondonSigner(big.NewInt(1))
		// Logging contract
		contract  = common.Address{0xfe}
		contract2 = common.Address{0xff}
		abiStr    = `[{"inputs":[],"name":"log0","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"t1","type":"uint256"}],"name":"log1","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"t1","type":"uint256"},{"internalType":"uint256","name":"t2","type":"uint256"}],"name":"log2","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"t1","type":"uint256"},{"internalType":"uint256","name":"t2","type":"uint256"},{"internalType":"uint256","name":"t3","type":"uint256"}],"name":"log3","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"t1","type":"uint256"},{"internalType":"uint256","name":"t2","type":"uint256"},{"internalType":"uint256","name":"t3","type":"uint256"},{"internalType":"uint256","name":"t4","type":"uint256"}],"name":"log4","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
		/*
			// SPDX-License-Identifier: GPL-3.0
			pragma solidity >=0.7.0 <0.9.0;

			contract Logger {
			    function log0() external {
			        assembly {
			            log0(0, 0)
			        }
			    }

			    function log1(uint t1) external {
			        assembly {
			            log1(0, 0, t1)
			        }
			    }

			    function log2(uint t1, uint t2) external {
			        assembly {
			            log2(0, 0, t1, t2)
			        }
			    }

			    function log3(uint t1, uint t2, uint t3) external {
			        assembly {
			            log3(0, 0, t1, t2, t3)
			        }
			    }

			    function log4(uint t1, uint t2, uint t3, uint t4) external {
			        assembly {
			            log4(0, 0, t1, t2, t3, t4)
			        }
			    }
			}
		*/
		bytecode = common.FromHex("608060405234801561001057600080fd5b50600436106100575760003560e01c80630aa731851461005c5780632a4c08961461006657806378b9a1f314610082578063c670f8641461009e578063c683d6a3146100ba575b600080fd5b6100646100d6565b005b610080600480360381019061007b9190610143565b6100dc565b005b61009c60048036038101906100979190610196565b6100e8565b005b6100b860048036038101906100b391906101d6565b6100f2565b005b6100d460048036038101906100cf9190610203565b6100fa565b005b600080a0565b808284600080a3505050565b8082600080a25050565b80600080a150565b80828486600080a450505050565b600080fd5b6000819050919050565b6101208161010d565b811461012b57600080fd5b50565b60008135905061013d81610117565b92915050565b60008060006060848603121561015c5761015b610108565b5b600061016a8682870161012e565b935050602061017b8682870161012e565b925050604061018c8682870161012e565b9150509250925092565b600080604083850312156101ad576101ac610108565b5b60006101bb8582860161012e565b92505060206101cc8582860161012e565b9150509250929050565b6000602082840312156101ec576101eb610108565b5b60006101fa8482850161012e565b91505092915050565b6000806000806080858703121561021d5761021c610108565b5b600061022b8782880161012e565b945050602061023c8782880161012e565b935050604061024d8782880161012e565b925050606061025e8782880161012e565b9150509295919450925056fea264697066735822122073a4b156f487e59970dc1ef449cc0d51467268f676033a17188edafcee861f9864736f6c63430008110033")

		hash1 = common.BytesToHash([]byte("topic1"))
		hash2 = common.BytesToHash([]byte("topic2"))
		hash3 = common.BytesToHash([]byte("topic3"))
		hash4 = common.BytesToHash([]byte("topic4"))
		hash5 = common.BytesToHash([]byte("topic5"))

		gspec = &core.Genesis{
			Config: &config,
			Alloc: types.GenesisAlloc{
				addr:      {Balance: big.NewInt(0).Mul(big.NewInt(100), big.NewInt(params.Ether))},
				contract:  {Balance: big.NewInt(0), Code: bytecode},
				contract2: {Balance: big.NewInt(0), Code: bytecode},
			},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		genesis = gspec.ToBlock()
	)
	defer db.Close()

	gspec.MustCommit(db)

	contractABI, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		t.Fatal(err)
	}

	chain, _ := core.GenerateChain(&config, genesis, ethash.NewFaker(), db, 1000, func(i int, gen *core.BlockGen) {
		switch i {
		case 1:
			data, err := contractABI.Pack("log1", hash1.Big())
			if err != nil {
				t.Fatal(err)
			}
			tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{
				Nonce:    0,
				GasPrice: gen.BaseFee(),
				Gas:      30000,
				To:       &contract,
				Data:     data,
			}), signer, key1)
			gen.AddTx(tx)
			tx2, _ := types.SignTx(types.NewTx(&types.LegacyTx{
				Nonce:    1,
				GasPrice: gen.BaseFee(),
				Gas:      30000,
				To:       &contract2,
				Data:     data,
			}), signer, key1)
			gen.AddTx(tx2)
		case 2:
			data, err := contractABI.Pack("log2", hash2.Big(), hash1.Big())
			if err != nil {
				t.Fatal(err)
			}
			tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{
				Nonce:    2,
				GasPrice: gen.BaseFee(),
				Gas:      30000,
				To:       &contract,
				Data:     data,
			}), signer, key1)
			gen.AddTx(tx)
		case 998:
			data, err := contractABI.Pack("log1", hash3.Big())
			if err != nil {
				t.Fatal(err)
			}
			tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{
				Nonce:    3,
				GasPrice: gen.BaseFee(),
				Gas:      30000,
				To:       &contract2,
				Data:     data,
			}), signer, key1)
			gen.AddTx(tx)
		case 999:
			data, err := contractABI.Pack("log1", hash4.Big())
			if err != nil {
				t.Fatal(err)
			}
			tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{
				Nonce:    4,
				GasPrice: gen.BaseFee(),
				Gas:      30000,
				To:       &contract,
				Data:     data,
			}), signer, key1)
			gen.AddTx(tx)
		}
	})

	bc, err := core.NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = bc.InsertChain(chain)
	if err != nil {
		t.Fatal(err)
	}

	// TODO(daniel): ref PR #27135
	// Set block 998 as Finalized (-3)
	// bc.SetFinalized(chain[998].Header())

	// Generate pending block
	pchain, preceipts := core.GenerateChain(gspec.Config, chain[len(chain)-1], ethash.NewFaker(), db, 1, func(i int, gen *core.BlockGen) {
		data, err := contractABI.Pack("log1", hash5.Big())
		if err != nil {
			t.Fatal(err)
		}
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{
			Nonce:    5,
			GasPrice: gen.BaseFee(),
			Gas:      30000,
			To:       &contract,
			Data:     data,
		}), signer, key1)
		gen.AddTx(tx)
	})
	sys.backend.(*testBackend).pendingBlock = pchain[0]
	sys.backend.(*testBackend).pendingReceipts = preceipts[0]

	for i, tc := range []struct {
		f    *Filter
		want string
		err  string
	}{
		{
			f:    sys.NewBlockFilter(chain[2].Hash(), []common.Address{contract}, nil),
			want: `[{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696332","0x0000000000000000000000000000000000000000000000000000746f70696331"],"data":"0x","blockNumber":"0x3","transactionHash":"0xac8b8343d69a5c46fef5af7158f42a389bc84093a88e97e184cc4263cf85dc54","transactionIndex":"0x0","blockHash":"0x6dca03904b22bf701dae59c7135dc3e0b578bd4c577d1c111d9d97776090ae09","logIndex":"0x0","removed":false}]`,
		}, {
			f:    sys.NewRangeFilter(0, int64(rpc.LatestBlockNumber), []common.Address{contract}, [][]common.Hash{{hash1, hash2, hash3, hash4}}, 0),
			want: `[{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696331"],"data":"0x","blockNumber":"0x2","transactionHash":"0x3ebf3c1ea6e0282cfe845f273b2cc7e97f02145e6d0577641cd1a82d532a19c1","transactionIndex":"0x0","blockHash":"0x0f4dea85fc816b6fd5eb4a0ba86eb00cc7d3397785336870bebd27f181a722da","logIndex":"0x0","removed":false},{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696332","0x0000000000000000000000000000000000000000000000000000746f70696331"],"data":"0x","blockNumber":"0x3","transactionHash":"0xac8b8343d69a5c46fef5af7158f42a389bc84093a88e97e184cc4263cf85dc54","transactionIndex":"0x0","blockHash":"0x6dca03904b22bf701dae59c7135dc3e0b578bd4c577d1c111d9d97776090ae09","logIndex":"0x0","removed":false},{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696334"],"data":"0x","blockNumber":"0x3e8","transactionHash":"0x21fd39694cbcc8cc5046b3b7d5200101edf9c85218da613a8851eb5e3d195241","transactionIndex":"0x0","blockHash":"0x8a956d79ca6468ff23c97615a4aa24a55bdaff78767ee28d3e2e02ecb407a0de","logIndex":"0x0","removed":false}]`,
		}, {
			f: sys.NewRangeFilter(900, 999, []common.Address{contract}, [][]common.Hash{{hash3}}, 0),
		}, {
			f:    sys.NewRangeFilter(990, int64(rpc.LatestBlockNumber), []common.Address{contract2}, [][]common.Hash{{hash3}}, 0),
			want: `[{"address":"0xff00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696333"],"data":"0x","blockNumber":"0x3e7","transactionHash":"0x7c42465b2bd34fb8e87bab0001ab97c47b6d57c65f17efe43e81461fef2f05a4","transactionIndex":"0x0","blockHash":"0x5112c98f7517100552d30734c46356db10494c90bb3bd0af90ef9ace2e692ad2","logIndex":"0x0","removed":false}]`,
		}, {
			f:    sys.NewRangeFilter(1, 10, []common.Address{contract}, [][]common.Hash{{hash2}, {hash1}}, 0),
			want: `[{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696332","0x0000000000000000000000000000000000000000000000000000746f70696331"],"data":"0x","blockNumber":"0x3","transactionHash":"0xac8b8343d69a5c46fef5af7158f42a389bc84093a88e97e184cc4263cf85dc54","transactionIndex":"0x0","blockHash":"0x6dca03904b22bf701dae59c7135dc3e0b578bd4c577d1c111d9d97776090ae09","logIndex":"0x0","removed":false}]`,
		}, {
			f:    sys.NewRangeFilter(1, 10, nil, [][]common.Hash{{hash1, hash2}}, 0),
			want: `[{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696331"],"data":"0x","blockNumber":"0x2","transactionHash":"0x3ebf3c1ea6e0282cfe845f273b2cc7e97f02145e6d0577641cd1a82d532a19c1","transactionIndex":"0x0","blockHash":"0x0f4dea85fc816b6fd5eb4a0ba86eb00cc7d3397785336870bebd27f181a722da","logIndex":"0x0","removed":false},{"address":"0xff00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696331"],"data":"0x","blockNumber":"0x2","transactionHash":"0x55b3a22ae885f9441ff8bd98fbfe54cc1b84799606aca159fac8d7a56551e426","transactionIndex":"0x1","blockHash":"0x0f4dea85fc816b6fd5eb4a0ba86eb00cc7d3397785336870bebd27f181a722da","logIndex":"0x1","removed":false},{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696332","0x0000000000000000000000000000000000000000000000000000746f70696331"],"data":"0x","blockNumber":"0x3","transactionHash":"0xac8b8343d69a5c46fef5af7158f42a389bc84093a88e97e184cc4263cf85dc54","transactionIndex":"0x0","blockHash":"0x6dca03904b22bf701dae59c7135dc3e0b578bd4c577d1c111d9d97776090ae09","logIndex":"0x0","removed":false}]`,
		}, {
			f: sys.NewRangeFilter(0, int64(rpc.LatestBlockNumber), nil, [][]common.Hash{{common.BytesToHash([]byte("fail"))}}, 0),
		}, {
			f: sys.NewRangeFilter(0, int64(rpc.LatestBlockNumber), []common.Address{common.BytesToAddress([]byte("failmenow"))}, nil, 0),
		}, {
			f: sys.NewRangeFilter(0, int64(rpc.LatestBlockNumber), nil, [][]common.Hash{{common.BytesToHash([]byte("fail"))}, {hash1}}, 0),
		}, {
			f:    sys.NewRangeFilter(int64(rpc.LatestBlockNumber), int64(rpc.LatestBlockNumber), nil, nil, 0),
			want: `[{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696334"],"data":"0x","blockNumber":"0x3e8","transactionHash":"0x21fd39694cbcc8cc5046b3b7d5200101edf9c85218da613a8851eb5e3d195241","transactionIndex":"0x0","blockHash":"0x8a956d79ca6468ff23c97615a4aa24a55bdaff78767ee28d3e2e02ecb407a0de","logIndex":"0x0","removed":false}]`,
		}, {
			f:   sys.NewRangeFilter(int64(rpc.FinalizedBlockNumber), int64(rpc.LatestBlockNumber), nil, nil, 0),
			err: "committed header not found",
		}, {
			f:   sys.NewRangeFilter(int64(rpc.FinalizedBlockNumber), int64(rpc.FinalizedBlockNumber), nil, nil, 0),
			err: "committed header not found",
		}, {
			f:   sys.NewRangeFilter(int64(rpc.LatestBlockNumber), int64(rpc.FinalizedBlockNumber), nil, nil, 0),
			err: "committed header not found",
		}, {
			f:    sys.NewRangeFilter(int64(rpc.PendingBlockNumber), int64(rpc.PendingBlockNumber), nil, nil, 0),
			want: `[{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696335"],"data":"0x","blockNumber":"0x3e9","transactionHash":"0x87d02f2dddb1941ff179ae5d5fbb123afb9c5f71220045bc1c48f3872be24d4a","transactionIndex":"0x0","blockHash":"0x5f2b35a350840476a43aa23c6ea031d6db277aef337775ebb8421df64f17723f","logIndex":"0x0","removed":false}]`,
		}, {
			f:    sys.NewRangeFilter(int64(rpc.LatestBlockNumber), int64(rpc.PendingBlockNumber), nil, nil, 0),
			want: `[{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696334"],"data":"0x","blockNumber":"0x3e8","transactionHash":"0x21fd39694cbcc8cc5046b3b7d5200101edf9c85218da613a8851eb5e3d195241","transactionIndex":"0x0","blockHash":"0x8a956d79ca6468ff23c97615a4aa24a55bdaff78767ee28d3e2e02ecb407a0de","logIndex":"0x0","removed":false},{"address":"0xfe00000000000000000000000000000000000000","topics":["0x0000000000000000000000000000000000000000000000000000746f70696335"],"data":"0x","blockNumber":"0x3e9","transactionHash":"0x87d02f2dddb1941ff179ae5d5fbb123afb9c5f71220045bc1c48f3872be24d4a","transactionIndex":"0x0","blockHash":"0x5f2b35a350840476a43aa23c6ea031d6db277aef337775ebb8421df64f17723f","logIndex":"0x0","removed":false}]`,
		}, {
			f:   sys.NewRangeFilter(int64(rpc.PendingBlockNumber), int64(rpc.LatestBlockNumber), nil, nil, 0),
			err: "invalid block range",
		},
	} {
		logs, err := tc.f.Logs(context.Background())
		if err == nil && tc.err != "" {
			t.Fatalf("test %d, expected error %q, got nil", i, tc.err)
		} else if err != nil && err.Error() != tc.err {
			t.Fatalf("test %d, expected error %q, got %q", i, tc.err, err.Error())
		}
		if tc.want == "" && len(logs) == 0 {
			continue
		}
		have, err := json.Marshal(logs)
		if err != nil {
			t.Fatal(err)
		}
		if string(have) != tc.want {
			t.Fatalf("test %d, have:\n%s\nwant:\n%s", i, have, tc.want)
		}
	}

	t.Run("timeout", func(t *testing.T) {
		f := sys.NewRangeFilter(0, -1, nil, nil, 0)
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
		defer cancel()
		_, err := f.Logs(ctx)
		if err == nil {
			t.Fatal("expected error")
		}
		if err != context.DeadlineExceeded {
			t.Fatalf("expected context.DeadlineExceeded, got %v", err)
		}
	})
}

func TestRangeLimit(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	_, sys := newTestFilterSystem(t, db, Config{})
	defer db.Close()

	gspec := &core.Genesis{
		Alloc:   types.GenesisAlloc{},
		BaseFee: big.NewInt(params.InitialBaseFee),
		Config:  params.TestChainConfig,
	}
	genesis := gspec.MustCommit(db)
	chain, _ := core.GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 10, func(i int, gen *core.BlockGen) {})
	bc, err := core.NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = bc.InsertChain(chain)
	if err != nil {
		t.Fatal(err)
	}

	// Set rangeLimit to 5, but request a range of 9 (end - begin = 9, from 0 to 9)
	filter := sys.NewRangeFilter(0, 9, nil, nil, 5)
	_, err = filter.Logs(context.Background())
	if err == nil || !strings.Contains(err.Error(), "exceed maximum block range") {
		t.Fatalf("expected range limit error, got %v", err)
	}
}
