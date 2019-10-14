// Copyright 2019 The go-ethereum Authors
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

package dnsdisc

import (
	"context"
	"crypto/ecdsa"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// clientTree is a full tree being synced.
type clientTree struct {
	c             *Client
	loc           *linkEntry
	root          *rootEntry
	lastRootCheck mclock.AbsTime // last revalidation of root
	enrs          *subtreeSync
	links         *subtreeSync
	linkCache     linkCache
}

func newClientTree(c *Client, loc *linkEntry) *clientTree {
	ct := &clientTree{c: c, loc: loc}
	ct.linkCache.self = ct
	return ct
}

func (ct *clientTree) matchPubkey(key *ecdsa.PublicKey) bool {
	return keysEqual(ct.loc.pubkey, key)
}

func keysEqual(k1, k2 *ecdsa.PublicKey) bool {
	return k1.Curve == k2.Curve && k1.X.Cmp(k2.X) == 0 && k1.Y.Cmp(k2.Y) == 0
}

// syncAll retrieves all entries of the tree.
func (ct *clientTree) syncAll(dest map[string]entry) error {
	if err := ct.updateRoot(); err != nil {
		return err
	}
	if err := ct.links.resolveAll(dest); err != nil {
		return err
	}
	if err := ct.enrs.resolveAll(dest); err != nil {
		return err
	}
	return nil
}

// syncRandom retrieves a single entry of the tree. The Node return value
// is non-nil if the entry was a node.
func (ct *clientTree) syncRandom(ctx context.Context) (*enode.Node, error) {
	if ct.rootUpdateDue() {
		if err := ct.updateRoot(); err != nil {
			return nil, err
		}
	}
	// Link tree sync has priority, run it to completion before syncing ENRs.
	if !ct.links.done() {
		err := ct.syncNextLink(ctx)
		return nil, err
	}

	// Sync next random entry in ENR tree. Once every node has been visited, we simply
	// start over. This is fine because entries are cached.
	if ct.enrs.done() {
		ct.enrs = newSubtreeSync(ct.c, ct.loc, ct.root.eroot, false)
	}
	return ct.syncNextRandomENR(ctx)
}

func (ct *clientTree) syncNextLink(ctx context.Context) error {
	hash := ct.links.missing[0]
	e, err := ct.links.resolveNext(ctx, hash)
	if err != nil {
		return err
	}
	ct.links.missing = ct.links.missing[1:]

	if le, ok := e.(*linkEntry); ok {
		lt, err := ct.c.ensureTree(le)
		if err != nil {
			return err
		}
		ct.linkCache.add(lt)
	}
	return nil
}

func (ct *clientTree) syncNextRandomENR(ctx context.Context) (*enode.Node, error) {
	index := rand.Intn(len(ct.enrs.missing))
	hash := ct.enrs.missing[index]
	e, err := ct.enrs.resolveNext(ctx, hash)
	if err != nil {
		return nil, err
	}
	ct.enrs.missing = removeHash(ct.enrs.missing, index)
	if ee, ok := e.(*enrEntry); ok {
		return ee.node, nil
	}
	return nil, nil
}

func (ct *clientTree) String() string {
	return ct.loc.String()
}

// removeHash removes the element at index from h.
func removeHash(h []string, index int) []string {
	if len(h) == 1 {
		return nil
	}
	last := len(h) - 1
	if index < last {
		h[index] = h[last]
		h[last] = ""
	}
	return h[:last]
}

// updateRoot ensures that the given tree has an up-to-date root.
func (ct *clientTree) updateRoot() error {
	ct.lastRootCheck = ct.c.clock.Now()
	ctx, cancel := context.WithTimeout(context.Background(), ct.c.cfg.Timeout)
	defer cancel()
	root, err := ct.c.resolveRoot(ctx, ct.loc)
	if err != nil {
		return err
	}
	ct.root = &root

	// Invalidate subtrees if changed.
	if ct.links == nil || root.lroot != ct.links.root {
		ct.links = newSubtreeSync(ct.c, ct.loc, root.lroot, true)
		ct.linkCache.reset()
	}
	if ct.enrs == nil || root.eroot != ct.enrs.root {
		ct.enrs = newSubtreeSync(ct.c, ct.loc, root.eroot, false)
	}
	return nil
}

// rootUpdateDue returns true when a root update is needed.
func (ct *clientTree) rootUpdateDue() bool {
	return ct.root == nil || time.Duration(ct.c.clock.Now()-ct.lastRootCheck) > ct.c.cfg.RecheckInterval
}

// subtreeSync is the sync of an ENR or link subtree.
type subtreeSync struct {
	c       *Client
	loc     *linkEntry
	root    string
	missing []string // missing tree node hashes
	link    bool     // true if this sync is for the link tree
}

func newSubtreeSync(c *Client, loc *linkEntry, root string, link bool) *subtreeSync {
	return &subtreeSync{c, loc, root, []string{root}, link}
}

func (ts *subtreeSync) done() bool {
	return len(ts.missing) == 0
}

func (ts *subtreeSync) resolveAll(dest map[string]entry) error {
	for !ts.done() {
		hash := ts.missing[0]
		ctx, cancel := context.WithTimeout(context.Background(), ts.c.cfg.Timeout)
		e, err := ts.resolveNext(ctx, hash)
		cancel()
		if err != nil {
			return err
		}
		dest[hash] = e
		ts.missing = ts.missing[1:]
	}
	return nil
}

func (ts *subtreeSync) resolveNext(ctx context.Context, hash string) (entry, error) {
	e, err := ts.c.resolveEntry(ctx, ts.loc.domain, hash)
	if err != nil {
		return nil, err
	}
	switch e := e.(type) {
	case *enrEntry:
		if ts.link {
			return nil, errENRInLinkTree
		}
	case *linkEntry:
		if !ts.link {
			return nil, errLinkInENRTree
		}
	case *branchEntry:
		ts.missing = append(ts.missing, e.children...)
	}
	return e, nil
}

// linkCache tracks the links of a tree.
type linkCache struct {
	self    *clientTree
	directM map[*clientTree]struct{} // direct links
	allM    map[*clientTree]struct{} // direct & transitive links
}

// reset clears the cache.
func (lc *linkCache) reset() {
	lc.directM = nil
	lc.allM = nil
}

// add adds a direct link to the cache.
func (lc *linkCache) add(ct *clientTree) {
	if lc.directM == nil {
		lc.directM = make(map[*clientTree]struct{})
	}
	if _, ok := lc.directM[ct]; !ok {
		lc.invalidate()
	}
	lc.directM[ct] = struct{}{}
}

// invalidate resets the cache of transitive links.
func (lc *linkCache) invalidate() {
	lc.allM = nil
}

// valid returns true when the cache of transitive links is up-to-date.
func (lc *linkCache) valid() bool {
	// Re-check validity of child caches to catch updates.
	for ct := range lc.allM {
		if ct != lc.self && !ct.linkCache.valid() {
			lc.allM = nil
			break
		}
	}
	return lc.allM != nil
}

// all returns all trees reachable through the cache.
func (lc *linkCache) all() map[*clientTree]struct{} {
	if lc.valid() {
		return lc.allM
	}
	// Remake lc.allM it by taking the union of all() across children.
	m := make(map[*clientTree]struct{})
	if lc.self != nil {
		m[lc.self] = struct{}{}
	}
	for ct := range lc.directM {
		m[ct] = struct{}{}
		for lt := range ct.linkCache.all() {
			m[lt] = struct{}{}
		}
	}
	lc.allM = m
	return m
}
