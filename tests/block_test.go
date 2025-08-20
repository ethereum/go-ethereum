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

package tests

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestBlockchain(t *testing.T) {
	bt := new(testMatcher)

	// We are running most of GeneralStatetests to tests witness support, even
	// though they are ran as state tests too. Still, the performance tests are
	// less about state andmore about EVM number crunching, so skip those.
	bt.skipLoad(`^GeneralStateTests/VMTests/vmPerformance`)

	// Skip random failures due to selfish mining test
	bt.skipLoad(`.*bcForgedTest/bcForkUncle\.json`)

	// Slow tests
	bt.slow(`.*bcExploitTest/DelegateCallSpam.json`)
	bt.slow(`.*bcExploitTest/ShanghaiLove.json`)
	bt.slow(`.*bcExploitTest/SuicideIssue.json`)
	bt.slow(`.*/bcForkStressTest/`)
	bt.slow(`.*/bcGasPricerTest/RPC_API_Test.json`)
	bt.slow(`.*/bcWalletTest/`)

	// Very slow test
	bt.skipLoad(`.*/stTimeConsuming/.*`)
	// test takes a lot for time and goes easily OOM because of sha3 calculation on a huge range,
	// using 4.6 TGas
	bt.skipLoad(`.*randomStatetest94.json.*`)

	// After the merge we would accept side chains as canonical even if they have lower td
	bt.skipLoad(`.*bcMultiChainTest/ChainAtoChainB_difficultyB.json`)
	bt.skipLoad(`.*bcMultiChainTest/CallContractFromNotBestBlock.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/uncleBlockAtBlock3afterBlock4.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/lotsOfBranchesOverrideAtTheMiddle.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/sideChainWithMoreTransactions.json`)
	bt.skipLoad(`.*bcForkStressTest/ForkStressTest.json`)
	bt.skipLoad(`.*bcMultiChainTest/lotsOfLeafs.json`)
	bt.skipLoad(`.*bcFrontierToHomestead/blockChainFrontierWithLargerTDvsHomesteadBlockchain.json`)
	bt.skipLoad(`.*bcFrontierToHomestead/blockChainFrontierWithLargerTDvsHomesteadBlockchain2.json`)

	// With chain history removal, TDs become unavailable, this transition tests based on TTD are unrunnable
	bt.skipLoad(`.*bcArrowGlacierToParis/powToPosBlockRejection.json`)

	// This directory contains no test.
	bt.skipLoad(`.*\.meta/.*`)

	bt.walk(t, blockTestDir, func(t *testing.T, name string, test *BlockTest) {
		execBlockTest(t, bt, test, false)
	})
	// There is also a LegacyTests folder, containing blockchain tests generated
	// prior to Istanbul. However, they are all derived from GeneralStateTests,
	// which run natively, so there's no reason to run them here.
}

func TestBlockchainBAL(t *testing.T) {
	bt := new(testMatcher)

	// We are running most of GeneralStatetests to tests witness support, even
	// though they are ran as state tests too. Still, the performance tests are
	// less about state andmore about EVM number crunching, so skip those.
	bt.skipLoad(`^GeneralStateTests/VMTests/vmPerformance`)

	// Skip random failures due to selfish mining test
	bt.skipLoad(`.*bcForgedTest/bcForkUncle\.json`)

	// Slow tests
	bt.slow(`.*bcExploitTest/DelegateCallSpam.json`)
	bt.slow(`.*bcExploitTest/ShanghaiLove.json`)
	bt.slow(`.*bcExploitTest/SuicideIssue.json`)
	bt.slow(`.*/bcForkStressTest/`)
	bt.slow(`.*/bcGasPricerTest/RPC_API_Test.json`)
	bt.slow(`.*/bcWalletTest/`)

	// Very slow test
	bt.skipLoad(`.*/stTimeConsuming/.*`)
	// test takes a lot for time and goes easily OOM because of sha3 calculation on a huge range,
	// using 4.6 TGas
	bt.skipLoad(`.*randomStatetest94.json.*`)

	// After the merge we would accept side chains as canonical even if they have lower td
	bt.skipLoad(`.*bcMultiChainTest/ChainAtoChainB_difficultyB.json`)
	bt.skipLoad(`.*bcMultiChainTest/CallContractFromNotBestBlock.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/uncleBlockAtBlock3afterBlock4.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/lotsOfBranchesOverrideAtTheMiddle.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/sideChainWithMoreTransactions.json`)
	bt.skipLoad(`.*bcForkStressTest/ForkStressTest.json`)
	bt.skipLoad(`.*bcMultiChainTest/lotsOfLeafs.json`)
	bt.skipLoad(`.*bcFrontierToHomestead/blockChainFrontierWithLargerTDvsHomesteadBlockchain.json`)
	bt.skipLoad(`.*bcFrontierToHomestead/blockChainFrontierWithLargerTDvsHomesteadBlockchain2.json`)

	// With chain history removal, TDs become unavailable, this transition tests based on TTD are unrunnable
	bt.skipLoad(`.*bcArrowGlacierToParis/powToPosBlockRejection.json`)

	// This directory contains no test.
	bt.skipLoad(`.*\.meta/.*`)

	// skip tests which use large balances (greater than 16 bytes)
	bt.skipLoad(`.*/stStaticCall/static_RETURN_BoundsOOG.json`)
	bt.skipLoad(`.*/stStaticCall/static_RETURN_Bounds.json`)
	bt.skipLoad(`.*/stStaticCall/static_RETURN_Bounds.json`)
	bt.skipLoad(`.*/stStaticCall/static_Call1024PreCalls.json`)
	bt.skipLoad(`.*/stStaticCall/static_Call1024PreCalls2.json`)
	bt.skipLoad(`.*/stStaticCall/static_Call1024PreCalls3.json`)
	bt.skipLoad(`.*/stStaticCall/static_CheckOpcodes5.json`)
	bt.skipLoad(`.*/stStaticCall/static_CallContractToCreateContractWhichWouldCreateContractIfCalled.json`)
	bt.skipLoad(`.*/stStaticCall/static_log0_logMemStartTooHigh.json`)
	bt.skipLoad(`.*/stStaticCall/static_Call50000_ecrec.json`)

	bt.skipLoad(`.*/stTransactionTest/HighGasLimit.json`)
	bt.skipLoad(`.*/stTransactionTest/OverflowGasRequire2.json`)

	bt.skipLoad(`.*/stMemoryStressTest/static_CALL_Bounds2.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CREATE_Bounds.json`)
	bt.skipLoad(`.*/stMemoryStressTest/DELEGATECALL_Bounds.json`)
	bt.skipLoad(`.*/stMemoryStressTest/MSTORE_Bounds2a.json`)
	bt.skipLoad(`.*/stMemoryStressTest/DELEGATECALL_Bounds3.json`)
	bt.skipLoad(`.*/stMemoryStressTest/static_CALL_Bounds3.json`)
	bt.skipLoad(`.*/stMemoryStressTest/MSTORE_Bounds2.json`)
	bt.skipLoad(`.*/stMemoryStressTest/static_CALL_Bounds2a.json`)
	bt.skipLoad(`.*/stMemoryStressTest/DELEGATECALL_Bounds2.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CALLCODE_Bounds.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CREATE_Bounds2.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CALLCODE_Bounds2.json`)
	bt.skipLoad(`.*/stMemoryStressTest/static_CALL_Bounds.json`)
	bt.skipLoad(`.*/stMemoryStressTest/mload32bitBound2.json`)
	bt.skipLoad(`.*/stMemoryStressTest/MSTORE_Bounds.json`)
	bt.skipLoad(`.*/stMemoryStressTest/mload32bitBound_Msize.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CALLCODE_Bounds3.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CALL_Bounds2a.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CALLCODE_Bounds4.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CALL_Bounds.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CALL_Bounds2.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CALL_Bounds3.json`)
	bt.skipLoad(`.*/stMemoryStressTest/CREATE_Bounds3.json`)
	bt.skipLoad(`.*/stMemoryStressTest/RETURN_Bounds.json`)
	bt.skipLoad(`.*/stRevertTest/RevertPrefoundEmpty_Paris.json`)
	bt.skipLoad(`.*/stInitCodeTest/OutOfGasContractCreation.json`)
	bt.skipLoad(`.*/stDelegatecallTestHomestead/Call1024PreCalls.json`)
	bt.skipLoad(`.*/stCreate2/Create2OnDepth1024.json`)
	bt.skipLoad(`.*/stCreate2/Create2OnDepth1023.json`)
	bt.skipLoad(`.*/stCreate2/CREATE2_Bounds.json`)
	bt.skipLoad(`.*/stCreate2/CREATE2_Bounds2.json`)
	bt.skipLoad(`.*/stCreate2/CREATE2_Bounds3.json`)
	bt.skipLoad(`.*/stCreate2/Create2Recursive.json`)
	bt.skipLoad(`.*/stCallCreateCallCodeTest/Call1024PreCalls.json`)
	bt.skipLoad(`.*/bcExploitTest/StrangeContractCreation.json`)
	bt.skipLoad(`.*/bcStateTests/OverflowGasRequire.json`)
	bt.skipLoad(`.*/bcExploitTest/StrangeContractCreation.json`)
	bt.skipLoad(`.*/bcStateTests/OverflowGasRequire.json`)
	bt.skipLoad(`.*/bcExploitTest/DelegateCallSpam.json`)
	bt.skipLoad(`.*/bcStateTests/OverflowGasRequire.json`)
	bt.skipLoad(`.*/bcExploitTest/ShanghaiLove.json`)
	bt.skipLoad(`.*/bcExploitTest/SuicideIssue.json`)

	bt.walk(t, blockTestDir, func(t *testing.T, name string, test *BlockTest) {
		config, ok := Forks[test.json.Network]
		if !ok {
			t.Fatalf("unsupported fork: %s\n", test.json.Network)
		}
		gspec := test.genesis(config)
		// skip any tests which are not past the cancun fork (selfdestruct removal)
		if gspec.Config.CancunTime == nil || *gspec.Config.CancunTime != 0 {
			return
		}
		execBlockTest(t, bt, test, true)
	})
	// There is also a LegacyTests folder, containing blockchain tests generated
	// prior to Istanbul. However, they are all derived from GeneralStateTests,
	// which run natively, so there's no reason to run them here.
}

// TestExecutionSpecBlocktests runs the test fixtures from execution-spec-tests.
func TestExecutionSpecBlocktestsBAL(t *testing.T) {
	if !common.FileExist(executionSpecBlockchainTestDir) {
		t.Skipf("directory %s does not exist", executionSpecBlockchainTestDir)
	}
	bt := new(testMatcher)

	bt.skipLoad(".*prague/eip7251_consolidations/contract_deployment/system_contract_deployment.json")
	bt.skipLoad(".*prague/eip7002_el_triggerable_withdrawals/contract_deployment/system_contract_deployment.json")

	bt.walk(t, executionSpecBlockchainTestDir, func(t *testing.T, name string, test *BlockTest) {
		config, ok := Forks[test.json.Network]
		if !ok {
			t.Fatalf("unsupported fork: %s\n", test.json.Network)
		}
		gspec := test.genesis(config)
		// skip any tests which are not past the cancun fork (selfdestruct removal)
		if gspec.Config.CancunTime == nil || *gspec.Config.CancunTime != 0 {
			return
		}
		execBlockTest(t, bt, test, true)
	})
}

// TestExecutionSpecBlocktests runs the test fixtures from execution-spec-tests.
func TestExecutionSpecBlocktests(t *testing.T) {
	if !common.FileExist(executionSpecBlockchainTestDir) {
		t.Skipf("directory %s does not exist", executionSpecBlockchainTestDir)
	}
	bt := new(testMatcher)

	bt.skipLoad(".*prague/eip7251_consolidations/contract_deployment/system_contract_deployment.json")
	bt.skipLoad(".*prague/eip7002_el_triggerable_withdrawals/contract_deployment/system_contract_deployment.json")

	bt.walk(t, executionSpecBlockchainTestDir, func(t *testing.T, name string, test *BlockTest) {
		execBlockTest(t, bt, test, false)
	})
}

func execBlockTest(t *testing.T, bt *testMatcher, test *BlockTest, testBAL bool) {
	// Define all the different flag combinations we should run the tests with,
	// picking only one for short tests.
	//
	// Note, witness building and self-testing is always enabled as it's a very
	// good test to ensure that we don't break it.
	var (
		snapshotConf = []bool{false, true}
		dbschemeConf = []string{rawdb.HashScheme, rawdb.PathScheme}
	)
	if testing.Short() {
		snapshotConf = []bool{snapshotConf[rand.Int()%2]}
		dbschemeConf = []string{dbschemeConf[rand.Int()%2]}
	}

	for _, snapshot := range snapshotConf {
		for _, dbscheme := range dbschemeConf {
			if err := bt.checkFailure(t, test.Run(snapshot, dbscheme, false, testBAL, nil, nil)); err != nil {
				t.Errorf("test with config {snapshotter:%v, scheme:%v} failed: %v", snapshot, dbscheme, err)
				return
			}
		}
	}
}
