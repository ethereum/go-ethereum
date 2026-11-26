// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/rlp"
)

// eth/76 relays mode 1 (recursive proof) wrappers, adds coverage hints to announcements, a
// compact flag to requests, and transaction-hash manifests.

const (
	eth76 = 76

	getProofWrapperManifestsMsg = 0x1b
	proofWrapperManifestsMsg    = 0x1c

	// aggregationTimeout bounds how long the node may take to aggregate pending wrappers
	// and announce the result. It covers several EIP-8288 aggregation intervals and the
	// proving time.
	aggregationTimeout = 90 * time.Second
)

// announcement76 is the eth/76 entry [mode, length, hash, deps-hash, txs-hash, txs-meta].
type announcement76 struct {
	Mode     uint64
	Length   uint64
	Hash     common.Hash
	DepsHash common.Hash
	TxsHash  common.Hash
	Meta     rlp.RawValue
}

// getProofWrappers76 is the eth/76 GetProofWrappers message [request-id, compact, [hash, ...]].
type getProofWrappers76 struct {
	RequestId uint64
	Compact   uint64
	Hashes    []common.Hash
}

// getProofWrapperManifestsPacket is the GetProofWrapperManifests message [request-id, [hash, ...]].
type getProofWrapperManifestsPacket struct {
	RequestId uint64
	Hashes    []common.Hash
}

// proofWrapperManifestsPacket is the ProofWrapperManifests message
// [request-id, [[hash, [txhash, ...]], ...]].
type proofWrapperManifestsPacket struct {
	RequestId uint64
	Entries   []manifestEntry
}

type manifestEntry struct {
	Hash     common.Hash
	TxHashes []common.Hash
}

// recursiveContent is the content [deps, recursive_stark] of a mode 1 wrapper.
type recursiveContent struct {
	Deps  [][]byte
	Proof []byte
}

// codec76 is eth/76: six-field announcements and three-field requests, for both modes.
type codec76 struct{}

func (codec76) encodeAnnouncements(anns []wrapperAnnouncement) ([]byte, error) {
	entries := make([]announcement76, len(anns))
	for i, a := range anns {
		entries[i] = announcement76{a.Mode, a.Length, a.Hash, a.DepsHash, a.TxsHash, a.Meta}
	}
	return rlp.EncodeToBytes(entries)
}

func (codec76) decodeAnnouncements(data []byte) ([]wrapperAnnouncement, error) {
	var entries []announcement76
	if err := rlp.DecodeBytes(data, &entries); err != nil {
		return nil, fmt.Errorf("invalid eth/76 announcement: %v", err)
	}
	anns := make([]wrapperAnnouncement, len(entries))
	for i, e := range entries {
		if e.Mode == wrapperModeRecursive && !bytes.Equal(e.Meta, emptyList) {
			return nil, fmt.Errorf("mode 1 announcement of %v carries inline metadata", e.Hash)
		}
		anns[i] = wrapperAnnouncement{Mode: e.Mode, Length: e.Length, Hash: e.Hash, DepsHash: e.DepsHash, TxsHash: e.TxsHash, Meta: e.Meta}
	}
	return anns, nil
}

func (codec76) encodeRequest(id uint64, hashes []common.Hash) any {
	return &getProofWrappers76{RequestId: id, Compact: 0, Hashes: hashes}
}

func (codec76) decodeRequest(data []byte) (*wrapperRequest, error) {
	var p getProofWrappers76
	if err := rlp.DecodeBytes(data, &p); err != nil {
		return nil, fmt.Errorf("invalid eth/76 GetProofWrappers: %v", err)
	}
	if p.Compact > 1 {
		return nil, fmt.Errorf("GetProofWrappers compact flag %d", p.Compact)
	}
	return &wrapperRequest{id: p.RequestId, compact: p.Compact == 1, hashes: p.Hashes}, nil
}

// serveManifests answers a GetProofWrapperManifests request with the manifests of the
// requested fixtures, in request order. A manifest lists all transaction hashes of a
// wrapper; manifestOf may replace it.
func (c *Conn) serveManifests(data []byte, manifestOf func(*wrapperFixture) []common.Hash, fixtures ...*wrapperFixture) error {
	var req getProofWrapperManifestsPacket
	if err := rlp.DecodeBytes(data, &req); err != nil {
		return fmt.Errorf("node sent an invalid GetProofWrapperManifests: %v", err)
	}
	if len(req.Hashes) == 0 || len(req.Hashes) > maxWrapperRequest {
		return fmt.Errorf("GetProofWrapperManifests for %d hashes", len(req.Hashes))
	}
	resp := proofWrapperManifestsPacket{RequestId: req.RequestId, Entries: []manifestEntry{}}
	for _, h := range req.Hashes {
		for _, f := range fixtures {
			if f.hash() == h {
				resp.Entries = append(resp.Entries, manifestEntry{Hash: h, TxHashes: manifestOf(f)})
			}
		}
	}
	return c.writeWrapperMsg(proofWrapperManifestsMsg, &resp)
}

func trueManifest(f *wrapperFixture) []common.Hash { return f.txHashes() }

// fetchManifests requests manifests from the node and returns its response.
func (c *Conn) fetchManifests(hashes ...common.Hash) (*proofWrapperManifestsPacket, error) {
	id := uint64(time.Now().UnixNano())
	if err := c.writeWrapperMsg(getProofWrapperManifestsMsg, &getProofWrapperManifestsPacket{RequestId: id, Hashes: hashes}); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wrapperTimeout)
	for {
		msg, err := c.readWrapperMsg(deadline)
		if err != nil {
			return nil, fmt.Errorf("no ProofWrapperManifests response: %v", err)
		}
		if msg.code != proofWrapperManifestsMsg {
			continue
		}
		var resp proofWrapperManifestsPacket
		if err := rlp.DecodeBytes(msg.data, &resp); err != nil {
			return nil, fmt.Errorf("invalid ProofWrapperManifests: %v", err)
		}
		if resp.RequestId != id {
			return nil, fmt.Errorf("ProofWrapperManifests for unknown request %d", resp.RequestId)
		}
		return &resp, nil
	}
}

// Eth76Tests returns the eth/76 aggregated proof wrapper tests. The node must run the
// chain in testdata/eip8288, with EIP-8288 active at its current time, and aggregate
// pending wrappers.
func (s *Suite) Eth76Tests() []utesting.Test {
	return []utesting.Test{
		{Name: "AggregateRelay", Fn: s.TestAggregateRelay},
		{Name: "Reaggregation", Fn: s.TestReaggregation, Slow: true},
		{Name: "ManifestRoundTrip", Fn: s.TestManifestRoundTrip},
		{Name: "LyingManifest", Fn: s.TestLyingManifest},
		{Name: "AggregateInvalidProof", Fn: s.TestAggregateInvalidProof},
		{Name: "AggregateHintMismatch", Fn: s.TestAggregateHintMismatch},
		{Name: "Eth76AnnouncementLimits", Fn: s.TestEth76AnnouncementLimits},
		{Name: "Eth76RequestLimits", Fn: s.TestEth76RequestLimits},
	}
}

func recursive(f *eip8288Fixtures) []*wrapperFixture { return f.Recursive }

func (s *Suite) TestAggregateRelay(t *utesting.T) {
	t.Log(`This test announces a valid mode 1 wrapper with correct coverage hints. Its
transactions fall into several transaction-hash prefixes, of which a node aggregates only
some; admission and relay of received aggregates must not depend on that choice. The node
must request it, validate it and re-announce it to another eth/76 peer, which can fetch it.`)
	s.testRelay(t, eth76, s.takeWrappers(t, recursive, 1)...)
}

// encodingHashes maps every fixture transaction encoding to its consensus hash, which
// go-ethereum cannot compute for frame transactions.
func (s *Suite) encodingHashes(t *utesting.T) map[string]common.Hash {
	f, err := loadFixtures(s.chainDir)
	if err != nil {
		t.Fatal(err)
	}
	hashes := make(map[string]common.Hash)
	for _, kind := range [][]*wrapperFixture{f.Direct, f.Inline, f.UnknownScheme, f.Recursive} {
		for _, w := range kind {
			b, err := w.body()
			if err != nil {
				t.Fatal(err)
			}
			for i, tx := range b.Transactions {
				hashes[string(tx)] = w.Transactions[i].Hash
			}
		}
	}
	return hashes
}

// checkAggregate checks a fetched mode 1 wrapper against its announcement and returns the
// transaction hashes it covers.
func checkAggregate(ann *wrapperAnnouncement, wrapper []byte, hashes map[string]common.Hash) ([]common.Hash, error) {
	f := &wrapperFixture{Wrapper: wrapper}
	b, err := f.body()
	if err != nil {
		return nil, err
	}
	var content recursiveContent
	if err := rlp.DecodeBytes(b.Content, &content); err != nil {
		return nil, fmt.Errorf("aggregate content is not [deps, recursive_stark]: %v", err)
	}
	if len(content.Proof) == 0 || len(content.Deps) == 0 {
		return nil, errors.New("aggregate has no dependencies or no proof")
	}
	for i, dep := range content.Deps {
		if len(dep) != 96 || i > 0 && bytes.Compare(content.Deps[i-1], dep) >= 0 {
			return nil, errors.New("aggregate dependencies are not canonical")
		}
	}
	if f.depsHash() != ann.DepsHash {
		return nil, errors.New("aggregate does not match its announced deps-hash")
	}
	covered := make([]common.Hash, len(b.Transactions))
	for i, tx := range b.Transactions {
		h, ok := hashes[string(tx)]
		if !ok {
			return nil, errors.New("aggregate carries a transaction the test never sent")
		}
		if i > 0 && covered[i-1].Cmp(h) >= 0 {
			return nil, errors.New("aggregate transactions are not in ascending hash order")
		}
		covered[i] = h
	}
	if hashesDigest(covered) != ann.TxsHash {
		return nil, errors.New("aggregate does not match its announced txs-hash")
	}
	return covered, nil
}

func (s *Suite) TestReaggregation(t *utesting.T) {
	t.Log(`This test sends the node several valid mode 0 wrappers, then waits for it to
aggregate their transactions into mode 1 wrappers. A peer connecting afterwards has no
coverage of these transactions, so the node should offer it the aggregates. Each must be
fetchable, match its coverage hints and carry no inline metadata, and cover only
transactions the test sent to the node. An eth/75 peer must never be offered mode 1.`)
	source, err := s.dialWrapperPeer(eth76)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	fixtures := s.takeWrappers(t, direct, 3)
	if err := relayWrappers(source, fixtures...); err != nil {
		t.Fatal(err)
	}
	sent := make(map[common.Hash]bool)
	for _, f := range fixtures {
		for _, h := range f.txHashes() {
			sent[h] = true
		}
	}

	observer, err := s.dialWrapperPeer(eth76)
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Close()
	var (
		encodings  = s.encodingHashes(t)
		aggregates = make(map[common.Hash]bool)
		covered    = make(map[common.Hash]bool)
		deadline   = time.Now().Add(aggregationTimeout)
	)
	for countTrue(covered) < len(sent) {
		msg, err := observer.readWrapperMsg(deadline)
		if isTimeout(err) && len(aggregates) > 0 {
			break
		}
		if err != nil {
			t.Fatalf("node did not offer an aggregate: %v", err)
		}
		if msg.code != newProofWrapperHashesMsg {
			continue
		}
		anns, err := observer.decodeAnnouncements(msg.data)
		if err != nil {
			t.Fatalf("node sent an invalid announcement: %v", err)
		}
		for i := range anns {
			ann := &anns[i]
			if ann.Mode != wrapperModeRecursive || aggregates[ann.Hash] {
				continue
			}
			wrapper, err := observer.fetchWrapper(ann)
			if err != nil {
				t.Fatal(err)
			}
			hashes, err := checkAggregate(ann, wrapper, encodings)
			if err != nil {
				t.Fatal(err)
			}
			// Aggregates of earlier tests' transactions may be offered too.
			if !slices.ContainsFunc(hashes, func(h common.Hash) bool { return sent[h] }) {
				continue
			}
			for _, h := range hashes {
				covered[h] = sent[h] || covered[h]
			}
			aggregates[ann.Hash] = true
			// Aggregates of the same cycle follow closely; stop once they stop coming.
			deadline = time.Now().Add(quietPeriod)
		}
	}
	t.Logf("%d aggregates cover %d of %d transactions", len(aggregates), countTrue(covered), len(sent))

	old, err := s.dialWrapperPeer(eth75)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	if err := old.expectNoAnnouncement(quietPeriod, slices.Collect(maps.Keys(aggregates))...); err != nil {
		t.Fatalf("mode 1 wrapper offered over eth/75: %v", err)
	}
}

func countTrue(m map[common.Hash]bool) (n int) {
	for _, v := range m {
		if v {
			n++
		}
	}
	return n
}

func (s *Suite) TestManifestRoundTrip(t *utesting.T) {
	t.Log(`This test sends the node a valid wrapper and requests its manifest, together
with the manifest of a wrapper the node never announced. The node must return the
wrapper's transaction hashes in ascending order, matching the txs-hash it announced, and
omit the unannounced wrapper without penalty.`)
	conns := s.dialWrapperPeers(t, eth76, 2)
	defer closeConns(conns)
	source, observer := conns[0], conns[1]

	f := s.takeWrappers(t, direct, 1)[0]
	if err := relayWrappers(source, f); err != nil {
		t.Fatal(err)
	}
	ann, err := observer.expectAnnouncement(f.hash(), wrapperTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if ann.DepsHash != f.depsHash() || ann.TxsHash != f.txsHash() {
		t.Fatal("node announced false coverage hints")
	}
	unknown := common.Hash{0xde, 0xad}
	resp, err := observer.fetchManifests(f.hash(), unknown)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Entries) != 1 || resp.Entries[0].Hash != f.hash() {
		t.Fatalf("node returned %d manifests, want only that of %v", len(resp.Entries), f.hash())
	}
	got := resp.Entries[0].TxHashes
	if !slices.Equal(got, f.txHashes()) || hashesDigest(got) != ann.TxsHash {
		t.Fatal("manifest does not list the wrapper's transaction hashes")
	}
	if err := observer.expectConnected(quietPeriod); err != nil {
		t.Fatal(err)
	}
}

// answerWrapperRequests serves the node's wrapper and manifest requests on a connection
// until it closes or stop is closed, and reports whether the node requested the wrapper.
func answerWrapperRequests(conn *Conn, f *wrapperFixture, manifestOf func(*wrapperFixture) []common.Hash, stop chan struct{}, requested chan<- struct{}) error {
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		msg, err := conn.readWrapperMsg(time.Now().Add(time.Second))
		if isTimeout(err) {
			continue
		}
		if err != nil {
			return err
		}
		switch msg.code {
		case getProofWrapperManifestsMsg:
			if err := conn.serveManifests(msg.data, manifestOf, f); err != nil {
				return err
			}
		case getProofWrappersMsg:
			req, err := conn.decodeWrapperRequest(msg.data)
			if err != nil {
				return err
			}
			if slices.Contains(req.hashes, f.hash()) && requested != nil {
				close(requested)
				requested = nil
			}
			if err := conn.serve(req, f); err != nil {
				return err
			}
		}
	}
}

func (s *Suite) TestLyingManifest(t *utesting.T) {
	t.Log(`This test first gives the node a valid mode 1 wrapper, so that it holds an
aggregate. A lying peer then announces another valid aggregate with a txs-hash and manifest
claiming only the transactions the node already holds, while an honest peer announces the
same aggregate truthfully. A claim of one peer must not suppress fetching from another:
the node must fetch the aggregate from the honest peer, admit it and relay it.`)
	conns := s.dialWrapperPeers(t, eth76, 4)
	defer closeConns(conns)
	seeder, liar, honest, observer := conns[0], conns[1], conns[2], conns[3]

	fixtures := s.takeWrappers(t, recursive, 2)
	held, offered := fixtures[0], fixtures[1]
	if err := relayWrappers(seeder, held); err != nil {
		t.Fatal(err)
	}
	if _, err := observer.expectAnnouncement(held.hash(), wrapperTimeout); err != nil {
		t.Fatalf("seed aggregate was not admitted: %v", err)
	}

	lie := offered.announcement()
	lie.TxsHash = held.txsHash()
	if err := liar.writeAnnouncements(lie); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// The liar may be disconnected once its claims are found false.
		answerWrapperRequests(liar, offered, func(*wrapperFixture) []common.Hash { return held.txHashes() }, stop, nil)
	}()
	defer func() {
		close(stop)
		wg.Wait()
	}()

	time.Sleep(time.Second)
	if err := honest.announce(offered); err != nil {
		t.Fatal(err)
	}
	requested := make(chan struct{})
	honestErr := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		honestErr <- answerWrapperRequests(honest, offered, trueManifest, stop, requested)
	}()

	anns, err := observer.expectAnnouncements(wrapperTimeout, offered.hash())
	select {
	case <-requested:
	default:
		t.Fatal("node did not fetch the aggregate from the honest peer")
	}
	if err != nil {
		t.Fatalf("aggregate was not admitted: %v", err)
	}
	if anns[offered.hash()].TxsHash != offered.txsHash() {
		t.Fatal("node relayed the liar's coverage hint")
	}
	select {
	case err := <-honestErr:
		t.Fatalf("node dropped the honest peer: %v", err)
	default:
	}
}

func (s *Suite) TestAggregateInvalidProof(t *utesting.T) {
	t.Log(`This test serves a well-formed mode 1 wrapper whose recursive proof does not
verify. Proof failures make a wrapper invalid, not malformed: the node must discard it
without penalty or disconnection, and must not relay it.`)
	f := s.takeWrappers(t, direct, 1)[0]
	proof := make([]byte, 4096)
	rand.Read(proof)
	s.testInvalidWrapper(t, eth76, f.withContent(wrapperModeRecursive, recursiveContent{Deps: f.deps(), Proof: proof}))
}

func (s *Suite) TestAggregateHintMismatch(t *utesting.T) {
	t.Log(`This test announces a valid mode 1 wrapper with a false deps-hash. Once the
wrapper arrives, the node must discard it before proof verification and disconnect the
peer. The same wrapper announced correctly by another peer must still be fetched.`)
	f := s.takeWrappers(t, recursive, 1)[0]
	ann := f.announcement()
	ann.DepsHash[0] ^= 0xff
	s.testAnnouncementMismatch(t, eth76, f, ann)
}

func (s *Suite) TestEth76AnnouncementLimits(t *utesting.T) {
	t.Log(`This test sends eth/76 NewProofWrapperHashes messages that violate the
item-count, byte-length and ordering rules, an eth/75-shaped announcement, and a mode 1
entry with inline metadata. Each must cause disconnection.`)
	s.testAnnouncementLimits(t, eth76)

	short, _ := codec75{}.encodeAnnouncements(randomAnnouncements(1))
	s.testViolation(t, eth76, "eth/75 announcement shape", newProofWrapperHashesMsg, short)

	meta := randomAnnouncements(1)
	meta[0].Mode = wrapperModeRecursive
	f := s.takeWrappers(t, inline, 1)[0]
	meta[0].Meta = f.meta()
	enc, _ := codec76{}.encodeAnnouncements(meta)
	s.testViolation(t, eth76, "mode 1 with inline metadata", newProofWrapperHashesMsg, enc)
}

func (s *Suite) TestEth76RequestLimits(t *utesting.T) {
	t.Log(`This test checks eth/76 GetProofWrappers bounds: a request for an unannounced
hash gets an empty response without penalty, and requests for no hashes or more than 32
hashes cause disconnection.`)
	s.testRequestLimits(t, eth76)
}
