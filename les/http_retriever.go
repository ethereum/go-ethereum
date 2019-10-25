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

package les

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// RequestByHTTP sends a HTTP request to les CDN and validate all replies with
// local header. If the passing context is canceled, then function returns.
//
// todo(rjl493456442) the code organization is really ugly, need big refactor.
func (r *BlockRequest) RequestByHTTP(ctx context.Context, url string /* todo auth credential */, db ethdb.Database) error {
	var (
		txs     types.Transactions
		uncles  []*types.Header
		pending int
		errch   = make(chan error)
	)
	// Retrieve our stored header and validate block content against it
	header := rawdb.ReadHeader(db, r.Hash, r.Number)
	if header == nil {
		return errHeaderUnavailable
	}
	// Send a http request if tx set is not empty.
	if header.TxHash != types.EmptyRootHash {
		pending += 1
		go func() {
			res, err := httpDo(ctx, fmt.Sprintf("%s/chain/0x%x/transactions", url, r.Hash))
			if err != nil {
				errch <- err
				return
			}
			if err := rlp.DecodeBytes(res, &txs); err != nil {
				errch <- err
				return
			}
			if header.TxHash != types.DeriveSha(txs) {
				errch <- errTxHashMismatch
				return
			}
			errch <- nil
		}()
	}
	// Send a http request if uncle set is not empty.
	if header.UncleHash != types.EmptyUncleHash {
		pending += 1
		go func() {
			res, err := httpDo(ctx, fmt.Sprintf("%s/chain/0x%x/uncles", url, r.Hash))
			if err != nil {
				errch <- err
				return
			}
			if err := rlp.DecodeBytes(res, &uncles); err != nil {
				errch <- err
				return
			}
			if header.UncleHash != types.CalcUncleHash(uncles) {
				errch <- errUncleHashMismatch
				return
			}
			errch <- nil
		}()
	}
	// Wait all retrieve workers and return any error.
	for i := 0; i < pending; i++ {
		if err := <-errch; err != nil {
			return err
		}
	}
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.Txs, r.Uncles = txs, uncles
	return nil
}

// RequestByHTTP sends a HTTP request to les CDN and validate all replies with
// local header. If the passing context is canceled, then function returns.
//
// todo(rjl493456442) the code organization is really ugly, need big refactor.
func (r *ReceiptsRequest) RequestByHTTP(ctx context.Context, url string /* todo auth credential */, db ethdb.Database) error {
	var receipts types.Receipts
	if r.Header == nil {
		r.Header = rawdb.ReadHeader(db, r.Hash, r.Number)
	}
	if r.Header.ReceiptHash != types.EmptyRootHash {
		res, err := httpDo(ctx, fmt.Sprintf("%s/chain/0x%x/receipts", url, r.Hash))
		if err != nil {
			return err
		}
		if err := rlp.DecodeBytes(res, &receipts); err != nil {
			return err
		}
		if r.Header.ReceiptHash != types.DeriveSha(receipts) {
			return errReceiptHashMismatch
		}
	}
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.Receipts = receipts
	return nil
}

// RequestByHTTP sends a HTTP request to les CDN and validate all replies with
// local header. If the passing context is canceled, then function returns.
//
// todo(rjl493456442) the code organization is really ugly, need big refactor.
func (r *CodeRequest) RequestByHTTP(ctx context.Context, url string /* todo auth credential */, db ethdb.Database) error {
	res, err := httpDo(ctx, fmt.Sprintf("%s/state/0x%x?target=%d&limit=%d&barrier=%d", url, r.Hash, 1, 1, 1))
	if err != nil {
		return err
	}
	var nodes [][]byte
	if err := rlp.DecodeBytes(res, &nodes); err != nil {
		return err
	}
	if len(nodes) != 1 {
		return errInvalidEntryCount
	}
	// Verify the data and store if checks out
	if hash := crypto.Keccak256Hash(nodes[0]); r.Hash != hash {
		return errDataHashMismatch
	}
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.Data = nodes[0]
	return nil
}

// RequestByHTTP sends a HTTP request to les CDN and validate all replies with
// local header. If the passing context is canceled, then function returns.
//
// todo(rjl493456442) the code organization is really ugly, need big refactor.
func (r *TrieRequest) RequestByHTTP(ctx context.Context, url string /* todo auth credential */, db ethdb.Database) error {
	res, err := httpDo(ctx, fmt.Sprintf("%s/state/0x%x?target=%d&limit=%d&barrier=%d", url, r.MissNodeHash, 16, 256, 2))
	if err != nil {
		return err
	}
	// Decode all received response.
	var nodes [][]byte
	if err := rlp.DecodeBytes(res, &nodes); err != nil {
		return err
	}
	// Validate the received sub trie.
	proofDb := light.NewNodeSet()
	for _, node := range nodes {
		proofDb.Put(crypto.Keccak256(node), node)
	}
	if err := trie.VerifyTrie(r.MissNodeHash, proofDb, proofDb.KeyCount()); err != nil {
		return err
	}
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.Proof = proofDb // Pass the validation, set it the result.
	return nil
}

// RequestByHTTPV1 sends a HTTP request to les CDN and validate all replies with
// local cht root. If the passing context is canceled, then function returns.
//
// todo(rjl493456442) the code organization is really ugly, need big refactor.
func (r *ChtRequest) RequestByHTTPV1(ctx context.Context, url string /* todo auth credential */, db ethdb.Database) error {
	res, err := httpDo(ctx, fmt.Sprintf("%s/misc/cht/0x%x?&number=%d", url, r.ChtRoot, r.BlockNum))
	if err != nil {
		return err
	}
	var list light.NodeList
	if err := rlp.DecodeBytes(res, &list); err != nil {
		return err
	}
	// Validate and resolve the response
	var (
		proof = list.NodeSet()
		key   = make([]byte, 8)
	)
	reads := &readTraceDB{db: proof}
	binary.BigEndian.PutUint64(key[:], r.BlockNum)
	value, _, err := trie.VerifyProof(r.ChtRoot, key, reads)
	if err != nil {
		return err
	}
	if len(reads.reads) != proof.KeyCount() {
		return errUselessNodes
	}
	// Decode concrete cht node.
	var chtNode light.ChtNode
	if err := rlp.DecodeBytes(value, &chtNode); err != nil {
		return err
	}
	// Request corresponding header now.
	res, err = httpDo(ctx, fmt.Sprintf("%s/chain/0x%x/header", url, chtNode.Hash))
	if err != nil {
		return err
	}
	var header *types.Header
	if err := rlp.DecodeBytes(res, &header); err != nil {
		return err
	}
	if header.Hash() != chtNode.Hash {
		return errInvalidHeader
	}
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.Header, r.Td, r.Proof = header, chtNode.Td, proof
	return nil
}

// RequestByHTTPV2 sends a HTTP request to les CDN and validate all replies with
// local cht root. If the passing context is canceled, then function returns.
//
// todo(rjl493456442) the code organization is really ugly, need big refactor.
func (r *ChtRequest) RequestByHTTP(ctx context.Context, url string /* todo auth credential */, db ethdb.Database) error {
	var (
		key     [8]byte
		chtNode light.ChtNode

		table  = rawdb.NewTable(db, light.ChtTablePrefix)
		triedb = trie.NewDatabase(table)
	)
	binary.BigEndian.PutUint64(key[:], r.BlockNum)
	resolve := func() (error, []*trie.TrieTrace) {
		t, err := trie.NewTraceTrie(r.ChtRoot, triedb, &trie.TraceConfig{RecordHash: true, RecordPath: true})
		if err != nil {
			return err, t.GetTraces()
		}
		blob, err := t.TryGet(key[:])
		if err != nil {
			return err, t.GetTraces()
		}
		err = rlp.DecodeBytes(blob, &chtNode)
		if err != nil {
			return err, nil
		}
		return nil, nil
	}
	for {
		err, traces := resolve()
		if err == nil {
			break
		}
		missError, ok := err.(*trie.MissingNodeError)
		if !ok {
			return err
		}
		levels := light.MaxTileLevels(r.Config.ChtSize)
		var nodeHash common.Hash
		switch {
		case missError.NodeHash == r.ChtRoot:
			nodeHash = r.ChtRoot
		case (16-len(missError.Path))/2 >= levels:
			nodeHash = r.ChtRoot
		case ((16-len(missError.Path))-1)%2 == 0:
			nodeHash = missError.NodeHash
		default:
			nodeHash = traces[len(traces)-1].Hash
		}
		// Send http request to fetch the missing tile.
		res, err := httpDo(ctx, fmt.Sprintf("%s/misc/chtv2/0x%x", url, nodeHash))
		if err != nil {
			return err
		}
		var tile [][]byte
		err = rlp.DecodeBytes(res, &tile)
		if err != nil {
			return err
		}
		// Validate the received sub trie.
		proofDb := light.NewNodeSet()
		for _, node := range tile {
			proofDb.Put(crypto.Keccak256(node), node)
		}
		if err := trie.VerifyTrie(nodeHash, proofDb, proofDb.KeyCount()); err != nil {
			return err
		}
		proofDb.Store(table) // Push all verified nodes into the disk
	}
	if chtNode.Hash == (common.Hash{}) {
		return errors.New("failed to retrieve cht node")
	}
	// Request corresponding header now.
	res, err := httpDo(ctx, fmt.Sprintf("%s/chain/0x%x/header", url, chtNode.Hash))
	if err != nil {
		return err
	}
	var header *types.Header
	if err := rlp.DecodeBytes(res, &header); err != nil {
		return err
	}
	if header.Hash() != chtNode.Hash {
		return errInvalidHeader
	}
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.Header, r.Td = header, chtNode.Td
	return nil
}

// RequestByHTTP sends a HTTP request to les CDN and validate all replies with
// local bloom trie root. If the passing context is canceled, then function returns.
//
// todo(rjl493456442) the code organization is really ugly, need big refactor.
func (r *BloomRequest) RequestByHTTP(ctx context.Context, url string /* todo auth credential */, db ethdb.Database) error {
	// Short circuit if request is empty
	if len(r.SectionList) == 0 {
		return nil
	}
	var (
		errch  = make(chan error)
		bits   = make([][]byte, len(r.SectionList))
		proofs = make([]*light.NodeSet, len(r.SectionList))
	)
	for index, section := range r.SectionList {
		go func(index int, section uint64) {
			res, err := httpDo(ctx, fmt.Sprintf("%s/misc/bloomtrie/0x%x?bit=%d&section=%d", url, r.BloomTrieRoot, r.BitIndex, section))
			if err != nil {
				errch <- err
				return
			}
			var list light.NodeList
			if err := rlp.DecodeBytes(res, &list); err != nil {
				errch <- err
				return
			}
			// Validate and resolve the response
			var (
				proof = list.NodeSet()
				key   = make([]byte, 10)
			)
			reads := &readTraceDB{db: proof}
			binary.BigEndian.PutUint16(key[:2], uint16(r.BitIndex))
			binary.BigEndian.PutUint64(key[2:], section)
			value, _, err := trie.VerifyProof(r.BloomTrieRoot, key, reads)
			if err != nil {
				errch <- err
				return
			}
			if len(reads.reads) != proof.KeyCount() {
				errch <- errUselessNodes
			}
			bits[index], proofs[index] = value, proof
			errch <- nil
		}(index, section)
	}
	// Return error if any of sub request failed.
	for i := 0; i < len(r.SectionList); i++ {
		if err := <-errch; err != nil {
			return err
		}
	}
	// Mix all proofs into single one
	mix := light.NewNodeSet()
	for _, proof := range proofs {
		proof.Store(mix)
	}
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.BloomBits, r.Proofs = bits, mix
	return nil
}

// RequestByHTTP sends a HTTP request to les CDN. Note for txstatus request
// there is no way to meaningfully validate the reply. If the passing context
// is canceled, then function returns.
//
// todo(rjl493456442) the code organization is really ugly, need big refactor.
func (r *TxStatusRequest) RequestByHTTP(ctx context.Context, url string /* todo auth credential */, db ethdb.Database) error {
	var (
		hashes = r.Hashes
		errch  = make(chan error)
		status = make([]light.TxStatus, len(hashes))
	)
	for index, hash := range hashes {
		go func(index int, hash common.Hash) {
			res, err := httpDo(ctx, fmt.Sprintf("%s/chain/0x%x/txstatus", url, hash))
			if err != nil {
				errch <- err
				return
			}
			var lookup rawdb.LegacyTxLookupEntry
			if err := rlp.DecodeBytes(res, &lookup); err != nil {
				errch <- err
				return
			}
			status[index] = light.TxStatus{
				Status: core.TxStatusIncluded,
				Lookup: &lookup,
			}
			errch <- nil
		}(index, hash)
	}
	for i := 0; i < len(hashes); i++ {
		if err := <-errch; err != nil {
			return err
		}
	}
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.Status = status
	return nil
}

// httpDo sends a http request based on the given URL, extracts the
// response and return.
func httpDo(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	blob, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return blob, nil
}

type httpRetriever struct {
	url     string
	trigger time.Duration
	db      ethdb.Database
}

func newHTTPRetriever(url string, trigger time.Duration, db ethdb.Database) *httpRetriever {
	// If external CDN is not configured, return an nil instance.
	if url == "" {
		return nil
	}
	// validate the given host url, adjust it if necessary.
	for strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}
	return &httpRetriever{
		url:     url,
		trigger: trigger,
		db:      db,
	}
}

// retrieve sends a request to specified CDN network and waits for an answer
// that is delivered and successfully validated by the validator callback.
// It returns when a valid answer is delivered or the context is cancelled.
func (r *httpRetriever) retrieve(ctx context.Context, req LesOdrRequest) error {
	select {
	case <-ctx.Done():
		return nil
	case <-time.NewTimer(r.trigger).C:
	}
	return req.RequestByHTTP(ctx, r.url, r.db)
}
