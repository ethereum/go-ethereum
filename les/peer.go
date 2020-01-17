// Copyright 2016 The go-ethereum Authors
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
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const (
	maxRequestErrors  = 20 // number of invalid requests tolerated (makes the protocol less brittle but still avoids spam)
	maxResponseErrors = 50 // number of invalid responses tolerated (makes the protocol less brittle but still avoids spam)
)

// capacity limitation for parameter updates
const (
	allowedUpdateBytes = 100000                // initial/maximum allowed update size
	allowedUpdateRate  = time.Millisecond * 10 // time constant for recharging one byte of allowance
)

const (
	freezeTimeBase    = time.Millisecond * 700 // fixed component of client freeze time
	freezeTimeRandom  = time.Millisecond * 600 // random component of client freeze time
	freezeCheckPeriod = time.Millisecond * 100 // buffer value recheck period after initial freeze time has elapsed
)

// if the total encoded size of a sent transaction batch is over txSizeCostLimit
// per transaction then the request cost is calculated as proportional to the
// encoded size instead of the transaction count
const txSizeCostLimit = 0x4000

const (
	announceTypeNone = iota
	announceTypeSimple
	announceTypeSigned
)

type peer struct {
	*p2p.Peer
	rw p2p.MsgReadWriter

	version int    // Protocol version negotiated
	network uint64 // Network ID being on

	announceType uint64

	// Checkpoint relative fields
	checkpoint       params.TrustedCheckpoint
	checkpointNumber uint64

	id string

	headInfo *announceData
	lock     sync.RWMutex

	sendQueue *execQueue

	errCh chan error

	// responseLock ensures that responses are queued in the same order as
	// RequestProcessed is called
	responseLock  sync.Mutex
	responseCount uint64
	invalidCount  uint32

	poolEntry      *poolEntry
	hasBlock       func(common.Hash, uint64, bool) bool
	responseErrors int
	updateCounter  uint64
	updateTime     mclock.AbsTime
	frozen         uint32 // 1 if client is in frozen state

	fcClient *flowcontrol.ClientNode // nil if the peer is server only
	fcServer *flowcontrol.ServerNode // nil if the peer is client only
	fcParams flowcontrol.ServerParams
	fcCosts  requestCostTable

	trusted, server         bool
	onlyAnnounce            bool
	chainSince, chainRecent uint64
	stateSince, stateRecent uint64
}

func newPeer(version int, network uint64, trusted bool, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		Peer:    p,
		rw:      rw,
		version: version,
		network: network,
		id:      peerIdToString(p.ID()),
		trusted: trusted,
		errCh:   make(chan error, 1),
	}
}

// peerIdToString converts enode.ID to a string form
func peerIdToString(id enode.ID) string {
	return fmt.Sprintf("%x", id.Bytes())
}

// freeClientId returns a string identifier for the peer. Multiple peers with the
// same identifier can not be connected in free mode simultaneously.
func (p *peer) freeClientId() string {
	if addr, ok := p.RemoteAddr().(*net.TCPAddr); ok {
		if addr.IP.IsLoopback() {
			// using peer id instead of loopback ip address allows multiple free
			// connections from local machine to own server
			return p.id
		} else {
			return addr.IP.String()
		}
	}
	return p.id
}

// rejectUpdate returns true if a parameter update has to be rejected because
// the size and/or rate of updates exceed the capacity limitation
func (p *peer) rejectUpdate(size uint64) bool {
	now := mclock.Now()
	if p.updateCounter == 0 {
		p.updateTime = now
	} else {
		dt := now - p.updateTime
		r := uint64(dt / mclock.AbsTime(allowedUpdateRate))
		if p.updateCounter > r {
			p.updateCounter -= r
			p.updateTime += mclock.AbsTime(allowedUpdateRate * time.Duration(r))
		} else {
			p.updateCounter = 0
			p.updateTime = now
		}
	}
	p.updateCounter += size
	return p.updateCounter > allowedUpdateBytes
}

// freezeClient temporarily puts the client in a frozen state which means all
// unprocessed and subsequent requests are dropped. Unfreezing happens automatically
// after a short time if the client's buffer value is at least in the slightly positive
// region. The client is also notified about being frozen/unfrozen with a Stop/Resume
// message.
func (p *peer) freezeClient() {
	if p.version < lpv3 {
		// if Stop/Resume is not supported then just drop the peer after setting
		// its frozen status permanently
		atomic.StoreUint32(&p.frozen, 1)
		p.Peer.Disconnect(p2p.DiscUselessPeer)
		return
	}
	if atomic.SwapUint32(&p.frozen, 1) == 0 {
		go func() {
			p.SendStop()
			time.Sleep(freezeTimeBase + time.Duration(rand.Int63n(int64(freezeTimeRandom))))
			for {
				bufValue, bufLimit := p.fcClient.BufferStatus()
				if bufLimit == 0 {
					return
				}
				if bufValue <= bufLimit/8 {
					time.Sleep(freezeCheckPeriod)
				} else {
					atomic.StoreUint32(&p.frozen, 0)
					p.SendResume(bufValue)
					break
				}
			}
		}()
	}
}

// freezeServer processes Stop/Resume messages from the given server
func (p *peer) freezeServer(frozen bool) {
	var f uint32
	if frozen {
		f = 1
	}
	if atomic.SwapUint32(&p.frozen, f) != f && frozen {
		p.sendQueue.clear()
	}
}

// isFrozen returns true if the client is frozen or the server has put our
// client in frozen state
func (p *peer) isFrozen() bool {
	return atomic.LoadUint32(&p.frozen) != 0
}

func (p *peer) canQueue() bool {
	return p.sendQueue.canQueue() && !p.isFrozen()
}

func (p *peer) queueSend(f func()) {
	p.sendQueue.queue(f)
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *eth.PeerInfo {
	return &eth.PeerInfo{
		Version:    p.version,
		Difficulty: p.Td(),
		Head:       fmt.Sprintf("%x", p.Head()),
	}
}

// Head retrieves a copy of the current head (most recent) hash of the peer.
func (p *peer) Head() (hash common.Hash) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.headInfo.Hash[:])
	return hash
}

func (p *peer) HeadAndTd() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.headInfo.Hash[:])
	return hash, p.headInfo.Td
}

func (p *peer) headBlockInfo() blockInfo {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return blockInfo{Hash: p.headInfo.Hash, Number: p.headInfo.Number, Td: p.headInfo.Td}
}

// Td retrieves the current total difficulty of a peer.
func (p *peer) Td() *big.Int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return new(big.Int).Set(p.headInfo.Td)
}

// waitBefore implements distPeer interface
func (p *peer) waitBefore(maxCost uint64) (time.Duration, float64) {
	return p.fcServer.CanSend(maxCost)
}

// updateCapacity updates the request serving capacity assigned to a given client
// and also sends an announcement about the updated flow control parameters
func (p *peer) updateCapacity(cap uint64) {
	p.responseLock.Lock()
	defer p.responseLock.Unlock()

	p.fcParams = flowcontrol.ServerParams{MinRecharge: cap, BufLimit: cap * bufLimitRatio}
	p.fcClient.UpdateParams(p.fcParams)
	var kvList keyValueList
	kvList = kvList.add("flowControl/MRR", cap)
	kvList = kvList.add("flowControl/BL", cap*bufLimitRatio)
	p.queueSend(func() { p.SendAnnounce(announceData{Update: kvList}) })
}

func (p *peer) responseID() uint64 {
	p.responseCount += 1
	return p.responseCount
}

func sendRequest(w p2p.MsgWriter, msgcode, reqID, cost uint64, data interface{}) error {
	type req struct {
		ReqID uint64
		Data  interface{}
	}
	return p2p.Send(w, msgcode, req{reqID, data})
}

// reply struct represents a reply with the actual data already RLP encoded and
// only the bv (buffer value) missing. This allows the serving mechanism to
// calculate the bv value which depends on the data size before sending the reply.
type reply struct {
	w              p2p.MsgWriter
	msgcode, reqID uint64
	data           rlp.RawValue
}

// send sends the reply with the calculated buffer value
func (r *reply) send(bv uint64) error {
	type resp struct {
		ReqID, BV uint64
		Data      rlp.RawValue
	}
	return p2p.Send(r.w, r.msgcode, resp{r.reqID, bv, r.data})
}

// size returns the RLP encoded size of the message data
func (r *reply) size() uint32 {
	return uint32(len(r.data))
}

func (p *peer) GetRequestCost(msgcode uint64, amount int) uint64 {
	p.lock.RLock()
	defer p.lock.RUnlock()

	costs := p.fcCosts[msgcode]
	if costs == nil {
		return 0
	}
	cost := costs.baseCost + costs.reqCost*uint64(amount)
	if cost > p.fcParams.BufLimit {
		cost = p.fcParams.BufLimit
	}
	return cost
}

func (p *peer) GetTxRelayCost(amount, size int) uint64 {
	p.lock.RLock()
	defer p.lock.RUnlock()

	costs := p.fcCosts[SendTxV2Msg]
	if costs == nil {
		return 0
	}
	cost := costs.baseCost + costs.reqCost*uint64(amount)
	sizeCost := costs.baseCost + costs.reqCost*uint64(size)/txSizeCostLimit
	if sizeCost > cost {
		cost = sizeCost
	}

	if cost > p.fcParams.BufLimit {
		cost = p.fcParams.BufLimit
	}
	return cost
}

// HasBlock checks if the peer has a given block
func (p *peer) HasBlock(hash common.Hash, number uint64, hasState bool) bool {
	var head, since, recent uint64
	p.lock.RLock()
	if p.headInfo != nil {
		head = p.headInfo.Number
	}
	if hasState {
		since = p.stateSince
		recent = p.stateRecent
	} else {
		since = p.chainSince
		recent = p.chainRecent
	}
	hasBlock := p.hasBlock
	p.lock.RUnlock()

	return head >= number && number >= since && (recent == 0 || number+recent+4 > head) && hasBlock != nil && hasBlock(hash, number, hasState)
}

// SendAnnounce announces the availability of a number of blocks through
// a hash notification.
func (p *peer) SendAnnounce(request announceData) error {
	return p2p.Send(p.rw, AnnounceMsg, request)
}

// SendStop notifies the client about being in frozen state
func (p *peer) SendStop() error {
	return p2p.Send(p.rw, StopMsg, struct{}{})
}

// SendResume notifies the client about getting out of frozen state
func (p *peer) SendResume(bv uint64) error {
	return p2p.Send(p.rw, ResumeMsg, bv)
}

// ReplyBlockHeaders creates a reply with a batch of block headers
func (p *peer) ReplyBlockHeaders(reqID uint64, headers []*types.Header) *reply {
	data, _ := rlp.EncodeToBytes(headers)
	return &reply{p.rw, BlockHeadersMsg, reqID, data}
}

// ReplyBlockBodiesRLP creates a reply with a batch of block contents from
// an already RLP encoded format.
func (p *peer) ReplyBlockBodiesRLP(reqID uint64, bodies []rlp.RawValue) *reply {
	data, _ := rlp.EncodeToBytes(bodies)
	return &reply{p.rw, BlockBodiesMsg, reqID, data}
}

// ReplyCode creates a reply with a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (p *peer) ReplyCode(reqID uint64, codes [][]byte) *reply {
	data, _ := rlp.EncodeToBytes(codes)
	return &reply{p.rw, CodeMsg, reqID, data}
}

// ReplyReceiptsRLP creates a reply with a batch of transaction receipts, corresponding to the
// ones requested from an already RLP encoded format.
func (p *peer) ReplyReceiptsRLP(reqID uint64, receipts []rlp.RawValue) *reply {
	data, _ := rlp.EncodeToBytes(receipts)
	return &reply{p.rw, ReceiptsMsg, reqID, data}
}

// ReplyProofsV2 creates a reply with a batch of merkle proofs, corresponding to the ones requested.
func (p *peer) ReplyProofsV2(reqID uint64, proofs light.NodeList) *reply {
	data, _ := rlp.EncodeToBytes(proofs)
	return &reply{p.rw, ProofsV2Msg, reqID, data}
}

// ReplyHelperTrieProofs creates a reply with a batch of HelperTrie proofs, corresponding to the ones requested.
func (p *peer) ReplyHelperTrieProofs(reqID uint64, resp HelperTrieResps) *reply {
	data, _ := rlp.EncodeToBytes(resp)
	return &reply{p.rw, HelperTrieProofsMsg, reqID, data}
}

// ReplyTxStatus creates a reply with a batch of transaction status records, corresponding to the ones requested.
func (p *peer) ReplyTxStatus(reqID uint64, stats []light.TxStatus) *reply {
	data, _ := rlp.EncodeToBytes(stats)
	return &reply{p.rw, TxStatusMsg, reqID, data}
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestHeadersByHash(reqID, cost uint64, origin common.Hash, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	return sendRequest(p.rw, GetBlockHeadersMsg, reqID, cost, &getBlockHeadersData{Origin: hashOrNumber{Hash: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *peer) RequestHeadersByNumber(reqID, cost, origin uint64, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	return sendRequest(p.rw, GetBlockHeadersMsg, reqID, cost, &getBlockHeadersData{Origin: hashOrNumber{Number: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *peer) RequestBodies(reqID, cost uint64, hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of block bodies", "count", len(hashes))
	return sendRequest(p.rw, GetBlockBodiesMsg, reqID, cost, hashes)
}

// RequestCode fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (p *peer) RequestCode(reqID, cost uint64, reqs []CodeReq) error {
	p.Log().Debug("Fetching batch of codes", "count", len(reqs))
	return sendRequest(p.rw, GetCodeMsg, reqID, cost, reqs)
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *peer) RequestReceipts(reqID, cost uint64, hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of receipts", "count", len(hashes))
	return sendRequest(p.rw, GetReceiptsMsg, reqID, cost, hashes)
}

// RequestProofs fetches a batch of merkle proofs from a remote node.
func (p *peer) RequestProofs(reqID, cost uint64, reqs []ProofReq) error {
	p.Log().Debug("Fetching batch of proofs", "count", len(reqs))
	return sendRequest(p.rw, GetProofsV2Msg, reqID, cost, reqs)
}

// RequestHelperTrieProofs fetches a batch of HelperTrie merkle proofs from a remote node.
func (p *peer) RequestHelperTrieProofs(reqID, cost uint64, reqs []HelperTrieReq) error {
	p.Log().Debug("Fetching batch of HelperTrie proofs", "count", len(reqs))
	return sendRequest(p.rw, GetHelperTrieProofsMsg, reqID, cost, reqs)
}

// RequestTxStatus fetches a batch of transaction status records from a remote node.
func (p *peer) RequestTxStatus(reqID, cost uint64, txHashes []common.Hash) error {
	p.Log().Debug("Requesting transaction status", "count", len(txHashes))
	return sendRequest(p.rw, GetTxStatusMsg, reqID, cost, txHashes)
}

// SendTxStatus creates a reply with a batch of transactions to be added to the remote transaction pool.
func (p *peer) SendTxs(reqID, cost uint64, txs rlp.RawValue) error {
	p.Log().Debug("Sending batch of transactions", "size", len(txs))
	return sendRequest(p.rw, SendTxV2Msg, reqID, cost, txs)
}

type keyValueEntry struct {
	Key   string
	Value rlp.RawValue
}
type keyValueList []keyValueEntry
type keyValueMap map[string]rlp.RawValue

func (l keyValueList) add(key string, val interface{}) keyValueList {
	var entry keyValueEntry
	entry.Key = key
	if val == nil {
		val = uint64(0)
	}
	enc, err := rlp.EncodeToBytes(val)
	if err == nil {
		entry.Value = enc
	}
	return append(l, entry)
}

func (l keyValueList) decode() (keyValueMap, uint64) {
	m := make(keyValueMap)
	var size uint64
	for _, entry := range l {
		m[entry.Key] = entry.Value
		size += uint64(len(entry.Key)) + uint64(len(entry.Value)) + 8
	}
	return m, size
}

func (m keyValueMap) get(key string, val interface{}) error {
	enc, ok := m[key]
	if !ok {
		return errResp(ErrMissingKey, "%s", key)
	}
	if val == nil {
		return nil
	}
	return rlp.DecodeBytes(enc, val)
}

func (p *peer) sendReceiveHandshake(sendList keyValueList) (keyValueList, error) {
	// Send out own handshake in a new thread
	errc := make(chan error, 1)
	go func() {
		errc <- p2p.Send(p.rw, StatusMsg, sendList)
	}()
	// In the mean time retrieve the remote status message
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return nil, err
	}
	if msg.Code != StatusMsg {
		return nil, errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return nil, errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// Decode the handshake
	var recvList keyValueList
	if err := msg.Decode(&recvList); err != nil {
		return nil, errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if err := <-errc; err != nil {
		return nil, err
	}
	return recvList, nil
}

// Handshake executes the les protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *peer) Handshake(td *big.Int, head common.Hash, headNum uint64, genesis common.Hash, server *LesServer) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	var send keyValueList

	// Add some basic handshake fields
	send = send.add("protocolVersion", uint64(p.version))
	send = send.add("networkId", p.network)
	send = send.add("headTd", td)
	send = send.add("headHash", head)
	send = send.add("headNum", headNum)
	send = send.add("genesisHash", genesis)
	if server != nil {
		// Add some information which services server can offer.
		if !server.config.UltraLightOnlyAnnounce {
			send = send.add("serveHeaders", nil)
			send = send.add("serveChainSince", uint64(0))
			send = send.add("serveStateSince", uint64(0))

			// If local ethereum node is running in archive mode, advertise ourselves we have
			// all version state data. Otherwise only recent state is available.
			stateRecent := uint64(core.TriesInMemory - 4)
			if server.archiveMode {
				stateRecent = 0
			}
			send = send.add("serveRecentState", stateRecent)
			send = send.add("txRelay", nil)
		}
		send = send.add("flowControl/BL", server.defParams.BufLimit)
		send = send.add("flowControl/MRR", server.defParams.MinRecharge)

		var costList RequestCostList
		if server.costTracker.testCostList != nil {
			costList = server.costTracker.testCostList
		} else {
			costList = server.costTracker.makeCostList(server.costTracker.globalFactor())
		}
		send = send.add("flowControl/MRC", costList)
		p.fcCosts = costList.decode(ProtocolLengths[uint(p.version)])
		p.fcParams = server.defParams

		// Add advertised checkpoint and register block height which
		// client can verify the checkpoint validity.
		if server.oracle != nil && server.oracle.IsRunning() {
			cp, height := server.oracle.StableCheckpoint()
			if cp != nil {
				send = send.add("checkpoint/value", cp)
				send = send.add("checkpoint/registerHeight", height)
			}
		}
	} else {
		// Add some client-specific handshake fields
		p.announceType = announceTypeSimple
		if p.trusted {
			p.announceType = announceTypeSigned
		}
		send = send.add("announceType", p.announceType)
	}

	recvList, err := p.sendReceiveHandshake(send)
	if err != nil {
		return err
	}
	recv, size := recvList.decode()
	if p.rejectUpdate(size) {
		return errResp(ErrRequestRejected, "")
	}

	var rGenesis, rHash common.Hash
	var rVersion, rNetwork, rNum uint64
	var rTd *big.Int

	if err := recv.get("protocolVersion", &rVersion); err != nil {
		return err
	}
	if err := recv.get("networkId", &rNetwork); err != nil {
		return err
	}
	if err := recv.get("headTd", &rTd); err != nil {
		return err
	}
	if err := recv.get("headHash", &rHash); err != nil {
		return err
	}
	if err := recv.get("headNum", &rNum); err != nil {
		return err
	}
	if err := recv.get("genesisHash", &rGenesis); err != nil {
		return err
	}

	if rGenesis != genesis {
		return errResp(ErrGenesisBlockMismatch, "%x (!= %x)", rGenesis[:8], genesis[:8])
	}
	if rNetwork != p.network {
		return errResp(ErrNetworkIdMismatch, "%d (!= %d)", rNetwork, p.network)
	}
	if int(rVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", rVersion, p.version)
	}

	if server != nil {
		p.server = recv.get("flowControl/MRR", nil) == nil
		if p.server {
			p.announceType = announceTypeNone // connected to another server, send no messages
		} else {
			if recv.get("announceType", &p.announceType) != nil {
				// set default announceType on server side
				p.announceType = announceTypeSimple
			}
			p.fcClient = flowcontrol.NewClientNode(server.fcManager, server.defParams)
		}
	} else {
		if recv.get("serveChainSince", &p.chainSince) != nil {
			p.onlyAnnounce = true
		}
		if recv.get("serveRecentChain", &p.chainRecent) != nil {
			p.chainRecent = 0
		}
		if recv.get("serveStateSince", &p.stateSince) != nil {
			p.onlyAnnounce = true
		}
		if recv.get("serveRecentState", &p.stateRecent) != nil {
			p.stateRecent = 0
		}
		if recv.get("txRelay", nil) != nil {
			p.onlyAnnounce = true
		}

		if p.onlyAnnounce && !p.trusted {
			return errResp(ErrUselessPeer, "peer cannot serve requests")
		}

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

		recv.get("checkpoint/value", &p.checkpoint)
		recv.get("checkpoint/registerHeight", &p.checkpointNumber)

		if !p.onlyAnnounce {
			for msgCode := range reqAvgTimeCost {
				if p.fcCosts[msgCode] == nil {
					return errResp(ErrUselessPeer, "peer does not support message %d", msgCode)
				}
			}
		}
		p.server = true
	}
	p.headInfo = &announceData{Td: rTd, Hash: rHash, Number: rNum}
	return nil
}

// updateFlowControl updates the flow control parameters belonging to the server
// node if the announced key/value set contains relevant fields
func (p *peer) updateFlowControl(update keyValueMap) {
	if p.fcServer == nil {
		return
	}
	// If any of the flow control params is nil, refuse to update.
	var params flowcontrol.ServerParams
	if update.get("flowControl/BL", &params.BufLimit) == nil && update.get("flowControl/MRR", &params.MinRecharge) == nil {
		// todo can light client set a minimal acceptable flow control params?
		p.fcParams = params
		p.fcServer.UpdateParams(params)
	}
	var MRC RequestCostList
	if update.get("flowControl/MRC", &MRC) == nil {
		costUpdate := MRC.decode(ProtocolLengths[uint(p.version)])
		for code, cost := range costUpdate {
			p.fcCosts[code] = cost
		}
	}
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%s]", p.id,
		fmt.Sprintf("les/%d", p.version),
	)
}

// peerSetNotify is a callback interface to notify services about added or
// removed peers
type peerSetNotify interface {
	registerPeer(*peer)
	unregisterPeer(*peer)
}

// peerSet represents the collection of active peers currently participating in
// the Light Ethereum sub-protocol.
type peerSet struct {
	peers      map[string]*peer
	lock       sync.RWMutex
	notifyList []peerSetNotify
	closed     bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// notify adds a service to be notified about added or removed peers
func (ps *peerSet) notify(n peerSetNotify) {
	ps.lock.Lock()
	ps.notifyList = append(ps.notifyList, n)
	peers := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		peers = append(peers, p)
	}
	ps.lock.Unlock()

	for _, p := range peers {
		n.registerPeer(p)
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	if ps.closed {
		ps.lock.Unlock()
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		ps.lock.Unlock()
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	p.sendQueue = newExecQueue(100)
	peers := make([]peerSetNotify, len(ps.notifyList))
	copy(peers, ps.notifyList)
	ps.lock.Unlock()

	for _, n := range peers {
		n.registerPeer(p)
	}
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity. It also initiates disconnection at the networking layer.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	if p, ok := ps.peers[id]; !ok {
		ps.lock.Unlock()
		return errNotRegistered
	} else {
		delete(ps.peers, id)
		peers := make([]peerSetNotify, len(ps.notifyList))
		copy(peers, ps.notifyList)
		ps.lock.Unlock()

		for _, n := range peers {
			n.unregisterPeer(p)
		}

		p.sendQueue.quit()
		p.Peer.Disconnect(p2p.DiscUselessPeer)

		return nil
	}
}

// AllPeerIDs returns a list of all registered peer IDs
func (ps *peerSet) AllPeerIDs() []string {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	res := make([]string, len(ps.peers))
	idx := 0
	for id := range ps.peers {
		res[idx] = id
		idx++
	}
	return res
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var (
		bestPeer *peer
		bestTd   *big.Int
	)
	for _, p := range ps.peers {
		if td := p.Td(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

// AllPeers returns all peers in a list
func (ps *peerSet) AllPeers() []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, len(ps.peers))
	i := 0
	for _, peer := range ps.peers {
		list[i] = peer
		i++
	}
	return list
}

// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.Disconnect(p2p.DiscQuitting)
	}
	ps.closed = true
}
