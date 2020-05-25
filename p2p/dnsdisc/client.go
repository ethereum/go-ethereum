// Copyright 2018 The go-ethereum Authors
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
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/time/rate"
)

// Client discovers nodes by querying DNS servers.
type Client struct {
	cfg     Config
	clock   mclock.Clock
	entries *lru.Cache
}

// Config holds configuration options for the client.
type Config struct {
	Timeout         time.Duration      // timeout used for DNS lookups (default 5s)
	RecheckInterval time.Duration      // time between tree root update checks (default 30min)
	CacheLimit      int                // maximum number of cached records (default 1000)
	RateLimit       float64            // maximum DNS requests / second (default 3)
	ValidSchemes    enr.IdentityScheme // acceptable ENR identity schemes (default enode.ValidSchemes)
	Resolver        Resolver           // the DNS resolver to use (defaults to system DNS)
	Logger          log.Logger         // destination of client log messages (defaults to root logger)
}

// Resolver is a DNS resolver that can query TXT records.
type Resolver interface {
	LookupTXT(ctx context.Context, domain string) ([]string, error)
}

func (cfg Config) withDefaults() Config {
	const (
		defaultTimeout   = 5 * time.Second
		defaultRecheck   = 30 * time.Minute
		defaultRateLimit = 3
		defaultCache     = 1000
	)
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.RecheckInterval == 0 {
		cfg.RecheckInterval = defaultRecheck
	}
	if cfg.CacheLimit == 0 {
		cfg.CacheLimit = defaultCache
	}
	if cfg.RateLimit == 0 {
		cfg.RateLimit = defaultRateLimit
	}
	if cfg.ValidSchemes == nil {
		cfg.ValidSchemes = enode.ValidSchemes
	}
	if cfg.Resolver == nil {
		cfg.Resolver = new(net.Resolver)
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Root()
	}
	return cfg
}

// NewClient creates a client.
func NewClient(cfg Config) *Client {
	cfg = cfg.withDefaults()
	cache, err := lru.New(cfg.CacheLimit)
	if err != nil {
		panic(err)
	}
	rlimit := rate.NewLimiter(rate.Limit(cfg.RateLimit), 10)
	cfg.Resolver = &rateLimitResolver{cfg.Resolver, rlimit}
	return &Client{cfg: cfg, entries: cache, clock: mclock.System{}}
}

// SyncTree downloads the entire node tree at the given URL.
func (c *Client) SyncTree(url string) (*Tree, error) {
	le, err := parseLink(url)
	if err != nil {
		return nil, fmt.Errorf("invalid enrtree URL: %v", err)
	}
	ct := newClientTree(c, new(linkCache), le)
	t := &Tree{entries: make(map[string]entry)}
	if err := ct.syncAll(t.entries); err != nil {
		return nil, err
	}
	t.root = ct.root
	return t, nil
}

// NewIterator creates an iterator that visits all nodes at the
// given tree URLs.
func (c *Client) NewIterator(urls ...string) (enode.Iterator, error) {
	it := c.newRandomIterator()
	for _, url := range urls {
		if err := it.addTree(url); err != nil {
			return nil, err
		}
	}
	return it, nil
}

// resolveRoot retrieves a root entry via DNS.
func (c *Client) resolveRoot(ctx context.Context, loc *linkEntry) (rootEntry, error) {
	txts, err := c.cfg.Resolver.LookupTXT(ctx, loc.domain)
	c.cfg.Logger.Trace("Updating DNS discovery root", "tree", loc.domain, "err", err)
	if err != nil {
		return rootEntry{}, err
	}
	for _, txt := range txts {
		if strings.HasPrefix(txt, rootPrefix) {
			return parseAndVerifyRoot(txt, loc)
		}
	}
	return rootEntry{}, nameError{loc.domain, errNoRoot}
}

func parseAndVerifyRoot(txt string, loc *linkEntry) (rootEntry, error) {
	e, err := parseRoot(txt)
	if err != nil {
		return e, err
	}
	if !e.verifySignature(loc.pubkey) {
		return e, entryError{typ: "root", err: errInvalidSig}
	}
	return e, nil
}

// resolveEntry retrieves an entry from the cache or fetches it from the network
// if it isn't cached.
func (c *Client) resolveEntry(ctx context.Context, domain, hash string) (entry, error) {
	cacheKey := truncateHash(hash)
	if e, ok := c.entries.Get(cacheKey); ok {
		return e.(entry), nil
	}
	e, err := c.doResolveEntry(ctx, domain, hash)
	if err != nil {
		return nil, err
	}
	c.entries.Add(cacheKey, e)
	return e, nil
}

// doResolveEntry fetches an entry via DNS.
func (c *Client) doResolveEntry(ctx context.Context, domain, hash string) (entry, error) {
	wantHash, err := b32format.DecodeString(hash)
	if err != nil {
		return nil, fmt.Errorf("invalid base32 hash")
	}
	name := hash + "." + domain
	txts, err := c.cfg.Resolver.LookupTXT(ctx, hash+"."+domain)
	c.cfg.Logger.Trace("DNS discovery lookup", "name", name, "err", err)
	if err != nil {
		return nil, err
	}
	for _, txt := range txts {
		e, err := parseEntry(txt, c.cfg.ValidSchemes)
		if err == errUnknownEntry {
			continue
		}
		if !bytes.HasPrefix(crypto.Keccak256([]byte(txt)), wantHash) {
			err = nameError{name, errHashMismatch}
		} else if err != nil {
			err = nameError{name, err}
		}
		return e, err
	}
	return nil, nameError{name, errNoEntry}
}

// rateLimitResolver applies a rate limit to a Resolver.
type rateLimitResolver struct {
	r       Resolver
	limiter *rate.Limiter
}

func (r *rateLimitResolver) LookupTXT(ctx context.Context, domain string) ([]string, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return r.r.LookupTXT(ctx, domain)
}

// randomIterator traverses a set of trees and returns nodes found in them.
type randomIterator struct {
	cur      *enode.Node
	ctx      context.Context
	cancelFn context.CancelFunc
	c        *Client

	mu    sync.Mutex
	trees map[string]*clientTree // all trees
	lc    linkCache              // tracks tree dependencies
}

func (c *Client) newRandomIterator() *randomIterator {
	ctx, cancel := context.WithCancel(context.Background())
	return &randomIterator{
		c:        c,
		ctx:      ctx,
		cancelFn: cancel,
		trees:    make(map[string]*clientTree),
	}
}

// Node returns the current node.
func (it *randomIterator) Node() *enode.Node {
	return it.cur
}

// Close closes the iterator.
func (it *randomIterator) Close() {
	it.mu.Lock()
	defer it.mu.Unlock()

	it.cancelFn()
	it.trees = nil
}

// Next moves the iterator to the next node.
func (it *randomIterator) Next() bool {
	it.cur = it.nextNode()
	return it.cur != nil
}

// addTree adds an enrtree:// URL to the iterator.
func (it *randomIterator) addTree(url string) error {
	le, err := parseLink(url)
	if err != nil {
		return fmt.Errorf("invalid enrtree URL: %v", err)
	}
	it.lc.addLink("", le.str)
	return nil
}

// nextNode syncs random tree entries until it finds a node.
func (it *randomIterator) nextNode() *enode.Node {
	for {
		ct := it.nextTree()
		if ct == nil {
			return nil
		}
		n, err := ct.syncRandom(it.ctx)
		if err != nil {
			if err == it.ctx.Err() {
				return nil // context canceled.
			}
			it.c.cfg.Logger.Debug("Error in DNS random node sync", "tree", ct.loc.domain, "err", err)
			continue
		}
		if n != nil {
			return n
		}
	}
}

// nextTree returns a random tree.
func (it *randomIterator) nextTree() *clientTree {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.lc.changed {
		it.rebuildTrees()
		it.lc.changed = false
	}
	if len(it.trees) == 0 {
		return nil
	}
	limit := rand.Intn(len(it.trees))
	for _, ct := range it.trees {
		if limit == 0 {
			return ct
		}
		limit--
	}
	return nil
}

// rebuildTrees rebuilds the 'trees' map.
func (it *randomIterator) rebuildTrees() {
	// Delete removed trees.
	for loc := range it.trees {
		if !it.lc.isReferenced(loc) {
			delete(it.trees, loc)
		}
	}
	// Add new trees.
	for loc := range it.lc.backrefs {
		if it.trees[loc] == nil {
			link, _ := parseLink(linkPrefix + loc)
			it.trees[loc] = newClientTree(it.c, &it.lc, link)
		}
	}
}
