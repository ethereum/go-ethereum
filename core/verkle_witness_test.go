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
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"slices"
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
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-verkle"
	"github.com/holiman/uint256"
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
		// TODO uncomment when proof generation is merged
		// ProofInBlocks:                 true,
	}
	testKaustinenLikeChainConfig = &params.ChainConfig{
		ChainID:                 big.NewInt(69420),
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
	cacheConfig := DefaultCacheConfigWithScheme(rawdb.PathScheme)
	cacheConfig.SnapshotLimit = 0
	blockchain, _ := NewBlockChain(bcdb, cacheConfig, gspec, nil, beacon.New(ethash.NewFaker()), vm.Config{}, nil)
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
	_, _, chain, _, proofs, statediffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 2, func(i int, gen *BlockGen) {
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

	// Check proof for both blocks
	err := verkle.Verify(proofs[0], gspec.ToBlock().Root().Bytes(), chain[0].Root().Bytes(), statediffs[0])
	if err != nil {
		t.Fatal(err)
	}
	err = verkle.Verify(proofs[1], chain[0].Root().Bytes(), chain[1].Root().Bytes(), statediffs[1])
	if err != nil {
		t.Fatal(err)
	}

	t.Log("verified verkle proof, inserting blocks into the chain")

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

func TestProcessParentBlockHash(t *testing.T) {
	// This test uses blocks where,
	// block 1 parent hash is 0x0100....
	// block 2 parent hash is 0x0200....
	// etc
	checkBlockHashes := func(statedb *state.StateDB, isVerkle bool) {
		statedb.SetNonce(params.HistoryStorageAddress, 1, tracing.NonceChangeUnspecified)
		statedb.SetCode(params.HistoryStorageAddress, params.HistoryStorageCode)
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
		cacheConfig := DefaultCacheConfigWithScheme(rawdb.PathScheme)
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

// TestProcessVerkleInvalidContractCreation checks for several modes of contract creation failures
func TestProcessVerkleInvalidContractCreation(t *testing.T) {
	var (
		account1 = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2 = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec    = verkleTestGenesis(testKaustinenLikeChainConfig)
	)
	// slightly modify it to suit the live txs from the testnet
	gspec.Alloc[account2] = types.Account{
		Balance: big.NewInt(1000000000000000000), // 1 ether
		Nonce:   1,
	}

	// Create two blocks that reproduce what is happening on kaustinen.
	// - The first block contains two failing contract creation transactions, that
	//   write to storage before they revert.
	//
	// - The second block contains a single failing contract creation transaction,
	//   that fails right off the bat.
	genesisH, _, chain, _, _, statediffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 2, func(i int, gen *BlockGen) {
		gen.SetPoS()

		if i == 0 {
			for _, rlpData := range []string{
				// SSTORE at slot 41 and reverts
				"f8d48084479c2c18830186a08080b8806000602955bda3f9600060ca55600060695523b360006039551983576000601255b0620c2fde2c592ac2600060bc55e0ac6000606455a63e22600060e655eb607e605c5360a2605d5360c7605e53601d605f5360eb606053606b606153608e60625360816063536079606453601e60655360fc60665360b7606753608b60685383021e7ca0cc20c65a97d2e526b8ec0f4266e8b01bdcde43b9aeb59d8bfb44e8eb8119c109a07a8e751813ae1b2ce734960dbc39a4f954917d7822a2c5d1dca18b06c584131f",
				// SSTORE at slot 133 and reverts
				"02f8db83010f2c01843b9aca0084479c2c18830186a08080b88060006085553fad6000600a55600060565555600060b55506600060cf557f1b8b38183e7bd1bdfaa7123c5a4976e54cce0e42049d841411978fd3595e25c66019527f0538943712953cf08900aae40222a40b2d5a4ac8075ad8cf0870e2be307edbb96039527f9f3174ff85024747041ae7a611acffb987c513c088d90ab288aec080a0cd6ac65ce2cb0a912371f6b5a551ba8caffc22ec55ad4d3cb53de41d05eb77b6a02e0dfe8513dfa6ec7bfd7eda6f5c0dac21b39b982436045e128cec46cfd3f960",
				// this one is a simple transfer that succeeds, necessary to get the correct nonce in the other block.
				"f8e80184479c2c18830186a094bbbbde4ca27f83fc18aa108170547ff57675936a80b8807ff71f7c15faadb969a76a5f54a81a0117e1e743cb7f24e378eda28442ea4c6eb6604a527fb5409e5718d44e23bfffac926e5ea726067f772772e7e19446acba0c853f62f5606a526020608a536088608b536039608c536004608d5360af608e537f7f7675d9f210e0a61564e6d11e7cd75f5bc9009ac9f6b94a0fc63035441a83021e7ba04a4a172d81ebb02847829b76a387ac09749c8b65668083699abe20c887fb9efca07c5b1a990702ec7b31a5e8e3935cd9a77649f8c25a84131229e24ab61aec6093",
			} {
				var tx = new(types.Transaction)
				if err := tx.UnmarshalBinary(common.Hex2Bytes(rlpData)); err != nil {
					t.Fatal(err)
				}
				gen.AddTx(tx)
			}
		} else {
			var tx = new(types.Transaction)
			// immediately reverts
			if err := tx.UnmarshalBinary(common.Hex2Bytes("01f8d683010f2c028443ad7d0e830186a08080b880b00e7fa3c849dce891cce5fae8a4c46cbb313d6aec0c0ffe7863e05fb7b22d4807674c6055527ffbfcb0938f3e18f7937aa8fa95d880afebd5c4cec0d85186095832d03c85cf8a60755260ab60955360cf6096536066609753606e60985360fa609953609e609a53608e609b536024609c5360f6609d536072609e5360a4609fc080a08fc6f7101f292ff1fb0de8ac69c2d320fbb23bfe61cf327173786ea5daee6e37a044c42d91838ef06646294bf4f9835588aee66243b16a66a2da37641fae4c045f")); err != nil {
				t.Fatal(err)
			}
			gen.AddTx(tx)
		}
	})

	tx1ContractAddress := crypto.CreateAddress(account1, 0)
	tx1ContractStem := utils.GetTreeKey(tx1ContractAddress[:], uint256.NewInt(0), 105)
	tx1ContractStem = tx1ContractStem[:31]

	tx2ContractAddress := crypto.CreateAddress(account2, 1)
	tx2SlotKey := [32]byte{}
	tx2SlotKey[31] = 133
	tx2ContractStem := utils.StorageSlotKey(tx2ContractAddress[:], tx2SlotKey[:])
	tx2ContractStem = tx2ContractStem[:31]

	eip2935Stem := utils.GetTreeKey(params.HistoryStorageAddress[:], uint256.NewInt(0), 0)
	eip2935Stem = eip2935Stem[:31]

	// Check that the witness contains what we expect: a storage entry for each of the two contract
	// creations that failed: one at 133 for the 2nd tx, and one at 105 for the first tx.
	for _, stemStateDiff := range statediffs[0] {
		// Check that the slot number 133, which is overflowing the account header,
		// is present. Note that the offset of the 2nd group (first group after the
		// header) is skipping the first 64 values, hence we still have an offset
		// of 133, and not 133 - 64.
		if bytes.Equal(stemStateDiff.Stem[:], tx2ContractStem[:]) {
			for _, suffixDiff := range stemStateDiff.SuffixDiffs {
				if suffixDiff.Suffix != 133 {
					t.Fatalf("invalid suffix diff found for %x in block #1: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
				}
				if suffixDiff.CurrentValue != nil {
					t.Fatalf("invalid prestate value found for %x in block #1: %v != nil\n", stemStateDiff.Stem, suffixDiff.CurrentValue)
				}
				if suffixDiff.NewValue != nil {
					t.Fatalf("invalid poststate value found for %x in block #1: %v != nil\n", stemStateDiff.Stem, suffixDiff.NewValue)
				}
			}
		} else if bytes.Equal(stemStateDiff.Stem[:], tx1ContractStem) {
			// For this contract creation, check that only the account header and storage slot 41
			// are found in the witness.
			for _, suffixDiff := range stemStateDiff.SuffixDiffs {
				if suffixDiff.Suffix != 105 && suffixDiff.Suffix != 0 && suffixDiff.Suffix != 1 {
					t.Fatalf("invalid suffix diff found for %x in block #1: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
				}
			}
		} else if bytes.Equal(stemStateDiff.Stem[:], eip2935Stem) {
			// Check the eip 2935 group of leaves.
			// Check that only one leaf was accessed, and is present in the witness.
			if len(stemStateDiff.SuffixDiffs) > 1 {
				t.Fatalf("invalid suffix diff count found for BLOCKHASH contract: %d != 1", len(stemStateDiff.SuffixDiffs))
			}
			// Check that this leaf is the first storage slot
			if stemStateDiff.SuffixDiffs[0].Suffix != 64 {
				t.Fatalf("invalid suffix diff value found for BLOCKHASH contract: %d != 64", stemStateDiff.SuffixDiffs[0].Suffix)
			}
			// check that the prestate value is nil and that the poststate value isn't.
			if stemStateDiff.SuffixDiffs[0].CurrentValue != nil {
				t.Fatalf("non-nil current value in BLOCKHASH contract insert: %x", stemStateDiff.SuffixDiffs[0].CurrentValue)
			}
			if stemStateDiff.SuffixDiffs[0].NewValue == nil {
				t.Fatalf("nil new value in BLOCKHASH contract insert")
			}
			if *stemStateDiff.SuffixDiffs[0].NewValue != genesisH {
				t.Fatalf("invalid BLOCKHASH value: %x != %x", *stemStateDiff.SuffixDiffs[0].NewValue, genesisH)
			}
		} else {
			// For all other entries present in the witness, check that nothing beyond
			// the account header was accessed.
			for _, suffixDiff := range stemStateDiff.SuffixDiffs {
				if suffixDiff.Suffix > 2 {
					t.Fatalf("invalid suffix diff found for %x in block #1: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
				}
			}
		}
	}

	// Check that no account has a value above 4 in the 2nd block as no storage nor
	// code should make it to the witness.
	for _, stemStateDiff := range statediffs[1] {
		for _, suffixDiff := range stemStateDiff.SuffixDiffs {
			if bytes.Equal(stemStateDiff.Stem[:], eip2935Stem) {
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
				if *stemStateDiff.SuffixDiffs[0].NewValue != chain[0].Hash() {
					t.Fatalf("invalid post state value for BLOCKHASH contract at block #2: %x != %x", chain[0].Hash(), (*stemStateDiff.SuffixDiffs[0].NewValue)[:])
				}
			} else if suffixDiff.Suffix > 4 {
				t.Fatalf("invalid suffix diff found for %x in block #2: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
			}
		}
	}
}

func verkleTestGenesis(config *params.ChainConfig) *Genesis {
	var (
		coinbase = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		account1 = common.HexToAddress("0x687704DB07e902e9A8B3754031D168D46E3D586e")
		account2 = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
	)
	return &Genesis{
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
			params.BeaconRootsAddress:        {Nonce: 1, Code: params.BeaconRootsCode, Balance: common.Big0},
			params.HistoryStorageAddress:     {Nonce: 1, Code: params.HistoryStorageCode, Balance: common.Big0},
			params.WithdrawalQueueAddress:    {Nonce: 1, Code: params.WithdrawalQueueCode, Balance: common.Big0},
			params.ConsolidationQueueAddress: {Nonce: 1, Code: params.ConsolidationQueueCode, Balance: common.Big0},
		},
	}
}

// TestProcessVerkleContractWithEmptyCode checks that the witness contains all valid
// entries, if the initcode returns an empty code.
func TestProcessVerkleContractWithEmptyCode(t *testing.T) {
	// The test txs were taken from a secondary testnet with chain id 69421
	config := *testKaustinenLikeChainConfig
	config.ChainID.SetUint64(69421)
	gspec := verkleTestGenesis(&config)

	genesisH, _, _, _, _, statediffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 1, func(i int, gen *BlockGen) {
		gen.SetPoS()
		var tx types.Transaction
		// a transaction that does some PUSH1n but returns a 0-sized contract
		txpayload := common.Hex2Bytes("02f8db83010f2d03843b9aca008444cf6a05830186a08080b8807fdfbbb59f2371a76485ce557fd0de00c298d3ede52a3eab56d35af674eb49ec5860335260826053536001605453604c60555360f3605653606060575360446058536096605953600c605a5360df605b5360f3605c5360fb605d53600c605e53609a605f53607f60605360fe606153603d60625360f4606353604b60645360cac001a0486b6dc55b8a311568b7239a2cae1d77e7446dba71df61eaafd53f73820a138fa010bd48a45e56133ac4c5645142c2ea48950d40eb35050e9510b6bad9e15c5865")
		if err := tx.UnmarshalBinary(txpayload); err != nil {
			t.Fatal(err)
		}
		gen.AddTx(&tx)
	})

	eip2935Stem := utils.GetTreeKey(params.HistoryStorageAddress[:], uint256.NewInt(0), 0)
	eip2935Stem = eip2935Stem[:31]

	for _, stemStateDiff := range statediffs[0] {
		// Handle the case of the history contract: make sure only the correct
		// slots are added to the witness.
		if bytes.Equal(stemStateDiff.Stem[:], eip2935Stem) {
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
			if *stemStateDiff.SuffixDiffs[0].NewValue != genesisH {
				t.Fatalf("invalid BLOCKHASH value: %x != %x", *stemStateDiff.SuffixDiffs[0].NewValue, genesisH)
			}
		} else {
			for _, suffixDiff := range stemStateDiff.SuffixDiffs {
				if suffixDiff.Suffix > 2 {
					// if d8898012c484fb48610ecb7963886339207dab004bce968b007b616ffa18e0 shows up, it means that the PUSHn
					// in the transaction above added entries into the witness, when they should not have since they are
					// part of a contract deployment.
					t.Fatalf("invalid suffix diff found for %x in block #1: %d\n", stemStateDiff.Stem, suffixDiff.Suffix)
				}
			}
		}
	}
}

// TestProcessVerkleExtCodeHashOpcode verifies that calling EXTCODEHASH on another
// deployed contract, creates all the right entries in the witness.
func TestProcessVerkleExtCodeHashOpcode(t *testing.T) {
	// The test txs were taken from a secondary testnet with chain id 69421
	config := *testKaustinenLikeChainConfig
	config.ChainID.SetUint64(69421)

	var (
		signer     = types.LatestSigner(&config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		gspec      = verkleTestGenesis(&config)
	)
	dummyContract := []byte{
		byte(vm.PUSH1), 2,
		byte(vm.PUSH1), 12,
		byte(vm.PUSH1), 0x00,
		byte(vm.CODECOPY),

		byte(vm.PUSH1), 2,
		byte(vm.PUSH1), 0x00,
		byte(vm.RETURN),

		byte(vm.PUSH1), 42,
	}
	deployer := crypto.PubkeyToAddress(testKey.PublicKey)
	dummyContractAddr := crypto.CreateAddress(deployer, 0)

	// contract that calls EXTCODEHASH on the dummy contract
	extCodeHashContract := []byte{
		byte(vm.PUSH1), 22,
		byte(vm.PUSH1), 12,
		byte(vm.PUSH1), 0x00,
		byte(vm.CODECOPY),

		byte(vm.PUSH1), 22,
		byte(vm.PUSH1), 0x00,
		byte(vm.RETURN),

		byte(vm.PUSH20),
		0x3a, 0x22, 0x0f, 0x35, 0x12, 0x52, 0x08, 0x9d, 0x38, 0x5b, 0x29, 0xbe, 0xca, 0x14, 0xe2, 0x7f, 0x20, 0x4c, 0x29, 0x6a,
		byte(vm.EXTCODEHASH),
	}
	extCodeHashContractAddr := crypto.CreateAddress(deployer, 1)

	_, _, _, _, _, statediffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 2, func(i int, gen *BlockGen) {
		gen.SetPoS()

		if i == 0 {
			// Create dummy contract.
			tx, _ := types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 0,
				Value:    big.NewInt(0),
				Gas:      100_000,
				GasPrice: big.NewInt(875000000),
				Data:     dummyContract,
			})
			gen.AddTx(tx)

			// Create contract with EXTCODEHASH opcode.
			tx, _ = types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 1,
				Value:    big.NewInt(0),
				Gas:      100_000,
				GasPrice: big.NewInt(875000000),
				Data:     extCodeHashContract})
			gen.AddTx(tx)
		} else {
			tx, _ := types.SignTx(types.NewTransaction(2, extCodeHashContractAddr, big.NewInt(0), 100_000, big.NewInt(875000000), nil), signer, testKey)
			gen.AddTx(tx)
		}
	})

	contractKeccakTreeKey := utils.CodeHashKey(dummyContractAddr[:])

	var stateDiffIdx = -1
	for i, stemStateDiff := range statediffs[1] {
		if bytes.Equal(stemStateDiff.Stem[:], contractKeccakTreeKey[:31]) {
			stateDiffIdx = i
			break
		}
	}
	if stateDiffIdx == -1 {
		t.Fatalf("no state diff found for stem")
	}

	codeHashStateDiff := statediffs[1][stateDiffIdx].SuffixDiffs[0]
	// Check location of code hash was accessed
	if codeHashStateDiff.Suffix != utils.CodeHashLeafKey {
		t.Fatalf("code hash invalid suffix")
	}
	// check the code hash wasn't present in the prestate, as
	// the contract was deployed in this block.
	if codeHashStateDiff.CurrentValue == nil {
		t.Fatalf("codeHash.CurrentValue must not be empty")
	}
	// check the poststate value corresponds to the code hash
	// of the deployed contract.
	expCodeHash := crypto.Keccak256Hash(dummyContract[12:])
	if *codeHashStateDiff.CurrentValue != expCodeHash {
		t.Fatalf("codeHash.CurrentValue unexpected code hash")
	}
	if codeHashStateDiff.NewValue != nil {
		t.Fatalf("codeHash.NewValue must be nil")
	}
}

// TestProcessVerkleBalanceOpcode checks that calling balance
// on another contract will add the correct entries to the witness.
func TestProcessVerkleBalanceOpcode(t *testing.T) {
	// The test txs were taken from a secondary testnet with chain id 69421
	config := *testKaustinenLikeChainConfig
	config.ChainID.SetUint64(69421)

	var (
		signer     = types.LatestSigner(&config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		account2   = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec      = verkleTestGenesis(&config)
	)
	_, _, _, _, _, statediffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 1, func(i int, gen *BlockGen) {
		gen.SetPoS()
		txData := slices.Concat(
			[]byte{byte(vm.PUSH20)},
			common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d").Bytes(),
			[]byte{byte(vm.BALANCE)})

		tx, _ := types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 0,
			Value:    big.NewInt(0),
			Gas:      100_000,
			GasPrice: big.NewInt(875000000),
			Data:     txData})
		gen.AddTx(tx)
	})

	account2BalanceTreeKey := utils.BasicDataKey(account2[:])

	var stateDiffIdx = -1
	for i, stemStateDiff := range statediffs[0] {
		if bytes.Equal(stemStateDiff.Stem[:], account2BalanceTreeKey[:31]) {
			stateDiffIdx = i
			break
		}
	}
	if stateDiffIdx == -1 {
		t.Fatalf("no state diff found for stem")
	}

	var zero [32]byte
	balanceStateDiff := statediffs[0][stateDiffIdx].SuffixDiffs[0]
	if balanceStateDiff.Suffix != utils.BasicDataLeafKey {
		t.Fatalf("invalid suffix diff")
	}
	// check the prestate balance wasn't 0 or missing
	if balanceStateDiff.CurrentValue == nil || *balanceStateDiff.CurrentValue == zero {
		t.Fatalf("invalid current value %v", *balanceStateDiff.CurrentValue)
	}
	// check that the poststate witness value for the balance is nil,
	// meaning that it didn't get updated.
	if balanceStateDiff.NewValue != nil {
		t.Fatalf("invalid new value")
	}
}

// TestProcessVerkleSelfDestructInSeparateTx controls the contents of the witness after
// a non-eip6780-compliant selfdestruct occurs.
func TestProcessVerkleSelfDestructInSeparateTx(t *testing.T) {
	// The test txs were taken from a secondary testnet with chain id 69421
	config := *testKaustinenLikeChainConfig
	config.ChainID.SetUint64(69421)

	var (
		signer     = types.LatestSigner(&config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		account2   = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec      = verkleTestGenesis(&config)
	)

	// runtime code: selfdestruct ( 0x6177843db3138ae69679A54b95cf345ED759450d )
	runtimeCode := slices.Concat(
		[]byte{byte(vm.PUSH20)},
		account2.Bytes(),
		[]byte{byte(vm.SELFDESTRUCT)})

	//The goal of this test is to test SELFDESTRUCT that happens in a contract
	// execution which is created in a previous transaction.
	selfDestructContract := slices.Concat([]byte{
		byte(vm.PUSH1), byte(len(runtimeCode)),
		byte(vm.PUSH1), 12,
		byte(vm.PUSH1), 0x00,
		byte(vm.CODECOPY), // Codecopy( to-offset: 0, code offset: 12, length: 22 )

		byte(vm.PUSH1), byte(len(runtimeCode)),
		byte(vm.PUSH1), 0x00,
		byte(vm.RETURN), // Return ( 0 : len(runtimecode)
	},
		runtimeCode)

	deployer := crypto.PubkeyToAddress(testKey.PublicKey)
	contract := crypto.CreateAddress(deployer, 0)

	_, _, _, _, _, statediffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 2, func(i int, gen *BlockGen) {
		gen.SetPoS()

		if i == 0 {
			// Create selfdestruct contract, sending 42 wei.
			tx, _ := types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 0,
				Value:    big.NewInt(42),
				Gas:      100_000,
				GasPrice: big.NewInt(875000000),
				Data:     selfDestructContract,
			})
			gen.AddTx(tx)
		} else {
			// Call it.
			tx, _ := types.SignTx(types.NewTransaction(1, contract, big.NewInt(0), 100_000, big.NewInt(875000000), nil), signer, testKey)
			gen.AddTx(tx)
		}
	})

	var zero [32]byte
	{ // Check self-destructed contract in the witness
		selfDestructContractTreeKey := utils.CodeHashKey(contract[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediffs[1] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructContractTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediffs[1][stateDiffIdx].SuffixDiffs[0]
		if balanceStateDiff.Suffix != utils.BasicDataLeafKey {
			t.Fatalf("balance invalid suffix")
		}

		// The original balance was 42.
		var oldBalance [16]byte
		oldBalance[15] = 42
		if !bytes.Equal((*balanceStateDiff.CurrentValue)[utils.BasicDataBalanceOffset:], oldBalance[:]) {
			t.Fatalf("the pre-state balance before self-destruct must be %x, got %x", oldBalance, *balanceStateDiff.CurrentValue)
		}

		// The new balance must be 0.
		if !bytes.Equal((*balanceStateDiff.NewValue)[utils.BasicDataBalanceOffset:], zero[utils.BasicDataBalanceOffset:]) {
			t.Fatalf("the post-state balance after self-destruct must be 0")
		}
	}
	{ // Check self-destructed target in the witness.
		selfDestructTargetTreeKey := utils.CodeHashKey(account2[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediffs[1] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructTargetTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediffs[1][stateDiffIdx].SuffixDiffs[0]
		if balanceStateDiff.Suffix != utils.BasicDataLeafKey {
			t.Fatalf("balance invalid suffix")
		}
		if balanceStateDiff.CurrentValue == nil {
			t.Fatalf("codeHash.CurrentValue must not be empty")
		}
		if balanceStateDiff.NewValue == nil {
			t.Fatalf("codeHash.NewValue must not be empty")
		}
		preStateBalance := binary.BigEndian.Uint64(balanceStateDiff.CurrentValue[utils.BasicDataBalanceOffset+8:])
		postStateBalance := binary.BigEndian.Uint64(balanceStateDiff.NewValue[utils.BasicDataBalanceOffset+8:])
		if postStateBalance-preStateBalance != 42 {
			t.Fatalf("the post-state balance after self-destruct must be 42, got %d-%d=%d", postStateBalance, preStateBalance, postStateBalance-preStateBalance)
		}
	}
}

// TestProcessVerkleSelfDestructInSeparateTx controls the contents of the witness after
// a eip6780-compliant selfdestruct occurs.
func TestProcessVerkleSelfDestructInSameTx(t *testing.T) {
	// The test txs were taken from a secondary testnet with chain id 69421
	config := *testKaustinenLikeChainConfig
	config.ChainID.SetUint64(69421)

	var (
		signer     = types.LatestSigner(&config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		account2   = common.HexToAddress("0x6177843db3138ae69679A54b95cf345ED759450d")
		gspec      = verkleTestGenesis(&config)
	)

	// The goal of this test is to test SELFDESTRUCT that happens in a contract
	// execution which is created in **the same** transaction sending the remaining
	// balance to an external (i.e: not itself) account.

	selfDestructContract := slices.Concat(
		[]byte{byte(vm.PUSH20)},
		account2.Bytes(),
		[]byte{byte(vm.SELFDESTRUCT)})
	deployer := crypto.PubkeyToAddress(testKey.PublicKey)
	contract := crypto.CreateAddress(deployer, 0)

	_, _, _, _, _, statediffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 1, func(i int, gen *BlockGen) {
		gen.SetPoS()
		tx, _ := types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 0,
			Value:    big.NewInt(42),
			Gas:      100_000,
			GasPrice: big.NewInt(875000000),
			Data:     selfDestructContract,
		})
		gen.AddTx(tx)
	})

	{ // Check self-destructed contract in the witness
		selfDestructContractTreeKey := utils.CodeHashKey(contract[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediffs[0] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructContractTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediffs[0][stateDiffIdx].SuffixDiffs[0]
		if balanceStateDiff.Suffix != utils.BasicDataLeafKey {
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
		selfDestructTargetTreeKey := utils.CodeHashKey(account2[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediffs[0] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructTargetTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatalf("no state diff found for stem")
		}

		balanceStateDiff := statediffs[0][stateDiffIdx].SuffixDiffs[0]
		if balanceStateDiff.Suffix != utils.BasicDataLeafKey {
			t.Fatalf("balance invalid suffix")
		}
		if balanceStateDiff.CurrentValue == nil {
			t.Fatalf("codeHash.CurrentValue must not be empty")
		}
		if balanceStateDiff.NewValue == nil {
			t.Fatalf("codeHash.NewValue must not be empty")
		}
		preStateBalance := binary.BigEndian.Uint64(balanceStateDiff.CurrentValue[utils.BasicDataBalanceOffset+8:])
		postStateBalance := binary.BigEndian.Uint64(balanceStateDiff.NewValue[utils.BasicDataBalanceOffset+8:])
		if postStateBalance-preStateBalance != 42 {
			t.Fatalf("the post-state balance after self-destruct must be 42. got %d", postStateBalance)
		}
	}
}

// TestProcessVerkleSelfDestructInSeparateTxWithSelfBeneficiary checks the content of the witness
// if a selfdestruct occurs in a different tx than the one that created it, but the beneficiary
// is the selfdestructed account.
func TestProcessVerkleSelfDestructInSeparateTxWithSelfBeneficiary(t *testing.T) {
	// The test txs were taken from a secondary testnet with chain id 69421
	config := *testKaustinenLikeChainConfig
	config.ChainID.SetUint64(69421)

	var (
		signer     = types.LatestSigner(&config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		gspec      = verkleTestGenesis(&config)
	)
	// The goal of this test is to test SELFDESTRUCT that happens in a contract
	// execution which is created in a *previous* transaction sending the remaining
	// balance to itself.
	selfDestructContract := []byte{
		byte(vm.PUSH1), 2, // PUSH1 2
		byte(vm.PUSH1), 10, // PUSH1 12
		byte(vm.PUSH0),    // PUSH0
		byte(vm.CODECOPY), // Codecopy ( to offset 0, code@offset: 10, length: 2)

		byte(vm.PUSH1), 22,
		byte(vm.PUSH0),
		byte(vm.RETURN), // RETURN( memory[0:2] )

		// Deployed code
		byte(vm.ADDRESS),
		byte(vm.SELFDESTRUCT),
	}
	deployer := crypto.PubkeyToAddress(testKey.PublicKey)
	contract := crypto.CreateAddress(deployer, 0)

	_, _, _, _, _, statediffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 2, func(i int, gen *BlockGen) {
		gen.SetPoS()
		if i == 0 {
			// Create self-destruct contract, sending 42 wei.
			tx, _ := types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 0,
				Value:    big.NewInt(42),
				Gas:      100_000,
				GasPrice: big.NewInt(875000000),
				Data:     selfDestructContract,
			})
			gen.AddTx(tx)
		} else {
			// Call it.
			tx, _ := types.SignTx(types.NewTransaction(1, contract, big.NewInt(0), 100_000, big.NewInt(875000000), nil), signer, testKey)
			gen.AddTx(tx)
		}
	})

	{
		// Check self-destructed contract in the witness.
		// The way 6780 is implemented today, it always SubBalance from the self-destructed contract, and AddBalance
		// to the beneficiary. In this case both addresses are the same, thus this might be optimizable from a gas
		// perspective. But until that happens, we need to honor this "balance reading" adding it to the witness.

		selfDestructContractTreeKey := utils.CodeHashKey(contract[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range statediffs[1] {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructContractTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatal("no state diff found for stem")
		}

		balanceStateDiff := statediffs[1][stateDiffIdx].SuffixDiffs[0]
		if balanceStateDiff.Suffix != utils.BasicDataLeafKey {
			t.Fatal("balance invalid suffix")
		}

		// The original balance was 42.
		var oldBalance [16]byte
		oldBalance[15] = 42
		if !bytes.Equal((*balanceStateDiff.CurrentValue)[utils.BasicDataBalanceOffset:], oldBalance[:]) {
			t.Fatal("the pre-state balance before self-destruct must be 42")
		}

		// Note that the SubBalance+AddBalance net effect is a 0 change, so NewValue
		// must be nil.
		if balanceStateDiff.NewValue != nil {
			t.Fatal("the post-state balance after self-destruct must be empty")
		}
	}
}

// TestProcessVerkleSelfDestructInSameTxWithSelfBeneficiary checks the content of the witness
// if a selfdestruct occurs in the same tx as the one that created it, but the beneficiary
// is the selfdestructed account.
func TestProcessVerkleSelfDestructInSameTxWithSelfBeneficiary(t *testing.T) {
	// The test txs were taken from a secondary testnet with chain id 69421
	config := *testKaustinenLikeChainConfig
	config.ChainID.SetUint64(69421)

	var (
		signer     = types.LatestSigner(&config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		gspec      = verkleTestGenesis(&config)
		deployer   = crypto.PubkeyToAddress(testKey.PublicKey)
		contract   = crypto.CreateAddress(deployer, 0)
	)

	// The goal of this test is to test SELFDESTRUCT that happens while executing
	// the init code of a contract creation, that occurs in **the same** transaction.
	// The balance is sent to itself.
	t.Logf("Contract: %v", contract.String())

	selfDestructContract := []byte{byte(vm.ADDRESS), byte(vm.SELFDESTRUCT)}

	_, _, _, _, _, stateDiffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 1, func(i int, gen *BlockGen) {
		gen.SetPoS()
		tx, _ := types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 0,
			Value:    big.NewInt(42),
			Gas:      100_000,
			GasPrice: big.NewInt(875000000),
			Data:     selfDestructContract,
		})
		gen.AddTx(tx)
	})
	stateDiff := stateDiffs[0] // state difference of block 1

	{ // Check self-destructed contract in the witness
		selfDestructContractTreeKey := utils.CodeHashKey(contract[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range stateDiff {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructContractTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatal("no state diff found for stem")
		}
		balanceStateDiff := stateDiff[stateDiffIdx].SuffixDiffs[0]
		if balanceStateDiff.Suffix != utils.BasicDataLeafKey {
			t.Fatal("balance invalid suffix")
		}
		if balanceStateDiff.CurrentValue != nil {
			t.Fatal("the pre-state balance before must be nil, since the contract didn't exist")
		}
		// Ensure that the value is burnt, and therefore that the balance of the self-destructed
		// contract isn't modified (it should remain missing from the state)
		if balanceStateDiff.NewValue != nil {
			t.Fatal("the post-state balance after self-destruct must be nil since the contract shouldn't be created at all")
		}
	}
}

// TestProcessVerkleSelfDestructInSameTxWithSelfBeneficiaryAndPrefundedAccount checks the
// content of the witness if a selfdestruct occurs in the same tx as the one that created it,
// it, but the beneficiary is the selfdestructed account. The difference with the test above,
// is that the created account is prefunded and so the final value should be 0.
func TestProcessVerkleSelfDestructInSameTxWithSelfBeneficiaryAndPrefundedAccount(t *testing.T) {
	// The test txs were taken from a secondary testnet with chain id 69421
	config := *testKaustinenLikeChainConfig
	config.ChainID.SetUint64(69421)

	var (
		signer     = types.LatestSigner(&config)
		testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		gspec      = verkleTestGenesis(&config)
		deployer   = crypto.PubkeyToAddress(testKey.PublicKey)
		contract   = crypto.CreateAddress(deployer, 0)
	)
	// Prefund the account, at an address that the contract will be deployed at,
	// before it selfdestrucs. We can therefore check that the account itseld is
	// NOT destroyed, which is what the current version of the spec requires.
	// TODO(gballet) revisit after the spec has been modified.
	gspec.Alloc[contract] = types.Account{
		Balance: big.NewInt(100),
	}

	selfDestructContract := []byte{byte(vm.ADDRESS), byte(vm.SELFDESTRUCT)}

	_, _, _, _, _, stateDiffs := GenerateVerkleChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 1, func(i int, gen *BlockGen) {
		gen.SetPoS()
		tx, _ := types.SignNewTx(testKey, signer, &types.LegacyTx{Nonce: 0,
			Value:    big.NewInt(42),
			Gas:      100_000,
			GasPrice: big.NewInt(875000000),
			Data:     selfDestructContract,
		})
		gen.AddTx(tx)
	})
	stateDiff := stateDiffs[0] // state difference of block 1

	{ // Check self-destructed contract in the witness
		selfDestructContractTreeKey := utils.CodeHashKey(contract[:])

		var stateDiffIdx = -1
		for i, stemStateDiff := range stateDiff {
			if bytes.Equal(stemStateDiff.Stem[:], selfDestructContractTreeKey[:31]) {
				stateDiffIdx = i
				break
			}
		}
		if stateDiffIdx == -1 {
			t.Fatal("no state diff found for stem")
		}
		balanceStateDiff := stateDiff[stateDiffIdx].SuffixDiffs[0]
		if balanceStateDiff.Suffix != utils.BasicDataLeafKey {
			t.Fatal("balance invalid suffix")
		}
		expected, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000064")
		if balanceStateDiff.CurrentValue == nil || !bytes.Equal(balanceStateDiff.CurrentValue[:], expected) {
			t.Fatalf("incorrect prestate balance: %x != %x", *balanceStateDiff.CurrentValue, expected)
		}
		// Ensure that the value is burnt, and therefore that the balance of the self-destructed
		// contract isn't modified (it should remain missing from the state)
		expected = make([]byte, 32)
		if balanceStateDiff.NewValue == nil {
			t.Fatal("incorrect nil poststate balance")
		}
		if !bytes.Equal(balanceStateDiff.NewValue[:], expected[:]) {
			t.Fatalf("incorrect poststate balance: %x != %x", *balanceStateDiff.NewValue, expected[:])
		}
	}
}
