package identify

import (
	"sync"
	"time"

	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

const ActivationThresh = 4

// ObservedAddr is an entry for an address reported by our peers.
// We only use addresses that:
// - have been observed at least 4 times in last 1h. (counter symmetric nats)
// - have been observed at least once recently (1h), because our position in the
//   network, or network port mapppings, may have changed.
type ObservedAddr struct {
	Addr      ma.Multiaddr
	SeenBy    map[string]time.Time
	LastSeen  time.Time
	Activated bool
}

func (oa *ObservedAddr) TryActivate(ttl time.Duration) bool {
	// cleanup SeenBy set
	now := time.Now()
	for k, t := range oa.SeenBy {
		if now.Sub(t) > ttl*ActivationThresh {
			delete(oa.SeenBy, k)
		}
	}

	// We only activate if in the TTL other peers observed the same address
	// of ours at least 4 times.
	return len(oa.SeenBy) >= ActivationThresh
}

// ObservedAddrSet keeps track of a set of ObservedAddrs
// the zero-value is ready to be used.
type ObservedAddrSet struct {
	sync.Mutex // guards whole datastruct.

	addrs map[string]*ObservedAddr
	ttl   time.Duration
}

func (oas *ObservedAddrSet) Addrs() []ma.Multiaddr {
	oas.Lock()
	defer oas.Unlock()

	// for zero-value.
	if oas.addrs == nil {
		return nil
	}

	now := time.Now()
	addrs := make([]ma.Multiaddr, 0, len(oas.addrs))
	for s, a := range oas.addrs {
		// remove timed out addresses.
		if now.Sub(a.LastSeen) > oas.ttl {
			delete(oas.addrs, s)
			continue
		}

		if a.Activated || a.TryActivate(oas.ttl) {
			addrs = append(addrs, a.Addr)
		}
	}
	return addrs
}

func (oas *ObservedAddrSet) Add(addr ma.Multiaddr, observer ma.Multiaddr) {
	oas.Lock()
	defer oas.Unlock()

	// for zero-value.
	if oas.addrs == nil {
		oas.addrs = make(map[string]*ObservedAddr)
		oas.ttl = pstore.OwnObservedAddrTTL
	}

	s := addr.String()
	oa, found := oas.addrs[s]

	// first time seeing address.
	if !found {
		oa = &ObservedAddr{
			Addr:   addr,
			SeenBy: make(map[string]time.Time),
		}
		oas.addrs[s] = oa
	}

	// mark the observer
	oa.SeenBy[observerGroup(observer)] = time.Now()
	oa.LastSeen = time.Now()
}

// observerGroup is a function that determines what part of
// a multiaddr counts as a different observer. for example,
// two ipfs nodes at the same IP/TCP transport would get
// the exact same NAT mapping; they would count as the
// same observer. This may protect against NATs who assign
// different ports to addresses at different IP hosts, but
// not TCP ports.
//
// Here, we use the root multiaddr address. This is mostly
// IP addresses. In practice, this is what we want.
func observerGroup(m ma.Multiaddr) string {
	//TODO: If IPv6 rolls out we should mark /64 routing zones as one group
	return ma.Split(m)[0].String()
}

func (oas *ObservedAddrSet) SetTTL(ttl time.Duration) {
	oas.Lock()
	defer oas.Unlock()
	oas.ttl = ttl
}

func (oas *ObservedAddrSet) TTL() time.Duration {
	oas.Lock()
	defer oas.Unlock()
	// for zero-value.
	if oas.addrs == nil {
		oas.ttl = pstore.OwnObservedAddrTTL
	}
	return oas.ttl
}
