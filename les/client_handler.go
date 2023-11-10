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
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/les/downloader"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

// clientHandler is responsible for receiving and processing all incoming server
// responses.
type clientHandler struct {
	ulc        *ulc
	forkFilter forkid.Filter
	fetcher    *lightFetcher
	downloader *downloader.Downloader
	backend    *LightEthereum

	closeCh chan struct{}
	wg      sync.WaitGroup // WaitGroup used to track all connected peers.

	// Hooks used in the testing
	syncStart func(header *types.Header) // Hook called when the syncing is started
	syncEnd   func(header *types.Header) // Hook called when the syncing is done
}

func newClientHandler(ulcServers []string, ulcFraction int, backend *LightEthereum) *clientHandler {
	handler := &clientHandler{
		forkFilter: forkid.NewFilter(backend.blockchain),
		backend:    backend,
		closeCh:    make(chan struct{}),
	}
	if ulcServers != nil {
		ulc, err := newULC(ulcServers, ulcFraction)
		if err != nil {
			log.Error("Failed to initialize ultra light client")
		}
		handler.ulc = ulc
		log.Info("Enable ultra light client mode")
	}
	handler.fetcher = newLightFetcher(backend.blockchain, backend.engine, backend.peers, handler.ulc, backend.chainDb, backend.reqDist, handler.synchronise)
	handler.downloader = downloader.New(0, backend.chainDb, backend.eventMux, nil, backend.blockchain, handler.removePeer)
	handler.backend.peers.subscribe((*downloaderPeerNotify)(handler))
	return handler
}

func (h *clientHandler) start() {
	h.fetcher.start()
}

func (h *clientHandler) stop() {
	close(h.closeCh)
	h.downloader.Terminate()
	h.fetcher.stop()
	h.wg.Wait()
}

// runPeer is the p2p protocol run function for the given version.
func (h *clientHandler) runPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter) error {
	trusted := false
	if h.ulc != nil {
		trusted = h.ulc.trusted(p.ID())
	}
	peer := newServerPeer(int(version), h.backend.config.NetworkId, trusted, p, newMeteredMsgWriter(rw, int(version)))
	defer peer.close()
	h.wg.Add(1)
	defer h.wg.Done()
	err := h.handle(peer, false)
	return err
}

func (h *clientHandler) handle(p *serverPeer, noInitAnnounce bool) error {
	if h.backend.peers.len() >= h.backend.config.LightPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("Light Ethereum peer connected", "name", p.Name())

	// Execute the LES handshake
	forkid := forkid.NewID(h.backend.blockchain.Config(), h.backend.genesis, h.backend.blockchain.CurrentHeader().Number.Uint64(), h.backend.blockchain.CurrentHeader().Time)
	if err := p.Handshake(h.backend.blockchain.Genesis().Hash(), forkid, h.forkFilter); err != nil {
		p.Log().Debug("Light Ethereum handshake failed", "err", err)
		return err
	}
	// Register peer with the server pool
	if h.backend.serverPool != nil {
		if nvt, err := h.backend.serverPool.RegisterNode(p.Node()); err == nil {
			p.setValueTracker(nvt)
			p.updateVtParams()
			defer func() {
				p.setValueTracker(nil)
				h.backend.serverPool.UnregisterNode(p.Node())
			}()
		} else {
			return err
		}
	}
	// Register the peer locally
	if err := h.backend.peers.register(p); err != nil {
		p.Log().Error("Light Ethereum peer registration failed", "err", err)
		return err
	}

	serverConnectionGauge.Update(int64(h.backend.peers.len()))

	connectedAt := mclock.Now()
	defer func() {
		h.backend.peers.unregister(p.id)
		connectionTimer.Update(time.Duration(mclock.Now() - connectedAt))
		serverConnectionGauge.Update(int64(h.backend.peers.len()))
	}()

	// Discard all the announces after the transition
	// Also discarding initial signal to prevent syncing during testing.
	if !(noInitAnnounce || h.backend.merger.TDDReached()) {
		h.fetcher.announce(p, &announceData{Hash: p.headInfo.Hash, Number: p.headInfo.Number, Td: p.headInfo.Td})
	}

	// Mark the peer starts to be served.
	p.serving.Store(true)
	defer p.serving.Store(false)

	// Spawn a main loop to handle all incoming messages.
	for {
		if err := h.handleMsg(p); err != nil {
			p.Log().Debug("Light Ethereum message handling failed", "err", err)
			p.fcServer.DumpLogs()
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (h *clientHandler) handleMsg(p *serverPeer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	p.Log().Trace("Light Ethereum message arrived", "code", msg.Code, "bytes", msg.Size)

	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	var deliverMsg *Msg

	// Handle the message depending on its contents
	switch {
	case msg.Code == AnnounceMsg:
		p.Log().Trace("Received announce message")
		var req announceData
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if err := req.sanityCheck(); err != nil {
			return err
		}
		update, size := req.Update.decode()
		if p.rejectUpdate(size) {
			return errResp(ErrRequestRejected, "")
		}
		p.updateFlowControl(update)
		p.updateVtParams()

		if req.Hash != (common.Hash{}) {
			if p.announceType == announceTypeNone {
				return errResp(ErrUnexpectedResponse, "")
			}
			if p.announceType == announceTypeSigned {
				if err := req.checkSignature(p.ID(), update); err != nil {
					p.Log().Trace("Invalid announcement signature", "err", err)
					return err
				}
				p.Log().Trace("Valid announcement signature")
			}
			p.Log().Trace("Announce message content", "number", req.Number, "hash", req.Hash, "td", req.Td, "reorg", req.ReorgDepth)

			// Update peer head information first and then notify the announcement
			p.updateHead(req.Hash, req.Number, req.Td)

			// Discard all the announces after the transition
			if !h.backend.merger.TDDReached() {
				h.fetcher.announce(p, &req)
			}
		}
	case msg.Code == BlockHeadersMsg:
		p.Log().Trace("Received block header response message")
		var resp struct {
			ReqID, BV uint64
			Headers   []*types.Header
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		headers := resp.Headers
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		p.answeredRequest(resp.ReqID)

		// Filter out the explicitly requested header by the retriever
		if h.backend.retriever.requested(resp.ReqID) {
			deliverMsg = &Msg{
				MsgType: MsgBlockHeaders,
				ReqID:   resp.ReqID,
				Obj:     resp.Headers,
			}
		} else {
			// Filter out any explicitly requested headers, deliver the rest to the downloader
			filter := len(headers) == 1
			if filter {
				headers = h.fetcher.deliverHeaders(p, resp.ReqID, resp.Headers)
			}
			if len(headers) != 0 || !filter {
				if err := h.downloader.DeliverHeaders(p.id, headers); err != nil {
					log.Debug("Failed to deliver headers", "err", err)
				}
			}
		}
	case msg.Code == BlockBodiesMsg:
		p.Log().Trace("Received block bodies response")
		var resp struct {
			ReqID, BV uint64
			Data      []*types.Body
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		p.answeredRequest(resp.ReqID)
		deliverMsg = &Msg{
			MsgType: MsgBlockBodies,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}
	case msg.Code == CodeMsg:
		p.Log().Trace("Received code response")
		var resp struct {
			ReqID, BV uint64
			Data      [][]byte
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		p.answeredRequest(resp.ReqID)
		deliverMsg = &Msg{
			MsgType: MsgCode,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}
	case msg.Code == ReceiptsMsg:
		p.Log().Trace("Received receipts response")
		var resp struct {
			ReqID, BV uint64
			Receipts  []types.Receipts
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		p.answeredRequest(resp.ReqID)
		deliverMsg = &Msg{
			MsgType: MsgReceipts,
			ReqID:   resp.ReqID,
			Obj:     resp.Receipts,
		}
	case msg.Code == ProofsV2Msg:
		p.Log().Trace("Received les/2 proofs response")
		var resp struct {
			ReqID, BV uint64
			Data      light.NodeList
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		p.answeredRequest(resp.ReqID)
		deliverMsg = &Msg{
			MsgType: MsgProofsV2,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}
	case msg.Code == HelperTrieProofsMsg:
		p.Log().Trace("Received helper trie proof response")
		var resp struct {
			ReqID, BV uint64
			Data      HelperTrieResps
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		p.answeredRequest(resp.ReqID)
		deliverMsg = &Msg{
			MsgType: MsgHelperTrieProofs,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}
	case msg.Code == TxStatusMsg:
		p.Log().Trace("Received tx status response")
		var resp struct {
			ReqID, BV uint64
			Status    []light.TxStatus
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		p.answeredRequest(resp.ReqID)
		deliverMsg = &Msg{
			MsgType: MsgTxStatus,
			ReqID:   resp.ReqID,
			Obj:     resp.Status,
		}
	case msg.Code == StopMsg && p.version >= lpv3:
		p.freeze()
		h.backend.retriever.frozen(p)
		p.Log().Debug("Service stopped")
	case msg.Code == ResumeMsg && p.version >= lpv3:
		var bv uint64
		if err := msg.Decode(&bv); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ResumeFreeze(bv)
		p.unfreeze()
		p.Log().Debug("Service resumed")
	default:
		p.Log().Trace("Received invalid message", "code", msg.Code)
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	// Deliver the received response to retriever.
	if deliverMsg != nil {
		if err := h.backend.retriever.deliver(p, deliverMsg); err != nil {
			if val := p.errCount.Add(1, mclock.Now()); val > maxResponseErrors {
				return err
			}
		}
	}
	return nil
}

func (h *clientHandler) removePeer(id string) {
	h.backend.peers.unregister(id)
}

type peerConnection struct {
	handler *clientHandler
	peer    *serverPeer
}

func (pc *peerConnection) Head() (common.Hash, *big.Int) {
	return pc.peer.HeadAndTd()
}

func (pc *peerConnection) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			peer := dp.(*serverPeer)
			return peer.getRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*serverPeer) == pc.peer
		},
		request: func(dp distPeer) func() {
			reqID := rand.Uint64()
			peer := dp.(*serverPeer)
			cost := peer.getRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.requestHeadersByHash(reqID, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pc.handler.backend.reqDist.queue(rq)
	if !ok {
		return light.ErrNoPeers
	}
	return nil
}

func (pc *peerConnection) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			peer := dp.(*serverPeer)
			return peer.getRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*serverPeer) == pc.peer
		},
		request: func(dp distPeer) func() {
			reqID := rand.Uint64()
			peer := dp.(*serverPeer)
			cost := peer.getRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.requestHeadersByNumber(reqID, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pc.handler.backend.reqDist.queue(rq)
	if !ok {
		return light.ErrNoPeers
	}
	return nil
}

// RetrieveSingleHeaderByNumber requests a single header by the specified block
// number. This function will wait the response until it's timeout or delivered.
func (pc *peerConnection) RetrieveSingleHeaderByNumber(context context.Context, number uint64) (*types.Header, error) {
	reqID := rand.Uint64()
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			peer := dp.(*serverPeer)
			return peer.getRequestCost(GetBlockHeadersMsg, 1)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*serverPeer) == pc.peer
		},
		request: func(dp distPeer) func() {
			peer := dp.(*serverPeer)
			cost := peer.getRequestCost(GetBlockHeadersMsg, 1)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.requestHeadersByNumber(reqID, number, 1, 0, false) }
		},
	}
	var header *types.Header
	if err := pc.handler.backend.retriever.retrieve(context, reqID, rq, func(peer distPeer, msg *Msg) error {
		if msg.MsgType != MsgBlockHeaders {
			return errInvalidMessageType
		}
		headers := msg.Obj.([]*types.Header)
		if len(headers) != 1 {
			return errInvalidEntryCount
		}
		header = headers[0]
		return nil
	}, nil); err != nil {
		return nil, err
	}
	return header, nil
}

// downloaderPeerNotify implements peerSetNotify
type downloaderPeerNotify clientHandler

func (d *downloaderPeerNotify) registerPeer(p *serverPeer) {
	h := (*clientHandler)(d)
	pc := &peerConnection{
		handler: h,
		peer:    p,
	}
	h.downloader.RegisterLightPeer(p.id, eth.ETH66, pc)
}

func (d *downloaderPeerNotify) unregisterPeer(p *serverPeer) {
	h := (*clientHandler)(d)
	h.downloader.UnregisterPeer(p.id)
}
