// This is a geth extension that allows for subscribing to block headers along
// with the peer infomration for the first peer to provide the block. This
// implementation is a bit hacky - It relies on global variables being set in
// the middle of the p2p protocol handler. Generally, I would rather pass such
// information through dependency injection, but because we're not in charge of
// the Geth codebase and dependency injection would have to pass through quite
// a few layers, it becomes very likely that future Geth updates would result
// in merge conflicts. The "global variable" approach, while not ideal,
// minimizes our code's footprint within the Geth codebase, allowing us to
// capture the information we need with minimal future conflicts.

package filters

import (
	"context"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	// "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru"
	"sync"
	"time"
)

var (
	blockPeerMap *lru.Cache
	txPeerMap *lru.Cache
	tsMap *lru.Cache
	peerIDMap *sync.Map
)

type peerInfo struct {
	Enode string `json:"enode"`
	ID string `json:"id"`
}


// SetBlockPeer is called when a block is received from a peer to track which
// peer was the first to provide a given block
func SetBlockPeer(hash common.Hash, peer string) {
	log.Debug("Recording block peer", "hash", hash, "peer", peer)
	if blockPeerMap == nil { blockPeerMap, _ = lru.New(250) }
	if tsMap == nil { tsMap, _ = lru.New(100000) }
	if _, ok := blockPeerMap.Get(hash); !ok {
		blockPeerMap.Add(hash, peer)
		tsMap.Add(hash, time.Now().UnixNano())
	}
}


// SetTxPeer is called when a transaction is received from a peer to track
// which peer was the first to provide a given transaction
func SetTxPeer(hash common.Hash, peer string) {
	if txPeerMap == nil { txPeerMap, _ = lru.New(100000) }
	if tsMap == nil { tsMap, _ = lru.New(100000) }
	if _, ok := txPeerMap.Get(hash); !ok {
		txPeerMap.Add(hash, peer)
		tsMap.Add(hash, time.Now().UnixNano())
	}
}

func ClearTxTimestamp(hash common.Hash) {
	tsMap.Remove(hash)
}

// SubscribePeerIDs tracks and populates the peerID map with the ID and enode,
// so that they can be provided in responses with transaction and block
// information
func SubscribePeerIDs(srv *p2p.Server) {
	go func(srv *p2p.Server) {
		ch := make(chan *p2p.PeerEvent, 1000)
		srv.SubscribeEvents(ch)
		for event := range ch {
			switch event.Type {
			case p2p.PeerEventTypeAdd:
				for _, peerInfo := range srv.PeersInfo() {
					if peerInfo.ID == event.Peer.String() {
						log.Debug("Adding peer", "id", event.Peer.String(), "enode", peerInfo.Enode)
						setPeerID(event.Peer.String(), peerInfo.Enode)
						break
					}
				}
			case p2p.PeerEventTypeDrop:
				log.Debug("Dropping peer", "id", event.Peer.String())
				dropPeerID(event.Peer.String())
			}
		}
	}(srv)
}

// setPeerID maps the truncated peerid to the full peerid and enode. The
// messages we will get later will only include the truncated peer id, so the
// full id and enode must be tracked based on connect / drop messages.
func setPeerID(peerid, enode string) {
	if peerIDMap == nil { peerIDMap = &sync.Map{} }
	if _, ok := peerIDMap.Load(peerid); !ok {
		peerIDMap.Store(peerid, peerInfo{ID: peerid, Enode: enode})
	}
}

// dropPeerID cleans up records when a peer drops
func dropPeerID(peerid string) {
	if peerIDMap == nil { return }
	peerIDMap.Delete(peerid)
}


// withPeer is a generic wrapper for different types of values distributed with
// peer information.
type withPeer struct {
	Value interface{} `json:"value"`
	Peer interface{} `json:"peer"`
	Time int64 `json:"ts"`
	P2PTime interface{} `json:"p2pts,omitempty"`
}

// NewHeadsWithPeers send a notification each time a new (header) block is
// appended to the chain, and includes the peer that first provided the block
func (api *PublicFilterAPI) NewHeadsWithPeers(ctx context.Context) (*rpc.Subscription, error) {
	if blockPeerMap == nil { blockPeerMap, _ = lru.New(250) }
	if peerIDMap == nil { peerIDMap = &sync.Map{} }
	if tsMap == nil { tsMap, _ = lru.New(100000) }
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		headers := make(chan *types.Header)
		headersSub := api.events.SubscribeNewHeads(headers)

		for {
			select {
			case h := <-headers:
				peerid, _ := blockPeerMap.Get(h.Hash())
				p2pts, _ := tsMap.Get(h.Hash())
				peer, _ := peerIDMap.Load(peerid)
				log.Debug("NewHeadsWithPeers", "hash", h.Hash(), "peer", peerid, "peer", peer)
				notifier.Notify(rpcSub.ID, withPeer{Value: h, Peer: peer, Time: time.Now().UnixNano(), P2PTime: p2pts} )
			case <-rpcSub.Err():
				headersSub.Unsubscribe()
				return
			case <-notifier.Closed():
				headersSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// NewFullBlocksWithPeers send a notification each time a new full block plus
// transactions and receipts is appended to the chain, and includes the peer
// that first provided the block
func (api *PublicFilterAPI) NewFullBlocksWithPeers(ctx context.Context) (*rpc.Subscription, error) {
	if blockPeerMap == nil { blockPeerMap, _ = lru.New(250) }
	if peerIDMap == nil { peerIDMap = &sync.Map{} }
	if tsMap == nil { tsMap, _ = lru.New(100000) }
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		headers := make(chan *types.Header)
		headersSub := api.events.SubscribeNewHeads(headers)

		for {
			select {
			case h := <-headers:
				hash := h.Hash()
				peerid, _ := blockPeerMap.Get(hash)

				block, err := api.backend.BlockByHash(ctx, hash)
				if err != nil { continue }
				marshalBlock, err := ethapi.RPCMarshalBlock(block, true, true)
				if err != nil { continue }

				marshalReceipts := make(map[common.Hash]map[string]interface{})
				receipts, err := api.backend.GetReceipts(ctx, hash)
				if err != nil {
					continue
				}
				for index, receipt := range receipts {
					fields := map[string]interface{}{
						"transactionIndex":  hexutil.Uint64(index),
						"gasUsed":           hexutil.Uint64(receipt.GasUsed),
						"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
						"contractAddress":   nil,
						"logs":              receipt.Logs,
						"logsBloom":         receipt.Bloom,
					}
					if receipt.Logs == nil {
						fields["logs"] = [][]*types.Log{}
					}
					// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
					if receipt.ContractAddress != (common.Address{}) {
						fields["contractAddress"] = receipt.ContractAddress
					}
					if reason, ok := core.GetRevertReason(receipt.TxHash); ok {
						fields["revertReason"] = reason
					}
					marshalReceipts[receipt.TxHash] = fields
				}
				marshalBlock["receipts"] = marshalReceipts


				p2pts, _ := tsMap.Get(hash)
				peer, _ := peerIDMap.Load(peerid)
				log.Debug("NewFullBlocksWithPeers", "hash", hash, "peer", peerid, "peer", peer)
				notifier.Notify(rpcSub.ID, withPeer{Value: marshalBlock, Peer: peer, Time: time.Now().UnixNano(), P2PTime: p2pts} )
			case <-rpcSub.Err():
				headersSub.Unsubscribe()
				return
			case <-notifier.Closed():
				headersSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// NewPendingTransactionsWithPeers creates a subscription that is triggered
// each time a transaction enters the transaction pool, and includes the peer
// that first provided the transaction
func (api *PublicFilterAPI) NewPendingTransactionsWithPeers(ctx context.Context) (*rpc.Subscription, error) {
	if txPeerMap == nil { txPeerMap, _ = lru.New(100000) }
	if peerIDMap == nil { peerIDMap = &sync.Map{} }
	if tsMap == nil { tsMap, _ = lru.New(100000) }
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		txHashes := make(chan []common.Hash, 128)
		pendingTxSub := api.events.SubscribePendingTxs(txHashes)

		for {
			select {
			case hashes := <-txHashes:
				for _, h := range hashes {
					peerid, _ := txPeerMap.Get(h)
					p2pts, _ := tsMap.Get(h)
					peer, _ := peerIDMap.Load(peerid)
					notifier.Notify(rpcSub.ID, withPeer{Value: newRPCPendingTransaction(api.backend.GetPoolTransaction(h)), Peer: peer, Time: time.Now().UnixNano(), P2PTime: p2pts})
				}
			case <-rpcSub.Err():
				pendingTxSub.Unsubscribe()
				return
			case <-notifier.Closed():
				pendingTxSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}


// NewTransactionReceipts creates a subscription that is triggered for each
// receipt in a newly confirmed block.
func (api *PublicFilterAPI) NewTransactionReceipts(ctx context.Context) (*rpc.Subscription, error) {
	if blockPeerMap == nil { blockPeerMap, _ = lru.New(250) }
	if peerIDMap == nil { peerIDMap = &sync.Map{} }
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		headers := make(chan *types.Header)
		headersSub := api.events.SubscribeNewHeads(headers)

		for {
			select {
			case h := <-headers:
				receipts, _ := api.backend.GetReceipts(ctx, h.Hash())
				for _, receipt := range receipts {
					notifier.Notify(rpcSub.ID, receipt )
				}
			case <-rpcSub.Err():
				headersSub.Unsubscribe()
				return
			case <-notifier.Closed():
				headersSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}
