// Copyright 2022 The go-ethereum Authors
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

package light

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

var (
	testGenesis  = newTestGenesis()
	testGenesis2 = newTestGenesis()

	tfBase = newTestForks(testGenesis, types.Forks{
		&types.Fork{Epoch: 0, Version: []byte{0}},
	})
	tfAlternative = newTestForks(testGenesis, types.Forks{
		&types.Fork{Epoch: 0, Version: []byte{0}},
		&types.Fork{Epoch: 0x700, Version: []byte{1}},
	})
	tfAnotherGenesis = newTestForks(testGenesis2, types.Forks{
		&types.Fork{Epoch: 0, Version: []byte{0}},
	})

	tcBase                      = newTestCommitteeChain(nil, tfBase, true, 0, 10, 400, false)
	tcBaseWithInvalidUpdates    = newTestCommitteeChain(tcBase, tfBase, false, 5, 10, 200, false) // signer count too low
	tcBaseWithBetterUpdates     = newTestCommitteeChain(tcBase, tfBase, false, 5, 10, 440, false)
	tcReorgWithWorseUpdates     = newTestCommitteeChain(tcBase, tfBase, true, 5, 10, 400, false)
	tcReorgWithWorseUpdates2    = newTestCommitteeChain(tcBase, tfBase, true, 5, 10, 380, false)
	tcReorgWithBetterUpdates    = newTestCommitteeChain(tcBase, tfBase, true, 5, 10, 420, false)
	tcReorgWithFinalizedUpdates = newTestCommitteeChain(tcBase, tfBase, true, 5, 10, 400, true)
	tcFork                      = newTestCommitteeChain(tcBase, tfAlternative, true, 7, 10, 400, false)
	tcAnotherGenesis            = newTestCommitteeChain(nil, tfAnotherGenesis, true, 0, 10, 400, false)
)

func TestCommitteeChainFixedCommitteeRoots(t *testing.T) {
	for _, reload := range []bool{false, true} {
		c := newCommitteeChainTest(t, tfBase, 300, true)
		c.setClockPeriod(7)
		c.addFixedCommitteeRoot(tcBase, 4, nil)
		c.addFixedCommitteeRoot(tcBase, 5, nil)
		c.addFixedCommitteeRoot(tcBase, 6, nil)
		c.addFixedCommitteeRoot(tcBase, 8, ErrInvalidPeriod) // range has to be continuous
		c.addFixedCommitteeRoot(tcBase, 3, nil)
		c.addFixedCommitteeRoot(tcBase, 2, nil)
		if reload {
			c.reloadChain()
		}
		c.addCommittee(tcBase, 4, nil)
		c.addCommittee(tcBase, 6, ErrInvalidPeriod) // range has to be continuous
		c.addCommittee(tcBase, 5, nil)
		c.addCommittee(tcBase, 6, nil)
		c.addCommittee(tcAnotherGenesis, 3, ErrWrongCommitteeRoot)
		c.addCommittee(tcBase, 3, nil)
		if reload {
			c.reloadChain()
		}
		c.verifyRange(tcBase, 3, 6)
	}
}

func TestCommitteeChainCheckpointSync(t *testing.T) {
	for _, enforceTime := range []bool{false, true} {
		for _, reload := range []bool{false, true} {
			c := newCommitteeChainTest(t, tfBase, 300, enforceTime)
			if enforceTime {
				c.setClockPeriod(6)
			}
			c.insertUpdate(tcBase, 3, true, ErrInvalidPeriod)
			c.addFixedCommitteeRoot(tcBase, 3, nil)
			c.addFixedCommitteeRoot(tcBase, 4, nil)
			c.insertUpdate(tcBase, 4, true, ErrInvalidPeriod) // still no committee
			c.addCommittee(tcBase, 3, nil)
			c.addCommittee(tcBase, 4, nil)
			if reload {
				c.reloadChain()
			}
			c.verifyRange(tcBase, 3, 4)
			c.insertUpdate(tcBase, 3, false, nil)              // update can be added without committee here
			c.insertUpdate(tcBase, 4, false, ErrNeedCommittee) // but not here as committee 5 is not there yet
			c.insertUpdate(tcBase, 4, true, nil)
			c.verifyRange(tcBase, 3, 5)
			c.insertUpdate(tcBaseWithInvalidUpdates, 5, true, ErrInvalidUpdate) // signer count too low
			c.insertUpdate(tcBase, 5, true, nil)
			if reload {
				c.reloadChain()
			}
			if enforceTime {
				c.insertUpdate(tcBase, 6, true, ErrInvalidUpdate) // future update rejected
				c.setClockPeriod(7)
			}
			c.insertUpdate(tcBase, 6, true, nil) // when the time comes it's accepted
			if reload {
				c.reloadChain()
			}
			if enforceTime {
				c.verifyRange(tcBase, 3, 6) // committee 7 is there but still in the future
				c.setClockPeriod(8)
			}
			c.verifyRange(tcBase, 3, 7) // now period 7 can also be verified
			// try reverse syncing an update
			c.insertUpdate(tcBase, 2, false, ErrInvalidPeriod) // fixed committee is needed first
			c.addFixedCommitteeRoot(tcBase, 2, nil)
			c.addCommittee(tcBase, 2, nil)
			c.insertUpdate(tcBase, 2, false, nil)
			c.verifyRange(tcBase, 2, 7)
		}
	}
}

func TestCommitteeChainReorg(t *testing.T) {
	for _, reload := range []bool{false, true} {
		for _, addBetterUpdates := range []bool{false, true} {
			c := newCommitteeChainTest(t, tfBase, 300, true)
			c.setClockPeriod(11)
			c.addFixedCommitteeRoot(tcBase, 3, nil)
			c.addFixedCommitteeRoot(tcBase, 4, nil)
			c.addCommittee(tcBase, 3, nil)
			for period := uint64(3); period < 10; period++ {
				c.insertUpdate(tcBase, period, true, nil)
			}
			if reload {
				c.reloadChain()
			}
			c.verifyRange(tcBase, 3, 10)
			c.insertUpdate(tcReorgWithWorseUpdates, 5, true, ErrCannotReorg)
			c.insertUpdate(tcReorgWithWorseUpdates2, 5, true, ErrCannotReorg)
			if addBetterUpdates {
				// add better updates for the base chain and expect first reorg to fail
				// (only add updates as committees should be the same)
				for period := uint64(5); period < 10; period++ {
					c.insertUpdate(tcBaseWithBetterUpdates, period, false, nil)
				}
				if reload {
					c.reloadChain()
				}
				c.verifyRange(tcBase, 3, 10) // still on the same chain
				c.insertUpdate(tcReorgWithBetterUpdates, 5, true, ErrCannotReorg)
			} else {
				// reorg with better updates
				c.insertUpdate(tcReorgWithBetterUpdates, 5, false, ErrNeedCommittee)
				c.verifyRange(tcBase, 3, 10) // no success yet, still on the base chain
				c.verifyRange(tcReorgWithBetterUpdates, 3, 5)
				c.insertUpdate(tcReorgWithBetterUpdates, 5, true, nil)
				// successful reorg, base chain should only match before the reorg period
				if reload {
					c.reloadChain()
				}
				c.verifyRange(tcBase, 3, 5)
				c.verifyRange(tcReorgWithBetterUpdates, 3, 6)
				for period := uint64(6); period < 10; period++ {
					c.insertUpdate(tcReorgWithBetterUpdates, period, true, nil)
				}
				c.verifyRange(tcReorgWithBetterUpdates, 3, 10)
			}
			// reorg with finalized updates; should succeed even if base chain updates
			// have been improved because a finalized update beats everything else
			c.insertUpdate(tcReorgWithFinalizedUpdates, 5, false, ErrNeedCommittee)
			c.insertUpdate(tcReorgWithFinalizedUpdates, 5, true, nil)
			if reload {
				c.reloadChain()
			}
			c.verifyRange(tcReorgWithFinalizedUpdates, 3, 6)
			for period := uint64(6); period < 10; period++ {
				c.insertUpdate(tcReorgWithFinalizedUpdates, period, true, nil)
			}
			c.verifyRange(tcReorgWithFinalizedUpdates, 3, 10)
		}
	}
}

func TestCommitteeChainFork(t *testing.T) {
	c := newCommitteeChainTest(t, tfAlternative, 300, true)
	c.setClockPeriod(11)
	// trying to sync a chain on an alternative fork with the base chain data
	c.addFixedCommitteeRoot(tcBase, 0, nil)
	c.addFixedCommitteeRoot(tcBase, 1, nil)
	c.addCommittee(tcBase, 0, nil)
	// shared section should sync without errors
	for period := uint64(0); period < 7; period++ {
		c.insertUpdate(tcBase, period, true, nil)
	}
	c.insertUpdate(tcBase, 7, true, ErrInvalidUpdate) // wrong fork
	// committee root #7 is still the same but signatures are already signed with
	// a different fork id so period 7 should only verify on the alternative fork
	c.verifyRange(tcBase, 0, 6)
	c.verifyRange(tcFork, 0, 7)
	for period := uint64(7); period < 10; period++ {
		c.insertUpdate(tcFork, period, true, nil)
	}
	c.verifyRange(tcFork, 0, 10)
	// reload the chain while switching to the base fork
	c.config = tfBase
	c.reloadChain()
	// updates 7..9 should be rolled back now
	c.verifyRange(tcFork, 0, 6) // again, period 7 only verifies on the right fork
	c.verifyRange(tcBase, 0, 7)
	c.insertUpdate(tcFork, 7, true, ErrInvalidUpdate) // wrong fork
	for period := uint64(7); period < 10; period++ {
		c.insertUpdate(tcBase, period, true, nil)
	}
	c.verifyRange(tcBase, 0, 10)
}

type committeeChainTest struct {
	t               *testing.T
	db              *memorydb.Database
	clock           *mclock.Simulated
	config          types.ChainConfig
	signerThreshold int
	enforceTime     bool
	chain           *CommitteeChain
}

func newCommitteeChainTest(t *testing.T, config types.ChainConfig, signerThreshold int, enforceTime bool) *committeeChainTest {
	c := &committeeChainTest{
		t:               t,
		db:              memorydb.New(),
		clock:           &mclock.Simulated{},
		config:          config,
		signerThreshold: signerThreshold,
		enforceTime:     enforceTime,
	}
	c.chain = NewTestCommitteeChain(c.db, &config, signerThreshold, enforceTime, c.clock)
	return c
}

func (c *committeeChainTest) reloadChain() {
	c.chain = NewTestCommitteeChain(c.db, &c.config, c.signerThreshold, c.enforceTime, c.clock)
}

func (c *committeeChainTest) setClockPeriod(period float64) {
	target := mclock.AbsTime(period * float64(time.Second*12*params.SyncPeriodLength))
	wait := time.Duration(target - c.clock.Now())
	if wait < 0 {
		c.t.Fatalf("Invalid setClockPeriod")
	}
	c.clock.Run(wait)
}

func (c *committeeChainTest) addFixedCommitteeRoot(tc *testCommitteeChain, period uint64, expErr error) {
	if err := c.chain.addFixedCommitteeRoot(period, tc.periods[period].committee.Root()); err != expErr {
		c.t.Errorf("Incorrect error output from addFixedCommitteeRoot at period %d (expected %v, got %v)", period, expErr, err)
	}
}

func (c *committeeChainTest) addCommittee(tc *testCommitteeChain, period uint64, expErr error) {
	if err := c.chain.addCommittee(period, tc.periods[period].committee); err != expErr {
		c.t.Errorf("Incorrect error output from addCommittee at period %d (expected %v, got %v)", period, expErr, err)
	}
}

func (c *committeeChainTest) insertUpdate(tc *testCommitteeChain, period uint64, addCommittee bool, expErr error) {
	var committee *types.SerializedSyncCommittee
	if addCommittee {
		committee = tc.periods[period+1].committee
	}
	if err := c.chain.InsertUpdate(tc.periods[period].update, committee); err != expErr {
		c.t.Errorf("Incorrect error output from InsertUpdate at period %d (expected %v, got %v)", period, expErr, err)
	}
}

func (c *committeeChainTest) verifySignedHeader(tc *testCommitteeChain, period float64, expOk bool) {
	slot := uint64(period * float64(params.SyncPeriodLength))
	signedHead := GenerateTestSignedHeader(types.Header{Slot: slot}, &tc.config, tc.periods[types.SyncPeriod(slot)].committee, slot+1, 400)
	if ok, _, _ := c.chain.VerifySignedHeader(signedHead); ok != expOk {
		c.t.Errorf("Incorrect output from VerifySignedHeader at period %f (expected %v, got %v)", period, expOk, ok)
	}
}

func (c *committeeChainTest) verifyRange(tc *testCommitteeChain, begin, end uint64) {
	if begin > 0 {
		c.verifySignedHeader(tc, float64(begin)-0.5, false)
	}
	for period := begin; period <= end; period++ {
		c.verifySignedHeader(tc, float64(period)+0.5, true)
	}
	c.verifySignedHeader(tc, float64(end)+1.5, false)
}

func newTestGenesis() types.ChainConfig {
	var config types.ChainConfig
	rand.Read(config.GenesisValidatorsRoot[:])
	return config
}

func newTestForks(config types.ChainConfig, forks types.Forks) types.ChainConfig {
	for _, fork := range forks {
		config.AddFork(fork.Name, fork.Epoch, fork.Version)
	}
	return config
}

func newTestCommitteeChain(parent *testCommitteeChain, config types.ChainConfig, newCommittees bool, begin, end int, signerCount int, finalizedHeader bool) *testCommitteeChain {
	tc := &testCommitteeChain{
		config: config,
	}
	if parent != nil {
		tc.periods = make([]testPeriod, len(parent.periods))
		copy(tc.periods, parent.periods)
	}
	if newCommittees {
		if begin == 0 {
			tc.fillCommittees(begin, end+1)
		} else {
			tc.fillCommittees(begin+1, end+1)
		}
	}
	tc.fillUpdates(begin, end, signerCount, finalizedHeader)
	return tc
}

type testPeriod struct {
	committee *types.SerializedSyncCommittee
	update    *types.LightClientUpdate
}

type testCommitteeChain struct {
	periods []testPeriod
	config  types.ChainConfig
}

func (tc *testCommitteeChain) fillCommittees(begin, end int) {
	if len(tc.periods) <= end {
		tc.periods = append(tc.periods, make([]testPeriod, end+1-len(tc.periods))...)
	}
	for i := begin; i <= end; i++ {
		tc.periods[i].committee = GenerateTestCommittee()
	}
}

func (tc *testCommitteeChain) fillUpdates(begin, end int, signerCount int, finalizedHeader bool) {
	for i := begin; i <= end; i++ {
		tc.periods[i].update = GenerateTestUpdate(&tc.config, uint64(i), tc.periods[i].committee, tc.periods[i+1].committee, signerCount, finalizedHeader)
	}
}
