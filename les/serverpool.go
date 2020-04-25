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
	"fmt"
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
	nodeHistoryFieldId                  int
	trusted                             []*enode.Node

	timeoutLock      sync.RWMutex
	timeout          time.Duration
	timeWeights      lpc.ResponseTimeWeights
	timeoutRefreshed mclock.AbsTime
}

// nodeHistory keeps track of dial costs which determine node weight together with the
// service value calculated by lpc.ValueTracker.
type nodeHistory struct {
	// only dialCost is saved
	dialCost    utils.ExpiredValue
	lastTimeout time.Duration
	totalValue  float64
}

var (
	//sfDiscovered    = utils.NewFlag("discovered")
	sfHasValue      = utils.NewPersistentFlag("hasValue")
	sfSelected      = utils.NewFlag("selected")
	sfDialed        = utils.NewFlag("dialed")
	sfConnected     = utils.NewFlag("connected")
	sfRedialWait    = utils.NewFlag("redialWait")
	sfAlwaysConnect = utils.NewFlag("alwaysConnect")

	disableSelection = []*utils.NodeStateFlag{sfSelected, sfDialed, sfConnected, sfRedialWait}

	errInvalidField = errors.New("invalid field type")

	sfiNodeHistory = utils.NewPersistentField("nodeHistory", reflect.TypeOf(nodeHistory{}),
		func(field interface{}) ([]byte, error) {
			if n, ok := field.(nodeHistory); ok {
				enc, err := rlp.EncodeToBytes(&n.dialCost)
				fmt.Println("enc nh", err)
				return enc, err
			} else {
				return nil, errInvalidField
			}
		},
		func(enc []byte) (interface{}, error) {
			n := nodeHistory{}
			err := rlp.DecodeBytes(enc, &n.dialCost)
			fmt.Println("dec nh", err)
			return n, err
		},
	)

	serverPoolSetup = utils.NodeStateSetup{
		Flags:  []*utils.NodeStateFlag{ /*sfDiscovered, */ sfHasValue, sfSelected, sfDialed, sfConnected, sfRedialWait, sfAlwaysConnect},
		Fields: []*utils.NodeField{sfiNodeHistory},
	}
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
	//stDiscovered := s.ns.StateMask(sfDiscovered)
	s.stHasValue = s.ns.StateMask(sfHasValue)
	s.stDialed = s.ns.StateMask(sfDialed)
	s.stConnected = s.ns.StateMask(sfConnected)
	s.stRedialWait = s.ns.StateMask(sfRedialWait)
	s.stAlwaysConnect = s.ns.StateMask(sfAlwaysConnect)

	// Register all serverpool-defined node fields.
	s.nodeHistoryFieldId = s.ns.FieldIndex(sfiNodeHistory)

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
	knownSelector := lpc.NewWrsIterator(s.ns, s.stHasValue, s.ns.StatesMask(disableSelection), sfSelected, s.knownSelectWeight)
	alwaysConnect := lpc.NewQueueIterator(s.ns, s.stAlwaysConnect, s.ns.StatesMask(disableSelection), sfSelected)
	s.mixSources = append(s.mixSources, knownSelector)
	s.mixSources = append(s.mixSources, alwaysConnect)
	if discovery != nil {
		/*		discEnrStored := enode.Filter(discovery, func(node *enode.Node) bool {
					s.ns.SetState(node, stDiscovered, 0, time.Hour)
					return true
				})
				s.mixSources = append(s.mixSources, discEnrStored)*/
		s.mixSources = append(s.mixSources, discovery)
	}

	// preNegotiationFilter will be added in series with iter here when les4 is available

	s.dialIterator = enode.Filter(s.mixer, func(node *enode.Node) bool {
		s.ns.SetState(node, s.stDialed, 0, time.Second*10)
		n, _ := s.ns.GetField(node, s.nodeHistoryFieldId).(nodeHistory)
		n.dialCost.Add(dialCost, s.vt.StatsExpirer().LogOffset(s.clock.Now()))
		s.ns.SetField(node, s.nodeHistoryFieldId, n)
		return true
	})

	ns.AddLogMetrics(s.stHasValue, s.ns.StatesMask(disableSelection), "wrs", nil, nil, nil)
	//ns.AddLogMetrics(stDiscovered, 0, "discovered", nil, nil, nil)
	ns.AddLogMetrics(s.stDialed, 0, "dialed", nil, nil, nil)
	ns.AddLogMetrics(s.stConnected, 0, "connected", nil, nil, nil)
	ns.AddLogMetrics(s.stHasValue, 0, "hasValue", nil, nil, nil)
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
		s.ns.SetState(node, s.stAlwaysConnect, 0, 0)
	}
}

// stop stops the server pool
func (s *serverPool) stop() {
	s.dialIterator.Close()
	s.ns.ForEach(s.stConnected, 0, func(n *enode.Node, state utils.NodeStateBitMask) {
		if s.nodeWeight(n, true) >= nodeWeightThreshold {
			s.ns.SetState(n, s.stHasValue, 0, 0)
			s.ns.Persist(n)
		}
	})
}

// registerPeer implements serverPeerSubscriber
func (s *serverPool) registerPeer(p *serverPeer) {
	s.ns.SetState(p.Node(), s.stConnected, s.stDialed, 0)
	p.setValueTracker(s.vt, s.vt.Register(p.ID()))
	p.updateVtParams()
}

// unregisterPeer implements serverPeerSubscriber
func (s *serverPool) unregisterPeer(p *serverPeer) {
	if s.nodeWeight(p.Node(), true) >= nodeWeightThreshold {
		s.ns.SetState(p.Node(), s.stHasValue, 0, 0)
		s.ns.Persist(p.Node())
	}
	s.ns.SetState(p.Node(), s.stRedialWait, s.stConnected, time.Second*10)
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
func (s *serverPool) nodeWeight(node *enode.Node, forceRecalc bool) uint64 {
	fmt.Println("nodeWeight", node.ID())
	n, _ := s.ns.GetField(node, s.nodeHistoryFieldId).(nodeHistory)
	nvt := s.vt.GetNode(node.ID())
	if nvt == nil {
		fmt.Println("no vt entry")
		return 0
	}
	div := n.dialCost.Value(s.vt.StatsExpirer().LogOffset(s.clock.Now()))
	fmt.Println(" dialCost", div)
	if div < dialCost {
		div = dialCost
	}
	timeout := s.getTimeout()

	if forceRecalc || timeout < n.lastTimeout-timeoutChangeThreshold || timeout > n.lastTimeout+timeoutChangeThreshold {
		s.timeoutLock.RLock()
		timeWeights := s.timeWeights
		s.timeoutLock.RUnlock()
		n.totalValue = s.vt.TotalServiceValue(nvt, timeWeights)
		n.lastTimeout = timeout
		s.ns.SetField(node, s.nodeHistoryFieldId, n)
	}
	fmt.Println(" value", n.totalValue)
	fmt.Println(" weight", uint64(n.totalValue*nodeWeightMul/float64(div)))
	return uint64(n.totalValue * nodeWeightMul / float64(div))
}

// knownSelectWeight is the selection weight callback function. It also takes care of
// removing nodes from the valuable set if their value has been expired.
func (s *serverPool) knownSelectWeight(i interface{}) uint64 {
	n := s.ns.GetNode(i.(enode.ID))
	if n == nil {
		return 0
	}
	wt := s.nodeWeight(n, false)
	if wt < nodeWeightThreshold {
		go func() {
			s.ns.SetState(n, 0, s.stHasValue, 0)
			s.ns.Persist(n)
		}()
		return 0
	}
	return wt
}
