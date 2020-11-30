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

package utils

import (
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

// Limiter protects a network request serving mechanism from denial-of-service attacks.
// It limits the total amount of resources used for serving requests while ensuring that
// the most valuable connections always have a reasonable chance of being served.
type Limiter struct {
	lock sync.Mutex
	cond *sync.Cond
	quit bool

	nodes                                map[enode.ID]*nodeQueue
	addresses                            map[string]*addressGroup
	addressSelect, valueSelect           *WeightedRandomSelect
	maxValue                             float64
	maxWeight, sumWeight, sumWeightLimit uint
	selectedByValue                      bool
}

// nodeQueue represents queued requests coming from a single node ID
type nodeQueue struct {
	queue                   []request
	id                      enode.ID
	address                 string
	value                   float64
	flatWeight, valueWeight uint64 // current selection weights in the address/value selectors
	nextWeight              uint   // next request weight
	sumWeight               uint   // summed weight of requests queued by the node
	groupIndex              int
}

// addressGroup is a group of node IDs that have sent their last requests from the same
// network address
type addressGroup struct {
	nodes                      []*nodeQueue
	nodeSelect                 *WeightedRandomSelect
	sumFlatWeight, groupWeight uint64
}

// request represents an incoming request scheduled for processing
type request struct {
	process chan chan struct{}
	weight  uint
}

// flatWeight distributes weights equally between each active network adress
func flatWeight(item interface{}) uint64 { return item.(*nodeQueue).flatWeight }

// add adds the node queue to the address group. It is the caller's responsibility to
// add the address group to the address map and the address selector if it wasn't
// there before.
func (ag *addressGroup) add(nq *nodeQueue) {
	if nq.groupIndex != -1 {
		panic("added node queue is already in an address group")
	}
	l := len(ag.nodes)
	if l == 1 {
		ag.nodeSelect = NewWeightedRandomSelect(flatWeight)
		ag.nodeSelect.Update(ag.nodes[0])
	}
	nq.groupIndex = l
	ag.nodes = append(ag.nodes, nq)
	ag.sumFlatWeight += nq.flatWeight
	ag.groupWeight = ag.sumFlatWeight / uint64(l+1)
	if l >= 1 {
		ag.nodeSelect.Update(ag.nodes[l])
	}
}

// update updates the selection weight of the node queue inside the address group.
// It is the caller's responsibility to update the group's selection weight in the
// address selector.
func (ag *addressGroup) update(nq *nodeQueue, weight uint64) {
	if nq.groupIndex == -1 || nq.groupIndex >= len(ag.nodes) || ag.nodes[nq.groupIndex] != nq {
		panic("updated node queue is not in this address group")
	}
	ag.sumFlatWeight += weight - nq.flatWeight
	nq.flatWeight = weight
	ag.groupWeight = ag.sumFlatWeight / uint64(len(ag.nodes))
	if ag.nodeSelect != nil {
		ag.nodeSelect.Update(nq)
	}
}

// remove removes the node queue from the address group. It is the caller's responsibility
// to remove the address group from the address map if it is empty.
func (ag *addressGroup) remove(nq *nodeQueue) {
	if nq.groupIndex == -1 || nq.groupIndex >= len(ag.nodes) || ag.nodes[nq.groupIndex] != nq {
		panic("removed node queue is not in this address group")
	}

	l := len(ag.nodes) - 1
	if nq.groupIndex != l {
		ag.nodes[nq.groupIndex] = ag.nodes[l]
		ag.nodes[nq.groupIndex].groupIndex = nq.groupIndex
	}
	nq.groupIndex = -1
	ag.nodes = ag.nodes[:l]
	ag.sumFlatWeight -= nq.flatWeight
	if l >= 1 {
		ag.groupWeight = ag.sumFlatWeight / uint64(l)
		if l == 1 {
			ag.nodeSelect = nil
		} else {
			ag.nodeSelect.Remove(nq)
		}
	} else {
		ag.groupWeight = 0
	}
}

// choose selects one of the node queues belonging to the address group
func (ag *addressGroup) choose() *nodeQueue {
	if ag.nodeSelect == nil { // nodes list should never be empty here
		return ag.nodes[0]
	}
	return ag.nodeSelect.Choose().(*nodeQueue)
}

// NewLimiter creates a new Limiter
func NewLimiter(sumWeightLimit uint) *Limiter {
	l := &Limiter{
		addressSelect:  NewWeightedRandomSelect(func(item interface{}) uint64 { return item.(*addressGroup).groupWeight }),
		valueSelect:    NewWeightedRandomSelect(func(item interface{}) uint64 { return item.(*nodeQueue).valueWeight }),
		nodes:          make(map[enode.ID]*nodeQueue),
		addresses:      make(map[string]*addressGroup),
		sumWeightLimit: sumWeightLimit,
	}
	l.cond = sync.NewCond(&l.lock)
	go l.processLoop()
	return l
}

// selectionWeights calculates the selection weights of a node for both the address and
// the value selector. The selection weight depends on the next request weight or the
// summed weights of recently dropped requests. relWeight is reqWeight/maxWeight.
func selectionWeights(relWeight, value float64) (flatWeight, valueWeight uint64) {
	var f float64
	if relWeight <= 0.001 {
		f = 1
	} else {
		f = 0.001 / relWeight
	}
	f *= 1000000000
	flatWeight, valueWeight = uint64(f), uint64(f*value)
	if flatWeight == 0 {
		flatWeight = 1
	}
	return
}

// Add adds a new request to the node queue belonging to the given id. Value belongs
// to the requesting node. A higher value gives the request a higher chance of being
// served quickly in case of heavy load or a DDoS attack. weight is a rough estimate
// of the serving cost of the request. A lower weight also gives the request a
// better chance.
func (l *Limiter) Add(id enode.ID, address string, value float64, reqWeight uint) chan chan struct{} {
	l.lock.Lock()
	defer l.lock.Unlock()

	process := make(chan chan struct{}, 1)
	if l.quit {
		close(process)
		return process
	}
	if value > l.maxValue {
		l.maxValue = value
	}
	if value > 0 {
		// normalize value to <= 1
		value /= l.maxValue
	}
	if reqWeight == 0 {
		reqWeight = 1
	}
	if reqWeight > l.maxWeight {
		l.maxWeight = reqWeight
	}

	if nq, ok := l.nodes[id]; ok {
		nq.queue = append(nq.queue, request{process, reqWeight})
		nq.sumWeight += reqWeight
		nq.value = value
		if address != nq.address {
			// known id sending request from a new address, move to different address group
			l.removeFromGroup(nq)
			l.addToGroup(nq, address)
		}
	} else {
		nq := &nodeQueue{
			queue:      []request{{process, reqWeight}},
			id:         id,
			value:      value,
			nextWeight: reqWeight,
			sumWeight:  reqWeight,
			groupIndex: -1,
		}
		nq.flatWeight, nq.valueWeight = selectionWeights(float64(reqWeight)/float64(l.maxWeight), value)
		if len(l.nodes) == 0 {
			l.cond.Signal()
		}
		l.nodes[id] = nq
		if nq.valueWeight != 0 {
			l.valueSelect.Update(nq)
		}
		l.addToGroup(nq, address)
	}
	l.sumWeight += reqWeight
	if l.sumWeight > l.sumWeightLimit {
		l.dropRequests()
	}
	return process
}

// update updates the selection weights of the node queue
func (l *Limiter) update(nq *nodeQueue) {
	flatWeight, valueWeight := selectionWeights(float64(nq.nextWeight)/float64(l.maxWeight), nq.value)
	ag := l.addresses[nq.address]
	ag.update(nq, flatWeight)
	l.addressSelect.Update(ag)
	if valueWeight != 0 {
		nq.valueWeight = valueWeight
		l.valueSelect.Update(nq)
	}
}

// addToGroup adds the node queue to the given address group. The group is created if
// it does not exist yet.
func (l *Limiter) addToGroup(nq *nodeQueue, address string) {
	nq.address = address
	ag := l.addresses[address]
	if ag == nil {
		ag = &addressGroup{}
		l.addresses[address] = ag
	}
	ag.add(nq)
	l.addressSelect.Update(ag)
}

// removeFromGroup removes the node queue from its address group
func (l *Limiter) removeFromGroup(nq *nodeQueue) {
	ag := l.addresses[nq.address]
	ag.remove(nq)
	if len(ag.nodes) == 0 {
		delete(l.addresses, nq.address)
	}
	l.addressSelect.Update(ag)
}

// remove removes the node queue from its address group, the nodes map and the value
// selector
func (l *Limiter) remove(nq *nodeQueue) {
	l.removeFromGroup(nq)
	if nq.valueWeight != 0 {
		l.valueSelect.Remove(nq)
	}
	delete(l.nodes, nq.id)
}

// choose selects the next node queue to process.
// Note: when a node queue becomes empty it stays in the random selectors for one more
// selection round before removed, with a weight based on the last relative cost.
// If no more requests are added to the queue before it is selected again then the
// queue is removed and the next time a request comes from the same node the queue is
// added with the highest possible weight.
func (l *Limiter) choose() *nodeQueue {
	if l.valueSelect.IsEmpty() || l.selectedByValue {
		if ag, ok := l.addressSelect.Choose().(*addressGroup); ok {
			l.selectedByValue = false
			return ag.choose()
		}
	}
	nq, _ := l.valueSelect.Choose().(*nodeQueue)
	l.selectedByValue = true
	return nq
}

// processLoop processes requests sequentially
func (l *Limiter) processLoop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	for {
		if l.quit {
			for _, nq := range l.nodes {
				for _, request := range nq.queue {
					close(request.process)
				}
			}
			return
		}
		nq := l.choose()
		if nq == nil {
			l.cond.Wait()
			continue
		}
		if len(nq.queue) > 0 {
			request := nq.queue[0]
			nq.queue = nq.queue[1:]
			nq.sumWeight -= request.weight
			l.sumWeight -= request.weight
			l.lock.Unlock()
			ch := make(chan struct{})
			request.process <- ch
			<-ch
			l.lock.Lock()
			if len(nq.queue) > 0 {
				nq.nextWeight = nq.queue[0].weight
				l.update(nq)
			} else {
				l.remove(nq)
			}
		} else {
			l.remove(nq)
		}
	}
}

// Stop stops the processing loop. All queued and future requests are rejected.
func (l *Limiter) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.quit = true
	l.cond.Signal()
}

type (
	dropList     []dropListItem
	dropListItem struct {
		nq       *nodeQueue
		priority float64
	}
)

func (l dropList) Len() int {
	return len(l)
}

func (l dropList) Less(i, j int) bool {
	return l[i].priority < l[j].priority
}

func (l dropList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// dropRequests selects the nodes with the highest queued request weight to selection
// weight ratio and drops their queued request. The empty node queues stay in the
// selectors with a low selection weight in order to penalize these nodes.
func (l *Limiter) dropRequests() {
	var (
		sumValue float64
		list     dropList
	)
	for _, nq := range l.nodes {
		sumValue += nq.value
	}
	for _, nq := range l.nodes {
		if nq.sumWeight == 0 {
			continue
		}
		w := 1 / float64(len(l.addresses)*len(l.addresses[nq.address].nodes))
		if sumValue > 0 {
			w += nq.value / sumValue
		}
		list = append(list, dropListItem{
			nq:       nq,
			priority: w / float64(nq.sumWeight),
		})
	}
	sort.Sort(list)
	for _, item := range list {
		for _, request := range item.nq.queue {
			close(request.process)
		}
		l.sumWeight -= item.nq.sumWeight
		// set the last relative cost as if all dropped requests were processed with
		// the highest possible cost (which equals their summed weight).
		item.nq.nextWeight = item.nq.sumWeight
		item.nq.sumWeight = 0
		l.update(item.nq)
		item.nq.queue = nil
		if l.sumWeight <= l.sumWeightLimit/2 {
			return
		}
	}
}
