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
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	lru "github.com/hashicorp/golang-lru"
)

// Client discovers nodes by querying DNS servers.
type Client struct {
	cfg       Config
	clock     mclock.Clock
	linkCache linkCache
	trees     map[string]*clientTree

	entries *lru.Cache
}

// Config holds configuration options for the client.
type Config struct {
	Timeout         time.Duration      // timeout used for DNS lookups (default 5s)
	RecheckInterval time.Duration      // time between tree root update checks (default 30min)
	CacheLimit      int                // maximum number of cached records (default 1000)
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
		defaultTimeout = 5 * time.Second
		defaultRecheck = 30 * time.Minute
		defaultCache   = 1000
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
func NewClient(cfg Config, urls ...string) (*Client, error) {
	c := &Client{
		cfg:   cfg.withDefaults(),
		clock: mclock.System{},
		trees: make(map[string]*clientTree),
	}
	var err error
	if c.entries, err = lru.New(c.cfg.CacheLimit); err != nil {
		return nil, err
	}
	for _, url := range urls {
		if err := c.AddTree(url); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// SyncTree downloads the entire node tree at the given URL. This doesn't add the tree for
// later use, but any previously-synced entries are reused.
func (c *Client) SyncTree(url string) (*Tree, error) {
	le, err := parseLink(url)
	if err != nil {
		return nil, fmt.Errorf("invalid enrtree URL: %v", err)
	}
	ct := newClientTree(c, le)
	t := &Tree{entries: make(map[string]entry)}
	if err := ct.syncAll(t.entries); err != nil {
		return nil, err
	}
	t.root = ct.root
	return t, nil
}

// AddTree adds a enrtree:// URL to crawl.
func (c *Client) AddTree(url string) error {
	le, err := parseLink(url)
	if err != nil {
		return fmt.Errorf("invalid enrtree URL: %v", err)
	}
	ct, err := c.ensureTree(le)
	if err != nil {
		return err
	}
	c.linkCache.add(ct)
	return nil
}

func (c *Client) ensureTree(le *linkEntry) (*clientTree, error) {
	if tree, ok := c.trees[le.domain]; ok {
		if !tree.matchPubkey(le.pubkey) {
			return nil, fmt.Errorf("conflicting public keys for domain %q", le.domain)
		}
		return tree, nil
	}
	ct := newClientTree(c, le)
	c.trees[le.domain] = ct
	return ct, nil
}

// RandomNode retrieves the next random node.
func (c *Client) RandomNode(ctx context.Context) *enode.Node {
	for {
		ct := c.randomTree()
		if ct == nil {
			return nil
		}
		n, err := ct.syncRandom(ctx)
		if err != nil {
			if err == ctx.Err() {
				return nil // context canceled.
			}
			c.cfg.Logger.Debug("Error in DNS random node sync", "tree", ct.loc.domain, "err", err)
			continue
		}
		if n != nil {
			return n
		}
	}
}

// randomTree returns a random tree.
func (c *Client) randomTree() *clientTree {
	if !c.linkCache.valid() {
		c.gcTrees()
	}
	limit := rand.Intn(len(c.trees))
	for _, ct := range c.trees {
		if limit == 0 {
			return ct
		}
		limit--
	}
	return nil
}

// gcTrees rebuilds the 'trees' map.
func (c *Client) gcTrees() {
	trees := make(map[string]*clientTree)
	for t := range c.linkCache.all() {
		trees[t.loc.domain] = t
	}
	c.trees = trees
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
