// Copyright 2025 The go-ethereum Authors
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

package tracetest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// accountState represents the expected final state of an account
type accountState struct {
	Balance *big.Int
	Nonce   uint64
	Code    []byte
	Exists  bool
}

// selfdestructStateTracer tracks state changes during selfdestruct operations
type selfdestructStateTracer struct {
	env      *tracing.VMContext
	accounts map[common.Address]*accountState
}

func newSelfdestructStateTracer() *selfdestructStateTracer {
	return &selfdestructStateTracer{
		accounts: make(map[common.Address]*accountState),
	}
}

func (t *selfdestructStateTracer) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.env = env
}

func (t *selfdestructStateTracer) OnTxEnd(receipt *types.Receipt, err error) {
	// Nothing to do
}

func (t *selfdestructStateTracer) getOrCreateAccount(addr common.Address) *accountState {
	if acc, ok := t.accounts[addr]; ok {
		return acc
	}

	// Initialize with current state from statedb
	acc := &accountState{
		Balance: t.env.StateDB.GetBalance(addr).ToBig(),
		Nonce:   t.env.StateDB.GetNonce(addr),
		Code:    t.env.StateDB.GetCode(addr),
		Exists:  t.env.StateDB.Exist(addr),
	}
	t.accounts[addr] = acc
	return acc
}

func (t *selfdestructStateTracer) OnBalanceChange(addr common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	acc := t.getOrCreateAccount(addr)
	acc.Balance = new
}

func (t *selfdestructStateTracer) OnNonceChangeV2(addr common.Address, prev, new uint64, reason tracing.NonceChangeReason) {
	acc := t.getOrCreateAccount(addr)
	acc.Nonce = new

	// If this is a selfdestruct nonce change, mark account as not existing
	if reason == tracing.NonceChangeSelfdestruct {
		acc.Exists = false
	}
}

func (t *selfdestructStateTracer) OnCodeChangeV2(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte, reason tracing.CodeChangeReason) {
	acc := t.getOrCreateAccount(addr)
	acc.Code = code

	// If this is a selfdestruct code change, mark account as not existing
	if reason == tracing.CodeChangeSelfDestruct {
		acc.Exists = false
	}
}

func (t *selfdestructStateTracer) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnTxStart:       t.OnTxStart,
		OnTxEnd:         t.OnTxEnd,
		OnBalanceChange: t.OnBalanceChange,
		OnNonceChangeV2: t.OnNonceChangeV2,
		OnCodeChangeV2:  t.OnCodeChangeV2,
	}
}

func (t *selfdestructStateTracer) Accounts() map[common.Address]*accountState {
	return t.accounts
}

// verifyAccountState compares actual and expected account state and reports any mismatches
func verifyAccountState(t *testing.T, addr common.Address, actual, expected *accountState) {
	if actual.Balance.Cmp(expected.Balance) != 0 {
		t.Errorf("address %s: balance mismatch: have %s, want %s",
			addr.Hex(), actual.Balance, expected.Balance)
	}
	if actual.Nonce != expected.Nonce {
		t.Errorf("address %s: nonce mismatch: have %d, want %d",
			addr.Hex(), actual.Nonce, expected.Nonce)
	}
	if len(actual.Code) != len(expected.Code) {
		t.Errorf("address %s: code length mismatch: have %d, want %d",
			addr.Hex(), len(actual.Code), len(expected.Code))
	}
	if actual.Exists != expected.Exists {
		t.Errorf("address %s: exists mismatch: have %v, want %v",
			addr.Hex(), actual.Exists, expected.Exists)
	}
}

// setupTestBlockchain creates a blockchain with the given genesis and transaction,
// returns the blockchain, the first block, and a statedb at genesis for testing
func setupTestBlockchain(t *testing.T, genesis *core.Genesis, tx *types.Transaction, useBeacon bool) (*core.BlockChain, *types.Block, *state.StateDB) {
	var engine consensus.Engine
	if useBeacon {
		engine = beacon.New(ethash.NewFaker())
	} else {
		engine = ethash.NewFaker()
	}

	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, 1, func(i int, b *core.BlockGen) {
		b.AddTx(tx)
	})
	db := rawdb.NewMemoryDatabase()
	blockchain, err := core.NewBlockChain(db, genesis, engine, nil)
	if err != nil {
		t.Fatalf("failed to create blockchain: %v", err)
	}
	if _, err := blockchain.InsertChain(blocks); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
	genesisBlock := blockchain.GetBlockByNumber(0)
	if genesisBlock == nil {
		t.Fatalf("failed to get genesis block")
	}
	statedb, err := blockchain.StateAt(genesisBlock.Root())
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}

	return blockchain, blocks[0], statedb
}

func TestSelfdestructStateTracer(t *testing.T) {
	t.Parallel()

	const (
		// Gas limit high enough for all test scenarios (factory creation + multiple calls)
		testGasLimit = 500000

		// Common balance amounts used across tests
		testBalanceInitial = 100 // Initial balance for contracts being tested
		testBalanceSent    = 50  // Amount sent back in sendback tests
		testBalanceFactory = 200 // Factory needs extra balance for contract creation
	)

	// Helper to create *big.Int for wei amounts
	wei := func(amount int64) *big.Int {
		return big.NewInt(amount)
	}

	// Test account (transaction sender)
	var (
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		caller = crypto.PubkeyToAddress(key.PublicKey)
	)

	// Simple selfdestruct test contracts
	var (
		contract  = common.HexToAddress("0x00000000000000000000000000000000000000bb")
		recipient = common.HexToAddress("0x00000000000000000000000000000000000000cc")
	)
	// Build selfdestruct code: PUSH20 <recipient> SELFDESTRUCT
	selfdestructCode := []byte{byte(vm.PUSH20)}
	selfdestructCode = append(selfdestructCode, recipient.Bytes()...)
	selfdestructCode = append(selfdestructCode, byte(vm.SELFDESTRUCT))

	// Factory test contracts (create-and-destroy pattern)
	var (
		factory = common.HexToAddress("0x00000000000000000000000000000000000000ff")
	)
	// Factory code: creates a contract with 100 wei and calls it to trigger selfdestruct back to factory
	// See selfdestruct_test_contracts/factory.yul for source
	// Runtime bytecode compiled with: solc --strict-assembly --evm-version paris factory.yul --bin
	// (Using paris to avoid PUSH0 opcode which is not available pre-Shanghai)
	var (
		factoryCode         = common.Hex2Bytes("6a6133ff6000526002601ef360a81b600052600080808080600b816064f05af100")
		createdContractAddr = crypto.CreateAddress(factory, 0) // Address where factory creates the contract
	)

	// Sendback test contracts (A→B→A pattern)
	// For the refund test: Coordinator calls A, then B
	// A selfdestructs to B, B sends funds back to A
	var (
		contractA   = common.HexToAddress("0x00000000000000000000000000000000000000aa")
		contractB   = common.HexToAddress("0x00000000000000000000000000000000000000bb")
		coordinator = common.HexToAddress("0x00000000000000000000000000000000000000cc")
	)
	// Contract A: if msg.value > 0, accept funds; else selfdestruct to B
	// See selfdestruct_test_contracts/contractA.yul for source
	// Runtime bytecode compiled with: solc --strict-assembly --evm-version paris contractA.yul --bin
	contractACode := common.Hex2Bytes("60003411600a5760bbff5b00")

	// Contract B: sends 50 wei back to contract A
	// See selfdestruct_test_contracts/contractB.yul for source
	// Runtime bytecode compiled with: solc --strict-assembly --evm-version paris contractB.yul --bin
	contractBCode := common.Hex2Bytes("6000808080603260aa5af100")

	// Coordinator: calls A (A selfdestructs to B), then calls B (B sends funds to A)
	// See selfdestruct_test_contracts/coordinator.yul for source
	// Runtime bytecode compiled with: solc --strict-assembly --evm-version paris coordinator.yul --bin
	coordinatorCode := common.Hex2Bytes("60008080808060aa818080808060bb955af1505af100")

	// Factory for create-and-refund test: creates A with 100 wei, calls A, calls B
	// See selfdestruct_test_contracts/factoryRefund.yul for source
	// Runtime bytecode compiled with: solc --strict-assembly --evm-version paris factoryRefund.yul --bin
	var (
		factoryRefund        = common.HexToAddress("0x00000000000000000000000000000000000000dd")
		factoryRefundCode    = common.Hex2Bytes("60008080808060bb78600c600d600039600c6000f3fe60003411600a5760bbff5b0082528180808080601960076064f05af1505af100")
		createdContractAddrA = crypto.CreateAddress(factoryRefund, 0) // Address where factory creates contract A
	)

	// Self-destruct-to-self test contracts
	var (
		contractSelfDestruct = common.HexToAddress("0x00000000000000000000000000000000000000aa")
		coordinatorSendAfter = common.HexToAddress("0x00000000000000000000000000000000000000ee")
	)
	// Contract that selfdestructs to self
	// See selfdestruct_test_contracts/contractSelfDestruct.yul
	contractSelfDestructCode := common.Hex2Bytes("30ff")

	// Coordinator: calls contract (triggers selfdestruct to self), stores balance, sends 50 wei, stores balance again
	// See selfdestruct_test_contracts/coordinatorSendAfter.yul
	coordinatorSendAfterCode := common.Hex2Bytes("60aa600080808080855af150803160005560008080806032855af1503160015500")

	// Factory with balance checking: creates contract, calls it, checks balances
	// See selfdestruct_test_contracts/factorySelfDestructBalanceCheck.yul
	var (
		factorySelfDestructBalanceCheck     = common.HexToAddress("0x00000000000000000000000000000000000000fd")
		factorySelfDestructBalanceCheckCode = common.Hex2Bytes("6e6002600d60003960026000f3fe30ff600052600f60116064f0600080808080855af150803160005560008080806032855af1503160015500")
		createdContractAddrSelfBalanceCheck = crypto.CreateAddress(factorySelfDestructBalanceCheck, 0)
	)

	tests := []struct {
		name            string
		description     string
		targetContract  common.Address
		genesis         *core.Genesis
		useBeacon       bool
		expectedResults map[common.Address]accountState
		expectedStorage map[common.Address]map[uint64]*big.Int
	}{
		{
			name:           "pre_6780_existing",
			description:    "Pre-EIP-6780: Existing contract selfdestructs to recipient. Contract should be destroyed and balance transferred.",
			targetContract: contract,
			genesis: &core.Genesis{
				Config: params.AllEthashProtocolChanges,
				Alloc: types.GenesisAlloc{
					caller: {Balance: big.NewInt(params.Ether)},
					contract: {
						Balance: wei(testBalanceInitial),
						Code:    selfdestructCode,
					},
				},
			},
			useBeacon: false,
			expectedResults: map[common.Address]accountState{
				contract: {
					Balance: wei(0),
					Nonce:   0,
					Code:    []byte{},
					Exists:  false,
				},
				recipient: {
					Balance: wei(testBalanceInitial), // Received contract's balance
					Nonce:   0,
					Code:    []byte{},
					Exists:  true,
				},
			},
		},
		{
			name:           "post_6780_existing",
			description:    "Post-EIP-6780: Existing contract selfdestructs to recipient. Balance transferred but contract NOT destroyed (code/storage remain).",
			targetContract: contract,
			genesis: &core.Genesis{
				Config: params.AllDevChainProtocolChanges,
				Alloc: types.GenesisAlloc{
					caller: {Balance: big.NewInt(params.Ether)},
					contract: {
						Balance: wei(testBalanceInitial),
						Code:    selfdestructCode,
					},
				},
			},
			useBeacon: true,
			expectedResults: map[common.Address]accountState{
				contract: {
					Balance: wei(0),
					Nonce:   0,
					Code:    selfdestructCode,
					Exists:  true,
				},
				recipient: {
					Balance: wei(testBalanceInitial),
					Nonce:   0,
					Code:    []byte{},
					Exists:  true,
				},
			},
		},
		{
			name:           "pre_6780_create_destroy",
			description:    "Pre-EIP-6780: Factory creates contract with 100 wei, contract selfdestructs back to factory. Contract destroyed, factory gets refund.",
			targetContract: factory,
			genesis: &core.Genesis{
				Config: params.AllEthashProtocolChanges,
				Alloc: types.GenesisAlloc{
					caller: {Balance: big.NewInt(params.Ether)},
					factory: {
						Balance: wei(testBalanceFactory),
						Code:    factoryCode,
					},
				},
			},
			useBeacon: false,
			expectedResults: map[common.Address]accountState{
				factory: {
					Balance: wei(testBalanceFactory),
					Nonce:   1,
					Code:    factoryCode,
					Exists:  true,
				},
				createdContractAddr: {
					Balance: wei(0),
					Nonce:   0,
					Code:    []byte{},
					Exists:  false,
				},
			},
		},
		{
			name:           "post_6780_create_destroy",
			description:    "Post-EIP-6780: Factory creates contract with 100 wei, contract selfdestructs back to factory. Contract destroyed (EIP-6780 exception for same-tx creation).",
			targetContract: factory,
			genesis: &core.Genesis{
				Config: params.AllDevChainProtocolChanges,
				Alloc: types.GenesisAlloc{
					caller: {Balance: big.NewInt(params.Ether)},
					factory: {
						Balance: wei(testBalanceFactory),
						Code:    factoryCode,
					},
				},
			},
			useBeacon: true,
			expectedResults: map[common.Address]accountState{
				factory: {
					Balance: wei(testBalanceFactory),
					Nonce:   1,
					Code:    factoryCode,
					Exists:  true,
				},
				createdContractAddr: {
					Balance: wei(0),
					Nonce:   0,
					Code:    []byte{},
					Exists:  false,
				},
			},
		},
		{
			name:           "pre_6780_sendback",
			description:    "Pre-EIP-6780: Contract A selfdestructs sending funds to B, then B sends funds back to A's address. Funds sent to destroyed address are burnt.",
			targetContract: coordinator,
			genesis: &core.Genesis{
				Config: params.AllEthashProtocolChanges,
				Alloc: types.GenesisAlloc{
					caller: {Balance: big.NewInt(params.Ether)},
					contractA: {
						Balance: wei(testBalanceInitial),
						Code:    contractACode,
					},
					contractB: {
						Balance: wei(0),
						Code:    contractBCode,
					},
					coordinator: {
						Code: coordinatorCode,
					},
				},
			},
			useBeacon: false,
			expectedResults: map[common.Address]accountState{
				contractA: {
					Balance: wei(0),
					Nonce:   0,
					Code:    []byte{},
					Exists:  false,
				},
				contractB: {
					// 100 received - 50 sent back
					Balance: wei(testBalanceSent),
					Nonce:   0,
					Code:    contractBCode,
					Exists:  true,
				},
			},
		},
		{
			name:           "post_6780_existing_sendback",
			description:    "Post-EIP-6780: Existing contract A selfdestructs to B, then B sends funds back to A. Funds are NOT burnt (A still exists post-6780).",
			targetContract: coordinator,
			genesis: &core.Genesis{
				Config: params.AllDevChainProtocolChanges,
				Alloc: types.GenesisAlloc{
					caller: {Balance: big.NewInt(params.Ether)},
					contractA: {
						Balance: wei(testBalanceInitial),
						Code:    contractACode,
					},
					contractB: {
						Balance: wei(0),
						Code:    contractBCode,
					},
					coordinator: {
						Code: coordinatorCode,
					},
				},
			},
			useBeacon: true,
			expectedResults: map[common.Address]accountState{
				contractA: {
					Balance: wei(testBalanceSent),
					Nonce:   0,
					Code:    contractACode,
					Exists:  true,
				},
				contractB: {
					Balance: wei(testBalanceSent),
					Nonce:   0,
					Code:    contractBCode,
					Exists:  true,
				},
			},
		},
		{
			name:           "post_6780_create_destroy_sendback",
			description:    "Post-EIP-6780: Factory creates A, A selfdestructs to B, B sends funds back to A. Funds are burnt (A was destroyed via EIP-6780 exception).",
			targetContract: factoryRefund,
			genesis: &core.Genesis{
				Config: params.AllDevChainProtocolChanges,
				Alloc: types.GenesisAlloc{
					caller: {Balance: big.NewInt(params.Ether)},
					contractB: {
						Balance: wei(0),
						Code:    contractBCode,
					},
					factoryRefund: {
						Balance: wei(testBalanceFactory),
						Code:    factoryRefundCode,
					},
				},
			},
			useBeacon: true,
			expectedResults: map[common.Address]accountState{
				createdContractAddrA: {
					// Funds sent back are burnt!
					Balance: wei(0),
					Nonce:   0,
					Code:    []byte{},
					Exists:  false,
				},
				contractB: {
					Balance: wei(testBalanceSent),
					Nonce:   0,
					Code:    contractBCode,
					Exists:  true,
				},
			},
		},
		{
			name:           "post_6780_existing_to_self",
			description:    "Post-EIP-6780: Pre-existing contract selfdestructs to itself. Balance NOT burnt (selfdestruct-to-self is no-op for existing contracts).",
			targetContract: coordinatorSendAfter,
			genesis: &core.Genesis{
				Config: params.AllDevChainProtocolChanges,
				Alloc: types.GenesisAlloc{
					caller: {Balance: big.NewInt(params.Ether)},
					contractSelfDestruct: {
						Balance: wei(testBalanceInitial),
						Code:    contractSelfDestructCode,
					},
					coordinatorSendAfter: {
						Balance: wei(testBalanceInitial),
						Code:    coordinatorSendAfterCode,
					},
				},
			},
			useBeacon: true,
			expectedResults: map[common.Address]accountState{
				contractSelfDestruct: {
					Balance: wei(150),
					Nonce:   0,
					Code:    contractSelfDestructCode,
					Exists:  true,
				},
				coordinatorSendAfter: {
					Balance: wei(testBalanceSent),
					Nonce:   0,
					Code:    coordinatorSendAfterCode,
					Exists:  true,
				},
			},
			expectedStorage: map[common.Address]map[uint64]*big.Int{
				coordinatorSendAfter: {
					0: wei(testBalanceInitial),
					1: wei(150),
				},
			},
		},
		{
			name:           "post_6780_create_destroy_to_self",
			description:    "Post-EIP-6780: Factory creates contract, contract selfdestructs to itself. Balance IS burnt and contract destroyed (EIP-6780 exception for same-tx creation).",
			targetContract: factorySelfDestructBalanceCheck,
			genesis: &core.Genesis{
				Config: params.AllDevChainProtocolChanges,
				Alloc: types.GenesisAlloc{
					caller: {Balance: big.NewInt(params.Ether)},
					factorySelfDestructBalanceCheck: {
						Balance: wei(testBalanceFactory),
						Code:    factorySelfDestructBalanceCheckCode,
					},
				},
			},
			useBeacon: true,
			expectedResults: map[common.Address]accountState{
				createdContractAddrSelfBalanceCheck: {
					Balance: wei(0),
					Nonce:   0,
					Code:    []byte{},
					Exists:  false,
				},
				factorySelfDestructBalanceCheck: {
					Balance: wei(testBalanceSent),
					Nonce:   1,
					Code:    factorySelfDestructBalanceCheckCode,
					Exists:  true,
				},
			},
			expectedStorage: map[common.Address]map[uint64]*big.Int{
				factorySelfDestructBalanceCheck: {
					0: wei(0),
					1: wei(0),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				signer = types.HomesteadSigner{}
				tx     *types.Transaction
				err    error
			)

			tx, err = types.SignTx(types.NewTx(&types.LegacyTx{
				Nonce:    0,
				To:       &tt.targetContract,
				Value:    big.NewInt(0),
				Gas:      testGasLimit,
				GasPrice: big.NewInt(params.InitialBaseFee * 2),
				Data:     nil,
			}), signer, key)
			if err != nil {
				t.Fatalf("failed to sign transaction: %v", err)
			}

			blockchain, block, statedb := setupTestBlockchain(t, tt.genesis, tx, tt.useBeacon)
			defer blockchain.Stop()

			tracer := newSelfdestructStateTracer()
			hookedState := state.NewHookedState(statedb, tracer.Hooks())
			msg, err := core.TransactionToMessage(tx, signer, nil)
			if err != nil {
				t.Fatalf("failed to prepare transaction for tracing: %v", err)
			}
			context := core.NewEVMBlockContext(block.Header(), blockchain, nil)
			evm := vm.NewEVM(context, hookedState, tt.genesis.Config, vm.Config{Tracer: tracer.Hooks()})
			usedGas := uint64(0)
			_, err = core.ApplyTransactionWithEVM(msg, new(core.GasPool).AddGas(tx.Gas()), statedb, block.Number(), block.Hash(), block.Time(), tx, &usedGas, evm)
			if err != nil {
				t.Fatalf("failed to execute transaction: %v", err)
			}

			results := tracer.Accounts()

			// Verify storage
			for addr, expectedSlots := range tt.expectedStorage {
				for slot, expectedValue := range expectedSlots {
					actualValue := statedb.GetState(addr, common.BigToHash(big.NewInt(int64(slot))))
					if actualValue.Big().Cmp(expectedValue) != 0 {
						t.Errorf("address %s slot %d: storage mismatch: have %s, want %s",
							addr.Hex(), slot, actualValue.Big(), expectedValue)
					}
				}
			}

			// Verify results
			for addr, expected := range tt.expectedResults {
				actual, ok := results[addr]
				if !ok {
					t.Errorf("address %s missing from results", addr.Hex())
					continue
				}
				verifyAccountState(t, addr, actual, &expected)
			}
		})
	}
}
