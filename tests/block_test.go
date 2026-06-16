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

	/*
		skip these for the payload building only:
		    --- FAIL: TestExecutionSpecBlocktests/osaka/eip7934_block_rlp_limit/test_block_at_rlp_limit_with_logs.json (0.45s)
		        block_test.go:110: test with config {snapshotter:false, scheme:hash} failed: mismatch in block hash. wanted 51a93fb2f038278424742e6ceaf476f83fd30ab84580216f08d124e258554b00, got 8dddbd82ec3e09b201230202207069fa64b6fb21a89a3b5e8ab2421adac933ea
		    --- FAIL: TestExecutionSpecBlocktests/osaka/eip7934_block_rlp_limit/test_block_at_rlp_size_limit_boundary.json (1.30s)
		        --- FAIL: TestExecutionSpecBlocktests/osaka/eip7934_block_rlp_limit/test_block_at_rlp_size_limit_boundary.json/tests/osaka/eip7934_block_rlp_limit/test_max_block_rlp_size.py::test_block_at_rlp_size_limit_boundary[fork_Osaka-blockchain_test-max_rlp_size] (0.10s)
		            block_test.go:110: test with config {snapshotter:false, scheme:hash} failed: mismatch in block hash. wanted bf2a96a93b4b519a36cc17ba7e5b9de7a2a8efef8f15f00e97d63c067fcae532, got 99be2c1ff6a5ab3b7c80eca8ed12e933417efc4c9fbc73a23873017a4888ac30
		        --- FAIL: TestExecutionSpecBlocktests/osaka/eip7934_block_rlp_limit/test_block_at_rlp_size_limit_boundary.json/tests/osaka/eip7934_block_rlp_limit/test_max_block_rlp_size.py::test_block_at_rlp_size_limit_boundary[fork_Osaka-blockchain_test-max_rlp_size_minus_1_byte] (0.09s)
		            block_test.go:110: test with config {snapshotter:false, scheme:hash} failed: mismatch in block hash. wanted f01227ceacf73b2fa7492cc36f307535d6734a0de468b42a136bd39561a613ec, got 5a5019a9d576c93e55ad94e2fe2b6ba8881d96c9df4b7d4ea8c4780520fa232e
		    --- FAIL: TestExecutionSpecBlocktests/osaka/eip7934_block_rlp_limit/test_block_rlp_size_at_limit_with_all_typed_transactions.json (2.51s)
		        --- FAIL: TestExecutionSpecBlocktests/osaka/eip7934_block_rlp_limit/test_block_rlp_size_at_limit_with_all_typed_transactions.json/tests/osaka/eip7934_block_rlp_limit/test_max_block_rlp_size.py::test_block_rlp_size_at_limit_with_all_typed_transactions[fork_Osaka-typed_transaction_0-blockchain_test] (0.09s)
		            block_test.go:110: test with config {snapshotter:false, scheme:hash} failed: mismatch in block hash. wanted e15a14788a8b960fa9951cdd8f059b9c53ac50ca12c2d63255a437fb2a10a93f, got 194fbad30f9fe3f0e60627a0fcc6eec78884ce117e9d13e04b7a44f8e14c0547
		        --- FAIL: TestExecutionSpecBlocktests/osaka/eip7934_block_rlp_limit/test_block_rlp_size_at_limit_with_all_typed_transactions.json/tests/osaka/eip7934_block_rlp_limit/test_max_block_rlp_size.py::test_block_rlp_size_at_limit_with_all_typed_transactions[fork_Osaka-typed_transaction_1-blockchain_test] (0.09s)
		            block_test.go:110: test with config {snapshotter:false, scheme:hash} failed: mismatch in block hash. wanted 5ad6568d4e64aa88faa2ad73a249bc7fcd9a36a23629ee6e914a05fa75a87293, got 0f9f047ddfc220c18b53cfc82871a67c4166eebda65bb0c137af87cf20e6c25a
		        --- FAIL: TestExecutionSpecBlocktests/osaka/eip7934_block_rlp_limit/test_block_rlp_size_at_limit_with_all_typed_transactions.json/tests/osaka/eip7934_block_rlp_limit/test_max_block_rlp_size.py::test_block_rlp_size_at_limit_with_all_typed_transactions[fork_Osaka-typed_transaction_2-blockchain_test] (0.09s)
		            block_test.go:110: test with config {snapshotter:false, scheme:hash} failed: mismatch in block hash. wanted 1570cd16ac3b096866bf5765fdb28571140f70392bda5ac7fbb034e5e9a93acb, got 0d254beb06fe3b4d4ffbe9cd6fa914f402a809e8f60ce311f849431e485ba8d9
		        --- FAIL: TestExecutionSpecBlocktests/osaka/eip7934_block_rlp_limit/test_block_rlp_size_at_limit_with_all_typed_transactions.json/tests/osaka/eip7934_block_rlp_limit/test_max_block_rlp_size.py::test_block_rlp_size_at_limit_with_all_typed_transactions[fork_Osaka-typed_transaction_4-blockchain_test] (0.11s)
		            block_test.go:110: test with config {snapshotter:false, scheme:hash} failed: mismatch in block hash. wanted 8a12b02aaf555c15e83ac9831205b35006c94e6c8aec7d2c99a135507fa4103b, got d65726606a9c28a68cea095b2682872d88cb535ff295b135d156f28d3c2c59f7
	*/

	// With chain history removal, TDs become unavailable, this transition tests based on TTD are unrunnable
	bt.skipLoad(`.*bcArrowGlacierToParis/powToPosBlockRejection.json`)

	// This directory contains no test.
	bt.skipLoad(`.*\.meta/.*`)

	// Broken tests
	bt.skipLoad(`RevertInCreateInInit`)
	bt.skipLoad(`InitCollisionParis`)
	bt.skipLoad(`dynamicAccountOverwriteEmpty_Paris`)
	bt.skipLoad(`create2collisionStorageParis`)

	bt.walk(t, blockTestDir, func(t *testing.T, name string, test *BlockTest) {
		execBlockTest(t, bt, test)
	})
	// There is also a LegacyTests folder, containing blockchain tests generated
	// prior to Istanbul. However, they are all derived from GeneralStateTests,
	// which run natively, so there's no reason to run them here.
}

// TestExecutionSpecBlocktests runs the test fixtures from execution-spec-tests.
func TestExecutionSpecBlocktests(t *testing.T) {
	if !common.FileExist(executionSpecBlockchainTestDir) {
		t.Skipf("directory %s does not exist", executionSpecBlockchainTestDir)
	}
	bt := new(testMatcher)

	// These tests require us to handle scenarios where a system contract is not deployed at a fork
	bt.skipLoad(".*prague/eip7251_consolidations/test_system_contract_deployment.json")
	bt.skipLoad(".*prague/eip7002_el_triggerable_withdrawals/test_system_contract_deployment.json")

	// Broken tests
	bt.skipLoad(`RevertInCreateInInit`)
	bt.skipLoad(`InitCollisionParis`)
	bt.skipLoad(`dynamicAccountOverwriteEmpty_Paris`)
	bt.skipLoad(`create2collisionStorageParis`)

	bt.walk(t, executionSpecBlockchainTestDir, func(t *testing.T, name string, test *BlockTest) {
		execBlockTest(t, bt, test)
	})
}

func execBlockTest(t *testing.T, bt *testMatcher, test *BlockTest) {
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
			if err := bt.checkFailure(t, test.Run(snapshot, dbscheme, true, nil, nil)); err != nil {
				t.Errorf("test with config {snapshotter:%v, scheme:%v} failed: %v", snapshot, dbscheme, err)
				return
			}
		}
	}
}
