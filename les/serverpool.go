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

package les

import (
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	lpc "github.com/ethereum/go-ethereum/les/lespay/client"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	minTimeout          = time.Millisecond * 500 // minimum request timeout suggested by the server pool
	timeoutRefresh      = time.Second * 5        // recalculate timeout if older than this
	dialCost            = 10000                  // cost of a TCP dial (used for known node selection weight calculation)
	dialWaitStep        = 1.5                    // exponential multiplier of redial wait time when no value was provided by the server
	queryCost           = 500                    // cost of a UDP pre-negotiation query
	queryWaitStep       = 1.02                   // exponential multiplier of redial wait time when no value was provided by the server
	nodeWeightMul       = 1000000                // multiplier constant for node weight calculation
	nodeWeightThreshold = 100                    // minimum weight for keeping a node in the the known (valuable) set
	minRedialWait       = time.Second * 10       // minimum redial wait time
)

// serverPool provides a node iterator for dial candidates. The output is a mix of newly discovered
// nodes, a weighted random selection of known (previously valuable) nodes and trusted/paid nodes.
type serverPool struct {
	mono, rtc mclock.Clock
	db        ethdb.KeyValueStore

	ns           *nodestate.NodeStateMachine
	vt           *lpc.ValueTracker
	mixer        *enode.FairMix
	mixSources   []enode.Iterator
	dialIterator enode.Iterator
	trusted      []*enode.Node

	timeoutLock      sync.RWMutex
	timeout          time.Duration
	timeWeights      lpc.ResponseTimeWeights
	timeoutRefreshed mclock.AbsTime
}

// nodeHistory keeps track of dial costs which determine node weight together with the
// service value calculated by lpc.ValueTracker.
type nodeHistory struct {
	dialCost   utils.ExpiredValue
	waitUntil  mclock.AbsTime
	waitFactor float64
}

type nodeHistoryEnc struct {
	DialCost              utils.ExpiredValue
	WaitFactor, WaitUntil uint64
}

var (
	serverPoolSetup    = &nodestate.Setup{}
	sfHasValue         = serverPoolSetup.NewPersistentFlag("hasValue")
	sfQueried          = serverPoolSetup.NewFlag("queried")
	sfCanDial          = serverPoolSetup.NewFlag("canDial")
	sfDialed           = serverPoolSetup.NewFlag("dialed")
	sfConnected        = serverPoolSetup.NewFlag("connected")
	sfRedialWait       = serverPoolSetup.NewFlag("redialWait")
	sfAlwaysConnect    = serverPoolSetup.NewFlag("alwaysConnect")
	sfDisableSelection = nodestate.MergeFlags(sfQueried, sfCanDial, sfDialed, sfConnected, sfRedialWait)

	sfiNodeWeight  = serverPoolSetup.NewField("nodeWeight", reflect.TypeOf(uint64(0)))
	sfiNodeHistory = serverPoolSetup.NewPersistentField("nodeHistory", reflect.TypeOf(nodeHistory{}),
		func(field interface{}) ([]byte, error) {
			if n, ok := field.(nodeHistory); ok {
				ne := nodeHistoryEnc{
					DialCost:   n.dialCost,
					WaitFactor: uint64(n.waitFactor * 256),
					WaitUntil:  uint64(n.waitUntil),
				}
				enc, err := rlp.EncodeToBytes(&ne)
				return enc, err
			} else {
				return nil, errors.New("invalid field type")
			}
		},
		func(enc []byte) (interface{}, error) {
			var ne nodeHistoryEnc
			err := rlp.DecodeBytes(enc, &ne)
			n := nodeHistory{
				dialCost:   ne.DialCost,
				waitFactor: float64(ne.WaitFactor) / 256,
				waitUntil:  mclock.AbsTime(ne.WaitUntil),
			}
			return n, err
		},
	)
	sfiConnectedStats = serverPoolSetup.NewField("connectedStats", reflect.TypeOf(lpc.ResponseTimeStats{}))
)

// newServerPool creates a new server pool
func newServerPool(db ethdb.KeyValueStore, dbKey []byte, vt *lpc.ValueTracker, discovery enode.Iterator, query lpc.PreNegQuery, mono, rtc mclock.Clock, trustedURLs []string, testing bool) *serverPool {
	s := &serverPool{
		db:   db,
		mono: mono,
		rtc:  rtc,
		vt:   vt,
		ns:   nodestate.NewNodeStateMachine(db, []byte(string(dbKey)+"ns:"), mono, serverPoolSetup),
	}
	s.recalTimeout()
	var (
		validSchemes enr.IdentityScheme
		mixerTimeout time.Duration
	)
	if testing {
		validSchemes = enode.ValidSchemesForTesting
	} else {
		validSchemes = enode.ValidSchemes
		mixerTimeout = time.Second
	}

	for _, url := range trustedURLs {
		if node, err := enode.Parse(validSchemes, url); err == nil {
			s.trusted = append(s.trusted, node)
		} else {
			log.Error("Invalid trusted server URL", "url", url, "error", err)
		}
	}

	s.mixer = enode.NewFairMix(mixerTimeout)
	knownSelector := lpc.NewWrsIterator(s.ns, sfHasValue, sfDisableSelection, sfiNodeWeight)
	alwaysConnect := lpc.NewQueueIterator(s.ns, sfAlwaysConnect, sfDisableSelection, true)
	s.mixSources = append(s.mixSources, knownSelector)
	s.mixSources = append(s.mixSources, alwaysConnect)
	if discovery != nil {
		s.mixSources = append(s.mixSources, discovery)
	}

	iter := enode.Iterator(s.mixer)
	if query != nil {
		var testClock *mclock.Simulated
		if testing {
			testClock = mono.(*mclock.Simulated)
		}
		iter = lpc.NewPreNegFilter(s.ns, iter, query, sfQueried, sfCanDial, 5, time.Second*5, time.Second*10, testClock)
	}
	s.dialIterator = enode.Filter(iter, func(node *enode.Node) bool {
		s.ns.SetState(node, sfDialed, sfCanDial, time.Second*10)
		return true
	})

	s.ns.SubscribeState(nodestate.MergeFlags(sfDialed, sfConnected), func(n *enode.Node, oldState, newState nodestate.Flags) {
		if oldState.Equals(sfDialed) && newState.IsEmpty() {
			// dial timeout, no connection
			s.setRedialWait(n, dialCost, dialWaitStep)
		}
	})

	if query != nil {
		s.ns.SubscribeState(nodestate.MergeFlags(sfQueried, sfCanDial), func(n *enode.Node, oldState, newState nodestate.Flags) {
			if oldState.Equals(sfQueried) && newState.IsEmpty() {
				// query timeout, no connection
				s.setRedialWait(n, queryCost, queryWaitStep)
			}
		})
	}

	s.ns.AddLogMetrics(sfHasValue, sfDisableSelection, "selectable", nil, nil, serverSelectableGauge)
	s.ns.AddLogMetrics(sfDialed, nodestate.Flags{}, "dialed", serverDialedMeter, nil, nil)
	s.ns.AddLogMetrics(sfConnected, nodestate.Flags{}, "connected", nil, nil, serverConnectedGauge)
	return s
}

// start starts the server pool. Note that NodeStateMachine should be started first.
func (s *serverPool) start() {
	s.ns.Start()
	for _, iter := range s.mixSources {
		// add sources to mixer at startup because the mixer instantly tries to read them
		// which should only happen after NodeStateMachine has been started
		s.mixer.AddSource(iter)
	}
	for _, node := range s.trusted {
		s.ns.SetState(node, sfAlwaysConnect, nodestate.Flags{}, 0)
	}
	now := s.rtc.Now()
	s.ns.ForEach(sfHasValue, nodestate.Flags{}, func(node *enode.Node, state nodestate.Flags) {
		s.calculateWeight(node)
		if n, ok := s.ns.GetField(node, sfiNodeHistory).(nodeHistory); ok && n.waitUntil > now {
			wait := time.Duration(n.waitUntil - now)
			maxWait := n.redialWait()
			if wait > maxWait {
				// if the time until expiration is larger than the last suggested
				// waiting time then the system clock was probably adjusted
				wait = maxWait
			}
			s.ns.SetState(node, sfRedialWait, nodestate.Flags{}, wait)
		}
	})
}

// stop stops the server pool
func (s *serverPool) stop() {
	s.dialIterator.Close()
	s.ns.ForEach(sfConnected, nodestate.Flags{}, func(n *enode.Node, state nodestate.Flags) {
		// recalculate weight of connected nodes in order to update hasValue flag if necessary
		s.calculateWeight(n)
	})
	s.ns.Stop()
}

// registerPeer implements serverPeerSubscriber
func (s *serverPool) registerPeer(p *serverPeer) {
	s.ns.SetState(p.Node(), sfConnected, sfDialed, 0)
	nvt := s.vt.Register(p.ID())
	s.ns.SetField(p.Node(), sfiConnectedStats, nvt.RtStats())
	p.setValueTracker(s.vt, nvt)
	p.updateVtParams()
}

// unregisterPeer implements serverPeerSubscriber
func (s *serverPool) unregisterPeer(p *serverPeer) {
	s.setRedialWait(p.Node(), dialCost, dialWaitStep)
	s.ns.SetState(p.Node(), nodestate.Flags{}, sfConnected, 0)
	s.ns.SetField(p.Node(), sfiConnectedStats, nil)
	s.vt.Unregister(p.ID())
	p.setValueTracker(nil, nil)
}

// recalTimeout calculates the current recommended timeout. This value is used by
// the client as a "soft timeout" value. It also affects the service value calculation
// of individual nodes.
func (s *serverPool) recalTimeout() {
	// Use cached result if possible, avoid recalculating too frequently.
	s.timeoutLock.RLock()
	refreshed := s.timeoutRefreshed
	s.timeoutLock.RUnlock()
	now := s.mono.Now()
	if refreshed != 0 && time.Duration(now-refreshed) < timeoutRefresh {
		return
	}
	// Cached result is stale, recalculate a new one.
	rts := s.vt.RtStats()

	// Add a fake statistic here. It is an easy way to initialize with some
	// conservative values when the database is new. As soon as we have a
	// considerable amount of real stats this small value won't matter.
	rts.Add(time.Second*2, 10, s.vt.StatsExpFactor())

	// Use either 10% failure rate timeout or twice the median response time
	// as the recommended timeout.
	timeout := minTimeout
	if t := rts.Timeout(0.1); t > timeout {
		timeout = t
	}
	if t := rts.Timeout(0.5) * 2; t > timeout {
		timeout = t
	}
	s.timeoutLock.Lock()
	if s.timeout != timeout {
		s.timeout = timeout
		s.timeWeights = lpc.TimeoutWeights(s.timeout)

		suggestedTimeoutGauge.Update(int64(s.timeout / time.Millisecond))
		totalValueGauge.Update(int64(rts.Value(s.timeWeights, s.vt.StatsExpFactor())))
	}
	s.timeoutRefreshed = now
	s.timeoutLock.Unlock()
}

// getTimeout returns the recommended request timeout.
func (s *serverPool) getTimeout() time.Duration {
	s.recalTimeout()
	s.timeoutLock.RLock()
	defer s.timeoutLock.RUnlock()
	return s.timeout
}

// getTimeoutAndWeight returns the recommended request timeout as well as the
// response time weight which is necessary to calculate service value.
func (s *serverPool) getTimeoutAndWeight() (time.Duration, lpc.ResponseTimeWeights) {
	s.recalTimeout()
	s.timeoutLock.RLock()
	defer s.timeoutLock.RUnlock()
	return s.timeout, s.timeWeights
}

// addDialCost adds the given amount of dial cost to the node history and returns the current
// amount of total dial cost
func (s *serverPool) addDialCost(n *nodeHistory, amount int64) uint64 {
	logOffset := s.vt.StatsExpirer().LogOffset(s.mono.Now())
	if amount > 0 {
		n.dialCost.Add(amount, logOffset)
	}
	totalDialCost := n.dialCost.Value(logOffset)
	if totalDialCost < dialCost {
		totalDialCost = dialCost
	}
	return totalDialCost
}

// serviceValue returns the service value accumulated in this session and in total
func (s *serverPool) serviceValue(node *enode.Node) (sessionValue, totalValue float64) {
	nvt := s.vt.GetNode(node.ID())
	if nvt == nil {
		return 0, 0
	}
	currentStats := nvt.RtStats()
	_, timeWeights := s.getTimeoutAndWeight()
	expFactor := s.vt.StatsExpFactor()

	totalValue = currentStats.Value(timeWeights, expFactor)
	if connStats, ok := s.ns.GetField(node, sfiConnectedStats).(lpc.ResponseTimeStats); ok {
		diff := currentStats
		diff.SubStats(&connStats)
		sessionValue = diff.Value(timeWeights, expFactor)
		sessionValueMeter.Mark(int64(sessionValue))
	}
	return
}

// updateWeight calculates the node weight and updates the nodeWeight field and the
// hasValue flag. It also saves the node state if necessary.
func (s *serverPool) updateWeight(node *enode.Node, totalValue float64, totalDialCost uint64) {
	weight := uint64(totalValue * nodeWeightMul / float64(totalDialCost))
	if weight >= nodeWeightThreshold {
		s.ns.SetState(node, sfHasValue, nodestate.Flags{}, 0)
		s.ns.SetField(node, sfiNodeWeight, weight)
	} else {
		s.ns.SetState(node, nodestate.Flags{}, sfHasValue, 0)
		s.ns.SetField(node, sfiNodeWeight, nil)
	}
	s.ns.Persist(node) // saved if node history or hasValue changed
}

// redialWait returns the suggested waiting time before the node should be queried or
// dialed again
func (n *nodeHistory) redialWait() time.Duration {
	return time.Duration(float64(minRedialWait) * n.waitFactor)
}

// setRedialWait calculates and sets the redialWait timeout based on the service value
// and dial cost accumulated during the last session/attempt and in total.
// The waiting time is raised exponentially if no service value has been received in order
// to prevent dialing an unresponsive node frequently for a very long time just because it
// was useful in the past. It can still be occasionally dialed though and once it provides
// a significant amount of service value again its waiting time is quickly reduced or reset
// to the minimum.
// Note: node weight is also recalculated and updated by this function.
func (s *serverPool) setRedialWait(node *enode.Node, addDialCost int64, waitStep float64) {
	n, _ := s.ns.GetField(node, sfiNodeHistory).(nodeHistory)
	sessionValue, totalValue := s.serviceValue(node)
	totalDialCost := s.addDialCost(&n, addDialCost)

	// if the current dial session has yielded at least the average value/dial cost ratio
	// then waitFactor should be reset to 1 (the minimum value). If the session value
	// is below average but still positive then waitFactor is limited to the ration of
	// average / current service value. If the attempt was unsuccessful then waitFactor
	// is raised exponentially without limitation.
	// Note: dialCost is used in the formula below even if dial was not attempted at all
	// because the pre-negotiation query did not return a positive result. In this case
	// the ratio has no meaning anyway and waitFactor is always raised, though in smaller
	// steps because queries are cheaper and therefore we can allow more failed attempts.
	a := totalValue * dialCost
	b := float64(totalDialCost) * sessionValue
	if n.waitFactor < 1 {
		n.waitFactor = 1
	}
	n.waitFactor *= waitStep
	if a < b*n.waitFactor {
		n.waitFactor = a / b
	}
	if n.waitFactor < 1 {
		n.waitFactor = 1
	}
	wait := n.redialWait()
	n.waitUntil = s.rtc.Now() + mclock.AbsTime(wait)
	s.ns.SetField(node, sfiNodeHistory, n)
	s.ns.SetState(node, sfRedialWait, nodestate.Flags{}, wait)
	s.updateWeight(node, totalValue, totalDialCost)
}

// calculateWeight calculates and sets the node weight without altering the node history.
// This function should be called during startup and shutdown only, otherwise setRedialWait
// will keep the weights updated as the underlying statistics are adjusted.
func (s *serverPool) calculateWeight(node *enode.Node) {
	n, _ := s.ns.GetField(node, sfiNodeHistory).(nodeHistory)
	_, totalValue := s.serviceValue(node)
	totalDialCost := s.addDialCost(&n, 0)
	s.updateWeight(node, totalValue, totalDialCost)
}
