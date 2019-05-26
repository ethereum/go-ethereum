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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/csvlogger"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
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
	MaxHelperTrieProofsFetch = 64  // Amount of merkle proofs to be fetched per retrieval request
	MaxTxSend                = 64  // Amount of transactions to be send per request
	MaxTxStatus              = 256 // Amount of transactions to queried per request
)

// serverHandler is responsible for serving light client and process
// all incoming light requests.
type serverHandler struct {
	blockchain *core.BlockChain
	chainDb    ethdb.Database
	txpool     *core.TxPool
	server     *LesServer
	logger     *csvlogger.Logger

	closeCh chan struct{}  // Channel used to exit all background routines of handler.
	wg      sync.WaitGroup // WaitGroup used to track all background routines of handler.
	synced  func() bool    // Callback function used to determine whether local node is synced.
}

func newServerHandler(server *LesServer, blockchain *core.BlockChain, chainDb ethdb.Database, txpool *core.TxPool, logger *csvlogger.Logger, synced func() bool) *serverHandler {
	handler := &serverHandler{
		server:     server,
		blockchain: blockchain,
		chainDb:    chainDb,
		txpool:     txpool,
		logger:     logger,
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
	peer := newPeer(int(version), h.server.config.NetworkId, false, p, rw)
	h.wg.Add(1)
	defer h.wg.Done()
	return h.handle(peer)
}

func (h *serverHandler) handle(p *peer) error {
	// Reject light clients if server is not synced.
	if !h.synced() {
		h.logger.Event("Rejected (server is not synced), " + p.id)
		return p2p.DiscRequested
	}
	p.Log().Debug("Light Ethereum peer connected", "name", p.Name())

	// Execute the LES handshake
	var (
		head   = h.blockchain.CurrentHeader()
		hash   = head.Hash()
		number = head.Number.Uint64()
		td     = h.blockchain.GetTd(hash, number)
	)
	if err := p.Handshake(td, hash, number, h.blockchain.Genesis().Hash(), h.server); err != nil {
		h.logger.Event("Handshake error: " + err.Error() + ", " + p.id)
		p.Log().Debug("Light Ethereum handshake failed", "err", err)
		return err
	}
	defer p.fcClient.Disconnect()

	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}
	// Register the peer locally
	if err := h.server.peers.Register(p); err != nil {
		h.logger.Event("Peer registration error: " + err.Error() + ", " + p.id)
		p.Log().Error("Light Ethereum peer registration failed", "err", err)
		return err
	}
	defer h.logger.Event("Closed connection, " + p.id)
	defer h.server.peers.Unregister(p.id)

	// Spawn a main loop to handle all incoming messages.
	for {
		select {
		case err := <-p.errCh:
			h.logger.Event("Failed to send response: " + err.Error() + ", " + p.id)
			p.Log().Debug("Failed to send light ethereum response", "err", err)
			return err
		default:
		}
		if err := h.handleMsg(p); err != nil {
			h.logger.Event("Message handling error: " + err.Error() + ", " + p.id)
			p.Log().Debug("Light Ethereum message handling failed", "err", err)
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (h *serverHandler) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	p.Log().Trace("Light Ethereum message arrived", "code", msg.Code, "bytes", msg.Size)

	// Discard large message which exceeds the limitation.
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	var (
		maxCost uint64
		task    *servingTask
		respId  = p.responseID()
	)
	// accept returns an indicator whether the request can be served.
	// If so, deduct the max cost from the flow control buffer.
	accept := func(reqID, reqCnt, maxCnt uint64) bool {
		// Short circuit if the peer is already frozen or the request number is invalid.
		inSizeCost := h.server.costTracker.realCost(0, msg.Size, 0)
		if p.isFrozen() || reqCnt == 0 || reqCnt > maxCnt {
			p.fcClient.OneTimeCost(inSizeCost)
			return false
		}
		// Prepaid max cost units before request been serving.
		maxCost = p.fcCosts.getMaxCost(msg.Code, reqCnt)
		accepted, bufShort, priority := p.fcClient.AcceptRequest(reqID, respId, maxCost)
		if !accepted {
			p.freezeClient()
			p.Log().Error("Request came too early", "remaining", common.PrettyDuration(time.Duration(bufShort*1000000/p.fcParams.MinRecharge)))
			p.fcClient.OneTimeCost(inSizeCost)
			return false
		}
		// Create a multi-stage task, estimate the time it takes for the task to
		// execute, and cache it in the request service queue.
		factor := h.server.costTracker.globalFactor()
		if factor < 0.001 {
			factor = 1
		}
		maxTime := uint64(float64(maxCost) / factor)
		task = h.server.servingQueue.newTask(p, maxTime, priority)
		if task.start() {
			return true
		}
		p.fcClient.RequestProcessed(reqID, respId, maxCost, inSizeCost)
		return false
	}
	// sendResponse sends back the response and updates the flow control statistic.
	sendResponse := func(reqID, amount uint64, reply *reply, servingTime uint64) {
		// Short circuit if the client is already frozen.
		if p.isFrozen() {
			realCost := h.server.costTracker.realCost(servingTime, msg.Size, 0)
			p.fcClient.RequestProcessed(reqID, respId, maxCost, realCost)
			return
		}
		// Positive correction buffer value with real cost.
		var replySize uint32
		if reply != nil {
			replySize = reply.size()
		}
		realCost := h.server.costTracker.realCost(servingTime, msg.Size, replySize)
		if h.server.costTracker.disableRealCost {
			realCost = maxCost
		}
		bv := p.fcClient.RequestProcessed(reqID, respId, maxCost, realCost)
		// Feed cost tracker request serving statistic.
		if amount != 0 {
			h.server.costTracker.updateStats(msg.Code, amount, servingTime, realCost)
		}
		if reply != nil {
			p.queueSend(func() {
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
		var req struct {
			ReqID uint64
			Query getBlockHeadersData
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		query := req.Query
		if accept(req.ReqID, query.Amount, MaxHeaderFetch) {
			go func() {
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
				sendResponse(req.ReqID, query.Amount, p.ReplyBlockHeaders(req.ReqID, headers), task.done())
			}()
		}

	case GetBlockBodiesMsg:
		p.Log().Trace("Received block bodies request")
		var req struct {
			ReqID  uint64
			Hashes []common.Hash
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		var (
			bytes  int
			bodies []rlp.RawValue
		)
		reqCnt := len(req.Hashes)
		if accept(req.ReqID, uint64(reqCnt), MaxBodyFetch) {
			go func() {
				for i, hash := range req.Hashes {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					if bytes >= softResponseLimit {
						break
					}
					// Retrieve the requested block body, stopping if enough was found
					if number := rawdb.ReadHeaderNumber(h.chainDb, hash); number != nil {
						if data := rawdb.ReadBodyRLP(h.chainDb, hash, *number); len(data) != 0 {
							bodies = append(bodies, data)
							bytes += len(data)
						}
					}
				}
				sendResponse(req.ReqID, uint64(reqCnt), p.ReplyBlockBodiesRLP(req.ReqID, bodies), task.done())
			}()
		}

	case GetCodeMsg:
		p.Log().Trace("Received code request")
		var req struct {
			ReqID uint64
			Reqs  []CodeReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		var (
			bytes int
			data  [][]byte
		)
		reqCnt := len(req.Reqs)
		if accept(req.ReqID, uint64(reqCnt), MaxCodeFetch) {
			go func() {
				for i, request := range req.Reqs {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					// Look up the root hash belonging to the request
					number := rawdb.ReadHeaderNumber(h.chainDb, request.BHash)
					if number == nil {
						p.Log().Warn("Failed to retrieve block num for code", "hash", request.BHash)
						continue
					}
					header := rawdb.ReadHeader(h.chainDb, request.BHash, *number)
					if header == nil {
						p.Log().Warn("Failed to retrieve header for code", "block", *number, "hash", request.BHash)
						continue
					}
					triedb := h.blockchain.StateCache().TrieDB()

					account, err := h.getAccount(triedb, header.Root, common.BytesToHash(request.AccKey))
					if err != nil {
						p.Log().Warn("Failed to retrieve account for code", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(request.AccKey), "err", err)
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
				sendResponse(req.ReqID, uint64(reqCnt), p.ReplyCode(req.ReqID, data), task.done())
			}()
		}

	case GetReceiptsMsg:
		p.Log().Trace("Received receipts request")
		var req struct {
			ReqID  uint64
			Hashes []common.Hash
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		var (
			bytes    int
			receipts []rlp.RawValue
		)
		reqCnt := len(req.Hashes)
		if accept(req.ReqID, uint64(reqCnt), MaxReceiptFetch) {
			go func() {
				for i, hash := range req.Hashes {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					if bytes >= softResponseLimit {
						break
					}
					// Retrieve the requested block's receipts, skipping if unknown to us
					var results types.Receipts
					if number := rawdb.ReadHeaderNumber(h.chainDb, hash); number != nil {
						results = rawdb.ReadRawReceipts(h.chainDb, hash, *number)
					}
					if results == nil {
						if header := h.blockchain.GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
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
				sendResponse(req.ReqID, uint64(reqCnt), p.ReplyReceiptsRLP(req.ReqID, receipts), task.done())
			}()
		}

	case GetProofsV2Msg:
		p.Log().Trace("Received les/2 proofs request")
		var req struct {
			ReqID uint64
			Reqs  []ProofReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			lastBHash common.Hash
			root      common.Hash
		)
		reqCnt := len(req.Reqs)
		if accept(req.ReqID, uint64(reqCnt), MaxProofsFetch) {
			go func() {
				nodes := light.NewNodeSet()

				for i, request := range req.Reqs {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					// Look up the root hash belonging to the request
					var (
						number *uint64
						header *types.Header
						trie   state.Trie
					)
					if request.BHash != lastBHash {
						root, lastBHash = common.Hash{}, request.BHash

						if number = rawdb.ReadHeaderNumber(h.chainDb, request.BHash); number == nil {
							p.Log().Warn("Failed to retrieve block num for proof", "hash", request.BHash)
							continue
						}
						if header = rawdb.ReadHeader(h.chainDb, request.BHash, *number); header == nil {
							p.Log().Warn("Failed to retrieve header for proof", "block", *number, "hash", request.BHash)
							continue
						}
						root = header.Root
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
				sendResponse(req.ReqID, uint64(reqCnt), p.ReplyProofsV2(req.ReqID, nodes.NodeList()), task.done())
			}()
		}

	case GetHelperTrieProofsMsg:
		p.Log().Trace("Received helper trie proof request")
		var req struct {
			ReqID uint64
			Reqs  []HelperTrieReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			auxBytes int
			auxData  [][]byte
		)
		reqCnt := len(req.Reqs)
		if accept(req.ReqID, uint64(reqCnt), MaxHelperTrieProofsFetch) {
			go func() {
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
				sendResponse(req.ReqID, uint64(reqCnt), p.ReplyHelperTrieProofs(req.ReqID, HelperTrieResps{Proofs: nodes.NodeList(), AuxData: auxData}), task.done())
			}()
		}

	case SendTxV2Msg:
		p.Log().Trace("Received new transactions")
		var req struct {
			ReqID uint64
			Txs   []*types.Transaction
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(req.Txs)
		if accept(req.ReqID, uint64(reqCnt), MaxTxSend) {
			go func() {
				stats := make([]light.TxStatus, len(req.Txs))
				for i, tx := range req.Txs {
					if i != 0 && !task.waitOrStop() {
						return
					}
					hash := tx.Hash()
					stats[i] = h.txStatus(hash)
					if stats[i].Status == core.TxStatusUnknown {
						if errs := h.txpool.AddRemotes([]*types.Transaction{tx}); errs[0] != nil {
							stats[i].Error = errs[0].Error()
							continue
						}
						stats[i] = h.txStatus(hash)
					}
				}
				sendResponse(req.ReqID, uint64(reqCnt), p.ReplyTxStatus(req.ReqID, stats), task.done())
			}()
		}

	case GetTxStatusMsg:
		p.Log().Trace("Received transaction status query request")
		var req struct {
			ReqID  uint64
			Hashes []common.Hash
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(req.Hashes)
		if accept(req.ReqID, uint64(reqCnt), MaxTxStatus) {
			go func() {
				stats := make([]light.TxStatus, len(req.Hashes))
				for i, hash := range req.Hashes {
					if i != 0 && !task.waitOrStop() {
						sendResponse(req.ReqID, 0, nil, task.servingTime)
						return
					}
					stats[i] = h.txStatus(hash)
				}
				sendResponse(req.ReqID, uint64(reqCnt), p.ReplyTxStatus(req.ReqID, stats), task.done())
			}()
		}

	default:
		p.Log().Trace("Received invalid message", "code", msg.Code)
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
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
		if tx, blockHash, blockNumber, txIndex := rawdb.ReadTransaction(h.chainDb, hash); tx != nil {
			stat.Status = core.TxStatusIncluded
			stat.Lookup = &rawdb.LegacyTxLookupEntry{BlockHash: blockHash, BlockIndex: blockNumber, Index: txIndex}
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
		lastHead        *types.Header
		lastBroadcastTd = common.Big0
	)
	for {
		select {
		case ev := <-headCh:
			peers := h.server.peers.AllPeers()
			if len(peers) == 0 {
				continue
			}
			header := ev.Block.Header()
			hash, number := header.Hash(), header.Number.Uint64()
			td := rawdb.ReadTd(h.chainDb, hash, number)
			if td == nil || td.Cmp(lastBroadcastTd) <= 0 {
				continue
			}
			var reorg uint64
			if lastHead != nil {
				reorg = lastHead.Number.Uint64() - rawdb.FindCommonAncestor(h.chainDb, header, lastHead).Number.Uint64()
			}
			lastHead, lastBroadcastTd = header, td

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
					p.queueSend(func() { p.SendAnnounce(announce) })
				case announceTypeSigned:
					if !signed {
						signedAnnounce = announce
						signedAnnounce.sign(h.server.privateKey)
						signed = true
					}
					p.queueSend(func() { p.SendAnnounce(signedAnnounce) })
				}
			}
		case <-h.closeCh:
			return
		}
	}
}
