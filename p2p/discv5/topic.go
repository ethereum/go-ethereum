// Copyright 2016 The go-ethereum Authors
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

package discv5

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/maticnetwork/bor/common/mclock"
	"github.com/maticnetwork/bor/log"
)

const (
	maxEntries         = 10000
	maxEntriesPerTopic = 50

	fallbackRegistrationExpiry = 1 * time.Hour
)

type Topic string

type topicEntry struct {
	topic   Topic
	fifoIdx uint64
	node    *Node
	expire  mclock.AbsTime
}

type topicInfo struct {
	entries            map[uint64]*topicEntry
	fifoHead, fifoTail uint64
	rqItem             *topicRequestQueueItem
	wcl                waitControlLoop
}

// removes tail element from the fifo
func (t *topicInfo) getFifoTail() *topicEntry {
	for t.entries[t.fifoTail] == nil {
		t.fifoTail++
	}
	tail := t.entries[t.fifoTail]
	t.fifoTail++
	return tail
}

type nodeInfo struct {
	entries                          map[Topic]*topicEntry
	lastIssuedTicket, lastUsedTicket uint32
	// you can't register a ticket newer than lastUsedTicket before noRegUntil (absolute time)
	noRegUntil mclock.AbsTime
}

type topicTable struct {
	db                    *nodeDB
	self                  *Node
	nodes                 map[*Node]*nodeInfo
	topics                map[Topic]*topicInfo
	globalEntries         uint64
	requested             topicRequestQueue
	requestCnt            uint64
	lastGarbageCollection mclock.AbsTime
}

func newTopicTable(db *nodeDB, self *Node) *topicTable {
	if printTestImgLogs {
		fmt.Printf("*N %016x\n", self.sha[:8])
	}
	return &topicTable{
		db:     db,
		nodes:  make(map[*Node]*nodeInfo),
		topics: make(map[Topic]*topicInfo),
		self:   self,
	}
}

func (t *topicTable) getOrNewTopic(topic Topic) *topicInfo {
	ti := t.topics[topic]
	if ti == nil {
		rqItem := &topicRequestQueueItem{
			topic:    topic,
			priority: t.requestCnt,
		}
		ti = &topicInfo{
			entries: make(map[uint64]*topicEntry),
			rqItem:  rqItem,
		}
		t.topics[topic] = ti
		heap.Push(&t.requested, rqItem)
	}
	return ti
}

func (t *topicTable) checkDeleteTopic(topic Topic) {
	ti := t.topics[topic]
	if ti == nil {
		return
	}
	if len(ti.entries) == 0 && ti.wcl.hasMinimumWaitPeriod() {
		delete(t.topics, topic)
		heap.Remove(&t.requested, ti.rqItem.index)
	}
}

func (t *topicTable) getOrNewNode(node *Node) *nodeInfo {
	n := t.nodes[node]
	if n == nil {
		//fmt.Printf("newNode %016x %016x\n", t.self.sha[:8], node.sha[:8])
		var issued, used uint32
		if t.db != nil {
			issued, used = t.db.fetchTopicRegTickets(node.ID)
		}
		n = &nodeInfo{
			entries:          make(map[Topic]*topicEntry),
			lastIssuedTicket: issued,
			lastUsedTicket:   used,
		}
		t.nodes[node] = n
	}
	return n
}

func (t *topicTable) checkDeleteNode(node *Node) {
	if n, ok := t.nodes[node]; ok && len(n.entries) == 0 && n.noRegUntil < mclock.Now() {
		//fmt.Printf("deleteNode %016x %016x\n", t.self.sha[:8], node.sha[:8])
		delete(t.nodes, node)
	}
}

func (t *topicTable) storeTicketCounters(node *Node) {
	n := t.getOrNewNode(node)
	if t.db != nil {
		t.db.updateTopicRegTickets(node.ID, n.lastIssuedTicket, n.lastUsedTicket)
	}
}

func (t *topicTable) getEntries(topic Topic) []*Node {
	t.collectGarbage()

	te := t.topics[topic]
	if te == nil {
		return nil
	}
	nodes := make([]*Node, len(te.entries))
	i := 0
	for _, e := range te.entries {
		nodes[i] = e.node
		i++
	}
	t.requestCnt++
	t.requested.update(te.rqItem, t.requestCnt)
	return nodes
}

func (t *topicTable) addEntry(node *Node, topic Topic) {
	n := t.getOrNewNode(node)
	// clear previous entries by the same node
	for _, e := range n.entries {
		t.deleteEntry(e)
	}
	// ***
	n = t.getOrNewNode(node)

	tm := mclock.Now()
	te := t.getOrNewTopic(topic)

	if len(te.entries) == maxEntriesPerTopic {
		t.deleteEntry(te.getFifoTail())
	}

	if t.globalEntries == maxEntries {
		t.deleteEntry(t.leastRequested()) // not empty, no need to check for nil
	}

	fifoIdx := te.fifoHead
	te.fifoHead++
	entry := &topicEntry{
		topic:   topic,
		fifoIdx: fifoIdx,
		node:    node,
		expire:  tm + mclock.AbsTime(fallbackRegistrationExpiry),
	}
	if printTestImgLogs {
		fmt.Printf("*+ %d %v %016x %016x\n", tm/1000000, topic, t.self.sha[:8], node.sha[:8])
	}
	te.entries[fifoIdx] = entry
	n.entries[topic] = entry
	t.globalEntries++
	te.wcl.registered(tm)
}

// removes least requested element from the fifo
func (t *topicTable) leastRequested() *topicEntry {
	for t.requested.Len() > 0 && t.topics[t.requested[0].topic] == nil {
		heap.Pop(&t.requested)
	}
	if t.requested.Len() == 0 {
		return nil
	}
	return t.topics[t.requested[0].topic].getFifoTail()
}

// entry should exist
func (t *topicTable) deleteEntry(e *topicEntry) {
	if printTestImgLogs {
		fmt.Printf("*- %d %v %016x %016x\n", mclock.Now()/1000000, e.topic, t.self.sha[:8], e.node.sha[:8])
	}
	ne := t.nodes[e.node].entries
	delete(ne, e.topic)
	if len(ne) == 0 {
		t.checkDeleteNode(e.node)
	}
	te := t.topics[e.topic]
	delete(te.entries, e.fifoIdx)
	if len(te.entries) == 0 {
		t.checkDeleteTopic(e.topic)
	}
	t.globalEntries--
}

// It is assumed that topics and waitPeriods have the same length.
func (t *topicTable) useTicket(node *Node, serialNo uint32, topics []Topic, idx int, issueTime uint64, waitPeriods []uint32) (registered bool) {
	log.Trace("Using discovery ticket", "serial", serialNo, "topics", topics, "waits", waitPeriods)
	//fmt.Println("useTicket", serialNo, topics, waitPeriods)
	t.collectGarbage()

	n := t.getOrNewNode(node)
	if serialNo < n.lastUsedTicket {
		return false
	}

	tm := mclock.Now()
	if serialNo > n.lastUsedTicket && tm < n.noRegUntil {
		return false
	}
	if serialNo != n.lastUsedTicket {
		n.lastUsedTicket = serialNo
		n.noRegUntil = tm + mclock.AbsTime(noRegTimeout())
		t.storeTicketCounters(node)
	}

	currTime := uint64(tm / mclock.AbsTime(time.Second))
	regTime := issueTime + uint64(waitPeriods[idx])
	relTime := int64(currTime - regTime)
	if relTime >= -1 && relTime <= regTimeWindow+1 { // give clients a little security margin on both ends
		if e := n.entries[topics[idx]]; e == nil {
			t.addEntry(node, topics[idx])
		} else {
			// if there is an active entry, don't move to the front of the FIFO but prolong expire time
			e.expire = tm + mclock.AbsTime(fallbackRegistrationExpiry)
		}
		return true
	}

	return false
}

func (t *topicTable) getTicket(node *Node, topics []Topic) *ticket {
	t.collectGarbage()

	now := mclock.Now()
	n := t.getOrNewNode(node)
	n.lastIssuedTicket++
	t.storeTicketCounters(node)

	tic := &ticket{
		issueTime: now,
		topics:    topics,
		serial:    n.lastIssuedTicket,
		regTime:   make([]mclock.AbsTime, len(topics)),
	}
	for i, topic := range topics {
		var waitPeriod time.Duration
		if topic := t.topics[topic]; topic != nil {
			waitPeriod = topic.wcl.waitPeriod
		} else {
			waitPeriod = minWaitPeriod
		}

		tic.regTime[i] = now + mclock.AbsTime(waitPeriod)
	}
	return tic
}

const gcInterval = time.Minute

func (t *topicTable) collectGarbage() {
	tm := mclock.Now()
	if time.Duration(tm-t.lastGarbageCollection) < gcInterval {
		return
	}
	t.lastGarbageCollection = tm

	for node, n := range t.nodes {
		for _, e := range n.entries {
			if e.expire <= tm {
				t.deleteEntry(e)
			}
		}

		t.checkDeleteNode(node)
	}

	for topic := range t.topics {
		t.checkDeleteTopic(topic)
	}
}

const (
	minWaitPeriod   = time.Minute
	regTimeWindow   = 10 // seconds
	avgnoRegTimeout = time.Minute * 10
	// target average interval between two incoming ad requests
	wcTargetRegInterval = time.Minute * 10 / maxEntriesPerTopic
	//
	wcTimeConst = time.Minute * 10
)

// initialization is not required, will set to minWaitPeriod at first registration
type waitControlLoop struct {
	lastIncoming mclock.AbsTime
	waitPeriod   time.Duration
}

func (w *waitControlLoop) registered(tm mclock.AbsTime) {
	w.waitPeriod = w.nextWaitPeriod(tm)
	w.lastIncoming = tm
}

func (w *waitControlLoop) nextWaitPeriod(tm mclock.AbsTime) time.Duration {
	period := tm - w.lastIncoming
	wp := time.Duration(float64(w.waitPeriod) * math.Exp((float64(wcTargetRegInterval)-float64(period))/float64(wcTimeConst)))
	if wp < minWaitPeriod {
		wp = minWaitPeriod
	}
	return wp
}

func (w *waitControlLoop) hasMinimumWaitPeriod() bool {
	return w.nextWaitPeriod(mclock.Now()) == minWaitPeriod
}

func noRegTimeout() time.Duration {
	e := rand.ExpFloat64()
	if e > 100 {
		e = 100
	}
	return time.Duration(float64(avgnoRegTimeout) * e)
}

type topicRequestQueueItem struct {
	topic    Topic
	priority uint64
	index    int
}

// A topicRequestQueue implements heap.Interface and holds topicRequestQueueItems.
type topicRequestQueue []*topicRequestQueueItem

func (tq topicRequestQueue) Len() int { return len(tq) }

func (tq topicRequestQueue) Less(i, j int) bool {
	return tq[i].priority < tq[j].priority
}

func (tq topicRequestQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
	tq[i].index = i
	tq[j].index = j
}

func (tq *topicRequestQueue) Push(x interface{}) {
	n := len(*tq)
	item := x.(*topicRequestQueueItem)
	item.index = n
	*tq = append(*tq, item)
}

func (tq *topicRequestQueue) Pop() interface{} {
	old := *tq
	n := len(old)
	item := old[n-1]
	item.index = -1
	*tq = old[0 : n-1]
	return item
}

func (tq *topicRequestQueue) update(item *topicRequestQueueItem, priority uint64) {
	item.priority = priority
	heap.Fix(tq, item.index)
}
