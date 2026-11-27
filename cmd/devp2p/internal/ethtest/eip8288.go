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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

// EIP-8288 proof wrappers are exchanged by eth/75 (EIP-8437) and later versions. go-ethereum
// does not implement these versions, so their messages are defined here rather than in the
// eth package.

const (
	eth75 = 75

	newProofWrapperHashesMsg = 0x18
	getProofWrappersMsg      = 0x19
	proofWrappersMsg         = 0x1a

	wrapperModeDirect    = 0
	wrapperModeRecursive = 1

	maxWrapperBytes         = 1 << 21
	maxAnnouncementBytes    = 1 << 18
	maxWrapperAnnouncements = 64
	maxWrapperRequest       = 32
	maxInlineTransactions   = 17

	// eip8288Dir holds the chain and wrapper fixtures, relative to the chain directory.
	eip8288Dir = "eip8288"
)

var (
	// wrapperTimeout bounds how long the node may take to request, validate and relay a
	// wrapper. Validation verifies post-quantum witnesses, which takes longer than ordinary
	// transaction checks.
	wrapperTimeout = 20 * time.Second
	// quietPeriod is how long a test waits to see that the node does not react.
	quietPeriod = 5 * time.Second
)

// wrapperAnnouncement is one NewProofWrapperHashes entry. DepsHash and TxsHash are the
// eth/76 coverage hints. Meta is the txs-meta item, the empty list when absent.
type wrapperAnnouncement struct {
	Mode     uint64
	Length   uint64
	Hash     common.Hash
	DepsHash common.Hash
	TxsHash  common.Hash
	Meta     rlp.RawValue
}

// announcement75 is the eth/75 entry [mode, length, hash, txs-meta].
type announcement75 struct {
	Mode   uint64
	Length uint64
	Hash   common.Hash
	Meta   rlp.RawValue
}

// getProofWrappers75 is the eth/75 GetProofWrappers message [request-id, [hash, ...]].
type getProofWrappers75 struct {
	RequestId uint64
	Hashes    []common.Hash
}

// proofWrappersPacket is the ProofWrappers message [request-id, [[hash, wrapper], ...]].
type proofWrappersPacket struct {
	RequestId uint64
	Entries   []proofWrapperEntry
}

type proofWrapperEntry struct {
	Hash    common.Hash
	Wrapper rlp.RawValue
}

// txsMeta is the inline transaction metadata of a mode 0 wrapper of at most 17
// transactions, in wrapper order.
type txsMeta struct {
	Sizes   []uint64
	Hashes  []common.Hash
	Sources []common.Address
	Fees    [][]*big.Int
	Gas     []uint64
}

var emptyList = rlp.RawValue{0xc0}

// wrapperBody is the canonical wrapper [transactions, mode, content].
type wrapperBody struct {
	Transactions [][]byte
	Mode         uint64
	Content      rlp.RawValue
}

// directContent is the content [deps, proofs] of a mode 0 wrapper.
type directContent struct {
	Deps   [][]byte
	Proofs [][]byte
}

// wrapperFixture is a pre-generated wrapper with the metadata of its transactions, in
// wrapper order. Valid wrappers cannot be built here: their dependency witnesses are
// post-quantum signatures and recursive proofs, which go-ethereum does not implement.
type wrapperFixture struct {
	Wrapper      hexutil.Bytes `json:"wrapper"`
	Transactions []fixtureTx   `json:"transactions"`
}

type fixtureTx struct {
	Size   uint64         `json:"size"`
	Hash   common.Hash    `json:"hash"`
	Source common.Address `json:"source"`
	Fees   []*hexutil.Big `json:"fees"`
	Gas    uint64         `json:"gas"`
}

// eip8288Fixtures is testdata/eip8288/fixtures.json; see mkfixtures.sh.
type eip8288Fixtures struct {
	// Direct are mode 0 wrappers of 18 transactions sharing one dependency, announced
	// without inline metadata.
	Direct []*wrapperFixture `json:"direct"`
	// Inline are mode 0 wrappers of one transaction, announced with inline metadata.
	Inline []*wrapperFixture `json:"inline"`
	// UnknownScheme are mode 0 wrappers whose dependency uses an unassigned scheme.
	UnknownScheme []*wrapperFixture `json:"unknownScheme"`
	// Recursive are mode 1 wrappers whose recursive proof is valid.
	Recursive []*wrapperFixture `json:"recursive"`
}

// fixturePool hands out each fixture once per run, so that every test announces
// transactions the node does not hold yet.
type fixturePool struct {
	mu       sync.Mutex
	err      error
	fixtures *eip8288Fixtures
	used     map[*wrapperFixture]bool
}

func loadFixtures(chainDir string) (*eip8288Fixtures, error) {
	data, err := os.ReadFile(filepath.Join(chainDir, eip8288Dir, "fixtures.json"))
	if err != nil {
		return nil, err
	}
	var f eip8288Fixtures
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("invalid wrapper fixtures: %v", err)
	}
	return &f, nil
}

// take returns n unused fixtures of the given kind.
func (p *fixturePool) take(chainDir string, kind func(*eip8288Fixtures) []*wrapperFixture, n int) ([]*wrapperFixture, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.fixtures == nil && p.err == nil {
		p.fixtures, p.err = loadFixtures(chainDir)
		p.used = make(map[*wrapperFixture]bool)
	}
	if p.err != nil {
		return nil, p.err
	}
	var taken []*wrapperFixture
	for _, f := range kind(p.fixtures) {
		if len(taken) == n {
			break
		}
		if !p.used[f] {
			p.used[f] = true
			taken = append(taken, f)
		}
	}
	if len(taken) < n {
		return nil, errors.New("wrapper fixtures exhausted; regenerate them with more wrappers")
	}
	return taken, nil
}

func (f *wrapperFixture) hash() common.Hash { return crypto.Keccak256Hash(f.Wrapper) }

func (f *wrapperFixture) body() (*wrapperBody, error) {
	var b wrapperBody
	if err := rlp.DecodeBytes(f.Wrapper, &b); err != nil {
		return nil, fmt.Errorf("invalid wrapper fixture: %v", err)
	}
	return &b, nil
}

func (f *wrapperFixture) mode() uint64 {
	b, err := f.body()
	if err != nil {
		panic(err)
	}
	return b.Mode
}

func (f *wrapperFixture) txHashes() []common.Hash {
	hashes := make([]common.Hash, len(f.Transactions))
	for i, tx := range f.Transactions {
		hashes[i] = tx.Hash
	}
	return hashes
}

// deps returns the wrapper's 96-byte dependency encodings. Both modes start their content
// with the dependency list.
func (f *wrapperFixture) deps() [][]byte {
	b, err := f.body()
	if err != nil {
		panic(err)
	}
	var content struct {
		Deps [][]byte
		Rest []rlp.RawValue `rlp:"tail"`
	}
	if err := rlp.DecodeBytes(b.Content, &content); err != nil {
		panic(fmt.Errorf("invalid wrapper fixture content: %v", err))
	}
	return content.Deps
}

// depsHash and txsHash are the eth/76 coverage hints.
func (f *wrapperFixture) depsHash() common.Hash { return crypto.Keccak256Hash(f.deps()...) }

func (f *wrapperFixture) txsHash() common.Hash { return hashesDigest(f.txHashes()) }

func hashesDigest(hashes []common.Hash) common.Hash {
	data := make([]byte, 0, len(hashes)*common.HashLength)
	for _, h := range hashes {
		data = append(data, h[:]...)
	}
	return crypto.Keccak256Hash(data)
}

// inlineMeta returns the txs-meta its announcements must carry: the metadata of every
// transaction of a mode 0 wrapper of at most 17 transactions, otherwise the empty list.
func (f *wrapperFixture) inlineMeta() rlp.RawValue {
	if f.mode() != wrapperModeDirect || len(f.Transactions) > maxInlineTransactions {
		return emptyList
	}
	return f.meta()
}

// meta returns the metadata of every transaction of the wrapper.
func (f *wrapperFixture) meta() rlp.RawValue {
	var m txsMeta
	for _, tx := range f.Transactions {
		m.Sizes = append(m.Sizes, tx.Size)
		m.Hashes = append(m.Hashes, tx.Hash)
		m.Sources = append(m.Sources, tx.Source)
		fees := make([]*big.Int, len(tx.Fees))
		for i, fee := range tx.Fees {
			fees[i] = fee.ToInt()
		}
		m.Fees = append(m.Fees, fees)
		m.Gas = append(m.Gas, tx.Gas)
	}
	enc, err := rlp.EncodeToBytes(&m)
	if err != nil {
		panic(err)
	}
	return enc
}

// announcement describes the fixture as its holder announces it. The hints are ignored by
// eth/75 encodings.
func (f *wrapperFixture) announcement() wrapperAnnouncement {
	return wrapperAnnouncement{
		Mode:     f.mode(),
		Length:   uint64(len(f.Wrapper)),
		Hash:     f.hash(),
		DepsHash: f.depsHash(),
		TxsHash:  f.txsHash(),
		Meta:     f.inlineMeta(),
	}
}

// withContent returns a wrapper of the same transactions with another mode and content.
// Its transaction metadata is unchanged.
func (f *wrapperFixture) withContent(mode uint64, content any) *wrapperFixture {
	b, err := f.body()
	if err != nil {
		panic(err)
	}
	if b.Content, err = rlp.EncodeToBytes(content); err != nil {
		panic(err)
	}
	b.Mode = mode
	enc, err := rlp.EncodeToBytes(b)
	if err != nil {
		panic(err)
	}
	return &wrapperFixture{Wrapper: enc, Transactions: f.Transactions}
}

// withCorruptWitness returns the mode 0 wrapper with one byte of its first witness
// flipped: well-formed, but its witness no longer verifies.
func (f *wrapperFixture) withCorruptWitness() *wrapperFixture {
	b, err := f.body()
	if err != nil {
		panic(err)
	}
	var content directContent
	if err := rlp.DecodeBytes(b.Content, &content); err != nil {
		panic(err)
	}
	content.Proofs[0] = bytes.Clone(content.Proofs[0])
	content.Proofs[0][len(content.Proofs[0])/2] ^= 0xff
	return f.withContent(wrapperModeDirect, content)
}

// dialWrapperPeer connects to the node offering only the given eth version and completes
// the status exchange. The node's chain is not ethtest's generated chain, since that chain
// predates EIP-8288; the connection adopts the network, genesis, fork ID and head the node
// reports.
func (s *Suite) dialWrapperPeer(version uint) (*Conn, error) {
	conn, err := s.dial()
	if err != nil {
		return nil, fmt.Errorf("dial failed: %v", err)
	}
	conn.caps = []p2p.Cap{{Name: "eth", Version: version}}
	conn.ourHighestProtoVersion = version
	if err := conn.handshake(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake failed: %v", err)
	}
	status, err := conn.adoptStatus()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("status exchange failed: %v", err)
	}
	s.headOnce.Do(func() { s.headErr = s.engine.selectHead(status.LatestBlockHash) })
	if s.headErr != nil {
		conn.Close()
		return nil, fmt.Errorf("forkchoice update failed: %v", s.headErr)
	}
	return conn, nil
}

// adoptStatus reads the node's Status and answers with the same chain description.
func (c *Conn) adoptStatus() (*eth.StatusPacket, error) {
	for {
		code, data, err := c.Read()
		if err != nil {
			return nil, err
		}
		switch code {
		case protoOffset(ethProto) + eth.StatusMsg:
			var status eth.StatusPacket
			if err := rlp.DecodeBytes(data, &status); err != nil {
				return nil, fmt.Errorf("error decoding status packet: %v", err)
			}
			if uint(status.ProtocolVersion) != c.negotiatedProtoVersion {
				return nil, fmt.Errorf("wrong protocol version in status: have %d, want %d", status.ProtocolVersion, c.negotiatedProtoVersion)
			}
			return &status, c.Write(ethProto, eth.StatusMsg, &status)
		case discMsg:
			return nil, errDisc
		case pingMsg:
			c.Write(baseProto, pongMsg, []byte{})
		default:
			return nil, fmt.Errorf("bad status message: code %d", code)
		}
	}
}

// wrapperMsg is a received wrapper message, still encoded.
type wrapperMsg struct {
	code uint64
	data []byte
}

// readWrapperMsg returns the next wrapper message the node sends before the deadline,
// answering pings and skipping other messages.
func (c *Conn) readWrapperMsg(deadline time.Time) (*wrapperMsg, error) {
	c.SetReadDeadline(deadline)
	for {
		code, data, _, err := c.Conn.Read()
		if err != nil {
			return nil, err
		}
		switch {
		case code == discMsg:
			return nil, errDisc
		case code == pingMsg:
			c.Write(baseProto, pongMsg, []byte{})
		case code >= protoOffset(ethProto)+newProofWrapperHashesMsg:
			return &wrapperMsg{code: code - protoOffset(ethProto), data: data}, nil
		}
	}
}

// writeWrapperMsg sends a wrapper message, encoding msg unless it is already encoded.
func (c *Conn) writeWrapperMsg(code uint64, msg any) error {
	c.SetWriteDeadline(time.Now().Add(timeout))
	payload, ok := msg.([]byte)
	if !ok {
		var err error
		if payload, err = rlp.EncodeToBytes(msg); err != nil {
			return err
		}
	}
	_, err := c.Conn.Write(protoOffset(ethProto)+code, payload)
	return err
}

// wrapperCodec encodes and decodes the wrapper messages whose shape depends on the
// negotiated version. Shapes follow the version, not the wrapper mode.
type wrapperCodec interface {
	encodeAnnouncements(anns []wrapperAnnouncement) ([]byte, error)
	decodeAnnouncements(data []byte) ([]wrapperAnnouncement, error)
	encodeRequest(id uint64, hashes []common.Hash) any
	decodeRequest(data []byte) (*wrapperRequest, error)
}

var wrapperCodecs = map[uint]wrapperCodec{
	eth75: codec75{},
}

func (c *Conn) codec() wrapperCodec { return wrapperCodecs[c.negotiatedProtoVersion] }

// codec75 is eth/75: four-field announcements and two-field requests.
type codec75 struct{}

func (codec75) encodeAnnouncements(anns []wrapperAnnouncement) ([]byte, error) {
	entries := make([]announcement75, len(anns))
	for i, a := range anns {
		entries[i] = announcement75{a.Mode, a.Length, a.Hash, a.Meta}
	}
	return rlp.EncodeToBytes(entries)
}

func (codec75) decodeAnnouncements(data []byte) ([]wrapperAnnouncement, error) {
	var entries []announcement75
	if err := rlp.DecodeBytes(data, &entries); err != nil {
		return nil, fmt.Errorf("invalid eth/75 announcement: %v", err)
	}
	anns := make([]wrapperAnnouncement, len(entries))
	for i, e := range entries {
		anns[i] = wrapperAnnouncement{Mode: e.Mode, Length: e.Length, Hash: e.Hash, Meta: e.Meta}
	}
	return anns, nil
}

func (codec75) encodeRequest(id uint64, hashes []common.Hash) any {
	return &getProofWrappers75{RequestId: id, Hashes: hashes}
}

func (codec75) decodeRequest(data []byte) (*wrapperRequest, error) {
	var p getProofWrappers75
	if err := rlp.DecodeBytes(data, &p); err != nil {
		return nil, fmt.Errorf("invalid eth/75 GetProofWrappers: %v", err)
	}
	return &wrapperRequest{id: p.RequestId, hashes: p.Hashes}, nil
}

// announce sends a NewProofWrapperHashes message announcing the given wrappers.
func (c *Conn) announce(fixtures ...*wrapperFixture) error {
	anns := make([]wrapperAnnouncement, len(fixtures))
	for i, f := range fixtures {
		anns[i] = f.announcement()
	}
	return c.writeAnnouncements(anns...)
}

// writeAnnouncements sends the entries as one NewProofWrapperHashes message, in ascending
// hash order.
func (c *Conn) writeAnnouncements(anns ...wrapperAnnouncement) error {
	anns = slices.Clone(anns)
	slices.SortFunc(anns, func(a, b wrapperAnnouncement) int { return a.Hash.Cmp(b.Hash) })
	enc, err := c.codec().encodeAnnouncements(anns)
	if err != nil {
		return err
	}
	return c.writeWrapperMsg(newProofWrapperHashesMsg, enc)
}

// decodeAnnouncements decodes a NewProofWrapperHashes message strictly: 1 to 64 entries
// of the negotiated version's shape in strictly ascending hash order, within the byte
// bound, with valid inline metadata.
func (c *Conn) decodeAnnouncements(data []byte) ([]wrapperAnnouncement, error) {
	if len(data) > maxAnnouncementBytes {
		return nil, fmt.Errorf("announcement of %d bytes exceeds %d", len(data), maxAnnouncementBytes)
	}
	anns, err := c.codec().decodeAnnouncements(data)
	if err != nil {
		return nil, err
	}
	if len(anns) == 0 || len(anns) > maxWrapperAnnouncements {
		return nil, fmt.Errorf("announcement of %d entries", len(anns))
	}
	for i, a := range anns {
		if i > 0 && anns[i-1].Hash.Cmp(a.Hash) >= 0 {
			return nil, errors.New("announcement hashes are not strictly ascending")
		}
		if a.Length == 0 || a.Length > maxWrapperBytes {
			return nil, fmt.Errorf("announced length %d out of bounds", a.Length)
		}
		if a.Mode == wrapperModeDirect && !bytes.Equal(a.Meta, emptyList) {
			if err := checkMeta(a.Meta, a.Length); err != nil {
				return nil, fmt.Errorf("announcement of %v: %v", a.Hash, err)
			}
		}
	}
	return anns, nil
}

// checkMeta applies the arrival checks of inline metadata.
func checkMeta(enc rlp.RawValue, length uint64) error {
	var m txsMeta
	if err := rlp.DecodeBytes(enc, &m); err != nil {
		return fmt.Errorf("malformed txs-meta: %v", err)
	}
	n := len(m.Hashes)
	if n == 0 || n > maxInlineTransactions || len(m.Sizes) != n || len(m.Sources) != n || len(m.Fees) != n || len(m.Gas) != n {
		return errors.New("txs-meta fields describe different or invalid transaction counts")
	}
	var total uint64
	for i, size := range m.Sizes {
		if size == 0 {
			return errors.New("txs-meta size is zero")
		}
		total += size
		if i > 0 && m.Hashes[i-1].Cmp(m.Hashes[i]) >= 0 {
			return errors.New("txs-meta hashes are not strictly ascending")
		}
	}
	if total >= length {
		return errors.New("txs-meta sizes do not fit the announced length")
	}
	for _, fees := range m.Fees {
		for _, fee := range fees {
			if fee.BitLen() > 256 {
				return errors.New("txs-meta fee exceeds 256 bits")
			}
		}
	}
	return nil
}

// expectAnnouncement waits for the node to announce the wrapper with the given hash and
// returns its entry.
func (c *Conn) expectAnnouncement(hash common.Hash, wait time.Duration) (*wrapperAnnouncement, error) {
	anns, err := c.expectAnnouncements(wait, hash)
	if err != nil {
		return nil, err
	}
	return anns[hash], nil
}

// expectAnnouncements waits for the node to announce all wrappers with the given hashes
// and returns their entries. Other announcements are checked for validity and skipped.
func (c *Conn) expectAnnouncements(wait time.Duration, hashes ...common.Hash) (map[common.Hash]*wrapperAnnouncement, error) {
	deadline := time.Now().Add(wait)
	found := make(map[common.Hash]*wrapperAnnouncement)
	for len(found) < len(hashes) {
		msg, err := c.readWrapperMsg(deadline)
		if err != nil {
			for _, h := range hashes {
				if found[h] == nil {
					return nil, fmt.Errorf("no announcement of %v: %v", h, err)
				}
			}
		}
		if msg.code != newProofWrapperHashesMsg {
			continue
		}
		anns, err := c.decodeAnnouncements(msg.data)
		if err != nil {
			return nil, fmt.Errorf("node sent an invalid announcement: %v", err)
		}
		for i := range anns {
			if slices.Contains(hashes, anns[i].Hash) {
				found[anns[i].Hash] = &anns[i]
			}
		}
	}
	return found, nil
}

// expectNoAnnouncement checks the node does not announce any of the hashes for the given
// period.
func (c *Conn) expectNoAnnouncement(wait time.Duration, hashes ...common.Hash) error {
	deadline := time.Now().Add(wait)
	for {
		msg, err := c.readWrapperMsg(deadline)
		if isTimeout(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if msg.code != newProofWrapperHashesMsg {
			continue
		}
		anns, err := c.decodeAnnouncements(msg.data)
		if err != nil {
			return fmt.Errorf("node sent an invalid announcement: %v", err)
		}
		for _, a := range anns {
			if slices.Contains(hashes, a.Hash) {
				return fmt.Errorf("node announced %v", a.Hash)
			}
		}
	}
}

// wrapperRequest is a received GetProofWrappers message.
type wrapperRequest struct {
	id      uint64
	compact bool
	hashes  []common.Hash
}

func (c *Conn) decodeWrapperRequest(data []byte) (*wrapperRequest, error) {
	req, err := c.codec().decodeRequest(data)
	if err != nil {
		return nil, err
	}
	if len(req.hashes) == 0 || len(req.hashes) > maxWrapperRequest {
		return nil, fmt.Errorf("GetProofWrappers for %d hashes", len(req.hashes))
	}
	for i, h := range req.hashes {
		if slices.Contains(req.hashes[:i], h) {
			return nil, errors.New("GetProofWrappers repeats a hash")
		}
	}
	return req, nil
}

// expectRequest waits for the node to request the wrapper with the given hash. Every
// request must name only wrappers announced on the connection.
func (c *Conn) expectRequest(hash common.Hash, announced []common.Hash, wait time.Duration) (*wrapperRequest, error) {
	deadline := time.Now().Add(wait)
	for {
		msg, err := c.readWrapperMsg(deadline)
		if err != nil {
			return nil, fmt.Errorf("no request for %v: %v", hash, err)
		}
		if msg.code != getProofWrappersMsg {
			continue
		}
		req, err := c.decodeWrapperRequest(msg.data)
		if err != nil {
			return nil, fmt.Errorf("node sent an invalid request: %v", err)
		}
		for _, h := range req.hashes {
			if !slices.Contains(announced, h) {
				return nil, fmt.Errorf("node requested %v, which was not announced to it", h)
			}
		}
		if slices.Contains(req.hashes, hash) {
			return req, nil
		}
	}
}

var errUnexpectedRequest = errors.New("node requested")

// expectNoRequest checks the node requests none of the hashes for the given period.
func (c *Conn) expectNoRequest(wait time.Duration, hashes ...common.Hash) error {
	deadline := time.Now().Add(wait)
	for {
		msg, err := c.readWrapperMsg(deadline)
		if isTimeout(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if msg.code != getProofWrappersMsg {
			continue
		}
		req, err := c.decodeWrapperRequest(msg.data)
		if err != nil {
			return fmt.Errorf("node sent an invalid request: %v", err)
		}
		for _, h := range req.hashes {
			if slices.Contains(hashes, h) {
				return fmt.Errorf("%w %v", errUnexpectedRequest, h)
			}
		}
	}
}

// serve answers a request with the requested fixtures, in request order.
func (c *Conn) serve(req *wrapperRequest, fixtures ...*wrapperFixture) error {
	resp := proofWrappersPacket{RequestId: req.id, Entries: []proofWrapperEntry{}}
	for _, h := range req.hashes {
		for _, f := range fixtures {
			if f.hash() == h {
				resp.Entries = append(resp.Entries, proofWrapperEntry{Hash: h, Wrapper: rlp.RawValue(f.Wrapper)})
			}
		}
	}
	return c.writeWrapperMsg(proofWrappersMsg, &resp)
}

// requestWrappers sends a GetProofWrappers message for complete wrappers.
func (c *Conn) requestWrappers(id uint64, hashes ...common.Hash) error {
	return c.writeWrapperMsg(getProofWrappersMsg, c.codec().encodeRequest(id, hashes))
}

// fetchWrappers requests complete wrappers from the node and returns its response.
func (c *Conn) fetchWrappers(hashes ...common.Hash) (*proofWrappersPacket, error) {
	id := uint64(time.Now().UnixNano())
	if err := c.requestWrappers(id, hashes...); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wrapperTimeout)
	for {
		msg, err := c.readWrapperMsg(deadline)
		if err != nil {
			return nil, fmt.Errorf("no ProofWrappers response: %v", err)
		}
		if msg.code != proofWrappersMsg {
			continue
		}
		var resp proofWrappersPacket
		if err := rlp.DecodeBytes(msg.data, &resp); err != nil {
			return nil, fmt.Errorf("invalid ProofWrappers: %v", err)
		}
		if resp.RequestId != id {
			return nil, fmt.Errorf("ProofWrappers for unknown request %d", resp.RequestId)
		}
		for _, e := range resp.Entries {
			if crypto.Keccak256Hash(e.Wrapper) != e.Hash {
				return nil, fmt.Errorf("served wrapper does not hash to %v", e.Hash)
			}
			if len(e.Wrapper) > maxWrapperBytes {
				return nil, fmt.Errorf("served wrapper of %d bytes exceeds %d", len(e.Wrapper), maxWrapperBytes)
			}
		}
		return &resp, nil
	}
}

// fetchWrapper fetches one wrapper the node announced and checks it against the
// announcement.
func (c *Conn) fetchWrapper(ann *wrapperAnnouncement) ([]byte, error) {
	resp, err := c.fetchWrappers(ann.Hash)
	if err != nil {
		return nil, err
	}
	if len(resp.Entries) != 1 || resp.Entries[0].Hash != ann.Hash {
		return nil, fmt.Errorf("node did not serve %v, which it announced", ann.Hash)
	}
	wrapper := resp.Entries[0].Wrapper
	if uint64(len(wrapper)) != ann.Length {
		return nil, fmt.Errorf("served wrapper has length %d, announced %d", len(wrapper), ann.Length)
	}
	var b wrapperBody
	if err := rlp.DecodeBytes(wrapper, &b); err != nil {
		return nil, fmt.Errorf("served wrapper is malformed: %v", err)
	}
	if b.Mode != ann.Mode {
		return nil, fmt.Errorf("served wrapper has mode %d, announced %d", b.Mode, ann.Mode)
	}
	return wrapper, nil
}

// expectDisconnect waits for the node to drop the connection.
func (c *Conn) expectDisconnect(wait time.Duration) error {
	deadline := time.Now().Add(wait)
	for {
		_, err := c.readWrapperMsg(deadline)
		switch {
		case err == nil:
			continue
		case isTimeout(err):
			return errors.New("node did not disconnect")
		default:
			return nil
		}
	}
}

// expectConnected checks the node keeps the connection open for the given period and
// still answers a ping.
func (c *Conn) expectConnected(wait time.Duration) error {
	deadline := time.Now().Add(wait)
	for {
		_, err := c.readWrapperMsg(deadline)
		if isTimeout(err) {
			break
		}
		if err != nil {
			return fmt.Errorf("node disconnected: %v", err)
		}
	}
	if err := c.Write(baseProto, pingMsg, []byte{}); err != nil {
		return fmt.Errorf("node disconnected: %v", err)
	}
	c.SetReadDeadline(time.Now().Add(timeout))
	for {
		code, _, _, err := c.Conn.Read()
		if err != nil {
			return fmt.Errorf("node disconnected: %v", err)
		}
		switch code {
		case pongMsg:
			return nil
		case discMsg:
			return errDisc
		}
	}
}

func isTimeout(err error) bool {
	var netErr interface{ Timeout() bool }
	return errors.As(err, &netErr) && netErr.Timeout()
}

// relayWrappers announces the fixtures from the given connection, serves the node's
// requests for them, and waits until the node requested all of them.
func relayWrappers(conn *Conn, fixtures ...*wrapperFixture) error {
	if err := conn.announce(fixtures...); err != nil {
		return fmt.Errorf("announce failed: %v", err)
	}
	return serveRequests(conn, fixtures...)
}

// serveRequests serves the node's requests for the fixtures, which the connection
// announced, until the node requested all of them.
func serveRequests(conn *Conn, fixtures ...*wrapperFixture) error {
	announced := make([]common.Hash, len(fixtures))
	for i, f := range fixtures {
		announced[i] = f.hash()
	}
	pending := slices.Clone(announced)
	deadline := time.Now().Add(wrapperTimeout)
	for len(pending) > 0 {
		msg, err := conn.readWrapperMsg(deadline)
		if err != nil {
			return fmt.Errorf("no request for %v: %v", pending[0], err)
		}
		switch msg.code {
		case getProofWrappersMsg:
			req, err := conn.decodeWrapperRequest(msg.data)
			if err != nil {
				return fmt.Errorf("node sent an invalid request: %v", err)
			}
			for _, h := range req.hashes {
				if !slices.Contains(announced, h) {
					return fmt.Errorf("node requested %v, which was not announced to it", h)
				}
			}
			if err := conn.serve(req, fixtures...); err != nil {
				return fmt.Errorf("serve failed: %v", err)
			}
			pending = slices.DeleteFunc(pending, func(h common.Hash) bool { return slices.Contains(req.hashes, h) })
		}
	}
	return nil
}
