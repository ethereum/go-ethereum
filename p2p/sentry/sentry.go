// Package sentry is a peer idleness, latency and bandwidth tracker.
package sentry

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// measurementImpact is the impact a single measurement has on a peer's final
// throughput value. This number ensures that we scale up (or down) the speed
// smoothly even if individual packets are jittery.
const measurementImpact = 0.1

// Sentry is a high level monitor to estimate the round trip time and serving
// bandwidth a peer has for specific network requests (e.g. eth GetNodeData).
// It's particularly useful for sizing requests when downloading concurrently
// from multiple peers, so that it can maximize throughput across all peers
// without overloading any of them.
//
// The reason it's useful to track these metrics per-type is because different
// implementations will have different serving times added to the raw network
// delays; furthermore, certain implementations might add various QoS limits
// or altogether not serve certain entities.
//
// The interesting aspect is that by tracking and constantly resizing requests
// according to measured capacities, the sentry doesn't only hone in on a single
// peer's capabilities, but will also globally tune them to our own resources.
//
// Of course, measuring the latency and bandwidth based on response times is semi-
// impossible due to the interplay between the two; but the goal is not accuracy,
// rather to have a dynamically updating number that prevents overloading a peer.
//
// Note, implementation wise maps keyed by type strings aren't the fastest, but
// given that the updates are bound by network requests in general, the mutex
// and hashing speed should be irrelevant compared to the wire wait times.
type Sentry struct {
	log log.Logger // allow per-peer messages to debug issues

	// pend tracks whether the peer has a pending network request for a certain
	// type or does not; and if so, when the request was issued.
	pend map[string]time.Time

	// rtt tracks the round trip time a peer takes to serve a certain request. It
	// is useful to track it per-packet as serving disk latency may make it wildly
	// different for different data types.
	rtt map[string]time.Duration

	// bandwidth tracks the number of bytes a peer is capable of returning per
	// second of a certain data type. A simpler approach would be to track the
	// number of data items per second, but that's not obvious for all types and
	// it also behaves badly on varying sizes.
	bandwidth map[string]float64

	// Protects the above maps from concurrent updates. Mostly the `pend` is the
	// high contention map, updating the others dwarfs in comparison with the
	// requred networking time.
	lock sync.Mutex
}

// New returns a new packet sentry to be used to track a single peer.
func New(log log.Logger) *Sentry {
	return &Sentry{
		log:       log,
		pend:      make(map[string]time.Time),
		rtt:       make(map[string]time.Duration),
		bandwidth: make(map[string]float64),
	}
}

// Reserve checks if the peer is idle wrt a specific packet type and if so,
// starts a pending request. The method doesn't do anything other than tell
// the caller it can do the request. It is the responsibility of the caller
// to update the peer when the response arrives.
func (s *Sentry) Reserve(kind string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.pend[kind]; ok {
		return false
	}
	s.pend[kind] = time.Now()
	return true
}

// Update relinquishes a previous reservation and updates the internal metrics
// with the values measured in this run.
func (s *Sentry) Update(kind string, elapsed time.Duration, delivered uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// If nothing was delivered (hard timeout / unavailable data), reduce throughput to minimum
	if delivered == 0 {
		s.bandwidth[kind] = 0
		return
	}
	// Otherwise update the throughput with a new measurement
	if elapsed <= 0 {
		elapsed = 1 // +1 (ns) to ensure non-zero divisor
	}
	measured := float64(delivered) / (float64(elapsed) / float64(time.Second))

	s.bandwidth[kind] = (1-measurementImpact)*(s.bandwidth[kind]) + measurementImpact*measured
	s.rtt[kind] = time.Duration((1-measurementImpact)*float64(s.rtt[kind]) + measurementImpact*float64(elapsed))

	s.log.Trace("Peer throughput measurements updated", "type", kind,
		"rtt", s.rtt[kind], "bandwidth", s.bandwidth[kind])
}
