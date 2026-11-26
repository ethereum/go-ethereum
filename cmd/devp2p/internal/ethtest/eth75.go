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
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/rlp"
)

// Eth75Tests returns the eth/75 (EIP-8437) proof wrapper tests. The node must run the
// chain in testdata/eip8288, with EIP-8288 active at its current time.
func (s *Suite) Eth75Tests() []utesting.Test {
	return []utesting.Test{
		{Name: "WrapperRelay", Fn: s.TestWrapperRelay},
		{Name: "WrapperRelayInlineMetadata", Fn: s.TestWrapperRelayInlineMetadata},
		{Name: "WrapperRelayAnyBucket", Fn: s.TestWrapperRelayAnyBucket},
		{Name: "WrapperInvalidWitness", Fn: s.TestWrapperInvalidWitness},
		{Name: "WrapperUnknownScheme", Fn: s.TestWrapperUnknownScheme},
		{Name: "WrapperUnsupportedMode", Fn: s.TestWrapperUnsupportedMode},
		{Name: "WrapperMetadataMismatch", Fn: s.TestWrapperMetadataMismatch},
		{Name: "WrapperMetadataMissing", Fn: s.TestWrapperMetadataMissing},
		{Name: "WrapperLengthMismatch", Fn: s.TestWrapperLengthMismatch},
		{Name: "WrapperAnnouncementLimits", Fn: s.TestWrapperAnnouncementLimits},
		{Name: "WrapperRequestLimits", Fn: s.TestWrapperRequestLimits},
	}
}

// Eth75PreForkTests returns the eth/75 tests for a node whose chain schedules EIP-8288
// after its current time.
func (s *Suite) Eth75PreForkTests() []utesting.Test {
	return []utesting.Test{
		{Name: "WrapperExchangeBeforeFork", Fn: s.TestWrapperExchangeBeforeFork},
	}
}

func (s *Suite) takeWrappers(t *utesting.T, kind func(*eip8288Fixtures) []*wrapperFixture, n int) []*wrapperFixture {
	fixtures, err := s.wrappers.take(s.chainDir, kind, n)
	if err != nil {
		t.Fatal(err)
	}
	return fixtures
}

// takeCovering takes wrappers until their transactions cover every first-nibble prefix of
// a transaction hash.
func (s *Suite) takeCovering(t *utesting.T, kind func(*eip8288Fixtures) []*wrapperFixture) []*wrapperFixture {
	var (
		fixtures []*wrapperFixture
		prefixes [16]bool
	)
	for slices.Contains(prefixes[:], false) {
		f := s.takeWrappers(t, kind, 1)[0]
		for _, h := range f.txHashes() {
			prefixes[h[0]>>4] = true
		}
		fixtures = append(fixtures, f)
	}
	return fixtures
}

func direct(f *eip8288Fixtures) []*wrapperFixture        { return f.Direct }
func inline(f *eip8288Fixtures) []*wrapperFixture        { return f.Inline }
func unknownScheme(f *eip8288Fixtures) []*wrapperFixture { return f.UnknownScheme }

// dialWrapperPeers connects n peers of the given version.
func (s *Suite) dialWrapperPeers(t *utesting.T, version uint, n int) []*Conn {
	conns := make([]*Conn, 0, n)
	for range n {
		conn, err := s.dialWrapperPeer(version)
		if err != nil {
			closeConns(conns)
			t.Fatalf("eth/%d peering failed: %v", version, err)
		}
		conns = append(conns, conn)
	}
	return conns
}

// testRelay sends the wrappers to the node from one peer and checks that the node
// re-announces each of them unchanged to another peer, which can fetch them.
func (s *Suite) testRelay(t *utesting.T, version uint, fixtures ...*wrapperFixture) {
	conns := s.dialWrapperPeers(t, version, 2)
	defer closeConns(conns)
	source, observer := conns[0], conns[1]

	if err := relayWrappers(source, fixtures...); err != nil {
		t.Fatal(err)
	}
	hashes := make([]common.Hash, len(fixtures))
	for i, f := range fixtures {
		hashes[i] = f.hash()
	}
	anns, err := observer.expectAnnouncements(wrapperTimeout, hashes...)
	if err != nil {
		t.Fatalf("wrapper was not relayed: %v", err)
	}
	for _, f := range fixtures {
		ann := anns[f.hash()]
		want := f.announcement()
		if ann.Mode != want.Mode || ann.Length != want.Length {
			t.Fatalf("relayed announcement has mode %d and length %d, want %d and %d", ann.Mode, ann.Length, want.Mode, want.Length)
		}
		if !bytes.Equal(ann.Meta, want.Meta) {
			t.Fatalf("relayed announcement has txs-meta %x, want %x", ann.Meta, want.Meta)
		}
		wrapper, err := observer.fetchWrapper(ann)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(wrapper, f.Wrapper) {
			t.Fatal("node relayed a different wrapper than it received")
		}
	}
}

func (s *Suite) TestWrapperRelay(t *utesting.T) {
	t.Log(`This test announces a valid mode 0 wrapper of more than 17 transactions, which
carries no inline metadata. The node must request it, validate it, and re-announce it
unchanged to another peer, which can then fetch it.`)
	s.testRelay(t, eth75, s.takeWrappers(t, direct, 1)...)
}

func (s *Suite) TestWrapperRelayInlineMetadata(t *utesting.T) {
	t.Log(`This test announces a valid single-transaction mode 0 wrapper with its inline
metadata. The node must accept the metadata, request and validate the wrapper, and
re-announce it to another peer with exactly the metadata its transactions give.`)
	s.testRelay(t, eth75, s.takeWrappers(t, inline, 1)...)
}

func (s *Suite) TestWrapperRelayAnyBucket(t *utesting.T) {
	t.Log(`This test announces several valid mode 0 wrappers in one message, whose
transactions together fall into every transaction-hash prefix a node could use to choose
what it aggregates. Admission and relay of direct witnesses must not depend on that
choice: the node must request, validate and relay all of them.`)
	fixtures := s.takeCovering(t, direct)
	s.testRelay(t, eth75, fixtures...)
}

// testInvalidWrapper serves a wrapper that fails EIP-8288 validation. The node must
// discard it without penalizing the peer, and neither announce nor serve it.
func (s *Suite) testInvalidWrapper(t *utesting.T, version uint, f *wrapperFixture) {
	conns := s.dialWrapperPeers(t, version, 2)
	defer closeConns(conns)
	source, observer := conns[0], conns[1]

	if err := relayWrappers(source, f); err != nil {
		t.Fatal(err)
	}
	if err := source.expectConnected(quietPeriod); err != nil {
		t.Fatalf("node penalized the peer for an invalid wrapper: %v", err)
	}
	if err := observer.expectNoAnnouncement(quietPeriod, f.hash()); err != nil {
		t.Fatalf("node relayed an invalid wrapper: %v", err)
	}
	resp, err := observer.fetchWrappers(f.hash())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Entries) != 0 {
		t.Fatal("node served an invalid wrapper")
	}
}

func (s *Suite) TestWrapperInvalidWitness(t *utesting.T) {
	t.Log(`This test serves a well-formed mode 0 wrapper whose witness does not verify.
Proof failures make a wrapper invalid, not malformed: the node must discard it without
penalty or disconnection, and must not relay it.`)
	s.testInvalidWrapper(t, eth75, s.takeWrappers(t, direct, 1)[0].withCorruptWitness())
}

func (s *Suite) TestWrapperUnknownScheme(t *utesting.T) {
	t.Log(`This test serves a structurally valid mode 0 wrapper whose dependency uses an
unassigned scheme identifier. The transport must decode it, and EIP-8288 validation must
reject it: the node must discard it without penalty and must not relay it.`)
	s.testInvalidWrapper(t, eth75, s.takeWrappers(t, unknownScheme, 1)[0])
}

func (s *Suite) TestWrapperUnsupportedMode(t *utesting.T) {
	t.Log(`This test announces wrappers of modes eth/75 does not support (1, and an
unassigned one), alone and mixed with a mode 0 wrapper. The node must ignore those entries
without penalty and request only the mode 0 wrapper.`)
	conn, err := s.dialWrapperPeer(eth75)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	f := s.takeWrappers(t, direct, 1)[0]
	unsupported := []wrapperAnnouncement{
		{Mode: wrapperModeRecursive, Length: 1000, Hash: common.Hash{0x01}, Meta: emptyList},
		{Mode: 0x80, Length: 1000, Hash: common.Hash{0x02}, Meta: emptyList},
	}
	if err := conn.writeAnnouncements(unsupported...); err != nil {
		t.Fatal(err)
	}
	if err := conn.expectNoRequest(quietPeriod, unsupported[0].Hash, unsupported[1].Hash); err != nil {
		t.Fatal(err)
	}
	mixed := append(slices.Clone(unsupported), f.announcement())
	if err := conn.writeAnnouncements(mixed...); err != nil {
		t.Fatal(err)
	}
	all := []common.Hash{unsupported[0].Hash, unsupported[1].Hash, f.hash()}
	req, err := conn.expectRequest(f.hash(), all, wrapperTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(req.hashes, unsupported[0].Hash) || slices.Contains(req.hashes, unsupported[1].Hash) {
		t.Fatal("node requested a wrapper of an unsupported mode")
	}
	if err := conn.serve(req, f); err != nil {
		t.Fatal(err)
	}
	if err := conn.expectConnected(quietPeriod); err != nil {
		t.Fatal(err)
	}
}

// testAnnouncementMismatch announces a wrapper with the given false announcement and
// serves the true wrapper when the node requests it. The node must discard the wrapper
// before verifying it and disconnect the peer; another peer can still provide it.
func (s *Suite) testAnnouncementMismatch(t *utesting.T, version uint, f *wrapperFixture, ann wrapperAnnouncement) {
	conns := s.dialWrapperPeers(t, version, 2)
	defer closeConns(conns)
	liar, honest := conns[0], conns[1]

	if err := liar.writeAnnouncements(ann); err != nil {
		t.Fatal(err)
	}
	if err := serveRequests(liar, f); err != nil {
		t.Fatalf("node did not fetch the wrapper: %v", err)
	}
	if err := liar.expectDisconnect(wrapperTimeout); err != nil {
		t.Fatalf("announcement mismatch: %v", err)
	}
	// One peer's false claims must not cause the discard of the wrapper served by another.
	if err := relayWrappers(honest, f); err != nil {
		t.Fatalf("node did not fetch the wrapper from an honest peer: %v", err)
	}
	if err := honest.expectConnected(quietPeriod); err != nil {
		t.Fatal(err)
	}
}

func (s *Suite) TestWrapperMetadataMismatch(t *utesting.T) {
	t.Log(`This test announces a single-transaction wrapper with well-formed inline
metadata whose frame gas differs from its transaction. Once the wrapper arrives, the node
must discard it before proof verification and disconnect the peer. The same wrapper
announced correctly by another peer must still be fetched.`)
	f := s.takeWrappers(t, inline, 1)[0]
	ann := f.announcement()
	var m txsMeta
	if err := rlp.DecodeBytes(ann.Meta, &m); err != nil {
		t.Fatal(err)
	}
	m.Gas[0]++
	ann.Meta, _ = rlp.EncodeToBytes(&m)
	s.testAnnouncementMismatch(t, eth75, f, ann)
}

func (s *Suite) TestWrapperMetadataMissing(t *utesting.T) {
	t.Log(`This test announces a single-transaction mode 0 wrapper without the inline
metadata such a wrapper requires. An empty txs-meta is checked once the wrapper arrives:
the node must then discard it and disconnect the peer.`)
	f := s.takeWrappers(t, inline, 1)[0]
	ann := f.announcement()
	ann.Meta = emptyList
	s.testAnnouncementMismatch(t, eth75, f, ann)
}

func (s *Suite) TestWrapperLengthMismatch(t *utesting.T) {
	t.Log(`This test announces a wrapper with a length one byte longer than its encoding.
When the node receives the wrapper it must discard it and disconnect the peer.`)
	f := s.takeWrappers(t, direct, 1)[0]
	ann := f.announcement()
	ann.Length++
	s.testAnnouncementMismatch(t, eth75, f, ann)
}

// testViolation sends one message and checks that the node disconnects.
func (s *Suite) testViolation(t *utesting.T, version uint, name string, code uint64, payload []byte) {
	conn, err := s.dialWrapperPeer(version)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.writeWrapperMsg(code, payload); err != nil {
		t.Fatalf("%s: write failed: %v", name, err)
	}
	if err := conn.expectDisconnect(wrapperTimeout); err != nil {
		t.Errorf("%s: %v", name, err)
	}
}

func randomAnnouncements(n int) []wrapperAnnouncement {
	anns := make([]wrapperAnnouncement, n)
	for i := range anns {
		var h common.Hash
		rand.Read(h[:])
		anns[i] = wrapperAnnouncement{Mode: wrapperModeDirect, Length: 1000, Hash: h, Meta: emptyList}
	}
	slices.SortFunc(anns, func(a, b wrapperAnnouncement) int { return a.Hash.Cmp(b.Hash) })
	return anns
}

func (s *Suite) TestWrapperAnnouncementLimits(t *utesting.T) {
	t.Log(`This test sends NewProofWrapperHashes messages that violate the item-count,
byte-length and ordering rules: no entries, 65 entries, unordered hashes, a length of zero
or above MAX_WRAPPER_BYTES, an extra field, and a message above 2 MiB. Each must cause
disconnection.`)
	s.testAnnouncementLimits(t, eth75)
}

func (s *Suite) testAnnouncementLimits(t *utesting.T, version uint) {
	var codec = wrapperCodecs[version]
	encode := func(anns []wrapperAnnouncement) []byte {
		enc, err := codec.encodeAnnouncements(anns)
		if err != nil {
			t.Fatal(err)
		}
		return enc
	}
	s.testViolation(t, version, "empty announcement", newProofWrapperHashesMsg, encode(nil))
	s.testViolation(t, version, "65 entries", newProofWrapperHashesMsg, encode(randomAnnouncements(maxWrapperAnnouncements+1)))

	unordered := randomAnnouncements(2)
	unordered[0], unordered[1] = unordered[1], unordered[0]
	s.testViolation(t, version, "unordered hashes", newProofWrapperHashesMsg, encode(unordered))

	zero := randomAnnouncements(1)
	zero[0].Length = 0
	s.testViolation(t, version, "zero length", newProofWrapperHashesMsg, encode(zero))

	oversize := randomAnnouncements(1)
	oversize[0].Length = maxWrapperBytes + 1
	s.testViolation(t, version, "length above MAX_WRAPPER_BYTES", newProofWrapperHashesMsg, encode(oversize))

	// An extra trailing field in an entry.
	var entry []rlp.RawValue
	if err := rlp.DecodeBytes(encode(randomAnnouncements(1)), &entry); err != nil {
		t.Fatal(err)
	}
	var fields []rlp.RawValue
	if err := rlp.DecodeBytes(entry[0], &fields); err != nil {
		t.Fatal(err)
	}
	fields = append(fields, rlp.RawValue{0x80})
	extra, _ := rlp.EncodeToBytes([][]rlp.RawValue{fields})
	s.testViolation(t, version, "extra announcement field", newProofWrapperHashesMsg, extra)

	// An entry of an unsupported mode is ignored, but the payload bound applies first.
	large := randomAnnouncements(1)
	large[0].Mode = 0x80
	large[0].Meta, _ = rlp.EncodeToBytes([][]byte{make([]byte, maxAnnouncementBytes)})
	s.testViolation(t, version, "announcement above 2 MiB", newProofWrapperHashesMsg, encode(large))
}

func (s *Suite) TestWrapperRequestLimits(t *utesting.T) {
	t.Log(`This test checks GetProofWrappers bounds. A request for a hash the node never
announced must get an empty response without penalty. Requests for no hashes or more than
32 hashes violate the item-count rule and must cause disconnection.`)
	s.testRequestLimits(t, eth75)
}

func (s *Suite) testRequestLimits(t *utesting.T, version uint) {
	conn, err := s.dialWrapperPeer(version)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	resp, err := conn.fetchWrappers(common.Hash{0xde, 0xad})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Entries) != 0 {
		t.Fatal("node served a wrapper it never announced")
	}
	if err := conn.expectConnected(quietPeriod); err != nil {
		t.Fatal(err)
	}

	codec := wrapperCodecs[version]
	encode := func(hashes []common.Hash) []byte {
		enc, err := rlp.EncodeToBytes(codec.encodeRequest(1, hashes))
		if err != nil {
			t.Fatal(err)
		}
		return enc
	}
	s.testViolation(t, version, "request for no hashes", getProofWrappersMsg, encode([]common.Hash{}))
	many := make([]common.Hash, maxWrapperRequest+1)
	for i := range many {
		many[i] = common.Hash{byte(i + 1)}
	}
	s.testViolation(t, version, "request for 33 hashes", getProofWrappersMsg, encode(many))
}

func (s *Suite) TestWrapperExchangeBeforeFork(t *utesting.T) {
	t.Log(`This test runs against a node whose chain schedules EIP-8288 in the future.
A node may already negotiate eth/75, but must not exchange wrappers before the fork: it
must ignore wrapper announcements and requests without penalty and send no wrapper
messages.`)
	conn, err := s.dialWrapperPeer(eth75)
	if err != nil {
		t.Logf("eth/75 was not negotiated before the fork: %v", err)
		return
	}
	defer conn.Close()

	f := s.takeWrappers(t, direct, 1)[0]
	if err := conn.announce(f); err != nil {
		t.Fatal(err)
	}
	if err := conn.requestWrappers(1, f.hash()); err != nil {
		t.Fatal(err)
	}
	msg, err := conn.readWrapperMsg(time.Now().Add(quietPeriod))
	switch {
	case err == nil:
		t.Fatalf("node sent wrapper message %#x before the fork", msg.code)
	case !isTimeout(err):
		t.Fatalf("node disconnected: %v", err)
	}
	if err := conn.expectConnected(0); err != nil {
		t.Fatal(err)
	}
}
