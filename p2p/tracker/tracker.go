// Copyright 2021 The go-ethereum Authors
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

package tracker

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	trackedGaugeName = "p2p/tracked"
	lostMeterName    = "p2p/lost"
	staleMeterName   = "p2p/stale"
	waitHistName     = "p2p/wait"

	// maxTrackedPackets is a huge number to act as a failsafe on the number of
	// pending requests the node will track. It should never be hit unless an
	// attacker figures out a way to spin requests.
	maxTrackedPackets = 10000
)

var (
	ErrNoMatchingRequest = errors.New("no matching request")
	ErrTooManyItems      = errors.New("response is larger than request allows")
	ErrCollision         = errors.New("request ID collision")
	ErrCodeMismatch      = errors.New("wrong response code for request")
	ErrLimitReached      = errors.New("request limit reached")
	ErrStopped           = errors.New("tracker stopped")
)

// Request tracks sent network requests which have not yet received a response.
type Request struct {
	ID       uint64 // Request ID
	Size     int    // Number/size of requested items
	ReqCode  uint64 // Protocol message code of the request
	RespCode uint64 // Protocol message code of the expected response

	time   time.Time     // Timestamp when the request was made
	expire *list.Element // Expiration marker to untrack it
}

type Response struct {
	ID      uint64 // Request ID of the response
	MsgCode uint64 // Protocol message code
	Size    int    // number/size of items in response
}

// Tracker is a pending network request tracker to measure how much time it takes
// a remote peer to respond.
type Tracker struct {
	cap p2p.Cap // Protocol capability identifier for the metrics

	peer    string        // Peer ID
	timeout time.Duration // Global timeout after which to drop a tracked packet

	pending map[uint64]*Request // Currently pending requests
	expire  *list.List          // Linked list tracking the expiration order
	wake    *time.Timer         // Timer tracking the expiration of the next item

	lock sync.Mutex // Lock protecting from concurrent updates
}

// New creates a new network request tracker to monitor how much time it takes to
// fill certain requests and how individual peers perform.
func New(cap p2p.Cap, peerID string, timeout time.Duration) *Tracker {
	return &Tracker{
		cap:     cap,
		peer:    peerID,
		timeout: timeout,
		pending: make(map[uint64]*Request),
		expire:  list.New(),
	}
}

// Track adds a network request to the tracker to wait for a response to arrive
// or until the request it cancelled or times out.
func (t *Tracker) Track(req Request) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.expire == nil {
		return ErrStopped
	}

	// If there's a duplicate request, we've just random-collided (or more probably,
	// we have a bug), report it. We could also add a metric, but we're not really
	// expecting ourselves to be buggy, so a noisy warning should be enough.
	if _, ok := t.pending[req.ID]; ok {
		log.Error("Network request id collision", "cap", t.cap, "code", req.ReqCode, "id", req.ID)
		return ErrCollision
	}
	// If we have too many pending requests, bail out instead of leaking memory
	if pending := len(t.pending); pending >= maxTrackedPackets {
		log.Error("Request tracker exceeded allowance", "pending", pending, "peer", t.peer, "cap", t.cap, "code", req.ReqCode)
		return ErrLimitReached
	}

	// Id doesn't exist yet, start tracking it
	req.time = time.Now()
	req.expire = t.expire.PushBack(req.ID)
	t.pending[req.ID] = &req

	if metrics.Enabled() {
		t.trackedGauge(req.ReqCode).Inc(1)
	}

	// If we've just inserted the first item, start the expiration timer
	if t.wake == nil {
		t.wake = time.AfterFunc(t.timeout, t.clean)
	}
	return nil
}

// clean is called automatically when a preset time passes without a response
// being delivered for the first network request.
func (t *Tracker) clean() {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Expire anything within a certain threshold (might be no items at all if
	// we raced with the delivery)
	for t.expire.Len() > 0 {
		// Stop iterating if the next pending request is still alive
		var (
			head = t.expire.Front()
			id   = head.Value.(uint64)
			req  = t.pending[id]
		)
		if time.Since(req.time) < t.timeout+5*time.Millisecond {
			break
		}
		// Nope, dead, drop it
		t.expire.Remove(head)
		delete(t.pending, id)

		if metrics.Enabled() {
			t.trackedGauge(req.ReqCode).Dec(1)
			t.lostMeter(req.ReqCode).Mark(1)
		}
	}
	t.schedule()
}

// schedule starts a timer to trigger on the expiration of the first network
// packet.
func (t *Tracker) schedule() {
	if t.expire.Len() == 0 {
		t.wake = nil
		return
	}
	t.wake = time.AfterFunc(time.Until(t.pending[t.expire.Front().Value.(uint64)].time.Add(t.timeout)), t.clean)
}

// Stop reclaims resources of the tracker.
func (t *Tracker) Stop() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.wake != nil {
		t.wake.Stop()
		t.wake = nil
	}
	if metrics.Enabled() {
		// Ensure metrics are decremented for pending requests.
		counts := make(map[uint64]int64)
		for _, req := range t.pending {
			counts[req.ReqCode]++
		}
		for code, count := range counts {
			t.trackedGauge(code).Dec(count)
		}
	}
	clear(t.pending)
	t.expire = nil
}

// Fulfil fills a pending request, if any is available, reporting on various metrics.
func (t *Tracker) Fulfil(resp Response) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// If it's a non existing request, track as stale response
	req, ok := t.pending[resp.ID]
	if !ok {
		if metrics.Enabled() {
			t.staleMeter(resp.MsgCode).Mark(1)
		}
		return ErrNoMatchingRequest
	}

	// If the response is funky, it might be some active attack
	if req.RespCode != resp.MsgCode {
		log.Warn("Network response code collision",
			"have", fmt.Sprintf("%s:%s/%d:%d", t.peer, t.cap.Name, t.cap.Version, resp.MsgCode),
			"want", fmt.Sprintf("%s:%s/%d:%d", t.peer, t.cap.Name, t.cap.Version, req.RespCode),
		)
		return ErrCodeMismatch
	}
	if resp.Size > req.Size {
		return ErrTooManyItems
	}

	// Everything matches, mark the request serviced and meter it
	wasHead := req.expire.Prev() == nil
	t.expire.Remove(req.expire)
	delete(t.pending, req.ID)
	if wasHead {
		if t.wake.Stop() {
			t.schedule()
		}
	}

	// Update request metrics.
	if metrics.Enabled() {
		t.trackedGauge(req.ReqCode).Dec(1)
		t.waitHistogram(req.ReqCode).Update(time.Since(req.time).Microseconds())
	}
	return nil
}

func (t *Tracker) trackedGauge(code uint64) *metrics.Gauge {
	name := fmt.Sprintf("%s/%s/%d/%#02x", trackedGaugeName, t.cap.Name, t.cap.Version, code)
	return metrics.GetOrRegisterGauge(name, nil)
}

func (t *Tracker) lostMeter(code uint64) *metrics.Meter {
	name := fmt.Sprintf("%s/%s/%d/%#02x", lostMeterName, t.cap.Name, t.cap.Version, code)
	return metrics.GetOrRegisterMeter(name, nil)
}

func (t *Tracker) staleMeter(code uint64) *metrics.Meter {
	name := fmt.Sprintf("%s/%s/%d/%#02x", staleMeterName, t.cap.Name, t.cap.Version, code)
	return metrics.GetOrRegisterMeter(name, nil)
}

func (t *Tracker) waitHistogram(code uint64) metrics.Histogram {
	name := fmt.Sprintf("%s/%s/%d/%#02x", waitHistName, t.cap.Name, t.cap.Version, code)
	sampler := func() metrics.Sample {
		return metrics.ResettingSample(metrics.NewExpDecaySample(1028, 0.015))
	}
	return metrics.GetOrRegisterHistogramLazy(name, nil, sampler)
}
