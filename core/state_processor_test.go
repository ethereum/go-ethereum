// Copyright 2020 The go-ethereum Authors
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

package core

import (
	"crypto/ecdsa"
	"encoding/binary"
	"math"
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/tracing"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/ethdb/memorydb"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/trie"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

// TestStateProcessorErrors tests the output from the 'core' errors
// as defined in core/error.go. These errors are generated when the
// blockchain imports bad blocks, meaning blocks which have valid headers but
// contain invalid transactions
func TestStateProcessorErrors(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      big.NewInt(0),
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ShanghaiBlock:       big.NewInt(0),
			Eip1559Block:        big.NewInt(0),
			CancunBlock:         big.NewInt(0),
			PragueBlock:         big.NewInt(0),
			Ethash:              new(params.EthashConfig),
		}
		signer  = types.LatestSigner(config)
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	)
	var makeTx = func(key *ecdsa.PrivateKey, nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *types.Transaction {
		tx, _ := types.SignTx(types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data), signer, key)
		return tx
	}
	var mkDynamicTx = func(nonce uint64, to common.Address, gasLimit uint64, gasTipCap, gasFeeCap *big.Int) *types.Transaction {
		tx, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
			Nonce:     nonce,
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Gas:       gasLimit,
			To:        &to,
			Value:     big.NewInt(0),
		}), signer, key1)
		return tx
	}
	var mkDynamicCreationTx = func(nonce uint64, gasLimit uint64, gasTipCap, gasFeeCap *big.Int, data []byte) *types.Transaction {
		tx, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
			Nonce:     nonce,
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Gas:       gasLimit,
			Value:     big.NewInt(0),
			Data:      data,
		}), signer, key1)
		return tx
	}
	var mkSetCodeTx = func(nonce uint64, to common.Address, gasLimit uint64, gasTipCap, gasFeeCap *big.Int, authlist []types.SetCodeAuthorization) *types.Transaction {
		tx, err := types.SignTx(types.NewTx(&types.SetCodeTx{
			Nonce:     nonce,
			GasTipCap: uint256.MustFromBig(gasTipCap),
			GasFeeCap: uint256.MustFromBig(gasFeeCap),
			Gas:       gasLimit,
			To:        to,
			Value:     new(uint256.Int),
			AuthList:  authlist,
		}), signer, key1)
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}

	{ // Tests against a 'recent' chain definition
		var (
			db    = rawdb.NewMemoryDatabase()
			gspec = &Genesis{
				Config: config,
				Alloc: types.GenesisAlloc{
					common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7"): types.Account{
						Balance: big.NewInt(1000000000000000000), // 1 ether
						Nonce:   0,
					},
					common.HexToAddress("0xfd0810DD14796680f72adf1a371963d0745BCc64"): types.Account{
						Balance: big.NewInt(1000000000000000000), // 1 ether
						Nonce:   math.MaxUint64,
					},
				},
			}
			genesis        = gspec.MustCommit(db)
			blockchain, _  = NewBlockChain(db, nil, gspec, ethash.NewFaker(), vm.Config{})
			tooBigInitCode = [params.MaxInitCodeSize + 1]byte{}
		)

		defer blockchain.Stop()
		bigNumber := new(big.Int).SetBytes(common.MaxHash.Bytes())
		tooBigNumber := new(big.Int).Set(bigNumber)
		tooBigNumber.Add(tooBigNumber, common.Big1)
		for i, tt := range []struct {
			txs  []*types.Transaction
			want string
		}{
			{ // ErrNonceTooLow
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(12500000000), nil),
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(12500000000), nil),
				},
				want: "could not apply tx 1 [0xecd6a889a307155b3562cd64c86957e36fa58267cb4efbbe39aa692fd7aab09a]: nonce too low: address xdc71562b71999873DB5b286dF957af199Ec94617F7, tx: 0 state: 1",
			},
			{ // ErrNonceTooHigh
				txs: []*types.Transaction{
					makeTx(key1, 100, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(875000000), nil),
				},
				want: "could not apply tx 0 [0xdebad714ca7f363bd0d8121c4518ad48fa469ca81b0a081be3d10c17460f751b]: nonce too high: address xdc71562b71999873DB5b286dF957af199Ec94617F7, tx: 100 state: 0",
			},
			{ // ErrNonceMax
				txs: []*types.Transaction{
					makeTx(key2, math.MaxUint64, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(875000000), nil),
				},
				want: "could not apply tx 0 [0x84ea18d60eb2bb3b040e3add0eb72f757727122cc257dd858c67cb6591a85986]: nonce has max value: address xdcfd0810DD14796680f72adf1a371963d0745BCc64, nonce: 18446744073709551615",
			},
			{ // ErrGasLimitReached
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), 21000000, big.NewInt(12500000000), nil),
				},
				want: "could not apply tx 0 [0x062b0e84f2d48f09f91e434fca8cb1fb864c4fb82f8bf27d58879ebe60c9f773]: gas limit reached, have: 4712388, need: 21000000",
			},
			{ // ErrInsufficientFundsForTransfer
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(1000000000000000000), params.TxGas, big.NewInt(12500000000), nil),
				},
				want: "could not apply tx 0 [0x50f89093bf5ad7f4ae6f9e3bad44d4dc130247ea0429df0cf78873584a76dfa1]: insufficient funds for gas * price + value: address xdc71562b71999873DB5b286dF957af199Ec94617F7 have 1000000000000000000 want 1000262500000000000",
			},
			{ // ErrInsufficientFunds
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(900000000000000000), nil),
				},
				want: "could not apply tx 0 [0x4a69690c4b0cd85e64d0d9ea06302455b01e10a83db964d60281739752003440]: insufficient funds for gas * price + value: address xdc71562b71999873DB5b286dF957af199Ec94617F7 have 1000000000000000000 want 18900000000000000000000",
			},
			// ErrGasUintOverflow
			// One missing 'core' error is ErrGasUintOverflow: "gas uint64 overflow",
			// In order to trigger that one, we'd have to allocate a _huge_ chunk of data, such that the
			// multiplication len(data) +gas_per_byte overflows uint64. Not testable at the moment
			{ // ErrIntrinsicGas
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas-1000, big.NewInt(12500000000), nil),
				},
				want: "could not apply tx 0 [0xa3484a466ffa8a88dc95e6ff520c853659dfc5507039c0b1452c2b845438771b]: intrinsic gas too low: have 20000, want 21000",
			},
			{ // ErrGasLimitReached
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas*1000, big.NewInt(12500000000), nil),
				},
				want: "could not apply tx 0 [0x062b0e84f2d48f09f91e434fca8cb1fb864c4fb82f8bf27d58879ebe60c9f773]: gas limit reached, have: 4712388, need: 21000000",
			},
			{ // ErrFeeCapTooLow
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, big.NewInt(0), big.NewInt(0)),
				},
				want: "could not apply tx 0 [0xc4ab868fef0c82ae0387b742aee87907f2d0fc528fc6ea0a021459fb0fc4a4a8]: max fee per gas less than block base fee: address xdc71562b71999873DB5b286dF957af199Ec94617F7, maxFeePerGas: 0 baseFee: 12500000000",
			},
			{ // ErrTipVeryHigh
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, tooBigNumber, big.NewInt(1)),
				},
				want: "could not apply tx 0 [0x15b8391b9981f266b32f3ab7da564bbeb3d6c21628364ea9b32a21139f89f712]: max priority fee per gas higher than 2^256-1: address xdc71562b71999873DB5b286dF957af199Ec94617F7, maxPriorityFeePerGas bit length: 257",
			},
			{ // ErrFeeCapVeryHigh
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, big.NewInt(1), tooBigNumber),
				},
				want: "could not apply tx 0 [0x48bc299b83fdb345c57478f239e89814bb3063eb4e4b49f3b6057a69255c16bd]: max fee per gas higher than 2^256-1: address xdc71562b71999873DB5b286dF957af199Ec94617F7, maxFeePerGas bit length: 257",
			},
			{ // ErrTipAboveFeeCap
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, big.NewInt(2), big.NewInt(1)),
				},
				want: "could not apply tx 0 [0xf987a31ff0c71895780a7612f965a0c8b056deb54e020bb44fa478092f14c9b4]: max priority fee per gas higher than max fee per gas: address xdc71562b71999873DB5b286dF957af199Ec94617F7, maxPriorityFeePerGas: 2, maxFeePerGas: 1",
			},
			{ // ErrInsufficientFunds
				// Available balance:           1000000000000000000
				// Effective cost:                   18375000021000
				// FeeCap * gas:                1050000000000000000
				// This test is designed to have the effective cost be covered by the balance, but
				// the extended requirement on FeeCap*gas < balance to fail
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, big.NewInt(1), big.NewInt(50000000000000)),
				},
				want: "could not apply tx 0 [0x413603cd096a87f41b1660d3ed3e27d62e1da78eac138961c0a1314ed43bd129]: insufficient funds for gas * price + value: address xdc71562b71999873DB5b286dF957af199Ec94617F7 have 1000000000000000000 want 1050000000000000000",
			},
			{ // Another ErrInsufficientFunds, this one to ensure that feecap/tip of max u256 is allowed
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, bigNumber, bigNumber),
				},
				want: "could not apply tx 0 [0xd82a0c2519acfeac9a948258c47e784acd20651d9d80f9a1c67b4137651c3a24]: insufficient funds for gas * price + value: address xdc71562b71999873DB5b286dF957af199Ec94617F7 have 1000000000000000000 want 2431633873983640103894990685182446064918669677978451844828609264166175722438635000",
			},
			{ // ErrMaxInitCodeSizeExceeded
				txs: []*types.Transaction{
					mkDynamicCreationTx(0, 520000, common.Big0, big.NewInt(params.InitialBaseFee), tooBigInitCode[:]),
				},
				want: "could not apply tx 0 [0x41d48b664cf891e625a16696a90e892ba3857c0b5ea759c3f2bdb4158338cb85]: max initcode size exceeded: code size 49153 limit 49152",
			},
			{ // ErrIntrinsicGas: Not enough gas to cover init code
				txs: []*types.Transaction{
					mkDynamicCreationTx(0, 54299, common.Big0, big.NewInt(params.InitialBaseFee), make([]byte, 320)),
				},
				want: "could not apply tx 0 [0x83f0bd65f2c2ad82de0da306aa93dea5e47d4ba0cd9f23ec4ce3fd0a3246da1c]: intrinsic gas too low: have 54299, want 54300",
			},
			{ // ErrEmptyAuthList
				txs: []*types.Transaction{
					mkSetCodeTx(0, common.Address{}, params.TxGas, big.NewInt(params.InitialBaseFee), big.NewInt(params.InitialBaseFee), nil),
				},
				want: "could not apply tx 0 [0x2fadb4fa7ccf8564edc21590f8d94a5b93a981b2bb2de8256978cb7361bc69de]: EIP-7702 transaction with empty auth list (sender 0x71562b71999873DB5b286dF957af199Ec94617F7)",
			},
			// ErrSetCodeTxCreate cannot be tested: it is impossible to create a SetCode-tx with nil `to`.
		} {
			block := GenerateBadBlock(t, genesis, ethash.NewFaker(), tt.txs, gspec.Config)
			_, err := blockchain.InsertChain(types.Blocks{block})
			if err == nil {
				t.Fatal("block imported without errors")
			}
			if have, want := err.Error(), tt.want; have != want {
				t.Errorf("test %d:\nhave \"%v\"\nwant \"%v\"\n", i, have, want)
			}
		}
	}

	// ErrTxTypeNotSupported, For this, we need an older chain
	{
		var (
			db    = rawdb.NewMemoryDatabase()
			gspec = &Genesis{
				Config: &params.ChainConfig{
					ChainID:             big.NewInt(1),
					HomesteadBlock:      big.NewInt(0),
					EIP150Block:         big.NewInt(0),
					EIP155Block:         big.NewInt(0),
					EIP158Block:         big.NewInt(0),
					ByzantiumBlock:      big.NewInt(0),
					ConstantinopleBlock: big.NewInt(0),
					PetersburgBlock:     big.NewInt(0),
					IstanbulBlock:       big.NewInt(0),
				},
				Alloc: types.GenesisAlloc{
					common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7"): types.Account{
						Balance: big.NewInt(1000000000000000000), // 1 ether
						Nonce:   0,
					},
				},
			}
			genesis       = gspec.MustCommit(db)
			blockchain, _ = NewBlockChain(db, nil, gspec, ethash.NewFaker(), vm.Config{})
		)
		defer blockchain.Stop()
		for i, tt := range []struct {
			txs  []*types.Transaction
			want string
		}{
			{ // ErrTxTypeNotSupported
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas-1000, big.NewInt(0), big.NewInt(0)),
				},
				want: "transaction type not supported",
			},
		} {
			block := GenerateBadBlock(t, genesis, ethash.NewFaker(), tt.txs, gspec.Config)
			_, err := blockchain.InsertChain(types.Blocks{block})
			if err == nil {
				t.Fatal("block imported without errors")
			}
			if have, want := err.Error(), tt.want; have != want {
				t.Errorf("test %d:\nhave \"%v\"\nwant \"%v\"\n", i, have, want)
			}
		}
	}
}

// GenerateBadBlock constructs a "block" which contains the transactions. The transactions are not expected to be
// valid, and no proper post-state can be made. But from the perspective of the blockchain, the block is sufficiently
// valid to be considered for import:
// - valid pow (fake), ancestry, difficulty, gaslimit etc
func GenerateBadBlock(t *testing.T, parent *types.Block, engine consensus.Engine, txs types.Transactions, config *params.ChainConfig) *types.Block {
	header := &types.Header{
		ParentHash: parent.Hash(),
		Coinbase:   parent.Coinbase(),
		Difficulty: engine.CalcDifficulty(&fakeChainReader{config: config, engine: engine}, parent.Time()+10, &types.Header{
			Number:     parent.Number(),
			Time:       parent.Time(),
			Difficulty: parent.Difficulty(),
			UncleHash:  parent.UncleHash(),
		}),
		GasLimit:  parent.GasLimit(),
		Number:    new(big.Int).Add(parent.Number(), common.Big1),
		Time:      parent.Time() + 10,
		UncleHash: types.EmptyUncleHash,
	}
	if config.IsEIP1559(header.Number) {
		header.BaseFee = common.BaseFee
	}
	var receipts []*types.Receipt
	// The post-state result doesn't need to be correct (this is a bad block), but we do need something there
	// Preferably something unique. So let's use a combo of blocknum + txhash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(header.Number.Bytes())
	var cumulativeGas uint64
	for _, tx := range txs {
		txh := tx.Hash()
		hasher.Write(txh[:])
		receipt := types.NewReceipt(nil, false, cumulativeGas+tx.Gas())
		receipt.TxHash = tx.Hash()
		receipt.GasUsed = tx.Gas()
		receipts = append(receipts, receipt)
		cumulativeGas += tx.Gas()
	}
	header.Root = common.BytesToHash(hasher.Sum(nil))
	// Assemble and return the final block for sealing
	return types.NewBlock(header, &types.Body{Transactions: txs}, receipts, trie.NewStackTrie(nil))
}

// TestApplyTransactionWithEVMTracer tests that tracer's OnTxStart and OnTxEnd
// are called for all transaction types, including non-EVM special transactions.
func TestApplyTransactionWithEVMTracer(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      big.NewInt(0),
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			Eip1559Block:        big.NewInt(0),
			Ethash:              new(params.EthashConfig),
		}
		signer     = types.LatestSigner(config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		testAddr   = crypto.PubkeyToAddress(testKey.PublicKey)
	)

	tests := []struct {
		name       string
		to         *common.Address
		expectOnTx bool // expect OnTxStart/OnTxEnd to be called
	}{
		{
			name:       "BlockSignersBinary transaction",
			to:         &common.BlockSignersBinary,
			expectOnTx: true,
		},
		{
			name:       "XDCXAddrBinary transaction",
			to:         &common.XDCXAddrBinary,
			expectOnTx: true,
		},
		{
			name: "Regular transaction",
			to: func() *common.Address {
				addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
				return &addr
			}(),
			expectOnTx: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test database and genesis
			db := rawdb.NewMemoryDatabase()
			gspec := &Genesis{
				Config: config,
				Alloc: types.GenesisAlloc{
					testAddr: types.Account{
						Balance: big.NewInt(1000000000000000000), // 1 ether
						Nonce:   0,
					},
				},
			}
			genesis := gspec.MustCommit(db)
			blockchain, _ := NewBlockChain(db, nil, gspec, ethash.NewFaker(), vm.Config{})
			defer blockchain.Stop()

			// Create state database
			statedb, err := blockchain.State()
			if err != nil {
				t.Fatalf("Failed to get state: %v", err)
			}

			// Create a transaction with sufficient gas price to avoid base fee errors
			tx := types.NewTransaction(0, *tt.to, big.NewInt(0), 100000, big.NewInt(20000000000), nil)
			signedTx, err := types.SignTx(tx, signer, testKey)
			if err != nil {
				t.Fatalf("Failed to sign transaction: %v", err)
			}

			// Create a mock tracer
			onTxStartCalled := false
			onTxEndCalled := false
			mockTracer := &tracing.Hooks{
				OnTxStart: func(vmContext *tracing.VMContext, tx *types.Transaction, from common.Address) {
					onTxStartCalled = true
					if tx == nil {
						t.Error("OnTxStart called with nil transaction")
					}
					if from != testAddr {
						t.Errorf("OnTxStart called with wrong from address: got %v, want %v", from, testAddr)
					}
				},
				OnTxEnd: func(receipt *types.Receipt, err error) {
					onTxEndCalled = true
				},
			}

			// Create EVM with tracer
			vmConfig := vm.Config{
				Tracer: mockTracer,
			}

			msg, err := TransactionToMessage(signedTx, signer, nil, nil, nil)
			if err != nil {
				t.Fatalf("Failed to create message: %v", err)
			}

			gasPool := new(GasPool).AddGas(1000000)
			blockNumber := big.NewInt(1)
			blockHash := genesis.Hash()

			vmContext := NewEVMBlockContext(blockchain.CurrentBlock(), blockchain, nil)
			evm := vm.NewEVM(vmContext, statedb, nil, blockchain.Config(), vmConfig)

			// Apply transaction
			var usedGas uint64
			_, _, _, err = ApplyTransactionWithEVM(msg, config, gasPool, statedb, blockNumber, blockHash, signedTx, &usedGas, evm, big.NewInt(0), common.Address{})
			// NOTE: Some special transactions (like BlockSignersBinary or XDCXAddrBinary)
			// may fail in test environment due to missing configuration or state, but
			// the tracer should still be called at the beginning of ApplyTransactionWithEVM.
			// We don't fail the test on transaction execution error as long as tracer was invoked.
			if err != nil {
				t.Logf("Transaction execution returned error (expected for some special txs): %v", err)
			}

			// Verify tracer was called
			if tt.expectOnTx {
				if !onTxStartCalled {
					t.Error("OnTxStart was not called")
				}
				if !onTxEndCalled {
					t.Error("OnTxEnd was not called")
				}
			}
		})
	}
}

func TestApplyTransactionWithEVMStateChangeHooks(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      big.NewInt(0),
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			Eip1559Block:        big.NewInt(0),
			Ethash:              new(params.EthashConfig),
		}
		signer      = types.LatestSigner(config)
		testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender      = crypto.PubkeyToAddress(testKey.PublicKey)
		recipient   = common.HexToAddress("0x1234567890123456789012345678901234567890")
		hookInvoked bool
	)

	db := rawdb.NewMemoryDatabase()
	gspec := &Genesis{
		Config: config,
		Alloc: types.GenesisAlloc{
			sender: {
				Balance: big.NewInt(1000000000000000000),
				Nonce:   0,
			},
		},
	}
	genesis := gspec.MustCommit(db)
	blockchain, _ := NewBlockChain(db, nil, gspec, ethash.NewFaker(), vm.Config{})
	defer blockchain.Stop()

	statedb, err := blockchain.State()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	tx := types.NewTransaction(0, recipient, big.NewInt(1), 21000, big.NewInt(20000000000), nil)
	signedTx, err := types.SignTx(tx, signer, testKey)
	if err != nil {
		t.Fatalf("Failed to sign tx: %v", err)
	}

	hooks := &tracing.Hooks{
		OnBalanceChange: func(addr common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
			hookInvoked = true
		},
	}
	hookedState := state.NewHookedState(statedb, hooks)

	vmContext := NewEVMBlockContext(blockchain.CurrentBlock(), blockchain, nil)
	evmenv := vm.NewEVM(vmContext, hookedState, nil, blockchain.Config(), vm.Config{Tracer: hooks})

	msg, err := TransactionToMessage(signedTx, signer, nil, big.NewInt(1), nil)
	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}

	gasPool := new(GasPool).AddGas(1000000)
	var usedGas uint64
	_, _, _, err = ApplyTransactionWithEVM(msg, config, gasPool, statedb, big.NewInt(1), genesis.Hash(), signedTx, &usedGas, evmenv, nil, common.Address{})
	if err != nil {
		t.Fatalf("ApplyTransactionWithEVM failed: %v", err)
	}
	if !hookInvoked {
		t.Fatal("expected OnBalanceChange to be invoked, but it was not")
	}
}

func TestProcessParentBlockHash(t *testing.T) {
	var (
		chainConfig = params.MergedTestChainConfig
		hashA       = common.Hash{0x01}
		hashB       = common.Hash{0x02}
		header      = &types.Header{ParentHash: hashA, Number: big.NewInt(2), Difficulty: big.NewInt(0)}
		parent      = &types.Header{ParentHash: hashB, Number: big.NewInt(1), Difficulty: big.NewInt(0)}
		coinbase    = common.Address{}
	)
	test := func(statedb *state.StateDB) {
		statedb.SetNonce(params.HistoryStorageAddress, 1)
		statedb.SetCode(params.HistoryStorageAddress, params.HistoryStorageCode)
		statedb.IntermediateRoot(true)

		vmContext := NewEVMBlockContext(header, nil, &coinbase)
		evm := vm.NewEVM(vmContext, statedb, nil, chainConfig, vm.Config{})
		ProcessParentBlockHash(header.ParentHash, evm, statedb)

		vmContext = NewEVMBlockContext(parent, nil, &coinbase)
		evm = vm.NewEVM(vmContext, statedb, nil, chainConfig, vm.Config{})
		ProcessParentBlockHash(parent.ParentHash, evm, statedb)

		// make sure that the state is correct
		if have := getParentBlockHash(statedb, 1); have != hashA {
			t.Errorf("want parent hash %v, have %v", hashA, have)
		}
		if have := getParentBlockHash(statedb, 0); have != hashB {
			t.Errorf("want parent hash %v, have %v", hashB, have)
		}
	}
	t.Run("MPT", func(t *testing.T) {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewDatabase(memorydb.New())))
		test(statedb)
	})
}

func TestProcessParentBlockHashPragueGuard(t *testing.T) {
	config := *params.MergedTestChainConfig
	config.PragueBlock = big.NewInt(10)

	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewDatabase(memorydb.New())))
	blockNumber := big.NewInt(5)
	random := common.Hash{}
	blockContext := vm.BlockContext{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		Coinbase:    common.Address{},
		BlockNumber: blockNumber,
		Time:        0,
		Difficulty:  big.NewInt(0),
		GasLimit:    0,
		BaseFee:     nil,
		Random:      &random,
	}
	evm := vm.NewEVM(blockContext, statedb, nil, &config, vm.Config{})
	ProcessParentBlockHash(common.Hash{0x01}, evm, statedb)

	if code := statedb.GetCode(params.HistoryStorageAddress); len(code) != 0 {
		t.Fatalf("unexpected history contract code predeploy: %x", code)
	}
	if have := getParentBlockHash(statedb, 0); have != (common.Hash{}) {
		t.Fatalf("expected empty history slot, have %v", have)
	}
}

func TestProcessParentBlockHashBackfillMissingHistory(t *testing.T) {
	config := *params.MergedTestChainConfig
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewDatabase(memorydb.New())))
	blockNumber := big.NewInt(int64(params.HistoryServeWindow + 1))
	available := map[uint64]common.Hash{
		1:   {0x11},
		100: {0x22},
	}

	random := common.Hash{}
	blockContext := vm.BlockContext{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash: func(n uint64) common.Hash {
			if hash, ok := available[n]; ok {
				return hash
			}
			return common.Hash{}
		},
		Coinbase:    common.Address{},
		BlockNumber: blockNumber,
		Time:        0,
		Difficulty:  big.NewInt(0),
		GasLimit:    0,
		BaseFee:     nil,
		Random:      &random,
	}
	evm := vm.NewEVM(blockContext, statedb, nil, &config, vm.Config{})
	ProcessParentBlockHash(common.Hash{0x01}, evm, statedb)

	if have := getParentBlockHash(statedb, 1); have != available[1] {
		t.Fatalf("expected hash at slot 1, have %v", have)
	}
	if have := getParentBlockHash(statedb, 100); have != available[100] {
		t.Fatalf("expected hash at slot 100, have %v", have)
	}
	if have := getParentBlockHash(statedb, 2); have != (common.Hash{}) {
		t.Fatalf("expected empty history slot, have %v", have)
	}
}

func TestProcessParentBlockHashCodeMismatchPanics(t *testing.T) {
	config := *params.MergedTestChainConfig
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewDatabase(memorydb.New())))
	statedb.SetCode(params.HistoryStorageAddress, []byte{0x01})

	blockNumber := big.NewInt(1)
	random := common.Hash{}
	blockContext := vm.BlockContext{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		Coinbase:    common.Address{},
		BlockNumber: blockNumber,
		Time:        0,
		Difficulty:  big.NewInt(0),
		GasLimit:    0,
		BaseFee:     nil,
		Random:      &random,
	}
	evm := vm.NewEVM(blockContext, statedb, nil, &config, vm.Config{})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on history storage code mismatch")
		}
	}()
	ProcessParentBlockHash(common.Hash{0x01}, evm, statedb)
}

func getParentBlockHash(statedb *state.StateDB, number uint64) common.Hash {
	ringIndex := number % params.HistoryServeWindow
	var key common.Hash
	binary.BigEndian.PutUint64(key[24:], ringIndex)
	return statedb.GetState(params.HistoryStorageAddress, key)
}
