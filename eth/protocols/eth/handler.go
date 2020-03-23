// Copyright 2020 The go-ethereum Authors
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

package eth

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	// softResponseLimit is the target maximum size of replies to data retrievals.
	softResponseLimit = 2 * 1024 * 1024

	// estHeaderSize is the approximate size of an RLP encoded block header.
	estHeaderSize = 500

	// maxHeadersServe is the maximum number of block headers to serve. This number
	// is there to limit the number of disk lookups.
	maxHeadersServe = 1024

	// maxBodiesServe is the maximum number of block bodies to serve. This number
	// is mostly there to limit the number of disk lookups. With 24KB block sizes
	// nowadays, the practical limit will always be softResponseLimit.
	maxBodiesServe = 1024

	// maxNodeDataServe is the maximum number of state trie nodes to serve. This
	// number is there to limit the number of disk lookups.
	maxNodeDataServe = 1024

	// maxReceiptsServe is the maximum number of block receipts to serve. This
	// number is mostly there to limit the number of disk lookups. With block
	// containing 200+ transactions nowadays, the practical limit will always
	// be softResponseLimit.
	maxReceiptsServe = 1024
)

// Handler is a callback to invoke from an outside runner after the boilerplate
// exchanges have passed.
type Handler func(peer *Peer) error

// Backend defines the data retrieval methods to serve remote requests and the
// callback methods to invoke on remote deliveries.
type Backend interface {
	// Chain retrieves the blockchain object to serve data.
	Chain() *core.BlockChain

	// StateBloom retrieves the bloom filter - if any - for state trie nodes.
	StateBloom() *trie.SyncBloom

	// TxPool retrieves the transaction pool object to serve data.
	TxPool() TxPool

	// RunPeer is invoked when a peer joins on the `eth` protocol. The handler
	// should do any peer maintenance work, handshakes and validations. If all
	// is passed, control should be given back to the `handler` to process the
	// inbound messages going forward.
	RunPeer(peer *Peer, handler Handler) error

	// PeerInfo retrieves all known `eth` information about a peer.
	PeerInfo(id enode.ID) interface{}

	// OnHeaders is a callback method to invoke when a batch of headers are
	// received from a remote peer.
	OnHeaders(peer *Peer, headers []*types.Header) error

	// OnBodies is a callback method to invoke when a batch of block bodies are
	// received from a remote peer.
	OnBodies(peer *Peer, txs [][]*types.Transaction, uncles [][]*types.Header) error

	// OnNodeData is a callback method to invoke when a batch of trie nodes are
	// received from a remote peer.
	OnNodeData(peer *Peer, nodes [][]byte) error

	// OnReceipts is a callback method to invoke when a batch of transaction
	// receipts are received from a remote peer.
	OnReceipts(peer *Peer, receipts [][]*types.Receipt) error

	// OnBlockAnnounces is a callback method to invoke when a batch of block
	// announcements are received from a remote peer.
	OnBlockAnnounces(peer *Peer, hashes []common.Hash, numbers []uint64) error

	// OnBlockBroadcast is a callback method to invoke when a batch of block
	// broadcast is received from a remote peer.
	OnBlockBroadcast(peer *Peer, block *types.Block, td *big.Int) error

	// AcceptTxs retrieves whether transaction processing is enabled on the node
	// or if inbound transactions should simply be dropped.
	AcceptTxs() bool

	// OnTxAnnounces is a callback method to invoke when a batch of transaction
	// announcements are received from a remote peer.
	OnTxAnnounces(peer *Peer, hashes []common.Hash) error

	// OnTxBroadcasts is a callback method to invoke when a batch of transaction
	// broadcasts are received from a remote peer.
	OnTxBroadcasts(peer *Peer, txs []*types.Transaction, direct bool) error
}

// TxPool defines the methods needed by the protocol handler to serve transactions.
type TxPool interface {
	// Get retrieves the the transaction from the local txpool with the given hash.
	Get(hash common.Hash) *types.Transaction
}

// MakeProtocols constructs the P2P protocol definitions for `eth`.
func MakeProtocols(backend Backend, network uint64, dnsdisc enode.Iterator) []p2p.Protocol {
	protocols := make([]p2p.Protocol, len(protocolVersions))
	for i, version := range protocolVersions {
		version := version // Closure

		protocols[i] = p2p.Protocol{
			Name:    protocolName,
			Version: version,
			Length:  protocolLengths[version],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := NewPeer(version, p, rw, backend.TxPool())
				defer peer.Close()

				return backend.RunPeer(peer, func(peer *Peer) error {
					return Handle(backend, peer)
				})
			},
			NodeInfo: func() interface{} {
				return nodeInfo(backend.Chain(), network)
			},
			PeerInfo: func(id enode.ID) interface{} {
				return backend.PeerInfo(id)
			},
			Attributes:     []enr.Entry{currentENREntry(backend.Chain())},
			DialCandidates: dnsdisc,
		}
	}
	return protocols
}

// NodeInfo represents a short summary of the `eth` sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network    uint64              `json:"network"`    // Ethereum network ID (1=Frontier, 2=Morden, Ropsten=3, Rinkeby=4)
	Difficulty *big.Int            `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    common.Hash         `json:"genesis"`    // SHA3 hash of the host's genesis block
	Config     *params.ChainConfig `json:"config"`     // Chain configuration for the fork rules
	Head       common.Hash         `json:"head"`       // Hex hash of the host's best owned block
}

// nodeInfo retrieves some `eth` protocol metadata about the running host node.
func nodeInfo(chain *core.BlockChain, network uint64) *NodeInfo {
	head := chain.CurrentBlock()
	return &NodeInfo{
		Network:    network,
		Difficulty: chain.GetTd(head.Hash(), head.NumberU64()),
		Genesis:    chain.Genesis().Hash(),
		Config:     chain.Config(),
		Head:       head.Hash(),
	}
}

// Handle is invoked whenever an `eth` connection is made that successfully passes
// the protocol handshake. This method will keep processing messages until the
// connection is torn down.
func Handle(backend Backend, peer *Peer) error {
	for {
		if err := handleMessage(backend, peer); err != nil {
			peer.Log().Debug("Message handling failed in `eth`", "err", err)
			return err
		}
	}
}

// handleMessage is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func handleMessage(backend Backend, peer *Peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := peer.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > maxMessageSize {
		return fmt.Errorf("%w: %v > %v", errMsgTooLarge, msg.Size, maxMessageSize)
	}
	defer msg.Discard()

	// Handle the message depending on its contents
	switch {
	case msg.Code == statusMsg:
		// Status messages should never arrive after the handshake
		return fmt.Errorf("%w: uncontrolled status message", errExtraStatusMsg)

	// Block header query, collect the requested headers and reply
	case msg.Code == getBlockHeadersMsg:
		// Decode the complex header query
		var query getBlockHeadersData
		if err := msg.Decode(&query); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		hashMode := query.Origin.Hash != (common.Hash{})
		first := true
		maxNonCanonical := uint64(100)

		// Gather headers until the fetch or network limits is reached
		var (
			bytes   common.StorageSize
			headers []*types.Header
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < maxHeadersServe {
			// Retrieve the next header satisfying the query
			var origin *types.Header
			if hashMode {
				if first {
					first = false
					origin = backend.Chain().GetHeaderByHash(query.Origin.Hash)
					if origin != nil {
						query.Origin.Number = origin.Number.Uint64()
					}
				} else {
					origin = backend.Chain().GetHeader(query.Origin.Hash, query.Origin.Number)
				}
			} else {
				origin = backend.Chain().GetHeaderByNumber(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			headers = append(headers, origin)
			bytes += estHeaderSize

			// Advance to the next header of the query
			switch {
			case hashMode && query.Reverse:
				// Hash based traversal towards the genesis block
				ancestor := query.Skip + 1
				if ancestor == 0 {
					unknown = true
				} else {
					query.Origin.Hash, query.Origin.Number = backend.Chain().GetAncestor(query.Origin.Hash, query.Origin.Number, ancestor, &maxNonCanonical)
					unknown = (query.Origin.Hash == common.Hash{})
				}
			case hashMode && !query.Reverse:
				// Hash based traversal towards the leaf block
				var (
					current = origin.Number.Uint64()
					next    = current + query.Skip + 1
				)
				if next <= current {
					infos, _ := json.MarshalIndent(peer.Peer.Info(), "", "  ")
					peer.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
					unknown = true
				} else {
					if header := backend.Chain().GetHeaderByNumber(next); header != nil {
						nextHash := header.Hash()
						expOldHash, _ := backend.Chain().GetAncestor(nextHash, next, query.Skip+1, &maxNonCanonical)
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
		}
		return peer.SendBlockHeaders(headers)

	case msg.Code == blockHeadersMsg:
		// A batch of headers arrived to one of our previous requests
		var headers []*types.Header
		if err := msg.Decode(&headers); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		return backend.OnHeaders(peer, headers)

	case msg.Code == getBlockBodiesMsg:
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
		for bytes < softResponseLimit && len(bodies) < maxBodiesServe {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
			}
			// Retrieve the requested block body, stopping if enough was found
			if data := backend.Chain().GetBodyRLP(hash); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)
			}
		}
		return peer.SendBlockBodiesRLP(bodies)

	case msg.Code == blockBodiesMsg:
		// A batch of block bodies arrived to one of our previous requests
		var request blockBodiesData
		if err := msg.Decode(&request); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		// Deliver them all to the backend for queuing
		txs := make([][]*types.Transaction, len(request))
		uncles := make([][]*types.Header, len(request))

		for i, body := range request {
			txs[i] = body.Transactions
			uncles[i] = body.Uncles
		}
		return backend.OnBodies(peer, txs, uncles)

	case msg.Code == getNodeDataMsg:
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
		for bytes < softResponseLimit && len(data) < maxNodeDataServe {
			// Retrieve the hash of the next state entry
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
			}
			// Retrieve the requested state entry, stopping if enough was found
			if bloom := backend.StateBloom(); bloom != nil && !bloom.Contains(hash[:]) {
				// Only lookup the trie node if there's chance that we actually have it
				continue
			}
			entry, err := backend.Chain().TrieNode(hash)
			if len(entry) == 0 || err != nil {
				// Read the contract code with prefix only to save unnecessary lookups.
				entry, err = backend.Chain().ContractCodeWithPrefix(hash)
			}
			if err == nil && len(entry) > 0 {
				data = append(data, entry)
				bytes += len(entry)
			}
		}
		return peer.SendNodeData(data)

	case msg.Code == nodeDataMsg:
		// A batch of node state data arrived to one of our previous requests
		var nodes [][]byte
		if err := msg.Decode(&nodes); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		return backend.OnNodeData(peer, nodes)

	case msg.Code == getReceiptsMsg:
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
		for bytes < softResponseLimit && len(receipts) < maxReceiptsServe {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
			}
			// Retrieve the requested block's receipts, skipping if unknown to us
			results := backend.Chain().GetReceiptsByHash(hash)
			if results == nil {
				if header := backend.Chain().GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
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
		return peer.SendReceiptsRLP(receipts)

	case msg.Code == receiptsMsg:
		// A batch of receipts arrived to one of our previous requests
		var receipts [][]*types.Receipt
		if err := msg.Decode(&receipts); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		return backend.OnReceipts(peer, receipts)

	case msg.Code == newBlockHashesMsg:
		var announces newBlockHashesData
		if err := msg.Decode(&announces); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		// Mark the hashes as present at the remote node
		for _, block := range announces {
			peer.markBlock(block.Hash)
		}
		// Deliver them all to the backend for queuing
		hashes := make([]common.Hash, len(announces))
		numbers := make([]uint64, len(announces))

		for i, body := range announces {
			hashes[i] = body.Hash
			numbers[i] = body.Number
		}
		return backend.OnBlockAnnounces(peer, hashes, numbers)

	case msg.Code == newBlockMsg:
		// Retrieve and decode the propagated block
		var request newBlockData
		if err := msg.Decode(&request); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		if hash := types.CalcUncleHash(request.Block.Uncles()); hash != request.Block.UncleHash() {
			log.Warn("Propagated block has invalid uncles", "have", hash, "exp", request.Block.UncleHash())
			break // TODO(karalabe): return error eventually, but wait a few releases
		}
		if hash := types.DeriveSha(request.Block.Transactions(), trie.NewStackTrie(nil)); hash != request.Block.TxHash() {
			log.Warn("Propagated block has invalid body", "have", hash, "exp", request.Block.TxHash())
			break // TODO(karalabe): return error eventually, but wait a few releases
		}
		if err := request.sanityCheck(); err != nil {
			return err
		}
		request.Block.ReceivedAt = msg.ReceivedAt
		request.Block.ReceivedFrom = peer

		// Mark the peer as owning the block
		peer.markBlock(request.Block.Hash())
		return backend.OnBlockBroadcast(peer, request.Block, request.TD)

	case msg.Code == newPooledTransactionHashesMsg && peer.version >= ETH65:
		// New transaction announcement arrived, make sure we have
		// a valid and fresh chain to handle them
		if !backend.AcceptTxs() {
			break
		}
		var hashes []common.Hash
		if err := msg.Decode(&hashes); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		// Schedule all the unknown hashes for retrieval
		for _, hash := range hashes {
			peer.markTransaction(hash)
		}
		return backend.OnTxAnnounces(peer, hashes)

	case msg.Code == getPooledTransactionsMsg && peer.version >= ETH65:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather transactions until the fetch or network limits is reached
		var (
			hash   common.Hash
			bytes  int
			hashes []common.Hash
			txs    []rlp.RawValue
		)
		for bytes < softResponseLimit {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
			}
			// Retrieve the requested transaction, skipping if unknown to us
			tx := backend.TxPool().Get(hash)
			if tx == nil {
				continue
			}
			// If known, encode and queue for response packet
			if encoded, err := rlp.EncodeToBytes(tx); err != nil {
				log.Error("Failed to encode transaction", "err", err)
			} else {
				hashes = append(hashes, hash)
				txs = append(txs, encoded)
				bytes += len(encoded)
			}
		}
		return peer.SendPooledTransactionsRLP(hashes, txs)

	case msg.Code == transactionMsg || (msg.Code == pooledTransactionsMsg && peer.version >= ETH65):
		// Transactions arrived, make sure we have a valid and fresh chain to handle them
		if !backend.AcceptTxs() {
			break
		}
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return fmt.Errorf("%w: transaction %d is nil", errDecode, i)
			}
			peer.markTransaction(tx.Hash())
		}
		return backend.OnTxBroadcasts(peer, txs, msg.Code == pooledTransactionsMsg)

	default:
		return fmt.Errorf("%w: %v", errInvalidMsgCode, msg.Code)
	}
	return nil
}
