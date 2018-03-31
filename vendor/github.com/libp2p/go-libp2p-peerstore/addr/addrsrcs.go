// Package addr provides utility functions to handle peer addresses.
package addr

import (
	ma "github.com/multiformats/go-multiaddr"
)

// AddrSource is a source of addresses. It allows clients to retrieve
// a set of addresses at a last possible moment in time. It is used
// to query a set of addresses that may change over time, as a result
// of the network changing interfaces or mappings.
type Source interface {
	Addrs() []ma.Multiaddr
}

// CombineSources returns a new AddrSource which is the
// concatenation of all input AddrSources:
//
//   combined := CombinedSources(a, b)
//   combined.Addrs() // append(a.Addrs(), b.Addrs()...)
//
func CombineSources(srcs ...Source) Source {
	return combinedAS(srcs)
}

type combinedAS []Source

func (cas combinedAS) Addrs() []ma.Multiaddr {
	var addrs []ma.Multiaddr
	for _, s := range cas {
		addrs = append(addrs, s.Addrs()...)
	}
	return addrs
}

// UniqueSource returns a new AddrSource which omits duplicate
// addresses from the inputs:
//
//   unique := UniqueSource(a, b)
//   unique.Addrs() // append(a.Addrs(), b.Addrs()...)
//                  // but only adds each addr once.
//
func UniqueSource(srcs ...Source) Source {
	return uniqueAS(srcs)
}

type uniqueAS []Source

func (uas uniqueAS) Addrs() []ma.Multiaddr {
	seen := make(map[string]struct{})
	var addrs []ma.Multiaddr
	for _, s := range uas {
		for _, a := range s.Addrs() {
			s := a.String()
			if _, found := seen[s]; !found {
				addrs = append(addrs, a)
				seen[s] = struct{}{}
			}
		}
	}
	return addrs
}

// Slice is a simple slice of addresses that implements
// the AddrSource interface.
type Slice []ma.Multiaddr

func (as Slice) Addrs() []ma.Multiaddr {
	return as
}
