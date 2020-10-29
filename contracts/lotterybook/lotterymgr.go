// Copyright 2020 The go-ethereum Authors
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

package lotterybook

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/contracts/lotterybook/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

var errLotteryManagerClosed = errors.New("lottery manager closed")

const (
	activeLotteryQuery = iota
	expiredLotteryQuery
)

const (
	maxVerifyRetry = 5
	verifyDistance = 2
)

type queryReq struct {
	typ int
	ret chan []*Lottery
}

// verifyset is the set used to maintain all "pending" lotteries
// waiting for present verification.
type verifyset struct {
	set   map[common.Hash]*Lottery
	calls map[common.Hash]func(bool)
	next  uint64
}

func newVerifySet() *verifyset {
	return &verifyset{
		set:   make(map[common.Hash]*Lottery),
		calls: make(map[common.Hash]func(bool)),
		next:  math.MaxUint64,
	}
}

func (s *verifyset) add(lottery *Lottery, confirmed func(bool)) {
	s.set[lottery.Id] = lottery
	if confirmed != nil {
		s.calls[lottery.Id] = confirmed
	}
	if lottery.NextCheck < s.next {
		s.next = lottery.NextCheck
	}
}

func (s *verifyset) checkItems(height uint64) ([]*Lottery, []func(bool)) {
	if len(s.set) == 0 || s.next > height {
		return nil, nil
	}
	var items []*Lottery
	var calls []func(bool)
	for _, l := range s.set {
		if l.NextCheck < height {
			items = append(items, l)
			calls = append(calls, s.calls[l.Id])
		}
	}
	return items, calls
}

// iterateAndUpdate iterates the whole set and update the set flag.
// If the callback returns true, the iterated element is deleted.
func (s *verifyset) iterateAndUpdate(callback func(l *Lottery) bool) {
	var next = uint64(math.MaxUint64)
	for id, l := range s.set {
		if callback(l) {
			delete(s.set, id)
			delete(s.calls, id)
			continue
		}
		if l.NextCheck < next {
			next = l.NextCheck
		}
	}
}

// lotteryManager the manager of local create lotteries for
// all life cycle management.
type lotteryManager struct {
	address     common.Address        // The address of payment sender
	chainReader Blockchain            // The instance use to access local chain
	contract    *contract.LotteryBook // The instance of lottery contract
	cdb         *chequeDB             // The database used to store all payment data
	scope       event.SubscriptionScope
	lotteryFeed event.Feed

	// Lottery sets
	pendingSet   map[uint64][]*Lottery    // A set of pending lotteries, the key is block height of creation
	verifyingSet *verifyset               // A set of pending lotteries which is waiting for present verification
	activeSet    map[uint64][]*Lottery    // A set of active lotteries, the key is block height of reveal
	revealedSet  map[common.Hash]*Lottery // A set of revealed lotteries
	expiredSet   map[common.Hash]*Lottery // A set of expired lotteries

	// Channels
	lotteryCh chan *Lottery
	queryCh   chan *queryReq
	deleteCh  chan common.Hash

	closeCh chan struct{}
	wg      sync.WaitGroup
	wipeFn  func(common.Hash, bool)
	storeFn func(*Lottery)

	// Testing
	verifyDone func()
	verifyHook func(*Lottery) bool
}

// newLotteryManager returns an instance of lottery manager and starts
// the underlying routines for status management.
func newLotteryManager(address common.Address, chainReader Blockchain, contract *contract.LotteryBook, cdb *chequeDB, wipeLottery func(common.Hash, bool), storeLottery func(*Lottery)) *lotteryManager {
	m := &lotteryManager{
		address:      address,
		chainReader:  chainReader,
		contract:     contract,
		cdb:          cdb,
		pendingSet:   make(map[uint64][]*Lottery),
		verifyingSet: newVerifySet(),
		activeSet:    make(map[uint64][]*Lottery),
		revealedSet:  make(map[common.Hash]*Lottery),
		expiredSet:   make(map[common.Hash]*Lottery),
		lotteryCh:    make(chan *Lottery),
		queryCh:      make(chan *queryReq),
		deleteCh:     make(chan common.Hash),
		closeCh:      make(chan struct{}),
		wipeFn:       wipeLottery,
		storeFn:      storeLottery,
	}
	m.wg.Add(1)
	go m.run()
	return m
}

// verifyLottery checks whether the lottery is present in the contract.
func (m *lotteryManager) verifyLottery(done chan struct{}, height uint64, lotteries []*Lottery, calls []func(bool)) {
	defer close(done)

	for index, lottery := range lotteries {
		// We are runnning unit tests, use the hook
		if m.verifyHook != nil {
			if m.verifyHook(lottery) {
				lottery.Confirmed = true
				if calls[index] != nil {
					calls[index](true)
				}
			} else {
				lottery.Checks += 1
				if lottery.Checks >= maxVerifyRetry {
					lottery.Lost = true
					if calls[index] != nil {
						calls[index](false)
					}
					continue
				}
				lottery.NextCheck = height + uint64(lottery.Checks*verifyDistance)
			}
			continue
		}
		// It's not unit tests, query contract
		ret, err := m.contract.Lotteries(nil, lottery.Id)
		if err == nil && ret.Amount != 0 {
			lottery.Confirmed = true
			if calls[index] != nil {
				calls[index](true)
			}
			continue
		}
		lottery.Checks += 1
		if lottery.Checks >= maxVerifyRetry {
			lottery.Lost = true
			if calls[index] != nil {
				calls[index](false)
			}
			log.Debug("Lottery is lost", "id", lottery.Id, "amount", lottery.Amount, "createAt", lottery.CreateAt)
			continue
		}
		lottery.NextCheck = height + uint64(lottery.Checks*verifyDistance)
		log.Debug("Lottery is not comfirmed", "id", lottery.Id, "amount", lottery.Amount, "createAt", lottery.CreateAt, "checks", lottery.Checks)
	}
}

// recoverCrash reloads all tmp lottery records which we (may or may not) send
// the transaction out but system crash. We need to ensure whether these lotteries
// are created or not.
func (m *lotteryManager) recoverCrash(current uint64) {
	tmpLotteries := m.cdb.listLotteries(m.address, true)
	for _, lottery := range tmpLotteries {
		ret, err := m.contract.Lotteries(nil, lottery.Id)
		if err != nil {
			continue
		}
		// Although the tmp record is not removed yet, but lottery is
		// already been confirmed, delete the stale record.
		if l := m.cdb.readLottery(m.address, lottery.Id); l != nil {
			m.cdb.deleteLottery(m.address, lottery.Id, true) // Delete the tmp record
			continue
		}
		// Yeah! We recover a unsaved lottery, assign the current
		// block number as the "creation height" and let it wait
		// a few confirms.
		if ret.Amount != 0 {
			lottery.CreateAt = current
			m.cdb.writeLottery(m.address, lottery.Id, false, lottery)
			log.Debug("Recovered unsaved lottery", "id", lottery.Id, "amount", lottery.Amount)
			continue
		}
		// The lottery is not registered in the contract yet. Is the transaction
		// still in the pending list? Or we just never send the transaction out?
		// We can't make any meaningful decision here, so put it in the verifying
		// set and wait it for several blocks.

		// The verification is not started immediately. There is a special case:
		// Client creates the lottery and then crash. It restarts immediately,
		// so there is no confirmed be applied yet. Wait a few blocks and then verify.
		lottery.NextCheck = current + lotteryProcessConfirms
		m.verifyingSet.add(lottery, func(confirmed bool) {
			if confirmed {
				// We really claim it back, now convert it to a spendable lottery.
				lottery.CreateAt = current
				m.cdb.writeLottery(m.address, lottery.Id, false, lottery)
				m.cdb.deleteLottery(m.address, lottery.Id, true) // Delete the tmp record
			} else {
				// After wait the lottery for several blocks, it's still
				// not confirmed, drop it right now.
				m.wipeFn(lottery.Id, true) // tmp = true, will delete associated cheques as well
			}
		})
	}
}

// recover reloads all stored lotteries in disk during setup(include the recovered)
// and classify to different categories.
func (m *lotteryManager) recover(current uint64) (events []LotteryEvent) {
	lotteries := m.cdb.listLotteries(m.address, false)
	for _, lottery := range lotteries {
		if current < lottery.RevealNumber {
			if lottery.CreateAt+lotteryProcessConfirms < current {
				m.pendingSet[lottery.CreateAt] = append(m.pendingSet[lottery.CreateAt], lottery)
				events = append(events, LotteryEvent{Id: lottery.Id, Status: LotteryPending})
			} else {
				if lottery.Confirmed {
					m.activeSet[lottery.RevealNumber] = append(m.activeSet[lottery.RevealNumber], lottery)
					events = append(events, LotteryEvent{Id: lottery.Id, Status: LotteryActive})
				} else {
					m.verifyingSet.add(lottery, nil)
				}
			}
		} else if current < lottery.RevealNumber+lotteryClaimPeriod+lotteryProcessConfirms {
			m.revealedSet[lottery.Id] = lottery
			events = append(events, LotteryEvent{Id: lottery.Id, Status: LotteryRevealed})
		} else {
			m.expiredSet[lottery.Id] = lottery
			events = append(events, LotteryEvent{Id: lottery.Id, Status: LotteryExpired})
		}
	}
	return events
}

// run is responsible for managing the entire life cycle of the
// lotteries created locally.
//
// The status of lottery can be classified as four types:
// * pending: lottery is just created, have to wait a few block
//    confirms upon it.
// * active: lottery can be used to make payment, the lottery
//    reveal time has not yet arrived.
// * revealed: lottery has been revealed, can't be used to make
//    payment anymore.
// * expired: no one picks up lottery ticket within required claim
//    time, owner can reown it via resetting or destruct.
//
// External modules can monitor lottery status via subscription or
// direct query.
func (m *lotteryManager) run() {
	defer m.wg.Done()

	// Establish subscriptions
	newHeadCh := make(chan core.ChainHeadEvent, 1024)
	sub := m.chainReader.SubscribeChainHeadEvent(newHeadCh)
	if sub == nil {
		return
	}
	defer sub.Unsubscribe()

	lotteryClaimedCh := make(chan *contract.LotteryBookLotteryClaimed)
	eventSub, err := m.contract.WatchLotteryClaimed(nil, lotteryClaimedCh, nil)
	if err != nil {
		return
	}
	defer eventSub.Unsubscribe()

	var (
		events  []LotteryEvent                                  // A batch of cumulative lottery events
		verify  chan struct{}                                   // Non-nil if the underlying verifying thread is running
		current = m.chainReader.CurrentHeader().Number.Uint64() // Current height of local blockchain
	)
	m.recoverCrash(current)
	events = m.recover(current)

	// Setup a ticker for expired lottery GC
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		if len(events) > 0 {
			m.lotteryFeed.Send(events)
			events = events[:0]
		}
		if verify == nil {
			needCheck, calls := m.verifyingSet.checkItems(current)
			if len(needCheck) > 0 {
				verify = make(chan struct{})
				go m.verifyLottery(verify, current, needCheck, calls)
			}
		}
		select {
		case ev := <-newHeadCh:
			current = ev.Block.NumberU64()

			// If newly created lotteries have enough confirms, move them to verifying set.
			for createdAt, lotteries := range m.pendingSet {
				if createdAt+lotteryProcessConfirms < current {
					continue
				}
				for _, lottery := range lotteries {
					m.verifyingSet.add(lottery, nil)
				}
				delete(m.pendingSet, createdAt)
			}
			// Clean stale lottery which is already revealed
			for revealAt, lotteries := range m.activeSet {
				if current < revealAt {
					continue
				}
				// Move all revealed lotteries into `revealed` set.
				for _, lottery := range lotteries {
					m.revealedSet[lottery.Id] = lottery
					events = append(events, LotteryEvent{Id: lottery.Id, Status: LotteryRevealed})
					log.Debug("Lottery revealed", "id", lottery.Id)
				}
				delete(m.activeSet, revealAt)
			}
			// Clean stale lottery which is already expired.
			for id, lottery := range m.revealedSet {
				// Move all expired lotteries into `expired` set.
				if lottery.RevealNumber+lotteryClaimPeriod+lotteryProcessConfirms <= current {
					events = append(events, LotteryEvent{Id: lottery.Id, Status: LotteryExpired})
					m.expiredSet[id] = lottery
					delete(m.revealedSet, id)
					log.Debug("Lottery expired", "id", lottery.Id)
				}
			}
		case <-verify:
			verify = nil

			m.verifyingSet.iterateAndUpdate(func(lottery *Lottery) bool {
				if lottery.Confirmed {
					m.storeFn(lottery) // Persist this confirmed flag
					m.activeSet[lottery.RevealNumber] = append(m.activeSet[lottery.RevealNumber], lottery)
					events = append(events, LotteryEvent{Id: lottery.Id, Status: LotteryActive})
					log.Debug("Lottery activated", "id", lottery.Id)
					return true
				}
				if lottery.Lost {
					m.wipeFn(lottery.Id, true) // tmp = true, will delete associated cheques as well
					events = append(events, LotteryEvent{Id: lottery.Id, Status: LotteryLost, Lottery: lottery})
					log.Debug("Lottery lost", "id", lottery.Id)
					return true
				}
				return false
			})

			if m.verifyDone != nil {
				m.verifyDone()
			}

		case lottery := <-m.lotteryCh:
			m.pendingSet[current] = append(m.pendingSet[current], lottery)
			events = append(events, LotteryEvent{Id: lottery.Id, Status: LotteryPending})
			log.Debug("Lottery created", "id", lottery.Id, "amout", lottery.Amount, "revealnumber", lottery.RevealNumber)

		case claimedEvent := <-lotteryClaimedCh:
			id := common.Hash(claimedEvent.Id)
			if _, exist := m.revealedSet[id]; exist {
				delete(m.revealedSet, id)
			}
			if _, exist := m.expiredSet[id]; exist {
				delete(m.expiredSet, id)
			}
			m.wipeFn(id, false)
			log.Debug("Lottery claimed", "id", id)

		case req := <-m.queryCh:
			if req.typ == activeLotteryQuery {
				var ret []*Lottery
				for _, lotteries := range m.activeSet {
					ret = append(ret, lotteries...)
				}
				req.ret <- ret
			} else if req.typ == expiredLotteryQuery {
				var ret []*Lottery
				for _, lottery := range m.expiredSet {
					ret = append(ret, lottery)
				}
				req.ret <- ret
			}

		case id := <-m.deleteCh:
			delete(m.expiredSet, id) // The expired lottery is reset or destroyed
			m.wipeFn(id, false)

		case <-ticker.C:
			for id := range m.expiredSet {
				// Note it might be expensive for light client to retrieve
				// information from contract.
				ret, err := m.contract.Lotteries(nil, id)
				if err != nil {
					continue
				}
				if ret.Amount == 0 {
					delete(m.expiredSet, id)
					m.wipeFn(id, false)
					log.Debug("Lottery removed", "id", id)
				}
				// Otherwise it can be reowned by sender, keep it.
			}

		case <-m.closeCh:
			return
		}
	}
}

// trackLottery adds a newly created lottery for life cycle management.
func (m *lotteryManager) trackLottery(l *Lottery) error {
	select {
	case m.lotteryCh <- l:
		return nil
	case <-m.closeCh:
		return errLotteryManagerClosed
	}
}

// activeLotteries returns all active lotteries which can be used
// to make payment.
func (m *lotteryManager) activeLotteries() ([]*Lottery, error) {
	reqCh := make(chan []*Lottery, 1)
	select {
	case m.queryCh <- &queryReq{
		typ: activeLotteryQuery,
		ret: reqCh,
	}:
		return <-reqCh, nil
	case <-m.closeCh:
		return nil, errLotteryManagerClosed
	}
}

// expiredLotteries returns all expired lotteries which can be reowned.
func (m *lotteryManager) expiredLotteries() ([]*Lottery, error) {
	reqCh := make(chan []*Lottery, 1)
	select {
	case m.queryCh <- &queryReq{
		typ: expiredLotteryQuery,
		ret: reqCh,
	}:
		return <-reqCh, nil
	case <-m.closeCh:
		return nil, errLotteryManagerClosed
	}
}

func (m *lotteryManager) deleteExpired(id common.Hash) error {
	select {
	case m.deleteCh <- id:
		return nil
	case <-m.closeCh:
		return errLotteryManagerClosed
	}
}

// subscribeLotteryEvent registers a subscription of LotteryEvent.
func (m *lotteryManager) subscribeLotteryEvent(ch chan<- []LotteryEvent) event.Subscription {
	return m.scope.Track(m.lotteryFeed.Subscribe(ch))
}

func (m *lotteryManager) close() {
	m.scope.Close()
	close(m.closeCh)
	m.wg.Wait()
}
