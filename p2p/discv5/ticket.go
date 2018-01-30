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
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

const (
	ticketTimeBucketLen = time.Minute
	timeWindow          = 10 // * ticketTimeBucketLen
	wantTicketsInWindow = 10
	collectFrequency    = time.Second * 30
	registerFrequency   = time.Second * 60
	maxCollectDebt      = 10
	maxRegisterDebt     = 5
	keepTicketConst     = time.Minute * 10
	keepTicketExp       = time.Minute * 5
	targetWaitTime      = time.Minute * 10
	topicQueryTimeout   = time.Second * 5
	topicQueryResend    = time.Minute
	// topic radius detection
	maxRadius           = 0xffffffffffffffff
	radiusTC            = time.Minute * 20
	radiusBucketsPerBit = 8
	minSlope            = 1
	minPeakSize         = 40
	maxNoAdjust         = 20
	lookupWidth         = 8
	minRightSum         = 20
	searchForceQuery    = 4
)

// timeBucket represents absolute monotonic time in minutes.
// It is used as the index into the per-topic ticket buckets.
type timeBucket int

type ticket struct {
	topics  []Topic
	regTime []mclock.AbsTime // Per-topic local absolute time when the ticket can be used.

	// The serial number that was issued by the server.
	serial uint32
	// Used by registrar, tracks absolute time when the ticket was created.
	issueTime mclock.AbsTime

	// Fields used only by registrants
	node   *Node  // the registrar node that signed this ticket
	refCnt int    // tracks number of topics that will be registered using this ticket
	pong   []byte // encoded pong packet signed by the registrar
}

// ticketRef refers to a single topic in a ticket.
type ticketRef struct {
	t   *ticket
	idx int // index of the topic in t.topics and t.regTime
}

func (ref ticketRef) topic() Topic {
	return ref.t.topics[ref.idx]
}

func (ref ticketRef) topicRegTime() mclock.AbsTime {
	return ref.t.regTime[ref.idx]
}

func pongToTicket(localTime mclock.AbsTime, topics []Topic, node *Node, p *ingressPacket) (*ticket, error) {
	wps := p.data.(*pong).WaitPeriods
	if len(topics) != len(wps) {
		return nil, fmt.Errorf("bad wait period list: got %d values, want %d", len(topics), len(wps))
	}
	if rlpHash(topics) != p.data.(*pong).TopicHash {
		return nil, fmt.Errorf("bad topic hash")
	}
	t := &ticket{
		issueTime: localTime,
		node:      node,
		topics:    topics,
		pong:      p.rawData,
		regTime:   make([]mclock.AbsTime, len(wps)),
	}
	// Convert wait periods to local absolute time.
	for i, wp := range wps {
		t.regTime[i] = localTime + mclock.AbsTime(time.Second*time.Duration(wp))
	}
	return t, nil
}

func ticketToPong(t *ticket, pong *pong) {
	pong.Expiration = uint64(t.issueTime / mclock.AbsTime(time.Second))
	pong.TopicHash = rlpHash(t.topics)
	pong.TicketSerial = t.serial
	pong.WaitPeriods = make([]uint32, len(t.regTime))
	for i, regTime := range t.regTime {
		pong.WaitPeriods[i] = uint32(time.Duration(regTime-t.issueTime) / time.Second)
	}
}

type ticketStore struct {
	// radius detector and target address generator
	// exists for both searched and registered topics
	radius map[Topic]*topicRadius

	// Contains buckets (for each absolute minute) of tickets
	// that can be used in that minute.
	// This is only set if the topic is being registered.
	tickets map[Topic]*topicTickets

	regQueue []Topic            // Topic registration queue for round robin attempts
	regSet   map[Topic]struct{} // Topic registration queue contents for fast filling

	nodes       map[*Node]*ticket
	nodeLastReq map[*Node]reqInfo

	lastBucketFetched timeBucket
	nextTicketCached  *ticketRef
	nextTicketReg     mclock.AbsTime

	searchTopicMap        map[Topic]searchTopic
	nextTopicQueryCleanup mclock.AbsTime
	queriesSent           map[*Node]map[common.Hash]sentQuery
}

type searchTopic struct {
	foundChn chan<- *Node
}

type sentQuery struct {
	sent   mclock.AbsTime
	lookup lookupInfo
}

type topicTickets struct {
	buckets    map[timeBucket][]ticketRef
	nextLookup mclock.AbsTime
	nextReg    mclock.AbsTime
}

func newTicketStore() *ticketStore {
	return &ticketStore{
		radius:         make(map[Topic]*topicRadius),
		tickets:        make(map[Topic]*topicTickets),
		regSet:         make(map[Topic]struct{}),
		nodes:          make(map[*Node]*ticket),
		nodeLastReq:    make(map[*Node]reqInfo),
		searchTopicMap: make(map[Topic]searchTopic),
		queriesSent:    make(map[*Node]map[common.Hash]sentQuery),
	}
}

// addTopic starts tracking a topic. If register is true,
// the local node will register the topic and tickets will be collected.
func (s *ticketStore) addTopic(topic Topic, register bool) {
	log.Trace("Adding discovery topic", "topic", topic, "register", register)
	if s.radius[topic] == nil {
		s.radius[topic] = newTopicRadius(topic)
	}
	if register && s.tickets[topic] == nil {
		s.tickets[topic] = &topicTickets{buckets: make(map[timeBucket][]ticketRef)}
	}
}

func (s *ticketStore) addSearchTopic(t Topic, foundChn chan<- *Node) {
	s.addTopic(t, false)
	if s.searchTopicMap[t].foundChn == nil {
		s.searchTopicMap[t] = searchTopic{foundChn: foundChn}
	}
}

func (s *ticketStore) removeSearchTopic(t Topic) {
	if st := s.searchTopicMap[t]; st.foundChn != nil {
		delete(s.searchTopicMap, t)
	}
}

// removeRegisterTopic deletes all tickets for the given topic.
func (s *ticketStore) removeRegisterTopic(topic Topic) {
	log.Trace("Removing discovery topic", "topic", topic)
	if s.tickets[topic] == nil {
		log.Warn("Removing non-existent discovery topic", "topic", topic)
		return
	}
	for _, list := range s.tickets[topic].buckets {
		for _, ref := range list {
			ref.t.refCnt--
			if ref.t.refCnt == 0 {
				delete(s.nodes, ref.t.node)
				delete(s.nodeLastReq, ref.t.node)
			}
		}
	}
	delete(s.tickets, topic)
}

func (s *ticketStore) regTopicSet() []Topic {
	topics := make([]Topic, 0, len(s.tickets))
	for topic := range s.tickets {
		topics = append(topics, topic)
	}
	return topics
}

// nextRegisterLookup returns the target of the next lookup for ticket collection.
func (s *ticketStore) nextRegisterLookup() (lookupInfo, time.Duration) {
	// Queue up any new topics (or discarded ones), preserving iteration order
	for topic := range s.tickets {
		if _, ok := s.regSet[topic]; !ok {
			s.regQueue = append(s.regQueue, topic)
			s.regSet[topic] = struct{}{}
		}
	}
	// Iterate over the set of all topics and look up the next suitable one
	for len(s.regQueue) > 0 {
		// Fetch the next topic from the queue, and ensure it still exists
		topic := s.regQueue[0]
		s.regQueue = s.regQueue[1:]
		delete(s.regSet, topic)

		if s.tickets[topic] == nil {
			continue
		}
		// If the topic needs more tickets, return it
		if s.tickets[topic].nextLookup < mclock.Now() {
			next, delay := s.radius[topic].nextTarget(false), 100*time.Millisecond
			log.Trace("Found discovery topic to register", "topic", topic, "target", next.target, "delay", delay)
			return next, delay
		}
	}
	// No registration topics found or all exhausted, sleep
	delay := 40 * time.Second
	log.Trace("No topic found to register", "delay", delay)
	return lookupInfo{}, delay
}

func (s *ticketStore) nextSearchLookup(topic Topic) lookupInfo {
	tr := s.radius[topic]
	target := tr.nextTarget(tr.radiusLookupCnt >= searchForceQuery)
	if target.radiusLookup {
		tr.radiusLookupCnt++
	} else {
		tr.radiusLookupCnt = 0
	}
	return target
}

// ticketsInWindow returns the tickets of a given topic in the registration window.
func (s *ticketStore) ticketsInWindow(topic Topic) []ticketRef {
	// Sanity check that the topic still exists before operating on it
	if s.tickets[topic] == nil {
		log.Warn("Listing non-existing discovery tickets", "topic", topic)
		return nil
	}
	// Gather all the tickers in the next time window
	var tickets []ticketRef

	buckets := s.tickets[topic].buckets
	for idx := timeBucket(0); idx < timeWindow; idx++ {
		tickets = append(tickets, buckets[s.lastBucketFetched+idx]...)
	}
	log.Trace("Retrieved discovery registration tickets", "topic", topic, "from", s.lastBucketFetched, "tickets", len(tickets))
	return tickets
}

func (s *ticketStore) removeExcessTickets(t Topic) {
	tickets := s.ticketsInWindow(t)
	if len(tickets) <= wantTicketsInWindow {
		return
	}
	sort.Sort(ticketRefByWaitTime(tickets))
	for _, r := range tickets[wantTicketsInWindow:] {
		s.removeTicketRef(r)
	}
}

type ticketRefByWaitTime []ticketRef

// Len is the number of elements in the collection.
func (s ticketRefByWaitTime) Len() int {
	return len(s)
}

func (r ticketRef) waitTime() mclock.AbsTime {
	return r.t.regTime[r.idx] - r.t.issueTime
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s ticketRefByWaitTime) Less(i, j int) bool {
	return s[i].waitTime() < s[j].waitTime()
}

// Swap swaps the elements with indexes i and j.
func (s ticketRefByWaitTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s *ticketStore) addTicketRef(r ticketRef) {
	topic := r.t.topics[r.idx]
	tickets := s.tickets[topic]
	if tickets == nil {
		log.Warn("Adding ticket to non-existent topic", "topic", topic)
		return
	}
	bucket := timeBucket(r.t.regTime[r.idx] / mclock.AbsTime(ticketTimeBucketLen))
	tickets.buckets[bucket] = append(tickets.buckets[bucket], r)
	r.t.refCnt++

	min := mclock.Now() - mclock.AbsTime(collectFrequency)*maxCollectDebt
	if tickets.nextLookup < min {
		tickets.nextLookup = min
	}
	tickets.nextLookup += mclock.AbsTime(collectFrequency)

	//s.removeExcessTickets(topic)
}

func (s *ticketStore) nextFilteredTicket() (*ticketRef, time.Duration) {
	now := mclock.Now()
	for {
		ticket, wait := s.nextRegisterableTicket()
		if ticket == nil {
			return ticket, wait
		}
		log.Trace("Found discovery ticket to register", "node", ticket.t.node, "serial", ticket.t.serial, "wait", wait)

		regTime := now + mclock.AbsTime(wait)
		topic := ticket.t.topics[ticket.idx]
		if s.tickets[topic] != nil && regTime >= s.tickets[topic].nextReg {
			return ticket, wait
		}
		s.removeTicketRef(*ticket)
	}
}

func (s *ticketStore) ticketRegistered(ref ticketRef) {
	now := mclock.Now()

	topic := ref.t.topics[ref.idx]
	tickets := s.tickets[topic]
	min := now - mclock.AbsTime(registerFrequency)*maxRegisterDebt
	if min > tickets.nextReg {
		tickets.nextReg = min
	}
	tickets.nextReg += mclock.AbsTime(registerFrequency)
	s.tickets[topic] = tickets

	s.removeTicketRef(ref)
}

// nextRegisterableTicket returns the next ticket that can be used
// to register.
//
// If the returned wait time <= zero the ticket can be used. For a positive
// wait time, the caller should requery the next ticket later.
//
// A ticket can be returned more than once with <= zero wait time in case
// the ticket contains multiple topics.
func (s *ticketStore) nextRegisterableTicket() (*ticketRef, time.Duration) {
	now := mclock.Now()
	if s.nextTicketCached != nil {
		return s.nextTicketCached, time.Duration(s.nextTicketCached.topicRegTime() - now)
	}

	for bucket := s.lastBucketFetched; ; bucket++ {
		var (
			empty      = true    // true if there are no tickets
			nextTicket ticketRef // uninitialized if this bucket is empty
		)
		for _, tickets := range s.tickets {
			//s.removeExcessTickets(topic)
			if len(tickets.buckets) != 0 {
				empty = false

				list := tickets.buckets[bucket]
				for _, ref := range list {
					//debugLog(fmt.Sprintf(" nrt bucket = %d node = %x sn = %v wait = %v", bucket, ref.t.node.ID[:8], ref.t.serial, time.Duration(ref.topicRegTime()-now)))
					if nextTicket.t == nil || ref.topicRegTime() < nextTicket.topicRegTime() {
						nextTicket = ref
					}
				}
			}
		}
		if empty {
			return nil, 0
		}
		if nextTicket.t != nil {
			s.nextTicketCached = &nextTicket
			return &nextTicket, time.Duration(nextTicket.topicRegTime() - now)
		}
		s.lastBucketFetched = bucket
	}
}

// removeTicket removes a ticket from the ticket store
func (s *ticketStore) removeTicketRef(ref ticketRef) {
	log.Trace("Removing discovery ticket reference", "node", ref.t.node.ID, "serial", ref.t.serial)

	// Make nextRegisterableTicket return the next available ticket.
	s.nextTicketCached = nil

	topic := ref.topic()
	tickets := s.tickets[topic]

	if tickets == nil {
		log.Trace("Removing tickets from unknown topic", "topic", topic)
		return
	}
	bucket := timeBucket(ref.t.regTime[ref.idx] / mclock.AbsTime(ticketTimeBucketLen))
	list := tickets.buckets[bucket]
	idx := -1
	for i, bt := range list {
		if bt.t == ref.t {
			idx = i
			break
		}
	}
	if idx == -1 {
		panic(nil)
	}
	list = append(list[:idx], list[idx+1:]...)
	if len(list) != 0 {
		tickets.buckets[bucket] = list
	} else {
		delete(tickets.buckets, bucket)
	}
	ref.t.refCnt--
	if ref.t.refCnt == 0 {
		delete(s.nodes, ref.t.node)
		delete(s.nodeLastReq, ref.t.node)
	}
}

type lookupInfo struct {
	target       common.Hash
	topic        Topic
	radiusLookup bool
}

type reqInfo struct {
	pingHash []byte
	lookup   lookupInfo
	time     mclock.AbsTime
}

// returns -1 if not found
func (t *ticket) findIdx(topic Topic) int {
	for i, tt := range t.topics {
		if tt == topic {
			return i
		}
	}
	return -1
}

func (s *ticketStore) registerLookupDone(lookup lookupInfo, nodes []*Node, ping func(n *Node) []byte) {
	now := mclock.Now()
	for i, n := range nodes {
		if i == 0 || (binary.BigEndian.Uint64(n.sha[:8])^binary.BigEndian.Uint64(lookup.target[:8])) < s.radius[lookup.topic].minRadius {
			if lookup.radiusLookup {
				if lastReq, ok := s.nodeLastReq[n]; !ok || time.Duration(now-lastReq.time) > radiusTC {
					s.nodeLastReq[n] = reqInfo{pingHash: ping(n), lookup: lookup, time: now}
				}
			} else {
				if s.nodes[n] == nil {
					s.nodeLastReq[n] = reqInfo{pingHash: ping(n), lookup: lookup, time: now}
				}
			}
		}
	}
}

func (s *ticketStore) searchLookupDone(lookup lookupInfo, nodes []*Node, ping func(n *Node) []byte, query func(n *Node, topic Topic) []byte) {
	now := mclock.Now()
	for i, n := range nodes {
		if i == 0 || (binary.BigEndian.Uint64(n.sha[:8])^binary.BigEndian.Uint64(lookup.target[:8])) < s.radius[lookup.topic].minRadius {
			if lookup.radiusLookup {
				if lastReq, ok := s.nodeLastReq[n]; !ok || time.Duration(now-lastReq.time) > radiusTC {
					s.nodeLastReq[n] = reqInfo{pingHash: ping(n), lookup: lookup, time: now}
				}
			} // else {
			if s.canQueryTopic(n, lookup.topic) {
				hash := query(n, lookup.topic)
				if hash != nil {
					s.addTopicQuery(common.BytesToHash(hash), n, lookup)
				}
			}
			//}
		}
	}
}

func (s *ticketStore) adjustWithTicket(now mclock.AbsTime, targetHash common.Hash, t *ticket) {
	for i, topic := range t.topics {
		if tt, ok := s.radius[topic]; ok {
			tt.adjustWithTicket(now, targetHash, ticketRef{t, i})
		}
	}
}

func (s *ticketStore) addTicket(localTime mclock.AbsTime, pingHash []byte, ticket *ticket) {
	log.Trace("Adding discovery ticket", "node", ticket.node.ID, "serial", ticket.serial)

	lastReq, ok := s.nodeLastReq[ticket.node]
	if !(ok && bytes.Equal(pingHash, lastReq.pingHash)) {
		return
	}
	s.adjustWithTicket(localTime, lastReq.lookup.target, ticket)

	if lastReq.lookup.radiusLookup || s.nodes[ticket.node] != nil {
		return
	}

	topic := lastReq.lookup.topic
	topicIdx := ticket.findIdx(topic)
	if topicIdx == -1 {
		return
	}

	bucket := timeBucket(localTime / mclock.AbsTime(ticketTimeBucketLen))
	if s.lastBucketFetched == 0 || bucket < s.lastBucketFetched {
		s.lastBucketFetched = bucket
	}

	if _, ok := s.tickets[topic]; ok {
		wait := ticket.regTime[topicIdx] - localTime
		rnd := rand.ExpFloat64()
		if rnd > 10 {
			rnd = 10
		}
		if float64(wait) < float64(keepTicketConst)+float64(keepTicketExp)*rnd {
			// use the ticket to register this topic
			//fmt.Println("addTicket", ticket.node.ID[:8], ticket.node.addr().String(), ticket.serial, ticket.pong)
			s.addTicketRef(ticketRef{ticket, topicIdx})
		}
	}

	if ticket.refCnt > 0 {
		s.nextTicketCached = nil
		s.nodes[ticket.node] = ticket
	}
}

func (s *ticketStore) getNodeTicket(node *Node) *ticket {
	if s.nodes[node] == nil {
		log.Trace("Retrieving node ticket", "node", node.ID, "serial", nil)
	} else {
		log.Trace("Retrieving node ticket", "node", node.ID, "serial", s.nodes[node].serial)
	}
	return s.nodes[node]
}

func (s *ticketStore) canQueryTopic(node *Node, topic Topic) bool {
	qq := s.queriesSent[node]
	if qq != nil {
		now := mclock.Now()
		for _, sq := range qq {
			if sq.lookup.topic == topic && sq.sent > now-mclock.AbsTime(topicQueryResend) {
				return false
			}
		}
	}
	return true
}

func (s *ticketStore) addTopicQuery(hash common.Hash, node *Node, lookup lookupInfo) {
	now := mclock.Now()
	qq := s.queriesSent[node]
	if qq == nil {
		qq = make(map[common.Hash]sentQuery)
		s.queriesSent[node] = qq
	}
	qq[hash] = sentQuery{sent: now, lookup: lookup}
	s.cleanupTopicQueries(now)
}

func (s *ticketStore) cleanupTopicQueries(now mclock.AbsTime) {
	if s.nextTopicQueryCleanup > now {
		return
	}
	exp := now - mclock.AbsTime(topicQueryResend)
	for n, qq := range s.queriesSent {
		for h, q := range qq {
			if q.sent < exp {
				delete(qq, h)
			}
		}
		if len(qq) == 0 {
			delete(s.queriesSent, n)
		}
	}
	s.nextTopicQueryCleanup = now + mclock.AbsTime(topicQueryTimeout)
}

func (s *ticketStore) gotTopicNodes(from *Node, hash common.Hash, nodes []rpcNode) (timeout bool) {
	now := mclock.Now()
	//fmt.Println("got", from.addr().String(), hash, len(nodes))
	qq := s.queriesSent[from]
	if qq == nil {
		return true
	}
	q, ok := qq[hash]
	if !ok || now > q.sent+mclock.AbsTime(topicQueryTimeout) {
		return true
	}
	inside := float64(0)
	if len(nodes) > 0 {
		inside = 1
	}
	s.radius[q.lookup.topic].adjust(now, q.lookup.target, from.sha, inside)
	chn := s.searchTopicMap[q.lookup.topic].foundChn
	if chn == nil {
		//fmt.Println("no channel")
		return false
	}
	for _, node := range nodes {
		ip := node.IP
		if ip.IsUnspecified() || ip.IsLoopback() {
			ip = from.IP
		}
		n := NewNode(node.ID, ip, node.UDP, node.TCP)
		select {
		case chn <- n:
		default:
			return false
		}
	}
	return false
}

type topicRadius struct {
	topic             Topic
	topicHashPrefix   uint64
	radius, minRadius uint64
	buckets           []topicRadiusBucket
	converged         bool
	radiusLookupCnt   int
}

type topicRadiusEvent int

const (
	trOutside topicRadiusEvent = iota
	trInside
	trNoAdjust
	trCount
)

type topicRadiusBucket struct {
	weights    [trCount]float64
	lastTime   mclock.AbsTime
	value      float64
	lookupSent map[common.Hash]mclock.AbsTime
}

func (b *topicRadiusBucket) update(now mclock.AbsTime) {
	if now == b.lastTime {
		return
	}
	exp := math.Exp(-float64(now-b.lastTime) / float64(radiusTC))
	for i, w := range b.weights {
		b.weights[i] = w * exp
	}
	b.lastTime = now

	for target, tm := range b.lookupSent {
		if now-tm > mclock.AbsTime(respTimeout) {
			b.weights[trNoAdjust] += 1
			delete(b.lookupSent, target)
		}
	}
}

func (b *topicRadiusBucket) adjust(now mclock.AbsTime, inside float64) {
	b.update(now)
	if inside <= 0 {
		b.weights[trOutside] += 1
	} else {
		if inside >= 1 {
			b.weights[trInside] += 1
		} else {
			b.weights[trInside] += inside
			b.weights[trOutside] += 1 - inside
		}
	}
}

func newTopicRadius(t Topic) *topicRadius {
	topicHash := crypto.Keccak256Hash([]byte(t))
	topicHashPrefix := binary.BigEndian.Uint64(topicHash[0:8])

	return &topicRadius{
		topic:           t,
		topicHashPrefix: topicHashPrefix,
		radius:          maxRadius,
		minRadius:       maxRadius,
	}
}

func (r *topicRadius) getBucketIdx(addrHash common.Hash) int {
	prefix := binary.BigEndian.Uint64(addrHash[0:8])
	var log2 float64
	if prefix != r.topicHashPrefix {
		log2 = math.Log2(float64(prefix ^ r.topicHashPrefix))
	}
	bucket := int((64 - log2) * radiusBucketsPerBit)
	max := 64*radiusBucketsPerBit - 1
	if bucket > max {
		return max
	}
	if bucket < 0 {
		return 0
	}
	return bucket
}

func (r *topicRadius) targetForBucket(bucket int) common.Hash {
	min := math.Pow(2, 64-float64(bucket+1)/radiusBucketsPerBit)
	max := math.Pow(2, 64-float64(bucket)/radiusBucketsPerBit)
	a := uint64(min)
	b := randUint64n(uint64(max - min))
	xor := a + b
	if xor < a {
		xor = ^uint64(0)
	}
	prefix := r.topicHashPrefix ^ xor
	var target common.Hash
	binary.BigEndian.PutUint64(target[0:8], prefix)
	globalRandRead(target[8:])
	return target
}

// package rand provides a Read function in Go 1.6 and later, but
// we can't use it yet because we still support Go 1.5.
func globalRandRead(b []byte) {
	pos := 0
	val := 0
	for n := 0; n < len(b); n++ {
		if pos == 0 {
			val = rand.Int()
			pos = 7
		}
		b[n] = byte(val)
		val >>= 8
		pos--
	}
}

func (r *topicRadius) isInRadius(addrHash common.Hash) bool {
	nodePrefix := binary.BigEndian.Uint64(addrHash[0:8])
	dist := nodePrefix ^ r.topicHashPrefix
	return dist < r.radius
}

func (r *topicRadius) chooseLookupBucket(a, b int) int {
	if a < 0 {
		a = 0
	}
	if a > b {
		return -1
	}
	c := 0
	for i := a; i <= b; i++ {
		if i >= len(r.buckets) || r.buckets[i].weights[trNoAdjust] < maxNoAdjust {
			c++
		}
	}
	if c == 0 {
		return -1
	}
	rnd := randUint(uint32(c))
	for i := a; i <= b; i++ {
		if i >= len(r.buckets) || r.buckets[i].weights[trNoAdjust] < maxNoAdjust {
			if rnd == 0 {
				return i
			}
			rnd--
		}
	}
	panic(nil) // should never happen
}

func (r *topicRadius) needMoreLookups(a, b int, maxValue float64) bool {
	var max float64
	if a < 0 {
		a = 0
	}
	if b >= len(r.buckets) {
		b = len(r.buckets) - 1
		if r.buckets[b].value > max {
			max = r.buckets[b].value
		}
	}
	if b >= a {
		for i := a; i <= b; i++ {
			if r.buckets[i].value > max {
				max = r.buckets[i].value
			}
		}
	}
	return maxValue-max < minPeakSize
}

func (r *topicRadius) recalcRadius() (radius uint64, radiusLookup int) {
	maxBucket := 0
	maxValue := float64(0)
	now := mclock.Now()
	v := float64(0)
	for i := range r.buckets {
		r.buckets[i].update(now)
		v += r.buckets[i].weights[trOutside] - r.buckets[i].weights[trInside]
		r.buckets[i].value = v
		//fmt.Printf("%v %v | ", v, r.buckets[i].weights[trNoAdjust])
	}
	//fmt.Println()
	slopeCross := -1
	for i, b := range r.buckets {
		v := b.value
		if v < float64(i)*minSlope {
			slopeCross = i
			break
		}
		if v > maxValue {
			maxValue = v
			maxBucket = i + 1
		}
	}

	minRadBucket := len(r.buckets)
	sum := float64(0)
	for minRadBucket > 0 && sum < minRightSum {
		minRadBucket--
		b := r.buckets[minRadBucket]
		sum += b.weights[trInside] + b.weights[trOutside]
	}
	r.minRadius = uint64(math.Pow(2, 64-float64(minRadBucket)/radiusBucketsPerBit))

	lookupLeft := -1
	if r.needMoreLookups(0, maxBucket-lookupWidth-1, maxValue) {
		lookupLeft = r.chooseLookupBucket(maxBucket-lookupWidth, maxBucket-1)
	}
	lookupRight := -1
	if slopeCross != maxBucket && (minRadBucket <= maxBucket || r.needMoreLookups(maxBucket+lookupWidth, len(r.buckets)-1, maxValue)) {
		for len(r.buckets) <= maxBucket+lookupWidth {
			r.buckets = append(r.buckets, topicRadiusBucket{lookupSent: make(map[common.Hash]mclock.AbsTime)})
		}
		lookupRight = r.chooseLookupBucket(maxBucket, maxBucket+lookupWidth-1)
	}
	if lookupLeft == -1 {
		radiusLookup = lookupRight
	} else {
		if lookupRight == -1 {
			radiusLookup = lookupLeft
		} else {
			if randUint(2) == 0 {
				radiusLookup = lookupLeft
			} else {
				radiusLookup = lookupRight
			}
		}
	}

	//fmt.Println("mb", maxBucket, "sc", slopeCross, "mrb", minRadBucket, "ll", lookupLeft, "lr", lookupRight, "mv", maxValue)

	if radiusLookup == -1 {
		// no more radius lookups needed at the moment, return a radius
		r.converged = true
		rad := maxBucket
		if minRadBucket < rad {
			rad = minRadBucket
		}
		radius = ^uint64(0)
		if rad > 0 {
			radius = uint64(math.Pow(2, 64-float64(rad)/radiusBucketsPerBit))
		}
		r.radius = radius
	}

	return
}

func (r *topicRadius) nextTarget(forceRegular bool) lookupInfo {
	if !forceRegular {
		_, radiusLookup := r.recalcRadius()
		if radiusLookup != -1 {
			target := r.targetForBucket(radiusLookup)
			r.buckets[radiusLookup].lookupSent[target] = mclock.Now()
			return lookupInfo{target: target, topic: r.topic, radiusLookup: true}
		}
	}

	radExt := r.radius / 2
	if radExt > maxRadius-r.radius {
		radExt = maxRadius - r.radius
	}
	rnd := randUint64n(r.radius) + randUint64n(2*radExt)
	if rnd > radExt {
		rnd -= radExt
	} else {
		rnd = radExt - rnd
	}

	prefix := r.topicHashPrefix ^ rnd
	var target common.Hash
	binary.BigEndian.PutUint64(target[0:8], prefix)
	globalRandRead(target[8:])
	return lookupInfo{target: target, topic: r.topic, radiusLookup: false}
}

func (r *topicRadius) adjustWithTicket(now mclock.AbsTime, targetHash common.Hash, t ticketRef) {
	wait := t.t.regTime[t.idx] - t.t.issueTime
	inside := float64(wait)/float64(targetWaitTime) - 0.5
	if inside > 1 {
		inside = 1
	}
	if inside < 0 {
		inside = 0
	}
	r.adjust(now, targetHash, t.t.node.sha, inside)
}

func (r *topicRadius) adjust(now mclock.AbsTime, targetHash, addrHash common.Hash, inside float64) {
	bucket := r.getBucketIdx(addrHash)
	//fmt.Println("adjust", bucket, len(r.buckets), inside)
	if bucket >= len(r.buckets) {
		return
	}
	r.buckets[bucket].adjust(now, inside)
	delete(r.buckets[bucket].lookupSent, targetHash)
}
