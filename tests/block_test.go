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
	"testing"
)

func TestBlockchain(t *testing.T) {
	t.Parallel()

	bt := new(testMatcher)
	// General state tests are 'exported' as blockchain tests, but we can run them natively.
	bt.skipLoad(`^ValidBlocks/bcStateTests/`)
	// Skip random failures due to selfish mining test.
	bt.skipLoad(`.*bcForgedTest/bcForkUncle\.json`)
	bt.skipLoad(`.*bcMultiChainTest/(ChainAtoChainB_blockorder|CallContractFromNotBestBlock)`)
	bt.skipLoad(`.*bcTotalDifficultyTest/(lotsOfLeafs|lotsOfBranches|sideChainWithMoreTransactions)`)

	// These are not formatted like the rest -- due to the large postState, the postState
	// was replaced by a hash, instead of a genesisAlloc map
	// See https://github.com/ethereum/tests/pull/616
	bt.skipLoad(`.*bcExploitTest/ShanghaiLove.json`)
	bt.skipLoad(`.*bcExploitTest/SuicideIssue.json`)

	// Slow tests
	bt.slow(`.*bcExploitTest/DelegateCallSpam.json`)
	bt.slow(`.*bcExploitTest/ShanghaiLove.json`)
	bt.slow(`.*bcExploitTest/SuicideIssue.json`)
	bt.slow(`.*/bcForkStressTest/`)
	bt.slow(`.*/bcGasPricerTest/RPC_API_Test.json`)
	bt.slow(`.*/bcWalletTest/`)

	bt.walk(t, blockTestDir, func(t *testing.T, name string, test *BlockTest) {
		if err := bt.checkFailure(t, name, test.Run()); err != nil {
			t.Error(err)
		}
	})
}
