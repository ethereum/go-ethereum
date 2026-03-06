// Copyright 2026 The XDPoSChain Authors
// This file is part of the XDPoSChain library.
//
// The XDPoSChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The XDPoSChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the XDPoSChain library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"math/big"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/params"
)

func newBlockingSubscription() event.Subscription {
	return event.NewSubscription(func(unsub <-chan struct{}) error {
		<-unsub
		return nil
	})
}

func TestWorkerUpdateNonXDPoSStaysRunning(t *testing.T) {
	worker := &worker{
		engine:       ethash.NewFaker(),
		chainHeadSub: newBlockingSubscription(),
		chainSideSub: newBlockingSubscription(),
		resetCh:      make(chan time.Duration, 1),
	}

	done := make(chan struct{})
	started := make(chan struct{})
	go func() {
		close(started)
		worker.update()
		close(done)
	}()
	select {
	case <-started:
		// worker.update has started; proceed with timing checks.
	case <-time.After(time.Second):
		t.Fatal("worker.update did not start in time")
	}

	select {
	case <-done:
		t.Fatal("worker.update returned before unsubscribe")
	default:
		// Expected: update is still running until subscription error.
	}
	worker.chainHeadSub.Unsubscribe()

	select {
	case <-done:
		// Expected: update exits after subscription error.
	case <-time.After(time.Second):
		t.Fatal("worker.update did not return after unsubscribe")
	}
}

func TestWorkerCheckPreCommitXDPoSMismatch(t *testing.T) {
	config := &params.ChainConfig{
		ChainID: big.NewInt(1),
		XDPoS: &params.XDPoSConfig{
			V2: &params.V2{
				SwitchBlock: big.NewInt(0),
				AllConfigs: map[uint64]*params.V2Config{
					0: {MinePeriod: 2},
				},
			},
		},
	}
	signer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	extraData := make([]byte, 0, utils.ExtraVanity+common.AddressLength+utils.ExtraSeal)
	extraData = append(extraData, make([]byte, utils.ExtraVanity)...)
	extraData = append(extraData, signer.Bytes()...)
	extraData = append(extraData, make([]byte, utils.ExtraSeal)...)
	genesis := &core.Genesis{
		Config:     config,
		GasLimit:   params.TargetGasLimit,
		Difficulty: big.NewInt(1),
		Alloc:      types.GenesisAlloc{},
		ExtraData:  extraData,
	}
	db := rawdb.NewMemoryDatabase()
	if _, err := genesis.Commit(db); err != nil {
		t.Fatalf("failed to commit genesis: %v", err)
	}
	engine := ethash.NewFaker()
	chain, err := core.NewBlockChain(db, nil, genesis, engine, vm.Config{})
	if err != nil {
		t.Fatalf("failed to create blockchain: %v", err)
	}
	defer chain.Stop()

	worker := &worker{
		config:      config,
		engine:      engine,
		chain:       chain,
		announceTxs: true,
	}

	parent, shouldReturn := worker.checkPreCommitWithLock()
	if parent == nil {
		t.Fatal("expected parent block, got nil")
	}
	if !shouldReturn {
		t.Fatal("expected checkPreCommitWithLock to skip when XDPoS config is enabled but engine is not XDPoS")
	}
	if parent.Number().Sign() != 0 {
		t.Fatalf("expected genesis parent, got number %v", parent.Number())
	}
}
