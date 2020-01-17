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
	"crypto/ecdsa"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// Tree is a merkle tree of node records.
type Tree struct {
	root    *rootEntry
	entries map[string]entry
}

// Sign signs the tree with the given private key and sets the sequence number.
func (t *Tree) Sign(key *ecdsa.PrivateKey, domain string) (url string, err error) {
	root := *t.root
	sig, err := crypto.Sign(root.sigHash(), key)
	if err != nil {
		return "", err
	}
	root.sig = sig
	t.root = &root
	link := newLinkEntry(domain, &key.PublicKey)
	return link.String(), nil
}

// SetSignature verifies the given signature and assigns it as the tree's current
// signature if valid.
func (t *Tree) SetSignature(pubkey *ecdsa.PublicKey, signature string) error {
	sig, err := b64format.DecodeString(signature)
	if err != nil || len(sig) != crypto.SignatureLength {
		return errInvalidSig
	}
	root := *t.root
	root.sig = sig
	if !root.verifySignature(pubkey) {
		return errInvalidSig
	}
	t.root = &root
	return nil
}

// Seq returns the sequence number of the tree.
func (t *Tree) Seq() uint {
	return t.root.seq
}

// Signature returns the signature of the tree.
func (t *Tree) Signature() string {
	return b64format.EncodeToString(t.root.sig)
}

// ToTXT returns all DNS TXT records required for the tree.
func (t *Tree) ToTXT(domain string) map[string]string {
	records := map[string]string{domain: t.root.String()}
	for _, e := range t.entries {
		sd := subdomain(e)
		if domain != "" {
			sd = sd + "." + domain
		}
		records[sd] = e.String()
	}
	return records
}

// Links returns all links contained in the tree.
func (t *Tree) Links() []string {
	var links []string
	for _, e := range t.entries {
		if le, ok := e.(*linkEntry); ok {
			links = append(links, le.String())
		}
	}
	return links
}

// Nodes returns all nodes contained in the tree.
func (t *Tree) Nodes() []*enode.Node {
	var nodes []*enode.Node
	for _, e := range t.entries {
		if ee, ok := e.(*enrEntry); ok {
			nodes = append(nodes, ee.node)
		}
	}
	return nodes
}

const (
	hashAbbrev    = 16
	maxChildren   = 300 / hashAbbrev * (13 / 8)
	minHashLength = 12
)

// MakeTree creates a tree containing the given nodes and links.
func MakeTree(seq uint, nodes []*enode.Node, links []string) (*Tree, error) {
	// Sort records by ID and ensure all nodes have a valid record.
	records := make([]*enode.Node, len(nodes))

	copy(records, nodes)
	sortByID(records)
	for _, n := range records {
		if len(n.Record().Signature()) == 0 {
			return nil, fmt.Errorf("can't add node %v: unsigned node record", n.ID())
		}
	}

	// Create the leaf list.
	enrEntries := make([]entry, len(records))
	for i, r := range records {
		enrEntries[i] = &enrEntry{r}
	}
	linkEntries := make([]entry, len(links))
	for i, l := range links {
		le, err := parseLink(l)
		if err != nil {
			return nil, err
		}
		linkEntries[i] = le
	}

	// Create intermediate nodes.
	t := &Tree{entries: make(map[string]entry)}
	eroot := t.build(enrEntries)
	t.entries[subdomain(eroot)] = eroot
	lroot := t.build(linkEntries)
	t.entries[subdomain(lroot)] = lroot
	t.root = &rootEntry{seq: seq, eroot: subdomain(eroot), lroot: subdomain(lroot)}
	return t, nil
}

func (t *Tree) build(entries []entry) entry {
	if len(entries) == 1 {
		return entries[0]
	}
	if len(entries) <= maxChildren {
		hashes := make([]string, len(entries))
		for i, e := range entries {
			hashes[i] = subdomain(e)
			t.entries[hashes[i]] = e
		}
		return &branchEntry{hashes}
	}
	var subtrees []entry
	for len(entries) > 0 {
		n := maxChildren
		if len(entries) < n {
			n = len(entries)
		}
		sub := t.build(entries[:n])
		entries = entries[n:]
		subtrees = append(subtrees, sub)
		t.entries[subdomain(sub)] = sub
	}
	return t.build(subtrees)
}

func sortByID(nodes []*enode.Node) []*enode.Node {
	sort.Slice(nodes, func(i, j int) bool {
		return bytes.Compare(nodes[i].ID().Bytes(), nodes[j].ID().Bytes()) < 0
	})
	return nodes
}

// Entry Types

type entry interface {
	fmt.Stringer
}

type (
	rootEntry struct {
		eroot string
		lroot string
		seq   uint
		sig   []byte
	}
	branchEntry struct {
		children []string
	}
	enrEntry struct {
		node *enode.Node
	}
	linkEntry struct {
		str    string
		domain string
		pubkey *ecdsa.PublicKey
	}
)

// Entry Encoding

var (
	b32format = base32.StdEncoding.WithPadding(base32.NoPadding)
	b64format = base64.RawURLEncoding
)

const (
	rootPrefix   = "enrtree-root:v1"
	linkPrefix   = "enrtree://"
	branchPrefix = "enrtree-branch:"
	enrPrefix    = "enr:"
)

func subdomain(e entry) string {
	h := sha3.NewLegacyKeccak256()
	io.WriteString(h, e.String())
	return b32format.EncodeToString(h.Sum(nil)[:16])
}

func (e *rootEntry) String() string {
	return fmt.Sprintf(rootPrefix+" e=%s l=%s seq=%d sig=%s", e.eroot, e.lroot, e.seq, b64format.EncodeToString(e.sig))
}

func (e *rootEntry) sigHash() []byte {
	h := sha3.NewLegacyKeccak256()
	fmt.Fprintf(h, rootPrefix+" e=%s l=%s seq=%d", e.eroot, e.lroot, e.seq)
	return h.Sum(nil)
}

func (e *rootEntry) verifySignature(pubkey *ecdsa.PublicKey) bool {
	sig := e.sig[:crypto.RecoveryIDOffset] // remove recovery id
	enckey := crypto.FromECDSAPub(pubkey)
	return crypto.VerifySignature(enckey, e.sigHash(), sig)
}

func (e *branchEntry) String() string {
	return branchPrefix + strings.Join(e.children, ",")
}

func (e *enrEntry) String() string {
	return e.node.String()
}

func (e *linkEntry) String() string {
	return linkPrefix + e.str
}

func newLinkEntry(domain string, pubkey *ecdsa.PublicKey) *linkEntry {
	key := b32format.EncodeToString(crypto.CompressPubkey(pubkey))
	str := key + "@" + domain
	return &linkEntry{str, domain, pubkey}
}

// Entry Parsing

func parseEntry(e string, validSchemes enr.IdentityScheme) (entry, error) {
	switch {
	case strings.HasPrefix(e, linkPrefix):
		return parseLinkEntry(e)
	case strings.HasPrefix(e, branchPrefix):
		return parseBranch(e)
	case strings.HasPrefix(e, enrPrefix):
		return parseENR(e, validSchemes)
	default:
		return nil, errUnknownEntry
	}
}

func parseRoot(e string) (rootEntry, error) {
	var eroot, lroot, sig string
	var seq uint
	if _, err := fmt.Sscanf(e, rootPrefix+" e=%s l=%s seq=%d sig=%s", &eroot, &lroot, &seq, &sig); err != nil {
		return rootEntry{}, entryError{"root", errSyntax}
	}
	if !isValidHash(eroot) || !isValidHash(lroot) {
		return rootEntry{}, entryError{"root", errInvalidChild}
	}
	sigb, err := b64format.DecodeString(sig)
	if err != nil || len(sigb) != crypto.SignatureLength {
		return rootEntry{}, entryError{"root", errInvalidSig}
	}
	return rootEntry{eroot, lroot, seq, sigb}, nil
}

func parseLinkEntry(e string) (entry, error) {
	le, err := parseLink(e)
	if err != nil {
		return nil, err
	}
	return le, nil
}

func parseLink(e string) (*linkEntry, error) {
	if !strings.HasPrefix(e, linkPrefix) {
		return nil, fmt.Errorf("wrong/missing scheme 'enrtree' in URL")
	}
	e = e[len(linkPrefix):]
	pos := strings.IndexByte(e, '@')
	if pos == -1 {
		return nil, entryError{"link", errNoPubkey}
	}
	keystring, domain := e[:pos], e[pos+1:]
	keybytes, err := b32format.DecodeString(keystring)
	if err != nil {
		return nil, entryError{"link", errBadPubkey}
	}
	key, err := crypto.DecompressPubkey(keybytes)
	if err != nil {
		return nil, entryError{"link", errBadPubkey}
	}
	return &linkEntry{e, domain, key}, nil
}

func parseBranch(e string) (entry, error) {
	e = e[len(branchPrefix):]
	if e == "" {
		return &branchEntry{}, nil // empty entry is OK
	}
	hashes := make([]string, 0, strings.Count(e, ","))
	for _, c := range strings.Split(e, ",") {
		if !isValidHash(c) {
			return nil, entryError{"branch", errInvalidChild}
		}
		hashes = append(hashes, c)
	}
	return &branchEntry{hashes}, nil
}

func parseENR(e string, validSchemes enr.IdentityScheme) (entry, error) {
	e = e[len(enrPrefix):]
	enc, err := b64format.DecodeString(e)
	if err != nil {
		return nil, entryError{"enr", errInvalidENR}
	}
	var rec enr.Record
	if err := rlp.DecodeBytes(enc, &rec); err != nil {
		return nil, entryError{"enr", err}
	}
	n, err := enode.New(validSchemes, &rec)
	if err != nil {
		return nil, entryError{"enr", err}
	}
	return &enrEntry{n}, nil
}

func isValidHash(s string) bool {
	dlen := b32format.DecodedLen(len(s))
	if dlen < minHashLength || dlen > 32 || strings.ContainsAny(s, "\n\r") {
		return false
	}
	buf := make([]byte, 32)
	_, err := b32format.Decode(buf, []byte(s))
	return err == nil
}

// truncateHash truncates the given base32 hash string to the minimum acceptable length.
func truncateHash(hash string) string {
	maxLen := b32format.EncodedLen(minHashLength)
	if len(hash) < maxLen {
		panic(fmt.Errorf("dnsdisc: hash %q is too short", hash))
	}
	return hash[:maxLen]
}

// URL encoding

// ParseURL parses an enrtree:// URL and returns its components.
func ParseURL(url string) (domain string, pubkey *ecdsa.PublicKey, err error) {
	le, err := parseLink(url)
	if err != nil {
		return "", nil, err
	}
	return le.domain, le.pubkey, nil
}
