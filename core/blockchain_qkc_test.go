// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"errors"
	"maps"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/posw"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// coinbaseTestEngine keeps PoSW's real reward finalization, while the existing
// test wrapper skips header/seal verification. Zero difficulty enables EVM forks.
type coinbaseTestEngine struct{ qkcTestEngine }

func (coinbaseTestEngine) CalcDifficulty(consensus.ChainHeaderReader, uint64, *types.Header) *big.Int {
	return new(big.Int)
}

// TestGetCoinbaseAmount compares the derived amount with actual miner credits.
// The miner never sends transactions or receives transfers/deposit values, so
// its balance delta contains only the real PoSW reward and transaction fees.
func TestGetCoinbaseAmount(t *testing.T) {
	for _, amsterdam := range []bool{false, true} {
		name := "pre-amsterdam"
		if amsterdam {
			name = "amsterdam"
		}
		t.Run(name, func(t *testing.T) {
			config := *params.NewTestQuarkChainConfig(10001, 10001, 10002)
			if amsterdam {
				config.AmsterdamTime = new(uint64)
			}
			// Exercise subsidy rounding, tax and decay as the test chain advances.
			config.QuarkChain.PoSW.CoinbaseAmount = big.NewInt(101)
			config.QuarkChain.PoSW.EpochInterval = 3
			config.QuarkChain.PoSW.DecayFactor = big.NewRat(1, 2)
			key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
			authorityKey, _ := crypto.ToECDSA(common.LeftPadBytes([]byte{2}, 32))
			sender := crypto.PubkeyToAddress(key.PublicKey)
			coinbase, recipient := common.HexToAddress("0xc0ffee"), common.HexToAddress("0xe0a")
			reverter, refunder := common.HexToAddress("0x7e7e7"), common.HexToAddress("0x5703e")
			alloc := withSystemContracts(types.GenesisAlloc{
				sender:    {Balance: big.NewInt(params.Ether)},
				coinbase:  {Balance: big.NewInt(1000)},
				recipient: {Balance: big.NewInt(1)}, // Keep a plain transfer independent of account-creation gas.
				reverter:  {Code: program.New().Op(vm.JUMPDEST).Push(0).Push(0).Op(vm.REVERT).Bytes()},
				refunder:  {Code: program.New().Push(0).Push(0).Op(vm.SSTORE).Bytes(), Storage: map[common.Hash]common.Hash{{}: common.HexToHash("0x01")}},
			})
			maps.Copy(alloc, PoSWStakingContractAllocs(&config))
			genesis := &Genesis{Config: &config, Alloc: alloc, GasLimit: 12_000_000, Difficulty: new(big.Int), ExtraData: (&types.QKCExtra{ChainID: config.ChainID}).Encode()}
			poswEngine, err := posw.New(&config, genesis.ToBlock().Header())
			if err != nil {
				t.Fatal(err)
			}
			defer poswEngine.Close()
			engine := coinbaseTestEngine{qkcTestEngine{poswEngine}}
			sendABI, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"send","inputs":[
				{"type":"uint256"},{"type":"address"},{"type":"uint256"},{"type":"uint64"},{"type":"bool"},{"type":"bytes"}]}]`))
			if err != nil {
				t.Fatal(err)
			}
			cases := []struct {
				name  string
				kinds []string
			}{
				{"empty", nil},
				{"legacy", []string{"legacy"}},
				{"access list", []string{"access list"}},
				{"dynamic fee", []string{"dynamic fee"}},
				{"fee cap limits tip", []string{"fee cap"}},
				{"set code", []string{"set code"}},
				{"outgoing cross shard", []string{"outgoing"}},
				{"incoming deposits", []string{"deposit", "free deposit"}},
				{"failed execution", []string{"failed"}},
				{"gas refund", []string{"refund"}},
				// Two odd fees distinguish per-transaction tax from tax on the sum.
				{"mixed", []string{"deposit", "legacy", "dynamic fee", "failed", "failed"}},
			}
			signer := types.LatestSigner(&config)
			genDB, blocks, _ := GenerateChainWithGenesis(genesis, engine, len(cases), func(i int, b *BlockGen) {
				b.SetCoinbase(coinbase)
				b.SetExtra((&types.QKCExtra{ChainID: config.ChainID}).Encode())
				for _, kind := range cases[i].kinds {
					if kind == "deposit" || kind == "free deposit" {
						// A price below base fee must still be paid in full; zero
						// price covers root-originated deposits without miner fees.
						price := uint64(1)
						if kind == "free deposit" {
							price = 0
						}
						b.AddTx(types.NewTx(xShardTestMove(uint64(i), sender, recipient, 123, price)))
						continue
					}
					tx := &types.DynamicFeeTx{ChainID: config.ChainID, Nonce: b.TxNonce(sender), To: &recipient, Gas: 100_000, GasFeeCap: newGwei(5), GasTipCap: newGwei(2)}
					var data types.TxData = tx
					switch kind {
					case "legacy":
						data = &types.LegacyTx{Nonce: tx.Nonce, To: &recipient, Value: big.NewInt(1), Gas: params.TxGas, GasPrice: newGwei(5)}
					case "access list":
						data = &types.AccessListTx{ChainID: config.ChainID, Nonce: tx.Nonce, To: &recipient, Gas: tx.Gas, GasPrice: newGwei(5), AccessList: types.AccessList{{Address: recipient, StorageKeys: []common.Hash{{1}}}}}
					case "fee cap":
						tx.GasFeeCap, tx.GasTipCap = newGwei(3), newGwei(3)
					case "set code":
						auth, err := types.SignSetCode(authorityKey, types.SetCodeAuthorization{ChainID: *uint256.MustFromBig(config.ChainID), Address: recipient})
						if err != nil {
							t.Fatal(err)
						}
						// Amsterdam charges state gas for the new authority and delegation.
						data = &types.SetCodeTx{ChainID: uint256.MustFromBig(config.ChainID), Nonce: tx.Nonce, To: recipient, Gas: 500_000, GasFeeCap: uint256.MustFromBig(tx.GasFeeCap), GasTipCap: uint256.MustFromBig(tx.GasTipCap), AuthList: []types.SetCodeAuthorization{auth}}
					case "outgoing":
						tx.To = &params.XShardOutboxAddress
						tx.Gas = 500_000 // Include Amsterdam's state gas when first funding the outbox.
						tx.Data, err = sendABI.Pack("send", big.NewInt(10002), recipient, big.NewInt(100), params.XShardSendGas, true, []byte{})
						if err != nil {
							t.Fatal(err)
						}
						tx.Value = new(big.Int).Add(big.NewInt(100), new(big.Int).Mul(new(big.Int).SetUint64(params.XShardSendGas), tx.GasFeeCap))
					case "failed":
						tx.To = &reverter
						tx.GasTipCap = big.NewInt(1)
					case "refund":
						tx.To = &refunder
					}
					b.AddTx(types.MustSignNewTx(key, signer, data))
				}
			})
			defer genDB.Close()
			db := rawdb.NewMemoryDatabase()
			defer db.Close()
			chain, err := NewBlockChain(db, genesis, engine, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer chain.Stop()
			for i, block := range blocks {
				t.Run(cases[i].name, func(t *testing.T) {
					parent := chain.GetHeaderByHash(block.ParentHash())
					before, err := chain.StateAt(parent.Root, parent.Number, parent.Time)
					if err != nil {
						t.Fatal(err)
					}
					balanceBefore := before.GetBalance(block.Coinbase()).ToBig()
					if _, err := chain.InsertChain(types.Blocks{block}); err != nil {
						t.Fatal(err)
					}
					res, err := chain.StateAt(block.Root(), block.Number(), block.Time())
					if err != nil {
						t.Fatal(err)
					}
					delta := new(big.Int).Sub(res.GetBalance(block.Coinbase()).ToBig(), balanceBefore)
					// Force receipt decoding from storage, where GasUsed is derived
					// from cumulative gas, rather than using execution's cached data.
					chain.receiptsCache.Purge()
					amount, err := chain.GetCoinbaseAmount(block.Hash())
					if err != nil || amount.ToBig().Cmp(delta) != 0 {
						t.Fatalf("coinbase amount %v, error %v; balance delta %v", amount, err, delta)
					}
					receipts := chain.GetReceiptsByHash(block.Hash())
					for j, kind := range cases[i].kinds {
						wantStatus := uint64(types.ReceiptStatusSuccessful)
						if kind == "failed" {
							wantStatus = types.ReceiptStatusFailed
							if receipts[j].GasUsed%2 != 1 {
								t.Fatal("failed transaction must pay an odd fee to exercise tax rounding")
							}
						}
						if receipts[j].Status != wantStatus {
							t.Fatalf("%s receipt status %d, want %d", kind, receipts[j].Status, wantStatus)
						}
						if kind == "outgoing" {
							// Amsterdam also emits ETH transfer logs for the call value
							// and its refund, so select the outbox event by topic.
							found := false
							for _, log := range receipts[j].Logs {
								if log.Address == params.XShardOutboxAddress && len(log.Topics) > 0 && log.Topics[0] == vm.XShardSentMessageTopic {
									found = true
								}
							}
							if !found {
								t.Fatal("cross-shard send did not emit its message")
							}
						}
					}
				})
				if t.Failed() {
					return
				}
			}
		})
	}
}

// TestGetCoinbaseAmountErrors stores incomplete or invalid input directly to
// exercise reporting failures without changing the execution/validation rules.
func TestGetCoinbaseAmountErrors(t *testing.T) {
	config := genesisConfig(10001)
	db := rawdb.NewMemoryDatabase()
	defer db.Close()
	genesis := &Genesis{Config: config, Alloc: PoSWStakingContractAllocs(config), GasLimit: 12_000_000, Difficulty: new(big.Int)}
	chain, err := NewBlockChain(db, genesis, ethash.NewFaker(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	if amount, err := chain.GetCoinbaseAmount(chain.Genesis().Hash()); err != nil || !amount.IsZero() {
		t.Fatalf("genesis amount %v, error %v; want zero", amount, err)
	}
	if amount, err := chain.GetCoinbaseAmount(common.Hash{1}); err == nil || amount != nil {
		t.Fatalf("unknown block: amount %v, error %v", amount, err)
	}
	nonShard := &BlockChain{chainConfig: params.MergedTestChainConfig}
	if amount, err := nonShard.GetCoinbaseAmount(common.Hash{}); err == nil || amount != nil {
		t.Fatalf("non-shard chain: amount %v, error %v", amount, err)
	}
	to := common.Address{1}
	legacy := &types.LegacyTx{To: &to, Gas: params.TxGas, GasPrice: newGwei(2)}
	maxUint256 := new(uint256.Int).SetAllOne()
	for _, test := range []struct {
		name    string
		tx      types.TxData
		gasUsed uint64
		wantErr string
	}{
		{name: "missing body", wantErr: "not found"},
		{name: "missing empty receipts", wantErr: "receipts"},
		{name: "missing receipts", tx: legacy, wantErr: "receipts"},
		{name: "receipt count mismatch", tx: legacy, wantErr: "receipts"},
		{name: "fee cap below base fee", tx: &types.LegacyTx{To: &to, GasPrice: big.NewInt(1)}, gasUsed: 1, wantErr: "fee cap"},
		{name: "invalid gas price", tx: &types.LegacyTx{To: &to, GasPrice: new(big.Int).Lsh(big.NewInt(1), 256)}, gasUsed: 1, wantErr: "256"},
		{name: "fee overflow", tx: &types.DepositTx{To: &to, Value: new(uint256.Int), GasPrice: maxUint256}, gasUsed: params.XShardSendGas, wantErr: "exceeds uint256"},
		{name: "total overflow", tx: xShardTestMove(0, common.Address{}, to, 0, 1), gasUsed: 1, wantErr: "exceeds uint256"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config.QuarkChain.PoSW.CoinbaseAmount = new(big.Int)
			if test.name == "total overflow" {
				config.QuarkChain.PoSW.CoinbaseAmount = maxUint256.ToBig()
				config.QuarkChain.RewardTaxRate = new(big.Rat)
			}
			header := &types.Header{Number: big.NewInt(1), Difficulty: new(big.Int), BaseFee: newGwei(1), Extra: []byte(test.name)}
			block := types.NewBlockWithHeader(header)
			if test.tx != nil {
				block = block.WithBody(types.Body{Transactions: types.Transactions{types.NewTx(test.tx)}})
			}
			if test.name == "missing body" {
				rawdb.WriteHeader(db, header)
			} else {
				rawdb.WriteBlock(db, block)
			}
			if test.name == "receipt count mismatch" {
				rawdb.WriteReceipts(db, block.Hash(), 1, types.Receipts{})
			} else if test.gasUsed != 0 {
				rawdb.WriteReceipts(db, block.Hash(), 1, types.Receipts{{Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: test.gasUsed}})
			}
			amount, err := chain.GetCoinbaseAmount(block.Hash())
			if err == nil || !strings.Contains(err.Error(), test.wantErr) || amount != nil {
				t.Fatalf("amount %v, error %v; want %q error and no amount", amount, err, test.wantErr)
			}
		})
	}
}

func TestLockedStakesParentState(t *testing.T) {
	locked, unlocked, funded := common.HexToAddress("0x1234"), common.HexToAddress("0x5678"), common.HexToAddress("0x9abc")
	// Solidity mapping(address => Stake) at slot zero, amount at offset two.
	slot := func(address common.Address, offset int64) common.Hash {
		key := crypto.Keccak256Hash(common.LeftPadBytes(address[:], 32), make([]byte, 32))
		return common.BigToHash(new(big.Int).Add(key.Big(), big.NewInt(offset)))
	}
	alloc := PoSWStakingContractAllocs(params.NewTestQuarkChainConfig(10001))
	contract := alloc[params.ShardStakingContractAddress]
	contract.Storage = map[common.Hash]common.Hash{
		slot(locked, 2):   common.BigToHash(big.NewInt(123)),
		slot(locked, 3):   common.BytesToHash(unlocked[:]), // Root signer must not change shard stake ownership.
		slot(unlocked, 0): common.BigToHash(common.Big1),
		slot(unlocked, 2): common.BigToHash(big.NewInt(789)),
	}
	alloc[params.ShardStakingContractAddress] = contract
	rootContract := alloc[params.RootStakingContractAddress]
	rootContract.Storage = map[common.Hash]common.Hash{
		slot(locked, 2):   common.BigToHash(big.NewInt(321)),
		slot(locked, 3):   common.BytesToHash(funded[:]),
		slot(unlocked, 0): common.BigToHash(common.Big1),
		slot(unlocked, 2): common.BigToHash(big.NewInt(789)),
		slot(funded, 2):   common.BigToHash(big.NewInt(654)), // No explicit signer: use the staker.
	}
	alloc[params.RootStakingContractAddress] = rootContract
	alloc[funded] = types.Account{Balance: big.NewInt(10000)}
	genesis := &Genesis{Config: genesisConfig(10001), GasLimit: 12000000, Difficulty: new(big.Int), Alloc: alloc}
	db := rawdb.NewMemoryDatabase()
	defer db.Close()
	chain, err := NewBlockChain(db, genesis, ethash.NewFaker(), DefaultConfig().WithStateScheme(rawdb.HashScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	parent := chain.Genesis().Header()
	for _, test := range []struct {
		address common.Address
		want    uint64
	}{{locked, 123}, {unlocked, 0}, {funded, 0}} {
		got, err := chain.GetShardLockedStakes(parent, test.address)
		if err != nil || got.Uint64() != test.want {
			t.Fatalf("%s: %v, %v", test.address, got, err)
		}
	}
	for _, test := range []struct {
		address common.Address
		want    uint64
		signer  common.Address
	}{{locked, 321, funded}, {unlocked, 0, common.Address{}}, {funded, 654, funded}} {
		got, signer, err := chain.GetRootChainStakes(parent, test.address)
		if err != nil || got.Uint64() != test.want || signer != test.signer {
			t.Fatalf("root stakes for %s: %v, signer %s, error %v", test.address, got, signer, err)
		}
	}
	statedb, err := chain.StateAt(parent.Root, parent.Number, parent.Time)
	if err != nil {
		t.Fatal(err)
	}
	statedb.SetState(params.ShardStakingContractAddress, slot(locked, 2), common.BigToHash(big.NewInt(456)))
	statedb.SetState(params.RootStakingContractAddress, slot(locked, 2), common.BigToHash(big.NewInt(987)))
	statedb.SetState(params.RootStakingContractAddress, slot(locked, 3), common.BytesToHash(unlocked[:]))
	fork := types.CopyHeader(parent)
	fork.Root, err = statedb.Commit(chain.Config().Rules(parent.Number, true, parent.Time), 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		parent   *types.Header
		want     uint64
		rootWant uint64
		signer   common.Address
	}{{fork, 456, 987, unlocked}, {parent, 123, 321, funded}} {
		got, err := chain.GetShardLockedStakes(test.parent, locked)
		if err != nil || got.Uint64() != test.want {
			t.Fatalf("fork state: %v %v", got, err)
		}
		rootGot, signer, err := chain.GetRootChainStakes(test.parent, locked)
		if err != nil || rootGot.Uint64() != test.rootWant || signer != test.signer {
			t.Fatalf("root fork state: %v, signer %s, error %v", rootGot, signer, err)
		}
	}
	missingState := types.CopyHeader(parent)
	missingState.Root = common.HexToHash("0xdead")
	if _, err := chain.GetShardLockedStakes(missingState, locked); !errors.Is(err, consensus.ErrPrunedAncestor) {
		t.Fatalf("missing shard state: have %v, want ErrPrunedAncestor", err)
	}
	if _, _, err := chain.GetRootChainStakes(missingState, locked); !errors.Is(err, consensus.ErrPrunedAncestor) {
		t.Fatalf("missing root staking state: have %v, want ErrPrunedAncestor", err)
	}
}

func TestRootChainStakesCoinbaseChainOnly(t *testing.T) {
	for _, config := range []*params.ChainConfig{{}, params.NewTestQuarkChainConfig(10002)} {
		chain := &BlockChain{chainConfig: config}
		if _, _, err := chain.GetRootChainStakes(&types.Header{}, common.Address{}); err == nil {
			t.Fatal("accepted root-chain stakes query outside the configured root coinbase chain")
		}
	}
}

func TestLockedStakesErrors(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	defer db.Close()
	genesis := &Genesis{Config: genesisConfig(10001), GasLimit: 12000000, Difficulty: new(big.Int), Alloc: PoSWStakingContractAllocs(params.NewTestQuarkChainConfig(10001))}
	chain, err := NewBlockChain(db, genesis, ethash.NewFaker(), DefaultConfig().WithStateScheme(rawdb.HashScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	for _, test := range []struct {
		name        string
		missingBlob bool
		wantError   string
	}{
		{name: "wrong runtime", wantError: "invalid staking contract runtime"},
		{name: "missing code blob", missingBlob: true, wantError: "code is not found"},
	} {
		t.Run(test.name, func(t *testing.T) {
			header := chain.Genesis().Header()
			if test.missingBlob {
				// Keep the account's valid code hash, but force a database read of
				// missing bytecode. This exercises the lazy StateDB error path.
				rawdb.DeleteCode(db, params.PoSWStakingRuntimeCodeHash)
				chain.codedb = state.NewCodeDB(db)
			} else {
				statedb, err := chain.StateAt(header.Root, header.Number, header.Time)
				if err != nil {
					t.Fatal(err)
				}
				statedb.SetCode(params.RootStakingContractAddress, []byte{0x00}, tracing.CodeChangeUnspecified)
				header.Root, err = statedb.Commit(chain.Config().Rules(header.Number, true, header.Time), 1)
				if err != nil {
					t.Fatal(err)
				}
			}
			if stakes, signer, err := chain.GetRootChainStakes(header, common.Address{}); err == nil || !strings.Contains(err.Error(), test.wantError) || stakes != nil || signer != (common.Address{}) {
				t.Fatalf("root stakes: got (%v, %s, %v), want %q error and no stakes or signer", stakes, signer, err, test.wantError)
			}
		})
	}
}
