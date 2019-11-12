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
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
)

// clientHandler is responsible for receiving and processing all incoming server
// responses.
type clientHandler struct {
	ulc        *ulc
	checkpoint *params.TrustedCheckpoint
	fetcher    *lightFetcher
	downloader *downloader.Downloader
	backend    *LightEthereum

	closeCh  chan struct{}
	wg       sync.WaitGroup // WaitGroup used to track all connected peers.
	syncDone func()         // Test hooks when syncing is done.
}

func newClientHandler(ulcServers []string, ulcFraction int, checkpoint *params.TrustedCheckpoint, backend *LightEthereum) *clientHandler {
	handler := &clientHandler{
		checkpoint: checkpoint,
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
	var height uint64
	if checkpoint != nil {
		height = (checkpoint.SectionIndex+1)*params.CHTFrequency - 1
	}
	handler.fetcher = newLightFetcher(handler)
	handler.downloader = downloader.New(height, backend.chainDb, nil, backend.eventMux, nil, backend.blockchain, handler.removePeer)
	handler.backend.peers.notify((*downloaderPeerNotify)(handler))
	return handler
}

func (h *clientHandler) stop() {
	close(h.closeCh)
	h.downloader.Terminate()
	h.fetcher.close()
	h.wg.Wait()
}

// runPeer is the p2p protocol run function for the given version.
func (h *clientHandler) runPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter) error {
	trusted := false
	if h.ulc != nil {
		trusted = h.ulc.trusted(p.ID())
	}
	peer := newPeer(int(version), h.backend.config.NetworkId, trusted, p, newMeteredMsgWriter(rw, int(version)))
	peer.poolEntry = h.backend.serverPool.connect(peer, peer.Node())
	if peer.poolEntry == nil {
		return p2p.DiscRequested
	}
	h.wg.Add(1)
	defer h.wg.Done()
	err := h.handle(peer)
	h.backend.serverPool.disconnect(peer.poolEntry)
	return err
}

func (h *clientHandler) handle(p *peer) error {
	if h.backend.peers.Len() >= h.backend.config.LightPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("Light Ethereum peer connected", "name", p.Name())

	// Execute the LES handshake
	var (
		head   = h.backend.blockchain.CurrentHeader()
		hash   = head.Hash()
		number = head.Number.Uint64()
		td     = h.backend.blockchain.GetTd(hash, number)
	)
	if err := p.Handshake(td, hash, number, h.backend.blockchain.Genesis().Hash(), nil); err != nil {
		p.Log().Debug("Light Ethereum handshake failed", "err", err)
		return err
	}
	// Register the peer locally
	if err := h.backend.peers.Register(p); err != nil {
		p.Log().Error("Light Ethereum peer registration failed", "err", err)
		return err
	}
	serverConnectionGauge.Update(int64(h.backend.peers.Len()))

	connectedAt := mclock.Now()
	defer func() {
		h.backend.peers.Unregister(p.id)
		connectionTimer.Update(time.Duration(mclock.Now() - connectedAt))
		serverConnectionGauge.Update(int64(h.backend.peers.Len()))
	}()

	h.fetcher.announce(p, p.headInfo)

	// pool entry can be nil during the unit test.
	if p.poolEntry != nil {
		h.backend.serverPool.registered(p.poolEntry)
	}
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
func (h *clientHandler) handleMsg(p *peer) error {
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
	switch msg.Code {
	case AnnounceMsg:
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
			h.fetcher.announce(p, &req)
		}
	case BlockHeadersMsg:
		p.Log().Trace("Received block header response message")
		var resp struct {
			ReqID, BV uint64
			Headers   []*types.Header
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		if h.fetcher.requestedID(resp.ReqID) {
			h.fetcher.deliverHeaders(p, resp.ReqID, resp.Headers)
		} else {
			if err := h.downloader.DeliverHeaders(p.id, resp.Headers); err != nil {
				log.Debug("Failed to deliver headers", "err", err)
			}
		}
	case BlockBodiesMsg:
		p.Log().Trace("Received block bodies response")
		var resp struct {
			ReqID, BV uint64
			Data      []*types.Body
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgBlockBodies,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}
	case CodeMsg:
		p.Log().Trace("Received code response")
		var resp struct {
			ReqID, BV uint64
			Data      [][]byte
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgCode,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}
	case ReceiptsMsg:
		p.Log().Trace("Received receipts response")
		var resp struct {
			ReqID, BV uint64
			Receipts  []types.Receipts
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgReceipts,
			ReqID:   resp.ReqID,
			Obj:     resp.Receipts,
		}
	case ProofsV2Msg:
		p.Log().Trace("Received les/2 proofs response")
		var resp struct {
			ReqID, BV uint64
			Data      light.NodeList
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgProofsV2,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}
	case HelperTrieProofsMsg:
		p.Log().Trace("Received helper trie proof response")
		var resp struct {
			ReqID, BV uint64
			Data      HelperTrieResps
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgHelperTrieProofs,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}
	case TxStatusMsg:
		p.Log().Trace("Received tx status response")
		var resp struct {
			ReqID, BV uint64
			Status    []light.TxStatus
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgTxStatus,
			ReqID:   resp.ReqID,
			Obj:     resp.Status,
		}
	case StopMsg:
		p.freezeServer(true)
		h.backend.retriever.frozen(p)
		p.Log().Debug("Service stopped")
	case ResumeMsg:
		var bv uint64
		if err := msg.Decode(&bv); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ResumeFreeze(bv)
		p.freezeServer(false)
		p.Log().Debug("Service resumed")
	default:
		p.Log().Trace("Received invalid message", "code", msg.Code)
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	// Deliver the received response to retriever.
	if deliverMsg != nil {
		if err := h.backend.retriever.deliver(p, deliverMsg); err != nil {
			p.responseErrors++
			if p.responseErrors > maxResponseErrors {
				return err
			}
		}
	}
	return nil
}

func (h *clientHandler) removePeer(id string) {
	h.backend.peers.Unregister(id)
}

type peerConnection struct {
	handler *clientHandler
	peer    *peer
}

func (pc *peerConnection) Head() (common.Hash, *big.Int) {
	return pc.peer.HeadAndTd()
}

func (pc *peerConnection) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			peer := dp.(*peer)
			return peer.GetRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*peer) == pc.peer
		},
		request: func(dp distPeer) func() {
			reqID := genReqID()
			peer := dp.(*peer)
			cost := peer.GetRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.RequestHeadersByHash(reqID, cost, origin, amount, skip, reverse) }
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
			peer := dp.(*peer)
			return peer.GetRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*peer) == pc.peer
		},
		request: func(dp distPeer) func() {
			reqID := genReqID()
			peer := dp.(*peer)
			cost := peer.GetRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.RequestHeadersByNumber(reqID, cost, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pc.handler.backend.reqDist.queue(rq)
	if !ok {
		return light.ErrNoPeers
	}
	return nil
}

// downloaderPeerNotify implements peerSetNotify
type downloaderPeerNotify clientHandler

func (d *downloaderPeerNotify) registerPeer(p *peer) {
	h := (*clientHandler)(d)
	pc := &peerConnection{
		handler: h,
		peer:    p,
	}
	h.downloader.RegisterLightPeer(p.id, ethVersion, pc)
}

func (d *downloaderPeerNotify) unregisterPeer(p *peer) {
	h := (*clientHandler)(d)
	h.downloader.UnregisterPeer(p.id)
}
