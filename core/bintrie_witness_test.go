// Copyright 2024 The go-ethereum Authors
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
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

var (
	testVerkleChainConfig = &params.ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		Ethash:                  new(params.EthashConfig),
		ShanghaiTime:            u64(0),
		VerkleTime:              u64(0),
		TerminalTotalDifficulty: common.Big0,
		EnableVerkleAtGenesis:   true,
		BlobScheduleConfig: &params.BlobScheduleConfig{
			Verkle: params.DefaultPragueBlobConfig,
		},
	}
)

func TestProcessVerkle(t *testing.T) {
	var (
		code                            = common.FromHex(`6060604052600a8060106000396000f360606040526008565b00`)
		intrinsicContractCreationGas, _ = IntrinsicGas(code, nil, nil, true, true, true, true)
		// A contract creation that calls EXTCODECOPY in the constructor. Used to ensure that the witness
		// will not contain that copied data.
		// Source: https://gist.github.com/gballet/a23db1e1cb4ed105616b5920feb75985
		codeWithExtCodeCopy                = common.FromHex(`0x60806040526040516100109061017b565b604051809103906000f08015801561002c573d6000803e3d6000fd5b506000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555034801561007857600080fd5b5060008067ffffffffffffffff8111156100955761009461024a565b5b6040519080825280601f01601f1916602001820160405280156100c75781602001600182028036833780820191505090505b50905060008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1690506020600083833c81610101906101e3565b60405161010d90610187565b61011791906101a3565b604051809103906000f080158015610133573d6000803e3d6000fd5b50600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505061029b565b60d58061046783390190565b6102068061053c83390190565b61019d816101d9565b82525050565b60006020820190506101b86000830184610194565b92915050565b6000819050602082019050919050565b600081519050919050565b6000819050919050565b60006101ee826101ce565b826101f8846101be565b905061020381610279565b925060208210156102435761023e7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8360200360080261028e565b831692505b5050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600061028582516101d9565b80915050919050565b600082821b905092915050565b6101bd806102aa6000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063f566852414610030575b600080fd5b61003861004e565b6040516100459190610146565b60405180910390f35b6000600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166381ca91d36040518163ffffffff1660e01b815260040160206040518083038186803b1580156100b857600080fd5b505afa1580156100cc573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906100f0919061010a565b905090565b60008151905061010481610170565b92915050565b6000602082840312156101205761011f61016b565b5b600061012e848285016100f5565b91505092915050565b61014081610161565b82525050565b600060208201905061015b6000830184610137565b92915050565b6000819050919050565b600080fd5b61017981610161565b811461018457600080fd5b5056fea2646970667358221220a6a0e11af79f176f9c421b7b12f441356b25f6489b83d38cc828a701720b41f164736f6c63430008070033608060405234801561001057600080fd5b5060b68061001f6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063ab5ed15014602d575b600080fd5b60336047565b604051603e9190605d565b60405180910390f35b60006001905090565b6057816076565b82525050565b6000602082019050607060008301846050565b92915050565b600081905091905056fea26469706673582212203a14eb0d5cd07c277d3e24912f110ddda3e553245a99afc4eeefb2fbae5327aa64736f6c63430008070033608060405234801561001057600080fd5b5060405161020638038061020683398181016040528101906100329190610063565b60018160001c6100429190610090565b60008190555050610145565b60008151905061005d8161012e565b92915050565b60006020828403121561007957610078610129565b5b60006100878482850161004e565b91505092915050565b600061009b826100f0565b91506100a6836100f0565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038211156100db576100da6100fa565b5b828201905092915050565b6000819050919050565b6000819050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600080fd5b610137816100e6565b811461014257600080fd5b50565b60b3806101536000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c806381ca91d314602d575b600080fd5b60336047565b604051603e9190605a565b60405180910390f35b60005481565b6054816073565b82525050565b6000602082019050606d6000830184604d565b92915050565b600081905091905056fea26469706673582212209bff7098a2f526de1ad499866f27d6d0d6f17b74a413036d6063ca6a0998ca4264736f6c63430008070033`)
		intrinsicCodeWithExtCodeCopyGas, _ = IntrinsicGas(codeWithExtCodeCopy, nil, nil, true, true, true, true)
		signer                             = types.LatestSigner(testVerkleChainConfig)
		testKey, _                         = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		bcdb                               = rawdb.NewMemoryDatabase() // Database for the blockchain
		coinbase                           = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		gspec                              = &Genesis{
			Config: testVerkleChainConfig,
			Alloc: GenesisAlloc{
				coinbase: {
					Balance: big.NewInt(1000000000000000000), // 1 ether
					Nonce:   0,
				},
				params.BeaconRootsAddress:        {Nonce: 1, Code: params.BeaconRootsCode, Balance: common.Big0},
				params.HistoryStorageAddress:     {Nonce: 1, Code: params.HistoryStorageCode, Balance: common.Big0},
				params.WithdrawalQueueAddress:    {Nonce: 1, Code: params.WithdrawalQueueCode, Balance: common.Big0},
				params.ConsolidationQueueAddress: {Nonce: 1, Code: params.ConsolidationQueueCode, Balance: common.Big0},
			},
		}
	)
	// Verkle trees use the snapshot, which must be enabled before the
	// data is saved into the tree+database.
	// genesis := gspec.MustCommit(bcdb, triedb)
	options := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	options.SnapshotLimit = 0
	blockchain, _ := NewBlockChain(bcdb, gspec, beacon.New(ethash.NewFaker()), options)
	defer blockchain.Stop()

	txCost1 := params.TxGas
	txCost2 := params.TxGas
	contractCreationCost := intrinsicContractCreationGas +
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + params.WitnessBranchReadCost + params.WitnessBranchWriteCost + /* creation */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* creation with value */
		739 /* execution costs */
	codeWithExtCodeCopyGas := intrinsicCodeWithExtCodeCopyGas +
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + params.WitnessBranchReadCost + params.WitnessBranchWriteCost + /* creation (tx) */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + params.WitnessBranchReadCost + params.WitnessBranchWriteCost + /* creation (CREATE at pc=0x20) */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* write code hash */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #0 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #1 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #2 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #3 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #4 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #5 */
		params.WitnessChunkReadCost + /* SLOAD in constructor */
		params.WitnessChunkWriteCost + /* SSTORE in constructor */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + params.WitnessBranchReadCost + params.WitnessBranchWriteCost + /* creation (CREATE at PC=0x121) */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* write code hash */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #0 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #1 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #2 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #3 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #4 */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* code chunk #5 */
		params.WitnessChunkReadCost + /* SLOAD in constructor */
		params.WitnessChunkWriteCost + /* SSTORE in constructor */
		params.WitnessChunkReadCost + params.WitnessChunkWriteCost + /* write code hash for tx creation */
		15*(params.WitnessChunkReadCost+params.WitnessChunkWriteCost) + /* code chunks #0..#14 */
		uint64(4844) /* execution costs */
	blockGasUsagesExpected := []uint64{
		txCost1*2 + txCost2,
		txCost1*2 + txCost2 + contractCreationCost + codeWithExtCodeCopyGas,
	}
	_, chain, _ := GenerateChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 2, func(i int, gen *BlockGen) {
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
			tx, _ = types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 6,
				Value:    big.NewInt(16),
				Gas:      3000000,
				GasPrice: big.NewInt(875000000),
				Data:     code,
			})
			gen.AddTx(tx)

			tx, _ = types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 7,
				Value:    big.NewInt(0),
				Gas:      3000000,
				GasPrice: big.NewInt(875000000),
				Data:     codeWithExtCodeCopy,
			})
			gen.AddTx(tx)
		}
	})

	for i, b := range chain {
		fmt.Printf("%d %x\n", i, b.Root())
	}
	endnum, err := blockchain.InsertChain(chain)
	if err != nil {
		t.Fatalf("block %d imported with error: %v", endnum, err)
	}

	for i := range 2 {
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

func TestProcessParentBlockHash(t *testing.T) {
	// This test uses blocks where,
	// block 1 parent hash is 0x0100....
	// block 2 parent hash is 0x0200....
	// etc
	checkBlockHashes := func(statedb *state.StateDB, isVerkle bool) {
		statedb.SetNonce(params.HistoryStorageAddress, 1, tracing.NonceChangeUnspecified)
		statedb.SetCode(params.HistoryStorageAddress, params.HistoryStorageCode, tracing.CodeChangeUnspecified)
		// Process n blocks, from 1 .. num
		var num = 2
		for i := 1; i <= num; i++ {
			header := &types.Header{ParentHash: common.Hash{byte(i)}, Number: big.NewInt(int64(i)), Difficulty: new(big.Int)}
			chainConfig := params.MergedTestChainConfig
			if isVerkle {
				chainConfig = testVerkleChainConfig
			}
			vmContext := NewEVMBlockContext(header, nil, new(common.Address))
			evm := vm.NewEVM(vmContext, statedb, chainConfig, vm.Config{})
			ProcessParentBlockHash(header.ParentHash, evm)
		}
		// Read block hashes for block 0 .. num-1
		for i := 0; i < num; i++ {
			have, want := getContractStoredBlockHash(statedb, uint64(i), isVerkle), common.Hash{byte(i + 1)}
			if have != want {
				t.Errorf("block %d, verkle=%v, have parent hash %v, want %v", i, isVerkle, have, want)
			}
		}
	}
	t.Run("MPT", func(t *testing.T) {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		checkBlockHashes(statedb, false)
	})
	t.Run("Verkle", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		cacheConfig := DefaultConfig().WithStateScheme(rawdb.PathScheme)
		cacheConfig.SnapshotLimit = 0
		triedb := triedb.NewDatabase(db, cacheConfig.triedbConfig(true))
		statedb, _ := state.New(types.EmptyVerkleHash, state.NewDatabase(triedb, nil))
		checkBlockHashes(statedb, true)
	})
}

// getContractStoredBlockHash is a utility method which reads the stored parent blockhash for block 'number'
func getContractStoredBlockHash(statedb *state.StateDB, number uint64, isVerkle bool) common.Hash {
	ringIndex := number % params.HistoryServeWindow
	var key common.Hash
	binary.BigEndian.PutUint64(key[24:], ringIndex)
	if isVerkle {
		return statedb.GetState(params.HistoryStorageAddress, key)
	}
	return statedb.GetState(params.HistoryStorageAddress, key)
}
