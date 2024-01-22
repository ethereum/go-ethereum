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
	"os"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/exp/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestBlockchain(t *testing.T) {
	bt := new(testMatcher)
	// General state tests are 'exported' as blockchain tests, but we can run them natively.
	// For speedier CI-runs, the line below can be uncommented, so those are skipped.
	// For now, in hardfork-times (Berlin), we run the tests both as StateTests and
	// as blockchain tests, since the latter also covers things like receipt root
	bt.skipLoad(`^GeneralStateTests/`)

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

	bt.walk(t, blockTestDir, func(t *testing.T, name string, test *BlockTest) {
		if runtime.GOARCH == "386" && runtime.GOOS == "windows" && rand.Int63()%2 == 0 {
			t.Skip("test (randomly) skipped on 32-bit windows")
		}
		execBlockTest(t, bt, test)
	})
	// There is also a LegacyTests folder, containing blockchain tests generated
	// prior to Istanbul. However, they are all derived from GeneralStateTests,
	// which run natively, so there's no reason to run them here.
}

func networkPostMerge(network string) bool {
	switch network {
	case "Frontier":
		return false
	case "EIP150":
		return false
	case "EIP158":
		return false
	case "Byzantium":
		return false
	case "Constantinople":
		return false
	case "ConstantinopleFix":
		return false
	case "Istanbul":
		return false
	case "MuirGlacier":
		return false
	case "Berlin":
		return false
	case "London":
		return false
	case "ArrowGlacier":
		return false
	case "GreyGlacier":
		return false
	case "Merge":
		return true
	case "Shanghai":
		return true
	case "Cancun":
		return true
	}
	return false
}

func TestStatelessBlockchain(t *testing.T) {
	bt := new(testMatcher)

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

	// skip uncle tests for stateless
	bt.skipLoad(`.*/UnclePopulation.json`)
	// skip this test in stateless because it uses 5000 blocks and the
	// historical state of older blocks is unavailable for stateless
	// test verification after importing the test set.
	bt.skipLoad(`.*/bcWalletTest/walletReorganizeOwners.json`)

	bt.walk(t, blockTestDir, func(t *testing.T, name string, test *BlockTest) {
		if runtime.GOARCH == "386" && runtime.GOOS == "windows" && rand.Int63()%2 == 0 {
			t.Skip("test (randomly) skipped on 32-bit windows")
		}

		config, ok := Forks[test.json.Network]
		if !ok {
			t.Fatalf("test malformed: doesn't have chain config embedded")
		}
		isMerged := config.TerminalTotalDifficulty != nil && config.TerminalTotalDifficulty.BitLen() == 0
		if isMerged {
			execBlockTestStateless(t, bt, test)
		} else {
			t.Skip("skipping pre-merge test")
		}
	})
	// There is also a LegacyTests folder, containing blockchain tests generated
	// prior to Istanbul. However, they are all derived from GeneralStateTests,
	// which run natively, so there's no reason to run them here.
}

// TestExecutionSpec runs the test fixtures from execution-spec-tests.
func TestExecutionSpec(t *testing.T) {
	if !common.FileExist(executionSpecDir) {
		t.Skipf("directory %s does not exist", executionSpecDir)
	}
	bt := new(testMatcher)

	bt.walk(t, executionSpecDir, func(t *testing.T, name string, test *BlockTest) {
		execBlockTest(t, bt, test)
	})
}

func execBlockTest(t *testing.T, bt *testMatcher, test *BlockTest) {
	if err := bt.checkFailure(t, test.Run(false, rawdb.HashScheme, nil)); err != nil {
		t.Errorf("test in hash mode without snapshotter failed: %v", err)
		return
	}
	if err := bt.checkFailure(t, test.Run(true, rawdb.HashScheme, nil)); err != nil {
		t.Errorf("test in hash mode with snapshotter failed: %v", err)
		return
	}
	if err := bt.checkFailure(t, test.Run(false, rawdb.PathScheme, nil)); err != nil {
		t.Errorf("test in path mode without snapshotter failed: %v", err)
		return
	}
	if err := bt.checkFailure(t, test.Run(true, rawdb.PathScheme, nil)); err != nil {
		t.Errorf("test in path mode with snapshotter failed: %v", err)
		return
	}
}

func execBlockTestStateless(t *testing.T, bt *testMatcher, test *BlockTest) {
	handler := log.NewTerminalHandlerWithLevel(os.Stdout, slog.Level(667), false)
	log.SetDefault(log.NewLogger(handler))
	logconfig := &logger.Config{
		EnableMemory:     false,
		DisableStack:     false,
		DisableStorage:   false,
		EnableReturnData: true,
		Debug:            true,
	}
	tracer := logger.NewJSONLogger(logconfig, os.Stdout)
	_ = tracer

	if err := bt.checkFailure(t, test.RunStateless(false, rawdb.HashScheme, nil)); err != nil {
		t.Errorf("test in hash mode without snapshotter failed: %v", err)
		return
	}

	if err := bt.checkFailure(t, test.RunStateless(true, rawdb.HashScheme, nil)); err != nil {
		t.Errorf("test in hash mode with snapshotter failed: %v", err)
		return
	}

	if err := bt.checkFailure(t, test.RunStateless(false, rawdb.PathScheme, nil)); err != nil {
		t.Errorf("test in path mode without snapshotter failed: %v", err)
		return
	}
	if err := bt.checkFailure(t, test.RunStateless(true, rawdb.PathScheme, nil)); err != nil {
		t.Errorf("test in path mode with snapshotter failed: %v", err)
		return
	}
}
