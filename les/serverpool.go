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
	"encoding/binary"
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
	queryCost           = 500                    // cost of a UDP pre-negotiation query
	nodeWeightMul       = 1000000                // multiplier constant for node weight calculation
	nodeWeightThreshold = 100                    // minimum weight for keeping a node in the the known (valuable) set
	redialWaitStep      = 2                      // exponential multiplier of redial wait time when no value was provided by the server
	minRedialWait       = time.Second * 10       // minimum redial wait time
)

// serverPool provides a node iterator for dial candidates. The output is a mix of newly discovered
// nodes, a weighted random selection of known (previously valuable) nodes and trusted/paid nodes.
type serverPool struct {
	clock       mclock.Clock
	clockOffset mclock.AbsTime
	db          ethdb.KeyValueStore
	dbClockKey  []byte
	quit        chan struct{}

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
	serverPoolSetup = &nodestate.Setup{}

	sfHasValue      = serverPoolSetup.NewPersistentFlag("hasValue")
	sfQueried       = serverPoolSetup.NewFlag("queried")
	sfCanDial       = serverPoolSetup.NewFlag("canDial")
	sfDialed        = serverPoolSetup.NewFlag("dialed")
	sfConnected     = serverPoolSetup.NewFlag("connected")
	sfRedialWait    = serverPoolSetup.NewFlag("redialWait")
	sfAlwaysConnect = serverPoolSetup.NewFlag("alwaysConnect")

	sfDisableSelection = nodestate.MergeFlags(sfQueried, sfCanDial, sfDialed, sfConnected, sfRedialWait)

	errInvalidField = errors.New("invalid field type")

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
				return nil, errInvalidField
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
func newServerPool(db ethdb.KeyValueStore, dbKey []byte, vt *lpc.ValueTracker, discovery enode.Iterator, query lpc.PreNegQuery, clock mclock.Clock, trustedURLs []string, testing bool) *serverPool {
	s := &serverPool{
		db:         db,
		dbClockKey: append(dbKey, []byte("persistentClock")...),
		clock:      clock,
		ns:         nodestate.NewNodeStateMachine(db, []byte(string(dbKey)+"ns:"), clock, serverPoolSetup),
		vt:         vt,
		quit:       make(chan struct{}),
	}
	s.getTimeout()
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
			testClock = clock.(*mclock.Simulated)
		}
		iter = lpc.NewPreNegFilter(s.ns, iter, query, sfQueried, sfCanDial, 5, testClock)
	}
	s.dialIterator = enode.Filter(iter, func(node *enode.Node) bool {
		s.ns.SetState(node, sfDialed, sfCanDial, time.Second*10)
		return true
	})

	s.ns.SubscribeState(nodestate.MergeFlags(sfDialed, sfConnected), func(n *enode.Node, oldState, newState nodestate.Flags) {
		if oldState.Equals(sfDialed) && newState.IsEmpty() {
			// dial timeout, no connection
			s.updateNode(n, false, true, dialCost)
		}
	})

	if query != nil {
		s.ns.SubscribeState(nodestate.MergeFlags(sfQueried, sfCanDial), func(n *enode.Node, oldState, newState nodestate.Flags) {
			if oldState.Equals(sfQueried) && newState.IsEmpty() {
				// query timeout, no connection
				s.updateNode(n, false, true, queryCost)
			}
		})
	}

	s.ns.AddLogMetrics(sfHasValue, sfDisableSelection, "selectable", nil, nil, serverSelectableGauge)
	s.ns.AddLogMetrics(sfDialed, nodestate.Flags{}, "dialed", serverDialedMeter, nil, nil)
	s.ns.AddLogMetrics(sfConnected, nodestate.Flags{}, "connected", nil, nil, serverConnectedGauge)
	return s
}

func dummyQuery(n *enode.Node, result func(canDial bool)) (cancel func()) { //TODO remove, for testing only
	cancelCh := make(chan struct{})
	go func() {
		select {
		case <-time.After(time.Second):
			result(true)
		case <-cancelCh:
			result(false)
		}
	}()
	return func() {
		close(cancelCh)
	}
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
	clockEnc, _ := s.db.Get(s.dbClockKey)
	var clockStart mclock.AbsTime
	if len(clockEnc) == 8 {
		clockStart = mclock.AbsTime(binary.BigEndian.Uint64(clockEnc))
	}
	s.clockOffset = clockStart - s.clock.Now()
	s.ns.ForEach(sfHasValue, nodestate.Flags{}, func(node *enode.Node, state nodestate.Flags) {
		s.updateNode(node, false, false, 0) // set weight flag
		if n, ok := s.ns.GetField(node, sfiNodeHistory).(nodeHistory); ok && n.waitUntil > clockStart {
			s.ns.SetState(node, sfRedialWait, nodestate.Flags{}, time.Duration(n.waitUntil-clockStart))
		}
	})
	go func() {
		for {
			select {
			case <-time.After(time.Minute * 5):
				s.persistClock()
				suggestedTimeoutGauge.Update(int64(s.getTimeout() / time.Millisecond))
				s.timeoutLock.RLock()
				timeWeights := s.timeWeights
				s.timeoutLock.RUnlock()
				totalValueGauge.Update(int64(s.vt.RtStats().Value(timeWeights, s.vt.StatsExpFactor())))
			case <-s.quit:
				return
			}
		}
	}()
}

// stop stops the server pool
func (s *serverPool) stop() {
	s.dialIterator.Close()
	s.ns.ForEach(sfConnected, nodestate.Flags{}, func(n *enode.Node, state nodestate.Flags) {
		s.updateNode(n, true, false, 0)
	})
	close(s.quit)
	s.persistClock()
	s.ns.Stop()
}

// persistClock stores the persistent absolute time into the database
func (s *serverPool) persistClock() {
	var clockEnc [8]byte
	binary.BigEndian.PutUint64(clockEnc[:], uint64(s.clock.Now()+s.clockOffset))
	s.db.Put(s.dbClockKey, clockEnc[:])
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
	s.updateNode(p.Node(), true, true, dialCost)
	s.ns.SetState(p.Node(), nodestate.Flags{}, sfConnected, 0)
	s.ns.SetField(p.Node(), sfiConnectedStats, nil)
	s.vt.Unregister(p.ID())
	p.setValueTracker(nil, nil)
}

// getTimeout calculates the current recommended timeout. This value is used by
// the client as a "soft timeout" value. It also affects the service value calculation
// of individual nodes.
func (s *serverPool) getTimeout() time.Duration {
	now := s.clock.Now()
	s.timeoutLock.RLock()
	timeout := s.timeout
	refreshed := s.timeoutRefreshed
	s.timeoutLock.RUnlock()
	if refreshed != 0 && time.Duration(now-refreshed) < timeoutRefresh {
		return timeout
	}
	rts := s.vt.RtStats()
	rts.Add(time.Second*2, 10, s.vt.StatsExpFactor())
	timeout = minTimeout
	if t := rts.Timeout(0.1); t > timeout {
		timeout = t
	}
	if t := rts.Timeout(0.5) * 2; t > timeout {
		timeout = t
	}
	s.timeoutLock.Lock()
	if s.timeout != timeout {
		s.timeout = timeout
		s.timeWeights = lpc.TimeoutWeights(timeout)
	}
	s.timeoutRefreshed = now
	s.timeoutLock.Unlock()
	return timeout
}

// updateNode calculates the selection weight and the proposed redial wait time of the given node
func (s *serverPool) updateNode(node *enode.Node, calculateSessionValue, redialWait bool, addDialCost int64) {
	n, _ := s.ns.GetField(node, sfiNodeHistory).(nodeHistory)
	nvt := s.vt.GetNode(node.ID())
	if nvt == nil {
		return
	}
	currentStats := nvt.RtStats()
	s.getTimeout() // updates s.timeWeights
	s.timeoutLock.RLock()
	timeWeights := s.timeWeights
	s.timeoutLock.RUnlock()
	expFactor := s.vt.StatsExpFactor()

	var sessionValue float64
	if calculateSessionValue {
		if connStats, ok := s.ns.GetField(node, sfiConnectedStats).(lpc.ResponseTimeStats); ok {
			diff := currentStats
			diff.SubStats(&connStats)
			sessionValue = diff.Value(timeWeights, expFactor)
			sessionValueMeter.Mark(int64(sessionValue))
		}
	}

	logOffset := s.vt.StatsExpirer().LogOffset(s.clock.Now())
	if addDialCost > 0 {
		n.dialCost.Add(addDialCost, logOffset)
	}
	totalDialCost := n.dialCost.Value(logOffset)
	if totalDialCost < dialCost {
		totalDialCost = dialCost
	}

	totalValue := currentStats.Value(timeWeights, expFactor)
	if redialWait {
		a := totalValue * dialCost
		b := float64(totalDialCost) * sessionValue
		if n.waitFactor < 1 {
			n.waitFactor = 1
		}
		n.waitFactor *= redialWaitStep
		if a < b*n.waitFactor {
			n.waitFactor = a / b
		}
		if n.waitFactor < 1 {
			n.waitFactor = 1
		}
		wait := time.Duration(float64(minRedialWait) * n.waitFactor)
		n.waitUntil = s.clock.Now() + s.clockOffset + mclock.AbsTime(wait)
		s.ns.SetField(node, sfiNodeHistory, n)
		s.ns.SetState(node, sfRedialWait, nodestate.Flags{}, wait)
	}

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
