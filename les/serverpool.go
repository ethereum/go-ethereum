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
	lpc "github.com/ethereum/go-ethereum/les/lespay/client"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	minTimeout             = time.Millisecond * 500 // minimum request timeout suggested by the server pool
	timeoutRefresh         = time.Second * 5        // recalculate timeout if older than this
	timeoutChangeThreshold = time.Millisecond * 10  // recalculate node values if timeout has changed more than this amount
	dialCost               = 10000                  // cost of a TCP dial (used for known node selection weight calculation)
	nodeWeightMul          = 1000000                // multiplier constant for node weight calculation
	nodeWeightThreshold    = 100                    // minimum weight for keeping a node in the the known (valuable) set
)

// serverPool provides a node iterator for dial candidates. The output is a mix of newly discovered
// nodes, a weighted random selection of known (previously valuable) nodes and trusted/paid nodes.
type serverPool struct {
	clock                               mclock.Clock
	ns                                  *utils.NodeStateMachine
	vt                                  *lpc.ValueTracker
	mixer                               *enode.FairMix
	mixSources                          []enode.Iterator
	dialIterator                        enode.Iterator
	stDialed, stConnected, stRedialWait utils.NodeStateBitMask
	stHasValue, stAlwaysConnect         utils.NodeStateBitMask
	enrFieldId, nodeHistoryFieldId      int
	trusted                             []enode.ID

	timeoutLock      sync.RWMutex
	timeout          time.Duration
	timeWeights      lpc.ResponseTimeWeights
	timeoutRefreshed mclock.AbsTime
}

// nodeHistory keeps track of dial costs which determine node weight together with the
// service value calculated by lpc.ValueTracker.
type nodeHistory struct {
	// only dialCost is saved
	lock        sync.Mutex
	dialCost    utils.ExpiredValue
	lastTimeout time.Duration
	totalValue  float64
}

var (
	sfDiscovered    = utils.NewNodeStateFlag("discovered", false, false)
	sfHasValue      = utils.NewNodeStateFlag("hasValue", true, false)
	sfSelected      = utils.NewNodeStateFlag("selected", false, false)
	sfDialed        = utils.NewNodeStateFlag("dialed", false, false)
	sfConnected     = utils.NewNodeStateFlag("connected", false, false)
	sfRedialWait    = utils.NewNodeStateFlag("redialWait", false, true)
	sfAlwaysConnect = utils.NewNodeStateFlag("alwaysConnect", false, false)

	keepNodeRecord   = []*utils.NodeStateFlag{sfDiscovered, sfHasValue, sfAlwaysConnect}
	disableSelection = []*utils.NodeStateFlag{sfSelected, sfDialed, sfConnected, sfRedialWait}

	errInvalidField = errors.New("invalid field type")

	sfiEnr = utils.NewNodeStateField("enr", reflect.TypeOf(&enr.Record{}), keepNodeRecord,
		func(field interface{}) ([]byte, error) {
			if e, ok := field.(*enr.Record); ok {
				enc, err := rlp.EncodeToBytes(e)
				return enc, err
			} else {
				return nil, errInvalidField
			}
		},
		func(enc []byte) (interface{}, error) {
			e := &enr.Record{}
			err := rlp.DecodeBytes(enc, e)
			return e, err
		},
	)
	sfiNodeHistory = utils.NewNodeStateField("nodeHistory", reflect.TypeOf(&nodeHistory{}), []*utils.NodeStateFlag{sfDiscovered, sfHasValue},
		func(field interface{}) ([]byte, error) {
			if n, ok := field.(*nodeHistory); ok {
				enc, err := rlp.EncodeToBytes(&n.dialCost)
				return enc, err
			} else {
				return nil, errInvalidField
			}
		},
		func(enc []byte) (interface{}, error) {
			n := &nodeHistory{}
			err := rlp.DecodeBytes(enc, &n.dialCost)
			return n, err
		},
	)
)

// newServerPool creates a new server pool
func newServerPool(ns *utils.NodeStateMachine, vt *lpc.ValueTracker, discovery enode.Iterator, clock mclock.Clock, trustedURLs []string, testing bool) *serverPool {
	s := &serverPool{
		clock: clock,
		ns:    ns,
		vt:    vt,
	}
	s.getTimeout()
	// Register all serverpool-defined states
	stDiscovered := s.ns.MustRegisterState(sfDiscovered)
	s.stHasValue = s.ns.MustRegisterState(sfHasValue)
	s.stDialed = s.ns.MustRegisterState(sfDialed)
	s.stConnected = s.ns.MustRegisterState(sfConnected)
	s.stRedialWait = s.ns.MustRegisterState(sfRedialWait)
	s.stAlwaysConnect = s.ns.MustRegisterState(sfAlwaysConnect)
	s.ns.MustRegisterState(sfSelected)

	// Register all serverpool-defined node fields.
	s.enrFieldId = s.ns.MustRegisterField(sfiEnr)
	s.nodeHistoryFieldId = s.ns.MustRegisterField(sfiNodeHistory)

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
			s.trusted = append(s.trusted, node.ID())
		} else {
			log.Error("Invalid trusted server URL", "url", url, "error", err)
		}
	}

	s.mixer = enode.NewFairMix(mixerTimeout)
	knownSelector := lpc.NewWrsIterator(s.ns, s.stHasValue, s.ns.StatesMask(disableSelection), sfSelected, sfiEnr, s.knownSelectWeight, validSchemes)
	alwaysConnect := lpc.NewQueueIterator(s.ns, s.stAlwaysConnect, s.ns.StatesMask(disableSelection), sfSelected, sfiEnr, validSchemes)
	s.mixSources = append(s.mixSources, knownSelector)
	s.mixSources = append(s.mixSources, alwaysConnect)
	if discovery != nil {
		discEnrStored := enode.Filter(discovery, func(node *enode.Node) bool {
			s.ns.UpdateState(node.ID(), stDiscovered, 0, time.Hour)
			s.ns.SetField(node.ID(), s.enrFieldId, node.Record())
			return true
		})
		s.mixSources = append(s.mixSources, discEnrStored)
	}

	// preNegotiationFilter will be added in series with iter here when les4 is available

	s.dialIterator = enode.Filter(s.mixer, func(node *enode.Node) bool {
		n, _ := s.ns.GetField(node.ID(), s.nodeHistoryFieldId).(*nodeHistory)
		if n == nil {
			n = &nodeHistory{}
			s.ns.SetField(node.ID(), s.nodeHistoryFieldId, n)
		}
		n.lock.Lock()
		n.dialCost.Add(dialCost, s.vt.StatsExpirer().LogOffset(s.clock.Now()))
		n.lock.Unlock()
		s.ns.UpdateState(node.ID(), s.stDialed, 0, time.Second*10)
		return true
	})
	return s
}

// start starts the server pool. Note that NodeStateMachine should be started first.
func (s *serverPool) start() {
	for _, iter := range s.mixSources {
		// add sources to mixer at startup because the mixer instantly tries to read them
		// which should only happen after NodeStateMachine has been started
		s.mixer.AddSource(iter)
	}
	for _, id := range s.trusted {
		s.ns.UpdateState(id, s.stAlwaysConnect, 0, 0)
	}
}

// stop stops the server pool
func (s *serverPool) stop() {
	s.dialIterator.Close()
}

// registerPeer implements serverPeerSubscriber
func (s *serverPool) registerPeer(p *serverPeer) {
	s.ns.UpdateState(p.ID(), s.stConnected, s.stDialed, 0)
	p.setValueTracker(s.vt, s.vt.Register(p.ID()))
	p.updateVtParams()
}

// unregisterPeer implements serverPeerSubscriber
func (s *serverPool) unregisterPeer(p *serverPeer) {
	if s.nodeWeight(p.ID(), true) >= nodeWeightThreshold {
		s.ns.UpdateState(p.ID(), s.stHasValue, 0, 0)
	}
	s.ns.UpdateState(p.ID(), s.stRedialWait, s.stConnected, time.Second*10)
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

// nodeWeight calculates the selection weight of an individual node
func (s *serverPool) nodeWeight(id enode.ID, forceRecalc bool) uint64 {
	nn := s.ns.GetField(id, s.nodeHistoryFieldId)
	n, ok := nn.(*nodeHistory)
	if !ok {
		return 0
	}
	if n == nil {
		n = &nodeHistory{}
		s.ns.SetField(id, s.nodeHistoryFieldId, n)
	}
	nvt := s.vt.GetNode(id)
	if nvt == nil {
		return 0
	}
	div := n.dialCost.Value(s.vt.StatsExpirer().LogOffset(s.clock.Now()))
	if div < dialCost {
		div = dialCost
	}
	timeout := s.getTimeout()

	n.lock.Lock()
	defer n.lock.Unlock()

	if forceRecalc || timeout < n.lastTimeout-timeoutChangeThreshold || timeout > n.lastTimeout+timeoutChangeThreshold {
		s.timeoutLock.RLock()
		timeWeights := s.timeWeights
		s.timeoutLock.RUnlock()
		n.totalValue = s.vt.TotalServiceValue(nvt, timeWeights)
		n.lastTimeout = timeout
	}
	return uint64(n.totalValue * nodeWeightMul / float64(div))
}

// knownSelectWeight is the selection weight callback function. It also takes care of
// removing nodes from the valuable set if their value has been expired.
func (s *serverPool) knownSelectWeight(i interface{}) uint64 {
	id := i.(enode.ID)
	wt := s.nodeWeight(id, false)
	if wt < nodeWeightThreshold {
		go s.ns.UpdateState(id, 0, s.stHasValue, 0)
		return 0
	}
	return wt
}
