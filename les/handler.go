// Copyright 2015 The go-ethereum Authors
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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
	"errors"
	"fmt"
	"sync"
	//"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/access"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	//"github.com/ethereum/go-ethereum/eth/fetcher"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header

	ethVersion = 63 // equivalent eth version for the downloader
)

// errIncompatibleConfig is returned if the requested protocols and configs are
// not compatible (low protocol version restrictions and high requirements).
var errIncompatibleConfig = errors.New("incompatible configuration")

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type hashFetcherFn func(common.Hash) error

type ProtocolManager struct {
	mode        Mode
	blockchain  *core.BlockChain
	chainAccess *access.ChainAccess

	downloader *downloader.Downloader
	//fetcher    *fetcher.Fetcher
	peers *peerSet

	SubProtocols []p2p.Protocol

	eventMux *event.TypeMux

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh chan *peer
	quitSync  chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg   sync.WaitGroup
	quit bool
}

// NewProtocolManager returns a new ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the ethereum network.
func NewProtocolManager(mode Mode, networkId int, mux *event.TypeMux, pow pow.PoW, blockchain *core.BlockChain, ca *access.ChainAccess) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		mode:        mode,
		eventMux:    mux,
		blockchain:  blockchain,
		chainAccess: ca,
		peers:       newPeerSet(),
		newPeerCh:   make(chan *peer, 1),
		quitSync:    make(chan struct{}),
	}
	// Initiate a sub-protocol for every implemented version we can handle
	manager.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		// Skip protocol version if incompatible with the mode of operation
		if minimumProtocolVersion[mode] > version {
			continue
		}
		// Compatible, initialize the sub-protocol
		version := version // Closure for the run
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    "les",
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := manager.newPeer(int(version), networkId, p, rw)
				manager.newPeerCh <- peer
				return manager.handle(peer)
			},
		})
	}
	if len(manager.SubProtocols) == 0 {
		return nil, errIncompatibleConfig
	}
	// Construct the different synchronisation mechanisms
	var syncMode downloader.SyncMode
	switch mode {
	case ArchiveMode:
		syncMode = downloader.NoSync
	case FullMode:
		syncMode = downloader.NoSync
	case LightMode:
		syncMode = downloader.LightSync
	}
	glog.V(access.LogLevel).Infof("LES: create downloader")

	manager.downloader = downloader.New(syncMode, ca.Db(), manager.eventMux, blockchain.HasHeader, blockchain.HasBlock, blockchain.GetHeader,
		blockchain.GetBlockNoOdr, blockchain.CurrentHeader, blockchain.CurrentBlock, blockchain.CurrentFastBlock, blockchain.FastSyncCommitHead, blockchain.GetTd,
		blockchain.InsertHeaderChain, blockchain.InsertChain, blockchain.InsertReceiptChain, blockchain.Rollback, manager.removePeer)

	/*validator := func(block *types.Block, parent *types.Block) error {
		return core.ValidateHeader(pow, block.Header(), parent.Header(), true, false)
	}
	heighter := func() uint64 {
		return chainman.LastBlockNumberU64()
	}
	manager.fetcher = fetcher.New(chainman.GetBlockNoOdr, validator, nil, heighter, chainman.InsertChain, manager.removePeer)
	*/
	return manager, nil
}

func (pm *ProtocolManager) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	glog.V(logger.Debug).Infoln("Removing peer", id)

	// Unregister the peer from the downloader and Ethereum peer set
	glog.V(access.LogLevel).Infof("LES: unregister peer %v", id)
	pm.downloader.UnregisterPeer(id)
	if err := pm.peers.Unregister(id); err != nil {
		glog.V(logger.Error).Infoln("Removal failed:", err)
	}
	pm.chainAccess.UnregisterPeer(id)
	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (pm *ProtocolManager) Start() {
	// start sync handler
	go pm.syncer()
}

func (pm *ProtocolManager) Stop() {
	// Showing a log message. During download / process this could actually
	// take between 5 to 10 seconds and therefor feedback is required.
	glog.V(logger.Info).Infoln("Stopping light ethereum protocol handler...")

	pm.quit = true
	close(pm.quitSync) // quits syncer, fetcher

	// Wait for any process action
	pm.wg.Wait()

	glog.V(logger.Info).Infoln("Ethereum protocol handler stopped")
}

func (pm *ProtocolManager) newPeer(pv, nv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, nv, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of a les peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
	glog.V(logger.Debug).Infof("%v: peer connected [%s]", p, p.Name())

	// Execute the LES handshake
	td, head, genesis := pm.blockchain.Status()
	if err := p.Handshake(td, head, genesis); err != nil {
		glog.V(logger.Debug).Infof("%v: handshake failed: %v", p, err)
		return err
	}
	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}
	// Register the peer locally
	glog.V(logger.Detail).Infof("%v: adding peer", p)
	if err := pm.peers.Register(p); err != nil {
		glog.V(logger.Error).Infof("%v: addition failed: %v", p, err)
		return err
	}
	defer pm.removePeer(p.id)

	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	glog.V(access.LogLevel).Infof("LES: register peer %v", p.id)
	if err := pm.downloader.RegisterPeer(p.id, ethVersion, p.Head(),
		nil, nil, nil, p.RequestHeadersByHash, p.RequestHeadersByNumber,
		p.RequestBodies, p.RequestReceipts, p.RequestNodeData); err != nil {
		return err
	}

	pm.chainAccess.RegisterPeer(p.id, p.version, p.Head(), p.RequestBodies, p.RequestNodeData, p.RequestReceipts, p.RequestProofs)

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			glog.V(logger.Debug).Infof("%v: message handling failed: %v", p, err)
			return err
		}
	}
	return nil
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pm *ProtocolManager) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	// Handle the message depending on its contents
	switch msg.Code {
	case StatusMsg:
		glog.V(access.LogLevel).Infof("LES: received StatusMsg from peer %v", p.id)
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	// Block header query, collect the requested headers and reply
	case GetBlockHeadersMsg:
		glog.V(access.LogLevel).Infof("LES: received GetBlockHeadersMsg from peer %v", p.id)
		// Decode the complex header query
		var query getBlockHeadersData
		if err := msg.Decode(&query); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		// Gather headers until the fetch or network limits is reached
		var (
			bytes   common.StorageSize
			headers []*types.Header
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {
			// Retrieve the next header satisfying the query
			var origin *types.Header
			if query.Origin.Hash != (common.Hash{}) {
				origin = pm.blockchain.GetHeader(query.Origin.Hash)
			} else {
				origin = pm.blockchain.GetHeaderByNumber(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			headers = append(headers, origin)
			bytes += estHeaderRlpSize

			// Advance to the next header of the query
			switch {
			case query.Origin.Hash != (common.Hash{}) && query.Reverse:
				// Hash based traversal towards the genesis block
				for i := 0; i < int(query.Skip)+1; i++ {
					if header := pm.blockchain.GetHeader(query.Origin.Hash); header != nil {
						query.Origin.Hash = header.ParentHash
					} else {
						unknown = true
						break
					}
				}
			case query.Origin.Hash != (common.Hash{}) && !query.Reverse:
				// Hash based traversal towards the leaf block
				if header := pm.blockchain.GetHeaderByNumber(origin.Number.Uint64() + query.Skip + 1); header != nil {
					if pm.blockchain.GetBlockHashesFromHash(header.Hash(), query.Skip+1)[query.Skip] == query.Origin.Hash {
						query.Origin.Hash = header.Hash()
					} else {
						unknown = true
					}
				} else {
					unknown = true
				}
			case query.Reverse:
				// Number based traversal towards the genesis block
				if query.Origin.Number >= query.Skip+1 {
					query.Origin.Number -= (query.Skip + 1)
				} else {
					unknown = true
				}

			case !query.Reverse:
				// Number based traversal towards the leaf block
				query.Origin.Number += (query.Skip + 1)
			}
		}
		return p.SendBlockHeaders(headers)

	case BlockHeadersMsg:
		glog.V(access.LogLevel).Infof("LES: received BlockHeadersMsg from peer %v", p.id)
		// A batch of headers arrived to one of our previous requests
		var headers []*types.Header
		if err := msg.Decode(&headers); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Filter out any explicitly requested headers, deliver the rest to the downloader
		/*filter := len(headers) == 1
		if filter {
			headers = pm.fetcher.FilterHeaders(headers, time.Now())
		}
		if len(headers) > 0 || !filter {*/
		err := pm.downloader.DeliverHeaders(p.id, headers)
		if err != nil {
			glog.V(logger.Debug).Infoln(err)
		}
		//}

	case GetBlockBodiesMsg:
		glog.V(access.LogLevel).Infof("LES: received GetBlockBodiesMsg from peer %v", p.id)
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather blocks until the fetch or network limits is reached
		var (
			hash   common.Hash
			bytes  int
			bodies []rlp.RawValue
		)
		for bytes < softResponseLimit && len(bodies) < downloader.MaxBlockFetch {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block body, stopping if enough was found
			if data := pm.blockchain.GetBodyRLP(hash, access.NoOdr); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)
			}
		}
		return p.SendBlockBodiesRLP(bodies)

	case BlockBodiesMsg:
		glog.V(access.LogLevel).Infof("LES: received BlockBodiesMsg from peer %v", p.id)
		// A batch of block bodies arrived to one of our previous requests
		var data []*types.Body
		if err := msg.Decode(&data); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		pm.chainAccess.Deliver(p.id, &access.Msg{
			MsgType: access.MsgBlockBodies,
			Obj:     data,
		})

	case GetNodeDataMsg:
		glog.V(access.LogLevel).Infof("LES: received GetNodeDataMsg from peer %v", p.id)
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash  common.Hash
			bytes int
			data  [][]byte
		)
		for bytes < softResponseLimit && len(data) < downloader.MaxStateFetch {
			// Retrieve the hash of the next state entry
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested state entry, stopping if enough was found
			if entry, err := pm.chainAccess.Db().Get(hash.Bytes()); err == nil {
				data = append(data, entry)
				bytes += len(entry)
			}
		}
		return p.SendNodeData(data)

	case NodeDataMsg:
		glog.V(access.LogLevel).Infof("LES: received NodeDataMsg from peer %v", p.id)
		// A batch of node state data arrived to one of our previous requests
		var data [][]byte
		if err := msg.Decode(&data); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		pm.chainAccess.Deliver(p.id, &access.Msg{
			MsgType: access.MsgNodeData,
			Obj:     data,
		})

	case GetReceiptsMsg:
		glog.V(access.LogLevel).Infof("LES: received GetReceiptsMsg from peer %v", p.id)
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash     common.Hash
			bytes    int
			receipts []rlp.RawValue
		)
		for bytes < softResponseLimit && len(receipts) < downloader.MaxReceiptFetch {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block's receipts, skipping if unknown to us
			results := core.GetBlockReceipts(pm.chainAccess, hash, access.NoOdr)
			if results == nil {
				if header := pm.blockchain.GetHeader(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
					continue
				}
			}
			// If known, encode and queue for response packet
			if encoded, err := rlp.EncodeToBytes(results); err != nil {
				glog.V(logger.Error).Infof("failed to encode receipt: %v", err)
			} else {
				receipts = append(receipts, encoded)
				bytes += len(encoded)
			}
		}
		return p.SendReceiptsRLP(receipts)

	case ReceiptsMsg:
		glog.V(access.LogLevel).Infof("LES: received ReceiptsMsg from peer %v", p.id)
		// A batch of receipts arrived to one of our previous requests
		var receipts []types.Receipts
		if err := msg.Decode(&receipts); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		pm.chainAccess.Deliver(p.id, &access.Msg{
			MsgType: access.MsgReceipts,
			Obj:     receipts,
		})

	case GetProofsMsg:
		glog.V(access.LogLevel).Infof("LES: received GetProofsMsg from peer %v", p.id)
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			req    access.ProofReq
			bytes  int
			proofs proofsData
		)
		for bytes < softResponseLimit && len(proofs) < downloader.MaxProofsFetch {
			// Retrieve the hash of the next state entry
			if err := msgStream.Decode(&req); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested state entry, stopping if enough was found
			if trie, _ := trie.NewSecure(req.Root, pm.chainAccess.Db(), nil); trie != nil {
				proof := trie.Prove(req.Key)
				proofs = append(proofs, proof)
				bytes += len(proof)
			}
		}
		return p.SendProofs(proofs)

	case ProofsMsg:
		glog.V(access.LogLevel).Infof("LES: received ProofsMsg from peer %v", p.id)
		// A batch of merkle proofs arrived to one of our previous requests
		var data []trie.MerkleProof
		if err := msg.Decode(&data); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		pm.chainAccess.Deliver(p.id, &access.Msg{
			MsgType: access.MsgProofs,
			Obj:     data,
		})

	case NewBlockHashesMsg:
		glog.V(access.LogLevel).Infof("LES: received NewBlockHashesMsg from peer %v", p.id)
		// Retrieve and deseralize the remote new block hashes notification
		type announce struct {
			Hash   common.Hash
			Number uint64
		}
		var announces = []announce{}

		// We're running the old protocol, make block number unknown (0)
		var hashes []common.Hash
		if err := msg.Decode(&hashes); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		for _, hash := range hashes {
			announces = append(announces, announce{hash, 0})
		}
		// Mark the hashes as present at the remote node
		for _, block := range announces {
			p.MarkBlock(block.Hash)
			p.SetHead(block.Hash)
		}
		// Schedule all the unknown hashes for retrieval
		unknown := make([]announce, 0, len(announces))
		for _, block := range announces {
			if !pm.blockchain.HasBlock(block.Hash) {
				unknown = append(unknown, block)
			}
		}
		/*for _, block := range unknown {
			pm.fetcher.Notify(p.id, block.Hash, block.Number, time.Now(), nil, p.RequestOneHeader, p.RequestBodies)
		}*/

	default:
		glog.V(access.LogLevel).Infof("LES: received unknown message with code %d from peer %v", msg.Code, p.id)
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}
