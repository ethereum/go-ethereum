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

	//"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// fcRequestWrapper performs flow control-related tasks for server side message handlers
type fcRequestWrapper struct {
	costTracker  *costTracker
	servingQueue *servingQueue
}

// wrapMessageHandlers returns the flow controlled version of the provided message handlers
func (f *fcRequestWrapper) wrapMessageHandlers(fcHandlers []FlowControlledHandler) messageHandlers {
	wrappedHandlers := make(messageHandlers, len(fcHandlers))
	for i, req := range fcHandlers {
		req := req
		wrappedHandlers[i] = messageHandlerWithCodeAndVersion{
			code:         req.Code,
			firstVersion: req.FirstVersion,
			lastVersion:  req.LastVersion,
			handler: func(p *peer, msg p2p.Msg) error {
				// Decode the p2p message, resolve the concrete handler for it.
				serve, reqID, reqCnt, err := req.Handle(msg)
				if err != nil {
					clientErrorMeter.Mark(1)
					return errResp(ErrDecode, "%v: %v", msg, err)
				}
				if metrics.EnabledExpensive {
					req.InPacketsMeter.Mark(1)
					req.InTrafficMeter.Mark(int64(msg.Size))
				}
				p.responseCount++
				responseCount := p.responseCount

				// Ensure that the request sent by client peer is valid
				inSizeCost := f.costTracker.realCost(0, msg.Size, 0)
				if reqCnt == 0 || reqCnt > req.MaxCount {
					p.fcClient.OneTimeCost(inSizeCost)
					return nil
				}
				// Ensure that the client peer complies with the flow control
				// rules agreed by both sides.
				if p.isFrozen() {
					p.fcClient.OneTimeCost(inSizeCost)
					return nil
				}
				maxCost := p.fcCosts.getMaxCost(msg.Code, reqCnt)
				accepted, bufShort, priority := p.fcClient.AcceptRequest(reqID, responseCount, maxCost)
				if !accepted {
					p.freezeClient()
					p.Log().Error("Flow control buffer too low", "time until sufficiently recharged", common.PrettyDuration(time.Duration(bufShort*1000000/p.fcParams.MinRecharge)))
					p.fcClient.OneTimeCost(inSizeCost)
					return nil
				}
				// Create a multi-stage task, estimate the time it takes for the task to
				// execute, and cache it in the request service queue.
				factor := f.costTracker.globalFactor()
				if factor < 0.001 {
					factor = 1
					p.Log().Error("Invalid global cost factor", "factor", factor)
				}
				maxTime := uint64(float64(maxCost) / factor)
				task := f.servingQueue.newTask(p, maxTime, priority)
				if !task.start() {
					p.fcClient.RequestProcessed(reqID, responseCount, maxCost, inSizeCost)
					return nil
				}
				p.wg.Add(1) //TODO ???
				go func() {
					defer p.wg.Done()

					reply := serve(p, task.waitOrStop)
					task.done()
					p.responseLock.Lock()
					defer p.responseLock.Unlock()

					// Short circuit if the client is already frozen.
					if p.isFrozen() {
						realCost := f.costTracker.realCost(task.servingTime, msg.Size, 0)
						p.fcClient.RequestProcessed(reqID, responseCount, maxCost, realCost)
						return
					}
					// Positive correction buffer value with real cost.
					var replySize uint32
					if reply != nil {
						replySize = reply.size()
					}
					var realCost uint64
					if f.costTracker.testing {
						realCost = maxCost // Assign a fake cost for testing purpose
					} else {
						realCost = f.costTracker.realCost(task.servingTime, msg.Size, replySize)
						if realCost > maxCost {
							realCost = maxCost
						}
					}
					bv := p.fcClient.RequestProcessed(reqID, responseCount, maxCost, realCost)
					if reply != nil {
						// Feed cost tracker request serving statistic.
						f.costTracker.updateStats(msg.Code, reqCnt, task.servingTime, realCost)
						// Reduce priority "balance" for the specific peer.
						if p.balance != nil {
							p.balance.RequestServed(realCost)
						}
						p.queueSend(func() {
							if err := reply.send(bv); err != nil {
								select {
								case p.errCh <- err:
								default:
								}
							}
						})
					}

					if metrics.EnabledExpensive {
						size := uint32(0)
						if reply != nil {
							size = reply.size()
						}
						req.OutPacketsMeter.Mark(1)
						req.OutTrafficMeter.Mark(int64(size))
						req.ServingTimeMeter.Update(time.Duration(task.servingTime))
					}
				}()
				// If the client has made too much invalid request(e.g. request a non-existent data),
				// reject them to prevent SPAM attack.
				if p.getInvalid() > maxRequestErrors {
					clientErrorMeter.Mark(1)
					return errTooManyInvalidRequest
				}
				return nil
			},
		}
	}
	return wrappedHandlers
}

// RequestServer serves general protocol messages (not including beacon chain-related ones)
type RequestServer struct {
	ArchiveMode   bool
	AddTxsSync    bool
	BlockChain    *core.BlockChain
	TxPool        *core.TxPool
	GetHelperTrie func(typ uint, index uint64) *trie.Trie
}

// Decoder is implemented by the messages passed to the handler functions
type Decoder interface {
	Decode(val interface{}) error
}

// FlowControlledRequest is a static struct that describes an LES request type and references
// its handler function.
type FlowControlledHandler struct {
	Name                                                             string
	Code                                                             uint64
	FirstVersion, LastVersion                                        int
	MaxCount                                                         uint64
	InPacketsMeter, InTrafficMeter, OutPacketsMeter, OutTrafficMeter metrics.Meter
	ServingTimeMeter                                                 metrics.Timer
	Handle                                                           func(msg Decoder) (serve serveRequestFn, reqID, amount uint64, err error)
}

// serveRequestFn is returned by the request handler functions after decoding the request.
// This function does the actual request serving using the supplied s waitOrStop is
// called between serving individual request items and may block if the serving process
// needs to be throttled. If it returns false then the process is terminated.
// The reply is not sent by this function yet. The flow control feedback value is supplied
// by the protocol handler when calling the send function of the returned reply struct.
type serveRequestFn func(peer *peer, waitOrStop func() bool) *reply

func (s *RequestServer) MessageHandlers() []FlowControlledHandler {
	return []FlowControlledHandler{
		{
			Code:             GetBlockHeadersMsg,
			Name:             "block header request",
			FirstVersion:     lpv2,
			LastVersion:      lpvLatest,
			MaxCount:         MaxHeaderFetch,
			InPacketsMeter:   miscInHeaderPacketsMeter,
			InTrafficMeter:   miscInHeaderTrafficMeter,
			OutPacketsMeter:  miscOutHeaderPacketsMeter,
			OutTrafficMeter:  miscOutHeaderTrafficMeter,
			ServingTimeMeter: miscServingTimeHeaderTimer,
			Handle:           s.handleGetBlockHeaders,
		},
		{
			Code:             GetBlockBodiesMsg,
			Name:             "block bodies request",
			FirstVersion:     lpv2,
			LastVersion:      lpvLatest,
			MaxCount:         MaxBodyFetch,
			InPacketsMeter:   miscInBodyPacketsMeter,
			InTrafficMeter:   miscInBodyTrafficMeter,
			OutPacketsMeter:  miscOutBodyPacketsMeter,
			OutTrafficMeter:  miscOutBodyTrafficMeter,
			ServingTimeMeter: miscServingTimeBodyTimer,
			Handle:           s.handleGetBlockBodies,
		},
		{
			Code:             GetCodeMsg,
			Name:             "code request",
			FirstVersion:     lpv2,
			LastVersion:      lpvLatest,
			MaxCount:         MaxCodeFetch,
			InPacketsMeter:   miscInCodePacketsMeter,
			InTrafficMeter:   miscInCodeTrafficMeter,
			OutPacketsMeter:  miscOutCodePacketsMeter,
			OutTrafficMeter:  miscOutCodeTrafficMeter,
			ServingTimeMeter: miscServingTimeCodeTimer,
			Handle:           s.handleGetCode,
		},
		{
			Code:             GetReceiptsMsg,
			Name:             "receipts request",
			FirstVersion:     lpv2,
			LastVersion:      lpvLatest,
			MaxCount:         MaxReceiptFetch,
			InPacketsMeter:   miscInReceiptPacketsMeter,
			InTrafficMeter:   miscInReceiptTrafficMeter,
			OutPacketsMeter:  miscOutReceiptPacketsMeter,
			OutTrafficMeter:  miscOutReceiptTrafficMeter,
			ServingTimeMeter: miscServingTimeReceiptTimer,
			Handle:           s.handleGetReceipts,
		},
		{
			Code:             GetProofsV2Msg,
			Name:             "les/2 proofs request",
			FirstVersion:     lpv2,
			LastVersion:      lpvLatest,
			MaxCount:         MaxProofsFetch,
			InPacketsMeter:   miscInTrieProofPacketsMeter,
			InTrafficMeter:   miscInTrieProofTrafficMeter,
			OutPacketsMeter:  miscOutTrieProofPacketsMeter,
			OutTrafficMeter:  miscOutTrieProofTrafficMeter,
			ServingTimeMeter: miscServingTimeTrieProofTimer,
			Handle:           s.handleGetProofs,
		},
		{
			Code:             GetHelperTrieProofsMsg,
			Name:             "helper trie proof request",
			FirstVersion:     lpv2,
			LastVersion:      lpv4,
			MaxCount:         MaxHelperTrieProofsFetch,
			InPacketsMeter:   miscInHelperTriePacketsMeter,
			InTrafficMeter:   miscInHelperTrieTrafficMeter,
			OutPacketsMeter:  miscOutHelperTriePacketsMeter,
			OutTrafficMeter:  miscOutHelperTrieTrafficMeter,
			ServingTimeMeter: miscServingTimeHelperTrieTimer,
			Handle:           s.handleGetHelperTrieProofs,
		},
		{
			Code:             SendTxV2Msg,
			Name:             "new transactions",
			FirstVersion:     lpv2,
			LastVersion:      lpvLatest,
			MaxCount:         MaxTxSend,
			InPacketsMeter:   miscInTxsPacketsMeter,
			InTrafficMeter:   miscInTxsTrafficMeter,
			OutPacketsMeter:  miscOutTxsPacketsMeter,
			OutTrafficMeter:  miscOutTxsTrafficMeter,
			ServingTimeMeter: miscServingTimeTxTimer,
			Handle:           s.handleSendTx,
		},
		{
			Code:             GetTxStatusMsg,
			Name:             "transaction status query request",
			FirstVersion:     lpv2,
			LastVersion:      lpvLatest,
			MaxCount:         MaxTxStatus,
			InPacketsMeter:   miscInTxStatusPacketsMeter,
			InTrafficMeter:   miscInTxStatusTrafficMeter,
			OutPacketsMeter:  miscOutTxStatusPacketsMeter,
			OutTrafficMeter:  miscOutTxStatusTrafficMeter,
			ServingTimeMeter: miscServingTimeTxStatusTimer,
			Handle:           s.handleGetTxStatus,
		},
	}
}

// handleGetBlockHeaders handles a block header request
func (s *RequestServer) handleGetBlockHeaders(msg Decoder) (serveRequestFn, uint64, uint64, error) {
	var r GetBlockHeadersPacket
	if err := msg.Decode(&r); err != nil {
		return nil, 0, 0, err
	}
	return func(p *peer, waitOrStop func() bool) *reply {
		// Gather headers until the fetch or network limits is reached
		var (
			bc              = s.BlockChain
			hashMode        = r.Query.Origin.Hash != (common.Hash{})
			first           = true
			maxNonCanonical = uint64(100)
			bytes           common.StorageSize
			headers         []*types.Header
			unknown         bool
		)
		for !unknown && len(headers) < int(r.Query.Amount) && bytes < softResponseLimit {
			if !first && !waitOrStop() {
				return nil
			}
			// Retrieve the next header satisfying the r
			var origin *types.Header
			if hashMode {
				if first {
					origin = bc.GetHeaderByHash(r.Query.Origin.Hash)
					if origin != nil {
						r.Query.Origin.Number = origin.Number.Uint64()
					}
				} else {
					origin = bc.GetHeader(r.Query.Origin.Hash, r.Query.Origin.Number)
				}
			} else {
				origin = bc.GetHeaderByNumber(r.Query.Origin.Number)
			}
			if origin == nil {
				break
			}
			headers = append(headers, origin)
			bytes += estHeaderRlpSize

			// Advance to the next header of the r
			switch {
			case hashMode && r.Query.Reverse:
				// Hash based traversal towards the genesis block
				ancestor := r.Query.Skip + 1
				if ancestor == 0 {
					unknown = true
				} else {
					r.Query.Origin.Hash, r.Query.Origin.Number = bc.GetAncestor(r.Query.Origin.Hash, r.Query.Origin.Number, ancestor, &maxNonCanonical)
					unknown = r.Query.Origin.Hash == common.Hash{}
				}
			case hashMode && !r.Query.Reverse:
				// Hash based traversal towards the leaf block
				var (
					current = origin.Number.Uint64()
					next    = current + r.Query.Skip + 1
				)
				if next <= current {
					infos, _ := json.Marshal(p.Peer.Info())
					p.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", r.Query.Skip, "next", next, "attacker", string(infos))
					unknown = true
				} else {
					if header := bc.GetHeaderByNumber(next); header != nil {
						nextHash := header.Hash()
						expOldHash, _ := bc.GetAncestor(nextHash, next, r.Query.Skip+1, &maxNonCanonical)
						if expOldHash == r.Query.Origin.Hash {
							r.Query.Origin.Hash, r.Query.Origin.Number = nextHash, next
						} else {
							unknown = true
						}
					} else {
						unknown = true
					}
				}
			case r.Query.Reverse:
				// Number based traversal towards the genesis block
				if r.Query.Origin.Number >= r.Query.Skip+1 {
					r.Query.Origin.Number -= r.Query.Skip + 1
				} else {
					unknown = true
				}

			case !r.Query.Reverse:
				// Number based traversal towards the leaf block
				r.Query.Origin.Number += r.Query.Skip + 1
			}
			first = false
		}
		return p.replyBlockHeaders(r.ReqID, headers)
	}, r.ReqID, r.Query.Amount, nil
}

// handleGetBlockBodies handles a block body request
func (s *RequestServer) handleGetBlockBodies(msg Decoder) (serveRequestFn, uint64, uint64, error) {
	var r GetBlockBodiesPacket
	if err := msg.Decode(&r); err != nil {
		return nil, 0, 0, err
	}
	return func(p *peer, waitOrStop func() bool) *reply {
		var (
			bytes  int
			bodies []rlp.RawValue
		)
		bc := s.BlockChain
		for i, hash := range r.Hashes {
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
		return p.replyBlockBodiesRLP(r.ReqID, bodies)
	}, r.ReqID, uint64(len(r.Hashes)), nil
}

// handleGetCode handles a contract code request
func (s *RequestServer) handleGetCode(msg Decoder) (serveRequestFn, uint64, uint64, error) {
	var r GetCodePacket
	if err := msg.Decode(&r); err != nil {
		return nil, 0, 0, err
	}
	return func(p *peer, waitOrStop func() bool) *reply {
		var (
			bytes int
			data  [][]byte
		)
		bc := s.BlockChain
		for i, request := range r.Reqs {
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
			if !s.ArchiveMode && header.Number.Uint64()+core.TriesInMemory <= local {
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
		return p.replyCode(r.ReqID, data)
	}, r.ReqID, uint64(len(r.Reqs)), nil
}

// handleGetReceipts handles a block receipts request
func (s *RequestServer) handleGetReceipts(msg Decoder) (serveRequestFn, uint64, uint64, error) {
	var r GetReceiptsPacket
	if err := msg.Decode(&r); err != nil {
		return nil, 0, 0, err
	}
	return func(p *peer, waitOrStop func() bool) *reply {
		var (
			bytes    int
			receipts []rlp.RawValue
		)
		bc := s.BlockChain
		for i, hash := range r.Hashes {
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
		return p.replyReceiptsRLP(r.ReqID, receipts)
	}, r.ReqID, uint64(len(r.Hashes)), nil
}

// handleGetProofs handles a proof request
func (s *RequestServer) handleGetProofs(msg Decoder) (serveRequestFn, uint64, uint64, error) {
	var r GetProofsPacket
	if err := msg.Decode(&r); err != nil {
		return nil, 0, 0, err
	}
	return func(p *peer, waitOrStop func() bool) *reply {
		var (
			lastBHash common.Hash
			root      common.Hash
			header    *types.Header
			err       error
		)
		bc := s.BlockChain
		nodes := light.NewNodeSet()

		for i, request := range r.Reqs {
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
				if !s.ArchiveMode && header.Number.Uint64()+core.TriesInMemory <= local {
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
		return p.replyProofsV2(r.ReqID, nodes.NodeList())
	}, r.ReqID, uint64(len(r.Reqs)), nil
}

// handleGetHelperTrieProofs handles a helper trie proof request
func (s *RequestServer) handleGetHelperTrieProofs(msg Decoder) (serveRequestFn, uint64, uint64, error) {
	var r GetHelperTrieProofsPacket
	if err := msg.Decode(&r); err != nil {
		return nil, 0, 0, err
	}
	return func(p *peer, waitOrStop func() bool) *reply {
		var (
			lastIdx  uint64
			lastType uint
			auxTrie  *trie.Trie
			auxBytes int
			auxData  [][]byte
		)
		bc := s.BlockChain
		nodes := light.NewNodeSet()
		for i, request := range r.Reqs {
			if i != 0 && !waitOrStop() {
				return nil
			}
			if auxTrie == nil || request.Type != lastType || request.TrieIdx != lastIdx {
				lastType, lastIdx = request.Type, request.TrieIdx
				auxTrie = s.GetHelperTrie(request.Type, request.TrieIdx)
			}
			if auxTrie == nil {
				return nil
			}
			// TODO(rjl493456442) short circuit if the proving is failed.
			// The original client side code has a dirty hack to retrieve
			// the headers with no valid proof. Keep the compatibility for
			// legacy les protocol and drop this hack when the les2/3 are
			// not supported.
			err := auxTrie.Prove(request.Key, request.FromLevel, nodes)
			if p.version >= lpv4 && err != nil {
				return nil
			}
			if request.Type == htCanonical && request.AuxReq == htAuxHeader && len(request.Key) == 8 {
				header := bc.GetHeaderByNumber(binary.BigEndian.Uint64(request.Key))
				data, err := rlp.EncodeToBytes(header)
				if err != nil {
					log.Error("Failed to encode header", "err", err)
					return nil
				}
				auxData = append(auxData, data)
				auxBytes += len(data)
			}
			if nodes.DataSize()+auxBytes >= softResponseLimit {
				break
			}
		}
		return p.replyHelperTrieProofs(r.ReqID, HelperTrieResps{Proofs: nodes.NodeList(), AuxData: auxData})
	}, r.ReqID, uint64(len(r.Reqs)), nil
}

// handleSendTx handles a transaction propagation request
func (s *RequestServer) handleSendTx(msg Decoder) (serveRequestFn, uint64, uint64, error) {
	var r SendTxPacket
	if err := msg.Decode(&r); err != nil {
		return nil, 0, 0, err
	}
	amount := uint64(len(r.Txs))
	return func(p *peer, waitOrStop func() bool) *reply {
		stats := make([]light.TxStatus, len(r.Txs))
		for i, tx := range r.Txs {
			if i != 0 && !waitOrStop() {
				return nil
			}
			hash := tx.Hash()
			stats[i] = s.txStatus(hash)
			if stats[i].Status == core.TxStatusUnknown {
				addFn := s.TxPool.AddRemotes
				// Add txs synchronously for testing purpose
				if s.AddTxsSync {
					addFn = s.TxPool.AddRemotesSync
				}
				if errs := addFn([]*types.Transaction{tx}); errs[0] != nil {
					stats[i].Error = errs[0].Error()
					continue
				}
				stats[i] = s.txStatus(hash)
			}
		}
		return p.replyTxStatus(r.ReqID, stats)
	}, r.ReqID, amount, nil
}

// handleGetTxStatus handles a transaction status query
func (s *RequestServer) handleGetTxStatus(msg Decoder) (serveRequestFn, uint64, uint64, error) {
	var r GetTxStatusPacket
	if err := msg.Decode(&r); err != nil {
		return nil, 0, 0, err
	}
	return func(p *peer, waitOrStop func() bool) *reply {
		stats := make([]light.TxStatus, len(r.Hashes))
		for i, hash := range r.Hashes {
			if i != 0 && !waitOrStop() {
				return nil
			}
			stats[i] = s.txStatus(hash)
		}
		return p.replyTxStatus(r.ReqID, stats)
	}, r.ReqID, uint64(len(r.Hashes)), nil
}

// txStatus returns the status of a specified transaction.
func (s *RequestServer) txStatus(hash common.Hash) light.TxStatus {
	var stat light.TxStatus
	// Looking the transaction in txpool first.
	stat.Status = s.TxPool.Status([]common.Hash{hash})[0]

	// If the transaction is unknown to the pool, try looking it up locally.
	if stat.Status == core.TxStatusUnknown {
		lookup := s.BlockChain.GetTransactionLookup(hash)
		if lookup != nil {
			stat.Status = core.TxStatusIncluded
			stat.Lookup = lookup
		}
	}
	return stat
}
