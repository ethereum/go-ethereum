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
	"encoding/binary"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header
	ethVersion        = 63              // equivalent eth version for the downloader

	MaxHeaderFetch           = 192 // Amount of block headers to be fetched per retrieval request
	MaxBodyFetch             = 32  // Amount of block bodies to be fetched per retrieval request
	MaxReceiptFetch          = 128 // Amount of transaction receipts to allow fetching per request
	MaxCodeFetch             = 64  // Amount of contract codes to allow fetching per request
	MaxProofsFetch           = 64  // Amount of merkle proofs to be fetched per retrieval request
	MaxHelperTrieProofsFetch = 64  // Amount of helper tries to be fetched per retrieval request
	MaxTxSend                = 64  // Amount of transactions to be send per request
	MaxTxStatus              = 256 // Amount of transactions to queried per request
)

var (
	errTooManyInvalidRequest = errors.New("too many invalid requests made")
	errFullClientPool        = errors.New("client pool is full")
)

// serverHandler is responsible for serving light client and process
// all incoming light requests.
type serverHandler struct {
	blockchain *core.BlockChain
	chainDb    ethdb.Database
	txpool     *core.TxPool
	server     *LesServer

	closeCh chan struct{}  // Channel used to exit all background routines of handler.
	wg      sync.WaitGroup // WaitGroup used to track all background routines of handler.
	synced  func() bool    // Callback function used to determine whether local node is synced.

	// Testing fields
	addTxsSync bool
}

func newServerHandler(server *LesServer, blockchain *core.BlockChain, chainDb ethdb.Database, txpool *core.TxPool, synced func() bool) *serverHandler {
	handler := &serverHandler{
		server:     server,
		blockchain: blockchain,
		chainDb:    chainDb,
		txpool:     txpool,
		closeCh:    make(chan struct{}),
		synced:     synced,
	}
	return handler
}

// start starts the server handler.
func (h *serverHandler) start() {
	h.wg.Add(1)
	go h.broadcastHeaders()
}

// stop stops the server handler.
func (h *serverHandler) stop() {
	close(h.closeCh)
	h.wg.Wait()
}

// runPeer is the p2p protocol run function for the given version.
func (h *serverHandler) runPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := newClientPeer(int(version), h.server.config.NetworkId, p, newMeteredMsgWriter(rw, int(version)))
	defer peer.close()
	h.wg.Add(1)
	defer h.wg.Done()
	return h.handle(peer)
}

func (h *serverHandler) handle(p *clientPeer) error {
	p.Log().Debug("Light Ethereum peer connected", "name", p.Name())

	// Execute the LES handshake
	var (
		head   = h.blockchain.CurrentHeader()
		hash   = head.Hash()
		number = head.Number.Uint64()
		td     = h.blockchain.GetTd(hash, number)
	)
	if err := p.Handshake(td, hash, number, h.blockchain.Genesis().Hash(), h.server); err != nil {
		p.Log().Debug("Light Ethereum handshake failed", "err", err)
		return err
	}
	if p.server {
		// connected to another server, no messages expected, just wait for disconnection
		_, err := p.rw.ReadMsg()
		return err
	}
	// Reject light clients if server is not synced.
	if !h.synced() {
		p.Log().Debug("Light server not synced, rejecting peer")
		return p2p.DiscRequested
	}
	defer p.fcClient.Disconnect()

	// Disconnect the inbound peer if it's rejected by clientPool
	if !h.server.clientPool.connect(p, 0) {
		p.Log().Debug("Light Ethereum peer registration failed", "err", errFullClientPool)
		return errFullClientPool
	}
	// Register the peer locally
	if err := h.server.peers.register(p); err != nil {
		h.server.clientPool.disconnect(p)
		p.Log().Error("Light Ethereum peer registration failed", "err", err)
		return err
	}
	clientConnectionGauge.Update(int64(h.server.peers.len()))

	var wg sync.WaitGroup // Wait group used to track all in-flight task routines.

	connectedAt := mclock.Now()
	defer func() {
		wg.Wait() // Ensure all background task routines have exited.
		h.server.peers.unregister(p.id)
		h.server.clientPool.disconnect(p)
		clientConnectionGauge.Update(int64(h.server.peers.len()))
		connectionTimer.Update(time.Duration(mclock.Now() - connectedAt))
	}()
	// Mark the peer starts to be served.
	atomic.StoreUint32(&p.serving, 1)
	defer atomic.StoreUint32(&p.serving, 0)

	// Spawn a main loop to handle all incoming messages.
	for {
		select {
		case err := <-p.errCh:
			p.Log().Debug("Failed to send light ethereum response", "err", err)
			return err
		default:
		}
		if err := h.handleMsg(p, &wg); err != nil {
			p.Log().Debug("Light Ethereum message handling failed", "err", err)
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (h *serverHandler) handleMsg(p *clientPeer, wg *sync.WaitGroup) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	p.Log().Trace("Light Ethereum message arrived", "code", msg.Code, "bytes", msg.Size)

	// Discard large message which exceeds the limitation.
	if msg.Size > ProtocolMaxMsgSize {
		clientErrorMeter.Mark(1)
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	var (
		maxCost uint64
		task    *servingTask
	)
	p.responseCount++
	responseCount := p.responseCount
	// accept returns an indicator whether the request can be served.
	// If so, deduct the max cost from the flow control buffer.
	accept := func(reqID, reqCnt, maxCnt uint64) bool {
		// Short circuit if the peer is already frozen or the request is invalid.
		inSizeCost := h.server.costTracker.realCost(0, msg.Size, 0)
		if p.isFrozen() || reqCnt == 0 || reqCnt > maxCnt {
			p.fcClient.OneTimeCost(inSizeCost)
			return false
		}
		// Prepaid max cost units before request been serving.
		maxCost = p.fcCosts.getMaxCost(msg.Code, reqCnt)
		accepted, bufShort, priority := p.fcClient.AcceptRequest(reqID, responseCount, maxCost)
		if !accepted {
			p.freeze()
			p.Log().Error("Request came too early", "remaining", common.PrettyDuration(time.Duration(bufShort*1000000/p.fcParams.MinRecharge)))
			p.fcClient.OneTimeCost(inSizeCost)
			return false
		}
		// Create a multi-stage task, estimate the time it takes for the task to
		// execute, and cache it in the request service queue.
		factor := h.server.costTracker.globalFactor()
		if factor < 0.001 {
			factor = 1
			p.Log().Error("Invalid global cost factor", "factor", factor)
		}
		maxTime := uint64(float64(maxCost) / factor)
		task = h.server.servingQueue.newTask(p, maxTime, priority)
		if task.start() {
			return true
		}
		p.fcClient.RequestProcessed(reqID, responseCount, maxCost, inSizeCost)
		return false
	}
	// sendResponse sends back the response and updates the flow control statistic.
	sendResponse := func(reqID, amount uint64, reply *reply, servingTime uint64) {
		p.responseLock.Lock()
		defer p.responseLock.Unlock()

		// Short circuit if the client is already frozen.
		if p.isFrozen() {
			realCost := h.server.costTracker.realCost(servingTime, msg.Size, 0)
			p.fcClient.RequestProcessed(reqID, responseCount, maxCost, realCost)
			return
		}
		// Positive correction buffer value with real cost.
		var replySize uint32
		if reply != nil {
			replySize = reply.size()
		}
		var realCost uint64
		if h.server.costTracker.testing {
			realCost = maxCost // Assign a fake cost for testing purpose
		} else {
			realCost = h.server.costTracker.realCost(servingTime, msg.Size, replySize)
		}
		bv := p.fcClient.RequestProcessed(reqID, responseCount, maxCost, realCost)
		if amount != 0 {
			// Feed cost tracker request serving statistic.
			h.server.costTracker.updateStats(msg.Code, amount, servingTime, realCost)
			// Reduce priority "balance" for the specific peer.
			h.server.clientPool.requestCost(p, realCost)
		}
		if reply != nil {
			p.mustQueueSend(func() {
				if err := reply.send(bv); err != nil {
					select {
					case p.errCh <- err:
					default:
					}
				}
			})
		}
	}
	switch msg.Code {
	case GetBlockHeadersMsg:
		p.Log().Trace("Received block header request")
		if metrics.EnabledExpensive {
			miscInHeaderPacketsMeter.Mark(1)
			miscInHeaderTrafficMeter.Mark(int64(msg.Size))
		}
		var req struct {
			ReqID uint64
			Query getBlockHeadersData
		}
		if err := msg.Decode(&req); err != nil {
			clientErrorMeter.Mark(1)
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		query := req.Query
		if accept(req.ReqID, query.Amount, MaxHeaderFetch) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				hashMode := query.Origin.Hash != (common.Hash{})
				first := true
				maxNonCanonical := uint64(100)

				// Gather headers until the fetch or network limits is reached
				var (
					bytes   common.StorageSize
					headers []*types.Header
					unknown bool
				)
				for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit {
					if !first && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					// Retrieve the next header satisfying the query
					var origin *types.Header
					if hashMode {
						if first {
							origin = h.blockchain.GetHeaderByHash(query.Origin.Hash)
							if origin != nil {
								query.Origin.Number = origin.Number.Uint64()
							}
						} else {
							origin = h.blockchain.GetHeader(query.Origin.Hash, query.Origin.Number)
						}
					} else {
						origin = h.blockchain.GetHeaderByNumber(query.Origin.Number)
					}
					if origin == nil {
						atomic.AddUint32(&p.invalidCount, 1)
						break
					}
					headers = append(headers, origin)
					bytes += estHeaderRlpSize

					// Advance to the next header of the query
					switch {
					case hashMode && query.Reverse:
						// Hash based traversal towards the genesis block
						ancestor := query.Skip + 1
						if ancestor == 0 {
							unknown = true
						} else {
							query.Origin.Hash, query.Origin.Number = h.blockchain.GetAncestor(query.Origin.Hash, query.Origin.Number, ancestor, &maxNonCanonical)
							unknown = query.Origin.Hash == common.Hash{}
						}
					case hashMode && !query.Reverse:
						// Hash based traversal towards the leaf block
						var (
							current = origin.Number.Uint64()
							next    = current + query.Skip + 1
						)
						if next <= current {
							infos, _ := json.MarshalIndent(p.Peer.Info(), "", "  ")
							p.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
							unknown = true
						} else {
							if header := h.blockchain.GetHeaderByNumber(next); header != nil {
								nextHash := header.Hash()
								expOldHash, _ := h.blockchain.GetAncestor(nextHash, next, query.Skip+1, &maxNonCanonical)
								if expOldHash == query.Origin.Hash {
									query.Origin.Hash, query.Origin.Number = nextHash, next
								} else {
									unknown = true
								}
							} else {
								unknown = true
							}
						}
					case query.Reverse:
						// Number based traversal towards the genesis block
						if query.Origin.Number >= query.Skip+1 {
							query.Origin.Number -= query.Skip + 1
						} else {
							unknown = true
						}

					case !query.Reverse:
						// Number based traversal towards the leaf block
						query.Origin.Number += query.Skip + 1
					}
					first = false
				}
				reply := p.replyBlockHeaders(req.ReqID, headers)
				sendResponse(req.ReqID, query.Amount, p.replyBlockHeaders(req.ReqID, headers), task.done())
				if metrics.EnabledExpensive {
					miscOutHeaderPacketsMeter.Mark(1)
					miscOutHeaderTrafficMeter.Mark(int64(reply.size()))
					miscServingTimeHeaderTimer.Update(time.Duration(task.servingTime))
				}
			}()
		}

	case GetBlockBodiesMsg:
		p.Log().Trace("Received block bodies request")
		if metrics.EnabledExpensive {
			miscInBodyPacketsMeter.Mark(1)
			miscInBodyTrafficMeter.Mark(int64(msg.Size))
		}
		var req struct {
			ReqID  uint64
			Hashes []common.Hash
		}
		if err := msg.Decode(&req); err != nil {
			clientErrorMeter.Mark(1)
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		var (
			bytes  int
			bodies []rlp.RawValue
		)
		reqCnt := len(req.Hashes)
		if accept(req.ReqID, uint64(reqCnt), MaxBodyFetch) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i, hash := range req.Hashes {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					if bytes >= softResponseLimit {
						break
					}
					body := h.blockchain.GetBodyRLP(hash)
					if body == nil {
						atomic.AddUint32(&p.invalidCount, 1)
						continue
					}
					bodies = append(bodies, body)
					bytes += len(body)
				}
				reply := p.replyBlockBodiesRLP(req.ReqID, bodies)
				sendResponse(req.ReqID, uint64(reqCnt), reply, task.done())
				if metrics.EnabledExpensive {
					miscOutBodyPacketsMeter.Mark(1)
					miscOutBodyTrafficMeter.Mark(int64(reply.size()))
					miscServingTimeBodyTimer.Update(time.Duration(task.servingTime))
				}
			}()
		}

	case GetCodeMsg:
		p.Log().Trace("Received code request")
		if metrics.EnabledExpensive {
			miscInCodePacketsMeter.Mark(1)
			miscInCodeTrafficMeter.Mark(int64(msg.Size))
		}
		var req struct {
			ReqID uint64
			Reqs  []CodeReq
		}
		if err := msg.Decode(&req); err != nil {
			clientErrorMeter.Mark(1)
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		var (
			bytes int
			data  [][]byte
		)
		reqCnt := len(req.Reqs)
		if accept(req.ReqID, uint64(reqCnt), MaxCodeFetch) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i, request := range req.Reqs {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					// Look up the root hash belonging to the request
					header := h.blockchain.GetHeaderByHash(request.BHash)
					if header == nil {
						p.Log().Warn("Failed to retrieve associate header for code", "hash", request.BHash)
						atomic.AddUint32(&p.invalidCount, 1)
						continue
					}
					// Refuse to search stale state data in the database since looking for
					// a non-exist key is kind of expensive.
					local := h.blockchain.CurrentHeader().Number.Uint64()
					if !h.server.archiveMode && header.Number.Uint64()+core.TriesInMemory <= local {
						p.Log().Debug("Reject stale code request", "number", header.Number.Uint64(), "head", local)
						atomic.AddUint32(&p.invalidCount, 1)
						continue
					}
					triedb := h.blockchain.StateCache().TrieDB()

					account, err := h.getAccount(triedb, header.Root, common.BytesToHash(request.AccKey))
					if err != nil {
						p.Log().Warn("Failed to retrieve account for code", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(request.AccKey), "err", err)
						atomic.AddUint32(&p.invalidCount, 1)
						continue
					}
					code, err := triedb.Node(common.BytesToHash(account.CodeHash))
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
				reply := p.replyCode(req.ReqID, data)
				sendResponse(req.ReqID, uint64(reqCnt), reply, task.done())
				if metrics.EnabledExpensive {
					miscOutCodePacketsMeter.Mark(1)
					miscOutCodeTrafficMeter.Mark(int64(reply.size()))
					miscServingTimeCodeTimer.Update(time.Duration(task.servingTime))
				}
			}()
		}

	case GetReceiptsMsg:
		p.Log().Trace("Received receipts request")
		if metrics.EnabledExpensive {
			miscInReceiptPacketsMeter.Mark(1)
			miscInReceiptTrafficMeter.Mark(int64(msg.Size))
		}
		var req struct {
			ReqID  uint64
			Hashes []common.Hash
		}
		if err := msg.Decode(&req); err != nil {
			clientErrorMeter.Mark(1)
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		var (
			bytes    int
			receipts []rlp.RawValue
		)
		reqCnt := len(req.Hashes)
		if accept(req.ReqID, uint64(reqCnt), MaxReceiptFetch) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i, hash := range req.Hashes {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					if bytes >= softResponseLimit {
						break
					}
					// Retrieve the requested block's receipts, skipping if unknown to us
					results := h.blockchain.GetReceiptsByHash(hash)
					if results == nil {
						if header := h.blockchain.GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
							atomic.AddUint32(&p.invalidCount, 1)
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
				reply := p.replyReceiptsRLP(req.ReqID, receipts)
				sendResponse(req.ReqID, uint64(reqCnt), reply, task.done())
				if metrics.EnabledExpensive {
					miscOutReceiptPacketsMeter.Mark(1)
					miscOutReceiptTrafficMeter.Mark(int64(reply.size()))
					miscServingTimeReceiptTimer.Update(time.Duration(task.servingTime))
				}
			}()
		}

	case GetProofsV2Msg:
		p.Log().Trace("Received les/2 proofs request")
		if metrics.EnabledExpensive {
			miscInTrieProofPacketsMeter.Mark(1)
			miscInTrieProofTrafficMeter.Mark(int64(msg.Size))
		}
		var req struct {
			ReqID uint64
			Reqs  []ProofReq
		}
		if err := msg.Decode(&req); err != nil {
			clientErrorMeter.Mark(1)
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			lastBHash common.Hash
			root      common.Hash
		)
		reqCnt := len(req.Reqs)
		if accept(req.ReqID, uint64(reqCnt), MaxProofsFetch) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				nodes := light.NewNodeSet()

				for i, request := range req.Reqs {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					// Look up the root hash belonging to the request
					var (
						header *types.Header
						trie   state.Trie
					)
					if request.BHash != lastBHash {
						root, lastBHash = common.Hash{}, request.BHash

						if header = h.blockchain.GetHeaderByHash(request.BHash); header == nil {
							p.Log().Warn("Failed to retrieve header for proof", "hash", request.BHash)
							atomic.AddUint32(&p.invalidCount, 1)
							continue
						}
						// Refuse to search stale state data in the database since looking for
						// a non-exist key is kind of expensive.
						local := h.blockchain.CurrentHeader().Number.Uint64()
						if !h.server.archiveMode && header.Number.Uint64()+core.TriesInMemory <= local {
							p.Log().Debug("Reject stale trie request", "number", header.Number.Uint64(), "head", local)
							atomic.AddUint32(&p.invalidCount, 1)
							continue
						}
						root = header.Root
					}
					// If a header lookup failed (non existent), ignore subsequent requests for the same header
					if root == (common.Hash{}) {
						atomic.AddUint32(&p.invalidCount, 1)
						continue
					}
					// Open the account or storage trie for the request
					statedb := h.blockchain.StateCache()

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
						account, err := h.getAccount(statedb.TrieDB(), root, common.BytesToHash(request.AccKey))
						if err != nil {
							p.Log().Warn("Failed to retrieve account for proof", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(request.AccKey), "err", err)
							atomic.AddUint32(&p.invalidCount, 1)
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
				reply := p.replyProofsV2(req.ReqID, nodes.NodeList())
				sendResponse(req.ReqID, uint64(reqCnt), reply, task.done())
				if metrics.EnabledExpensive {
					miscOutTrieProofPacketsMeter.Mark(1)
					miscOutTrieProofTrafficMeter.Mark(int64(reply.size()))
					miscServingTimeTrieProofTimer.Update(time.Duration(task.servingTime))
				}
			}()
		}

	case GetHelperTrieProofsMsg:
		p.Log().Trace("Received helper trie proof request")
		if metrics.EnabledExpensive {
			miscInHelperTriePacketsMeter.Mark(1)
			miscInHelperTrieTrafficMeter.Mark(int64(msg.Size))
		}
		var req struct {
			ReqID uint64
			Reqs  []HelperTrieReq
		}
		if err := msg.Decode(&req); err != nil {
			clientErrorMeter.Mark(1)
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			auxBytes int
			auxData  [][]byte
		)
		reqCnt := len(req.Reqs)
		if accept(req.ReqID, uint64(reqCnt), MaxHelperTrieProofsFetch) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var (
					lastIdx  uint64
					lastType uint
					root     common.Hash
					auxTrie  *trie.Trie
				)
				nodes := light.NewNodeSet()
				for i, request := range req.Reqs {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					if auxTrie == nil || request.Type != lastType || request.TrieIdx != lastIdx {
						auxTrie, lastType, lastIdx = nil, request.Type, request.TrieIdx

						var prefix string
						if root, prefix = h.getHelperTrie(request.Type, request.TrieIdx); root != (common.Hash{}) {
							auxTrie, _ = trie.New(root, trie.NewDatabase(rawdb.NewTable(h.chainDb, prefix)))
						}
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
						if request.AuxReq != 0 {
							data := h.getAuxiliaryHeaders(request)
							auxData = append(auxData, data)
							auxBytes += len(data)
						}
					}
					if nodes.DataSize()+auxBytes >= softResponseLimit {
						break
					}
				}
				reply := p.replyHelperTrieProofs(req.ReqID, HelperTrieResps{Proofs: nodes.NodeList(), AuxData: auxData})
				sendResponse(req.ReqID, uint64(reqCnt), reply, task.done())
				if metrics.EnabledExpensive {
					miscOutHelperTriePacketsMeter.Mark(1)
					miscOutHelperTrieTrafficMeter.Mark(int64(reply.size()))
					miscServingTimeHelperTrieTimer.Update(time.Duration(task.servingTime))
				}
			}()
		}

	case SendTxV2Msg:
		p.Log().Trace("Received new transactions")
		if metrics.EnabledExpensive {
			miscInTxsPacketsMeter.Mark(1)
			miscInTxsTrafficMeter.Mark(int64(msg.Size))
		}
		var req struct {
			ReqID uint64
			Txs   []*types.Transaction
		}
		if err := msg.Decode(&req); err != nil {
			clientErrorMeter.Mark(1)
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(req.Txs)
		if accept(req.ReqID, uint64(reqCnt), MaxTxSend) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				stats := make([]light.TxStatus, len(req.Txs))
				for i, tx := range req.Txs {
					if i != 0 && !task.waitOrStop() {
						return
					}
					hash := tx.Hash()
					stats[i] = h.txStatus(hash)
					if stats[i].Status == core.TxStatusUnknown {
						addFn := h.txpool.AddRemotes
						// Add txs synchronously for testing purpose
						if h.addTxsSync {
							addFn = h.txpool.AddRemotesSync
						}
						if errs := addFn([]*types.Transaction{tx}); errs[0] != nil {
							stats[i].Error = errs[0].Error()
							continue
						}
						stats[i] = h.txStatus(hash)
					}
				}
				reply := p.replyTxStatus(req.ReqID, stats)
				sendResponse(req.ReqID, uint64(reqCnt), reply, task.done())
				if metrics.EnabledExpensive {
					miscOutTxsPacketsMeter.Mark(1)
					miscOutTxsTrafficMeter.Mark(int64(reply.size()))
					miscServingTimeTxTimer.Update(time.Duration(task.servingTime))
				}
			}()
		}

	case GetTxStatusMsg:
		p.Log().Trace("Received transaction status query request")
		if metrics.EnabledExpensive {
			miscInTxStatusPacketsMeter.Mark(1)
			miscInTxStatusTrafficMeter.Mark(int64(msg.Size))
		}
		var req struct {
			ReqID  uint64
			Hashes []common.Hash
		}
		if err := msg.Decode(&req); err != nil {
			clientErrorMeter.Mark(1)
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(req.Hashes)
		if accept(req.ReqID, uint64(reqCnt), MaxTxStatus) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				stats := make([]light.TxStatus, len(req.Hashes))
				for i, hash := range req.Hashes {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					stats[i] = h.txStatus(hash)
				}
				reply := p.replyTxStatus(req.ReqID, stats)
				sendResponse(req.ReqID, uint64(reqCnt), reply, task.done())
				if metrics.EnabledExpensive {
					miscOutTxStatusPacketsMeter.Mark(1)
					miscOutTxStatusTrafficMeter.Mark(int64(reply.size()))
					miscServingTimeTxStatusTimer.Update(time.Duration(task.servingTime))
				}
			}()
		}

	default:
		p.Log().Trace("Received invalid message", "code", msg.Code)
		clientErrorMeter.Mark(1)
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	// If the client has made too much invalid request(e.g. request a non-exist data),
	// reject them to prevent SPAM attack.
	if atomic.LoadUint32(&p.invalidCount) > maxRequestErrors {
		clientErrorMeter.Mark(1)
		return errTooManyInvalidRequest
	}
	return nil
}

// getAccount retrieves an account from the state based on root.
func (h *serverHandler) getAccount(triedb *trie.Database, root, hash common.Hash) (state.Account, error) {
	trie, err := trie.New(root, triedb)
	if err != nil {
		return state.Account{}, err
	}
	blob, err := trie.TryGet(hash[:])
	if err != nil {
		return state.Account{}, err
	}
	var account state.Account
	if err = rlp.DecodeBytes(blob, &account); err != nil {
		return state.Account{}, err
	}
	return account, nil
}

// getHelperTrie returns the post-processed trie root for the given trie ID and section index
func (h *serverHandler) getHelperTrie(typ uint, index uint64) (common.Hash, string) {
	switch typ {
	case htCanonical:
		sectionHead := rawdb.ReadCanonicalHash(h.chainDb, (index+1)*h.server.iConfig.ChtSize-1)
		return light.GetChtRoot(h.chainDb, index, sectionHead), light.ChtTablePrefix
	case htBloomBits:
		sectionHead := rawdb.ReadCanonicalHash(h.chainDb, (index+1)*h.server.iConfig.BloomTrieSize-1)
		return light.GetBloomTrieRoot(h.chainDb, index, sectionHead), light.BloomTrieTablePrefix
	}
	return common.Hash{}, ""
}

// getAuxiliaryHeaders returns requested auxiliary headers for the CHT request.
func (h *serverHandler) getAuxiliaryHeaders(req HelperTrieReq) []byte {
	if req.Type == htCanonical && req.AuxReq == auxHeader && len(req.Key) == 8 {
		blockNum := binary.BigEndian.Uint64(req.Key)
		hash := rawdb.ReadCanonicalHash(h.chainDb, blockNum)
		return rawdb.ReadHeaderRLP(h.chainDb, hash, blockNum)
	}
	return nil
}

// txStatus returns the status of a specified transaction.
func (h *serverHandler) txStatus(hash common.Hash) light.TxStatus {
	var stat light.TxStatus
	// Looking the transaction in txpool first.
	stat.Status = h.txpool.Status([]common.Hash{hash})[0]

	// If the transaction is unknown to the pool, try looking it up locally.
	if stat.Status == core.TxStatusUnknown {
		lookup := h.blockchain.GetTransactionLookup(hash)
		if lookup != nil {
			stat.Status = core.TxStatusIncluded
			stat.Lookup = lookup
		}
	}
	return stat
}

// broadcastHeaders broadcasts new block information to all connected light
// clients. According to the agreement between client and server, server should
// only broadcast new announcement if the total difficulty is higher than the
// last one. Besides server will add the signature if client requires.
func (h *serverHandler) broadcastHeaders() {
	defer h.wg.Done()

	headCh := make(chan core.ChainHeadEvent, 10)
	headSub := h.blockchain.SubscribeChainHeadEvent(headCh)
	defer headSub.Unsubscribe()

	var (
		lastHead *types.Header
		lastTd   = common.Big0
	)
	for {
		select {
		case ev := <-headCh:
			peers := h.server.peers.allPeers()
			if len(peers) == 0 {
				continue
			}
			header := ev.Block.Header()
			hash, number := header.Hash(), header.Number.Uint64()
			td := h.blockchain.GetTd(hash, number)
			if td == nil || td.Cmp(lastTd) <= 0 {
				continue
			}
			var reorg uint64
			if lastHead != nil {
				reorg = lastHead.Number.Uint64() - rawdb.FindCommonAncestor(h.chainDb, header, lastHead).Number.Uint64()
			}
			lastHead, lastTd = header, td

			log.Debug("Announcing block to peers", "number", number, "hash", hash, "td", td, "reorg", reorg)
			var (
				signed         bool
				signedAnnounce announceData
			)
			announce := announceData{Hash: hash, Number: number, Td: td, ReorgDepth: reorg}
			for _, p := range peers {
				p := p
				switch p.announceType {
				case announceTypeSimple:
					if !p.queueSend(func() { p.sendAnnounce(announce) }) {
						log.Debug("Drop announcement because queue is full", "number", number, "hash", hash)
					}
				case announceTypeSigned:
					if !signed {
						signedAnnounce = announce
						signedAnnounce.sign(h.server.privateKey)
						signed = true
					}
					if !p.queueSend(func() { p.sendAnnounce(signedAnnounce) }) {
						log.Debug("Drop announcement because queue is full", "number", number, "hash", hash)
					}
				}
			}
		case <-h.closeCh:
			return
		}
	}
}
