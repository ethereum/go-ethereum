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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/les/downloader"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	vfc "github.com/ethereum/go-ethereum/les/vflux/client"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

// clientHandler is responsible for receiving and processing all incoming server
// responses.
type clientHandler struct {
	forkFilter forkid.Filter
	blockchain *light.LightChain
	peers      *peerSet
	retriever  *retrieveManager

	// only for PoW mode
	fetcher        *lightFetcher
	downloader     *downloader.Downloader
	noInitAnnounce bool
}

// sendHandshake implements handshakeModule
func (h *clientHandler) sendHandshake(p *peer, send *keyValueList) {
	sendGeneralInfo(p, send, h.blockchain.Genesis().Hash(), forkid.NewID(h.blockchain.Config(), h.blockchain.Genesis().Hash(), h.blockchain.CurrentHeader().Number.Uint64()))
	p.announceType = announceTypeSimple
	send.add("announceType", p.announceType)
	sendHeadInfo(send, blockInfo{})
}

// receiveHandshake implements handshakeModule
func (h *clientHandler) receiveHandshake(p *peer, recv keyValueMap) error {
	if err := receiveGeneralInfo(p, recv, h.blockchain.Genesis().Hash(), h.forkFilter); err != nil {
		return err
	}

	var (
		rHash common.Hash
		rNum  uint64
		rTd   *big.Int
	)
	if err := recv.get("headTd", &rTd); err != nil {
		return err
	}
	if err := recv.get("headHash", &rHash); err != nil {
		return err
	}
	if err := recv.get("headNum", &rNum); err != nil {
		return err
	}
	p.headInfo = blockInfo{Hash: rHash, Number: rNum, Td: rTd}
	if recv.get("serveChainSince", &p.chainSince) != nil {
		return errResp(ErrUselessPeer, "peer cannot serve requests")
	}
	if recv.get("serveRecentChain", &p.chainRecent) != nil {
		p.chainRecent = 0
	}
	if recv.get("serveStateSince", &p.stateSince) != nil {
		return errResp(ErrUselessPeer, "peer cannot serve requests")
	}
	if recv.get("serveRecentState", &p.stateRecent) != nil {
		p.stateRecent = 0
	}
	if recv.get("txRelay", nil) != nil {
		return errResp(ErrUselessPeer, "peer cannot serve requests")
	}
	if p.version >= lpv4 {
		var recentTx uint
		if err := recv.get("recentTxLookup", &recentTx); err != nil {
			return err
		}
		p.txHistory = uint64(recentTx)
	} else {
		// The weak assumption is held here that legacy les server(les2,3)
		// has unlimited transaction history. The les serving in these legacy
		// versions is disabled if the transaction is unindexed.
		p.txHistory = txIndexUnlimited
	}

	recv.get("checkpoint/value", &p.checkpoint)
	recv.get("checkpoint/registerHeight", &p.checkpointNumber)
	return nil
}

// peerConnected implements connectionModule
func (h *clientHandler) peerConnected(p *peer) (func(), error) {
	/*if h.peers.len() >= h.blockchain.Config().LightPeers && !p.Peer.Info().Network.Trusted { //TODO ???
		return nil, p2p.DiscTooManyPeers
	}*/
	serverConnectionGauge.Update(int64(h.peers.len()))
	// Discard all the announces after the transition
	// Also discarding initial signal to prevent syncing during testing.
	if h.fetcher != nil && !h.noInitAnnounce {
		h.fetcher.announce(p, &announceData{Hash: p.headInfo.Hash, Number: p.headInfo.Number, Td: p.headInfo.Td})
	}

	return func() {
		p.fcServer.DumpLogs()
		serverConnectionGauge.Update(int64(h.peers.len()))
	}, nil
}

// messageHandlers implements messageHandlerModule
func (h *clientHandler) messageHandlers() messageHandlers {
	return messageHandlers{
		messageHandlerWithCodeAndVersion{
			code:         AnnounceMsg,
			firstVersion: lpv2,
			lastVersion:  lpv4,
			handler:      h.handleAnnounce,
		},
		messageHandlerWithCodeAndVersion{
			code:         BlockHeadersMsg,
			firstVersion: lpv2,
			lastVersion:  lpvLatest,
			handler:      h.handleBlockHeaders,
		},
		messageHandlerWithCodeAndVersion{
			code:         BlockBodiesMsg,
			firstVersion: lpv2,
			lastVersion:  lpvLatest,
			handler:      h.handleBlockBodies,
		},
		messageHandlerWithCodeAndVersion{
			code:         CodeMsg,
			firstVersion: lpv2,
			lastVersion:  lpvLatest,
			handler:      h.handleCode,
		},
		messageHandlerWithCodeAndVersion{
			code:         ReceiptsMsg,
			firstVersion: lpv2,
			lastVersion:  lpvLatest,
			handler:      h.handleReceipts,
		},
		messageHandlerWithCodeAndVersion{
			code:         ProofsV2Msg,
			firstVersion: lpv2,
			lastVersion:  lpvLatest,
			handler:      h.handleProofsV2,
		},
		messageHandlerWithCodeAndVersion{
			code:         HelperTrieProofsMsg,
			firstVersion: lpv2,
			lastVersion:  lpv4,
			handler:      h.handleHelperTrieProofs,
		},
		messageHandlerWithCodeAndVersion{
			code:         TxStatusMsg,
			firstVersion: lpv2,
			lastVersion:  lpvLatest,
			handler:      h.handleTxStatus,
		},
	}
}

func (h *clientHandler) handleAnnounce(p *peer, msg p2p.Msg) error {
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
		if h.fetcher != nil {
			h.fetcher.announce(p, &req)
		}
	}
	return nil
}

func (h *clientHandler) handleBlockHeaders(p *peer, msg p2p.Msg) error {
	p.Log().Trace("Received block header response message")
	var resp struct {
		ReqID, BV uint64
		Headers   []*types.Header
	}
	if err := msg.Decode(&resp); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
	p.answeredRequest(resp.ReqID)

	// Filter out the explicitly requested header by the retriever
	if h.retriever.requested(resp.ReqID) {
		return deliverResponse(h.retriever, p, &Msg{
			MsgType: MsgBlockHeaders,
			ReqID:   resp.ReqID,
			Obj:     resp.Headers,
		})
	} else {
		// Filter out any explicitly requested headers, deliver the rest to the downloader
		headers := resp.Headers
		filter := len(headers) == 1
		if filter {
			headers = h.fetcher.deliverHeaders(p, resp.ReqID, resp.Headers)
		}
		if len(headers) != 0 || !filter {
			if err := h.downloader.DeliverHeaders(p.id, headers); err != nil {
				log.Debug("Failed to deliver headers", "err", err)
			}
		}
		return nil
	}
}

func (h *clientHandler) handleBlockBodies(p *peer, msg p2p.Msg) error {
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
	return deliverResponse(h.retriever, p, &Msg{
		MsgType: MsgBlockBodies,
		ReqID:   resp.ReqID,
		Obj:     resp.Data,
	})
}

func (h *clientHandler) handleCode(p *peer, msg p2p.Msg) error {
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
	return deliverResponse(h.retriever, p, &Msg{
		MsgType: MsgCode,
		ReqID:   resp.ReqID,
		Obj:     resp.Data,
	})
}

func (h *clientHandler) handleReceipts(p *peer, msg p2p.Msg) error {
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
	return deliverResponse(h.retriever, p, &Msg{
		MsgType: MsgReceipts,
		ReqID:   resp.ReqID,
		Obj:     resp.Receipts,
	})
}

func (h *clientHandler) handleProofsV2(p *peer, msg p2p.Msg) error {
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
	return deliverResponse(h.retriever, p, &Msg{
		MsgType: MsgProofsV2,
		ReqID:   resp.ReqID,
		Obj:     resp.Data,
	})
}

func (h *clientHandler) handleHelperTrieProofs(p *peer, msg p2p.Msg) error {
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
	return deliverResponse(h.retriever, p, &Msg{
		MsgType: MsgHelperTrieProofs,
		ReqID:   resp.ReqID,
		Obj:     resp.Data,
	})
}

func (h *clientHandler) handleTxStatus(p *peer, msg p2p.Msg) error {
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
	return deliverResponse(h.retriever, p, &Msg{
		MsgType: MsgTxStatus,
		ReqID:   resp.ReqID,
		Obj:     resp.Status,
	})
}

func deliverResponse(retriever *retrieveManager, p *peer, deliverMsg *Msg) error {
	if err := retriever.deliver(p, deliverMsg); err != nil {
		if val := p.bumpInvalid(); val > maxResponseErrors {
			return err
		}
	}
	return nil
}

// fcClientHandler performs flow control-related protocol handler tasks
type fcClientHandler struct {
	retriever *retrieveManager
}

// sendHandshake implements handshakeModule
func (h *fcClientHandler) sendHandshake(p *peer, send *keyValueList) {}

// receiveHandshake implements handshakeModule
func (h *fcClientHandler) receiveHandshake(p *peer, recv keyValueMap) error {
	// Parse flow control handshake packet.
	var sParams flowcontrol.ServerParams
	if err := recv.get("flowControl/BL", &sParams.BufLimit); err != nil {
		return err
	}
	if err := recv.get("flowControl/MRR", &sParams.MinRecharge); err != nil {
		return err
	}
	var MRC RequestCostList
	if err := recv.get("flowControl/MRC", &MRC); err != nil {
		return err
	}
	p.fcParams = sParams
	p.fcServer = flowcontrol.NewServerNode(sParams, &mclock.System{})
	p.fcCosts = MRC.decode(ProtocolLengths[uint(p.version)])
	for msgCode := range reqAvgTimeCost {
		if p.fcCosts[msgCode] == nil {
			return errResp(ErrUselessPeer, "peer does not support message %d", msgCode)
		}
	}
	return nil
}

// messageHandlers implements messageHandlerModule
func (h *fcClientHandler) messageHandlers() messageHandlers {
	return messageHandlers{
		messageHandlerWithCodeAndVersion{
			code:         StopMsg,
			firstVersion: lpv3,
			lastVersion:  lpvLatest,
			handler:      h.handleStop,
		},
		messageHandlerWithCodeAndVersion{
			code:         ResumeMsg,
			firstVersion: lpv3,
			lastVersion:  lpvLatest,
			handler:      h.handleResume,
		},
	}
}

func (h *fcClientHandler) handleStop(p *peer, msg p2p.Msg) error {
	p.freezeServer()
	h.retriever.frozen(p)
	p.Log().Debug("Service stopped")
	return nil
}

func (h *fcClientHandler) handleResume(p *peer, msg p2p.Msg) error {
	var bv uint64
	if err := msg.Decode(&bv); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	p.fcServer.ResumeFreeze(bv)
	p.unfreezeServer()
	p.Log().Debug("Service resumed")
	return nil
}

// vfxClientHandler performs vflux-related protocol handler tasks (registers servers into the server pool)
type vfxClientHandler struct {
	serverPool *vfc.ServerPool
}

// peerConnected implements connectionModule
func (h *vfxClientHandler) peerConnected(p *peer) (func(), error) {
	// Register peer with the server pool
	if h.serverPool != nil {
		if nvt, err := h.serverPool.RegisterNode(p.Node()); err == nil {
			p.setValueTracker(nvt)
			p.updateVtParams()
		} else {
			return nil, err
		}
	}

	return func() {
		p.setValueTracker(nil)
		h.serverPool.UnregisterNode(p.Node())
	}, nil
}

// downloaderPeerNotify implements peerSetNotify
type downloaderPeerNotify struct {
	retriever  *retrieveManager
	downloader *downloader.Downloader
}

type downloaderPeer struct {
	peer *peer
	*downloaderPeerNotify
}

func (pc *downloaderPeer) Head() (common.Hash, *big.Int) {
	return pc.peer.HeadAndTd()
}

func (pc *downloaderPeer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			peer := dp.(*peer)
			return peer.getRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*peer) == pc.peer
		},
		request: func(dp distPeer) func() {
			reqID := rand.Uint64()
			peer := dp.(*peer)
			cost := peer.getRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.requestHeadersByHash(reqID, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pc.retriever.dist.queue(rq)
	if !ok {
		return light.ErrNoPeers
	}
	return nil
}

func (pc *downloaderPeer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			peer := dp.(*peer)
			return peer.getRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*peer) == pc.peer
		},
		request: func(dp distPeer) func() {
			reqID := rand.Uint64()
			peer := dp.(*peer)
			cost := peer.getRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.requestHeadersByNumber(reqID, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pc.retriever.dist.queue(rq)
	if !ok {
		return light.ErrNoPeers
	}
	return nil
}

// RetrieveSingleHeaderByNumber requests a single header by the specified block
// number. This function will wait the response until it's timeout or delivered.
func (pc *downloaderPeer) RetrieveSingleHeaderByNumber(context context.Context, number uint64) (*types.Header, error) {
	reqID := rand.Uint64()
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			peer := dp.(*peer)
			return peer.getRequestCost(GetBlockHeadersMsg, 1)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*peer) == pc.peer
		},
		request: func(dp distPeer) func() {
			peer := dp.(*peer)
			cost := peer.getRequestCost(GetBlockHeadersMsg, 1)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.requestHeadersByNumber(reqID, number, 1, 0, false) }
		},
	}
	var header *types.Header
	if err := pc.retriever.retrieve(context, reqID, rq, func(peer distPeer, msg *Msg) error {
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

func (d *downloaderPeerNotify) registerPeer(p *peer) {
	pc := &downloaderPeer{
		peer:                 p,
		downloaderPeerNotify: d,
	}
	d.downloader.RegisterLightPeer(p.id, eth.ETH66, pc)
}

func (d *downloaderPeerNotify) unregisterPeer(p *peer) {
	d.downloader.UnregisterPeer(p.id)
}
