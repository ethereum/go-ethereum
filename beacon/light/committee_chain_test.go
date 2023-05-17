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
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/minio/sha256-simd"
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

	tcBase                      = newTestCommitteeChain(nil, tfBase, true, 0, 10, 400, 400)
	tcBaseWithInvalidUpdates    = newTestCommitteeChain(tcBase, tfBase, false, 5, 10, 400, 200) // signer count too low
	tcBaseWithBetterUpdates     = newTestCommitteeChain(tcBase, tfBase, false, 5, 10, 400, 440)
	tcReorgWithWorseUpdates     = newTestCommitteeChain(tcBase, tfBase, true, 5, 10, 20, 400)
	tcReorgWithWorseUpdates2    = newTestCommitteeChain(tcBase, tfBase, true, 5, 10, 400, 380)
	tcReorgWithBetterUpdates    = newTestCommitteeChain(tcBase, tfBase, true, 5, 10, 400, 420)
	tcReorgWithFinalizedUpdates = newTestCommitteeChain(tcBase, tfBase, true, 5, 10, finalizedTestUpdate, 400)
	tcFork                      = newTestCommitteeChain(tcBase, tfAlternative, true, 7, 10, 400, 400)
	tcAnotherGenesis            = newTestCommitteeChain(nil, tfAnotherGenesis, true, 0, 10, 400, 400)
)

func TestCommitteeChainFixedRoots(t *testing.T) {
	for _, reload := range []bool{false, true} {
		c := newCommitteeChainTest(t, tfBase, 300, true)
		c.setClockPeriod(7)
		c.addFixedRoot(tcBase, 4, nil)
		c.addFixedRoot(tcBase, 5, nil)
		c.addFixedRoot(tcBase, 6, nil)
		c.addFixedRoot(tcBase, 8, ErrInvalidPeriod) // range has to be continuoous
		c.addFixedRoot(tcBase, 3, nil)
		c.addFixedRoot(tcBase, 2, nil)
		if reload {
			c.reloadChain()
		}
		c.addCommittee(tcBase, 4, nil)
		c.addCommittee(tcBase, 6, ErrInvalidPeriod) // range has to be continuoous
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
			c.addFixedRoot(tcBase, 3, nil)
			c.addFixedRoot(tcBase, 4, nil)
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
			c.addFixedRoot(tcBase, 2, nil)
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
			c.addFixedRoot(tcBase, 3, nil)
			c.addFixedRoot(tcBase, 4, nil)
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
			// have been improved beacuse a finalized update beats everything else
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
	c.addFixedRoot(tcBase, 0, nil)
	c.addFixedRoot(tcBase, 1, nil)
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
	c.chain = NewCommitteeChain(c.db, config, signerThreshold, enforceTime, dummyVerifier{}, c.clock, func() int64 { return int64(c.clock.Now()) })
	return c
}

func (c *committeeChainTest) reloadChain() {
	c.chain = NewCommitteeChain(c.db, c.config, c.signerThreshold, c.enforceTime, dummyVerifier{}, c.clock, func() int64 { return int64(c.clock.Now()) })
}

func (c *committeeChainTest) setClockPeriod(period float64) {
	target := mclock.AbsTime(period * float64(time.Second*12*params.SyncPeriodLength))
	wait := time.Duration(target - c.clock.Now())
	if wait < 0 {
		c.t.Fatalf("Invalid setClockPeriod")
	}
	c.clock.Run(wait)
}

func (c *committeeChainTest) addFixedRoot(tc *testCommitteeChain, period uint64, expErr error) {
	if err := c.chain.AddFixedRoot(period, tc.periods[period].committeeRoot); err != expErr {
		c.t.Errorf("Incorrect error output from AddFixedRoot at period %d (expected %v, got %v)", period, expErr, err)
	}
}

func (c *committeeChainTest) addCommittee(tc *testCommitteeChain, period uint64, expErr error) {
	if err := c.chain.AddCommittee(period, serializeDummySyncCommittee(tc.periods[period].committee)); err != expErr {
		c.t.Errorf("Incorrect error output from AddCommittee at period %d (expected %v, got %v)", period, expErr, err)
	}
}

func (c *committeeChainTest) insertUpdate(tc *testCommitteeChain, period uint64, addCommittee bool, expErr error) {
	var committee *types.SerializedSyncCommittee
	if addCommittee {
		committee = serializeDummySyncCommittee(tc.periods[period+1].committee)
	}
	if err := c.chain.InsertUpdate(&tc.periods[period].update, committee); err != expErr {
		c.t.Errorf("Incorrect error output from InsertUpdate at period %d (expected %v, got %v)", period, expErr, err)
	}
}

func (c *committeeChainTest) verifySignedHeader(tc *testCommitteeChain, period float64, expOk bool) {
	signedHead := tc.makeTestSignedHead(types.Header{Slot: uint64(period * float64(params.SyncPeriodLength))}, 400)
	if ok, _ := c.chain.VerifySignedHeader(signedHead); ok != expOk {
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

func newTestCommitteeChain(parent *testCommitteeChain, config types.ChainConfig, newCommittees bool, begin, end int, subPeriodIndex uint64, signerCount int) *testCommitteeChain {
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
	tc.fillUpdates(begin, end, subPeriodIndex, signerCount)
	return tc
}

func makeTestHeaderWithSingleProof(slot, index uint64, value merkle.Value) (types.Header, merkle.Values) {
	var branch merkle.Values
	hasher := sha256.New()
	for index > 1 {
		var proofHash merkle.Value
		rand.Read(proofHash[:])
		hasher.Reset()
		if index&1 == 0 {
			hasher.Write(value[:])
			hasher.Write(proofHash[:])
		} else {
			hasher.Write(proofHash[:])
			hasher.Write(value[:])
		}
		hasher.Sum(value[:0])
		index /= 2
		branch = append(branch, proofHash)
	}
	return types.Header{Slot: slot, StateRoot: common.Hash(value)}, branch
}

func makeBitmask(signerCount int) (bitmask [params.SyncCommitteeBitmaskSize]byte) {
	for i := 0; i < params.SyncCommitteeSize; i++ {
		if rand.Intn(params.SyncCommitteeSize-i) < signerCount {
			bitmask[i/8] += byte(1) << (i & 7)
			signerCount--
		}
	}
	return
}

type testPeriod struct {
	committee     dummySyncCommittee
	committeeRoot common.Hash
	update        types.LightClientUpdate
}

type testCommitteeChain struct {
	periods []testPeriod
	config  types.ChainConfig
}

func (tc *testCommitteeChain) makeTestSignedHead(header types.Header, signerCount int) types.SignedHeader {
	bitmask := makeBitmask(signerCount)
	signingRoot, _ := tc.config.Forks.SigningRoot(header)
	return types.SignedHeader{
		Header: header,
		Signature: types.SyncAggregate{
			Signers:   bitmask,
			Signature: makeDummySignature(tc.periods[types.SyncPeriod(header.Slot+1)].committee, signingRoot, bitmask),
		},
		SignatureSlot: header.Slot + 1,
	}
}

const finalizedTestUpdate = params.SyncPeriodLength - 1 // if subPeriodIndex == finalizedTestUpdate then a finalized update is generated

func (tc *testCommitteeChain) makeTestUpdate(period, subPeriodIndex uint64, signerCount int) types.LightClientUpdate {
	var update types.LightClientUpdate
	update.NextSyncCommitteeRoot = tc.periods[period+1].committeeRoot
	if subPeriodIndex == finalizedTestUpdate {
		update.FinalizedHeader = new(types.Header)
		*update.FinalizedHeader, update.NextSyncCommitteeBranch = makeTestHeaderWithSingleProof(types.SyncPeriodStart(period)+100, params.StateIndexNextSyncCommittee, merkle.Value(update.NextSyncCommitteeRoot))
		update.AttestedHeader.Header, update.FinalityBranch = makeTestHeaderWithSingleProof(types.SyncPeriodStart(period)+200, params.StateIndexFinalBlock, merkle.Value(update.FinalizedHeader.Hash()))
	} else {
		update.AttestedHeader.Header, update.NextSyncCommitteeBranch = makeTestHeaderWithSingleProof(types.SyncPeriodStart(period)+subPeriodIndex, params.StateIndexNextSyncCommittee, merkle.Value(update.NextSyncCommitteeRoot))
	}
	signedHead := tc.makeTestSignedHead(update.AttestedHeader.Header, signerCount)
	update.AttestedHeader.Signature = signedHead.Signature
	update.AttestedHeader.SignatureSlot = update.AttestedHeader.Header.Slot
	return update
}

func (tc *testCommitteeChain) fillCommittees(begin, end int) {
	if len(tc.periods) <= end {
		tc.periods = append(tc.periods, make([]testPeriod, end+1-len(tc.periods))...)
	}
	for i := begin; i <= end; i++ {
		tc.periods[i].committee = randomDummySyncCommittee()
		tc.periods[i].committeeRoot = serializeDummySyncCommittee(tc.periods[i].committee).Root()
	}
}

func (tc *testCommitteeChain) fillUpdates(begin, end int, subPeriodIndex uint64, signerCount int) {
	for i := begin; i <= end; i++ {
		tc.periods[i].update = tc.makeTestUpdate(uint64(i), subPeriodIndex, signerCount)
	}
}
