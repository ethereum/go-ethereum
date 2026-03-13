// Copyright 2026 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/params"
)

func TestBinaryTransitionRegistryBootstrap(t *testing.T) {
	var (
		verkleTime uint64 = 30
		coinbase          = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
		gspec             = &Genesis{
			Config: &params.ChainConfig{
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
				CancunTime:              u64(0),
				PragueTime:              u64(0),
				VerkleTime:              &verkleTime,
				TerminalTotalDifficulty: common.Big0,
				EnableVerkleAtGenesis:   false,
				BlobScheduleConfig: &params.BlobScheduleConfig{
					Cancun: params.DefaultCancunBlobConfig,
					Prague: params.DefaultPragueBlobConfig,
					Verkle: params.DefaultPragueBlobConfig,
				},
			},
			Alloc: GenesisAlloc{
				coinbase: {
					Balance: big.NewInt(1000000000000000000),
				},
				params.BeaconRootsAddress:        {Nonce: 1, Code: params.BeaconRootsCode, Balance: common.Big0},
				params.HistoryStorageAddress:     {Nonce: 1, Code: params.HistoryStorageCode, Balance: common.Big0},
				params.WithdrawalQueueAddress:    {Nonce: 1, Code: params.WithdrawalQueueCode, Balance: common.Big0},
				params.ConsolidationQueueAddress: {Nonce: 1, Code: params.ConsolidationQueueCode, Balance: common.Big0},
			},
		}
	)
	config := gspec.Config
	engine := beacon.New(ethash.NewFaker())

	registryAddr := params.BinaryTransitionRegistryAddress
	slotStarted := common.Hash{}
	slotBaseRoot := common.BytesToHash([]byte{5})

	GenerateChainWithGenesis(gspec, engine, 6, func(i int, gen *BlockGen) {
		gen.SetPoS()

		blockNum := gen.Number()
		blockTime := gen.Timestamp()
		isVerkle := config.IsVerkle(blockNum, blockTime)

		t.Logf("generating block %d: num=%d time=%d isVerkle=%v", i, blockNum.Uint64(), blockTime, isVerkle)

		if !isVerkle {
			started := gen.GetState(registryAddr, slotStarted)
			if started != (common.Hash{}) {
				t.Errorf("block %d: pre-transition block should not have registry initialized", i)
			}
			return
		}

		started := gen.GetState(registryAddr, slotStarted)
		baseRoot := gen.GetState(registryAddr, slotBaseRoot)
		t.Logf("block %d: started=%x baseRoot=%x", i, started, baseRoot)

		if i == 2 {
			if started != (common.Hash{}) {
				t.Errorf("block %d: first verkle block gen callback runs before Phase 1, registry should not be visible yet", i)
			}
		} else {
			if started == (common.Hash{}) {
				t.Errorf("block %d: verkle block should have registry slot 0 (started) set", i)
			}
		}

		if i >= 3 {
			if baseRoot == (common.Hash{}) {
				t.Errorf("block %d: should have base root set by Phase 2", i)
			}
		}
	})
}
