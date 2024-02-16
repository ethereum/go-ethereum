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
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"

	//"fmt"
	"math/big"
	//"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"

	//"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

func u64(val uint64) *uint64 { return &val }

// TestStateProcessorErrors tests the output from the 'core' errors
// as defined in core/error.go. These errors are generated when the
// blockchain imports bad blocks, meaning blocks which have valid headers but
// contain invalid transactions
func TestStateProcessorErrors(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(1),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			TerminalTotalDifficulty:       big.NewInt(0),
			TerminalTotalDifficultyPassed: true,
			ShanghaiTime:                  new(uint64),
			CancunTime:                    new(uint64),
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
	var mkBlobTx = func(nonce uint64, to common.Address, gasLimit uint64, gasTipCap, gasFeeCap *big.Int, hashes []common.Hash) *types.Transaction {
		tx, err := types.SignTx(types.NewTx(&types.BlobTx{
			Nonce:      nonce,
			GasTipCap:  uint256.MustFromBig(gasTipCap),
			GasFeeCap:  uint256.MustFromBig(gasFeeCap),
			Gas:        gasLimit,
			To:         to,
			BlobHashes: hashes,
			Value:      new(uint256.Int),
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
				Alloc: GenesisAlloc{
					common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7"): GenesisAccount{
						Balance: big.NewInt(1000000000000000000), // 1 ether
						Nonce:   0,
					},
					common.HexToAddress("0xfd0810DD14796680f72adf1a371963d0745BCc64"): GenesisAccount{
						Balance: big.NewInt(1000000000000000000), // 1 ether
						Nonce:   math.MaxUint64,
					},
				},
			}
			blockchain, _  = NewBlockChain(db, nil, gspec, nil, beacon.New(ethash.NewFaker()), vm.Config{}, nil, nil)
			tooBigInitCode = [params.MaxInitCodeSize + 1]byte{}
		)

		defer blockchain.Stop()
		bigNumber := new(big.Int).SetBytes(common.FromHex("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
		tooBigNumber := new(big.Int).Set(bigNumber)
		tooBigNumber.Add(tooBigNumber, common.Big1)
		for i, tt := range []struct {
			txs  []*types.Transaction
			want string
		}{
			{ // ErrNonceTooLow
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(875000000), nil),
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(875000000), nil),
				},
				want: "could not apply tx 1 [0x0026256b3939ed97e2c4a6f3fce8ecf83bdcfa6d507c47838c308a1fb0436f62]: nonce too low: address 0x71562b71999873DB5b286dF957af199Ec94617F7, tx: 0 state: 1",
			},
			{ // ErrNonceTooHigh
				txs: []*types.Transaction{
					makeTx(key1, 100, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(875000000), nil),
				},
				want: "could not apply tx 0 [0xdebad714ca7f363bd0d8121c4518ad48fa469ca81b0a081be3d10c17460f751b]: nonce too high: address 0x71562b71999873DB5b286dF957af199Ec94617F7, tx: 100 state: 0",
			},
			{ // ErrNonceMax
				txs: []*types.Transaction{
					makeTx(key2, math.MaxUint64, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(875000000), nil),
				},
				want: "could not apply tx 0 [0x84ea18d60eb2bb3b040e3add0eb72f757727122cc257dd858c67cb6591a85986]: nonce has max value: address 0xfd0810DD14796680f72adf1a371963d0745BCc64, nonce: 18446744073709551615",
			},
			{ // ErrGasLimitReached
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), 21000000, big.NewInt(875000000), nil),
				},
				want: "could not apply tx 0 [0xbd49d8dadfd47fb846986695f7d4da3f7b2c48c8da82dbc211a26eb124883de9]: gas limit reached",
			},
			{ // ErrInsufficientFundsForTransfer
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(1000000000000000000), params.TxGas, big.NewInt(875000000), nil),
				},
				want: "could not apply tx 0 [0x98c796b470f7fcab40aaef5c965a602b0238e1034cce6fb73823042dd0638d74]: insufficient funds for gas * price + value: address 0x71562b71999873DB5b286dF957af199Ec94617F7 have 1000000000000000000 want 1000018375000000000",
			},
			{ // ErrInsufficientFunds
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(900000000000000000), nil),
				},
				want: "could not apply tx 0 [0x4a69690c4b0cd85e64d0d9ea06302455b01e10a83db964d60281739752003440]: insufficient funds for gas * price + value: address 0x71562b71999873DB5b286dF957af199Ec94617F7 have 1000000000000000000 want 18900000000000000000000",
			},
			// ErrGasUintOverflow
			// One missing 'core' error is ErrGasUintOverflow: "gas uint64 overflow",
			// In order to trigger that one, we'd have to allocate a _huge_ chunk of data, such that the
			// multiplication len(data) +gas_per_byte overflows uint64. Not testable at the moment
			{ // ErrIntrinsicGas
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas-1000, big.NewInt(875000000), nil),
				},
				want: "could not apply tx 0 [0xcf3b049a0b516cb4f9274b3e2a264359e2ba53b2fb64b7bda2c634d5c9d01fca]: intrinsic gas too low: have 20000, want 21000",
			},
			{ // ErrGasLimitReached
				txs: []*types.Transaction{
					makeTx(key1, 0, common.Address{}, big.NewInt(0), params.TxGas*1000, big.NewInt(875000000), nil),
				},
				want: "could not apply tx 0 [0xbd49d8dadfd47fb846986695f7d4da3f7b2c48c8da82dbc211a26eb124883de9]: gas limit reached",
			},
			{ // ErrFeeCapTooLow
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, big.NewInt(0), big.NewInt(0)),
				},
				want: "could not apply tx 0 [0xc4ab868fef0c82ae0387b742aee87907f2d0fc528fc6ea0a021459fb0fc4a4a8]: max fee per gas less than block base fee: address 0x71562b71999873DB5b286dF957af199Ec94617F7, maxFeePerGas: 0 baseFee: 875000000",
			},
			{ // ErrTipVeryHigh
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, tooBigNumber, big.NewInt(1)),
				},
				want: "could not apply tx 0 [0x15b8391b9981f266b32f3ab7da564bbeb3d6c21628364ea9b32a21139f89f712]: max priority fee per gas higher than 2^256-1: address 0x71562b71999873DB5b286dF957af199Ec94617F7, maxPriorityFeePerGas bit length: 257",
			},
			{ // ErrFeeCapVeryHigh
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, big.NewInt(1), tooBigNumber),
				},
				want: "could not apply tx 0 [0x48bc299b83fdb345c57478f239e89814bb3063eb4e4b49f3b6057a69255c16bd]: max fee per gas higher than 2^256-1: address 0x71562b71999873DB5b286dF957af199Ec94617F7, maxFeePerGas bit length: 257",
			},
			{ // ErrTipAboveFeeCap
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, big.NewInt(2), big.NewInt(1)),
				},
				want: "could not apply tx 0 [0xf987a31ff0c71895780a7612f965a0c8b056deb54e020bb44fa478092f14c9b4]: max priority fee per gas higher than max fee per gas: address 0x71562b71999873DB5b286dF957af199Ec94617F7, maxPriorityFeePerGas: 2, maxFeePerGas: 1",
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
				want: "could not apply tx 0 [0x413603cd096a87f41b1660d3ed3e27d62e1da78eac138961c0a1314ed43bd129]: insufficient funds for gas * price + value: address 0x71562b71999873DB5b286dF957af199Ec94617F7 have 1000000000000000000 want 1050000000000000000",
			},
			{ // Another ErrInsufficientFunds, this one to ensure that feecap/tip of max u256 is allowed
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas, bigNumber, bigNumber),
				},
				want: "could not apply tx 0 [0xd82a0c2519acfeac9a948258c47e784acd20651d9d80f9a1c67b4137651c3a24]: insufficient funds for gas * price + value: address 0x71562b71999873DB5b286dF957af199Ec94617F7 have 1000000000000000000 want 2431633873983640103894990685182446064918669677978451844828609264166175722438635000",
			},
			{ // ErrMaxInitCodeSizeExceeded
				txs: []*types.Transaction{
					mkDynamicCreationTx(0, 500000, common.Big0, big.NewInt(params.InitialBaseFee), tooBigInitCode[:]),
				},
				want: "could not apply tx 0 [0xd491405f06c92d118dd3208376fcee18a57c54bc52063ee4a26b1cf296857c25]: max initcode size exceeded: code size 49153 limit 49152",
			},
			{ // ErrIntrinsicGas: Not enough gas to cover init code
				txs: []*types.Transaction{
					mkDynamicCreationTx(0, 54299, common.Big0, big.NewInt(params.InitialBaseFee), make([]byte, 320)),
				},
				want: "could not apply tx 0 [0xfd49536a9b323769d8472fcb3ebb3689b707a349379baee3e2ee3fe7baae06a1]: intrinsic gas too low: have 54299, want 54300",
			},
			{ // ErrBlobFeeCapTooLow
				txs: []*types.Transaction{
					mkBlobTx(0, common.Address{}, params.TxGas, big.NewInt(1), big.NewInt(1), []common.Hash{(common.Hash{1})}),
				},
				want: "could not apply tx 0 [0x6c11015985ce82db691d7b2d017acda296db88b811c3c60dc71449c76256c716]: max fee per gas less than block base fee: address 0x71562b71999873DB5b286dF957af199Ec94617F7, maxFeePerGas: 1 baseFee: 875000000",
			},
		} {
			block := GenerateBadBlock(gspec.ToBlock(), beacon.New(ethash.NewFaker()), tt.txs, gspec.Config)
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
					MuirGlacierBlock:    big.NewInt(0),
				},
				Alloc: GenesisAlloc{
					common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7"): GenesisAccount{
						Balance: big.NewInt(1000000000000000000), // 1 ether
						Nonce:   0,
					},
				},
			}
			blockchain, _ = NewBlockChain(db, nil, gspec, nil, ethash.NewFaker(), vm.Config{}, nil, nil)
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
				want: "could not apply tx 0 [0x88626ac0d53cb65308f2416103c62bb1f18b805573d4f96a3640bbbfff13c14f]: transaction type not supported",
			},
		} {
			block := GenerateBadBlock(gspec.ToBlock(), ethash.NewFaker(), tt.txs, gspec.Config)
			_, err := blockchain.InsertChain(types.Blocks{block})
			if err == nil {
				t.Fatal("block imported without errors")
			}
			if have, want := err.Error(), tt.want; have != want {
				t.Errorf("test %d:\nhave \"%v\"\nwant \"%v\"\n", i, have, want)
			}
		}
	}

	// ErrSenderNoEOA, for this we need the sender to have contract code
	{
		var (
			db    = rawdb.NewMemoryDatabase()
			gspec = &Genesis{
				Config: config,
				Alloc: GenesisAlloc{
					common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7"): GenesisAccount{
						Balance: big.NewInt(1000000000000000000), // 1 ether
						Nonce:   0,
						Code:    common.FromHex("0xB0B0FACE"),
					},
				},
			}
			blockchain, _ = NewBlockChain(db, nil, gspec, nil, beacon.New(ethash.NewFaker()), vm.Config{}, nil, nil)
		)
		defer blockchain.Stop()
		for i, tt := range []struct {
			txs  []*types.Transaction
			want string
		}{
			{ // ErrSenderNoEOA
				txs: []*types.Transaction{
					mkDynamicTx(0, common.Address{}, params.TxGas-1000, big.NewInt(0), big.NewInt(0)),
				},
				want: "could not apply tx 0 [0x88626ac0d53cb65308f2416103c62bb1f18b805573d4f96a3640bbbfff13c14f]: sender not an eoa: address 0x71562b71999873DB5b286dF957af199Ec94617F7, codehash: 0x9280914443471259d4570a8661015ae4a5b80186dbc619658fb494bebc3da3d1",
			},
		} {
			block := GenerateBadBlock(gspec.ToBlock(), beacon.New(ethash.NewFaker()), tt.txs, gspec.Config)
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
func GenerateBadBlock(parent *types.Block, engine consensus.Engine, txs types.Transactions, config *params.ChainConfig) *types.Block {
	difficulty := big.NewInt(0)
	if !config.TerminalTotalDifficultyPassed {
		difficulty = engine.CalcDifficulty(&fakeChainReader{config}, parent.Time()+10, &types.Header{
			Number:     parent.Number(),
			Time:       parent.Time(),
			Difficulty: parent.Difficulty(),
			UncleHash:  parent.UncleHash(),
		})
	}

	header := &types.Header{
		ParentHash: parent.Hash(),
		Coinbase:   parent.Coinbase(),
		Difficulty: difficulty,
		GasLimit:   parent.GasLimit(),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Time:       parent.Time() + 10,
		UncleHash:  types.EmptyUncleHash,
	}
	if config.IsLondon(header.Number) {
		header.BaseFee = eip1559.CalcBaseFee(config, parent.Header())
	}
	if config.IsShanghai(header.Number, header.Time) {
		header.WithdrawalsHash = &types.EmptyWithdrawalsHash
	}
	var receipts []*types.Receipt
	// The post-state result doesn't need to be correct (this is a bad block), but we do need something there
	// Preferably something unique. So let's use a combo of blocknum + txhash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(header.Number.Bytes())
	var cumulativeGas uint64
	var nBlobs int
	for _, tx := range txs {
		txh := tx.Hash()
		hasher.Write(txh[:])
		receipt := types.NewReceipt(nil, false, cumulativeGas+tx.Gas())
		receipt.TxHash = tx.Hash()
		receipt.GasUsed = tx.Gas()
		receipts = append(receipts, receipt)
		cumulativeGas += tx.Gas()
		nBlobs += len(tx.BlobHashes())
	}
	header.Root = common.BytesToHash(hasher.Sum(nil))
	if config.IsCancun(header.Number, header.Time) {
		var pExcess, pUsed = uint64(0), uint64(0)
		if parent.ExcessBlobGas() != nil {
			pExcess = *parent.ExcessBlobGas()
			pUsed = *parent.BlobGasUsed()
		}
		excess := eip4844.CalcExcessBlobGas(pExcess, pUsed)
		used := uint64(nBlobs * params.BlobTxBlobGasPerBlob)
		header.ExcessBlobGas = &excess
		header.BlobGasUsed = &used
	}
	// Assemble and return the final block for sealing
	if config.IsShanghai(header.Number, header.Time) {
		return types.NewBlockWithWithdrawals(header, txs, nil, receipts, []*types.Withdrawal{}, trie.NewStackTrie(nil))
	}
	return types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil))
}

// A contract creation that calls EXTCODECOPY in the constructor. Used to ensure that the witness
// will not contain that copied data.
// Source: https://gist.github.com/gballet/a23db1e1cb4ed105616b5920feb75985
var (
	code                               = common.FromHex(`6060604052600a8060106000396000f360606040526008565b00`)
	intrinsicContractCreationGas, _    = IntrinsicGas(code, nil, true, true, true, true)
	codeWithExtCodeCopy                = common.FromHex(`0x60806040526040516100109061017b565b604051809103906000f08015801561002c573d6000803e3d6000fd5b506000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555034801561007857600080fd5b5060008067ffffffffffffffff8111156100955761009461024a565b5b6040519080825280601f01601f1916602001820160405280156100c75781602001600182028036833780820191505090505b50905060008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1690506020600083833c81610101906101e3565b60405161010d90610187565b61011791906101a3565b604051809103906000f080158015610133573d6000803e3d6000fd5b50600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505061029b565b60d58061046783390190565b6102068061053c83390190565b61019d816101d9565b82525050565b60006020820190506101b86000830184610194565b92915050565b6000819050602082019050919050565b600081519050919050565b6000819050919050565b60006101ee826101ce565b826101f8846101be565b905061020381610279565b925060208210156102435761023e7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8360200360080261028e565b831692505b5050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600061028582516101d9565b80915050919050565b600082821b905092915050565b6101bd806102aa6000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063f566852414610030575b600080fd5b61003861004e565b6040516100459190610146565b60405180910390f35b6000600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166381ca91d36040518163ffffffff1660e01b815260040160206040518083038186803b1580156100b857600080fd5b505afa1580156100cc573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906100f0919061010a565b905090565b60008151905061010481610170565b92915050565b6000602082840312156101205761011f61016b565b5b600061012e848285016100f5565b91505092915050565b61014081610161565b82525050565b600060208201905061015b6000830184610137565b92915050565b6000819050919050565b600080fd5b61017981610161565b811461018457600080fd5b5056fea2646970667358221220a6a0e11af79f176f9c421b7b12f441356b25f6489b83d38cc828a701720b41f164736f6c63430008070033608060405234801561001057600080fd5b5060b68061001f6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063ab5ed15014602d575b600080fd5b60336047565b604051603e9190605d565b60405180910390f35b60006001905090565b6057816076565b82525050565b6000602082019050607060008301846050565b92915050565b600081905091905056fea26469706673582212203a14eb0d5cd07c277d3e24912f110ddda3e553245a99afc4eeefb2fbae5327aa64736f6c63430008070033608060405234801561001057600080fd5b5060405161020638038061020683398181016040528101906100329190610063565b60018160001c6100429190610090565b60008190555050610145565b60008151905061005d8161012e565b92915050565b60006020828403121561007957610078610129565b5b60006100878482850161004e565b91505092915050565b600061009b826100f0565b91506100a6836100f0565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038211156100db576100da6100fa565b5b828201905092915050565b6000819050919050565b6000819050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600080fd5b610137816100e6565b811461014257600080fd5b50565b60b3806101536000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c806381ca91d314602d575b600080fd5b60336047565b604051603e9190605a565b60405180910390f35b60005481565b6054816073565b82525050565b6000602082019050606d6000830184604d565b92915050565b600081905091905056fea26469706673582212209bff7098a2f526de1ad499866f27d6d0d6f17b74a413036d6063ca6a0998ca4264736f6c63430008070033`)
	intrinsicCodeWithExtCodeCopyGas, _ = IntrinsicGas(codeWithExtCodeCopy, nil, true, true, true, true)
)

func TestProcessVerkle(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(1),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			ShanghaiTime:                  u64(0),
			PragueTime:                    u64(0),
			TerminalTotalDifficulty:       common.Big0,
			TerminalTotalDifficultyPassed: true,
			ProofInBlocks:                 true,
		}
		signer     = types.LatestSigner(config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		bcdb       = rawdb.NewMemoryDatabase() // Database for the blockchain
		gendb      = rawdb.NewMemoryDatabase() // Database for the block-generation code, they must be separate as they are path-based.
		coinbase   = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		gspec      = &Genesis{
			Config: config,
			Alloc: GenesisAlloc{
				coinbase: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
			},
		}
	)
	// Verkle trees use the snapshot, which must be enabled before the
	// data is saved into the tree+database.
	genesis := gspec.MustCommit(bcdb)
	blockchain, _ := NewBlockChain(bcdb, nil, gspec, nil, beacon.New(ethash.NewFaker()), vm.Config{}, nil, nil)
	defer blockchain.Stop()

	// Commit the genesis block to the block-generation database as it
	// is now independent of the blockchain database.
	gspec.MustCommit(gendb)

	txCost1 := params.TxGas
	txCost2 := params.TxGas
	contractCreationCost := intrinsicContractCreationGas + uint64(7700 /* creation */ +2939 /* execution costs */)
	codeWithExtCodeCopyGas := intrinsicCodeWithExtCodeCopyGas + uint64(7000 /* creation */ +299744 /* execution costs */)
	blockGasUsagesExpected := []uint64{
		txCost1*2 + txCost2,
		txCost1*2 + txCost2 + contractCreationCost + codeWithExtCodeCopyGas,
	}
	// TODO utiliser GenerateChainWithGenesis pour le rendre plus pratique
	chain, _, proofs, keyvals := GenerateVerkleChain(gspec.Config, genesis, beacon.New(ethash.NewFaker()), gendb, 2, func(i int, gen *BlockGen) {
		gen.SetPoS()

		// TODO need to check that the tx cost provided is the exact amount used (no remaining left-over)
		tx, _ := types.SignTx(types.NewTransaction(uint64(i)*3, common.Address{byte(i), 2, 3}, big.NewInt(999), txCost1, big.NewInt(875000000), nil), signer, testKey)
		gen.AddTx(tx)
		tx, _ = types.SignTx(types.NewTransaction(uint64(i)*3+1, common.Address{}, big.NewInt(999), txCost1, big.NewInt(875000000), nil), signer, testKey)
		gen.AddTx(tx)
		tx, _ = types.SignTx(types.NewTransaction(uint64(i)*3+2, common.Address{}, big.NewInt(0), txCost2, big.NewInt(875000000), nil), signer, testKey)
		gen.AddTx(tx)

		// Add two contract creations in block #2
		if i == 1 {
			tx, _ = types.SignTx(types.NewContractCreation(6, big.NewInt(16), 3000000, big.NewInt(875000000), code), signer, testKey)
			gen.AddTx(tx)

			tx, _ = types.SignTx(types.NewContractCreation(7, big.NewInt(0), 3000000, big.NewInt(875000000), codeWithExtCodeCopy), signer, testKey)
			gen.AddTx(tx)
		}
	})

	// Uncomment to extract block #2
	//f, _ := os.Create("block2.rlp")
	//defer f.Close()
	//var buf bytes.Buffer
	//rlp.Encode(&buf, chain[1])
	//f.Write(buf.Bytes())
	//fmt.Printf("root= %x\n", chain[0].Root())
	// check the proof for the last block
	err := trie.DeserializeAndVerifyVerkleProof(proofs[1], chain[0].Root().Bytes(), chain[1].Root().Bytes(), keyvals[1])
	if err != nil {
		t.Fatal(err)
	}
	t.Log("verfied verkle proof")

	endnum, err := blockchain.InsertChain(chain)
	if err != nil {
		t.Fatalf("block %d imported with error: %v", endnum, err)
	}

	for i := 0; i < 2; i++ {
		b := blockchain.GetBlockByNumber(uint64(i) + 1)
		if b == nil {
			t.Fatalf("expected block %d to be present in chain", i+1)
		}
		if b.Hash() != chain[i].Hash() {
			t.Fatalf("block #%d not found at expected height", b.NumberU64())
		}
		if b.GasUsed() != blockGasUsagesExpected[i] {
			t.Fatalf("expected block #%d txs to use %d, got %d\n", b.NumberU64(), blockGasUsagesExpected[i], b.GasUsed())
		}
	}
}

func TestProcessVerkleInvalidContractCreation(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(69420),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			ShanghaiTime:                  u64(0),
			PragueTime:                    u64(0),
			TerminalTotalDifficulty:       common.Big0,
			TerminalTotalDifficultyPassed: true,
			ProofInBlocks:                 true,
		}
		bcdb     = rawdb.NewMemoryDatabase() // Database for the blockchain
		gendb    = rawdb.NewMemoryDatabase() // Database for the block-generation code, they must be separate as they are path-based.
		coinbase = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		account1 = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2 = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec    = &Genesis{
			Config: config,
			Alloc: GenesisAlloc{
				coinbase: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account1: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account2: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   1,
				},
			},
		}
	)
	// Verkle trees use the snapshot, which must be enabled before the
	// data is saved into the tree+database.
	genesis := gspec.MustCommit(bcdb)

	// Commit the genesis block to the block-generation database as it
	// is now independent of the blockchain database.
	gspec.MustCommit(gendb)

	// Create two blocks that reproduce what is happening on kaustinen.
	// - The first block contains two failing contract creation transactions, that write to storage before they revert.
	// - The second block contains a single failing contract creation transaction, that fails right off the bat.
	_, _, _, statediff := GenerateVerkleChain(gspec.Config, genesis, beacon.New(ethash.NewFaker()), gendb, 2, func(i int, gen *BlockGen) {
		gen.SetPoS()

		if i == 0 {
			var tx1, tx2, tx3 types.Transaction
			// SSTORE at slot 105 and reverts
			tx1payload := common.Hex2Bytes("f8d48084479c2c18830186a08080b8806000602955bda3f9600060ca55600060695523b360006039551983576000601255b0620c2fde2c592ac2600060bc55e0ac6000606455a63e22600060e655eb607e605c5360a2605d5360c7605e53601d605f5360eb606053606b606153608e60625360816063536079606453601e60655360fc60665360b7606753608b60685383021e7ca0cc20c65a97d2e526b8ec0f4266e8b01bdcde43b9aeb59d8bfb44e8eb8119c109a07a8e751813ae1b2ce734960dbc39a4f954917d7822a2c5d1dca18b06c584131f")
			if err := tx1.UnmarshalBinary(tx1payload); err != nil {
				t.Fatal(err)
			}
			gen.AddTx(&tx1)

			// SSTORE at slot 133 and reverts
			tx2payload := common.Hex2Bytes("02f8db83010f2c01843b9aca0084479c2c18830186a08080b88060006085553fad6000600a55600060565555600060b55506600060cf557f1b8b38183e7bd1bdfaa7123c5a4976e54cce0e42049d841411978fd3595e25c66019527f0538943712953cf08900aae40222a40b2d5a4ac8075ad8cf0870e2be307edbb96039527f9f3174ff85024747041ae7a611acffb987c513c088d90ab288aec080a0cd6ac65ce2cb0a912371f6b5a551ba8caffc22ec55ad4d3cb53de41d05eb77b6a02e0dfe8513dfa6ec7bfd7eda6f5c0dac21b39b982436045e128cec46cfd3f960")
			if err := tx2.UnmarshalBinary(tx2payload); err != nil {
				t.Fatal(err)
			}
			gen.AddTx(&tx2)

			// this one is a simple transfer that succeeds, necessary to get the correct nonce in the other block.
			tx3payload := common.Hex2Bytes("f8e80184479c2c18830186a094bbbbde4ca27f83fc18aa108170547ff57675936a80b8807ff71f7c15faadb969a76a5f54a81a0117e1e743cb7f24e378eda28442ea4c6eb6604a527fb5409e5718d44e23bfffac926e5ea726067f772772e7e19446acba0c853f62f5606a526020608a536088608b536039608c536004608d5360af608e537f7f7675d9f210e0a61564e6d11e7cd75f5bc9009ac9f6b94a0fc63035441a83021e7ba04a4a172d81ebb02847829b76a387ac09749c8b65668083699abe20c887fb9efca07c5b1a990702ec7b31a5e8e3935cd9a77649f8c25a84131229e24ab61aec6093")
			if err := tx3.UnmarshalBinary(tx3payload); err != nil {
				t.Fatal(err)
			}
			gen.AddTx(&tx3)
		} else {
			var tx types.Transaction
			// immediately reverts
			txpayload := common.Hex2Bytes("01f8d683010f2c028443ad7d0e830186a08080b880b00e7fa3c849dce891cce5fae8a4c46cbb313d6aec0c0ffe7863e05fb7b22d4807674c6055527ffbfcb0938f3e18f7937aa8fa95d880afebd5c4cec0d85186095832d03c85cf8a60755260ab60955360cf6096536066609753606e60985360fa609953609e609a53608e609b536024609c5360f6609d536072609e5360a4609fc080a08fc6f7101f292ff1fb0de8ac69c2d320fbb23bfe61cf327173786ea5daee6e37a044c42d91838ef06646294bf4f9835588aee66243b16a66a2da37641fae4c045f")
			if err := tx.UnmarshalBinary(txpayload); err != nil {
				t.Fatal(err)
			}
			gen.AddTx(&tx)
		}
	})

	// Check that values 0x29 and 0x05 are found in the storage (and that they lead
	// to no update, since the contract creation code reverted)
	for _, stemStateDiff := range statediff[0] {
		// Check that the value 0x85, which is overflowing the account header,
		// is present.
		if bytes.Equal(stemStateDiff.Stem[:], common.Hex2Bytes("a10042195481d30478251625e1ccef0e2174dc4e083e81d2566d880373f791")) {
			for _, suffixDiff := range stemStateDiff.SuffixDiffs {
				if suffixDiff.Suffix != 133 {
					t.Fatalf("invalid suffix diff found for %x in block #1: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
				}
			}
		} else if bytes.Equal(stemStateDiff.Stem[:], common.Hex2Bytes("b24fa84f214459af17d6e3f604811f252cac93146f02d67d7811bbcdfa448b")) {
			for _, suffixDiff := range stemStateDiff.SuffixDiffs {
				if suffixDiff.Suffix != 105 && suffixDiff.Suffix != 0 && suffixDiff.Suffix != 2 && suffixDiff.Suffix != 3 {
					t.Fatalf("invalid suffix diff found for %x in block #1: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
				}
			}
		} else if bytes.Equal(stemStateDiff.Stem[:], common.Hex2Bytes("97f2911f5efe08b74c28727d004e36d260225e73525fe2a300c8f58c7ffd76")) {
			// BLOCKHASH contract stem
			if len(stemStateDiff.SuffixDiffs) > 1 {
				t.Fatalf("invalid suffix diff count found for BLOCKHASH contract: %d != 1", len(stemStateDiff.SuffixDiffs))
			}
			if stemStateDiff.SuffixDiffs[0].Suffix != 64 {
				t.Fatalf("invalid suffix diff value found for BLOCKHASH contract: %d != 64", stemStateDiff.SuffixDiffs[0].Suffix)
			}
			// check that the "current value" is nil and that the new value isn't.
			if stemStateDiff.SuffixDiffs[0].CurrentValue != nil {
				t.Fatalf("non-nil current value in BLOCKHASH contract insert: %x", stemStateDiff.SuffixDiffs[0].CurrentValue)
			}
			if stemStateDiff.SuffixDiffs[0].NewValue == nil {
				t.Fatalf("nil new value in BLOCKHASH contract insert")
			}
		} else {
			for _, suffixDiff := range stemStateDiff.SuffixDiffs {
				if suffixDiff.Suffix > 4 {
					t.Fatalf("invalid suffix diff found for %x in block #1: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
				}
			}
		}
	}

	// Check that no account has a value above 4 in the 2nd block as no storage nor
	// code should make it to the witness.
	for _, stemStateDiff := range statediff[1] {
		for _, suffixDiff := range stemStateDiff.SuffixDiffs {
			if bytes.Equal(stemStateDiff.Stem[:], common.Hex2Bytes("97f2911f5efe08b74c28727d004e36d260225e73525fe2a300c8f58c7ffd76")) {
				// BLOCKHASH contract stem
				if len(stemStateDiff.SuffixDiffs) > 1 {
					t.Fatalf("invalid suffix diff count found for BLOCKHASH contract at block #2: %d != 1", len(stemStateDiff.SuffixDiffs))
				}
				if stemStateDiff.SuffixDiffs[0].Suffix != 65 {
					t.Fatalf("invalid suffix diff value found for BLOCKHASH contract at block #2: %d != 65", stemStateDiff.SuffixDiffs[0].Suffix)
				}
				if stemStateDiff.SuffixDiffs[0].NewValue == nil {
					t.Fatalf("missing post state value for BLOCKHASH contract at block #2")
				}
				if *stemStateDiff.SuffixDiffs[0].NewValue != common.HexToHash("53abcdfb284720ea59efe923d3dc774bbb7e787d829599f8ec7a81d344dd3d17") {
					t.Fatalf("invalid post state value for BLOCKHASH contract at block #2: %x != ", (*stemStateDiff.SuffixDiffs[0].NewValue)[:])
				}
			} else if suffixDiff.Suffix > 4 {
				t.Fatalf("invalid suffix diff found for %x in block #2: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
			}
		}
	}
}

func TestProcessVerkleContractWithEmptyCode(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(69421),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			ShanghaiTime:                  u64(0),
			PragueTime:                    u64(0),
			TerminalTotalDifficulty:       common.Big0,
			TerminalTotalDifficultyPassed: true,
			ProofInBlocks:                 true,
		}
		bcdb     = rawdb.NewMemoryDatabase() // Database for the blockchain
		gendb    = rawdb.NewMemoryDatabase() // Database for the block-generation code, they must be separate as they are path-based.
		coinbase = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		account1 = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2 = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec    = &Genesis{
			Config: config,
			Alloc: GenesisAlloc{
				coinbase: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account1: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account2: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   3,
				},
			},
		}
	)
	// Verkle trees use the snapshot, which must be enabled before the
	// data is saved into the tree+database.
	genesis := gspec.MustCommit(bcdb)

	// Commit the genesis block to the block-generation database as it
	// is now independent of the blockchain database.
	gspec.MustCommit(gendb)

	_, _, _, statediff := GenerateVerkleChain(gspec.Config, genesis, beacon.New(ethash.NewFaker()), gendb, 1, func(i int, gen *BlockGen) {
		gen.SetPoS()
		var tx types.Transaction
		// a transaction that does some PUSH1n but returns a 0-sized contract
		txpayload := common.Hex2Bytes("02f8db83010f2d03843b9aca008444cf6a05830186a08080b8807fdfbbb59f2371a76485ce557fd0de00c298d3ede52a3eab56d35af674eb49ec5860335260826053536001605453604c60555360f3605653606060575360446058536096605953600c605a5360df605b5360f3605c5360fb605d53600c605e53609a605f53607f60605360fe606153603d60625360f4606353604b60645360cac001a0486b6dc55b8a311568b7239a2cae1d77e7446dba71df61eaafd53f73820a138fa010bd48a45e56133ac4c5645142c2ea48950d40eb35050e9510b6bad9e15c5865")
		if err := tx.UnmarshalBinary(txpayload); err != nil {
			t.Fatal(err)
		}
		gen.AddTx(&tx)
	})

	for _, stemStateDiff := range statediff[0] {
		if bytes.Equal(stemStateDiff.Stem[:], common.Hex2Bytes("97f2911f5efe08b74c28727d004e36d260225e73525fe2a300c8f58c7ffd76")) {
			// BLOCKHASH contract stem
			if len(stemStateDiff.SuffixDiffs) > 1 {
				t.Fatalf("invalid suffix diff count found for BLOCKHASH contract: %d != 1", len(stemStateDiff.SuffixDiffs))
			}
			if stemStateDiff.SuffixDiffs[0].Suffix != 64 {
				t.Fatalf("invalid suffix diff value found for BLOCKHASH contract: %d != 64", stemStateDiff.SuffixDiffs[0].Suffix)
			}
			// check that the "current value" is nil and that the new value isn't.
			if stemStateDiff.SuffixDiffs[0].CurrentValue != nil {
				t.Fatalf("non-nil current value in BLOCKHASH contract insert: %x", stemStateDiff.SuffixDiffs[0].CurrentValue)
			}
			if stemStateDiff.SuffixDiffs[0].NewValue == nil {
				t.Fatalf("nil new value in BLOCKHASH contract insert")
			}
		} else {
			for _, suffixDiff := range stemStateDiff.SuffixDiffs {
				if suffixDiff.Suffix > 4 {
					// if d8898012c484fb48610ecb7963886339207dab004bce968b007b616ffa18e0 shows up, it means that the PUSHn
					// in the transaction above added entries into the witness, when they should not have since they are
					// part of a contract deployment.
					t.Fatalf("invalid suffix diff found for %x in block #1: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
				}
			}
		}
	}
}

func TestProcessVerklExtCodeHashOpcode(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(69421),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			ShanghaiTime:                  u64(0),
			PragueTime:                    u64(0),
			TerminalTotalDifficulty:       common.Big0,
			TerminalTotalDifficultyPassed: true,
			ProofInBlocks:                 true,
		}
		signer     = types.LatestSigner(config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		bcdb       = rawdb.NewMemoryDatabase() // Database for the blockchain
		gendb      = rawdb.NewMemoryDatabase() // Database for the block-generation code, they must be separate as they are path-based.
		coinbase   = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		account1   = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2   = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec      = &Genesis{
			Config: config,
			Alloc: GenesisAlloc{
				coinbase: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account1: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account2: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   3,
				},
			},
		}
	)
	// Verkle trees use the snapshot, which must be enabled before the
	// data is saved into the tree+database.
	genesis := gspec.MustCommit(bcdb)

	// Commit the genesis block to the block-generation database as it
	// is now independent of the blockchain database.
	gspec.MustCommit(gendb)

	dummyContract := []byte{
		0x60, 2, // PUSH1 2
		0x60, 12, // PUSH1 12
		0x60, 0x00, // PUSH1 0
		0x39, // CODECOPY

		0x60, 2, // PUSH1 2
		0x60, 0x00, // PUSH1 0
		0xF3, // RETURN

		// Contract that auto-calls EXTCODEHASH
		0x60, 42, // PUSH1 42
	}
	dummyContractAddr := common.HexToAddress("3a220f351252089d385b29beca14e27f204c296a")
	extCodeHashContract := []byte{
		0x60, 22, // PUSH1 22
		0x60, 12, // PUSH1 12
		0x60, 0x00, // PUSH1 0
		0x39, // CODECOPY

		0x60, 22, // PUSH1 22
		0x60, 0x00, // PUSH1 0
		0xF3, // RETURN

		// Contract that auto-calls EXTCODEHASH
		0x73, // PUSH20
		0x3a, 0x22, 0x0f, 0x35, 0x12, 0x52, 0x08, 0x9d, 0x38, 0x5b, 0x29, 0xbe, 0xca, 0x14, 0xe2, 0x7f, 0x20, 0x4c, 0x29, 0x6a,
		0x3F, // EXTCODEHASH
	}
	extCodeHashContractAddr := common.HexToAddress("db7d6ab1f17c6b31909ae466702703daef9269cf")
	_, _, _, statediff := GenerateVerkleChain(gspec.Config, genesis, beacon.New(ethash.NewFaker()), gendb, 2, func(i int, gen *BlockGen) {
		gen.SetPoS()

		if i == 0 {
			// Create dummy contract.
			tx, _ := types.SignTx(types.NewContractCreation(0, big.NewInt(0), 100_000, big.NewInt(875000000), dummyContract), signer, testKey)
			gen.AddTx(tx)

			// Create contract with EXTCODEHASH opcode.
			tx, _ = types.SignTx(types.NewContractCreation(1, big.NewInt(0), 100_000, big.NewInt(875000000), extCodeHashContract), signer, testKey)
			gen.AddTx(tx)
		} else {
			tx, _ := types.SignTx(types.NewTransaction(2, extCodeHashContractAddr, big.NewInt(0), 100_000, big.NewInt(875000000), nil), signer, testKey)
			gen.AddTx(tx)
		}
	})

	contractKeccakTreeKey := utils.GetTreeKeyCodeKeccak(dummyContractAddr[:])

	var stateDiffIdx = -1
	for i, stemStateDiff := range statediff[1] {
		if bytes.Equal(stemStateDiff.Stem[:], contractKeccakTreeKey[:31]) {
			stateDiffIdx = i
			break
		}
	}
	if stateDiffIdx == -1 {
		t.Fatalf("no state diff found for stem")
	}

	codeHashStateDiff := statediff[1][stateDiffIdx].SuffixDiffs[0]
	if codeHashStateDiff.Suffix != utils.CodeKeccakLeafKey {
		t.Fatalf("code hash invalid suffix")
	}
	if codeHashStateDiff.CurrentValue == nil {
		t.Fatalf("codeHash.CurrentValue must not be empty")
	}
	expCodeHash := crypto.Keccak256Hash(dummyContract[12:])
	if *codeHashStateDiff.CurrentValue != expCodeHash {
		t.Fatalf("codeHash.CurrentValue unexpected code hash")
	}
	if codeHashStateDiff.NewValue != nil {
		t.Fatalf("codeHash.NewValue must be nil")
	}
}

func TestProcessVerkleBalanceOpcode(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(69421),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			ShanghaiTime:                  u64(0),
			PragueTime:                    u64(0),
			TerminalTotalDifficulty:       common.Big0,
			TerminalTotalDifficultyPassed: true,
			ProofInBlocks:                 true,
		}
		signer     = types.LatestSigner(config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		bcdb       = rawdb.NewMemoryDatabase() // Database for the blockchain
		gendb      = rawdb.NewMemoryDatabase() // Database for the block-generation code, they must be separate as they are path-based.
		coinbase   = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		account1   = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2   = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec      = &Genesis{
			Config: config,
			Alloc: GenesisAlloc{
				coinbase: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account1: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account2: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   3,
				},
			},
		}
	)
	// Verkle trees use the snapshot, which must be enabled before the
	// data is saved into the tree+database.
	genesis := gspec.MustCommit(bcdb)

	// Commit the genesis block to the block-generation database as it
	// is now independent of the blockchain database.
	gspec.MustCommit(gendb)

	_, _, _, statediff := GenerateVerkleChain(gspec.Config, genesis, beacon.New(ethash.NewFaker()), gendb, 1, func(i int, gen *BlockGen) {
		gen.SetPoS()
		txData := []byte{
			0x73,                                                                                                                   // PUSH20
			0x61, 0x77, 0x84, 0x3d, 0xb3, 0x13, 0x8a, 0xe6, 0x96, 0x79, 0xA5, 0x4b, 0x95, 0xcf, 0x34, 0x5E, 0xD7, 0x59, 0x45, 0x0d, // 0x6177843db3138ae69679A54b95cf345ED759450d
			0x31, // BALANCE
		}
		tx, _ := types.SignTx(types.NewContractCreation(0, big.NewInt(0), 100_000, big.NewInt(875000000), txData), signer, testKey)
		gen.AddTx(tx)
	})

	account2BalanceTreeKey := utils.GetTreeKeyBalance(account2[:])

	var stateDiffIdx = -1
	for i, stemStateDiff := range statediff[0] {
		if bytes.Equal(stemStateDiff.Stem[:], account2BalanceTreeKey[:31]) {
			stateDiffIdx = i
			break
		}
	}
	if stateDiffIdx == -1 {
		t.Fatalf("no state diff found for stem")
	}

	var zero [32]byte
	balanceStateDiff := statediff[0][stateDiffIdx].SuffixDiffs[0]
	if balanceStateDiff.Suffix != utils.BalanceLeafKey {
		t.Fatalf("invalid suffix diff")
	}
	if balanceStateDiff.CurrentValue == nil {
		t.Fatalf("invalid current value")
	}
	if *balanceStateDiff.CurrentValue == zero {
		t.Fatalf("invalid current value")
	}
	if balanceStateDiff.NewValue != nil {
		t.Fatalf("invalid new value")
	}
}

func TestProcessVerkleSelfDestructInSeparateTx(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(69421),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			ShanghaiTime:                  u64(0),
			PragueTime:                    u64(0),
			TerminalTotalDifficulty:       common.Big0,
			TerminalTotalDifficultyPassed: true,
			ProofInBlocks:                 true,
		}
		signer     = types.LatestSigner(config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		bcdb       = rawdb.NewMemoryDatabase() // Database for the blockchain
		gendb      = rawdb.NewMemoryDatabase() // Database for the block-generation code, they must be separate as they are path-based.
		coinbase   = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		account1   = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2   = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec      = &Genesis{
			Config: config,
			Alloc: GenesisAlloc{
				coinbase: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account1: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account2: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   3,
				},
			},
		}
	)
	genesis := gspec.MustCommit(bcdb)
	gspec.MustCommit(gendb)

	// The goal of this test is to test SELFDESTRUCT that happens in a contract execution which is created
	// in a previous transaction.

	selfDestructContract := []byte{
		0x60, 22, // PUSH1 22
		0x60, 12, // PUSH1 12
		0x60, 0x00, // PUSH1 0
		0x39, // CODECOPY

		0x60, 22, // PUSH1 22
		0x60, 0x00, // PUSH1 0
		0xF3, // RETURN

		// Deployed code
		0x73,                                                                                                                   // PUSH20
		0x61, 0x77, 0x84, 0x3d, 0xb3, 0x13, 0x8a, 0xe6, 0x96, 0x79, 0xA5, 0x4b, 0x95, 0xcf, 0x34, 0x5E, 0xD7, 0x59, 0x45, 0x0d, // 0x6177843db3138ae69679A54b95cf345ED759450d
		0xFF, // SELFDESTRUCT
	}
	selfDestructContractAddr := common.HexToAddress("3a220f351252089d385b29beca14e27f204c296a")
	_, _, _, statediff := GenerateVerkleChain(gspec.Config, genesis, beacon.New(ethash.NewFaker()), gendb, 2, func(i int, gen *BlockGen) {
		gen.SetPoS()

		if i == 0 {
			// Create selfdestruct contract, sending 42 wei.
			tx, _ := types.SignTx(types.NewContractCreation(0, big.NewInt(42), 100_000, big.NewInt(875000000), selfDestructContract), signer, testKey)
			gen.AddTx(tx)
		} else {
			// Call it.
			tx, _ := types.SignTx(types.NewTransaction(1, selfDestructContractAddr, big.NewInt(0), 100_000, big.NewInt(875000000), nil), signer, testKey)
			gen.AddTx(tx)
		}
	})

	var zero [32]byte
	{ // Check self-destructed contract in the witness
		selfDestructContractTreeKey := utils.GetTreeKeyCodeKeccak(selfDestructContractAddr[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediff[1] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructContractTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediff[1][stateDiffIdx].SuffixDiffs[1]
		if balanceStateDiff.Suffix != utils.BalanceLeafKey {
			t.Fatalf("balance invalid suffix")
		}

		// The original balance was 42.
		var fourtyTwo [32]byte
		fourtyTwo[0] = 42
		if *balanceStateDiff.CurrentValue != fourtyTwo {
			t.Fatalf("the pre-state balance before self-destruct must be 42")
		}

		// The new balance must be 0.
		if *balanceStateDiff.NewValue != zero {
			t.Fatalf("the post-state balance after self-destruct must be 0")
		}
	}
	{ // Check self-destructed target in the witness.
		selfDestructTargetTreeKey := utils.GetTreeKeyCodeKeccak(account2[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediff[1] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructTargetTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediff[1][stateDiffIdx].SuffixDiffs[0]
		if balanceStateDiff.Suffix != utils.BalanceLeafKey {
			t.Fatalf("balance invalid suffix")
		}
		if balanceStateDiff.CurrentValue == nil {
			t.Fatalf("codeHash.CurrentValue must not be empty")
		}
		if balanceStateDiff.NewValue == nil {
			t.Fatalf("codeHash.NewValue must not be empty")
		}
		preStateBalance := binary.LittleEndian.Uint64(balanceStateDiff.CurrentValue[:])
		postStateBalance := binary.LittleEndian.Uint64(balanceStateDiff.NewValue[:])
		if postStateBalance-preStateBalance != 42 {
			t.Fatalf("the post-state balance after self-destruct must be 42")
		}
	}
}

func TestProcessVerkleSelfDestructInSameTx(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(69421),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			ShanghaiTime:                  u64(0),
			PragueTime:                    u64(0),
			TerminalTotalDifficulty:       common.Big0,
			TerminalTotalDifficultyPassed: true,
			ProofInBlocks:                 true,
		}
		signer     = types.LatestSigner(config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		bcdb       = rawdb.NewMemoryDatabase() // Database for the blockchain
		gendb      = rawdb.NewMemoryDatabase() // Database for the block-generation code, they must be separate as they are path-based.
		coinbase   = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		account1   = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2   = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec      = &Genesis{
			Config: config,
			Alloc: GenesisAlloc{
				coinbase: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account1: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account2: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   3,
				},
			},
		}
	)
	genesis := gspec.MustCommit(bcdb)
	gspec.MustCommit(gendb)

	// The goal of this test is to test SELFDESTRUCT that happens in a contract execution which is created
	// in **the same** transaction sending the remaining balance to an external (i.e: not itself) account.

	selfDestructContract := []byte{
		0x73,                                                                                                                   // PUSH20
		0x61, 0x77, 0x84, 0x3d, 0xb3, 0x13, 0x8a, 0xe6, 0x96, 0x79, 0xA5, 0x4b, 0x95, 0xcf, 0x34, 0x5E, 0xD7, 0x59, 0x45, 0x0d, // 0x6177843db3138ae69679A54b95cf345ED759450d
		0xFF, // SELFDESTRUCT
	}
	selfDestructContractAddr := common.HexToAddress("3a220f351252089d385b29beca14e27f204c296a")
	_, _, _, statediff := GenerateVerkleChain(gspec.Config, genesis, beacon.New(ethash.NewFaker()), gendb, 1, func(i int, gen *BlockGen) {
		gen.SetPoS()
		tx, _ := types.SignTx(types.NewContractCreation(0, big.NewInt(42), 100_000, big.NewInt(875000000), selfDestructContract), signer, testKey)
		gen.AddTx(tx)
	})

	{ // Check self-destructed contract in the witness
		selfDestructContractTreeKey := utils.GetTreeKeyCodeKeccak(selfDestructContractAddr[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediff[0] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructContractTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediff[0][stateDiffIdx].SuffixDiffs[1]
		if balanceStateDiff.Suffix != utils.BalanceLeafKey {
			t.Fatalf("balance invalid suffix")
		}

		if balanceStateDiff.CurrentValue != nil {
			t.Fatalf("the pre-state balance before must be nil, since the contract didn't exist")
		}

		if balanceStateDiff.NewValue != nil {
			t.Fatalf("the post-state balance after self-destruct must be nil since the contract shouldn't be created at all")
		}
	}
	{ // Check self-destructed target in the witness.
		selfDestructTargetTreeKey := utils.GetTreeKeyCodeKeccak(account2[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediff[0] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructTargetTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediff[0][stateDiffIdx].SuffixDiffs[0]
		if balanceStateDiff.Suffix != utils.BalanceLeafKey {
			t.Fatalf("balance invalid suffix")
		}
		if balanceStateDiff.CurrentValue == nil {
			t.Fatalf("codeHash.CurrentValue must not be empty")
		}
		if balanceStateDiff.NewValue == nil {
			t.Fatalf("codeHash.NewValue must not be empty")
		}
		preStateBalance := binary.LittleEndian.Uint64(balanceStateDiff.CurrentValue[:])
		postStateBalance := binary.LittleEndian.Uint64(balanceStateDiff.NewValue[:])
		if postStateBalance-preStateBalance != 42 {
			t.Fatalf("the post-state balance after self-destruct must be 42")
		}
	}
}

func TestProcessVerkleSelfDestructInSeparateTxWithSelfBeneficiary(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(69421),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			ShanghaiTime:                  u64(0),
			PragueTime:                    u64(0),
			TerminalTotalDifficulty:       common.Big0,
			TerminalTotalDifficultyPassed: true,
			ProofInBlocks:                 true,
		}
		signer     = types.LatestSigner(config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		bcdb       = rawdb.NewMemoryDatabase() // Database for the blockchain
		gendb      = rawdb.NewMemoryDatabase() // Database for the block-generation code, they must be separate as they are path-based.
		coinbase   = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		account1   = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2   = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec      = &Genesis{
			Config: config,
			Alloc: GenesisAlloc{
				coinbase: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account1: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account2: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   3,
				},
			},
		}
	)
	genesis := gspec.MustCommit(bcdb)
	gspec.MustCommit(gendb)

	// The goal of this test is to test SELFDESTRUCT that happens in a contract execution which is created
	// in a *previous* transaction sending the remaining balance to itself.

	selfDestructContract := []byte{
		0x60, 22, // PUSH1 22
		0x60, 12, // PUSH1 12
		0x60, 0x00, // PUSH1 0
		0x39, // CODECOPY

		0x60, 22, // PUSH1 22
		0x60, 0x00, // PUSH1 0
		0xF3, // RETURN

		// Deployed code
		0x73,                                                                                                                   // PUSH20
		0x3a, 0x22, 0x0f, 0x35, 0x12, 0x52, 0x08, 0x9d, 0x38, 0x5b, 0x29, 0xbe, 0xca, 0x14, 0xe2, 0x7f, 0x20, 0x4c, 0x29, 0x6a, // 0x3a220f351252089d385b29beca14e27f204c296a
		0xFF, // SELFDESTRUCT
	}
	selfDestructContractAddr := common.HexToAddress("3a220f351252089d385b29beca14e27f204c296a")
	_, _, _, statediff := GenerateVerkleChain(gspec.Config, genesis, beacon.New(ethash.NewFaker()), gendb, 2, func(i int, gen *BlockGen) {
		gen.SetPoS()
		if i == 0 {
			// Create selfdestruct contract, sending 42 wei.
			tx, _ := types.SignTx(types.NewContractCreation(0, big.NewInt(42), 100_000, big.NewInt(875000000), selfDestructContract), signer, testKey)
			gen.AddTx(tx)
		} else {
			// Call it.
			tx, _ := types.SignTx(types.NewTransaction(1, selfDestructContractAddr, big.NewInt(0), 100_000, big.NewInt(875000000), nil), signer, testKey)
			gen.AddTx(tx)
		}
	})

	{
		// Check self-destructed contract in the witness.
		// The way 6780 is implemented today, it always SubBalance from the self-destructed contract, and AddBalance
		// to the beneficiary. In this case both addresses are the same, thus this might be optimizable from a gas
		// perspective. But until that happens, we need to honor this "balance reading" adding it to the witness.

		selfDestructContractTreeKey := utils.GetTreeKeyCodeKeccak(selfDestructContractAddr[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediff[1] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructContractTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediff[1][stateDiffIdx].SuffixDiffs[1]
		if balanceStateDiff.Suffix != utils.BalanceLeafKey {
			t.Fatalf("balance invalid suffix")
		}

		// The original balance was 42.
		var fourtyTwo [32]byte
		fourtyTwo[0] = 42
		if *balanceStateDiff.CurrentValue != fourtyTwo {
			t.Fatalf("the pre-state balance before self-destruct must be 42")
		}

		// Note that the SubBalance+AddBalance net effect is a 0 change, so NewValue
		// must be nil.
		if balanceStateDiff.NewValue != nil {
			t.Fatalf("the post-state balance after self-destruct must be empty")
		}
	}
}

func TestProcessVerkleSelfDestructInSameTxWithSelfBeneficiary(t *testing.T) {
	var (
		config = &params.ChainConfig{
			ChainID:                       big.NewInt(69421),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			MuirGlacierBlock:              big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			Ethash:                        new(params.EthashConfig),
			ShanghaiTime:                  u64(0),
			PragueTime:                    u64(0),
			TerminalTotalDifficulty:       common.Big0,
			TerminalTotalDifficultyPassed: true,
			ProofInBlocks:                 true,
		}
		signer     = types.LatestSigner(config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		bcdb       = rawdb.NewMemoryDatabase() // Database for the blockchain
		gendb      = rawdb.NewMemoryDatabase() // Database for the block-generation code, they must be separate as they are path-based.
		coinbase   = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		account1   = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2   = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec      = &Genesis{
			Config: config,
			Alloc: GenesisAlloc{
				coinbase: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account1: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				account2: GenesisAccount{
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   3,
				},
			},
		}
	)
	genesis := gspec.MustCommit(bcdb)
	gspec.MustCommit(gendb)

	// The goal of this test is to test SELFDESTRUCT that happens in a contract execution which is created
	// in **the same** transaction sending the remaining balance to itself.

	selfDestructContract := []byte{
		0x73,                                                                                                                   // PUSH20
		0x3a, 0x22, 0x0f, 0x35, 0x12, 0x52, 0x08, 0x9d, 0x38, 0x5b, 0x29, 0xbe, 0xca, 0x14, 0xe2, 0x7f, 0x20, 0x4c, 0x29, 0x6a, // 0x3a220f351252089d385b29beca14e27f204c296a
		0xFF, // SELFDESTRUCT
	}
	selfDestructContractAddr := common.HexToAddress("3a220f351252089d385b29beca14e27f204c296a")
	_, _, _, statediff := GenerateVerkleChain(gspec.Config, genesis, beacon.New(ethash.NewFaker()), gendb, 1, func(i int, gen *BlockGen) {
		gen.SetPoS()
		tx, _ := types.SignTx(types.NewContractCreation(0, big.NewInt(42), 100_000, big.NewInt(875000000), selfDestructContract), signer, testKey)
		gen.AddTx(tx)
	})

	{ // Check self-destructed contract in the witness
		selfDestructContractTreeKey := utils.GetTreeKeyCodeKeccak(selfDestructContractAddr[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediff[0] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructContractTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediff[0][stateDiffIdx].SuffixDiffs[1]
		if balanceStateDiff.Suffix != utils.BalanceLeafKey {
			t.Fatalf("balance invalid suffix")
		}

		if balanceStateDiff.CurrentValue != nil {
			t.Fatalf("the pre-state balance before must be nil, since the contract didn't exist")
		}

		if balanceStateDiff.NewValue != nil {
			t.Fatalf("the post-state balance after self-destruct must be nil since the contract shouldn't be created at all")
		}
	}
}
