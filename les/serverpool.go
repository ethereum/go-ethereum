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
	minTimeout             = time.Millisecond * 500 // minimum request timeout suggested by the server pool
	timeoutRefresh         = time.Second * 5        // recalculate timeout if older than this
	timeoutChangeThreshold = time.Millisecond * 10  // recalculate node values if timeout has changed more than this amount
	dialCost               = 10000                  // cost of a TCP dial (used for known node selection weight calculation)
	nodeWeightMul          = 1000000                // multiplier constant for node weight calculation
	nodeWeightThreshold    = 100                    // minimum weight for keeping a node in the the known (valuable) set
	redialWaitStep         = 2                      // exponential multiplier of redial wait time when no value was provided by the server
	minRedialWait          = time.Second * 10       // minimum redial wait time
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
	// only dialCost, waitFactor and waitUntil are saved
	dialCost               utils.ExpiredValue
	waitUntil              mclock.AbsTime
	lastTimeout            time.Duration
	totalValue, waitFactor float64
}

type nodeHistoryEnc struct {
	DialCost              utils.ExpiredValue
	WaitFactor, WaitUntil uint64
}

var (
	serverPoolSetup = &nodestate.Setup{}

	sfHasValue      = serverPoolSetup.NewPersistentFlag("hasValue")
	sfSelected      = serverPoolSetup.NewFlag("selected")
	sfDialed        = serverPoolSetup.NewFlag("dialed")
	sfConnected     = serverPoolSetup.NewFlag("connected")
	sfRedialWait    = serverPoolSetup.NewFlag("redialWait")
	sfAlwaysConnect = serverPoolSetup.NewFlag("alwaysConnect")

	sfDisableSelection = nodestate.MergeFlags(sfSelected, sfDialed, sfConnected, sfRedialWait)

	errInvalidField = errors.New("invalid field type")

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
func newServerPool(db ethdb.KeyValueStore, dbKey []byte, ns *nodestate.NodeStateMachine, vt *lpc.ValueTracker, discovery enode.Iterator, clock mclock.Clock, trustedURLs []string, testing bool) *serverPool {
	s := &serverPool{
		db:         db,
		dbClockKey: append(dbKey, []byte("persistentClock")...),
		clock:      clock,
		ns:         ns,
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
	knownSelector := lpc.NewWrsIterator(s.ns, sfHasValue, sfDisableSelection, sfSelected, s.knownSelectWeight)
	alwaysConnect := lpc.NewQueueIterator(s.ns, sfAlwaysConnect, sfDisableSelection, sfSelected)
	s.mixSources = append(s.mixSources, knownSelector)
	s.mixSources = append(s.mixSources, alwaysConnect)
	if discovery != nil {
		s.mixSources = append(s.mixSources, discovery)
	}

	// preNegotiationFilter will be added in series with iter here when les4 is available

	s.dialIterator = enode.Filter(s.mixer, func(node *enode.Node) bool {
		s.ns.SetState(node, sfDialed, nodestate.Flags{}, time.Second*10)
		return true
	})

	ns.SubscribeState(sfDialed.Or(sfConnected), func(n *enode.Node, oldState, newState nodestate.Flags) {
		if oldState.Equals(sfDialed) && newState.IsEmpty() {
			// dial timeout, no connection
			_, wait := s.calculateNode(n, true, false)
			s.ns.SetState(n, sfRedialWait, nodestate.Flags{}, wait)
		}
	})

	ns.AddLogMetrics(sfHasValue, sfDisableSelection, "selectable", nil, nil, serverSelectableGauge)
	ns.AddLogMetrics(sfDialed, nodestate.Flags{}, "dialed", serverDialedMeter, nil, nil)
	ns.AddLogMetrics(sfConnected, nodestate.Flags{}, "connected", nil, nil, serverConnectedGauge)
	return s
}

// start starts the server pool. Note that NodeStateMachine should be started first.
func (s *serverPool) start() {
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
		wt, _ := s.calculateNode(n, false, false)
		if wt >= nodeWeightThreshold {
			s.ns.SetState(n, sfHasValue, nodestate.Flags{}, 0)
			s.ns.Persist(n)
		}
	})
	close(s.quit)
	s.persistClock()
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
	wt, wait := s.calculateNode(p.Node(), false, true)
	s.ns.SetField(p.Node(), sfiConnectedStats, nil)
	s.ns.SetState(p.Node(), sfRedialWait, sfConnected, wait)
	if wt >= nodeWeightThreshold {
		s.ns.SetState(p.Node(), sfHasValue, nodestate.Flags{}, 0)
		s.ns.Persist(p.Node())
	}
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

// calculateNode calculates the selection weight and the proposed redial wait time of the given node
func (s *serverPool) calculateNode(node *enode.Node, failedConnection, remoteDisconnect bool) (uint64, time.Duration) {
	n, _ := s.ns.GetField(node, sfiNodeHistory).(nodeHistory)
	nvt := s.vt.GetNode(node.ID())
	if nvt == nil {
		return 0, 0
	}
	currentStats := nvt.RtStats()
	timeout := s.getTimeout()
	s.timeoutLock.RLock()
	timeWeights := s.timeWeights
	s.timeoutLock.RUnlock()
	expFactor := s.vt.StatsExpFactor()

	var sessionValue float64
	if remoteDisconnect {
		if connStats, ok := s.ns.GetField(node, sfiConnectedStats).(lpc.ResponseTimeStats); ok {
			diff := currentStats
			diff.SubStats(&connStats)
			sessionValue = diff.Value(timeWeights, expFactor)
			sessionValueMeter.Mark(int64(sessionValue))
		}
	}

	logOffset := s.vt.StatsExpirer().LogOffset(s.clock.Now())
	if failedConnection || remoteDisconnect {
		n.dialCost.Add(dialCost, logOffset)
	}
	totalDialCost := n.dialCost.Value(logOffset)
	if totalDialCost < dialCost {
		totalDialCost = dialCost
	}

	var storeField bool
	if remoteDisconnect || timeout < n.lastTimeout-timeoutChangeThreshold || timeout > n.lastTimeout+timeoutChangeThreshold {
		n.totalValue = currentStats.Value(timeWeights, expFactor)
		n.lastTimeout = timeout
		storeField = true
	}

	var wait time.Duration
	if failedConnection || remoteDisconnect {
		a := n.totalValue * dialCost
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
		wait = time.Duration(float64(minRedialWait) * n.waitFactor)
		n.waitUntil = s.clock.Now() + s.clockOffset + mclock.AbsTime(wait)
		storeField = true
	}

	if storeField {
		s.ns.SetField(node, sfiNodeHistory, n)
	}
	return uint64(n.totalValue * nodeWeightMul / float64(totalDialCost)), wait
}

// knownSelectWeight is the selection weight callback function. It also takes care of
// removing nodes from the valuable set if their value has been expired.
func (s *serverPool) knownSelectWeight(i interface{}) uint64 {
	n := s.ns.GetNode(i.(enode.ID))
	if n == nil {
		return 0
	}
	wt, _ := s.calculateNode(n, false, false)
	if wt < nodeWeightThreshold {
		s.ns.SetState(n, nodestate.Flags{}, sfHasValue, 0)
		s.ns.Persist(n)
		return 0
	}
	return wt
}
