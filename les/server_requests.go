// Copyright 2021 The go-ethereum Authors
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
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type serverBackend interface {
	ArchiveMode() bool
	AddTxsSync() bool
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	GetHelperTrie(typ uint, index uint64) *trie.Trie
}

type HandlerRequest interface {
	ReqAmount() (uint64, uint64)
	InMetrics(size int64)
	OutMetrics(size int64, servingTime time.Duration)
	Serve(b serverBackend, reqID uint64, p *clientPeer, waitOrStop func() bool) *reply
}

type GetBlockHeadersReq struct {
	Origin  hashOrNumber // Block from which to retrieve headers
	Amount  uint64       // Maximum number of headers to retrieve
	Skip    uint64       // Blocks to skip between consecutive headers
	Reverse bool         // Query direction (false = rising towards latest, true = falling towards genesis)
}

func (r *GetBlockHeadersReq) ReqAmount() (uint64, uint64) { return r.Amount, MaxHeaderFetch }

func (r *GetBlockHeadersReq) InMetrics(size int64) {
	miscInHeaderPacketsMeter.Mark(1)
	miscInHeaderTrafficMeter.Mark(size)
}

func (r *GetBlockHeadersReq) OutMetrics(size int64, servingTime time.Duration) {
	miscOutHeaderPacketsMeter.Mark(1)
	miscOutHeaderTrafficMeter.Mark(size)
	miscServingTimeHeaderTimer.Update(servingTime)
}

func (r *GetBlockHeadersReq) Serve(backend serverBackend, reqID uint64, p *clientPeer, waitOrStop func() bool) *reply {
	bc := backend.BlockChain()
	hashMode := r.Origin.Hash != (common.Hash{})
	first := true
	maxNonCanonical := uint64(100)

	// Gather headers until the fetch or network limits is reached
	var (
		bytes   common.StorageSize
		headers []*types.Header
		unknown bool
	)
	for !unknown && len(headers) < int(r.Amount) && bytes < softResponseLimit {
		if !first && !waitOrStop() {
			return nil
		}
		// Retrieve the next header satisfying the r
		var origin *types.Header
		if hashMode {
			if first {
				origin = bc.GetHeaderByHash(r.Origin.Hash)
				if origin != nil {
					r.Origin.Number = origin.Number.Uint64()
				}
			} else {
				origin = bc.GetHeader(r.Origin.Hash, r.Origin.Number)
			}
		} else {
			origin = bc.GetHeaderByNumber(r.Origin.Number)
		}
		if origin == nil {
			break
		}
		headers = append(headers, origin)
		bytes += estHeaderRlpSize

		// Advance to the next header of the r
		switch {
		case hashMode && r.Reverse:
			// Hash based traversal towards the genesis block
			ancestor := r.Skip + 1
			if ancestor == 0 {
				unknown = true
			} else {
				r.Origin.Hash, r.Origin.Number = bc.GetAncestor(r.Origin.Hash, r.Origin.Number, ancestor, &maxNonCanonical)
				unknown = r.Origin.Hash == common.Hash{}
			}
		case hashMode && !r.Reverse:
			// Hash based traversal towards the leaf block
			var (
				current = origin.Number.Uint64()
				next    = current + r.Skip + 1
			)
			if next <= current {
				infos, _ := json.MarshalIndent(p.Peer.Info(), "", "  ")
				p.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", r.Skip, "next", next, "attacker", infos)
				unknown = true
			} else {
				if header := bc.GetHeaderByNumber(next); header != nil {
					nextHash := header.Hash()
					expOldHash, _ := bc.GetAncestor(nextHash, next, r.Skip+1, &maxNonCanonical)
					if expOldHash == r.Origin.Hash {
						r.Origin.Hash, r.Origin.Number = nextHash, next
					} else {
						unknown = true
					}
				} else {
					unknown = true
				}
			}
		case r.Reverse:
			// Number based traversal towards the genesis block
			if r.Origin.Number >= r.Skip+1 {
				r.Origin.Number -= r.Skip + 1
			} else {
				unknown = true
			}

		case !r.Reverse:
			// Number based traversal towards the leaf block
			r.Origin.Number += r.Skip + 1
		}
		first = false
	}
	return p.replyBlockHeaders(reqID, headers)
}

type GetBlockBodiesReq []common.Hash

func (r GetBlockBodiesReq) ReqAmount() (uint64, uint64) { return uint64(len(r)), MaxBodyFetch }

func (r GetBlockBodiesReq) InMetrics(size int64) {
	miscInBodyPacketsMeter.Mark(1)
	miscInBodyTrafficMeter.Mark(size)
}

func (r GetBlockBodiesReq) OutMetrics(size int64, servingTime time.Duration) {
	miscOutBodyPacketsMeter.Mark(1)
	miscOutBodyTrafficMeter.Mark(size)
	miscServingTimeBodyTimer.Update(servingTime)
}

func (r GetBlockBodiesReq) Serve(backend serverBackend, reqID uint64, p *clientPeer, waitOrStop func() bool) *reply {
	var (
		bytes  int
		bodies []rlp.RawValue
	)
	bc := backend.BlockChain()
	for i, hash := range r {
		if i != 0 && !waitOrStop() {
			return nil
		}
		if bytes >= softResponseLimit {
			break
		}
		body := bc.GetBodyRLP(hash)
		if body == nil {
			p.bumpInvalid()
			continue
		}
		bodies = append(bodies, body)
		bytes += len(body)
	}
	return p.replyBlockBodiesRLP(reqID, bodies)
}

type GetCodeReq []CodeReq

func (r GetCodeReq) ReqAmount() (uint64, uint64) { return uint64(len(r)), MaxCodeFetch }

func (r GetCodeReq) InMetrics(size int64) {
	miscInCodePacketsMeter.Mark(1)
	miscInCodeTrafficMeter.Mark(size)
}

func (r GetCodeReq) OutMetrics(size int64, servingTime time.Duration) {
	miscOutCodePacketsMeter.Mark(1)
	miscOutCodeTrafficMeter.Mark(size)
	miscServingTimeCodeTimer.Update(servingTime)
}

func (r GetCodeReq) Serve(backend serverBackend, reqID uint64, p *clientPeer, waitOrStop func() bool) *reply {
	var (
		bytes int
		data  [][]byte
	)
	bc := backend.BlockChain()
	for i, request := range r {
		if i != 0 && !waitOrStop() {
			return nil
		}
		// Look up the root hash belonging to the request
		header := bc.GetHeaderByHash(request.BHash)
		if header == nil {
			p.Log().Warn("Failed to retrieve associate header for code", "hash", request.BHash)
			p.bumpInvalid()
			continue
		}
		// Refuse to search stale state data in the database since looking for
		// a non-exist key is kind of expensive.
		local := bc.CurrentHeader().Number.Uint64()
		if !backend.ArchiveMode() && header.Number.Uint64()+core.TriesInMemory <= local {
			p.Log().Debug("Reject stale code request", "number", header.Number.Uint64(), "head", local)
			p.bumpInvalid()
			continue
		}
		triedb := bc.StateCache().TrieDB()

		account, err := getAccount(triedb, header.Root, common.BytesToHash(request.AccKey))
		if err != nil {
			p.Log().Warn("Failed to retrieve account for code", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(request.AccKey), "err", err)
			p.bumpInvalid()
			continue
		}
		code, err := bc.StateCache().ContractCode(common.BytesToHash(request.AccKey), common.BytesToHash(account.CodeHash))
		if err != nil {
			p.Log().Warn("Failed to retrieve account code", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(request.AccKey), "codehash", common.BytesToHash(account.CodeHash), "err", err)
			continue
		}
		// Accumulate the code and abort if enough data was retrieved
		data = append(data, code)
		if bytes += len(code); bytes >= softResponseLimit {
			break
		}
	}
	return p.replyCode(reqID, data)
}

type GetReceiptsReq []common.Hash

func (r GetReceiptsReq) ReqAmount() (uint64, uint64) { return uint64(len(r)), MaxReceiptFetch }

func (r GetReceiptsReq) InMetrics(size int64) {
	miscInReceiptPacketsMeter.Mark(1)
	miscInReceiptTrafficMeter.Mark(size)
}

func (r GetReceiptsReq) OutMetrics(size int64, servingTime time.Duration) {
	miscOutReceiptPacketsMeter.Mark(1)
	miscOutReceiptTrafficMeter.Mark(size)
	miscServingTimeReceiptTimer.Update(servingTime)
}

func (r GetReceiptsReq) Serve(backend serverBackend, reqID uint64, p *clientPeer, waitOrStop func() bool) *reply {
	var (
		bytes    int
		receipts []rlp.RawValue
	)
	bc := backend.BlockChain()
	for i, hash := range r {
		if i != 0 && !waitOrStop() {
			return nil
		}
		if bytes >= softResponseLimit {
			break
		}
		// Retrieve the requested block's receipts, skipping if unknown to us
		results := bc.GetReceiptsByHash(hash)
		if results == nil {
			if header := bc.GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
				p.bumpInvalid()
				continue
			}
		}
		// If known, encode and queue for response packet
		if encoded, err := rlp.EncodeToBytes(results); err != nil {
			log.Error("Failed to encode receipt", "err", err)
		} else {
			receipts = append(receipts, encoded)
			bytes += len(encoded)
		}
	}
	return p.replyReceiptsRLP(reqID, receipts)
}

type GetProofsReq []ProofReq

func (r GetProofsReq) ReqAmount() (uint64, uint64) { return uint64(len(r)), MaxProofsFetch }

func (r GetProofsReq) InMetrics(size int64) {
	miscInTrieProofPacketsMeter.Mark(1)
	miscInTrieProofTrafficMeter.Mark(size)
}

func (r GetProofsReq) OutMetrics(size int64, servingTime time.Duration) {
	miscOutTrieProofPacketsMeter.Mark(1)
	miscOutTrieProofTrafficMeter.Mark(size)
	miscServingTimeTrieProofTimer.Update(servingTime)
}

func (r GetProofsReq) Serve(backend serverBackend, reqID uint64, p *clientPeer, waitOrStop func() bool) *reply {
	var (
		lastBHash common.Hash
		root      common.Hash
		header    *types.Header
		err       error
	)
	bc := backend.BlockChain()
	nodes := light.NewNodeSet()

	for i, request := range r {
		if i != 0 && !waitOrStop() {
			return nil
		}
		// Look up the root hash belonging to the request
		if request.BHash != lastBHash {
			root, lastBHash = common.Hash{}, request.BHash

			if header = bc.GetHeaderByHash(request.BHash); header == nil {
				p.Log().Warn("Failed to retrieve header for proof", "hash", request.BHash)
				p.bumpInvalid()
				continue
			}
			// Refuse to search stale state data in the database since looking for
			// a non-exist key is kind of expensive.
			local := bc.CurrentHeader().Number.Uint64()
			if !backend.ArchiveMode() && header.Number.Uint64()+core.TriesInMemory <= local {
				p.Log().Debug("Reject stale trie request", "number", header.Number.Uint64(), "head", local)
				p.bumpInvalid()
				continue
			}
			root = header.Root
		}
		// If a header lookup failed (non existent), ignore subsequent requests for the same header
		if root == (common.Hash{}) {
			p.bumpInvalid()
			continue
		}
		// Open the account or storage trie for the request
		statedb := bc.StateCache()

		var trie state.Trie
		switch len(request.AccKey) {
		case 0:
			// No account key specified, open an account trie
			trie, err = statedb.OpenTrie(root)
			if trie == nil || err != nil {
				p.Log().Warn("Failed to open storage trie for proof", "block", header.Number, "hash", header.Hash(), "root", root, "err", err)
				continue
			}
		default:
			// Account key specified, open a storage trie
			account, err := getAccount(statedb.TrieDB(), root, common.BytesToHash(request.AccKey))
			if err != nil {
				p.Log().Warn("Failed to retrieve account for proof", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(request.AccKey), "err", err)
				p.bumpInvalid()
				continue
			}
			trie, err = statedb.OpenStorageTrie(common.BytesToHash(request.AccKey), account.Root)
			if trie == nil || err != nil {
				p.Log().Warn("Failed to open storage trie for proof", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(request.AccKey), "root", account.Root, "err", err)
				continue
			}
		}
		// Prove the user's request from the account or stroage trie
		if err := trie.Prove(request.Key, request.FromLevel, nodes); err != nil {
			p.Log().Warn("Failed to prove state request", "block", header.Number, "hash", header.Hash(), "err", err)
			continue
		}
		if nodes.DataSize() >= softResponseLimit {
			break
		}
	}
	return p.replyProofsV2(reqID, nodes.NodeList())
}

type GetHelperTrieProofsReq []HelperTrieReq

func (r GetHelperTrieProofsReq) ReqAmount() (uint64, uint64) {
	return uint64(len(r)), MaxHelperTrieProofsFetch
}

func (r GetHelperTrieProofsReq) InMetrics(size int64) {
	miscInHelperTriePacketsMeter.Mark(1)
	miscInHelperTrieTrafficMeter.Mark(size)
}

func (r GetHelperTrieProofsReq) OutMetrics(size int64, servingTime time.Duration) {
	miscOutHelperTriePacketsMeter.Mark(1)
	miscOutHelperTrieTrafficMeter.Mark(size)
	miscServingTimeHelperTrieTimer.Update(servingTime)
}

func (r GetHelperTrieProofsReq) Serve(backend serverBackend, reqID uint64, p *clientPeer, waitOrStop func() bool) *reply {
	var (
		lastIdx  uint64
		lastType uint
		root     common.Hash
		auxTrie  *trie.Trie
		auxBytes int
		auxData  [][]byte
	)
	bc := backend.BlockChain()
	nodes := light.NewNodeSet()
	for i, request := range r {
		if i != 0 && !waitOrStop() {
			return nil
		}
		if auxTrie == nil || request.Type != lastType || request.TrieIdx != lastIdx {
			lastType, lastIdx = request.Type, request.TrieIdx
			auxTrie = backend.GetHelperTrie(request.Type, request.TrieIdx)
		}
		if request.AuxReq == auxRoot {
			var data []byte
			if root != (common.Hash{}) {
				data = root[:]
			}
			auxData = append(auxData, data)
			auxBytes += len(data)
		} else {
			if auxTrie != nil {
				auxTrie.Prove(request.Key, request.FromLevel, nodes)
			}
			if request.Type == htCanonical && request.AuxReq == auxHeader && len(request.Key) == 8 {
				header := bc.GetHeaderByNumber(binary.BigEndian.Uint64(request.Key))
				data, err := rlp.EncodeToBytes(header)
				if err != nil {
					log.Error("Failed to encode header", "err", err)
				}
				auxData = append(auxData, data)
				auxBytes += len(data)
			}
		}
		if nodes.DataSize()+auxBytes >= softResponseLimit {
			break
		}
	}
	return p.replyHelperTrieProofs(reqID, HelperTrieResps{Proofs: nodes.NodeList(), AuxData: auxData})
}

type SendTxReq []*types.Transaction

func (r SendTxReq) ReqAmount() (uint64, uint64) { return uint64(len(r)), MaxTxSend }

func (r SendTxReq) InMetrics(size int64) {
	miscInTxsPacketsMeter.Mark(1)
	miscInTxsTrafficMeter.Mark(size)
}

func (r SendTxReq) OutMetrics(size int64, servingTime time.Duration) {
	miscOutTxsPacketsMeter.Mark(1)
	miscOutTxsTrafficMeter.Mark(size)
	miscServingTimeTxTimer.Update(servingTime)
}

func (r SendTxReq) Serve(backend serverBackend, reqID uint64, p *clientPeer, waitOrStop func() bool) *reply {
	stats := make([]light.TxStatus, len(r))
	for i, tx := range r {
		if i != 0 && !waitOrStop() {
			return nil
		}
		hash := tx.Hash()
		stats[i] = txStatus(backend, hash)
		if stats[i].Status == core.TxStatusUnknown {
			addFn := backend.TxPool().AddRemotes
			// Add txs synchronously for testing purpose
			if backend.AddTxsSync() {
				addFn = backend.TxPool().AddRemotesSync
			}
			if errs := addFn([]*types.Transaction{tx}); errs[0] != nil {
				stats[i].Error = errs[0].Error()
				continue
			}
			stats[i] = txStatus(backend, hash)
		}
	}
	return p.replyTxStatus(reqID, stats)
}

type GetTxStatusReq []common.Hash

func (r GetTxStatusReq) ReqAmount() (uint64, uint64) { return uint64(len(r)), MaxTxStatus }

func (r GetTxStatusReq) InMetrics(size int64) {
	miscInTxStatusPacketsMeter.Mark(1)
	miscInTxStatusTrafficMeter.Mark(size)
}

func (r GetTxStatusReq) OutMetrics(size int64, servingTime time.Duration) {
	miscOutTxStatusPacketsMeter.Mark(1)
	miscOutTxStatusTrafficMeter.Mark(size)
	miscServingTimeTxStatusTimer.Update(servingTime)
}

func (r GetTxStatusReq) Serve(backend serverBackend, reqID uint64, p *clientPeer, waitOrStop func() bool) *reply {
	stats := make([]light.TxStatus, len(r))
	for i, hash := range r {
		if i != 0 && !waitOrStop() {
			return nil
		}
		stats[i] = txStatus(backend, hash)
	}
	return p.replyTxStatus(reqID, stats)
}

// txStatus returns the status of a specified transaction.
func txStatus(b serverBackend, hash common.Hash) light.TxStatus {
	var stat light.TxStatus
	// Looking the transaction in txpool first.
	stat.Status = b.TxPool().Status([]common.Hash{hash})[0]

	// If the transaction is unknown to the pool, try looking it up locally.
	if stat.Status == core.TxStatusUnknown {
		lookup := b.BlockChain().GetTransactionLookup(hash)
		if lookup != nil {
			stat.Status = core.TxStatusIncluded
			stat.Lookup = lookup
		}
	}
	return stat
}
